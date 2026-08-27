package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"tasks/internal/models"
	"tasks/internal/runner"
)

// Les épics ne sont pas des cartes : le tracker les traite comme des conteneurs
// et la synchro n'importe que Task et Story. Leur horizon — NOW, NEXT, LATER —
// est une décision produit, pas une déduction, et le travail de cadrage qu'on
// mène dessus (description, TODO) n'a nulle part où vivre côté tracker. Cette
// table est donc la mémoire propre de Taskacao sur les épics, indexée par la clé
// du ticket parent.

const (
	HorizonNow   = "now"
	HorizonNext  = "next"
	HorizonLater = "later"
	// HorizonHidden est le tout-venant : des épics fourre-tout qui n'ont pas
	// vocation à apparaître dans la roadmap, mais qu'on ne veut pas voir
	// remonter indéfiniment dans les non classés.
	HorizonHidden = "hidden"
)

// Le label de roadmap porté par l'épic dans Jira. L'horizon est une décision
// d'équipe : la garder en base locale la rendrait invisible aux collègues, hors
// de Jira et absente d'un autre poste. La description et la TODO, elles, restent
// locales : les écrire dans la description de l'épic détruirait sa mise en forme.
const RoadmapLabelPrefix = "roadmap:"

// RoadmapLabel is the Jira label for an horizon, empty for "unclassified".
func RoadmapLabel(horizon string) string {
	h := normalizeHorizon(horizon)
	if h == "" {
		return ""
	}
	return RoadmapLabelPrefix + h
}

// AllRoadmapLabels lists the three labels, to remove the ones that no longer apply.
func AllRoadmapLabels() []string {
	return []string{RoadmapLabelPrefix + HorizonNow, RoadmapLabelPrefix + HorizonNext, RoadmapLabelPrefix + HorizonLater, RoadmapLabelPrefix + HorizonHidden}
}

// HorizonFromLabels reads the horizon a Jira epic carries, empty when it carries none.
func HorizonFromLabels(labels []string) string {
	for _, l := range labels {
		clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(l, "#")))
		if strings.HasPrefix(clean, RoadmapLabelPrefix) {
			if h := normalizeHorizon(strings.TrimPrefix(clean, RoadmapLabelPrefix)); h != "" {
				return h
			}
		}
	}
	return ""
}

func normalizeHorizon(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case HorizonNow:
		return HorizonNow
	case HorizonNext:
		return HorizonNext
	case HorizonLater, "future":
		return HorizonLater
	case HorizonHidden:
		return HorizonHidden
	default:
		// Chaîne vide = non classé, ce qui est un état légitime : un épic qui
		// vient d'apparaître n'a pas encore été arbitré.
		return ""
	}
}

func (d *DB) ensureEpicsTable() {
	_, _ = d.conn.Exec(`CREATE TABLE IF NOT EXISTS epics (
		project_id TEXT NOT NULL,
		key TEXT NOT NULL,
		horizon TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		todos TEXT NOT NULL DEFAULT '[]',
		title TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT '',
		closed INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (project_id, key)
	);`)
	// Colonnes ajoutées après coup : les bases existantes les reçoivent ici.
	_, _ = d.conn.Exec("ALTER TABLE epics ADD COLUMN title TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE epics ADD COLUMN status TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE epics ADD COLUMN closed INTEGER NOT NULL DEFAULT 0;")
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_epics_project ON epics(project_id, horizon);")
}

// GetProjectEpics returns the epic metadata of a project, keyed by epic key.
func (d *DB) GetProjectEpics(projectID string) ([]models.EpicMeta, error) {
	d.mu.Lock()
	d.ensureEpicsTable()
	d.mu.Unlock()

	d.mu.RLock()
	rows, err := d.conn.Query(`
		SELECT project_id, key, horizon, description, todos, title, status, closed, updated_at
		FROM epics WHERE project_id = ? ORDER BY key ASC
	`, projectID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.EpicMeta{}
	for rows.Next() {
		var e models.EpicMeta
		var todosJSON string
		var closed int
		if err := rows.Scan(&e.ProjectID, &e.Key, &e.Horizon, &e.Description, &todosJSON, &e.Title, &e.Status, &closed, &e.UpdatedAt); err != nil {
			continue
		}
		e.Closed = closed == 1
		e.Todos = parseEpicTodos(todosJSON)
		out = append(out, e)
	}
	return out, nil
}

func parseEpicTodos(raw string) []models.EpicTodo {
	if strings.TrimSpace(raw) == "" || raw == "[]" {
		return []models.EpicTodo{}
	}
	var list []models.EpicTodo
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []models.EpicTodo{}
	}
	return list
}

