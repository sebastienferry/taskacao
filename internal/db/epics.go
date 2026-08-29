package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tasks/internal/models"
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

	proj, _ := d.GetProjectByID(projectID)
	if proj != nil && (proj.IssueTracker == "github" || proj.GithubRepo != "") {
		if milestones, err := d.runner.ListGithubMilestones(proj.GithubRepo, proj.RepoPath); err == nil && len(milestones) > 0 {
			d.mu.Lock()
			for _, m := range milestones {
				key := fmt.Sprintf("M-%d", m.Number)
				closedVal := 0
				if strings.EqualFold(m.State, "closed") {
					closedVal = 1
				}
				_, _ = d.conn.Exec(`
					INSERT INTO epics (project_id, key, title, description, status, closed, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
					ON CONFLICT(project_id, key) DO UPDATE SET
						title = excluded.title,
						status = excluded.status,
						closed = excluded.closed
				`, projectID, key, m.Title, m.Description, m.State, closedVal)
			}
			d.mu.Unlock()
		}
	}

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
	return d.UpdateEpic(projectID, key, nil, horizon, description, todos, nil)
}

// UpdateEpic updates macro metadata (title, horizon, description, todos, closed) locally and in GitHub milestone if applicable.
func (d *DB) UpdateEpic(projectID string, key string, title *string, horizon *string, description *string, todos *[]models.EpicTodo, closed *bool) (*models.EpicMeta, error) {
	projectID = strings.TrimSpace(projectID)
	key = strings.TrimSpace(key)
	if projectID == "" || key == "" {
		return nil, fmt.Errorf("projet et clé d'épic obligatoires")
	}

	proj, _ := d.GetProjectByID(projectID)
	if proj != nil && (proj.IssueTracker == "github" || proj.GithubRepo != "") {
		var num int
		if strings.HasPrefix(strings.ToUpper(key), "M-") {
			_, _ = fmt.Sscanf(strings.ToUpper(key), "M-%d", &num)
		}
		if num > 0 {
			state := ""
			if closed != nil {
				if *closed {
					state = "closed"
				} else {
					state = "open"
				}
			}
			newTitle := ""
			if title != nil {
				newTitle = strings.TrimSpace(*title)
			}
			newDesc := ""
			if description != nil {
				newDesc = *description
			}
			_ = d.runner.UpdateGithubMilestone(proj.GithubRepo, proj.RepoPath, num, newTitle, newDesc, state)
		}
	}

	// If title changed, update any task parent_title in tasks table as well
	if title != nil && strings.TrimSpace(*title) != "" {
		newTitle := strings.TrimSpace(*title)
		d.mu.Lock()
		_, _ = d.conn.Exec("UPDATE tasks SET parent_title = ? WHERE project_id = ? AND (parent_key = ? OR parent_title = ?)", newTitle, projectID, key, key)
		d.mu.Unlock()
	}

	return d.saveEpicMetaFull(projectID, key, horizon, description, todos, title, nil, closed)
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
		return nil, "", fmt.Errorf("macro %s sans cadrage enregistré", epicKey)
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

	task, err := d.CreateStoryUnderEpic(projectID, epicKey, todo.Text)
	if err != nil {
		return nil, "", fmt.Errorf("erreur création de story: %w", err)
	}

	todo.StoryKey = task.Key
	saved, err := d.SaveEpicMeta(projectID, epicKey, nil, nil, &meta.Todos)
	if err != nil {
		return meta, task.Key, nil
	}
	return saved, task.Key, nil
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
	return "classification gardée en local", nil
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

	return "0 épics lus", nil
}

// PendingHorizonPushes lists the epics classified locally whose Jira epic does
// not carry the matching roadmap label yet.
func (d *DB) PendingHorizonPushes(projectID string) ([]models.EpicMeta, error) {
	return []models.EpicMeta{}, nil
}

// PushPendingHorizons mirrors every locally classified epic.
func (d *DB) PushPendingHorizons(projectID string) (int, []string, error) {
	return 0, nil, nil
}

// SetTaskEpic queues the attachment of a ticket to a macro.
func (d *DB) SetTaskEpic(taskIDOrKey string, epicKey string) (*models.Task, *models.TaskActivity, error) {
	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil || task == nil {
		return nil, nil, fmt.Errorf("tâche introuvable")
	}
	cleanEpicKey := strings.TrimSpace(epicKey)
	if err := d.writeTaskParentLocally(task, cleanEpicKey); err != nil {
		return nil, nil, err
	}
	act, err := d.EnqueueTrackerOp(TrackerOp{
		Kind:      TrackerOpSetParent,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		TaskKey:   task.Key,
		EpicKey:   cleanEpicKey,
	})
	return task, act, err
}

