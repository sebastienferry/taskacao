package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"tasks/internal/models"
	"tasks/internal/runner"
)

type DB struct {
	conn   *sql.DB
	runner *runner.Runner
	mu     sync.RWMutex
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite single-writer safety

	db := &DB{
		conn:   conn,
		runner: runner.NewRunner(),
	}
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := db.seedIfEmpty(); err != nil {
		log.Printf("Warning: error seeding default data: %v", err)
	}

	return db, nil
}

func (d *DB) GetRunner() *runner.Runner {
	return d.runner
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			theme TEXT NOT NULL DEFAULT 'dark',
			accent_color TEXT NOT NULL DEFAULT 'indigo',
			language TEXT NOT NULL DEFAULT 'fr',
			density TEXT NOT NULL DEFAULT 'standard',
			default_view TEXT NOT NULL DEFAULT 'board',
			detail_mode TEXT NOT NULL DEFAULT 'panel',
			user_name TEXT NOT NULL DEFAULT 'Sylvain Ferry',
			user_email TEXT NOT NULL DEFAULT 'sylvain@fretzee.com',
			user_avatar TEXT NOT NULL DEFAULT '',
			ai_provider TEXT NOT NULL DEFAULT 'agy',
			ai_command_template TEXT NOT NULL DEFAULT 'agy -p "{prompt}"',
			repo_path TEXT NOT NULL DEFAULT '/Users/sferry/Sources/fretzee-studio',
			issue_tracker TEXT NOT NULL DEFAULT 'linear',
			linear_team TEXT NOT NULL DEFAULT 'FRE',
			github_repo TEXT NOT NULL DEFAULT 'sebastienferry/fretzee-studio',
			prompt_clarify TEXT NOT NULL DEFAULT '',
			prompt_specify TEXT NOT NULL DEFAULT '',
			prompt_implement TEXT NOT NULL DEFAULT '',
			prompt_create_pr TEXT NOT NULL DEFAULT '',
			prompt_pick TEXT NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'backlog',
			priority TEXT NOT NULL DEFAULT 'medium',
			labels TEXT NOT NULL DEFAULT '[]',
			assignee TEXT NOT NULL DEFAULT '',
			assignee_avatar TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			due_date TEXT,
			branch_name TEXT,
			pr_url TEXT,
			source TEXT NOT NULL DEFAULT 'local',
			external_url TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS task_activities (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			skill_id TEXT NOT NULL,
			skill_name TEXT NOT NULL,
			action TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'completed',
			summary TEXT NOT NULL DEFAULT '',
			output TEXT NOT NULL DEFAULT '',
			steps TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(status, position);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_task ON task_activities(task_id, created_at DESC);`,
	}

	for _, query := range queries {
		if _, err := d.conn.Exec(query); err != nil {
			return err
		}
	}

	// Migrations for existing database instances
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN branch_name TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN pr_url TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN source TEXT NOT NULL DEFAULT 'local';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN external_url TEXT;")

	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN detail_mode TEXT NOT NULL DEFAULT 'panel';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN ai_provider TEXT NOT NULL DEFAULT 'agy';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN ai_command_template TEXT NOT NULL DEFAULT 'agy -p \"{prompt}\"';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN repo_path TEXT NOT NULL DEFAULT '/Users/sferry/Sources/fretzee-studio';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN issue_tracker TEXT NOT NULL DEFAULT 'linear';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN linear_team TEXT NOT NULL DEFAULT 'FRE';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN github_repo TEXT NOT NULL DEFAULT 'sebastienferry/fretzee-studio';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_clarify TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_specify TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_implement TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_create_pr TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_pick TEXT NOT NULL DEFAULT '';")

	// Migrate legacy stage names to 5-stage workflow:
	// A clarifier (to_clarify) -> A spécifier (to_specify) -> A implémenter (to_implement) -> A tester (to_test) -> A fermer (to_close)
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_clarify' WHERE status = 'backlog';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_specify' WHERE status = 'specified';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_implement' WHERE status = 'in_progress';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_test' WHERE status = 'to_validate';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_close' WHERE status = 'done';")

	return nil
}

func (d *DB) seedIfEmpty() error {
	var settingsCount int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM settings WHERE id = 1").Scan(&settingsCount)
	if err != nil {
		return err
	}
	if settingsCount == 0 {
		_, err = d.conn.Exec(`
			INSERT INTO settings (id, theme, accent_color, language, density, default_view, user_name, user_email, user_avatar, ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo)
			VALUES (1, 'dark', 'indigo', 'fr', 'standard', 'board', 'Sylvain Ferry', 'sylvain@fretzee.com', '', 'agy', 'agy -p "{prompt}"', '/Users/sferry/Sources/fretzee-studio', 'linear', 'FRE', 'sebastienferry/fretzee-studio')
		`)
		if err != nil {
			return err
		}
	}

	var count int
	err = d.conn.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return d.SeedDemoData()
	}

	return nil
}

func (d *DB) SeedDemoData() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Ensure default settings
	_, err := d.conn.Exec(`
		INSERT INTO settings (id, theme, accent_color, language, density, default_view, user_name, user_email, user_avatar, ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo)
		VALUES (1, 'dark', 'indigo', 'fr', 'standard', 'board', 'Sylvain Ferry', 'sylvain@fretzee.com', '', 'agy', 'agy -p "{prompt}"', '/Users/sferry/Sources/fretzee-studio', 'linear', 'FRE', 'sebastienferry/fretzee-studio')
		ON CONFLICT(id) DO UPDATE SET updated_at = CURRENT_TIMESTAMP;
	`)
	if err != nil {
		return err
	}

	// Clean tasks and activities for fresh sync
	if _, err := d.conn.Exec("DELETE FROM task_activities"); err != nil {
		return err
	}
	if _, err := d.conn.Exec("DELETE FROM tasks"); err != nil {
		return err
	}

	// Live sync from Linear
	linTasks, err := d.runner.SyncFromLinear("FRE")
	if err == nil && len(linTasks) > 0 {
		now := time.Now()
		for _, t := range linTasks {
			labelsJSON, _ := json.Marshal(t.Labels)
			_, _ = d.conn.Exec(`
				INSERT INTO tasks (id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, t.ID, t.Key, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.AssigneeAvatar, t.Position, t.DueDate, t.BranchName, t.PrURL, t.Source, t.ExternalURL, t.CreatedAt, now)
		}
	}

	// Live sync from GitHub
	ghTasks, err := d.runner.SyncFromGithub("sebastienferry/fretzee-studio", "/Users/sferry/Sources/fretzee-studio")
	if err == nil && len(ghTasks) > 0 {
		now := time.Now()
		for _, t := range ghTasks {
			labelsJSON, _ := json.Marshal(t.Labels)
			_, _ = d.conn.Exec(`
				INSERT INTO tasks (id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, t.ID, t.Key, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.AssigneeAvatar, t.Position, t.DueDate, t.BranchName, t.PrURL, t.Source, t.ExternalURL, t.CreatedAt, now)
		}
	}

	return nil
}

func (d *DB) ImportOrUpdateTasks(syncedTasks []models.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for _, t := range syncedTasks {
		labelsJSON, _ := json.Marshal(t.Labels)
		var existingID string
		err := d.conn.QueryRow("SELECT id FROM tasks WHERE key = ? OR id = ?", t.Key, t.ID).Scan(&existingID)

		if err == sql.ErrNoRows {
			// Insert new task
			newID := t.ID
			if newID == "" {
				newID = uuid.New().String()
			}
			_, _ = d.conn.Exec(`
				INSERT INTO tasks (id, key, title, description, status, priority, labels, assignee, position, due_date, source, external_url, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newID, t.Key, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.Position, t.DueDate, t.Source, t.ExternalURL, t.CreatedAt, now)
		} else if err == nil {
			// Update existing task title/desc/status/labels
			_, _ = d.conn.Exec(`
				UPDATE tasks
				SET title = ?, description = ?, status = ?, priority = ?, labels = ?, assignee = ?, external_url = ?, updated_at = ?
				WHERE id = ?
			`, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.ExternalURL, now, existingID)
		}
	}
	return nil
}

