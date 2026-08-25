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
)

func normalizeHorizon(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case HorizonNow:
		return HorizonNow
	case HorizonNext:
		return HorizonNext
	case HorizonLater, "future":
		return HorizonLater
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
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (project_id, key)
	);`)
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_epics_project ON epics(project_id, horizon);")
}

// GetProjectEpics returns the epic metadata of a project, keyed by epic key.
func (d *DB) GetProjectEpics(projectID string) ([]models.EpicMeta, error) {
	d.mu.Lock()
	d.ensureEpicsTable()
	d.mu.Unlock()

	d.mu.RLock()
	rows, err := d.conn.Query(`
		SELECT project_id, key, horizon, description, todos, updated_at
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
		if err := rows.Scan(&e.ProjectID, &e.Key, &e.Horizon, &e.Description, &todosJSON, &e.UpdatedAt); err != nil {
			continue
		}
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
	projectID = strings.TrimSpace(projectID)
	key = strings.TrimSpace(key)
	if projectID == "" || key == "" {
		return nil, fmt.Errorf("projet et clé d'épic obligatoires")
	}

	d.mu.Lock()
	d.ensureEpicsTable()

	current := models.EpicMeta{ProjectID: projectID, Key: key, Todos: []models.EpicTodo{}}
	var todosJSON string
	err := d.conn.QueryRow(`
		SELECT horizon, description, todos FROM epics WHERE project_id = ? AND key = ?
	`, projectID, key).Scan(&current.Horizon, &current.Description, &todosJSON)
	if err == nil {
		current.Todos = parseEpicTodos(todosJSON)
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
	_, execErr := d.conn.Exec(`
		INSERT INTO epics (project_id, key, horizon, description, todos, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, key) DO UPDATE SET
			horizon = excluded.horizon,
			description = excluded.description,
			todos = excluded.todos,
			updated_at = excluded.updated_at
	`, projectID, key, current.Horizon, current.Description, string(payload), current.UpdatedAt)
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
