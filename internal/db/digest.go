package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"tasks/internal/models"
)

// -------------------------------------------------------------
// DAILY DIGEST
// -------------------------------------------------------------
//
// Modelled on the Obsidian "Daily Brief" workflow: an agenda, a prioritised
// list for today, a watch list for the week, stale items, and what closed
// recently. Everything except the agenda is derived from the project's own
// tasks, so the digest is instant and reproducible; the agenda needs the AI
// agent because meetings live in a calendar Taskacao has no access to.

const (
	// staleAfterDays is how long a high-priority task may stay open before the
	// digest flags it. Mirrors the reference brief's 7-day rule.
	staleAfterDays = 7
	// dueSoonWindowDays bounds the "due soon" section.
	dueSoonWindowDays = 7
	// recentlyDoneWindowDays bounds the "closed recently" section.
	recentlyDoneWindowDays = 7
	// digestSectionCap keeps a section readable. Whatever is dropped is
	// reported in the rendered Markdown rather than silently truncated.
	digestSectionCap = 15
)

// hasReliableDates reports whether a task's CreatedAt / UpdatedAt reflect the
// real tracker dates. They do not for Jira: acli's 'workitem search --fields'
// allow-list rejects 'created' and 'updated', so the sync stamps the import
// time instead. Deriving "open for N days" or "closed recently" from that would
// be fabricated, so those sections skip such tasks and the digest says so.
func hasReliableDates(t models.Task) bool {
	return strings.ToLower(t.Source) != "jira"
}

// isOpenStatus reports whether a task still represents work to do.
func isOpenStatus(s models.Status) bool {
	switch s {
	case models.StatusFinished, models.StatusDone:
		return false
	}
	return true
}

// isReviewStatus reports whether a task sits in a review stage.
func isReviewStatus(s models.Status) bool {
	switch s {
	case models.StatusToTest, models.StatusToValidate, models.StatusToClose:
		return true
	}
	return false
}

func priorityRank(p models.Priority) int {
	switch p {
	case models.PriorityUrgent:
		return 0
	case models.PriorityHigh:
		return 1
	case models.PriorityMedium:
		return 2
	default:
		return 3
	}
}

// parseDigestDate accepts YYYY-MM-DD and defaults to today in the local zone.
func parseDigestDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	d, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("date invalide %q: attendu AAAA-MM-JJ", raw)
	}
	return d, nil
}

// toDigestRef projects a task into its digest representation, relative to the
// digest's reference day.
func toDigestRef(t models.Task, ref time.Time) models.DigestTaskRef {
	age := int(ref.Sub(t.CreatedAt).Hours() / 24)
	if age < 0 {
		age = 0
	}

	out := models.DigestTaskRef{
		Key:         t.Key,
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		IssueType:   t.IssueType,
		Assignee:    t.Assignee,
		ParentKey:   t.ParentKey,
		ParentTitle: t.ParentTitle,
		ExternalURL: t.ExternalURL,
		BranchName:  t.BranchName,
		PrURL:       t.PrURL,
		DueDate:     t.DueDate,
		AgeDays:     age,
	}

	if !hasReliableDates(t) {
		// Age is unknown: do not surface a number that would be read as the
		// tracker's own age.
		out.AgeDays = 0
		out.DatesUnknown = true
	}

	out.IsStale = !out.DatesUnknown &&
		age > staleAfterDays &&
		(t.Priority == models.PriorityUrgent || t.Priority == models.PriorityHigh) &&
		isOpenStatus(t.Status)

	if t.DueDate != nil && strings.TrimSpace(*t.DueDate) != "" {
		if due, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*t.DueDate)[:min(10, len(strings.TrimSpace(*t.DueDate)))], time.Local); err == nil {
			days := int(due.Sub(ref).Hours() / 24)
			out.DaysToDue = &days
		}
	}

	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sortRefs orders a section: priority first, then the oldest work, then key.
func sortRefs(refs []models.DigestTaskRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		pi, pj := priorityRank(refs[i].Priority), priorityRank(refs[j].Priority)
		if pi != pj {
			return pi < pj
		}
		if refs[i].AgeDays != refs[j].AgeDays {
			return refs[i].AgeDays > refs[j].AgeDays
		}
		return refs[i].Key < refs[j].Key
	})
}