func (d *DB) GetTasks(query, status, priority, label string) ([]models.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var conditions []string
	var args []interface{}

	if query != "" {
		conditions = append(conditions, "(key LIKE ? OR title LIKE ? OR description LIKE ? OR labels LIKE ? OR assignee LIKE ?)")
		pattern := "%" + query + "%"
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}

	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	if priority != "" {
		conditions = append(conditions, "priority = ?")
		args = append(args, priority)
	}

	if label != "" {
		conditions = append(conditions, "labels LIKE ?")
		args = append(args, "%"+label+"%")
	}

	sqlQuery := "SELECT id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at FROM tasks"
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	sqlQuery += " ORDER BY status, position ASC, created_at DESC"

	rows, err := d.conn.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var labelsJSON string
		var dueDate, branchName, prURL, source, extURL sql.NullString
		var statusStr, priorityStr string

		err := rows.Scan(
			&t.ID,
			&t.Key,
			&t.Title,
			&t.Description,
			&statusStr,
			&priorityStr,
			&labelsJSON,
			&t.Assignee,
			&t.AssigneeAvatar,
			&t.Position,
			&dueDate,
			&branchName,
			&prURL,
			&source,
			&extURL,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		t.Status = models.Status(statusStr)
		t.Priority = models.Priority(priorityStr)
		if dueDate.Valid {
			t.DueDate = &dueDate.String
		}
		if branchName.Valid {
			t.BranchName = &branchName.String
		}
		if prURL.Valid {
			t.PrURL = &prURL.String
		}
		if extURL.Valid && extURL.String != "" {
			t.ExternalURL = &extURL.String
		} else if t.Source == "linear" {
			url := fmt.Sprintf("https://linear.app/fretzee/issue/%s", t.Key)
			t.ExternalURL = &url
		} else if t.Source == "github" {
			cleanNum := strings.TrimPrefix(t.Key, "GH-#")
			cleanNum = strings.TrimPrefix(cleanNum, "gh-")
			url := fmt.Sprintf("https://github.com/sebastienferry/fretzee-studio/issues/%s", cleanNum)
			t.ExternalURL = &url
		}

		_ = json.Unmarshal([]byte(labelsJSON), &t.Labels)
		if t.Labels == nil {
			t.Labels = []string{}
		}

		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	return tasks, nil
}

func (d *DB) GetTaskByID(id string) (*models.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var t models.Task
	var labelsJSON string
	var dueDate, branchName, prURL, source, extURL sql.NullString
	var statusStr, priorityStr string

	err := d.conn.QueryRow(`
		SELECT id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at
		FROM tasks WHERE id = ? OR key = ?
	`, id, id).Scan(
		&t.ID,
		&t.Key,
		&t.Title,
		&t.Description,
		&statusStr,
		&priorityStr,
		&labelsJSON,
		&t.Assignee,
		&t.AssigneeAvatar,
		&t.Position,
		&dueDate,
		&branchName,
		&prURL,
		&source,
		&extURL,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	t.Status = models.Status(statusStr)
	t.Priority = models.Priority(priorityStr)
	if dueDate.Valid {
		t.DueDate = &dueDate.String
	}
	if branchName.Valid {
		t.BranchName = &branchName.String
	}
	if prURL.Valid {
		t.PrURL = &prURL.String
	}
	if source.Valid {
		t.Source = source.String
	}
	if extURL.Valid && extURL.String != "" {
		t.ExternalURL = &extURL.String
	} else if t.Source == "linear" {
		url := fmt.Sprintf("https://linear.app/fretzee/issue/%s", t.Key)
		t.ExternalURL = &url
	} else if t.Source == "github" {
		cleanNum := strings.TrimPrefix(t.Key, "GH-#")
		cleanNum = strings.TrimPrefix(cleanNum, "gh-")
		url := fmt.Sprintf("https://github.com/sebastienferry/fretzee-studio/issues/%s", cleanNum)
		t.ExternalURL = &url
	}
	_ = json.Unmarshal([]byte(labelsJSON), &t.Labels)
	if t.Labels == nil {
		t.Labels = []string{}
	}

	activities, _ := d.getTaskActivitiesUnsafe(t.ID)
	t.Activities = activities

	return &t, nil
}

func (d *DB) getNextTaskKey() (string, error) {
	rows, err := d.conn.Query("SELECT key FROM tasks")
	if err != nil {
		return "FRE-1", nil
	}
	defer rows.Close()

	maxNum := 0
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			var num int
			if _, err := fmt.Sscanf(k, "FRE-%d", &num); err == nil {
				if num > maxNum {
					maxNum = num
				}
			}
		}
	}

	if maxNum == 0 {
		maxNum = 100
	}
	return fmt.Sprintf("FRE-%d", maxNum+1), nil
}