// SaveEpicMeta upserts an epic's horizon, description and todos. Only the fields
// provided are touched: classifying an epic must not wipe the shaping notes, and
// editing the notes must not reset the horizon.
func (d *DB) SaveEpicMeta(projectID string, key string, horizon *string, description *string, todos *[]models.EpicTodo) (*models.EpicMeta, error) {
	return d.saveEpicMetaFull(projectID, key, horizon, description, todos, nil, nil, nil)
}

// saveEpicMetaFull ajoute les champs que seule la synchro renseigne : titre,
// statut et « terminé ». Un champ nil n'est pas touché, pour qu'un classement
// n'écrase pas ce que la synchro a lu, et inversement.
func (d *DB) saveEpicMetaFull(projectID string, key string, horizon *string, description *string, todos *[]models.EpicTodo, title *string, status *string, closed *bool) (*models.EpicMeta, error) {
	projectID = strings.TrimSpace(projectID)
	key = strings.TrimSpace(key)
	if projectID == "" || key == "" {
		return nil, fmt.Errorf("projet et clé d'épic obligatoires")
	}

	d.mu.Lock()
	d.ensureEpicsTable()

	current := models.EpicMeta{ProjectID: projectID, Key: key, Todos: []models.EpicTodo{}}
	var todosJSON string
	var closedInt int
	err := d.conn.QueryRow(`
		SELECT horizon, description, todos, title, status, closed FROM epics WHERE project_id = ? AND key = ?
	`, projectID, key).Scan(&current.Horizon, &current.Description, &todosJSON, &current.Title, &current.Status, &closedInt)
	if err == nil {
		current.Todos = parseEpicTodos(todosJSON)
		current.Closed = closedInt == 1
	}

	if title != nil {
		current.Title = *title
	}
	if status != nil {
		current.Status = *status
	}
	if closed != nil {
		current.Closed = *closed
	}

	if horizon != nil {
		current.Horizon = normalizeHorizon(*horizon)
	}
	if description != nil {
		current.Description = *description
	}
	if todos != nil {
		cleaned := make([]models.EpicTodo, 0, len(*todos))
		for _, todo := range *todos {
			text := strings.TrimSpace(todo.Text)
			if text == "" {
				continue
			}
			if strings.TrimSpace(todo.ID) == "" {
				todo.ID = uuid.New().String()
			}
			todo.Text = text
			cleaned = append(cleaned, todo)
		}
		current.Todos = cleaned
	}
	current.UpdatedAt = time.Now()

	payload, _ := json.Marshal(current.Todos)
	closedValue := 0
	if current.Closed {
		closedValue = 1
	}
	_, execErr := d.conn.Exec(`
		INSERT INTO epics (project_id, key, horizon, description, todos, title, status, closed, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, key) DO UPDATE SET
			horizon = excluded.horizon,
			description = excluded.description,
			todos = excluded.todos,
			title = excluded.title,
			status = excluded.status,
			closed = excluded.closed,
			updated_at = excluded.updated_at
	`, projectID, key, current.Horizon, current.Description, string(payload), current.Title, current.Status, closedValue, current.UpdatedAt)
	d.mu.Unlock()
	if execErr != nil {
		return nil, execErr
	}

	return &current, nil
}