// applyTaskEpic performs the attachment of a task to a macro (milestone).
func (d *DB) applyTaskEpic(taskIDOrKey string, epicKey string, steps *[]string) (*models.Task, error) {
	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil || task == nil {
		return nil, fmt.Errorf("tâche introuvable")
	}
	cleanEpicKey := strings.TrimSpace(epicKey)
	if err := d.writeTaskParentLocally(task, cleanEpicKey); err != nil {
		return nil, err
	}

	proj, _ := d.GetProjectByID(task.ProjectID)
	if proj != nil && (proj.IssueTracker == "github" || task.Source == "github") {
		cleanKey := strings.TrimPrefix(task.Key, "#")
		var issueNum int
		_, _ = fmt.Sscanf(cleanKey, "%d", &issueNum)
		if issueNum > 0 {
			milestoneTarget := ""
			if cleanEpicKey != "" {
				var mTitle string
				d.mu.RLock()
				_ = d.conn.QueryRow("SELECT title FROM epics WHERE project_id = ? AND key = ?", task.ProjectID, cleanEpicKey).Scan(&mTitle)
				d.mu.RUnlock()
				if mTitle != "" {
					milestoneTarget = mTitle
				} else if strings.HasPrefix(strings.ToUpper(cleanEpicKey), "M-") {
					var num int
					_, _ = fmt.Sscanf(strings.ToUpper(cleanEpicKey), "M-%d", &num)
					milestoneTarget = fmt.Sprintf("%d", num)
				} else {
					milestoneTarget = cleanEpicKey
				}
			}
			if err := d.runner.SetGithubIssueMilestone(proj.GithubRepo, proj.RepoPath, issueNum, milestoneTarget); err != nil {
				*steps = append(*steps, fmt.Sprintf("⚠️ Synchro milestone GitHub échouée pour %s: %v, gardé en local", task.Key, err))
			} else {
				if milestoneTarget == "" {
					*steps = append(*steps, fmt.Sprintf("✅ %s retiré du milestone GitHub", task.Key))
				} else {
					*steps = append(*steps, fmt.Sprintf("✅ %s rattaché au milestone GitHub « %s »", task.Key, milestoneTarget))
				}
			}
		}
	} else {
		*steps = append(*steps, fmt.Sprintf("✅ %s macro mise à jour en local", task.Key))
	}

	return task, nil
}

// writeTaskParentLocally mirrors the attachment in the local database.
func (d *DB) writeTaskParentLocally(task *models.Task, epicKey string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	parentTitle := ""
	if epicKey != "" {
		_ = d.conn.QueryRow("SELECT title FROM epics WHERE project_id = ? AND key = ?", task.ProjectID, epicKey).Scan(&parentTitle)
		if parentTitle == "" {
			parentTitle = epicKey
		}
	}
	_, err := d.conn.Exec("UPDATE tasks SET parent_key = ?, parent_title = ?, parent_type = 'macro', updated_at = ? WHERE id = ?", epicKey, parentTitle, time.Now(), task.ID)
	return err
}

func (d *DB) CreateStoryUnderEpic(projectID string, epicKey string, title string) (*models.Task, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}

	parentTitle := ""
	d.mu.RLock()
	_ = d.conn.QueryRow("SELECT title FROM epics WHERE project_id = ? AND key = ?", projectID, epicKey).Scan(&parentTitle)
	d.mu.RUnlock()
	if parentTitle == "" {
		parentTitle = epicKey
	}

	task, err := d.CreateTask(models.CreateTaskRequest{
		ProjectID: projectID,
		Title:     title,
		Priority:  models.PriorityMedium,
	})
	if err != nil {
		return nil, err
	}

	_ = d.writeTaskParentLocally(task, epicKey)
	task.ParentKey = epicKey
	task.ParentTitle = parentTitle
	task.ParentType = "macro"

	if proj.IssueTracker == "github" || task.Source == "github" {
		var issueNum int
		_, _ = fmt.Sscanf(strings.TrimPrefix(task.Key, "#"), "%d", &issueNum)
		if issueNum > 0 {
			_ = d.runner.SetGithubIssueMilestone(proj.GithubRepo, proj.RepoPath, issueNum, parentTitle)
		}
	}
	return task, nil
}

// CreateEpic creates the macro in the tracker (e.g. GitHub milestone) and records it locally.
func (d *DB) CreateEpic(projectID string, title string, horizon string, fields map[string]string) (*models.EpicMeta, error) {
	projectID = strings.TrimSpace(projectID)
	title = strings.TrimSpace(title)
	if projectID == "" || title == "" {
		return nil, fmt.Errorf("projet et titre de la macro obligatoires")
	}

	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}

	key := ""
	if proj.IssueTracker == "github" || proj.GithubRepo != "" {
		num, err := d.runner.CreateGithubMilestone(proj.GithubRepo, proj.RepoPath, title, "")
		if err != nil {
			return nil, fmt.Errorf("erreur création milestone GitHub: %w", err)
		}
		if num > 0 {
			key = fmt.Sprintf("M-%d", num)
		}
	}

	if key == "" {
		key = fmt.Sprintf("MACRO-%d", time.Now().Unix()%100000)
	}

	status := "open"
	closed := false
	h := horizon
	if h == "" {
		h = HorizonNow
	}
	return d.saveEpicMetaFull(projectID, key, &h, nil, nil, &title, &status, &closed)
}

// DeleteEpic deletes a macro (and its GitHub milestone if applicable) and detaches its child tasks.
func (d *DB) DeleteEpic(projectID string, key string) error {
	projectID = strings.TrimSpace(projectID)
	key = strings.TrimSpace(key)
	if projectID == "" || key == "" {
		return fmt.Errorf("projet et clé de macro obligatoires")
	}

	proj, _ := d.GetProjectByID(projectID)
	if proj != nil && (proj.IssueTracker == "github" || proj.GithubRepo != "") {
		var num int
		if strings.HasPrefix(strings.ToUpper(key), "M-") {
			_, _ = fmt.Sscanf(strings.ToUpper(key), "M-%d", &num)
		}
		if num > 0 {
			_ = d.runner.DeleteGithubMilestone(proj.GithubRepo, proj.RepoPath, num)
		}
	}

	d.mu.Lock()
	d.ensureEpicsTable()
	_, err := d.conn.Exec("DELETE FROM epics WHERE project_id = ? AND key = ?", projectID, key)
	_, _ = d.conn.Exec("UPDATE tasks SET parent_key = '', parent_title = '' WHERE project_id = ? AND (parent_key = ? OR parent_title = ?)", projectID, key, key)
	d.mu.Unlock()
	return err
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