// Workflow labels following the lifecycle:
// Create -> Clarify -> Specify -> Implement -> Review
// Label after action: New -> Clarified -> Specified -> Implemented -> Reviewed
var WorkflowLabels = []string{"New", "Clarified", "Specified", "Implemented", "Reviewed"}

func GetStageLabelForStatus(status models.Status) string {
	switch status {
	case models.StatusToClarify, models.StatusBacklog:
		return "New"
	case models.StatusToSpecify, models.StatusSpecified:
		return "Clarified"
	case models.StatusToImplement, models.StatusInProgress:
		return "Specified"
	case models.StatusToTest, models.StatusToValidate:
		return "Implemented"
	case models.StatusToClose, models.StatusDone:
		return "Reviewed"
	default:
		return "New"
	}
}

func SetWorkflowLabel(existingLabels []string, targetLabel string) []string {
	var result []string
	for _, l := range existingLabels {
		isWorkflow := false
		for _, wl := range []string{"New", "Clarified", "Specified", "Implemented", "Reviewed", "new", "clarified", "specified", "implemented", "reviewed"} {
			if strings.EqualFold(l, wl) {
				isWorkflow = true
				break
			}
		}
		if !isWorkflow {
			result = append(result, l)
		}
	}
	if targetLabel != "" {
		result = append(result, targetLabel)
	}
	return result
}