// CreateStoryFromEpicTodo turns a line of epic shaping into a real story in the
// tracker, under that epic, and records the created key on the todo so the line
// is never turned into a second ticket.
//
// The task is also inserted locally: waiting for the next sync would leave the
// user staring at a story that exists in Jira but nowhere on the board.
func (d *DB) CreateStoryFromEpicTodo(projectID string, epicKey string, todoID string) (*models.EpicMeta, string, error) {
	projectID = strings.TrimSpace(projectID)
	epicKey = strings.TrimSpace(epicKey)
	todoID = strings.TrimSpace(todoID)
	if projectID == "" || epicKey == "" || todoID == "" {
		return nil, "", fmt.Errorf("projet, épic et ligne de TODO obligatoires")
	}

	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, "", fmt.Errorf("projet non trouvé")
	}
	if proj.IssueTracker != "jira" {
		return nil, "", fmt.Errorf("la création de story depuis une TODO n'est disponible que sur un projet Jira")
	}

	metas, err := d.GetProjectEpics(projectID)
	if err != nil {
		return nil, "", err
	}
	var meta *models.EpicMeta
	for i := range metas {
		if metas[i].Key == epicKey {
			meta = &metas[i]
			break
		}
	}
	if meta == nil {
		return nil, "", fmt.Errorf("épic %s sans cadrage enregistré", epicKey)
	}

	var todo *models.EpicTodo
	for i := range meta.Todos {
		if meta.Todos[i].ID == todoID {
			todo = &meta.Todos[i]
			break
		}
	}
	if todo == nil {
		return nil, "", fmt.Errorf("ligne de TODO introuvable")
	}
	if strings.TrimSpace(todo.StoryKey) != "" {
		return nil, "", fmt.Errorf("cette ligne a déjà produit %s", todo.StoryKey)
	}

	projectKey := jiraProjectKeyFor(proj)
	if projectKey == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			projectKey = settings.JiraProject
		}
	}

	// La description rappelle d'où vient la story : sans ça, un lecteur du
	// ticket ne sait pas qu'il vient d'un cadrage d'épic.
	description := fmt.Sprintf("Créé depuis le cadrage de l'épic %s dans Taskacao.", epicKey)
	key, err := d.runner.CreateJiraChildIssue(projectKey, projectRepoPath(proj), epicKey, "Story", todo.Text, description, nil)
	if err != nil {
		return nil, "", err
	}

	// Insertion locale immédiate, avec le parent renseigné pour que la story
	// apparaisse sous son épic dans la roadmap.
	task := models.Task{
		ID:          "jira-" + key,
		ProjectID:   proj.ID,
		Key:         key,
		Title:       todo.Text,
		Description: description,
		Status:      models.StatusToClarify,
		Priority:    models.PriorityMedium,
		Source:      "jira",
		IssueType:   "Story",
		ParentKey:   epicKey,
		ParentTitle: "",
		ParentType:  "Epic",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if impErr := d.ImportOrUpdateTasks([]models.Task{task}); impErr != nil {
		// La story existe côté Jira : on le dit plutôt que de faire échouer.
		return nil, key, fmt.Errorf("%s créé dans Jira mais non inséré localement: %w", key, impErr)
	}

	todos := append([]models.EpicTodo{}, meta.Todos...)
	for i := range todos {
		if todos[i].ID == todoID {
			todos[i].StoryKey = key
		}
	}
	saved, err := d.SaveEpicMeta(projectID, epicKey, nil, nil, &todos)
	if err != nil {
		return nil, key, err
	}
	return saved, key, nil
}

// projectRepoPath is the working directory acli runs in for this project: the
// CLI reads its credentials from the user profile, but the directory still
// matters for consistency with the rest of the tracker calls.
func projectRepoPath(proj *models.Project) string {
	if proj == nil {
		return ""
	}
	return strings.TrimSpace(proj.RepoPath)
}

// PushEpicHorizonLabel mirrors the classification onto the Jira epic: it adds the
// label of the chosen horizon and removes the other two. It performs the tracker
// call itself, so it is only ever run from a queued activity (TrackerOpEpicHorizon
// or TrackerOpPushHorizons): a click must not wait on it, and its failure has to
// stay readable in the activity rather than vanish.
func (d *DB) PushEpicHorizonLabel(projectID string, epicKey string, horizon string) (string, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return "", fmt.Errorf("projet non trouvé")
	}
	if proj.IssueTracker != "jira" {
		return "tracker non Jira : classification gardée en local", nil
	}

	target := RoadmapLabel(horizon)
	removed := []string{}
	for _, label := range AllRoadmapLabels() {
		if label != target {
			removed = append(removed, label)
		}
	}

	add := []string{}
	if target != "" {
		add = append(add, target)
	}

	// REST d'abord : six fois plus rapide qu'un lancement d'acli, ce qui compte
	// quand on trie épic par épic. Dans les deux cas, seuls les labels bougent :
	// la description riche de l'épic n'est jamais renvoyée.
	settings, _ := d.GetSettings()
	if client := runner.NewJiraRESTClient(settings, proj.TrackerUrl); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := client.UpdateIssueLabels(ctx, epicKey, add, removed); err != nil {
			return "", err
		}
	} else if err := d.runner.UpdateJiraIssue(epicKey, projectRepoPath(proj), nil, nil, nil, nil, add, removed); err != nil {
		return "", err
	}
	if target == "" {
		return fmt.Sprintf("labels roadmap retirés de %s", epicKey), nil
	}
	return fmt.Sprintf("%s posé sur %s", target, epicKey), nil
}