// ComputeDailyDigest builds the deterministic part of a project's digest from
// its tasks. It never calls out to the network or to an AI agent.
//
// assignee narrows the digest to one person. This matters on a team backlog:
// Jira's default "Major" priority maps to high, so an unfiltered digest of a
// 1300-ticket project lists hundreds of "do today" items and stops being a
// brief. An empty assignee keeps every task.
func (d *DB) ComputeDailyDigest(projectID string, dateRaw string, assignee string) (*models.DailyDigest, error) {
	ref, err := parseDigestDate(dateRaw)
	if err != nil {
		return nil, err
	}

	d.mu.RLock()
	proj, _ := d.getProjectByIDUnsafe(projectID)
	d.mu.RUnlock()
	if proj == nil {
		return nil, fmt.Errorf("projet non trouvé: %s", projectID)
	}
	// The daily digest is a personal-board feature: it reads as a brief for one
	// person, not for a delivery project shared by a team.
	if NormalizeProjectType(proj.ProjectType) != "personal" {
		return nil, fmt.Errorf("le digest quotidien est réservé aux projets de type personnel (projet %s)", proj.Name)
	}

	tasks, err := d.GetTasks("", "", "", "", proj.ID, "", "", "", nil, false)
	if err != nil {
		return nil, err
	}

	// Initialise every slice: a nil slice marshals to JSON null, and the UI
	// would then read .length off null.
	assignee = strings.TrimSpace(assignee)

	digest := &models.DailyDigest{
		Assignee:       assignee,
		ProjectID:      proj.ID,
		ProjectName:    proj.Name,
		Date:           ref.Format("2006-01-02"),
		AIStatus:       "none",
		GeneratedAt:    time.Now(),
		Focus:          []models.DigestTaskRef{},
		Watch:          []models.DigestTaskRef{},
		Stale:          []models.DigestTaskRef{},
		DueSoon:        []models.DigestTaskRef{},
		AwaitingReview: []models.DigestTaskRef{},
		RecentlyDone:   []models.DigestTaskRef{},
		ByEpic:         []models.DigestEpicGroup{},
	}

	seenAssignees := map[string]bool{}
	epicOpen := map[string]int{}
	epicDone := map[string]int{}
	epicTitle := map[string]string{}

	for _, t := range tasks {
		if assignee != "" && !strings.EqualFold(strings.TrimSpace(t.Assignee), assignee) {
			continue
		}

		// Every assignee seen in the project, so the UI can offer a real list
		// instead of guessing how the tracker spells names.
		if a := strings.TrimSpace(t.Assignee); a != "" {
			seenAssignees[a] = true
		}

		r := toDigestRef(t, ref)
		open := isOpenStatus(t.Status)

		if t.ParentKey != "" {
			epicTitle[t.ParentKey] = t.ParentTitle
			if open {
				epicOpen[t.ParentKey]++
			} else {
				epicDone[t.ParentKey]++
			}
		}

		if !open {
			// Closed: keep it only if it closed inside the window, and only when
			// the closing date is trustworthy.
			if !hasReliableDates(t) {
				digest.Stats.ClosedDateUnknown++
				continue
			}
			if int(ref.Sub(t.UpdatedAt).Hours()/24) <= recentlyDoneWindowDays && !t.UpdatedAt.After(ref.AddDate(0, 0, 1)) {
				digest.RecentlyDone = append(digest.RecentlyDone, r)
				digest.Stats.DoneLast7Days++
			}
			continue
		}

		digest.Stats.TotalOpen++
		if r.DatesUnknown {
			digest.Stats.OpenDateUnknown++
		}
		switch t.Priority {
		case models.PriorityUrgent:
			digest.Stats.Urgent++
		case models.PriorityHigh:
			digest.Stats.High++
		}

		if r.IsStale {
			digest.Stale = append(digest.Stale, r)
			digest.Stats.Stale++
		}

		if r.DaysToDue != nil && *r.DaysToDue <= dueSoonWindowDays {
			digest.DueSoon = append(digest.DueSoon, r)
			if *r.DaysToDue < 0 {
				digest.Stats.Overdue++
			}
		}

		// A task carrying a PR, or sitting in a review stage, is waiting on
		// someone rather than on the assignee.
		if (t.PrURL != nil && strings.TrimSpace(*t.PrURL) != "") || isReviewStatus(t.Status) {
			digest.AwaitingReview = append(digest.AwaitingReview, r)
			digest.Stats.AwaitingReview++
			continue
		}

		switch t.Priority {
		case models.PriorityUrgent, models.PriorityHigh:
			digest.Focus = append(digest.Focus, r)
		default:
			digest.Watch = append(digest.Watch, r)
		}
	}

	sortRefs(digest.Focus)
	sortRefs(digest.Watch)
	sortRefs(digest.Stale)
	sortRefs(digest.AwaitingReview)
	sortRefs(digest.RecentlyDone)

	sort.SliceStable(digest.DueSoon, func(i, j int) bool {
		di, dj := 0, 0
		if digest.DueSoon[i].DaysToDue != nil {
			di = *digest.DueSoon[i].DaysToDue
		}
		if digest.DueSoon[j].DaysToDue != nil {
			dj = *digest.DueSoon[j].DaysToDue
		}
		return di < dj
	})

	for key, openCount := range epicOpen {
		digest.ByEpic = append(digest.ByEpic, models.DigestEpicGroup{
			ParentKey:   key,
			ParentTitle: epicTitle[key],
			OpenCount:   openCount,
			DoneCount:   epicDone[key],
		})
	}
	// Epics with no open work left are noise in a daily brief.
	for key, doneCount := range epicDone {
		if _, ok := epicOpen[key]; !ok {
			digest.ByEpic = append(digest.ByEpic, models.DigestEpicGroup{
				ParentKey:   key,
				ParentTitle: epicTitle[key],
				DoneCount:   doneCount,
			})
		}
	}
	sort.SliceStable(digest.ByEpic, func(i, j int) bool {
		if digest.ByEpic[i].OpenCount != digest.ByEpic[j].OpenCount {
			return digest.ByEpic[i].OpenCount > digest.ByEpic[j].OpenCount
		}
		return digest.ByEpic[i].ParentKey < digest.ByEpic[j].ParentKey
	})

	for a := range seenAssignees {
		digest.Assignees = append(digest.Assignees, a)
	}
	sort.Strings(digest.Assignees)

	// Carry over a previously computed agenda so recomputing the task sections
	// does not throw away the AI pass.
	if stored, _ := d.getStoredDigest(proj.ID, digest.Date); stored != nil {
		digest.Agenda = stored.Agenda
		digest.AIStatus = stored.AIStatus
		digest.AIError = stored.AIError
		digest.AIActivityID = stored.AIActivityID
		digest.AIUpdatedAt = stored.AIUpdatedAt
	}

	digest.Markdown = renderDigestMarkdown(digest)
	return digest, nil
}