func (d *DB) CreateTask(req models.CreateTaskRequest) (*models.Task, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	settings, _ := d.getSettingsUnsafe()
	if settings == nil {
		settings = &models.Settings{
			LinearTeam: "FRE",
			GithubRepo: "sebastienferry/fretzee-studio",
			RepoPath:   "/Users/sferry/Sources/fretzee-studio",
		}
	}

	// Action Create -> Status: to_clarify, Label: New
	if req.Status == "" {
		req.Status = models.StatusToClarify
	}
	req.Labels = SetWorkflowLabel(req.Labels, "New")

	var key string
	var extURL *string
	var id string = uuid.New().String()
	now := time.Now()

	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}
	if req.Source == "" {
		req.Source = "local"
	}

	// Real creation via CLI if Linear or GitHub requested
	if req.Source == "linear" {
		created, err := d.runner.CreateLinearIssue(settings.LinearTeam, req.Title, req.Description, req.Priority, req.Labels)
		if err == nil && created != nil {
			id = created.ID
			key = created.Key
			extURL = created.ExternalURL
		} else {
			// Fallback local key if CLI fails
			key, _ = d.getNextTaskKey()
		}
	} else if req.Source == "github" {
		created, err := d.runner.CreateGithubIssue(settings.GithubRepo, settings.RepoPath, req.Title, req.Description, req.Labels)
		if err == nil && created != nil {
			id = created.ID
			key = created.Key
			extURL = created.ExternalURL
		} else {
			key, _ = d.getNextTaskKey()
		}
	} else {
		key, _ = d.getNextTaskKey()
	}

	if req.ExternalURL != nil && *req.ExternalURL != "" {
		extURL = req.ExternalURL
	}

	var maxPos int
	_ = d.conn.QueryRow("SELECT COALESCE(MAX(position), -1) FROM tasks WHERE status = ?", req.Status).Scan(&maxPos)
	newPos := maxPos + 1

	labelsJSON, _ := json.Marshal(req.Labels)
	if req.Labels == nil {
		labelsJSON = []byte("[]")
	}

	_, err := d.conn.Exec(`
		INSERT INTO tasks (id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, source, external_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, key, req.Title, req.Description, string(req.Status), string(req.Priority), string(labelsJSON), req.Assignee, req.AssigneeAvatar, newPos, req.DueDate, req.Source, extURL, now, now)

	if err != nil {
		return nil, err
	}

	task := &models.Task{
		ID:             id,
		Key:            key,
		Title:          req.Title,
		Description:    req.Description,
		Status:         req.Status,
		Priority:       req.Priority,
		Labels:         req.Labels,
		Assignee:       req.Assignee,
		AssigneeAvatar: req.AssigneeAvatar,
		Position:       newPos,
		DueDate:        req.DueDate,
		Source:         req.Source,
		ExternalURL:    extURL,
		Activities:     []models.TaskActivity{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return task, nil
}

func (d *DB) UpdateTask(id string, req models.UpdateTaskRequest) (*models.Task, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	existing, err := d.getTaskByIDUnsafe(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("task not found")
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Status != nil {
		existing.Status = *req.Status
		if req.Labels == nil {
			existing.Labels = SetWorkflowLabel(existing.Labels, GetStageLabelForStatus(*req.Status))
		}
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Labels != nil {
		existing.Labels = *req.Labels
	}
	if req.Assignee != nil {
		existing.Assignee = *req.Assignee
	}
	if req.AssigneeAvatar != nil {
		existing.AssigneeAvatar = *req.AssigneeAvatar
	}
	if req.Position != nil {
		existing.Position = *req.Position
	}
	if req.DueDate != nil {
		existing.DueDate = req.DueDate
	}
	if req.BranchName != nil {
		existing.BranchName = req.BranchName
	}
	if req.PrURL != nil {
		existing.PrURL = req.PrURL
	}
	if req.Source != nil {
		existing.Source = *req.Source
	}
	if req.ExternalURL != nil {
		existing.ExternalURL = req.ExternalURL
	}
	existing.UpdatedAt = time.Now()

	labelsJSON, _ := json.Marshal(existing.Labels)

	_, err = d.conn.Exec(`
		UPDATE tasks
		SET title = ?, description = ?, status = ?, priority = ?, labels = ?, assignee = ?, assignee_avatar = ?, position = ?, due_date = ?, branch_name = ?, pr_url = ?, source = ?, external_url = ?, updated_at = ?
		WHERE id = ?
	`, existing.Title, existing.Description, string(existing.Status), string(existing.Priority), string(labelsJSON), existing.Assignee, existing.AssigneeAvatar, existing.Position, existing.DueDate, existing.BranchName, existing.PrURL, existing.Source, existing.ExternalURL, existing.UpdatedAt, id)

	if err != nil {
		return nil, err
	}

	// Background issue update via Linear / GitHub CLI
	settings, _ := d.getSettingsUnsafe()
	isLinear := existing.Source == "linear"
	isGithub := existing.Source == "github"

	if isLinear {
		go func(key string, t, desc *string, p *models.Priority, st *models.Status, lbls []string) {
			err := d.runner.UpdateLinearIssue(key, t, desc, p, st, lbls)
			if err != nil {
				log.Printf("[SYNC] Failed to update Linear issue %s: %v", key, err)
			} else {
				log.Printf("[SYNC] Successfully updated Linear issue %s via CLI", key)
			}
		}(existing.Key, req.Title, req.Description, req.Priority, req.Status, existing.Labels)
	} else if isGithub && settings != nil {
		go func(repo, rPath, key string, t, desc *string, st *models.Status, lbls []string) {
			err := d.runner.UpdateGithubIssue(repo, rPath, key, t, desc, st, lbls)
			if err != nil {
				log.Printf("[SYNC] Failed to update GitHub issue %s: %v", key, err)
			} else {
				log.Printf("[SYNC] Successfully updated GitHub issue %s via CLI", key)
			}
		}(settings.GithubRepo, settings.RepoPath, existing.Key, req.Title, req.Description, req.Status, existing.Labels)
	}

	acts, _ := d.getTaskActivitiesUnsafe(existing.ID)
	existing.Activities = acts

	return existing, nil
}

func (d *DB) getSettingsUnsafe() (*models.Settings, error) {
	var s models.Settings
	var detMode, aiProv, aiCmd, repoP, issTrk, linTm, ghRepo, pClar, pSpec, pImpl, pPR, pPick sql.NullString

	err := d.conn.QueryRow(`
		SELECT id, theme, accent_color, language, density, default_view, detail_mode, user_name, user_email, user_avatar,
		       ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo,
		       prompt_clarify, prompt_specify, prompt_implement, prompt_create_pr, prompt_pick, updated_at
		FROM settings WHERE id = 1
	`).Scan(
		&s.ID,
		&s.Theme,
		&s.AccentColor,
		&s.Language,
		&s.Density,
		&s.DefaultView,
		&detMode,
		&s.UserName,
		&s.UserEmail,
		&s.UserAvatar,
		&aiProv,
		&aiCmd,
		&repoP,
		&issTrk,
		&linTm,
		&ghRepo,
		&pClar,
		&pSpec,
		&pImpl,
		&pPR,
		&pPick,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if detMode.Valid {
		s.DetailMode = detMode.String
	} else {
		s.DetailMode = "panel"
	}
	if aiProv.Valid {
		s.AIProvider = aiProv.String
	}
	if aiCmd.Valid {
		s.AICommandTemplate = aiCmd.String
	}
	if repoP.Valid {
		s.RepoPath = repoP.String
	}
	if issTrk.Valid {
		s.IssueTracker = issTrk.String
	}
	if linTm.Valid {
		s.LinearTeam = linTm.String
	}
	if ghRepo.Valid {
		s.GithubRepo = ghRepo.String
	}
	if pClar.Valid {
		s.PromptClarify = pClar.String
	}
	if pSpec.Valid {
		s.PromptSpecify = pSpec.String
	}
	if pImpl.Valid {
		s.PromptImplement = pImpl.String
	}
	if pPR.Valid {
		s.PromptCreatePR = pPR.String
	}
	if pPick.Valid {
		s.PromptPick = pPick.String
	}
	return &s, nil
}

func (d *DB) MoveTask(id string, newStatus models.Status, newPosition int) (*models.Task, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	existing, err := d.getTaskByIDUnsafe(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("task not found")
	}

	now := time.Now()
	_, _ = d.conn.Exec(`
		UPDATE tasks
		SET position = position + 1
		WHERE status = ? AND position >= ? AND id != ?
	`, string(newStatus), newPosition, id)

	existing.Status = newStatus
	existing.Position = newPosition
	existing.Labels = SetWorkflowLabel(existing.Labels, GetStageLabelForStatus(newStatus))
	existing.UpdatedAt = now

	labelsJSON, _ := json.Marshal(existing.Labels)
	_, err = d.conn.Exec(`
		UPDATE tasks
		SET status = ?, labels = ?, position = ?, updated_at = ?
		WHERE id = ?
	`, string(newStatus), string(labelsJSON), newPosition, now, id)
	if err != nil {
		return nil, err
	}

	// Background state and label sync with Linear / GitHub CLI
	settings, _ := d.getSettingsUnsafe()
	isLinear := existing.Source == "linear"
	isGithub := existing.Source == "github"

	if isLinear {
		go func(key string, st models.Status, lbls []string) {
			_ = d.runner.UpdateLinearIssueState(key, st)
			_ = d.runner.UpdateLinearIssue(key, nil, nil, nil, &st, lbls)
		}(existing.Key, newStatus, existing.Labels)
	} else if isGithub && settings != nil {
		go func(repo, rPath, key string, st models.Status, lbls []string) {
			_ = d.runner.UpdateGithubIssueState(repo, rPath, key, st)
			_ = d.runner.UpdateGithubIssue(repo, rPath, key, nil, nil, &st, lbls)
		}(settings.GithubRepo, settings.RepoPath, existing.Key, newStatus, existing.Labels)
	}

	acts, _ := d.getTaskActivitiesUnsafe(existing.ID)
	existing.Activities = acts

	return existing, nil
}

func (d *DB) DeleteTask(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, _ = d.conn.Exec("DELETE FROM task_activities WHERE task_id = ?", id)
	_, err := d.conn.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

func (d *DB) getTaskByIDUnsafe(id string) (*models.Task, error) {
	var t models.Task
	var labelsJSON string
	var dueDate, branchName, prURL, source, extURL sql.NullString
	var statusStr, priorityStr string

	err := d.conn.QueryRow(`
		SELECT id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at
		FROM tasks WHERE id = ? OR key = ?
	`, id, id).Scan(
		&t.ID,
		&t.Key,
		&t.Title,
		&t.Description,
		&statusStr,
		&priorityStr,
		&labelsJSON,
		&t.Assignee,
		&t.AssigneeAvatar,
		&t.Position,
		&dueDate,
		&branchName,
		&prURL,
		&source,
		&extURL,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	t.Status = models.Status(statusStr)
	t.Priority = models.Priority(priorityStr)
	if dueDate.Valid {
		t.DueDate = &dueDate.String
	}
	if branchName.Valid {
		t.BranchName = &branchName.String
	}
	if prURL.Valid {
		t.PrURL = &prURL.String
	}
	if source.Valid {
		t.Source = source.String
	}
	if extURL.Valid {
		t.ExternalURL = &extURL.String
	}
	_ = json.Unmarshal([]byte(labelsJSON), &t.Labels)
	if t.Labels == nil {
		t.Labels = []string{}
	}

	return &t, nil
}

func (d *DB) addTaskActivityDirect(act models.TaskActivity) error {
	stepsJSON, _ := json.Marshal(act.Steps)
	if act.Steps == nil {
		stepsJSON = []byte("[]")
	}
	_, err := d.conn.Exec(`
		INSERT INTO task_activities (id, task_id, skill_id, skill_name, action, status, summary, output, steps, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, act.ID, act.TaskID, act.SkillID, act.SkillName, act.Action, act.Status, act.Summary, act.Output, string(stepsJSON), act.CreatedAt)
	return err
}