// ImportEpicHorizons reads the roadmap labels of a project's epics and records
// them locally. Jira wins when an epic carries a label, since that is the shared
// source; an epic without label keeps whatever was decided locally, which will be
// pushed the next time it is touched.
func (d *DB) ImportEpicHorizons(projectID string) (string, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return "", fmt.Errorf("projet non trouvé")
	}
	projectKey := jiraProjectKeyFor(proj)
	if projectKey == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			projectKey = settings.JiraProject
		}
	}
	if projectKey == "" {
		return "", fmt.Errorf("clé de projet Jira absente")
	}

	settings, _ := d.GetSettings()
	epics, err := d.runner.FetchJiraEpics(settings, proj.TrackerUrl, projectKey, projectRepoPath(proj))
	if err != nil {
		return "", err
	}

	// Mêmes épics hors projet à l'import : leur classement par un collègue doit
	// revenir, et leur titre comme leur statut doivent être connus.
	if metas, _ := d.GetProjectEpics(proj.ID); metas != nil {
		missing := []string{}
		for _, m := range metas {
			if _, known := epics[m.Key]; !known {
				missing = append(missing, m.Key)
			}
		}
		if len(missing) > 0 {
			if extra, err := d.runner.FetchJiraIssuesByKeys(settings, proj.TrackerUrl, missing, projectRepoPath(proj)); err == nil {
				for key, epic := range extra {
					epics[key] = epic
				}
			}
		}
	}

	classified := 0
	closed := 0
	for key, epic := range epics {
		horizon := HorizonFromLabels(epic.Labels)
		var horizonPtr *string
		if horizon != "" {
			horizonPtr = &horizon
			classified++
		}
		// Le titre et le statut viennent du ticket épic : sans eux, la roadmap
		// devinait le titre depuis les enfants et ne pouvait pas savoir qu'un
		// épic était terminé.
		title := epic.Summary
		status := epic.StatusName
		isClosed := strings.EqualFold(epic.StatusCategory, "done")
		if isClosed {
			closed++
		}
		_, _ = d.saveEpicMetaFull(proj.ID, key, horizonPtr, nil, nil, &title, &status, &isClosed)
	}
	return fmt.Sprintf("%d épics lus (%d classés, %d terminés)", len(epics), classified, closed), nil
}

// PendingHorizonPushes lists the epics classified locally whose Jira epic does
// not carry the matching roadmap label yet. That happens for anything classified
// before the label mirroring existed, and after a failed push.
func (d *DB) PendingHorizonPushes(projectID string) ([]models.EpicMeta, error) {
	metas, err := d.GetProjectEpics(projectID)
	if err != nil {
		return nil, err
	}
	classified := make([]models.EpicMeta, 0, len(metas))
	for _, m := range metas {
		if m.Horizon != "" {
			classified = append(classified, m)
		}
	}
	if len(classified) == 0 {
		return []models.EpicMeta{}, nil
	}

	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}
	projectKey := jiraProjectKeyFor(proj)
	if projectKey == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			projectKey = settings.JiraProject
		}
	}

	settings, _ := d.GetSettings()
	remote, err := d.runner.FetchJiraEpics(settings, proj.TrackerUrl, projectKey, projectRepoPath(proj))
	if err != nil {
		return nil, err
	}

	// Les épics d'un autre projet Jira ne sortent jamais de la requête par
	// projet : sans cette seconde lecture par clé, ils réapparaissaient
	// indéfiniment comme « à pousser », même après une poussée réussie.
	missing := []string{}
	for _, m := range classified {
		if _, known := remote[m.Key]; !known {
			missing = append(missing, m.Key)
		}
	}
	if len(missing) > 0 {
		if extra, err := d.runner.FetchJiraIssuesByKeys(settings, proj.TrackerUrl, missing, projectRepoPath(proj)); err == nil {
			for key, epic := range extra {
				remote[key] = epic
			}
		}
	}

	pending := []models.EpicMeta{}
	for _, m := range classified {
		epic, known := remote[m.Key]
		// Un épic d'un autre projet Jira n'est pas dans la requête : on le
		// considère à pousser, la poussée dira si elle est possible.
		if !known || HorizonFromLabels(epic.Labels) != m.Horizon {
			pending = append(pending, m)
		}
	}
	return pending, nil
}