// renderRefLine formats one task as a Markdown bullet.
func renderRefLine(r models.DigestTaskRef) string {
	var b strings.Builder
	b.WriteString("- **")
	b.WriteString(r.Key)
	b.WriteString("**")

	var flags []string
	if r.IsStale {
		flags = append(flags, fmt.Sprintf("%dj ouvert", r.AgeDays))
	}
	if r.DaysToDue != nil {
		switch {
		case *r.DaysToDue < 0:
			flags = append(flags, fmt.Sprintf("échéance dépassée de %dj", -*r.DaysToDue))
		case *r.DaysToDue == 0:
			flags = append(flags, "échéance aujourd'hui")
		default:
			flags = append(flags, fmt.Sprintf("échéance dans %dj", *r.DaysToDue))
		}
	}
	if r.PrURL != nil && strings.TrimSpace(*r.PrURL) != "" {
		flags = append(flags, "PR ouverte")
	}
	if len(flags) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(flags, ", "))
		b.WriteString(")")
	}

	b.WriteString(" — ")
	b.WriteString(r.Title)

	if r.ParentKey != "" {
		b.WriteString(fmt.Sprintf(" · epic %s", r.ParentKey))
		if r.ParentTitle != "" {
			b.WriteString(fmt.Sprintf(" %s", r.ParentTitle))
		}
	}
	return b.String()
}