func (d *DB) getTaskActivitiesUnsafe(taskID string) ([]models.TaskActivity, error) {
	rows, err := d.conn.Query(`
		SELECT id, task_id, skill_id, skill_name, action, status, summary, output, steps, created_at
		FROM task_activities WHERE task_id = ? ORDER BY created_at DESC
	`, taskID)
	if err != nil {
		return []models.TaskActivity{}, nil
	}
	defer rows.Close()

	var list []models.TaskActivity
	for rows.Next() {
		var a models.TaskActivity
		var stepsJSON string
		err := rows.Scan(&a.ID, &a.TaskID, &a.SkillID, &a.SkillName, &a.Action, &a.Status, &a.Summary, &a.Output, &stepsJSON, &a.CreatedAt)
		if err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(stepsJSON), &a.Steps)
		if a.Steps == nil {
			a.Steps = []string{}
		}
		list = append(list, a)
	}
	if list == nil {
		list = []models.TaskActivity{}
	}
	return list, nil
}

func (d *DB) GetTaskActivities(taskID string) ([]models.TaskActivity, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getTaskActivitiesUnsafe(taskID)
}

func (d *DB) GetSettings() (*models.Settings, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var s models.Settings
	var detMode, aiProv, aiCmd, repoP, issTrk, linTm, ghRepo, pClar, pSpec, pImpl, pPR, pPick sql.NullString

	err := d.conn.QueryRow(`
		SELECT id, theme, accent_color, language, density, default_view, detail_mode, user_name, user_email, user_avatar,
		       ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo,
		       prompt_clarify, prompt_specify, prompt_implement, prompt_create_pr, prompt_pick, updated_at
		FROM settings WHERE id = 1
	`).Scan(
		&s.ID,
		&s.Theme,
		&s.AccentColor,
		&s.Language,
		&s.Density,
		&s.DefaultView,
		&detMode,
		&s.UserName,
		&s.UserEmail,
		&s.UserAvatar,
		&aiProv,
		&aiCmd,
		&repoP,
		&issTrk,
		&linTm,
		&ghRepo,
		&pClar,
		&pSpec,
		&pImpl,
		&pPR,
		&pPick,
		&s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.Settings{
				ID:                 1,
				Theme:              "dark",
				AccentColor:        "indigo",
				Language:           "fr",
				Density:            "standard",
				DefaultView:        "board",
				DetailMode:         "panel",
				UserName:           "Sylvain Ferry",
				UserEmail:          "sylvain@fretzee.com",
				UserAvatar:         "",
				AIProvider:         "agy",
				AICommandTemplate:  "agy -p \"{prompt}\"",
				RepoPath:           "/Users/sferry/Sources/fretzee-studio",
				IssueTracker:       "linear",
				LinearTeam:         "FRE",
				GithubRepo:         "sebastienferry/fretzee-studio",
				PromptClarify:      "",
				PromptSpecify:      "",
				PromptImplement:    "",
				PromptCreatePR:     "",
				PromptPick:         "",
				UpdatedAt:          time.Now(),
			}, nil
		}
		return nil, err
	}

	if detMode.Valid {
		s.DetailMode = detMode.String
	} else {
		s.DetailMode = "panel"
	}
	if aiProv.Valid {
		s.AIProvider = aiProv.String
	} else {
		s.AIProvider = "agy"
	}
	if aiCmd.Valid {
		s.AICommandTemplate = aiCmd.String
	} else {
		s.AICommandTemplate = "agy -p \"{prompt}\""
	}
	if repoP.Valid {
		s.RepoPath = repoP.String
	} else {
		s.RepoPath = "/Users/sferry/Sources/fretzee-studio"
	}
	if issTrk.Valid {
		s.IssueTracker = issTrk.String
	} else {
		s.IssueTracker = "linear"
	}
	if linTm.Valid {
		s.LinearTeam = linTm.String
	} else {
		s.LinearTeam = "FRE"
	}
	if ghRepo.Valid {
		s.GithubRepo = ghRepo.String
	} else {
		s.GithubRepo = "sebastienferry/fretzee-studio"
	}
	if pClar.Valid {
		s.PromptClarify = pClar.String
	}
	if pSpec.Valid {
		s.PromptSpecify = pSpec.String
	}
	if pImpl.Valid {
		s.PromptImplement = pImpl.String
	}
	if pPR.Valid {
		s.PromptCreatePR = pPR.String
	}
	if pPick.Valid {
		s.PromptPick = pPick.String
	}

	return &s, nil
}