// PushPendingHorizons mirrors every locally classified epic whose Jira label is
// missing or stale. Explicit rather than automatic: it edits a ticket per epic,
// which is not something a sync should decide on its own.
func (d *DB) PushPendingHorizons(projectID string) (int, []string, error) {
	pending, err := d.PendingHorizonPushes(projectID)
	if err != nil {
		return 0, nil, err
	}

	pushed := 0
	failures := []string{}
	for _, m := range pending {
		if _, err := d.PushEpicHorizonLabel(projectID, m.Key, m.Horizon); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", m.Key, err))
			continue
		}
		pushed++
	}
	return pushed, failures, nil
}

// SetTaskEpic queues the attachment of a ticket to an epic, or its detachment
// when epicKey is empty. The tracker call itself runs in the activity queue: it
// takes seconds, and a failure has to be readable rather than swallowed by an
// HTTP timeout. This is what makes an epic prototypable: pull existing tickets
// in, push the ones that do not belong out.
func (d *DB) SetTaskEpic(taskIDOrKey string, epicKey string) (*models.Task, *models.TaskActivity, error) {
	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil || task == nil {
		return nil, nil, fmt.Errorf("tâche non trouvée")
	}
	if task.Source != "jira" {
		return nil, nil, fmt.Errorf("le rattachement à un épic n'est disponible que sur un ticket Jira")
	}
	if _, err := d.jiraRESTClientForTask(task); err != nil {
		// acli n'a pas de --parent : sans jeton, l'opération est impossible et il
		// faut le dire tout de suite plutôt que de mettre en file un échec.
		return nil, nil, fmt.Errorf("rattachement impossible sans jeton d'API Jira : acli n'expose pas le champ parent")
	}

	// L'état local est écrit tout de suite, comme pour le sprint et l'équipe :
	// sans cela le ticket paraît toujours détaché tant que la file n'a pas tourné.
	if err := d.writeTaskParentLocally(task, epicKey); err != nil {
		return nil, nil, err
	}

	activity, err := d.EnqueueTrackerOp(TrackerOp{
		Kind:      TrackerOpSetParent,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		TaskKey:   task.Key,
		EpicKey:   strings.ToUpper(strings.TrimSpace(epicKey)),
	})
	if err != nil {
		return nil, nil, err
	}

	updated, err := d.GetTaskByID(task.ID)
	if err != nil {
		return nil, activity, err
	}
	return updated, activity, nil
}