// renderSection writes a capped section, stating explicitly what it left out.
func renderSection(b *strings.Builder, title string, refs []models.DigestTaskRef, emptyText string) {
	b.WriteString("\n## ")
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(refs) == 0 {
		b.WriteString(emptyText)
		b.WriteString("\n")
		return
	}

	shown := refs
	if len(shown) > digestSectionCap {
		shown = shown[:digestSectionCap]
	}
	for _, r := range shown {
		b.WriteString(renderRefLine(r))
		b.WriteString("\n")
	}
	if len(refs) > len(shown) {
		b.WriteString(fmt.Sprintf("\n*%d autres non détaillés ici.*\n", len(refs)-len(shown)))
	}
}

// renderDigestMarkdown renders the digest in the shape of the reference brief.
func renderDigestMarkdown(dg *models.DailyDigest) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# 📋 Daily Digest — %s — %s\n\n", dg.ProjectName, dg.Date))
	if dg.Assignee != "" {
		b.WriteString(fmt.Sprintf("Périmètre : tâches assignées à **%s**\n\n", dg.Assignee))
	}
	b.WriteString(fmt.Sprintf("%d ouvertes · %d urgentes · %d hautes · %d stale · %d en retard · %d en revue · %d terminées (7j)\n",
		dg.Stats.TotalOpen, dg.Stats.Urgent, dg.Stats.High, dg.Stats.Stale,
		dg.Stats.Overdue, dg.Stats.AwaitingReview, dg.Stats.DoneLast7Days))

	b.WriteString("\n## 📅 Agenda du jour\n\n")
	switch {
	case strings.TrimSpace(dg.Agenda) != "":
		b.WriteString(strings.TrimSpace(dg.Agenda))
		b.WriteString("\n")
	case dg.AIStatus == "queued" || dg.AIStatus == "running":
		b.WriteString("*Récupération de l'agenda par l'agent en cours…*\n")
	case dg.AIStatus == "failed":
		b.WriteString(fmt.Sprintf("*Agenda indisponible : %s*\n", dg.AIError))
	default:
		b.WriteString("*Agenda non récupéré. Taskacao ne voit pas votre calendrier : lancez l'enrichissement pour que l'agent du projet le remonte.*\n")
	}

	renderSection(&b, "🔥 À traiter aujourd'hui (urgent / haute)", dg.Focus,
		"Rien d'urgent ou de haute priorité en attente.")
	staleEmpty := fmt.Sprintf("Aucune tâche prioritaire ouverte depuis plus de %d jours.", staleAfterDays)
	if dg.Stats.OpenDateUnknown > 0 {
		staleEmpty = fmt.Sprintf(
			"Ancienneté indisponible pour %d tâches ouvertes : la CLI Atlassian ne renvoie pas les champs `created` / `updated`, la synchro ne peut donc pas connaître la date réelle du ticket.",
			dg.Stats.OpenDateUnknown)
	}
	renderSection(&b, "⏳ Trop longtemps ouvertes", dg.Stale, staleEmpty)
	renderSection(&b, "📆 Échéances", dg.DueSoon,
		"Aucune échéance dans les 7 prochains jours.")
	renderSection(&b, "👀 En attente de revue", dg.AwaitingReview,
		"Rien en attente de revue.")
	renderSection(&b, "🟡 À ne pas oublier cette semaine", dg.Watch,
		"Rien d'autre en cours.")
	doneEmpty := "Aucune tâche terminée sur les 7 derniers jours."
	if dg.Stats.ClosedDateUnknown > 0 {
		doneEmpty = fmt.Sprintf(
			"%d tâches terminées, mais la date de clôture est indisponible (champ `updated` non exposé par la CLI Atlassian), donc aucune fenêtre de 7 jours ne peut être calculée.",
			dg.Stats.ClosedDateUnknown)
	}
	renderSection(&b, "✅ Terminées récemment", dg.RecentlyDone, doneEmpty)

	b.WriteString("\n## 🗂 Charge par epic\n\n")
	if len(dg.ByEpic) == 0 {
		b.WriteString("Aucun epic rattaché aux tâches de ce projet.\n")
	} else {
		b.WriteString("| Epic | Titre | Ouvertes | Terminées |\n|---|---|---:|---:|\n")
		shown := dg.ByEpic
		if len(shown) > digestSectionCap {
			shown = shown[:digestSectionCap]
		}
		for _, g := range shown {
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %d |\n", g.ParentKey, g.ParentTitle, g.OpenCount, g.DoneCount))
		}
		if len(dg.ByEpic) > len(shown) {
			b.WriteString(fmt.Sprintf("\n*%d autres epics non détaillés ici.*\n", len(dg.ByEpic)-len(shown)))
		}
	}

	return b.String()
}