func (d *DB) UpdateSettings(s models.Settings) (*models.Settings, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	current, _ := d.getSettingsUnsafe()
	if current != nil {
		if s.Theme == "" { s.Theme = current.Theme }
		if s.AccentColor == "" { s.AccentColor = current.AccentColor }
		if s.Language == "" { s.Language = current.Language }
		if s.Density == "" { s.Density = current.Density }
		if s.DefaultView == "" { s.DefaultView = current.DefaultView }
		if s.DetailMode == "" { s.DetailMode = current.DetailMode }
		if s.UserName == "" { s.UserName = current.UserName }
		if s.UserEmail == "" { s.UserEmail = current.UserEmail }
		if s.AIProvider == "" { s.AIProvider = current.AIProvider }
		if s.AICommandTemplate == "" { s.AICommandTemplate = current.AICommandTemplate }
		if s.RepoPath == "" { s.RepoPath = current.RepoPath }
		if s.IssueTracker == "" { s.IssueTracker = current.IssueTracker }
		if s.LinearTeam == "" { s.LinearTeam = current.LinearTeam }
		if s.GithubRepo == "" { s.GithubRepo = current.GithubRepo }
		if s.PromptClarify == "" { s.PromptClarify = current.PromptClarify }
		if s.PromptSpecify == "" { s.PromptSpecify = current.PromptSpecify }
		if s.PromptImplement == "" { s.PromptImplement = current.PromptImplement }
		if s.PromptCreatePR == "" { s.PromptCreatePR = current.PromptCreatePR }
		if s.PromptPick == "" { s.PromptPick = current.PromptPick }
	}

	if s.Theme == "" { s.Theme = "dark" }
	if s.AccentColor == "" { s.AccentColor = "indigo" }
	if s.Language == "" { s.Language = "fr" }
	if s.Density == "" { s.Density = "standard" }
	if s.DefaultView == "" { s.DefaultView = "board" }
	if s.DetailMode == "" { s.DetailMode = "panel" }
	if s.UserName == "" { s.UserName = "Sylvain Ferry" }
	if s.UserEmail == "" { s.UserEmail = "sylvain@fretzee.com" }
	if s.AIProvider == "" { s.AIProvider = "agy" }
	if s.AICommandTemplate == "" { s.AICommandTemplate = "agy -p \"{prompt}\"" }
	if s.RepoPath == "" { s.RepoPath = "/Users/sferry/Sources/fretzee-studio" }
	if s.IssueTracker == "" { s.IssueTracker = "linear" }
	if s.LinearTeam == "" { s.LinearTeam = "FRE" }
	if s.GithubRepo == "" { s.GithubRepo = "sebastienferry/fretzee-studio" }

	now := time.Now()
	_, err := d.conn.Exec(`
		INSERT INTO settings (id, theme, accent_color, language, density, default_view, detail_mode, user_name, user_email, user_avatar, ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo, prompt_clarify, prompt_specify, prompt_implement, prompt_create_pr, prompt_pick, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			theme = excluded.theme,
			accent_color = excluded.accent_color,
			language = excluded.language,
			density = excluded.density,
			default_view = excluded.default_view,
			detail_mode = excluded.detail_mode,
			user_name = excluded.user_name,
			user_email = excluded.user_email,
			user_avatar = excluded.user_avatar,
			ai_provider = excluded.ai_provider,
			ai_command_template = excluded.ai_command_template,
			repo_path = excluded.repo_path,
			issue_tracker = excluded.issue_tracker,
			linear_team = excluded.linear_team,
			github_repo = excluded.github_repo,
			prompt_clarify = excluded.prompt_clarify,
			prompt_specify = excluded.prompt_specify,
			prompt_implement = excluded.prompt_implement,
			prompt_create_pr = excluded.prompt_create_pr,
			prompt_pick = excluded.prompt_pick,
			updated_at = excluded.updated_at
	`, s.Theme, s.AccentColor, s.Language, s.Density, s.DefaultView, s.DetailMode, s.UserName, s.UserEmail, s.UserAvatar, s.AIProvider, s.AICommandTemplate, s.RepoPath, s.IssueTracker, s.LinearTeam, s.GithubRepo, s.PromptClarify, s.PromptSpecify, s.PromptImplement, s.PromptCreatePR, s.PromptPick, now)

	if err != nil {
		return nil, err
	}

	s.ID = 1
	s.UpdatedAt = now
	return &s, nil
}