// applyTaskEpic performs the attachment: the tracker write first, then the local
// mirror. It runs inside a queued job, never on an HTTP request.
func (d *DB) applyTaskEpic(taskIDOrKey string, epicKey string, steps *[]string) (*models.Task, error) {
	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil || task == nil {
		return nil, fmt.Errorf("tâche non trouvée")
	}
	if task.Source != "jira" {
		return nil, fmt.Errorf("le rattachement à un épic n'est disponible que sur un ticket Jira")
	}

	proj, _ := d.GetProjectByID(task.ProjectID)
	settings, _ := d.GetSettings()
	trackerURL := ""
	if proj != nil {
		trackerURL = proj.TrackerUrl
	}

	client := runner.NewJiraRESTClient(settings, trackerURL)
	if client == nil {
		return nil, fmt.Errorf("rattachement impossible sans jeton d'API Jira : acli n'expose pas le champ parent")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.SetIssueParent(ctx, task.Key, epicKey); err != nil {
		return nil, err
	}
	if steps != nil {
		if strings.TrimSpace(epicKey) == "" {
			*steps = append(*steps, fmt.Sprintf("✅ %s détaché de son épic sur Jira", task.Key))
		} else {
			*steps = append(*steps, fmt.Sprintf("✅ %s rattaché à %s sur Jira", task.Key, strings.ToUpper(strings.TrimSpace(epicKey))))
		}
	}

	if err := d.writeTaskParentLocally(task, epicKey); err != nil {
		return nil, err
	}

	return d.GetTaskByID(task.ID)
}

// writeTaskParentLocally mirrors the attachment in the local database. It runs
// twice for the same change: once when the write is queued, so the board reflects
// the intent immediately, and once when the tracker has accepted it. Writing it
// only in the worker left a ticket looking unattached for the seconds the queue
// took, and a triage view kept showing the line it had just handled.
func (d *DB) writeTaskParentLocally(task *models.Task, epicKey string) error {
	if task == nil {
		return fmt.Errorf("tâche manquante")
	}

	// Titre de l'épic : celui que la synchro a lu, sinon celui que portent déjà
	// ses autres enfants.
	key := strings.ToUpper(strings.TrimSpace(epicKey))
	parentTitle := ""
	parentType := ""
	if key != "" {
		parentType = "Epic"
		if metas, _ := d.GetProjectEpics(task.ProjectID); metas != nil {
			for _, m := range metas {
				if m.Key == key && m.Title != "" {
					parentTitle = m.Title
					break
				}
			}
		}
		if parentTitle == "" {
			d.mu.RLock()
			_ = d.conn.QueryRow("SELECT parent_title FROM tasks WHERE parent_key = ? AND parent_title != '' LIMIT 1", key).Scan(&parentTitle)
			d.mu.RUnlock()
		}
	}

	d.mu.Lock()
	_, err := d.conn.Exec(
		"UPDATE tasks SET parent_key = ?, parent_title = ?, parent_type = ?, updated_at = ? WHERE id = ?",
		key, parentTitle, parentType, time.Now(), task.ID,
	)
	d.mu.Unlock()
	return err
}

// CreateStoryUnderEpic creates a story under an epic from a plain title, for the
// tickets a triage session decides to add on the spot.
func (d *DB) CreateStoryUnderEpic(projectID string, epicKey string, title string) (*models.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("intitulé manquant")
	}

	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}
	if proj.IssueTracker != "jira" {
		return nil, fmt.Errorf("création de story disponible sur un projet Jira uniquement")
	}

	projectKey := jiraProjectKeyFor(proj)
	if projectKey == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			projectKey = settings.JiraProject
		}
	}

	key, err := d.runner.CreateJiraChildIssue(projectKey, projectRepoPath(proj), epicKey, "Story", title, "", nil)
	if err != nil {
		return nil, err
	}

	parentTitle := ""
	if metas, _ := d.GetProjectEpics(projectID); metas != nil {
		for _, m := range metas {
			if m.Key == epicKey {
				parentTitle = m.Title
				break
			}
		}
	}

	task := models.Task{
		ID:          "jira-" + key,
		ProjectID:   proj.ID,
		Key:         key,
		Title:       title,
		Status:      models.StatusToClarify,
		Priority:    models.PriorityMedium,
		Source:      "jira",
		IssueType:   "Story",
		ParentKey:   epicKey,
		ParentTitle: parentTitle,
		ParentType:  "Epic",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := d.ImportOrUpdateTasks([]models.Task{task}); err != nil {
		return nil, fmt.Errorf("%s créé dans Jira mais non inséré localement: %w", key, err)
	}
	return d.GetTaskByID(task.ID)
}