// -------------------------------------------------------------
// Persistence
// -------------------------------------------------------------

func (d *DB) getStoredDigest(projectID string, date string) (*models.DailyDigest, error) {
	var payload, agenda, aiStatus, aiErr, aiActivity string
	var aiUpdated *time.Time

	d.mu.RLock()
	err := d.conn.QueryRow(`
		SELECT payload, agenda, ai_status, ai_error, ai_activity_id, ai_updated_at
		FROM daily_digests WHERE project_id = ? AND date = ?
	`, projectID, date).Scan(&payload, &agenda, &aiStatus, &aiErr, &aiActivity, &aiUpdated)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	var dg models.DailyDigest
	if payload != "" {
		_ = json.Unmarshal([]byte(payload), &dg)
	}
	dg.Agenda = agenda
	dg.AIStatus = aiStatus
	dg.AIError = aiErr
	dg.AIActivityID = aiActivity
	dg.AIUpdatedAt = aiUpdated
	return &dg, nil
}

// SaveDailyDigest upserts a digest for one project and day.
func (d *DB) SaveDailyDigest(dg *models.DailyDigest) error {
	if dg == nil {
		return fmt.Errorf("digest vide")
	}
	payload, _ := json.Marshal(dg)

	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.Exec(`
		INSERT INTO daily_digests (id, project_id, date, payload, agenda, ai_status, ai_error, ai_activity_id, ai_updated_at, markdown, generated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, date) DO UPDATE SET
			payload = excluded.payload,
			agenda = excluded.agenda,
			ai_status = excluded.ai_status,
			ai_error = excluded.ai_error,
			ai_activity_id = excluded.ai_activity_id,
			ai_updated_at = excluded.ai_updated_at,
			markdown = excluded.markdown,
			generated_at = excluded.generated_at
	`, uuid.New().String(), dg.ProjectID, dg.Date, string(payload), dg.Agenda,
		dg.AIStatus, dg.AIError, dg.AIActivityID, dg.AIUpdatedAt, dg.Markdown, dg.GeneratedAt)
	return err
}

// ListDigestDates returns the days a digest exists for, most recent first.
func (d *DB) ListDigestDates(projectID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 30
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT date FROM daily_digests WHERE project_id = ? ORDER BY date DESC LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var dt string
		if rows.Scan(&dt) == nil {
			out = append(out, dt)
		}
	}
	return out, nil
}

// -------------------------------------------------------------
// AI enrichment (agenda)
// -------------------------------------------------------------

// digestAgendaPrompt is what the project's agent is asked for. It is
// deliberately narrow: Taskacao already owns the task sections, so the agent is
// asked only for what it alone can see, and told to say so when it cannot.
func digestAgendaPrompt(projectName string, date string) string {
	return fmt.Sprintf(`Tu prépares la section "Agenda du jour" d'un daily brief pour le projet %s, à la date du %s.

Réponds UNIQUEMENT en Markdown, sans préambule ni conclusion.

1. Un tableau des réunions du jour, colonnes : Heure | Événement | Participants notables.
   Utilise les noms des personnes, jamais les adresses e-mail, et ignore les salles de réunion.
2. Sous le tableau, une ligne par annonce notable détectée (départ, deadline, changement
   d'organisation), sous forme de citation Markdown. S'il n'y en a pas, écris exactement :
   "Pas d'annonce notable détectée aujourd'hui."

Si tu n'as pas accès au calendrier, écris exactement : "Agenda indisponible : pas d'accès au
calendrier depuis cet agent." N'invente aucune réunion.`, projectName, date)
}