func (d *DB) GetAvailableSkills() []models.Skill {
	return []models.Skill{
		{
			ID:           "clarify",
			Name:         "Clarify Issue",
			Command:      "/clarify-issue",
			Description:  "Analyse les ambiguïtés via l'IA, pose les questions de cadrage, avance vers 'À spécifier' et applique le label 'Clarified'.",
			InputStatus:  models.StatusToClarify,
			OutputStatus: models.StatusToSpecify,
			Icon:         "HelpCircle",
			Color:        "amber",
			Steps: []string{
				"Vérification de l'issue et du repo",
				"Exécution du prompt de clarification avec l'IA",
				"Génération des questions d'alignement",
				"Mise à jour du label vers 'Clarified'",
				"Transition vers 'À spécifier'",
			},
		},
		{
			ID:           "specify",
			Name:         "Specify Issue (Speckit)",
			Command:      "/specify-issue",
			Description:  "Rédige la spec technique Speckit via l'IA, initialise la branche Git, avance vers 'À implémenter' et applique le label 'Specified'.",
			InputStatus:  models.StatusToSpecify,
			OutputStatus: models.StatusToImplement,
			Icon:         "FileCode",
			Color:        "blue",
			Steps: []string{
				"Génération de la spec Speckit par l'IA",
				"Création de la branche Git",
				"Mise à jour du label vers 'Specified'",
				"Transition vers 'À implémenter'",
			},
		},
		{
			ID:           "implement",
			Name:         "Implement Code",
			Command:      "/code-issue",
			Description:  "Exécute le plan de code par l'IA, valide les tests, avance vers 'À tester' et applique le label 'Implemented'.",
			InputStatus:  models.StatusToImplement,
			OutputStatus: models.StatusToTest,
			Icon:         "Flame",
			Color:        "indigo",
			Steps: []string{
				"Analyse des modifications requises par l'IA",
				"Checkout de la branche de développement",
				"Mise à jour du label vers 'Implemented'",
				"Transition vers 'À tester'",
			},
		},
		{
			ID:           "create_pr",
			Name:         "Review & Pull Request",
			Command:      "/create-pr",
			Description:  "Génère la revue de code et la PR GitHub via gh CLI, avance vers 'À fermer' et applique le label 'Reviewed'.",
			InputStatus:  models.StatusToTest,
			OutputStatus: models.StatusToClose,
			Icon:         "ShieldCheck",
			Color:        "purple",
			Steps: []string{
				"Revue de code & génération du commit conventionnel",
				"Création de la Pull Request",
				"Mise à jour du label vers 'Reviewed'",
				"Transition vers 'À fermer'",
			},
		},
		{
			ID:           "pick",
			Name:         "Auto-Pilot Orchestrator",
			Command:      "/pick-issue",
			Description:  "Routeur intelligent : détecte automatiquement l'étape et enchaîne les 5 étapes (À clarifier ➔ À spécifier ➔ À implémenter ➔ À tester ➔ À fermer).",
			InputStatus:  models.StatusToClarify,
			OutputStatus: models.StatusToClose,
			Icon:         "Sparkles",
			Color:        "emerald",
			Steps: []string{
				"Analyse automatique de l'étape courante",
				"Sélection de la skill optimale",
				"Transition du statut et mise à jour du label",
			},
		},
	}
}