// CreateEpic creates the epic in the tracker and records it locally so it shows
// up in the roadmap immediately, before any sync — that is what makes it usable
// as a target for a split.
func (d *DB) CreateEpic(projectID string, title string, horizon string, fields map[string]string) (*models.EpicMeta, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("intitulé de l'épic manquant")
	}

	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}
	if proj.IssueTracker != "jira" {
		return nil, fmt.Errorf("création d'épic disponible sur un projet Jira uniquement")
	}

	projectKey := jiraProjectKeyFor(proj)
	if projectKey == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			projectKey = settings.JiraProject
		}
	}

	settings, _ := d.GetSettings()
	key, err := d.runner.CreateJiraEpicWith(settings, proj.TrackerUrl, projectKey, projectRepoPath(proj), title, fields)
	if err != nil {
		return nil, err
	}

	open := false
	status := "Open"
	normalized := normalizeHorizon(horizon)
	var horizonPtr *string
	if normalized != "" {
		horizonPtr = &normalized
	}
	meta, err := d.saveEpicMetaFull(proj.ID, key, horizonPtr, nil, nil, &title, &status, &open)
	if err != nil {
		return nil, err
	}

	if horizonPtr != nil {
		// Le label part dans la file d'activités, comme tout classement : une
		// goroutine anonyme n'écrivait son échec que dans le log du serveur, là
		// où personne ne le lit.
		if _, opErr := d.EnqueueTrackerOp(TrackerOp{
			Kind:      TrackerOpEpicHorizon,
			ProjectID: proj.ID,
			TaskKey:   key,
			EpicKey:   key,
			Horizon:   normalized,
		}); opErr != nil {
			log.Printf("[epics] label roadmap de %s non mis en file: %v", key, opErr)
		}
	}
	return meta, nil
}

// MoveTasksToEpic queues the epic split: a batch of tickets moved to another
// epic, created on the fly when only a title is given. One tracker call per
// ticket plus a possible epic creation is well past what an HTTP request should
// hold, so the whole batch runs as a single activity whose steps say what
// happened to each ticket.
func (d *DB) MoveTasksToEpic(projectID string, taskIDs []string, targetEpicKey string, newEpicTitle string, fields map[string]string) (*models.TaskActivity, error) {
	if len(taskIDs) == 0 {
		return nil, fmt.Errorf("aucun ticket sélectionné")
	}

	targetEpicKey = strings.ToUpper(strings.TrimSpace(targetEpicKey))
	if targetEpicKey == "" && strings.TrimSpace(newEpicTitle) == "" {
		return nil, fmt.Errorf("épic cible ou intitulé du nouvel épic obligatoire")
	}

	// Cible connue : l'état local suit tout de suite. Sur un épic encore à créer,
	// sa clé n'existe pas avant que la file ne tourne, et les tickets ne bougent
	// donc qu'à ce moment là.
	if targetEpicKey != "" {
		for _, id := range taskIDs {
			if task, err := d.GetTaskByID(id); err == nil && task != nil {
				_ = d.writeTaskParentLocally(task, targetEpicKey)
			}
		}
	}

	return d.EnqueueTrackerOp(TrackerOp{
		Kind:         TrackerOpMoveToEpic,
		ProjectID:    projectID,
		TaskIDs:      taskIDs,
		EpicKey:      targetEpicKey,
		NewEpicTitle: strings.TrimSpace(newEpicTitle),
		Fields:       fields,
	})
}

// appendActivityStep adds a line to an activity's step list, to trace what the
// autonomous chain decided without inventing a second log.
func (d *DB) appendActivityStep(activityID string, step string) {
	if strings.TrimSpace(activityID) == "" || strings.TrimSpace(step) == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	var raw string
	if err := d.conn.QueryRow("SELECT steps FROM task_activities WHERE id = ?", activityID).Scan(&raw); err != nil {
		return
	}
	steps := []string{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &steps)
	}
	steps = append(steps, step)
	payload, err := json.Marshal(steps)
	if err != nil {
		return
	}
	_, _ = d.conn.Exec("UPDATE task_activities SET steps = ? WHERE id = ?", string(payload), activityID)
}

// EpicRequiredFields lists the creation fields the tracker imposes for an epic on
// this project.
func (d *DB) EpicRequiredFields(projectID string) ([]runner.JiraRequiredField, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}
	if proj.IssueTracker != "jira" {
		return []runner.JiraRequiredField{}, nil
	}

	projectKey := jiraProjectKeyFor(proj)
	if projectKey == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			projectKey = settings.JiraProject
		}
	}
	settings, _ := d.GetSettings()
	return d.runner.EpicRequiredFields(settings, proj.TrackerUrl, projectKey)
}