// EnqueueDigestAgenda runs the agenda pass for a digest and records it as an
// activity, so the run is visible in the Activities view like any other job.
// It is synchronous from the caller's point of view but bounded by a timeout.
func (d *DB) EnqueueDigestAgenda(projectID string, dateRaw string, assignee string) (*models.DailyDigest, error) {
	ref, err := parseDigestDate(dateRaw)
	if err != nil {
		return nil, err
	}
	date := ref.Format("2006-01-02")

	dg, err := d.ComputeDailyDigest(projectID, date, assignee)
	if err != nil {
		return nil, err
	}

	settings, _ := d.GetSettings()
	if settings == nil {
		return nil, fmt.Errorf("réglages indisponibles")
	}
	runnerSettings := *settings

	d.mu.RLock()
	proj, _ := d.getProjectByIDUnsafe(projectID)
	d.mu.RUnlock()
	if proj != nil {
		if proj.RepoPath != "" {
			runnerSettings.RepoPath = proj.RepoPath
		}
		if proj.AIProvider != "" {
			runnerSettings.AIProvider = proj.AIProvider
		}
		if proj.AICommandTemplate != "" {
			runnerSettings.AICommandTemplate = proj.AICommandTemplate
		}
	}

	activityID := uuid.New().String()
	now := time.Now()
	prompt := digestAgendaPrompt(dg.ProjectName, date)

	act := models.TaskActivity{
		ID:        activityID,
		TaskID:    "digest-" + projectID,
		ProjectID: projectID,
		SkillID:   "daily_digest",
		SkillName: "Daily Digest — Agenda",
		Action:    fmt.Sprintf("Récupération de l'agenda du %s", date),
		Status:    string(models.ActivityStatusRunning),
		Summary:   fmt.Sprintf("Agenda du %s pour %s", date, dg.ProjectName),
		Steps:     []string{"Interrogation de l'agent du projet pour l'agenda du jour"},
		Prompt:    prompt,
		CreatedAt: now,
		StartedAt: &now,
	}
	d.mu.Lock()
	_ = d.addTaskActivityDirect(act)
	d.mu.Unlock()

	dg.AIStatus = "running"
	dg.AIActivityID = activityID
	dg.Markdown = renderDigestMarkdown(dg)
	_ = d.SaveDailyDigest(dg)

	// The agent may need to reach a calendar over the network; give it room but
	// never let it hang the request forever.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	output, steps, runErr := d.runner.RunAgentPrompt(ctx, &runnerSettings, prompt)
	completed := time.Now()

	if runErr != nil {
		dg.AIStatus = "failed"
		dg.AIError = runErr.Error()
	} else if strings.TrimSpace(output) == "" {
		dg.AIStatus = "failed"
		dg.AIError = "l'agent n'a renvoyé aucun contenu"
	} else {
		dg.AIStatus = "completed"
		dg.AIError = ""
		dg.Agenda = strings.TrimSpace(output)
	}
	dg.AIUpdatedAt = &completed
	dg.Markdown = renderDigestMarkdown(dg)
	if saveErr := d.SaveDailyDigest(dg); saveErr != nil {
		return dg, saveErr
	}

	status := string(models.ActivityStatusCompleted)
	if dg.AIStatus == "failed" {
		status = string(models.ActivityStatusFailed)
	}
	stepsJSON, _ := json.Marshal(append(steps, "Agenda enregistré dans le digest"))

	d.mu.Lock()
	_, _ = d.conn.Exec(`
		UPDATE task_activities
		SET status = ?, output = ?, steps = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, status, dg.Agenda, string(stepsJSON), dg.AIError, completed, activityID)
	d.mu.Unlock()

	return dg, nil
}