func (d *DB) RunSkillOnTask(taskID string, skillID string, prompt string) (*models.Task, *models.TaskActivity, error) {
	// 1. Fetch settings and task safely
	d.mu.RLock()
	task, err := d.getTaskByIDUnsafe(taskID)
	d.mu.RUnlock()
	if err != nil {
		return nil, nil, err
	}
	if task == nil {
		return nil, nil, fmt.Errorf("task not found")
	}

	settings, err := d.GetSettings()
	if err != nil {
		return nil, nil, err
	}

	var skill models.Skill
	for _, s := range d.GetAvailableSkills() {
		if s.ID == skillID || (skillID == "review" && s.ID == "create_pr") {
			skill = s
			break
		}
	}
	if skill.ID == "" {
		return nil, nil, fmt.Errorf("unknown skill: %s", skillID)
	}

	// 2. Compute branch name if not set
	cleanTitle := strings.ToLower(task.Title)
	cleanTitle = strings.ReplaceAll(cleanTitle, " ", "-")
	cleanTitle = strings.ReplaceAll(cleanTitle, "'", "-")
	cleanTitle = strings.ReplaceAll(cleanTitle, "\"", "")
	if len(cleanTitle) > 30 {
		cleanTitle = cleanTitle[:30]
	}
	branch := fmt.Sprintf("%s-%s", task.Key, cleanTitle)
	if task.BranchName == nil || *task.BranchName == "" {
		task.BranchName = &branch
	}

	// 3. Run Real AI CLI Execution via runner
	realAIOutput, steps, _ := d.runner.RunAI(settings, skillID, task, prompt)

	now := time.Now()
	activityID := uuid.New().String()
	var act models.TaskActivity
	act.ID = activityID
	act.TaskID = task.ID
	act.SkillID = skill.ID
	act.SkillName = skill.Name
	act.Status = "completed"
	act.CreatedAt = now
	act.Output = realAIOutput
	act.Steps = steps

	// 4. Update status & workflow labels according to action
	// Workflow stages:
	// - A clarifier (New)
	// - A spécifier (Clarified)
	// - A implémenter (Specified)
	// - A tester (Implemented)
	// - A fermer (Reviewed)
	switch skill.ID {
	case "clarify":
		task.Status = models.StatusToSpecify
		task.Labels = SetWorkflowLabel(task.Labels, "Clarified")
		act.Action = fmt.Sprintf("Clarification exécutée avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		act.Summary = fmt.Sprintf("Questions de cadrage générées ➔ Étape: À spécifier [Label: Clarified]")

	case "specify":
		task.Status = models.StatusToImplement
		task.Labels = SetWorkflowLabel(task.Labels, "Specified")
		act.Action = fmt.Sprintf("Spécification Speckit rédigée avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		act.Summary = fmt.Sprintf("Spec technique créée sur la branche %s ➔ Étape: À implémenter [Label: Specified]", *task.BranchName)

	case "implement":
		task.Status = models.StatusToTest
		task.Labels = SetWorkflowLabel(task.Labels, "Implemented")
		act.Action = fmt.Sprintf("Implémentation exécutée avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		act.Summary = fmt.Sprintf("Développement terminé sur la branche %s ➔ Étape: À tester [Label: Implemented]", *task.BranchName)

	case "create_pr", "review":
		task.Status = models.StatusToClose
		task.Labels = SetWorkflowLabel(task.Labels, "Reviewed")
		prURL := fmt.Sprintf("https://github.com/%s/pull/%s", settings.GithubRepo, strings.TrimPrefix(task.Key, "FRE-"))
		task.PrURL = &prURL
		act.Action = fmt.Sprintf("Revue & Pull Request préparées avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		act.Summary = fmt.Sprintf("PR prête pour revue : %s ➔ Étape: À fermer [Label: Reviewed]", prURL)

	case "pick":
		if task.Status == models.StatusToClarify || task.Status == models.StatusBacklog {
			task.Status = models.StatusToSpecify
		} else if task.Status == models.StatusToSpecify || task.Status == models.StatusSpecified {
			task.Status = models.StatusToImplement
		} else if task.Status == models.StatusToImplement || task.Status == models.StatusInProgress {
			task.Status = models.StatusToTest
		} else {
			task.Status = models.StatusToClose
		}
		targetLabel := GetStageLabelForStatus(task.Status)
		task.Labels = SetWorkflowLabel(task.Labels, targetLabel)
		act.Action = fmt.Sprintf("Auto-Pilot exécuté avec %s ➔ %s", strings.ToUpper(settings.AIProvider), task.Status)
		act.Summary = fmt.Sprintf("Statut mis à jour vers '%s' [Label: %s]", task.Status, targetLabel)
	}

	task.UpdatedAt = now

	// 5. Persist task and activity in SQLite
	d.mu.Lock()
	labelsJSON, _ := json.Marshal(task.Labels)
	_, err = d.conn.Exec(`
		UPDATE tasks
		SET status = ?, labels = ?, branch_name = ?, pr_url = ?, updated_at = ?
		WHERE id = ?
	`, string(task.Status), string(labelsJSON), task.BranchName, task.PrURL, now, task.ID)
	if err != nil {
		d.mu.Unlock()
		return nil, nil, err
	}

	_ = d.addTaskActivityDirect(act)
	acts, _ := d.getTaskActivitiesUnsafe(task.ID)
	task.Activities = acts
	d.mu.Unlock()

	// Background state and label sync with Linear / GitHub CLI
	if task.Source == "linear" {
		go func(k string, st models.Status, lbls []string) {
			_ = d.runner.UpdateLinearIssueState(k, st)
			_ = d.runner.UpdateLinearIssue(k, nil, nil, nil, &st, lbls)
		}(task.Key, task.Status, task.Labels)
	} else if task.Source == "github" && settings != nil {
		go func(repo, rPath, k string, st models.Status, lbls []string) {
			_ = d.runner.UpdateGithubIssueState(repo, rPath, k, st)
			_ = d.runner.UpdateGithubIssue(repo, rPath, k, nil, nil, &st, lbls)
		}(settings.GithubRepo, settings.RepoPath, task.Key, task.Status, task.Labels)
	}

	return task, &act, nil
}

func (d *DB) AddTaskComment(taskID string, body string) error {
	d.mu.RLock()
	task, err := d.getTaskByIDUnsafe(taskID)
	settings, _ := d.getSettingsUnsafe()
	d.mu.RUnlock()

	if err != nil || task == nil {
		return fmt.Errorf("task not found")
	}

	repo := ""
	repoPath := ""
	if settings != nil {
		repo = settings.GithubRepo
		repoPath = settings.RepoPath
	}

	return d.runner.AddIssueComment(task.Source, repo, repoPath, task.Key, body)
}

func (d *DB) ConvertTaskToRemote(taskID string, target string) (*models.Task, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	task, err := d.getTaskByIDUnsafe(taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

	settings, _ := d.getSettingsUnsafe()
	if settings == nil {
		settings = &models.Settings{
			LinearTeam: "FRE",
			GithubRepo: "sebastienferry/fretzee-studio",
			RepoPath:   "/Users/sferry/Sources/fretzee-studio",
		}
	}

	var newKey string
	var extURL *string
	now := time.Now()

	// Ensure stage label is properly set based on current status
	task.Labels = SetWorkflowLabel(task.Labels, GetStageLabelForStatus(task.Status))

	switch target {
	case "linear":
		team := settings.LinearTeam
		if team == "" {
			team = "FRE"
		}
		created, err := d.runner.CreateLinearIssue(team, task.Title, task.Description, task.Priority, task.Labels)
		if err != nil {
			return nil, fmt.Errorf("création Linear impossible: %w", err)
		}
		newKey = created.Key
		extURL = created.ExternalURL

		// Sync status to Linear if not backlog
		if task.Status != models.StatusBacklog {
			_ = d.runner.UpdateLinearIssueState(newKey, task.Status)
		}

	case "github":
		repo := settings.GithubRepo
		if repo == "" {
			repo = "sebastienferry/fretzee-studio"
		}
		repoPath := settings.RepoPath
		if repoPath == "" {
			repoPath = "/Users/sferry/Sources/fretzee-studio"
		}
		created, err := d.runner.CreateGithubIssue(repo, repoPath, task.Title, task.Description, task.Labels)
		if err != nil {
			return nil, fmt.Errorf("création GitHub impossible: %w", err)
		}
		newKey = created.Key
		extURL = created.ExternalURL

		// Sync status if done
		if task.Status == models.StatusDone {
			_ = d.runner.UpdateGithubIssue(repo, repoPath, newKey, nil, nil, &task.Status, task.Labels)
		}

	default:
		return nil, fmt.Errorf("tracker distant non supporté: %s (choisir 'linear' ou 'github')", target)
	}

	task.Key = newKey
	task.Source = target
	task.ExternalURL = extURL
	task.UpdatedAt = now

	labelsJSON, _ := json.Marshal(task.Labels)
	_, err = d.conn.Exec(`
		UPDATE tasks
		SET key = ?, source = ?, external_url = ?, labels = ?, updated_at = ?
		WHERE id = ?
	`, task.Key, task.Source, task.ExternalURL, string(labelsJSON), now, task.ID)
	if err != nil {
		return nil, err
	}

	act := models.TaskActivity{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		SkillID:   "convert",
		SkillName: "Export Tracker",
		Action:    fmt.Sprintf("Tâche convertie vers %s (%s)", strings.ToUpper(target), task.Key),
		Status:    "completed",
		Summary:   fmt.Sprintf("Issue distante créée avec succès : %s", task.Key),
		Output:    fmt.Sprintf("Tâche locale convertie vers %s.\nClé distante : %s\nURL : %s", strings.ToUpper(target), task.Key, *extURL),
		Steps:     []string{fmt.Sprintf("Création du ticket distant sur %s", strings.ToUpper(target)), fmt.Sprintf("Mise à jour de la clé (%s) et de la source", task.Key)},
		CreatedAt: now,
	}
	_ = d.addTaskActivityDirect(act)
	acts, _ := d.getTaskActivitiesUnsafe(task.ID)
	task.Activities = acts

	return task, nil
}

