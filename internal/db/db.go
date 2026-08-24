package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"tasks/internal/models"
	"tasks/internal/runner"
)

type SkillJob struct {
	ActivityID    string
	TaskID        string
	ProjectID     string
	SkillID       string
	Prompt        string
	RemovedLabels []string
}

type DB struct {
	conn      *sql.DB
	runner    *runner.Runner
	mu        sync.RWMutex
	jobQueue  chan SkillJob
	cancelMap map[string]context.CancelFunc
	cancelMu  sync.Mutex
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)

	db := &DB{
		conn:      conn,
		runner:    runner.NewRunner(),
		jobQueue:  make(chan SkillJob, 100),
		cancelMap: make(map[string]context.CancelFunc),
	}
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Start background queue worker
	go db.startQueueWorker()

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
			user_name TEXT NOT NULL DEFAULT 'Developer',
			user_email TEXT NOT NULL DEFAULT 'dev@example.com',
			user_avatar TEXT NOT NULL DEFAULT '',
			ai_provider TEXT NOT NULL DEFAULT 'agy',
			ai_command_template TEXT NOT NULL DEFAULT 'agy -p "{prompt}"',
			repo_path TEXT NOT NULL DEFAULT '.',
			issue_tracker TEXT NOT NULL DEFAULT 'local',
			linear_team TEXT NOT NULL DEFAULT '',
			github_repo TEXT NOT NULL DEFAULT '',
			prompt_clarify TEXT NOT NULL DEFAULT '',
			prompt_specify TEXT NOT NULL DEFAULT '',
			prompt_implement TEXT NOT NULL DEFAULT '',
			prompt_create_pr TEXT NOT NULL DEFAULT '',
			prompt_pick TEXT NOT NULL DEFAULT '',
			editor_command TEXT NOT NULL DEFAULT 'code',
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT 'Folder',
			color TEXT NOT NULL DEFAULT 'indigo',
			repo_path TEXT NOT NULL DEFAULT '',
			repo_paths TEXT NOT NULL DEFAULT '[]',
			linear_team TEXT NOT NULL DEFAULT '',
			github_repo TEXT NOT NULL DEFAULT '',
			issue_tracker TEXT NOT NULL DEFAULT 'local',
			project_type TEXT NOT NULL DEFAULT 'standard',
			is_default INTEGER NOT NULL DEFAULT 0,
			stage_mapping TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL DEFAULT 'default',
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
			repo_path TEXT NOT NULL DEFAULT '',
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
			prompt TEXT NOT NULL DEFAULT '',
			started_at DATETIME,
			completed_at DATETIME,
			error TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(status, position);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_task ON task_activities(task_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_status ON task_activities(status);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_created ON task_activities(created_at DESC);`,
	}

	for _, query := range queries {
		if _, err := d.conn.Exec(query); err != nil {
			return err
		}
	}

	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN stage_mapping TEXT NOT NULL DEFAULT '{}';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN git_remote_url TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN tracker_url TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN skill_overrides TEXT NOT NULL DEFAULT '{}';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN repo_paths TEXT NOT NULL DEFAULT '[]';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN ai_provider TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN ai_command_template TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN spec_framework TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN jira_project TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN project_type TEXT NOT NULL DEFAULT 'standard';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN project_id TEXT NOT NULL DEFAULT 'default';")
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN branch_name TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN pr_url TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN repo_path TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN source TEXT NOT NULL DEFAULT 'local';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN external_url TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN issue_type TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN parent_key TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN parent_title TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN parent_type TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_key);")
	_, _ = d.conn.Exec(`CREATE TABLE IF NOT EXISTS daily_digests (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		date TEXT NOT NULL,
		payload TEXT NOT NULL DEFAULT '{}',
		agenda TEXT NOT NULL DEFAULT '',
		ai_status TEXT NOT NULL DEFAULT 'none',
		ai_error TEXT NOT NULL DEFAULT '',
		ai_activity_id TEXT NOT NULL DEFAULT '',
		ai_updated_at DATETIME,
		markdown TEXT NOT NULL DEFAULT '',
		generated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(project_id, date)
	);`)
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_digests_project_date ON daily_digests(project_id, date DESC);")

	_, _ = d.conn.Exec("ALTER TABLE task_activities ADD COLUMN prompt TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE task_activities ADD COLUMN started_at DATETIME;")
	_, _ = d.conn.Exec("ALTER TABLE task_activities ADD COLUMN completed_at DATETIME;")
	_, _ = d.conn.Exec("ALTER TABLE task_activities ADD COLUMN error TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("UPDATE task_activities SET status = 'failed', error = 'Interrompu lors du redémarrage du serveur' WHERE status IN ('running', 'queued', 'pending');")

	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN detail_mode TEXT NOT NULL DEFAULT 'panel';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN ai_provider TEXT NOT NULL DEFAULT 'agy';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN ai_command_template TEXT NOT NULL DEFAULT 'agy -p \"{prompt}\"';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN repo_path TEXT NOT NULL DEFAULT '.';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN issue_tracker TEXT NOT NULL DEFAULT 'local';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN linear_team TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN github_repo TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_clarify TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_specify TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_implement TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_create_pr TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_pick TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN editor_command TEXT NOT NULL DEFAULT 'code';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN spec_framework TEXT NOT NULL DEFAULT 'speckit';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN jira_project TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN jira_url TEXT NOT NULL DEFAULT '';")

	// Migrate the legacy 'openfeature' Spec-Driven Design option to 'openspec'.
	// OpenFeature is a feature-flag standard, not an SDD framework: the two
	// supported frameworks are GitHub Spec Kit and OpenSpec.
	_, _ = d.conn.Exec("UPDATE projects SET spec_framework = 'openspec' WHERE spec_framework = 'openfeature';")
	_, _ = d.conn.Exec("UPDATE settings SET spec_framework = 'openspec' WHERE spec_framework = 'openfeature';")

	// Migrate legacy stage names to 5-stage workflow
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_clarify' WHERE status = 'backlog';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_specify' WHERE status = 'specified';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_implement' WHERE status = 'in_progress';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_test' WHERE status = 'to_validate';")
	_, _ = d.conn.Exec("UPDATE tasks SET status = 'to_close' WHERE status = 'done';")

	// Seed default workspace only if projects table is completely empty
	var projectsCount int
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectsCount)
	if projectsCount == 0 {
		_, _ = d.conn.Exec(`
			INSERT INTO projects (id, name, slug, description, icon, color, repo_path, linear_team, github_repo, issue_tracker, is_default)
			VALUES ('default', 'Default Project', 'default', 'Primary workspace repository', 'Folder', 'emerald', '.', '', '', 'local', 1);
		`)
	}

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
			VALUES (1, 'dark', 'indigo', 'en', 'standard', 'board', 'Developer', 'dev@example.com', '', 'agy', 'agy -p "{prompt}"', '.', 'local', '', '')
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DB) SeedDemoData() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Ensure default settings
	_, err := d.conn.Exec(`
		INSERT INTO settings (id, theme, accent_color, language, density, default_view, user_name, user_email, user_avatar, ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo)
		VALUES (1, 'dark', 'indigo', 'en', 'standard', 'board', 'Developer', 'dev@example.com', '', 'agy', 'agy -p "{prompt}"', '.', 'local', '', '')
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

	// Seed clean demo tasks for default project
	now := time.Now()
	demoTasks := []struct {
		ID, Key, Title, Desc, Status, Priority, Branch, Labels string
		Pos                                                    int
	}{
		{"task-1", "TASK-1", "Initialize workspace configuration and metadata", "Setup project structure, metadata, and continuous integration pipeline.", "finished", "high", "TASK-1-init-workspace", `["devops", "repo"]`, 1},
		{"task-2", "TASK-2", "Configure multi-tracker sync and issue mappings", "Implement generic abstractions for Linear, GitHub, Jira, and Local SQLite storage.", "to_implement", "high", "TASK-2-configure-trackers", `["tracker", "sync", "backend"]`, 2},
		{"task-3", "TASK-3", "Refine Kanban board drag and drop interactions", "Ensure optimistic UI updates and smooth animations across all workflow stages.", "to_specify", "medium", "TASK-3-kanban-board-dnd", `["ui", "kanban", "frontend"]`, 3},
		{"task-4", "TASK-4", "Implement interactive terminal session manager", "Provide browser-based PTY terminal with contextual environment variables and WebSocket streaming.", "to_test", "high", "TASK-4-terminal-session", `["pty", "terminal", "websocket"]`, 4},
		{"task-5", "TASK-5", "Integrate automated AI skill runner pipeline", "Orchestrate clarify, specify, code, and PR generation skills directly in isolated worktrees.", "to_close", "high", "TASK-5-ai-skills-pipeline", `["ai", "agent", "skills"]`, 5},
		{"task-6", "TASK-6", "Add live Git diff and branch inspector", "Display syntax-highlighted file diffs and branch status against the main repository.", "to_clarify", "low", "TASK-6-git-diff-inspector", `["git", "diff", "ui"]`, 6},
	}

	for _, dt := range demoTasks {
		_, _ = d.conn.Exec(`
			INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, position, branch_name, source, created_at, updated_at)
			VALUES (?, 'default', ?, ?, ?, ?, ?, ?, 'Developer', ?, ?, 'local', ?, ?)
		`, dt.ID, dt.Key, dt.Title, dt.Desc, dt.Status, dt.Priority, dt.Labels, dt.Pos, dt.Branch, now.Format(time.RFC3339), now)
	}

	return nil
}

func (d *DB) ImportOrUpdateTasks(syncedTasks []models.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	var importErrs []string
	for _, t := range syncedTasks {
		labelsJSON, _ := json.Marshal(t.Labels)
		var existingID string
		err := d.conn.QueryRow("SELECT id FROM tasks WHERE key = ? OR id = ?", t.Key, t.ID).Scan(&existingID)

		src := t.Source
		if src == "" {
			if strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") || strings.HasPrefix(t.Key, "#") {
				src = "github"
			} else if strings.Contains(t.Key, "-") {
				src = "linear"
			} else {
				src = "local"
			}
		}

		projID := t.ProjectID
		if projID == "" {
			var defaultProjID string
			_ = d.conn.QueryRow("SELECT id FROM projects WHERE is_default = 1 LIMIT 1").Scan(&defaultProjID)
			if defaultProjID == "" {
				defaultProjID = "default"
			}
			projID = defaultProjID
		}

		if err == sql.ErrNoRows {
			// Insert new task
			newID := t.ID
			if newID == "" {
				newID = uuid.New().String()
			}
			if _, insErr := d.conn.Exec(`
				INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, position, due_date, source, external_url, issue_type, parent_key, parent_title, parent_type, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newID, projID, t.Key, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.Position, t.DueDate, src, t.ExternalURL, t.IssueType, t.ParentKey, t.ParentTitle, t.ParentType, t.CreatedAt, now); insErr != nil {
				// Never swallow this: a silent failure here makes a sync report
				// "N tickets imported" while the board stays empty.
				log.Printf("[DB.ImportOrUpdateTasks] insert of %s failed: %v", t.Key, insErr)
				importErrs = append(importErrs, fmt.Sprintf("%s: %v", t.Key, insErr))
			}
		} else if err == nil {
			// Update existing task title/desc/status/labels/source
			if _, updErr := d.conn.Exec(`
				UPDATE tasks
				SET title = ?, description = ?, status = ?, priority = ?, labels = ?, assignee = ?, source = ?, external_url = ?,
				    issue_type = CASE WHEN ? != '' THEN ? ELSE issue_type END,
				    parent_key = CASE WHEN ? != '' THEN ? ELSE parent_key END,
				    parent_title = CASE WHEN ? != '' THEN ? ELSE parent_title END,
				    parent_type = CASE WHEN ? != '' THEN ? ELSE parent_type END,
				    updated_at = ?
				WHERE id = ?
			`, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, src, t.ExternalURL,
				t.IssueType, t.IssueType, t.ParentKey, t.ParentKey, t.ParentTitle, t.ParentTitle, t.ParentType, t.ParentType, now, existingID); updErr != nil {
				log.Printf("[DB.ImportOrUpdateTasks] update of %s failed: %v", t.Key, updErr)
				importErrs = append(importErrs, fmt.Sprintf("%s: %v", t.Key, updErr))
			}
		}
	}

	if len(importErrs) > 0 {
		shown := importErrs
		if len(shown) > 3 {
			shown = shown[:3]
		}
		return fmt.Errorf("%d/%d tickets n'ont pas pu être enregistrés (ex: %s)",
			len(importErrs), len(syncedTasks), strings.Join(shown, "; "))
	}
	return nil
}

func (d *DB) computeExternalURLUnsafe(t *models.Task) *string {
	if t.ExternalURL != nil && *t.ExternalURL != "" {
		return t.ExternalURL
	}
	var proj *models.Project
	if t.ProjectID != "" {
		proj, _ = d.getProjectByIDUnsafe(t.ProjectID)
	}
	source := t.Source
	if source == "" && proj != nil {
		source = proj.IssueTracker
	}
	switch source {
	case "linear":
		if proj != nil && proj.TrackerUrl != "" {
			u := fmt.Sprintf("%s/issue/%s", strings.TrimSuffix(proj.TrackerUrl, "/"), t.Key)
			return &u
		}
		if proj != nil && proj.LinearTeam != "" {
			u := fmt.Sprintf("https://linear.app/%s/issue/%s", proj.LinearTeam, t.Key)
			return &u
		}
		u := fmt.Sprintf("https://linear.app/issue/%s", t.Key)
		return &u
	case "github":
		var repo string
		if proj != nil && proj.GithubRepo != "" {
			repo = proj.GithubRepo
		} else if proj != nil && proj.RepoPath != "" {
			repo, _ = runner.ResolveGithubRepo("", proj.RepoPath)
		}
		cleanNum := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(t.Key, "GH-#"), "gh-"), "#")
		if repo != "" {
			u := fmt.Sprintf("https://github.com/%s/issues/%s", repo, cleanNum)
			return &u
		}
	case "jira":
		base := ""
		if proj != nil && proj.TrackerUrl != "" {
			base = proj.TrackerUrl
		} else if s, _ := d.getSettingsUnsafe(); s != nil && s.JiraUrl != "" {
			base = s.JiraUrl
		}
		if base != "" {
			u := fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(base, "/"), t.Key)
			return &u
		}
	}
	return nil
}

func (d *DB) GetTasks(query, status, priority, label, projectID string) ([]models.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var conditions []string
	var args []interface{}

	if projectID != "" && projectID != "all" {
		conditions = append(conditions, "(project_id = ? OR project_id = (SELECT slug FROM projects WHERE id = ?) OR project_id = (SELECT id FROM projects WHERE slug = ?))")
		args = append(args, projectID, projectID, projectID)
	}

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

	sqlQuery := "SELECT id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, repo_path, source, external_url, issue_type, parent_key, parent_title, parent_type, created_at, updated_at FROM tasks"
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
		var dueDate, branchName, prURL, repoPath, source, extURL, issueType, parentKey, parentTitle, parentType sql.NullString
		var statusStr, priorityStr string

		err := rows.Scan(
			&t.ID,
			&t.ProjectID,
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
			&repoPath,
			&source,
			&extURL,
			&issueType,
			&parentKey,
			&parentTitle,
			&parentType,
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
		if repoPath.Valid && repoPath.String != "" {
			p := repoPath.String
			t.RepoPath = &p
		}
		if source.Valid && source.String != "" {
			t.Source = source.String
		} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
			t.Source = "github"
		} else if strings.Contains(t.Key, "-") {
			t.Source = "linear"
		} else {
			t.Source = "local"
		}

		if extURL.Valid && extURL.String != "" {
			t.ExternalURL = &extURL.String
		} else {
			t.ExternalURL = d.computeExternalURLUnsafe(&t)
		}

		t.IssueType = issueType.String
		t.ParentKey = parentKey.String
		t.ParentTitle = parentTitle.String
		t.ParentType = parentType.String

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
	var dueDate, branchName, prURL, repoPath, source, extURL, issueType, parentKey, parentTitle, parentType sql.NullString
	var statusStr, priorityStr string

	err := d.conn.QueryRow(`
		SELECT id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, repo_path, source, external_url, issue_type, parent_key, parent_title, parent_type, created_at, updated_at
		FROM tasks WHERE id = ? OR key = ?
	`, id, id).Scan(
		&t.ID,
		&t.ProjectID,
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
		&repoPath,
		&source,
		&extURL,
		&issueType,
		&parentKey,
		&parentTitle,
		&parentType,
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
	if repoPath.Valid && repoPath.String != "" {
		p := repoPath.String
		t.RepoPath = &p
	}
	if source.Valid && source.String != "" {
		t.Source = source.String
	} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
		t.Source = "github"
	} else if strings.Contains(t.Key, "-") {
		t.Source = "linear"
	} else {
		t.Source = "local"
	}

	if extURL.Valid && extURL.String != "" {
		t.ExternalURL = &extURL.String
	} else {
		t.ExternalURL = d.computeExternalURLUnsafe(&t)
	}

	t.IssueType = issueType.String
	t.ParentKey = parentKey.String
	t.ParentTitle = parentTitle.String
	t.ParentType = parentType.String
	_ = json.Unmarshal([]byte(labelsJSON), &t.Labels)
	if t.Labels == nil {
		t.Labels = []string{}
	}

	activities, _ := d.getTaskActivitiesUnsafe(t.ID)
	t.Activities = activities

	return &t, nil
}

func (d *DB) EnsureGitIgnoreTasks(repoPath string) error {
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(gitignorePath, []byte("# Taskacao worktrees\n.tasks/\n"), 0644)
		}
		return nil
	}
	content := string(data)
	if !strings.Contains(content, ".tasks") {
		newContent := strings.TrimRight(content, "\r\n") + "\n\n# Taskacao parallel agent worktrees\n.tasks/\n"
		return os.WriteFile(gitignorePath, []byte(newContent), 0644)
	}
	return nil
}

// repoPathValue flattens the optional per-task repository into the empty string
// the column stores when the ticket inherits its project's path.
func repoPathValue(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// ResolveTaskRepoPath returns the repository a task works in: its own pinned
// path first, then its project's, then the global setting. Trackers where one
// epic spans several codebases need the per-ticket override.
func (d *DB) ResolveTaskRepoPath(task *models.Task) string {
	if task == nil {
		return ""
	}
	if p := repoPathValue(task.RepoPath); p != "" {
		return p
	}
	if task.ProjectID != "" {
		if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil && proj.RepoPath != "" {
			return proj.RepoPath
		}
	}
	if settings, _ := d.GetSettings(); settings != nil && settings.RepoPath != "" {
		return settings.RepoPath
	}
	return ""
}

func (d *DB) EnsureTaskWorktree(mainRepoPath string, task *models.Task) (string, string, error) {
	if mainRepoPath == "" || task == nil {
		return mainRepoPath, "", nil
	}

	mainRepoPath = strings.TrimSpace(mainRepoPath)
	gitDir := filepath.Join(mainRepoPath, ".git")
	if fi, err := os.Stat(gitDir); err != nil || !fi.IsDir() {
		return mainRepoPath, "", nil
	}

	// Ensure .tasks/ is ignored by Git
	_ = d.EnsureGitIgnoreTasks(mainRepoPath)

	// Compute branch name if not set
	if task.BranchName == nil || *task.BranchName == "" {
		cleanTitle := strings.ToLower(task.Title)
		cleanTitle = strings.ReplaceAll(cleanTitle, " ", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "'", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "\"", "")
		cleanTitle = strings.ReplaceAll(cleanTitle, "/", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "\\", "-")
		if len(cleanTitle) > 30 {
			cleanTitle = cleanTitle[:30]
		}
		branch := fmt.Sprintf("%s-%s", task.Key, cleanTitle)
		task.BranchName = &branch

		d.mu.Lock()
		_, _ = d.conn.Exec("UPDATE tasks SET branch_name = ? WHERE id = ?", branch, task.ID)
		d.mu.Unlock()
	}

	targetBranch := *task.BranchName
	worktreeBase := filepath.Join(mainRepoPath, ".tasks", "worktrees")
	worktreePath := filepath.Join(worktreeBase, task.Key)

	// Check if worktree directory already exists
	if fi, err := os.Stat(worktreePath); err == nil && fi.IsDir() {
		checkCmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--is-inside-work-tree")
		if out, err := checkCmd.Output(); err == nil && strings.TrimSpace(string(out)) == "true" {
			_ = exec.Command("git", "-C", worktreePath, "checkout", "-B", targetBranch).Run()
			return worktreePath, targetBranch, nil
		}
		_ = exec.Command("git", "-C", mainRepoPath, "worktree", "remove", "--force", worktreePath).Run()
		_ = os.RemoveAll(worktreePath)
		_ = exec.Command("git", "-C", mainRepoPath, "worktree", "prune").Run()
	}

	_ = os.MkdirAll(worktreeBase, 0755)

	// Check if target branch exists in repo
	checkBranchCmd := exec.Command("git", "-C", mainRepoPath, "rev-parse", "--verify", targetBranch)
	branchExists := checkBranchCmd.Run() == nil

	if branchExists {
		addCmd := exec.Command("git", "-C", mainRepoPath, "worktree", "add", worktreePath, targetBranch)
		if _, err := addCmd.CombinedOutput(); err != nil {
			addCmd2 := exec.Command("git", "-C", mainRepoPath, "worktree", "add", "--force", "-B", targetBranch, worktreePath, targetBranch)
			if out2, err2 := addCmd2.CombinedOutput(); err2 != nil {
				return mainRepoPath, targetBranch, fmt.Errorf("erreur git worktree add: %s (%w)", string(out2), err2)
			}
		}
	} else {
		baseBranch := "main"
		if err := exec.Command("git", "-C", mainRepoPath, "rev-parse", "--verify", "main").Run(); err != nil {
			if err2 := exec.Command("git", "-C", mainRepoPath, "rev-parse", "--verify", "master").Run(); err2 == nil {
				baseBranch = "master"
			}
		}

		addCmd := exec.Command("git", "-C", mainRepoPath, "worktree", "add", "-b", targetBranch, worktreePath, baseBranch)
		if _, err := addCmd.CombinedOutput(); err != nil {
			addCmd2 := exec.Command("git", "-C", mainRepoPath, "worktree", "add", "-B", targetBranch, worktreePath, baseBranch)
			if out2, err2 := addCmd2.CombinedOutput(); err2 != nil {
				return mainRepoPath, targetBranch, fmt.Errorf("erreur création worktree: %s (%w)", string(out2), err2)
			}
		}
	}

	// Symlink dependencies to speed up builds and avoid redundant node_modules downloads
	mainNodeModules := filepath.Join(mainRepoPath, "node_modules")
	wtNodeModules := filepath.Join(worktreePath, "node_modules")
	if fi, err := os.Stat(mainNodeModules); err == nil && fi.IsDir() {
		if _, err := os.Stat(wtNodeModules); os.IsNotExist(err) {
			_ = os.Symlink(mainNodeModules, wtNodeModules)
		}
	}

	mainWebNodeModules := filepath.Join(mainRepoPath, "web", "node_modules")
	wtWebNodeModules := filepath.Join(worktreePath, "web", "node_modules")
	if fi, err := os.Stat(mainWebNodeModules); err == nil && fi.IsDir() {
		_ = os.MkdirAll(filepath.Join(worktreePath, "web"), 0755)
		if _, err := os.Stat(wtWebNodeModules); os.IsNotExist(err) {
			_ = os.Symlink(mainWebNodeModules, wtWebNodeModules)
		}
	}

	// Symlink / propagate skills and agent configurations (.agents, .gemini, .agy, .taskacao)
	agentDirs := []string{".agents", ".gemini", ".agy", ".taskacao"}
	for _, ad := range agentDirs {
		mainAd := filepath.Join(mainRepoPath, ad)
		wtAd := filepath.Join(worktreePath, ad)
		if fi, err := os.Stat(mainAd); err == nil && fi.IsDir() {
			if _, err := os.Stat(wtAd); os.IsNotExist(err) {
				_ = os.Symlink(mainAd, wtAd)
			}
		}
	}

	// Ensure default skills are present in the worktree (.agents/skills, .gemini/skills, .agy/skills)
	for _, s := range DefaultProjectSkills {
		dirs := []string{
			filepath.Join(worktreePath, ".agents", "skills", s.DirName),
			filepath.Join(worktreePath, ".gemini", "skills", s.DirName),
			filepath.Join(worktreePath, ".agy", "skills", s.DirName),
		}
		for _, dir := range dirs {
			_ = os.MkdirAll(dir, 0755)
			filePath := filepath.Join(dir, "SKILL.md")
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				_ = os.WriteFile(filePath, []byte(s.Content), 0644)
			}
		}
	}

	// Symlink all .env* execution environment files if present in root
	envMatches, _ := filepath.Glob(filepath.Join(mainRepoPath, ".env*"))
	for _, envFile := range envMatches {
		base := filepath.Base(envFile)
		wtFile := filepath.Join(worktreePath, base)
		if fi, err := os.Stat(envFile); err == nil && !fi.IsDir() {
			if _, err := os.Stat(wtFile); os.IsNotExist(err) {
				_ = os.Symlink(envFile, wtFile)
			}
		}
	}

	return worktreePath, targetBranch, nil
}

func (d *DB) RemoveTaskWorktree(mainRepoPath string, taskKey string) error {
	if mainRepoPath == "" || taskKey == "" {
		return nil
	}
	worktreePath := filepath.Join(mainRepoPath, ".tasks", "worktrees", taskKey)
	_ = exec.Command("git", "-C", mainRepoPath, "worktree", "remove", "--force", worktreePath).Run()
	_ = exec.Command("git", "-C", mainRepoPath, "worktree", "prune").Run()
	_ = os.RemoveAll(worktreePath)
	return nil
}

func (d *DB) GetTaskWorktreeInfo(taskIDOrKey string) (*models.WorktreeInfo, error) {
	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil || task == nil {
		return nil, fmt.Errorf("tâche non trouvée")
	}

	mainRepoPath := d.ResolveTaskRepoPath(task)

	branch := ""
	if task.BranchName != nil {
		branch = *task.BranchName
	}

	worktreePath := filepath.Join(mainRepoPath, ".tasks", "worktrees", task.Key)
	exists := false
	if fi, err := os.Stat(worktreePath); err == nil && fi.IsDir() {
		exists = true
	}

	return &models.WorktreeInfo{
		TaskKey:      task.Key,
		Branch:       branch,
		WorktreePath: worktreePath,
		Exists:       exists,
		MainRepoPath: mainRepoPath,
	}, nil
}

func (d *DB) GetTaskGitDiff(taskIDOrKey string) (*models.GitDiffResult, error) {
	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("tâche non trouvée")
	}

	repoPath := d.ResolveTaskRepoPath(task)

	branchName := ""
	if task.BranchName != nil && *task.BranchName != "" {
		branchName = *task.BranchName
	}

	worktreePath := filepath.Join(repoPath, ".tasks", "worktrees", task.Key)
	diffTargetDir := repoPath
	if fi, err := os.Stat(worktreePath); err == nil && fi.IsDir() {
		diffTargetDir = worktreePath
	}

	res, err := d.runner.GetGitDiff(diffTargetDir, branchName, task.Key, task.PrURL)
	if res != nil && diffTargetDir != repoPath {
		res.WorktreePath = worktreePath
	}
	return res, err
}

// EnsureTaskGitBranch ensures that the project git repository is switched to the task's dedicated branch.
// It auto-commits any pending changes on previous branches, creates the branch if non-existent, and switches to it.
func (d *DB) EnsureTaskGitBranch(repoPath string, task *models.Task) (string, error) {
	if repoPath == "" || task == nil {
		return "", nil
	}

	gitDir := filepath.Join(repoPath, ".git")
	if fi, err := os.Stat(gitDir); err != nil || !fi.IsDir() {
		return "", nil
	}

	// Compute branch name if not set
	if task.BranchName == nil || *task.BranchName == "" {
		cleanTitle := strings.ToLower(task.Title)
		cleanTitle = strings.ReplaceAll(cleanTitle, " ", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "'", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "\"", "")
		cleanTitle = strings.ReplaceAll(cleanTitle, "/", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "\\", "-")
		if len(cleanTitle) > 30 {
			cleanTitle = cleanTitle[:30]
		}
		branch := fmt.Sprintf("%s-%s", task.Key, cleanTitle)
		task.BranchName = &branch

		d.mu.Lock()
		_, _ = d.conn.Exec("UPDATE tasks SET branch_name = ? WHERE id = ?", branch, task.ID)
		d.mu.Unlock()
	}

	targetBranch := *task.BranchName

	// 1. Get current branch
	currentBranchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	curOut, curErr := currentBranchCmd.Output()
	currentBranch := ""
	if curErr == nil {
		currentBranch = strings.TrimSpace(string(curOut))
	}

	// 2. If already on the target branch, we are good
	if currentBranch == targetBranch {
		return targetBranch, nil
	}

	// 3. If there are uncommitted changes on the old branch, auto-commit them cleanly to prevent checkout collisions
	statusOut, _ := exec.Command("git", "-C", repoPath, "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(statusOut))) > 0 {
		_ = exec.Command("git", "-C", repoPath, "add", "-A").Run()
		commitMsg := fmt.Sprintf("chore: auto-save progress on '%s' before switching to task '%s'", currentBranch, task.Key)
		_ = exec.Command("git", "-C", repoPath, "commit", "-m", commitMsg).Run()
	}

	// 4. Check if target branch already exists
	checkBranchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", targetBranch)
	if err := checkBranchCmd.Run(); err == nil {
		// Branch exists: switch to it
		switchCmd := exec.Command("git", "-C", repoPath, "checkout", targetBranch)
		if out, err := switchCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("erreur bascule branche %s: %s (%w)", targetBranch, string(out), err)
		}
	} else {
		// Branch does not exist: create and switch
		createCmd := exec.Command("git", "-C", repoPath, "checkout", "-b", targetBranch)
		if _, err := createCmd.CombinedOutput(); err != nil {
			// Fallback with -B
			createCmd2 := exec.Command("git", "-C", repoPath, "checkout", "-B", targetBranch)
			if out2, err2 := createCmd2.CombinedOutput(); err2 != nil {
				return "", fmt.Errorf("erreur création branche %s: %s (%w)", targetBranch, string(out2), err2)
			}
		}
	}

	return targetBranch, nil
}

func (d *DB) GetGitStatus(projectIDOrPath string) (*models.GitStatusInfo, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	if projectIDOrPath != "" {
		if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil && proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	if repoPath == "" {
		settings, _ := d.getSettingsUnsafe()
		if settings != nil && settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}
	d.mu.RUnlock()

	return d.runner.GetCwdGitStatus(repoPath)
}

func (d *DB) GetGitBranches(projectIDOrPath string) (*models.GitBranchesInfo, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	if projectIDOrPath != "" {
		if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil && proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	if repoPath == "" {
		settings, _ := d.getSettingsUnsafe()
		if settings != nil && settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}
	d.mu.RUnlock()

	if repoPath == "" {
		return nil, fmt.Errorf("aucun chemin de dépôt Git configuré")
	}

	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("le dossier %s n'existe pas", repoPath)
	}

	// Current active branch
	curCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	curOut, _ := curCmd.Output()
	currentBranch := strings.TrimSpace(string(curOut))

	// Get all branches (local + remote)
	format := "%(HEAD)|%(refname:short)|%(objectname:short)|%(contents:subject)"
	cmd := exec.Command("git", "-C", repoPath, "branch", "-a", "--format="+format)
	out, err := cmd.Output()
	if err != nil {
		return &models.GitBranchesInfo{
			RepoPath:      repoPath,
			CurrentBranch: currentBranch,
			Branches: []models.GitBranchItem{
				{Name: currentBranch, IsCurrent: true},
			},
		}, nil
	}

	var branches []models.GitBranchItem
	seen := make(map[string]bool)

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 2 {
			continue
		}
		isHead := strings.TrimSpace(parts[0]) == "*"
		rawName := strings.TrimSpace(parts[1])
		commit := ""
		if len(parts) >= 3 {
			commit = strings.TrimSpace(parts[2])
		}
		message := ""
		if len(parts) >= 4 {
			message = strings.TrimSpace(parts[3])
		}

		if rawName == "" || strings.HasPrefix(rawName, "origin/HEAD") || strings.HasPrefix(rawName, "remotes/origin/HEAD") {
			continue
		}

		isRemote := strings.HasPrefix(rawName, "origin/") || strings.HasPrefix(rawName, "remotes/")
		cleanName := strings.TrimPrefix(rawName, "remotes/")
		cleanName = strings.TrimPrefix(cleanName, "origin/")

		if seen[cleanName] {
			continue
		}
		seen[cleanName] = true

		isCurrent := isHead || cleanName == currentBranch

		branches = append(branches, models.GitBranchItem{
			Name:      cleanName,
			IsCurrent: isCurrent,
			IsRemote:  isRemote,
			Commit:    commit,
			Message:   message,
		})
	}

	if currentBranch != "" && !seen[currentBranch] {
		branches = append([]models.GitBranchItem{{
			Name:      currentBranch,
			IsCurrent: true,
		}}, branches...)
	}

	return &models.GitBranchesInfo{
		RepoPath:      repoPath,
		CurrentBranch: currentBranch,
		Branches:      branches,
	}, nil
}

func (d *DB) SwitchGitBranch(projectIDOrPath, targetBranch string, create bool) (*models.GitStatusInfo, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	if projectIDOrPath != "" {
		if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil && proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	if repoPath == "" {
		settings, _ := d.getSettingsUnsafe()
		if settings != nil && settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}
	d.mu.RUnlock()

	if repoPath == "" {
		return nil, fmt.Errorf("aucun chemin de dépôt Git configuré")
	}

	targetBranch = strings.TrimSpace(targetBranch)
	if targetBranch == "" {
		return nil, fmt.Errorf("nom de branche cible obligatoire")
	}

	// Clean stale index.lock if present
	lockPath := filepath.Join(repoPath, ".git", "index.lock")
	if info, err := os.Stat(lockPath); err == nil {
		if time.Since(info.ModTime()) > 3*time.Second {
			_ = os.Remove(lockPath)
		}
	}

	// 1. Get current branch
	curCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	curOut, _ := curCmd.Output()
	currentBranch := strings.TrimSpace(string(curOut))

	if currentBranch == targetBranch && !create {
		return d.runner.GetCwdGitStatus(repoPath)
	}

	// 2. If uncommitted changes exist, safely commit them
	statusOut, _ := exec.Command("git", "-C", repoPath, "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(statusOut))) > 0 {
		_ = exec.Command("git", "-C", repoPath, "add", "-A").Run()
		commitMsg := fmt.Sprintf("chore: auto-save work on '%s' before switching to '%s'", currentBranch, targetBranch)
		_ = exec.Command("git", "-C", repoPath, "commit", "-m", commitMsg).Run()
	}

	// 3. Checkout branch
	if create {
		createCmd := exec.Command("git", "-C", repoPath, "checkout", "-b", targetBranch)
		if _, err := createCmd.CombinedOutput(); err != nil {
			createCmd2 := exec.Command("git", "-C", repoPath, "checkout", "-B", targetBranch)
			if out2, err2 := createCmd2.CombinedOutput(); err2 != nil {
				return nil, fmt.Errorf("erreur création branche %s: %s (%w)", targetBranch, string(out2), err2)
			}
		}
	} else {
		// Check if branch exists locally
		if err := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", targetBranch).Run(); err == nil {
			switchCmd := exec.Command("git", "-C", repoPath, "checkout", targetBranch)
			if out, err := switchCmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("erreur bascule branche %s: %s (%w)", targetBranch, string(out), err)
			}
		} else {
			// Try checkout remote or create
			trackCmd := exec.Command("git", "-C", repoPath, "checkout", "--track", "origin/"+targetBranch)
			if _, err := trackCmd.CombinedOutput(); err != nil {
				// Fallback checkout -b
				createCmd := exec.Command("git", "-C", repoPath, "checkout", "-b", targetBranch)
				if out2, err2 := createCmd.CombinedOutput(); err2 != nil {
					return nil, fmt.Errorf("erreur bascule branche %s: %s (%w)", targetBranch, string(out2), err2)
				}
			}
		}
	}

	return d.runner.GetCwdGitStatus(repoPath)
}

func (d *DB) CleanAllLocalBranches(projectIDOrPath string) (*models.CleanBranchesResult, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	if projectIDOrPath != "" {
		if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil && proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	if repoPath == "" {
		settings, _ := d.getSettingsUnsafe()
		if settings != nil && settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}
	d.mu.RUnlock()

	if repoPath == "" {
		repoPath = "."
	}

	if fi, err := os.Stat(repoPath); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("dossier introuvable: %s", repoPath)
	}

	// 1. Detect default branch (main or master)
	defaultBranch := "main"
	if err := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "main").Run(); err != nil {
		if err2 := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "master").Run(); err2 == nil {
			defaultBranch = "master"
		}
	}

	// 2. Remove any active worktrees under .tasks/worktrees/
	worktreesDir := filepath.Join(repoPath, ".tasks", "worktrees")
	if entries, err := os.ReadDir(worktreesDir); err == nil {
		for _, e := range entries {
			wtPath := filepath.Join(worktreesDir, e.Name())
			_ = exec.Command("git", "-C", repoPath, "worktree", "remove", "--force", wtPath).Run()
			_ = os.RemoveAll(wtPath)
		}
	}
	_ = exec.Command("git", "-C", repoPath, "worktree", "prune").Run()

	// 3. Checkout default branch in main repo
	_ = exec.Command("git", "-C", repoPath, "checkout", defaultBranch).Run()

	// 4. List all local branches
	out, err := exec.Command("git", "-C", repoPath, "branch", "--format=%(refname:short)").Output()
	if err != nil {
		return nil, fmt.Errorf("erreur liste des branches: %w", err)
	}

	var deleted []string
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		branch := strings.TrimSpace(l)
		if branch == "" || branch == defaultBranch || branch == "main" || branch == "master" {
			continue
		}
		// Force delete local branch
		delCmd := exec.Command("git", "-C", repoPath, "branch", "-D", branch)
		if delOut, delErr := delCmd.CombinedOutput(); delErr == nil {
			deleted = append(deleted, branch)
		} else {
			log.Printf("[GIT] Failed to delete branch %s: %s", branch, string(delOut))
		}
	}

	return &models.CleanBranchesResult{
		RepoPath:        repoPath,
		DefaultBranch:   defaultBranch,
		DeletedBranches: deleted,
		Message:         fmt.Sprintf("%d branches locales nettoyées avec succès.", len(deleted)),
	}, nil
}

func (d *DB) DeleteGitBranch(projectIDOrPath string, branchName string, deleteRemote bool) error {
	branchName = strings.TrimSpace(branchName)
	if branchName == "" {
		return fmt.Errorf("nom de branche requis")
	}
	if branchName == "main" || branchName == "master" || branchName == "HEAD" {
		return fmt.Errorf("impossible de supprimer la branche principale '%s'", branchName)
	}

	d.mu.RLock()
	repoPath := projectIDOrPath
	if projectIDOrPath != "" {
		if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil && proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	if repoPath == "" {
		settings, _ := d.getSettingsUnsafe()
		if settings != nil && settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}
	d.mu.RUnlock()

	if repoPath == "" {
		repoPath = "."
	}

	// 1. If a worktree is checked out on this branch, remove it
	worktreesDir := filepath.Join(repoPath, ".tasks", "worktrees")
	if entries, err := os.ReadDir(worktreesDir); err == nil {
		for _, e := range entries {
			wtPath := filepath.Join(worktreesDir, e.Name())
			curCmd := exec.Command("git", "-C", wtPath, "rev-parse", "--abbrev-ref", "HEAD")
			if curOut, curErr := curCmd.Output(); curErr == nil && strings.TrimSpace(string(curOut)) == branchName {
				_ = exec.Command("git", "-C", repoPath, "worktree", "remove", "--force", wtPath).Run()
				_ = os.RemoveAll(wtPath)
			}
		}
	}
	_ = exec.Command("git", "-C", repoPath, "worktree", "prune").Run()

	// 2. If main repo is currently on this branch, switch to main/master first
	curCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if curOut, curErr := curCmd.Output(); curErr == nil && strings.TrimSpace(string(curOut)) == branchName {
		defaultBranch := "main"
		if err := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "main").Run(); err != nil {
			defaultBranch = "master"
		}
		_ = exec.Command("git", "-C", repoPath, "checkout", defaultBranch).Run()
	}

	// 3. Delete local branch
	delCmd := exec.Command("git", "-C", repoPath, "branch", "-D", branchName)
	if out, err := delCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("erreur suppression branche locale %s: %s", branchName, string(out))
	}

	// 4. Optionally delete remote branch
	if deleteRemote {
		_ = exec.Command("git", "-C", repoPath, "push", "origin", "--delete", branchName).Run()
	}

	return nil
}

func (d *DB) getNextTaskKey(prefix string) (string, error) {
	if prefix == "" {
		prefix = "TASK"
	}
	prefix = strings.ToUpper(strings.TrimSpace(prefix))

	rows, err := d.conn.Query("SELECT key FROM tasks WHERE UPPER(key) LIKE ?", prefix+"-%")
	if err != nil {
		return fmt.Sprintf("%s-1", prefix), nil
	}
	defer rows.Close()

	maxNum := 0
	prefixWithDash := prefix + "-"
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			upperK := strings.ToUpper(strings.TrimSpace(k))
			if strings.HasPrefix(upperK, prefixWithDash) {
				numPart := strings.TrimPrefix(upperK, prefixWithDash)
				if num, err := strconv.Atoi(numPart); err == nil {
					if num > maxNum {
						maxNum = num
					}
				}
			}
		}
	}

	return fmt.Sprintf("%s-%d", prefix, maxNum+1), nil
}

func (d *DB) getNextGithubTaskKey() string {
	rows, err := d.conn.Query("SELECT key FROM tasks WHERE key LIKE '#%' OR key LIKE 'GH-%'")
	if err != nil {
		return "#1"
	}
	defer rows.Close()

	maxNum := 0
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			s := strings.TrimSpace(k)
			s = strings.TrimPrefix(s, "GH-#")
			s = strings.TrimPrefix(s, "gh-")
			s = strings.TrimPrefix(s, "GH-")
			s = strings.TrimPrefix(s, "#")
			if num, err := strconv.Atoi(s); err == nil {
				if num > maxNum {
					maxNum = num
				}
			}
		}
	}
	return fmt.Sprintf("#%d", maxNum+1)
}

// Workflow labels following the AI lifecycle:
// new -> clarified -> specified -> implemented -> reviewed -> finished
var WorkflowLabels = []string{"new", "clarified", "specified", "implemented", "reviewed", "finished", "untouched", "New", "Clarified", "Specified", "Implemented", "Reviewed", "Finished", "Untouched"}

func GetStageLabelForStatus(status models.Status) string {
	switch status {
	case models.StatusToClarify, models.StatusBacklog:
		return "new"
	case models.StatusToSpecify, models.StatusSpecified:
		return "clarified"
	case models.StatusToImplement, models.StatusInProgress:
		return "specified"
	case models.StatusToTest, models.StatusToValidate:
		return "implemented"
	case models.StatusToClose:
		return "reviewed"
	case models.StatusFinished, models.StatusDone:
		return "finished"
	default:
		return "new"
	}
}

func SetWorkflowLabel(existingLabels []string, targetLabel string) []string {
	var result []string
	for _, l := range existingLabels {
		isWorkflow := false
		for _, wl := range []string{"untouched", "new", "clarified", "specified", "implemented", "reviewed", "finished", "closed", "New", "Clarified", "Specified", "Implemented", "Reviewed", "Finished", "Untouched"} {
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
			LinearTeam: "TASK",
			GithubRepo: "",
			RepoPath:   ".",
		}
	}

	projID := req.ProjectID
	proj, _ := d.getProjectByIDUnsafe(projID)
	if proj == nil {
		projects, _ := d.getProjectsUnsafe()
		if len(projects) > 0 {
			proj = &projects[0]
			projID = proj.ID
		}
	}

	linearTeam := settings.LinearTeam
	githubRepo := settings.GithubRepo
	jiraProject := settings.JiraProject
	jiraUrl := settings.JiraUrl
	repoPath := settings.RepoPath
	tracker := settings.IssueTracker
	if tracker == "" {
		tracker = "linear"
	}

	prefix := "TASK"
	if proj != nil {
		if proj.IssueTracker != "" {
			tracker = proj.IssueTracker
		}
		if proj.LinearTeam != "" {
			linearTeam = proj.LinearTeam
		}
		if proj.GithubRepo != "" {
			githubRepo = proj.GithubRepo
		} else if proj.GitRemoteUrl != "" {
			githubRepo = runner.CleanGithubRepo(proj.GitRemoteUrl)
		}
		if proj.JiraProject != "" {
			jiraProject = proj.JiraProject
		}
		if proj.TrackerUrl != "" {
			jiraUrl = proj.TrackerUrl
		}
		if proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
		if tracker == "jira" && proj.JiraProject != "" {
			prefix = proj.JiraProject
		} else if proj.LinearTeam != "" {
			prefix = proj.LinearTeam
		} else if proj.Slug != "" {
			cleanSlug := strings.ToUpper(strings.ReplaceAll(proj.Slug, "-", ""))
			if len(cleanSlug) > 6 {
				prefix = cleanSlug[:6]
			} else {
				prefix = cleanSlug
			}
		}
	}

	// A Jira-tracked project with no explicit key falls back to the slug, which
	// is what earlier Taskacao builds used as the acli --project argument.
	if jiraProject == "" && proj != nil {
		jiraProject = strings.ToUpper(strings.ReplaceAll(proj.Slug, "-", ""))
	}

	// Issue tracker / source defaults to project tracker, but respects requested source if specified
	if req.Source == "" {
		req.Source = tracker
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

	// Real creation via CLI if Linear or GitHub requested by the project
	if req.Source == "linear" {
		created, err := d.runner.CreateLinearIssue(linearTeam, req.Title, req.Description, req.Priority, req.Labels)
		if err == nil && created != nil {
			id = created.ID
			key = created.Key
			extURL = created.ExternalURL
		} else {
			log.Printf("[DB.CreateTask] Warning: Linear issue creation failed: %v. Using fallback key.", err)
			key, _ = d.getNextTaskKey(prefix)
		}
	} else if req.Source == "github" {
		created, err := d.runner.CreateGithubIssue(githubRepo, repoPath, req.Title, req.Description, req.Labels)
		if err == nil && created != nil {
			id = created.ID
			key = created.Key
			extURL = created.ExternalURL
		} else {
			log.Printf("[DB.CreateTask] Warning: GitHub issue creation failed: %v. Using fallback key.", err)
			key = d.getNextGithubTaskKey()
			if githubRepo != "" && strings.HasPrefix(key, "#") {
				cleanNum := strings.TrimPrefix(key, "#")
				url := fmt.Sprintf("https://github.com/%s/issues/%s", runner.CleanGithubRepo(githubRepo), cleanNum)
				extURL = &url
			}
		}
	} else if req.Source == "jira" {
		created, err := d.runner.CreateJiraIssue(jiraProject, repoPath, jiraUrl, req.Title, req.Description, req.Priority, req.Labels)
		if err == nil && created != nil {
			id = created.ID
			key = created.Key
			extURL = created.ExternalURL
			// Jira creates the work item in its own initial status; align it with
			// the Taskacao stage so the board and Jira agree straight away.
			go func(k string, st models.Status, rp string) {
				_ = d.runner.UpdateJiraIssueState(k, st, rp)
			}(key, req.Status, repoPath)
		} else {
			log.Printf("[DB.CreateTask] Warning: Jira issue creation failed: %v. Using fallback key.", err)
			key, _ = d.getNextTaskKey(prefix)
		}
	} else {
		// Local project tracker
		key, _ = d.getNextTaskKey(prefix)
	}

	if req.ExternalURL != nil && *req.ExternalURL != "" {
		extURL = req.ExternalURL
	} else if extURL == nil && req.Source == "github" && githubRepo != "" && strings.HasPrefix(key, "#") {
		cleanNum := strings.TrimPrefix(key, "#")
		url := fmt.Sprintf("https://github.com/%s/issues/%s", runner.CleanGithubRepo(githubRepo), cleanNum)
		extURL = &url
	} else if extURL == nil && req.Source == "jira" && jiraUrl != "" {
		url := fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(jiraUrl, "/"), key)
		extURL = &url
	}

	var maxPos int
	_ = d.conn.QueryRow("SELECT COALESCE(MAX(position), -1) FROM tasks WHERE status = ?", req.Status).Scan(&maxPos)
	newPos := maxPos + 1

	labelsJSON, _ := json.Marshal(req.Labels)
	if req.Labels == nil {
		labelsJSON = []byte("[]")
	}

	_, err := d.conn.Exec(`
		INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, source, external_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, projID, key, req.Title, req.Description, string(req.Status), string(req.Priority), string(labelsJSON), req.Assignee, req.AssigneeAvatar, newPos, req.DueDate, req.Source, extURL, now, now)

	if err != nil {
		return nil, err
	}

	task := &models.Task{
		ID:             id,
		ProjectID:      projID,
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

	oldLabels := existing.Labels
	var removedLabels []string

	if req.ProjectID != nil && *req.ProjectID != "" {
		existing.ProjectID = *req.ProjectID
	}
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Status != nil {
		oldStage := GetStageLabelForStatus(existing.Status)
		existing.Status = *req.Status
		if req.Labels == nil {
			newStage := GetStageLabelForStatus(*req.Status)
			if oldStage != newStage {
				removedLabels = append(removedLabels, oldStage)
			}
			existing.Labels = SetWorkflowLabel(existing.Labels, newStage)
		}
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Labels != nil {
		newMap := make(map[string]bool)
		for _, l := range *req.Labels {
			newMap[strings.ToLower(strings.TrimPrefix(l, "#"))] = true
		}
		for _, ol := range oldLabels {
			cleanOl := strings.ToLower(strings.TrimPrefix(ol, "#"))
			if !newMap[cleanOl] {
				removedLabels = append(removedLabels, ol)
			}
		}
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
	if req.RepoPath != nil {
		trimmed := strings.TrimSpace(*req.RepoPath)
		if trimmed == "" {
			// An empty string is an explicit "inherit again", not a stored path.
			existing.RepoPath = nil
		} else {
			existing.RepoPath = &trimmed
			// Feed the project's list so the next ticket picks it from a menu
			// instead of retyping the path.
			d.registerProjectRepoPathUnsafe(existing.ProjectID, trimmed)
		}
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
		SET project_id = ?, title = ?, description = ?, status = ?, priority = ?, labels = ?, assignee = ?, assignee_avatar = ?, position = ?, due_date = ?, branch_name = ?, pr_url = ?, repo_path = ?, source = ?, external_url = ?, updated_at = ?
		WHERE id = ? OR key = ?
	`, existing.ProjectID, existing.Title, existing.Description, string(existing.Status), string(existing.Priority), string(labelsJSON), existing.Assignee, existing.AssigneeAvatar, existing.Position, existing.DueDate, existing.BranchName, existing.PrURL, repoPathValue(existing.RepoPath), existing.Source, existing.ExternalURL, existing.UpdatedAt, existing.ID, existing.Key)

	if err != nil {
		return nil, err
	}

	// Enqueue async CLI tracker sync in task activities queue whenever task is modified
	if req.Status != nil || req.Labels != nil || req.Title != nil || req.Description != nil || req.Priority != nil {
		d.enqueueTrackerUpdateUnsafe(existing, req.Status, existing.Labels, removedLabels)
	}

	acts, _ := d.getTaskActivitiesUnsafe(existing.ID)
	existing.Activities = acts

	return existing, nil
}

func (d *DB) getSettingsUnsafe() (*models.Settings, error) {
	var s models.Settings
	var detMode, aiProv, aiCmd, repoP, issTrk, linTm, ghRepo, jiraProj, jiraUrl, pClar, pSpec, pImpl, pPR, pPick, editCmd, specFw sql.NullString

	err := d.conn.QueryRow(`
		SELECT id, theme, accent_color, language, density, default_view, detail_mode, user_name, user_email, user_avatar,
		       ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo, jira_project, jira_url,
		       prompt_clarify, prompt_specify, prompt_implement, prompt_create_pr, prompt_pick, editor_command, spec_framework, updated_at
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
		&jiraProj,
		&jiraUrl,
		&pClar,
		&pSpec,
		&pImpl,
		&pPR,
		&pPick,
		&editCmd,
		&specFw,
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
	if jiraProj.Valid {
		s.JiraProject = jiraProj.String
	}
	if jiraUrl.Valid {
		s.JiraUrl = jiraUrl.String
	}
	s.SpecFramework = runner.NormalizeSpecFramework(specFw.String)
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
	if editCmd.Valid && editCmd.String != "" {
		s.EditorCommand = editCmd.String
	} else {
		s.EditorCommand = "code"
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

	oldStage := GetStageLabelForStatus(existing.Status)
	newStage := GetStageLabelForStatus(newStatus)
	var removedLabels []string
	if oldStage != newStage {
		removedLabels = append(removedLabels, oldStage)
	}

	existing.Status = newStatus
	existing.Position = newPosition
	existing.Labels = SetWorkflowLabel(existing.Labels, newStage)
	existing.UpdatedAt = now

	labelsJSON, _ := json.Marshal(existing.Labels)
	_, err = d.conn.Exec(`
		UPDATE tasks
		SET status = ?, labels = ?, position = ?, updated_at = ?
		WHERE id = ? OR key = ?
	`, string(newStatus), string(labelsJSON), newPosition, now, existing.ID, existing.Key)
	if err != nil {
		return nil, err
	}

	// Enqueue async CLI tracker sync in task activities queue
	d.enqueueTrackerUpdateUnsafe(existing, &newStatus, existing.Labels, removedLabels)

	acts, _ := d.getTaskActivitiesUnsafe(existing.ID)
	existing.Activities = acts

	return existing, nil
}

func (d *DB) DeleteTask(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	existing, _ := d.getTaskByIDUnsafe(id)
	if existing != nil {
		_, _ = d.conn.Exec("DELETE FROM task_activities WHERE task_id = ? OR task_id = ?", existing.ID, existing.Key)
		_, err := d.conn.Exec("DELETE FROM tasks WHERE id = ? OR key = ?", existing.ID, existing.Key)
		return err
	}

	_, _ = d.conn.Exec("DELETE FROM task_activities WHERE task_id = ?", id)
	_, err := d.conn.Exec("DELETE FROM tasks WHERE id = ? OR key = ?", id, id)
	return err
}

func (d *DB) getTaskByIDUnsafe(id string) (*models.Task, error) {
	var t models.Task
	var labelsJSON string
	var dueDate, branchName, prURL, repoPath, source, extURL, issueType, parentKey, parentTitle, parentType sql.NullString
	var statusStr, priorityStr string

	err := d.conn.QueryRow(`
		SELECT id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, repo_path, source, external_url, issue_type, parent_key, parent_title, parent_type, created_at, updated_at
		FROM tasks WHERE id = ? OR key = ?
	`, id, id).Scan(
		&t.ID,
		&t.ProjectID,
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
		&repoPath,
		&source,
		&extURL,
		&issueType,
		&parentKey,
		&parentTitle,
		&parentType,
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
	if repoPath.Valid && repoPath.String != "" {
		p := repoPath.String
		t.RepoPath = &p
	}
	if source.Valid && source.String != "" {
		t.Source = source.String
	} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
		t.Source = "github"
	} else if strings.Contains(t.Key, "-") {
		t.Source = "linear"
	} else {
		t.Source = "local"
	}

	if extURL.Valid && extURL.String != "" {
		t.ExternalURL = &extURL.String
	} else {
		t.ExternalURL = d.computeExternalURLUnsafe(&t)
	}

	t.IssueType = issueType.String
	t.ParentKey = parentKey.String
	t.ParentTitle = parentTitle.String
	t.ParentType = parentType.String
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
		INSERT INTO task_activities (id, task_id, skill_id, skill_name, action, status, summary, output, steps, prompt, started_at, completed_at, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, act.ID, act.TaskID, act.SkillID, act.SkillName, act.Action, act.Status, act.Summary, act.Output, string(stepsJSON), act.Prompt, act.StartedAt, act.CompletedAt, act.Error, act.CreatedAt)
	return err
}

func (d *DB) AddTaskActivity(act models.TaskActivity) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.addTaskActivityDirect(act)
}

func (d *DB) getTaskActivitiesUnsafe(taskID string) ([]models.TaskActivity, error) {
	rows, err := d.conn.Query(`
		SELECT id, task_id, skill_id, skill_name, action, status, summary, output, steps, prompt, started_at, completed_at, error, created_at
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
		var prompt, errStr sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(&a.ID, &a.TaskID, &a.SkillID, &a.SkillName, &a.Action, &a.Status, &a.Summary, &a.Output, &stepsJSON, &prompt, &startedAt, &completedAt, &errStr, &a.CreatedAt)
		if err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(stepsJSON), &a.Steps)
		if a.Steps == nil {
			a.Steps = []string{}
		}
		if prompt.Valid {
			a.Prompt = prompt.String
		}
		if errStr.Valid {
			a.Error = errStr.String
		}
		if startedAt.Valid {
			a.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}
		if a.StartedAt != nil && a.CompletedAt != nil {
			dur := a.CompletedAt.Sub(*a.StartedAt)
			if dur.Seconds() < 1 {
				a.Duration = fmt.Sprintf("%dms", dur.Milliseconds())
			} else if dur.Seconds() < 60 {
				a.Duration = fmt.Sprintf("%.1fs", dur.Seconds())
			} else {
				a.Duration = fmt.Sprintf("%dm%ds", int(dur.Minutes()), int(dur.Seconds())%60)
			}
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
	var detMode, aiProv, aiCmd, repoP, issTrk, linTm, ghRepo, jiraProj, jiraUrl, pClar, pSpec, pImpl, pPR, pPick, specFw sql.NullString

	err := d.conn.QueryRow(`
		SELECT id, theme, accent_color, language, density, default_view, detail_mode, user_name, user_email, user_avatar,
		       ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo, jira_project, jira_url,
		       prompt_clarify, prompt_specify, prompt_implement, prompt_create_pr, prompt_pick, editor_command, spec_framework, updated_at
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
		&jiraProj,
		&jiraUrl,
		&pClar,
		&pSpec,
		&pImpl,
		&pPR,
		&pPick,
		&s.EditorCommand,
		&specFw,
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
				UserName:           "Developer",
				UserEmail:          "dev@example.com",
				UserAvatar:         "",
				AIProvider:         "agy",
				AICommandTemplate:  "agy --dangerously-skip-permissions -p \"{prompt}\"",
				RepoPath:           ".",
				IssueTracker:       "local",
				LinearTeam:         "",
				GithubRepo:         "",
				PromptClarify:      "",
				PromptSpecify:      "",
				PromptImplement:    "",
				PromptCreatePR:     "",
				PromptPick:         "",
				EditorCommand:      "code",
				SpecFramework:      "speckit",
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
	if aiCmd.Valid && aiCmd.String != "" {
		s.AICommandTemplate = aiCmd.String
	} else {
		s.AICommandTemplate = "agy --dangerously-skip-permissions -p \"{prompt}\""
	}
	if repoP.Valid {
		s.RepoPath = repoP.String
	} else {
		s.RepoPath = "."
	}
	if issTrk.Valid {
		s.IssueTracker = issTrk.String
	} else {
		s.IssueTracker = "local"
	}
	if linTm.Valid {
		s.LinearTeam = linTm.String
	} else {
		s.LinearTeam = ""
	}
	if ghRepo.Valid {
		s.GithubRepo = ghRepo.String
	} else {
		s.GithubRepo = ""
	}
	if jiraProj.Valid {
		s.JiraProject = jiraProj.String
	}
	if jiraUrl.Valid {
		s.JiraUrl = jiraUrl.String
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
	s.SpecFramework = runner.NormalizeSpecFramework(specFw.String)

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
		if s.JiraProject == "" { s.JiraProject = current.JiraProject }
		if s.JiraUrl == "" { s.JiraUrl = current.JiraUrl }
		if s.PromptClarify == "" { s.PromptClarify = current.PromptClarify }
		if s.PromptSpecify == "" { s.PromptSpecify = current.PromptSpecify }
		if s.PromptImplement == "" { s.PromptImplement = current.PromptImplement }
		if s.PromptCreatePR == "" { s.PromptCreatePR = current.PromptCreatePR }
		if s.PromptPick == "" { s.PromptPick = current.PromptPick }
		if s.EditorCommand == "" { s.EditorCommand = current.EditorCommand }
		if s.SpecFramework == "" { s.SpecFramework = current.SpecFramework }
	}

	if s.Theme == "" { s.Theme = "dark" }
	if s.AccentColor == "" { s.AccentColor = "indigo" }
	if s.Language == "" { s.Language = "fr" }
	if s.Density == "" { s.Density = "standard" }
	if s.DefaultView == "" { s.DefaultView = "board" }
	if s.DetailMode == "" { s.DetailMode = "panel" }
	if s.UserName == "" { s.UserName = "Developer" }
	if s.UserEmail == "" { s.UserEmail = "dev@example.com" }
	if s.AIProvider == "" { s.AIProvider = "agy" }
	if s.AICommandTemplate == "" { s.AICommandTemplate = "agy -p \"{prompt}\"" }
	if s.RepoPath == "" { s.RepoPath = "." }
	if s.IssueTracker == "" { s.IssueTracker = "local" }
	if s.LinearTeam == "" { s.LinearTeam = "" }
	if s.GithubRepo == "" { s.GithubRepo = "" }
	if s.EditorCommand == "" { s.EditorCommand = "code" }
	s.SpecFramework = runner.NormalizeSpecFramework(s.SpecFramework)
	s.JiraProject = strings.ToUpper(strings.TrimSpace(s.JiraProject))

	now := time.Now()
	_, err := d.conn.Exec(`
		INSERT INTO settings (id, theme, accent_color, language, density, default_view, detail_mode, user_name, user_email, user_avatar, ai_provider, ai_command_template, repo_path, issue_tracker, linear_team, github_repo, jira_project, jira_url, prompt_clarify, prompt_specify, prompt_implement, prompt_create_pr, prompt_pick, editor_command, spec_framework, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			jira_project = excluded.jira_project,
			jira_url = excluded.jira_url,
			prompt_clarify = excluded.prompt_clarify,
			prompt_specify = excluded.prompt_specify,
			prompt_implement = excluded.prompt_implement,
			prompt_create_pr = excluded.prompt_create_pr,
			prompt_pick = excluded.prompt_pick,
			editor_command = excluded.editor_command,
			spec_framework = excluded.spec_framework,
			updated_at = excluded.updated_at
	`, s.Theme, s.AccentColor, s.Language, s.Density, s.DefaultView, s.DetailMode, s.UserName, s.UserEmail, s.UserAvatar, s.AIProvider, s.AICommandTemplate, s.RepoPath, s.IssueTracker, s.LinearTeam, s.GithubRepo, s.JiraProject, s.JiraUrl, s.PromptClarify, s.PromptSpecify, s.PromptImplement, s.PromptCreatePR, s.PromptPick, s.EditorCommand, s.SpecFramework, now)

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

func (d *DB) startQueueWorker() {
	for job := range d.jobQueue {
		d.processSkillJob(job)
	}
}

func (d *DB) processSkillJob(job SkillJob) {
	// 1. Check if activity was canceled before starting
	d.mu.RLock()
	var currentStatus string
	_ = d.conn.QueryRow("SELECT status FROM task_activities WHERE id = ?", job.ActivityID).Scan(&currentStatus)
	d.mu.RUnlock()

	if currentStatus == string(models.ActivityStatusCanceled) {
		return
	}

	// 2. Mark activity as running
	now := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	d.cancelMu.Lock()
	d.cancelMap[job.ActivityID] = cancel
	d.cancelMu.Unlock()

	defer func() {
		cancel()
		d.cancelMu.Lock()
		delete(d.cancelMap, job.ActivityID)
		d.cancelMu.Unlock()
	}()

	d.mu.Lock()
	_, _ = d.conn.Exec(`
		UPDATE task_activities
		SET status = 'running', started_at = ?
		WHERE id = ?
	`, now, job.ActivityID)
	d.mu.Unlock()

	// 3. Special handling for background Sync jobs
	if strings.HasPrefix(job.SkillID, "sync_") || job.SkillID == "sync_all" {
		d.mu.RLock()
		settings, _ := d.getSettingsUnsafe()
		d.mu.RUnlock()
		if settings == nil {
			settings = &models.Settings{
				AIProvider: "agy",
				RepoPath:   ".",
			}
		}
		d.processSyncJob(ctx, job, settings)
		return
	}

	// 3b. Special handling for background Tracker update jobs
	if job.SkillID == "tracker_update" {
		d.processTrackerUpdateJob(ctx, job)
		return
	}

	// 4. Fetch latest task and settings
	d.mu.RLock()
	task, err := d.getTaskByIDUnsafe(job.TaskID)
	settings, _ := d.getSettingsUnsafe()
	d.mu.RUnlock()

	if err != nil || task == nil {
		d.mu.Lock()
		errMsg := "Task not found during execution"
		_, _ = d.conn.Exec(`
			UPDATE task_activities
			SET status = 'failed', error = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, errMsg, job.ActivityID)
		d.mu.Unlock()
		return
	}

	if settings == nil {
		settings = &models.Settings{
			AIProvider: "agy",
			RepoPath:   ".",
		}
	}

	// Dynamic per-project configuration override. This must cover the AI engine
	// too: a project configured for Claude was previously executed with the
	// global provider (agy), silently ignoring its own setting.
	if task.ProjectID != "" {
		if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil {
			if proj.RepoPath != "" {
				settings.RepoPath = proj.RepoPath
			}
			if proj.LinearTeam != "" {
				settings.LinearTeam = proj.LinearTeam
			}
			if proj.GithubRepo != "" {
				settings.GithubRepo = proj.GithubRepo
			}
			if proj.JiraProject != "" {
				settings.JiraProject = proj.JiraProject
			}
			if proj.TrackerUrl != "" {
				settings.JiraUrl = proj.TrackerUrl
			}
			if proj.IssueTracker != "" {
				settings.IssueTracker = proj.IssueTracker
			}
			if proj.AIProvider != "" {
				settings.AIProvider = proj.AIProvider
			}
			if proj.AICommandTemplate != "" {
				settings.AICommandTemplate = proj.AICommandTemplate
			}
			if proj.SpecFramework != "" {
				settings.SpecFramework = proj.SpecFramework
			}

			// A project may point a workflow stage at a different skill than the
			// scaffolded default (for instance /clarify-workitem instead of
			// /clarify-issue). The executed slash command has to follow the
			// override, otherwise the board shows one command and runs another.
			applySkillCommandOverride(settings, proj, job.SkillID)
		}
	}

	// A single ticket may pin its own repository, which wins over the project's
	// path: on a tracker where one epic spans several codebases, the project
	// path would send the agent into the wrong checkout.
	if p := repoPathValue(task.RepoPath); p != "" {
		settings.RepoPath = p
	}

	var skill models.Skill
	for _, s := range d.GetAvailableSkills() {
		if s.ID == job.SkillID || (job.SkillID == "review" && s.ID == "create_pr") {
			skill = s
			break
		}
	}

	mainRepoPath := settings.RepoPath

	// 5. Dynamically acquire or create the dedicated Git Worktree for this task
	executionDir := mainRepoPath
	var worktreeStep string
	if mainRepoPath != "" {
		wtPath, branch, wtErr := d.EnsureTaskWorktree(mainRepoPath, task)
		if wtErr != nil {
			worktreeStep = fmt.Sprintf("⚠️ Avertissement Worktree : %v (repli sur %s)", wtErr, filepath.Base(mainRepoPath))
		} else if wtPath != "" {
			executionDir = wtPath
			worktreeStep = fmt.Sprintf("🌳 Worktree Git isolé actif : .tasks/worktrees/%s (branche: %s)", task.Key, branch)
		}
	}

	// Override settings RepoPath with the isolated worktree directory for AI runner
	runnerSettings := *settings
	runnerSettings.RepoPath = executionDir

	// Run AI execution via runner in the isolated worktree
	realAIOutput, runnerSteps, execErr := d.runner.RunAI(&runnerSettings, job.SkillID, task, job.Prompt)
	completedTime := time.Now()

	if worktreeStep != "" {
		runnerSteps = append([]string{worktreeStep}, runnerSteps...)
	}

	// Check if canceled during execution
	select {
	case <-ctx.Done():
		d.mu.Lock()
		_, _ = d.conn.Exec(`
			UPDATE task_activities
			SET status = 'canceled', summary = 'Exécution annulée par l''utilisateur', completed_at = ?
			WHERE id = ?
		`, completedTime, job.ActivityID)
		d.mu.Unlock()
		return
	default:
	}

	// Steps construction
	var steps []string
	steps = append(steps, fmt.Sprintf("Prise en charge par le moteur d'exécution (%s)", strings.ToUpper(settings.AIProvider)))
	steps = append(steps, runnerSteps...)

	var summary string
	var action string

	if execErr != nil {
		steps = append(steps, fmt.Sprintf("⚠️ Erreur : %v", execErr))
		stepsJSON, _ := json.Marshal(steps)
		d.mu.Lock()
		_, _ = d.conn.Exec(`
			UPDATE task_activities
			SET status = 'failed', summary = ?, output = ?, steps = ?, error = ?, completed_at = ?
			WHERE id = ?
		`, "Échec de l'exécution de la skill", realAIOutput, string(stepsJSON), execErr.Error(), completedTime, job.ActivityID)
		d.mu.Unlock()
		return
	}

	// Determine project stage mapping
	resolveMappedStatus := func(targetStage string, defaultStatus models.Status) models.Status {
		if task.ProjectID != "" {
			if proj, _ := d.getProjectByIDUnsafe(task.ProjectID); proj != nil && len(proj.StageMapping) > 0 {
				if mapped, ok := proj.StageMapping[targetStage]; ok && mapped != "" {
					return models.Status(mapped)
				}
			}
		}
		return defaultStatus
	}

	// Determine next status & workflow labels
	switch skill.ID {
	case "clarify":
		task.Status = resolveMappedStatus("clarified", models.StatusToSpecify)
		task.Labels = SetWorkflowLabel(task.Labels, "clarified")
		action = fmt.Sprintf("Clarification exécutée avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		summary = fmt.Sprintf("Questions de cadrage générées ➔ Étape: %s [Label: #clarified]", task.Status)

	case "specify":
		task.Status = resolveMappedStatus("specified", models.StatusToImplement)
		task.Labels = SetWorkflowLabel(task.Labels, "specified")
		action = fmt.Sprintf("Spécification Speckit rédigée avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		summary = fmt.Sprintf("Spec technique créée sur la branche %s ➔ Étape: %s [Label: #specified]", *task.BranchName, task.Status)

	case "implement":
		task.Status = resolveMappedStatus("implemented", models.StatusToTest)
		task.Labels = SetWorkflowLabel(task.Labels, "implemented")
		action = fmt.Sprintf("Implémentation exécutée avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
		summary = fmt.Sprintf("Développement terminé sur la branche %s ➔ Étape: %s [Label: #implemented]", *task.BranchName, task.Status)

		// Check if changes exist in worktree to stage and commit
		if executionDir != "" && task.BranchName != nil {
			diffCheck := exec.Command("git", "-C", executionDir, "status", "--porcelain")
			if diffOut, err := diffCheck.Output(); err == nil && len(strings.TrimSpace(string(diffOut))) > 0 {
				_ = exec.Command("git", "-C", executionDir, "add", "-A").Run()
				commitMsg := fmt.Sprintf("feat(%s): %s", task.Key, task.Title)
				_ = exec.Command("git", "-C", executionDir, "commit", "-m", commitMsg).Run()
			}
		}

	case "create_pr", "review":
		task.Status = resolveMappedStatus("reviewed", models.StatusToClose)
		task.Labels = SetWorkflowLabel(task.Labels, "reviewed")

		// Check if remote repository is configured
		hasRemote := false
		if mainRepoPath != "" {
			remotesCmd := exec.Command("git", "-C", mainRepoPath, "remote")
			if remOut, err := remotesCmd.Output(); err == nil && len(strings.TrimSpace(string(remOut))) > 0 {
				hasRemote = true
			}
		}

		if hasRemote && settings.GithubRepo != "" {
			prURL := fmt.Sprintf("https://github.com/%s/pull/%s", settings.GithubRepo, strings.TrimPrefix(task.Key, "FRE-"))
			task.PrURL = &prURL
			action = fmt.Sprintf("Revue & Pull Request préparées avec %s (%s)", strings.ToUpper(settings.AIProvider), skill.Command)
			summary = fmt.Sprintf("PR prête pour revue : %s ➔ Étape: À fermer [Label: Reviewed]", prURL)
		} else {
			// No remote configured: Perform safe local git merge into main/master in the main repository
			baseBranch := "main"
			if mainRepoPath != "" {
				if err := exec.Command("git", "-C", mainRepoPath, "rev-parse", "--verify", "main").Run(); err != nil {
					if err2 := exec.Command("git", "-C", mainRepoPath, "rev-parse", "--verify", "master").Run(); err2 == nil {
						baseBranch = "master"
					}
				}

				if task.BranchName != nil && *task.BranchName != "" {
					_ = exec.Command("git", "-C", mainRepoPath, "checkout", baseBranch).Run()
					mergeMsg := fmt.Sprintf("Merge branch '%s' for %s: %s", *task.BranchName, task.Key, task.Title)
					mergeCmd := exec.Command("git", "-C", mainRepoPath, "merge", "--no-ff", *task.BranchName, "-m", mergeMsg)
					if _, mErr := mergeCmd.CombinedOutput(); mErr != nil {
						_ = exec.Command("git", "-C", mainRepoPath, "merge", *task.BranchName).Run()
					}
				}
			}
			branchDisplay := "active"
			if task.BranchName != nil {
				branchDisplay = *task.BranchName
			}
			action = fmt.Sprintf("Fusion locale Git (%s ➔ %s) exécutée avec %s", branchDisplay, baseBranch, strings.ToUpper(settings.AIProvider))
			summary = fmt.Sprintf("Branche '%s' fusionnée localement dans '%s' (aucun remote configuré) ➔ Étape: À fermer [Label: Reviewed]", branchDisplay, baseBranch)
		}

		// Clean up temporary worktree after PR / review
		if mainRepoPath != "" && task.Key != "" {
			_ = d.RemoveTaskWorktree(mainRepoPath, task.Key)
		}

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
		action = fmt.Sprintf("Auto-Pilot exécuté avec %s ➔ %s", strings.ToUpper(settings.AIProvider), task.Status)
		summary = fmt.Sprintf("Statut mis à jour vers '%s' [Label: %s]", task.Status, targetLabel)
	}

	task.UpdatedAt = completedTime

	// Save task and update activity in SQLite
	d.mu.Lock()
	labelsJSON, _ := json.Marshal(task.Labels)
	_, _ = d.conn.Exec(`
		UPDATE tasks
		SET status = ?, labels = ?, branch_name = ?, pr_url = ?, updated_at = ?
		WHERE id = ? OR key = ?
	`, string(task.Status), string(labelsJSON), task.BranchName, task.PrURL, completedTime, task.ID, task.Key)

	stepsJSON, _ := json.Marshal(steps)
	_, _ = d.conn.Exec(`
		UPDATE task_activities
		SET action = ?, status = 'completed', summary = ?, output = ?, steps = ?, completed_at = ?
		WHERE id = ?
	`, action, summary, realAIOutput, string(stepsJSON), completedTime, job.ActivityID)
	d.mu.Unlock()

	// Background state, label, and report comment sync with Linear / GitHub / Jira CLI
	if task.Source == "linear" || task.Source == "github" || task.Source == "jira" || strings.HasPrefix(task.Key, "FRE-") || strings.HasPrefix(task.Key, "#") || strings.HasPrefix(task.Key, "gh-") || strings.HasPrefix(task.Key, "GH-#") {
		var commentHeader string
		switch skill.ID {
		case "clarify":
			commentHeader = "### 💬 [Taskacao] Rapport de Clarification\n\n"
		case "specify":
			commentHeader = "### 📋 [Taskacao] Spécification Technique & Plan d'Implémentation\n\n"
		case "implement":
			commentHeader = "### ⚡ [Taskacao] Rapport d'Implémentation\n\n"
		case "create_pr", "review":
			commentHeader = "### 🚀 [Taskacao] Revue de Code & Préparation PR\n\n"
		default:
			commentHeader = fmt.Sprintf("### 🤖 [Taskacao] Rapport d'exécution : %s\n\n", skill.Name)
		}

		commentBody := commentHeader + realAIOutput

		go func(src, repo, rPath, key, body string, st models.Status, lbls []string) {
			if src == "linear" || strings.HasPrefix(key, "FRE-") {
				_ = d.runner.UpdateLinearIssueState(key, st)
				_ = d.runner.UpdateLinearIssue(key, nil, nil, nil, &st, lbls)
				if strings.TrimSpace(body) != "" {
					_ = d.runner.AddIssueComment(src, repo, rPath, key, body)
				}
			} else if src == "jira" {
				_ = d.runner.UpdateJiraIssue(key, rPath, nil, nil, nil, &st, lbls, nil)
				if strings.TrimSpace(body) != "" {
					_ = d.runner.AddIssueComment(src, repo, rPath, key, body)
				}
			} else if src == "github" || strings.HasPrefix(key, "#") || strings.HasPrefix(key, "gh-") || strings.HasPrefix(key, "GH-#") {
				_ = d.runner.UpdateGithubIssueState(repo, rPath, key, st)
				_ = d.runner.UpdateGithubIssue(repo, rPath, key, nil, nil, &st, lbls, nil)
				if strings.TrimSpace(body) != "" {
					_ = d.runner.AddIssueComment(src, repo, rPath, key, body)
				}
			}
		}(task.Source, settings.GithubRepo, settings.RepoPath, task.Key, commentBody, task.Status, task.Labels)
	}
}

func (d *DB) processSyncJob(ctx context.Context, job SkillJob, settings *models.Settings) {
	var steps []string
	var summary string
	var outputLines []string
	var hasError bool
	var totalImported int

	switch job.SkillID {
	case "sync_linear":
		team := ""
		if job.ProjectID != "" {
			if p, _ := d.getProjectByIDUnsafe(job.ProjectID); p != nil {
				team = p.LinearTeam
			}
		}
		if team == "" && job.Prompt != "" {
			team = job.Prompt
		}
		if team == "" {
			team = settings.LinearTeam
		}
		steps = append(steps, fmt.Sprintf("1. Connecting to Linear CLI for team %s...", team))
		outputLines = append(outputLines, fmt.Sprintf("### 🔄 Linear Synchronization (Team: %s)\n", team))

		tasks, err := d.runner.SyncFromLinear(team)
		if err != nil {
			hasError = true
			errMsg := fmt.Sprintf("Linear synchronization failed: %v", err)
			steps = append(steps, "⚠️ "+errMsg)
			outputLines = append(outputLines, "**Error:** "+errMsg)
			summary = "Error during Linear sync"
		} else {
			steps = append(steps, fmt.Sprintf("2. %d tickets fetched from Linear", len(tasks)))
			if job.ProjectID != "" {
				for i := range tasks {
					tasks[i].ProjectID = job.ProjectID
				}
			}
			if impErr := d.ImportOrUpdateTasks(tasks); impErr != nil {
				hasError = true
				steps = append(steps, "⚠️ 3. Local database write failed: "+impErr.Error())
				outputLines = append(outputLines, "**Error:** "+impErr.Error())
			} else {
				steps = append(steps, "3. Local database updated successfully")
			}
			totalImported = len(tasks)
			summary = fmt.Sprintf("%d Linear issues synchronized successfully", len(tasks))

			outputLines = append(outputLines, fmt.Sprintf("✅ **%d tickets imported / updated from Linear:**\n", len(tasks)))
			for _, t := range tasks {
				outputLines = append(outputLines, fmt.Sprintf("- **[%s]** %s *(Status: %s, Priority: %s)*", t.Key, t.Title, t.Status, t.Priority))
			}
		}

	case "sync_github":
		repo := ""
		repoPath := ""
		if job.ProjectID != "" {
			if p, _ := d.getProjectByIDUnsafe(job.ProjectID); p != nil {
				repo = p.GithubRepo
				repoPath = p.RepoPath
			}
		}
		if repo == "" && job.Prompt != "" {
			repo = job.Prompt
		}
		if repo == "" {
			repo = settings.GithubRepo
		}
		if repoPath == "" {
			repoPath = settings.RepoPath
		}
		steps = append(steps, fmt.Sprintf("1. Connecting to GitHub CLI for repository %s...", repo))
		outputLines = append(outputLines, fmt.Sprintf("### 🐙 GitHub Synchronization (%s)\n", repo))

		tasks, err := d.runner.SyncFromGithub(repo, repoPath)
		if err != nil {
			hasError = true
			errMsg := fmt.Sprintf("GitHub synchronization failed: %v", err)
			steps = append(steps, "⚠️ "+errMsg)
			outputLines = append(outputLines, "**Error:** "+errMsg)
			summary = "Error during GitHub sync"
		} else {
			steps = append(steps, fmt.Sprintf("2. %d tickets fetched from GitHub Issues", len(tasks)))
			if job.ProjectID != "" {
				for i := range tasks {
					tasks[i].ProjectID = job.ProjectID
				}
			}
			if impErr := d.ImportOrUpdateTasks(tasks); impErr != nil {
				hasError = true
				steps = append(steps, "⚠️ 3. Local database write failed: "+impErr.Error())
				outputLines = append(outputLines, "**Error:** "+impErr.Error())
			} else {
				steps = append(steps, "3. Local database updated successfully")
			}
			totalImported = len(tasks)
			summary = fmt.Sprintf("%d GitHub issues synchronized successfully", len(tasks))

			outputLines = append(outputLines, fmt.Sprintf("✅ **%d tickets imported / updated from GitHub:**\n", len(tasks)))
			for _, t := range tasks {
				outputLines = append(outputLines, fmt.Sprintf("- **[%s]** %s *(Status: %s, Priority: %s)*", t.Key, t.Title, t.Status, t.Priority))
			}
		}

	case "sync_jira":
		projectKey := ""
		trackerUrl := ""
		repoPath := ""
		if job.ProjectID != "" {
			if p, _ := d.getProjectByIDUnsafe(job.ProjectID); p != nil {
				projectKey = jiraProjectKeyFor(p)
				trackerUrl = p.TrackerUrl
				repoPath = p.RepoPath
			}
		}
		if projectKey == "" && job.Prompt != "" {
			projectKey = strings.ToUpper(strings.TrimSpace(job.Prompt))
		}
		if projectKey == "" {
			projectKey = settings.JiraProject
		}
		if trackerUrl == "" {
			trackerUrl = settings.JiraUrl
		}
		if repoPath == "" {
			repoPath = settings.RepoPath
		}
		steps = append(steps, fmt.Sprintf("1. Connecting to Atlassian / Jira CLI for project %s...", projectKey))
		outputLines = append(outputLines, fmt.Sprintf("### 🔷 Jira Synchronization (Project: %s)\n", projectKey))

		tasks, err := d.runner.SyncFromJira(projectKey, repoPath, trackerUrl)
		if err != nil {
			hasError = true
			errMsg := fmt.Sprintf("Jira synchronization failed: %v", err)
			steps = append(steps, "⚠️ "+errMsg)
			outputLines = append(outputLines, "**Error:** "+errMsg)
			summary = "Error during Jira sync"
		} else {
			steps = append(steps, fmt.Sprintf("2. %d tickets fetched from Jira", len(tasks)))
			if job.ProjectID != "" {
				for i := range tasks {
					tasks[i].ProjectID = job.ProjectID
				}
			}
			if impErr := d.ImportOrUpdateTasks(tasks); impErr != nil {
				hasError = true
				steps = append(steps, "⚠️ 3. Local database write failed: "+impErr.Error())
				outputLines = append(outputLines, "**Error:** "+impErr.Error())
			} else {
				steps = append(steps, "3. Local database updated successfully")
			}
			totalImported = len(tasks)
			summary = fmt.Sprintf("%d Jira issues synchronized successfully", len(tasks))

			outputLines = append(outputLines, fmt.Sprintf("✅ **%d tickets imported / updated from Jira:**\n", len(tasks)))
			for _, t := range tasks {
				outputLines = append(outputLines, fmt.Sprintf("- **[%s]** %s *(Status: %s, Priority: %s)*", t.Key, t.Title, t.Status, t.Priority))
			}
		}

	case "sync_all":
		steps = append(steps, "1. Starting global multi-tracker synchronization...")
		outputLines = append(outputLines, "### 🌐 Global Synchronization\n")

		projects, _ := d.getProjectsUnsafe()
		if len(projects) == 0 {
			projects = []models.Project{
				{
					ID:           "default",
					LinearTeam:   settings.LinearTeam,
					GithubRepo:   settings.GithubRepo,
					RepoPath:     settings.RepoPath,
					IssueTracker: settings.IssueTracker,
				},
			}
		}

		for _, p := range projects {
			switch p.IssueTracker {
			case "linear":
				if p.LinearTeam != "" || settings.LinearTeam != "" {
					tm := p.LinearTeam
					if tm == "" {
						tm = settings.LinearTeam
					}
					linTasks, linErr := d.runner.SyncFromLinear(tm)
					if linErr != nil {
						steps = append(steps, fmt.Sprintf("⚠️ Linear (%s): %v", tm, linErr))
						outputLines = append(outputLines, fmt.Sprintf("❌ Linear (%s): %v", tm, linErr))
					} else {
						for i := range linTasks {
							linTasks[i].ProjectID = p.ID
						}
						if impErr := d.ImportOrUpdateTasks(linTasks); impErr != nil {
							hasError = true
							steps = append(steps, fmt.Sprintf("⚠️ Linear: écriture locale échouée: %v", impErr))
						}
						steps = append(steps, fmt.Sprintf("✅ Linear (%s): %d issues imported", tm, len(linTasks)))
						outputLines = append(outputLines, fmt.Sprintf("✅ Linear (%s): %d issues synced", tm, len(linTasks)))
						totalImported += len(linTasks)
					}
				}
			case "github":
				ghRepo := p.GithubRepo
				if ghRepo == "" {
					ghRepo = settings.GithubRepo
				}
				ghPath := p.RepoPath
				if ghPath == "" {
					ghPath = settings.RepoPath
				}
				if ghRepo != "" || ghPath != "" {
					ghTasks, ghErr := d.runner.SyncFromGithub(ghRepo, ghPath)
					if ghErr != nil {
						steps = append(steps, fmt.Sprintf("⚠️ GitHub (%s): %v", ghRepo, ghErr))
						outputLines = append(outputLines, fmt.Sprintf("❌ GitHub (%s): %v", ghRepo, ghErr))
					} else {
						for i := range ghTasks {
							ghTasks[i].ProjectID = p.ID
						}
						if impErr := d.ImportOrUpdateTasks(ghTasks); impErr != nil {
							hasError = true
							steps = append(steps, fmt.Sprintf("⚠️ GitHub: écriture locale échouée: %v", impErr))
						}
						steps = append(steps, fmt.Sprintf("✅ GitHub (%s): %d issues imported", ghRepo, len(ghTasks)))
						outputLines = append(outputLines, fmt.Sprintf("✅ GitHub (%s): %d issues synced", ghRepo, len(ghTasks)))
						totalImported += len(ghTasks)
					}
				}
			case "jira":
				jKey := jiraProjectKeyFor(&p)
				jUrl := p.TrackerUrl
				jPath := p.RepoPath
				if jKey == "" {
					jKey = settings.JiraProject
				}
				if jUrl == "" {
					jUrl = settings.JiraUrl
				}
				if jPath == "" {
					jPath = settings.RepoPath
				}
				jTasks, jErr := d.runner.SyncFromJira(jKey, jPath, jUrl)
				if jErr != nil {
					steps = append(steps, fmt.Sprintf("⚠️ Jira (%s): %v", jKey, jErr))
					outputLines = append(outputLines, fmt.Sprintf("❌ Jira (%s): %v", jKey, jErr))
				} else {
					for i := range jTasks {
						jTasks[i].ProjectID = p.ID
					}
					if impErr := d.ImportOrUpdateTasks(jTasks); impErr != nil {
						hasError = true
						steps = append(steps, fmt.Sprintf("⚠️ Jira: écriture locale échouée: %v", impErr))
					}
					steps = append(steps, fmt.Sprintf("✅ Jira (%s): %d issues imported", jKey, len(jTasks)))
					outputLines = append(outputLines, fmt.Sprintf("✅ Jira (%s): %d issues synced", jKey, len(jTasks)))
					totalImported += len(jTasks)
				}
			}
		}

		steps = append(steps, "Global synchronization completed")
		summary = fmt.Sprintf("Global synchronization finished (%d tickets updated)", totalImported)
	}

	completedTime := time.Now()
	status := string(models.ActivityStatusCompleted)
	errText := ""
	if hasError {
		status = string(models.ActivityStatusFailed)
		errText = summary
	}

	stepsJSON, _ := json.Marshal(steps)
	d.mu.Lock()
	_, _ = d.conn.Exec(`
		UPDATE task_activities
		SET status = ?, summary = ?, output = ?, steps = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, status, summary, strings.Join(outputLines, "\n"), string(stepsJSON), errText, completedTime, job.ActivityID)
	d.mu.Unlock()
}

func (d *DB) enqueueTrackerUpdateUnsafe(task *models.Task, status *models.Status, labels []string, removedLabels []string) {
	if task == nil {
		return
	}

	proj, _ := d.getProjectByIDUnsafe(task.ProjectID)
	tracker := task.Source
	if proj != nil && proj.IssueTracker != "" && tracker == "" {
		tracker = proj.IssueTracker
	}

	isLinear := tracker == "linear" || task.Source == "linear" || strings.HasPrefix(task.Key, "FRE-")
	isGithub := tracker == "github" || task.Source == "github" || strings.HasPrefix(task.Key, "#") || strings.HasPrefix(task.Key, "gh-") || strings.HasPrefix(task.Key, "GH-")
	isJira := tracker == "jira" || task.Source == "jira"

	if !isLinear && !isGithub && !isJira {
		return
	}

	activityID := uuid.New().String()
	now := time.Now()

	var trackerName string
	var initialSteps []string
	stStr := string(task.Status)
	if status != nil {
		stStr = string(*status)
	}

	var activeStage string
	for _, l := range labels {
		cleanL := strings.TrimPrefix(strings.ToLower(l), "#")
		if cleanL == "new" || cleanL == "clarified" || cleanL == "specified" || cleanL == "implemented" || cleanL == "reviewed" || cleanL == "finished" {
			activeStage = "#" + cleanL
			break
		}
	}

	var changesSummary []string
	if status != nil {
		changesSummary = append(changesSummary, fmt.Sprintf("Statut ➔ %s", *status))
	}
	if len(labels) > 0 {
		changesSummary = append(changesSummary, fmt.Sprintf("Labels : %s", strings.Join(labels, ", ")))
	}
	if len(removedLabels) > 0 {
		changesSummary = append(changesSummary, fmt.Sprintf("Labels retirés : -%s", strings.Join(removedLabels, ", -")))
	}

	if isLinear {
		trackerName = "Linear"
		initialSteps = []string{
			fmt.Sprintf("Mise à jour issue Linear [%s] %s", task.Key, task.Title),
			fmt.Sprintf("Statut cible : %s | Étape IA : %s", stStr, activeStage),
		}
		if len(changesSummary) > 0 {
			initialSteps = append(initialSteps, strings.Join(changesSummary, " | "))
		}
		initialSteps = append(initialSteps, "Poussée dans la file d'attente d'exécution...")
	} else if isJira {
		trackerName = "Jira"
		jiraKey := ""
		if proj != nil && proj.JiraProject != "" {
			jiraKey = proj.JiraProject
		}
		initialSteps = []string{
			fmt.Sprintf("Mise à jour du ticket Jira [%s] (projet %s)", task.Key, jiraKey),
			fmt.Sprintf("Statut cible : %s | Étape IA : %s", stStr, activeStage),
		}
		if len(changesSummary) > 0 {
			initialSteps = append(initialSteps, strings.Join(changesSummary, " | "))
		}
		initialSteps = append(initialSteps, "Poussée dans la file d'attente d'exécution...")
	} else {
		trackerName = "GitHub"
		repo := ""
		if proj != nil && proj.GithubRepo != "" {
			repo = proj.GithubRepo
		}
		initialSteps = []string{
			fmt.Sprintf("Mise à jour issue GitHub [%s] (%s)", task.Key, repo),
			fmt.Sprintf("Statut cible : %s | Étape IA : %s", stStr, activeStage),
		}
		if len(changesSummary) > 0 {
			initialSteps = append(initialSteps, strings.Join(changesSummary, " | "))
		}
		initialSteps = append(initialSteps, "Poussée dans la file d'attente d'exécution...")
	}

	actionTitle := fmt.Sprintf("Sync %s : %s", trackerName, task.Key)
	if activeStage != "" {
		actionTitle = fmt.Sprintf("Sync %s : %s (%s)", trackerName, task.Key, activeStage)
	}

	summaryText := fmt.Sprintf("Mise à jour asynchrone sur %s (%s)", trackerName, stStr)
	if len(changesSummary) > 0 {
		summaryText = fmt.Sprintf("Mise à jour sur %s : %s", trackerName, strings.Join(changesSummary, " | "))
	}

	act := models.TaskActivity{
		ID:        activityID,
		TaskID:    task.ID,
		TaskKey:   task.Key,
		TaskTitle: task.Title,
		SkillID:   "tracker_update",
		SkillName: fmt.Sprintf("Sync %s CLI", trackerName),
		Action:    actionTitle,
		Status:    "queued",
		Summary:   summaryText,
		Output:    "",
		Steps:     initialSteps,
		Prompt:    "",
		CreatedAt: now,
	}

	_ = d.addTaskActivityDirect(act)

	job := SkillJob{
		ActivityID:    activityID,
		TaskID:        task.ID,
		SkillID:       "tracker_update",
		Prompt:        stStr,
		RemovedLabels: removedLabels,
	}

	select {
	case d.jobQueue <- job:
	default:
		go func() {
			d.jobQueue <- job
		}()
	}
}

func (d *DB) processTrackerUpdateJob(ctx context.Context, job SkillJob) {
	d.mu.RLock()
	task, err := d.getTaskByIDUnsafe(job.TaskID)
	settings, _ := d.getSettingsUnsafe()
	d.mu.RUnlock()

	if err != nil || task == nil {
		d.mu.Lock()
		_, _ = d.conn.Exec(`
			UPDATE task_activities
			SET status = 'failed', error = 'Tâche introuvable pour la synchronisation tracker', completed_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, job.ActivityID)
		d.mu.Unlock()
		return
	}

	proj, _ := d.getProjectByIDUnsafe(task.ProjectID)
	repo := ""
	repoPath := ""
	tracker := task.Source
	if proj != nil {
		if proj.GithubRepo != "" { repo = proj.GithubRepo }
		if proj.RepoPath != "" { repoPath = proj.RepoPath }
		if proj.IssueTracker != "" && tracker == "" { tracker = proj.IssueTracker }
	}
	if repo == "" && settings != nil { repo = settings.GithubRepo }
	if repoPath == "" && settings != nil { repoPath = settings.RepoPath }

	isLinear := tracker == "linear" || task.Source == "linear" || strings.HasPrefix(task.Key, "FRE-")
	isGithub := tracker == "github" || task.Source == "github" || strings.HasPrefix(task.Key, "#") || strings.HasPrefix(task.Key, "gh-") || strings.HasPrefix(task.Key, "GH-")
	isJira := tracker == "jira" || task.Source == "jira"

	steps := []string{
		fmt.Sprintf("Démarrage de la mise à jour CLI pour [%s] %s", task.Key, task.Title),
	}
	var outputText string
	var hasError bool

	if isLinear {
		steps = append(steps, fmt.Sprintf("Exécution: linear issue update %s (statut: %s, labels: %v)", task.Key, task.Status, task.Labels))
		err := d.runner.UpdateLinearIssue(task.Key, &task.Title, &task.Description, &task.Priority, &task.Status, task.Labels)
		if err != nil {
			hasError = true
			outputText = fmt.Sprintf("Erreur Linear CLI : %v", err)
			steps = append(steps, fmt.Sprintf("❌ Échec : %v", err))
		} else {
			outputText = fmt.Sprintf("Issue Linear %s synchronisée avec succès (Titre: %s, Statut: %s, Labels: %v)", task.Key, task.Title, task.Status, task.Labels)
			steps = append(steps, fmt.Sprintf("✅ Issue Linear %s mise à jour avec succès via la CLI", task.Key))
		}
	} else if isJira {
		steps = append(steps, fmt.Sprintf("Exécution: acli jira workitem edit %s (Statut: %s, Labels: %v, Supprimés: %v)", task.Key, task.Status, task.Labels, job.RemovedLabels))
		err := d.runner.UpdateJiraIssue(task.Key, repoPath, &task.Title, &task.Description, &task.Priority, &task.Status, task.Labels, job.RemovedLabels)
		if err != nil {
			hasError = true
			outputText = fmt.Sprintf("Erreur Atlassian CLI (acli) : %v", err)
			steps = append(steps, fmt.Sprintf("❌ Échec : %v", err))
		} else {
			outputText = fmt.Sprintf("Ticket Jira %s synchronisé avec succès (Titre: %s, Statut: %s, Labels: %v)", task.Key, task.Title, task.Status, task.Labels)
			steps = append(steps, fmt.Sprintf("✅ Ticket Jira %s mis à jour avec succès via acli", task.Key))
		}
	} else if isGithub {
		steps = append(steps, fmt.Sprintf("Exécution: gh issue edit %s (Dépôt: %s, Statut: %s, Labels: %v, Supprimés: %v)", task.Key, repo, task.Status, task.Labels, job.RemovedLabels))
		err := d.runner.UpdateGithubIssue(repo, repoPath, task.Key, &task.Title, &task.Description, &task.Status, task.Labels, job.RemovedLabels)
		if err != nil {
			hasError = true
			outputText = fmt.Sprintf("Erreur GitHub CLI : %v", err)
			steps = append(steps, fmt.Sprintf("❌ Échec : %v", err))
		} else {
			outputText = fmt.Sprintf("Issue GitHub %s synchronisée avec succès dans %s (Titre: %s, Statut: %s, Labels: %v)", task.Key, repo, task.Title, task.Status, task.Labels)
			steps = append(steps, fmt.Sprintf("✅ Issue GitHub %s synchronisée avec succès via la CLI", task.Key))
		}
	} else {
		outputText = "Aucun tracker distant configuré pour cette tâche"
		steps = append(steps, "ℹ️ Tâche locale uniquement, aucune commande distante requise")
	}

	completedTime := time.Now()
	status := string(models.ActivityStatusCompleted)
	errText := ""
	summary := outputText
	if hasError {
		status = string(models.ActivityStatusFailed)
		errText = outputText
		summary = "Échec de la synchronisation tracker"
	}

	stepsJSON, _ := json.Marshal(steps)
	d.mu.Lock()
	_, _ = d.conn.Exec(`
		UPDATE task_activities
		SET status = ?, summary = ?, output = ?, steps = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, status, summary, outputText, string(stepsJSON), errText, completedTime, job.ActivityID)
	d.mu.Unlock()
}

func (d *DB) EnqueueSync(syncType string, param string, projectID string) (*models.TaskActivity, error) {
	d.mu.RLock()
	settings, _ := d.getSettingsUnsafe()
	var proj *models.Project
	if projectID != "" && projectID != "all" {
		proj, _ = d.getProjectByIDUnsafe(projectID)
	}
	d.mu.RUnlock()

	linearTeam := ""
	githubRepo := ""
	if proj != nil {
		linearTeam = proj.LinearTeam
		githubRepo = proj.GithubRepo
	}
	if linearTeam == "" && settings != nil {
		linearTeam = settings.LinearTeam
	}
	if githubRepo == "" && settings != nil {
		githubRepo = settings.GithubRepo
	}

	activityID := uuid.New().String()
	now := time.Now()

	var skillName string
	var summary string
	var steps []string
	targetTaskID := "sync-" + syncType
	if proj != nil {
		targetTaskID = "sync-" + proj.ID
	}

	switch syncType {
	case "linear", "sync_linear":
		syncType = "sync_linear"
		team := linearTeam
		if param != "" {
			team = param
		}
		skillName = "Sync Linear"
		summary = fmt.Sprintf("Synchronisation Linear (Équipe %s) en file d'attente", team)
		steps = []string{
			fmt.Sprintf("Cible : Linear Workspace (Team: %s)", team),
			"Poussée dans la file d'attente d'exécution...",
		}
	case "github", "sync_github":
		syncType = "sync_github"
		repo := githubRepo
		if param != "" {
			repo = param
		}
		skillName = "Sync GitHub"
		summary = fmt.Sprintf("Synchronisation GitHub (%s) en file d'attente", repo)
		steps = []string{
			fmt.Sprintf("Cible : GitHub Repository (%s)", repo),
			"Poussée dans la file d'attente d'exécution...",
		}
	case "jira", "sync_jira":
		syncType = "sync_jira"
		jKey := jiraProjectKeyFor(proj)
		if param != "" {
			jKey = strings.ToUpper(strings.TrimSpace(param))
		}
		if jKey == "" && settings != nil {
			jKey = settings.JiraProject
		}
		skillName = "Sync Jira"
		summary = fmt.Sprintf("Synchronisation Jira (%s) en file d'attente", jKey)
		steps = []string{
			fmt.Sprintf("Cible : Jira Project (%s)", jKey),
			"Poussée dans la file d'attente d'exécution...",
		}
	default:
		syncType = "sync_all"
		skillName = "Sync Globale"
		summary = "Synchronisation multi-trackers en file d'attente"
		steps = []string{
			"Cibles : Tous les projets et trackers distants configurés",
			"Poussée dans la file d'attente d'exécution...",
		}
	}

	act := models.TaskActivity{
		ID:        activityID,
		TaskID:    targetTaskID,
		SkillID:   syncType,
		SkillName: skillName,
		Action:    "Synchronisation des tickets distants",
		Status:    string(models.ActivityStatusQueued),
		Summary:   summary,
		Output:    "",
		Steps:     steps,
		Prompt:    param,
		CreatedAt: now,
	}

	d.mu.Lock()
	_ = d.addTaskActivityDirect(act)
	d.mu.Unlock()

	// Push job to worker queue
	d.jobQueue <- SkillJob{
		ActivityID: activityID,
		TaskID:     targetTaskID,
		ProjectID:  projectID,
		SkillID:    syncType,
		Prompt:     param,
	}

	return &act, nil
}

func (d *DB) EnqueueSkillOnTask(taskID string, skillID string, prompt string) (*models.Task, *models.TaskActivity, error) {
	d.mu.RLock()
	task, err := d.getTaskByIDUnsafe(taskID)
	d.mu.RUnlock()
	if err != nil || task == nil {
		return nil, nil, fmt.Errorf("task not found: %s", taskID)
	}

	skills := d.GetAvailableSkills()
	var targetSkill *models.Skill
	for _, s := range skills {
		if s.ID == skillID {
			targetSkill = &s
			break
		}
	}
	if targetSkill == nil {
		return nil, nil, fmt.Errorf("skill not found: %s", skillID)
	}

	activityID := uuid.New().String()
	now := time.Now()

	d.mu.RLock()
	skillName := d.resolveSkillNameUnsafe(task.ProjectID, targetSkill.ID, targetSkill.Name)
	d.mu.RUnlock()

	act := models.TaskActivity{
		ID:        activityID,
		TaskID:    task.ID,
		SkillID:   targetSkill.ID,
		SkillName: skillName,
		Action:    fmt.Sprintf("Exécution de la compétence %s", skillName),
		Status:    string(models.ActivityStatusQueued),
		Summary:   fmt.Sprintf("Compétence %s en file d'attente pour la tâche %s", skillName, task.Key),
		Output:    "",
		Steps: []string{
			fmt.Sprintf("Tâche ciblée : %s - %s", task.Key, task.Title),
			"Poussée dans la file d'attente du worker...",
		},
		Prompt:    prompt,
		CreatedAt: now,
	}

	d.mu.Lock()
	_ = d.addTaskActivityDirect(act)
	d.mu.Unlock()

	// Push to background channel worker
	d.jobQueue <- SkillJob{
		ActivityID: activityID,
		TaskID:     task.ID,
		ProjectID:  task.ProjectID,
		SkillID:    targetSkill.ID,
		Prompt:     prompt,
	}

	return task, &act, nil
}

func (d *DB) RunSkillOnTask(taskID string, skillID string, prompt string) (*models.Task, *models.TaskActivity, error) {
	return d.EnqueueSkillOnTask(taskID, skillID, prompt)
}

func (d *DB) GetActivities(projectID, status, skillID, taskID, search string, limit int) ([]models.TaskActivity, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var conditions []string
	var args []interface{}

	if projectID != "" && projectID != "all" {
		conditions = append(conditions, "((t.project_id = ? OR t.project_id = (SELECT slug FROM projects WHERE id = ?) OR t.project_id = (SELECT id FROM projects WHERE slug = ?)) OR a.task_id LIKE ? OR a.prompt LIKE ?)")
		args = append(args, projectID, projectID, projectID, "%"+projectID+"%", "%"+projectID+"%")
	}

	if status != "" && status != "all" {
		if status == "queued" || status == "pending" {
			conditions = append(conditions, "a.status IN ('queued', 'pending')")
		} else {
			conditions = append(conditions, "a.status = ?")
			args = append(args, status)
		}
	}
	if skillID != "" && skillID != "all" {
		conditions = append(conditions, "a.skill_id = ?")
		args = append(args, skillID)
	}
	if taskID != "" {
		conditions = append(conditions, "a.task_id = ?")
		args = append(args, taskID)
	}
	if search != "" {
		conditions = append(conditions, "(a.skill_name LIKE ? OR a.summary LIKE ? OR a.output LIKE ? OR t.key LIKE ? OR t.title LIKE ?)")
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}

	sqlQuery := `
		SELECT a.id, a.task_id, COALESCE(t.key, ''), COALESCE(t.title, ''), a.skill_id, a.skill_name,
		       a.action, a.status, a.summary, a.output, a.steps, a.prompt,
		       a.created_at, a.started_at, a.completed_at, a.error
		FROM task_activities a
		LEFT JOIN tasks t ON a.task_id = t.id
	`
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	sqlQuery += " ORDER BY a.created_at DESC"
	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.conn.Query(sqlQuery, args...)
	if err != nil {
		return []models.TaskActivity{}, err
	}
	defer rows.Close()

	var list []models.TaskActivity
	for rows.Next() {
		var a models.TaskActivity
		var stepsJSON string
		var prompt, errStr sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&a.ID,
			&a.TaskID,
			&a.TaskKey,
			&a.TaskTitle,
			&a.SkillID,
			&a.SkillName,
			&a.Action,
			&a.Status,
			&a.Summary,
			&a.Output,
			&stepsJSON,
			&prompt,
			&a.CreatedAt,
			&startedAt,
			&completedAt,
			&errStr,
		)
		if err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(stepsJSON), &a.Steps)
		if a.Steps == nil {
			a.Steps = []string{}
		}
		if prompt.Valid {
			a.Prompt = prompt.String
		}
		if errStr.Valid {
			a.Error = errStr.String
		}
		if startedAt.Valid {
			a.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}

		if a.StartedAt != nil && a.CompletedAt != nil {
			dur := a.CompletedAt.Sub(*a.StartedAt)
			if dur.Seconds() < 1 {
				a.Duration = fmt.Sprintf("%dms", dur.Milliseconds())
			} else if dur.Seconds() < 60 {
				a.Duration = fmt.Sprintf("%.1fs", dur.Seconds())
			} else {
				a.Duration = fmt.Sprintf("%dm%ds", int(dur.Minutes()), int(dur.Seconds())%60)
			}
		}

		list = append(list, a)
	}

	if list == nil {
		list = []models.TaskActivity{}
	}
	return list, nil
}

func (d *DB) GetActivityByID(id string) (*models.TaskActivity, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var a models.TaskActivity
	var stepsJSON string
	var prompt, errStr sql.NullString
	var startedAt, completedAt sql.NullTime

	err := d.conn.QueryRow(`
		SELECT a.id, a.task_id, COALESCE(t.key, ''), COALESCE(t.title, ''), a.skill_id, a.skill_name,
		       a.action, a.status, a.summary, a.output, a.steps, a.prompt,
		       a.created_at, a.started_at, a.completed_at, a.error
		FROM task_activities a
		LEFT JOIN tasks t ON a.task_id = t.id
		WHERE a.id = ?
	`, id).Scan(
		&a.ID,
		&a.TaskID,
		&a.TaskKey,
		&a.TaskTitle,
		&a.SkillID,
		&a.SkillName,
		&a.Action,
		&a.Status,
		&a.Summary,
		&a.Output,
		&stepsJSON,
		&prompt,
		&a.CreatedAt,
		&startedAt,
		&completedAt,
		&errStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal([]byte(stepsJSON), &a.Steps)
	if a.Steps == nil {
		a.Steps = []string{}
	}
	if prompt.Valid {
		a.Prompt = prompt.String
	}
	if errStr.Valid {
		a.Error = errStr.String
	}
	if startedAt.Valid {
		a.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}

	if a.StartedAt != nil && a.CompletedAt != nil {
		dur := a.CompletedAt.Sub(*a.StartedAt)
		if dur.Seconds() < 1 {
			a.Duration = fmt.Sprintf("%dms", dur.Milliseconds())
		} else if dur.Seconds() < 60 {
			a.Duration = fmt.Sprintf("%.1fs", dur.Seconds())
		} else {
			a.Duration = fmt.Sprintf("%dm%ds", int(dur.Minutes()), int(dur.Seconds())%60)
		}
	}

	return &a, nil
}

func (d *DB) GetActivityStats(projectID string) (*models.ActivityStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var stats models.ActivityStats
	var query string
	var args []interface{}

	if projectID != "" && projectID != "all" {
		query = `
			SELECT a.status, COUNT(*) 
			FROM task_activities a
			LEFT JOIN tasks t ON a.task_id = t.id
			WHERE ((t.project_id = ? OR t.project_id = (SELECT slug FROM projects WHERE id = ?) OR t.project_id = (SELECT id FROM projects WHERE slug = ?)) OR a.task_id LIKE ? OR a.prompt LIKE ?)
			GROUP BY a.status
		`
		args = append(args, projectID, projectID, projectID, "%"+projectID+"%", "%"+projectID+"%")
	} else {
		query = "SELECT status, COUNT(*) FROM task_activities GROUP BY status"
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return &stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var st string
		var cnt int
		if err := rows.Scan(&st, &cnt); err == nil {
			stats.Total += cnt
			switch st {
			case "queued", "pending":
				stats.Queued += cnt
			case "running":
				stats.Running += cnt
			case "completed":
				stats.Completed += cnt
			case "failed":
				stats.Failed += cnt
			case "canceled":
				stats.Canceled += cnt
			}
		}
	}
	return &stats, nil
}

func (d *DB) RetryActivity(activityID string) (*models.TaskActivity, error) {
	act, err := d.GetActivityByID(activityID)
	if err != nil || act == nil {
		return nil, fmt.Errorf("activity not found")
	}

	_, newAct, err := d.EnqueueSkillOnTask(act.TaskID, act.SkillID, act.Prompt)
	return newAct, err
}

func (d *DB) CancelActivity(activityID string) error {
	d.cancelMu.Lock()
	if cancel, exists := d.cancelMap[activityID]; exists {
		cancel()
		delete(d.cancelMap, activityID)
	}
	d.cancelMu.Unlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec(`
		UPDATE task_activities
		SET status = 'canceled', summary = 'Annulée par l''utilisateur', completed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN ('queued', 'pending', 'running')
	`, activityID)
	return err
}

func (d *DB) DeleteActivity(activityID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec("DELETE FROM task_activities WHERE id = ?", activityID)
	return err
}

func (d *DB) ClearCompletedActivities() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	res, err := d.conn.Exec("DELETE FROM task_activities WHERE status IN ('completed', 'failed', 'canceled')")
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	return int(affected), nil
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
	d.mu.RLock()
	task, err := d.getTaskByIDUnsafe(taskID)
	settings, _ := d.getSettingsUnsafe()
	var proj *models.Project
	if task != nil && task.ProjectID != "" {
		proj, _ = d.getProjectByIDUnsafe(task.ProjectID)
	}
	d.mu.RUnlock()

	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

	if settings == nil {
		settings = &models.Settings{
			RepoPath: ".",
		}
	}

	var newKey string
	var extURL *string
	now := time.Now()

	// Ensure stage label is properly set based on current status
	task.Labels = SetWorkflowLabel(task.Labels, GetStageLabelForStatus(task.Status))

	switch target {
	case "linear":
		team := ""
		if proj != nil {
			team = proj.LinearTeam
		}
		if team == "" {
			team = settings.LinearTeam
		}
		created, err := d.runner.CreateLinearIssue(team, task.Title, task.Description, task.Priority, task.Labels)
		if err != nil {
			return nil, fmt.Errorf("création Linear impossible: %w", err)
		}
		newKey = created.Key
		extURL = created.ExternalURL

		// Sync status to Linear if not backlog
		if task.Status != models.StatusBacklog && task.Status != models.StatusToClarify {
			_ = d.runner.UpdateLinearIssueState(newKey, task.Status)
		}

	case "github":
		repo := ""
		repoPath := ""
		if proj != nil {
			repo = proj.GithubRepo
			repoPath = proj.RepoPath
		}
		if repo == "" {
			repo = settings.GithubRepo
		}
		if repoPath == "" {
			repoPath = settings.RepoPath
		}
		created, err := d.runner.CreateGithubIssue(repo, repoPath, task.Title, task.Description, task.Labels)
		if err != nil {
			return nil, fmt.Errorf("création GitHub impossible: %w", err)
		}
		newKey = created.Key
		extURL = created.ExternalURL

		// Sync status if done
		if task.Status == models.StatusDone || task.Status == models.StatusFinished || task.Status == models.StatusToClose {
			_ = d.runner.UpdateGithubIssue(repo, repoPath, newKey, nil, nil, &task.Status, task.Labels, nil)
		}

	case "jira":
		projectKey := ""
		trackerUrl := ""
		repoPath := ""
		if proj != nil {
			projectKey = jiraProjectKeyFor(proj)
			trackerUrl = proj.TrackerUrl
			repoPath = proj.RepoPath
		}
		if projectKey == "" {
			projectKey = settings.JiraProject
		}
		if trackerUrl == "" {
			trackerUrl = settings.JiraUrl
		}
		if repoPath == "" {
			repoPath = settings.RepoPath
		}
		created, err := d.runner.CreateJiraIssue(projectKey, repoPath, trackerUrl, task.Title, task.Description, task.Priority, task.Labels)
		if err != nil {
			return nil, fmt.Errorf("création Jira impossible: %w", err)
		}
		newKey = created.Key
		extURL = created.ExternalURL

		_ = d.runner.UpdateJiraIssueState(newKey, task.Status, repoPath)

	default:
		return nil, fmt.Errorf("tracker distant non supporté: %s (choisir 'linear', 'github' ou 'jira')", target)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

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

	outputMsg := fmt.Sprintf("Tâche locale convertie vers %s.\nClé distante : %s", strings.ToUpper(target), task.Key)
	if extURL != nil {
		outputMsg += fmt.Sprintf("\nURL : %s", *extURL)
	}

	act := models.TaskActivity{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		SkillID:   "convert",
		SkillName: "Export Tracker",
		Action:    fmt.Sprintf("Tâche convertie vers %s (%s)", strings.ToUpper(target), task.Key),
		Status:    string(models.ActivityStatusCompleted),
		Summary:   fmt.Sprintf("Issue distante créée avec succès : %s", task.Key),
		Output:    outputMsg,
		Steps:     []string{fmt.Sprintf("Création du ticket distant sur %s", strings.ToUpper(target)), fmt.Sprintf("Mise à jour de la clé (%s) et de la source", task.Key)},
		CreatedAt: now,
	}
	_ = d.addTaskActivityDirect(act)
	acts, _ := d.getTaskActivitiesUnsafe(task.ID)
	task.Activities = acts

	return task, nil
}

// -------------------------------------------------------------
// PROJECTS CRUD & MANAGEMENT
// -------------------------------------------------------------

// resolveSkillNameUnsafe returns the display name of a skill for a given
// project, honouring the project's SkillOverrides map (skillId -> custom label).
// Without this, a renamed skill would keep its default name everywhere outside
// the project settings form.
func (d *DB) resolveSkillNameUnsafe(projectID string, skillID string, defaultName string) string {
	proj, _ := d.getProjectByIDUnsafe(projectID)
	if proj == nil || proj.SkillOverrides == nil {
		return defaultName
	}
	if override := strings.TrimSpace(proj.SkillOverrides[skillID]); override != "" {
		return override
	}
	return defaultName
}

// applySkillCommandOverride rewrites the default prompt of a workflow stage so
// it invokes the slash command the project configured through SkillOverrides.
// An explicit custom prompt in the settings always wins: the user wrote it on
// purpose and it may already name its own command.
func applySkillCommandOverride(settings *models.Settings, proj *models.Project, skillID string) {
	if proj == nil || proj.SkillOverrides == nil || settings == nil {
		return
	}
	override := strings.TrimSpace(proj.SkillOverrides[skillID])
	if override == "" {
		return
	}
	cmd := "/" + strings.TrimPrefix(override, "/")

	switch skillID {
	case "clarify":
		if settings.PromptClarify == "" {
			settings.PromptClarify = cmd + " {issueKey} tracked on {tracker} in {repo}"
		}
	case "specify":
		if settings.PromptSpecify == "" {
			settings.PromptSpecify = cmd + " {issueKey}"
		}
	case "implement":
		if settings.PromptImplement == "" {
			settings.PromptImplement = cmd + " {issueKey}"
		}
	case "create_pr", "review":
		if settings.PromptCreatePR == "" {
			settings.PromptCreatePR = cmd + " {issueKey}"
		}
	case "pick":
		if settings.PromptPick == "" {
			settings.PromptPick = cmd + " {issueKey}"
		}
	}
}

// jiraProjectKeyFor resolves the Jira project key of a project. Projects
// created before the dedicated jira_project column existed stored nothing, and
// Taskacao used to pass the slug to acli, so the slug remains the fallback.
func jiraProjectKeyFor(p *models.Project) string {
	if p == nil {
		return ""
	}
	if key := strings.ToUpper(strings.TrimSpace(p.JiraProject)); key != "" {
		return key
	}
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(p.Slug), "-", ""))
}

func defaultStageMapping() map[string]string {
	return map[string]string{
		"new":         "to_clarify",
		"untouched":   "to_clarify",
		"clarified":   "to_specify",
		"specified":   "to_implement",
		"implemented": "to_test",
		"reviewed":    "to_test",
		"finished":    "to_close",
	}
}

func ParseGitRepoFromURL(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	raw = strings.TrimSuffix(raw, ".git")
	if strings.HasPrefix(raw, "git@github.com:") {
		return strings.TrimPrefix(raw, "git@github.com:")
	}
	if strings.HasPrefix(raw, "https://github.com/") {
		return strings.TrimPrefix(raw, "https://github.com/")
	}
	if strings.HasPrefix(raw, "http://github.com/") {
		return strings.TrimPrefix(raw, "http://github.com/")
	}
	if strings.HasPrefix(raw, "ssh://git@github.com/") {
		return strings.TrimPrefix(raw, "ssh://git@github.com/")
	}
	return raw
}

// NormalizeProjectType keeps the stored project type to the two values the app
// knows: "personal" (a personal board, the only type the daily digest is served
// for) and "standard" (a delivery project), the default.
func NormalizeProjectType(raw string) string {
	if strings.ToLower(strings.TrimSpace(raw)) == "personal" {
		return "personal"
	}
	return "standard"
}

// parseRepoPaths decodes the project's known working directories, tolerating an
// empty column on projects created before the field existed.
func parseRepoPaths(raw string) []string {
	if strings.TrimSpace(raw) == "" || raw == "[]" {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return normalizeRepoPaths(list)
}

// normalizeRepoPaths trims, drops blanks and de-duplicates while keeping the
// order the paths were added in, so the list stays predictable in the UI.
func normalizeRepoPaths(list []string) []string {
	seen := make(map[string]bool, len(list))
	out := make([]string, 0, len(list))
	for _, p := range list {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if len(out) >= maxProjectRepoPaths {
			break
		}
	}
	return out
}

// maxProjectRepoPaths caps the auto-fed list so a long-lived project does not
// accumulate an unusable dropdown.
const maxProjectRepoPaths = 25

// registerProjectRepoPathUnsafe records a working directory on the project when
// a ticket pins one, so the next ticket can pick it from the list. Caller must
// hold the write lock.
func (d *DB) registerProjectRepoPathUnsafe(projectID string, repoPath string) {
	repoPath = strings.TrimSpace(repoPath)
	if projectID == "" || repoPath == "" {
		return
	}
	proj, err := d.getProjectByIDUnsafe(projectID)
	if err != nil || proj == nil {
		return
	}
	// The project's own repoPath is always offered, no need to store it twice.
	if strings.TrimSpace(proj.RepoPath) == repoPath {
		return
	}
	for _, existing := range proj.RepoPaths {
		if existing == repoPath {
			return
		}
	}
	updated := normalizeRepoPaths(append(proj.RepoPaths, repoPath))
	payload, err := json.Marshal(updated)
	if err != nil {
		return
	}
	_, _ = d.conn.Exec("UPDATE projects SET repo_paths = ?, updated_at = ? WHERE id = ?", string(payload), time.Now(), proj.ID)
}

func (d *DB) getProjectsUnsafe() ([]models.Project, error) {
	rows, err := d.conn.Query(`
		SELECT p.id, p.name, p.slug, p.description, p.icon, p.color, p.repo_path, p.repo_paths, p.git_remote_url, p.linear_team, p.github_repo, p.jira_project, p.issue_tracker, p.tracker_url, p.project_type, p.is_default, p.stage_mapping, p.skill_overrides, p.ai_provider, p.ai_command_template, p.spec_framework, p.created_at, p.updated_at,
		       COUNT(t.id) as task_count
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		GROUP BY p.id
		ORDER BY p.is_default DESC, p.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		var isDefault int
		var stageMappingJSON, skillOverridesJSON, repoPathsJSON string
		var aiProv, aiCmd, specFw, jiraProj, projType sql.NullString
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Icon, &p.Color, &p.RepoPath, &repoPathsJSON, &p.GitRemoteUrl, &p.LinearTeam, &p.GithubRepo, &jiraProj, &p.IssueTracker, &p.TrackerUrl, &projType, &isDefault, &stageMappingJSON, &skillOverridesJSON, &aiProv, &aiCmd, &specFw, &p.CreatedAt, &p.UpdatedAt, &p.TaskCount,
		)
		if err != nil {
			return nil, err
		}
		p.IsDefault = isDefault == 1
		p.StageMapping = defaultStageMapping()
		if stageMappingJSON != "" && stageMappingJSON != "{}" {
			_ = json.Unmarshal([]byte(stageMappingJSON), &p.StageMapping)
		}
		p.SkillOverrides = map[string]string{}
		if skillOverridesJSON != "" && skillOverridesJSON != "{}" {
			_ = json.Unmarshal([]byte(skillOverridesJSON), &p.SkillOverrides)
		}
		p.RepoPaths = parseRepoPaths(repoPathsJSON)
		if aiProv.Valid {
			p.AIProvider = aiProv.String
		}
		if aiCmd.Valid {
			p.AICommandTemplate = aiCmd.String
		}
		if jiraProj.Valid {
			p.JiraProject = jiraProj.String
		}
		p.SpecFramework = runner.NormalizeSpecFramework(specFw.String)
		p.ProjectType = NormalizeProjectType(projType.String)
		projects = append(projects, p)
	}
	if projects == nil {
		projects = []models.Project{}
	}
	return projects, nil
}

func (d *DB) GetProjects() ([]models.Project, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getProjectsUnsafe()
}

func (d *DB) GetProjectByID(id string) (*models.Project, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getProjectByIDUnsafe(id)
}

func (d *DB) getProjectByIDUnsafe(id string) (*models.Project, error) {
	var p models.Project
	var isDefault int
	var stageMappingJSON, skillOverridesJSON, repoPathsJSON string
	var aiProv, aiCmd, specFw, jiraProj, projType sql.NullString
	err := d.conn.QueryRow(`
		SELECT p.id, p.name, p.slug, p.description, p.icon, p.color, p.repo_path, p.repo_paths, p.git_remote_url, p.linear_team, p.github_repo, p.jira_project, p.issue_tracker, p.tracker_url, p.project_type, p.is_default, p.stage_mapping, p.skill_overrides, p.ai_provider, p.ai_command_template, p.spec_framework, p.created_at, p.updated_at,
		       (SELECT COUNT(*) FROM tasks WHERE project_id = p.id) as task_count
		FROM projects p
		WHERE p.id = ? OR p.slug = ?
	`, id, id).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Icon, &p.Color, &p.RepoPath, &repoPathsJSON, &p.GitRemoteUrl, &p.LinearTeam, &p.GithubRepo, &jiraProj, &p.IssueTracker, &p.TrackerUrl, &projType, &isDefault, &stageMappingJSON, &skillOverridesJSON, &aiProv, &aiCmd, &specFw, &p.CreatedAt, &p.UpdatedAt, &p.TaskCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.StageMapping = defaultStageMapping()
	if stageMappingJSON != "" && stageMappingJSON != "{}" {
		_ = json.Unmarshal([]byte(stageMappingJSON), &p.StageMapping)
	}
	p.SkillOverrides = map[string]string{}
	if skillOverridesJSON != "" && skillOverridesJSON != "{}" {
		_ = json.Unmarshal([]byte(skillOverridesJSON), &p.SkillOverrides)
	}
	p.RepoPaths = parseRepoPaths(repoPathsJSON)
	if aiProv.Valid {
		p.AIProvider = aiProv.String
	}
	if aiCmd.Valid {
		p.AICommandTemplate = aiCmd.String
	}
	if jiraProj.Valid {
		p.JiraProject = jiraProj.String
	}
	p.SpecFramework = runner.NormalizeSpecFramework(specFw.String)
	p.ProjectType = NormalizeProjectType(projType.String)
	return &p, nil
}

func (d *DB) CreateProject(req models.CreateProjectRequest) (*models.Project, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("nom du projet obligatoire")
	}

	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = strings.ToLower(name)
		slug = strings.ReplaceAll(slug, " ", "-")
		slug = strings.ReplaceAll(slug, "_", "-")
		slug = strings.ReplaceAll(slug, "'", "-")
	}

	id := uuid.New().String()
	icon := req.Icon
	if icon == "" {
		icon = "Folder"
	}
	color := req.Color
	if color == "" {
		color = "indigo"
	}
	issueTracker := req.IssueTracker
	if issueTracker == "" {
		issueTracker = "linear"
	}

	gitRemote := strings.TrimSpace(req.GitRemoteUrl)
	githubRepo := strings.TrimSpace(req.GithubRepo)
	if githubRepo == "" && gitRemote != "" {
		githubRepo = ParseGitRepoFromURL(gitRemote)
	}

	// A Jira project key is always uppercase (PE, ENG, OPS…). Fall back to the
	// project slug so an existing Jira-tracked project keeps working.
	jiraProject := strings.ToUpper(strings.TrimSpace(req.JiraProject))
	if jiraProject == "" && issueTracker == "jira" {
		jiraProject = strings.ToUpper(strings.ReplaceAll(slug, "-", ""))
	}

	aiProvider := strings.TrimSpace(req.AIProvider)
	aiCmd := strings.TrimSpace(req.AICommandTemplate)
	specFramework := runner.NormalizeSpecFramework(req.SpecFramework)
	projectType := NormalizeProjectType(req.ProjectType)

	now := time.Now()
	isDefInt := 0
	if req.IsDefault {
		isDefInt = 1
		_, _ = d.conn.Exec("UPDATE projects SET is_default = 0")
	}

	stageMapping := req.StageMapping
	if stageMapping == nil || len(stageMapping) == 0 {
		stageMapping = defaultStageMapping()
	}
	stageMappingBytes, _ := json.Marshal(stageMapping)

	skillOverrides := req.SkillOverrides
	if skillOverrides == nil {
		skillOverrides = map[string]string{}
	}
	skillOverridesBytes, _ := json.Marshal(skillOverrides)

	repoPathsBytes, _ := json.Marshal(normalizeRepoPaths(req.RepoPaths))

	_, err := d.conn.Exec(`
		INSERT INTO projects (id, name, slug, description, icon, color, repo_path, repo_paths, git_remote_url, linear_team, github_repo, jira_project, issue_tracker, tracker_url, project_type, is_default, stage_mapping, skill_overrides, ai_provider, ai_command_template, spec_framework, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, name, slug, req.Description, icon, color, req.RepoPath, string(repoPathsBytes), gitRemote, req.LinearTeam, githubRepo, jiraProject, issueTracker, req.TrackerUrl, projectType, isDefInt, string(stageMappingBytes), string(skillOverridesBytes), aiProvider, aiCmd, specFramework, now, now)
	if err != nil {
		return nil, err
	}

	return d.getProjectByIDUnsafe(id)
}

func (d *DB) UpdateProject(id string, req models.UpdateProjectRequest) (*models.Project, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	p, err := d.getProjectByIDUnsafe(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		p.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Icon != nil && *req.Icon != "" {
		p.Icon = *req.Icon
	}
	if req.Color != nil && *req.Color != "" {
		p.Color = *req.Color
	}
	if req.RepoPath != nil {
		p.RepoPath = *req.RepoPath
	}
	if req.GitRemoteUrl != nil {
		p.GitRemoteUrl = strings.TrimSpace(*req.GitRemoteUrl)
		if p.GithubRepo == "" && p.GitRemoteUrl != "" {
			p.GithubRepo = ParseGitRepoFromURL(p.GitRemoteUrl)
		}
	}
	if req.LinearTeam != nil {
		p.LinearTeam = *req.LinearTeam
	}
	if req.GithubRepo != nil {
		p.GithubRepo = *req.GithubRepo
	}
	if req.JiraProject != nil {
		p.JiraProject = strings.ToUpper(strings.TrimSpace(*req.JiraProject))
	}
	if req.IssueTracker != nil {
		p.IssueTracker = *req.IssueTracker
	}
	if req.TrackerUrl != nil {
		p.TrackerUrl = *req.TrackerUrl
	}
	if req.StageMapping != nil {
		p.StageMapping = *req.StageMapping
	}
	if req.SkillOverrides != nil {
		p.SkillOverrides = *req.SkillOverrides
	}
	if req.RepoPaths != nil {
		p.RepoPaths = normalizeRepoPaths(*req.RepoPaths)
	}
	if req.AIProvider != nil {
		p.AIProvider = *req.AIProvider
	}
	if req.AICommandTemplate != nil {
		p.AICommandTemplate = *req.AICommandTemplate
	}
	if req.SpecFramework != nil {
		p.SpecFramework = runner.NormalizeSpecFramework(*req.SpecFramework)
	}
	if req.ProjectType != nil {
		p.ProjectType = NormalizeProjectType(*req.ProjectType)
	}
	if req.IsDefault != nil {
		p.IsDefault = *req.IsDefault
		if p.IsDefault {
			_, _ = d.conn.Exec("UPDATE projects SET is_default = 0 WHERE id != ?", p.ID)
		}
	}
	p.UpdatedAt = time.Now()
	isDefInt := 0
	if p.IsDefault {
		isDefInt = 1
	}

	if p.StageMapping == nil || len(p.StageMapping) == 0 {
		p.StageMapping = defaultStageMapping()
	}
	stageMappingBytes, _ := json.Marshal(p.StageMapping)

	if p.SkillOverrides == nil {
		p.SkillOverrides = map[string]string{}
	}
	skillOverridesBytes, _ := json.Marshal(p.SkillOverrides)
	repoPathsBytes, _ := json.Marshal(normalizeRepoPaths(p.RepoPaths))

	_, err = d.conn.Exec(`
		UPDATE projects
		SET name = ?, slug = ?, description = ?, icon = ?, color = ?, repo_path = ?, repo_paths = ?, git_remote_url = ?, linear_team = ?, github_repo = ?, jira_project = ?, issue_tracker = ?, tracker_url = ?, project_type = ?, is_default = ?, stage_mapping = ?, skill_overrides = ?, ai_provider = ?, ai_command_template = ?, spec_framework = ?, updated_at = ?
		WHERE id = ?
	`, p.Name, p.Slug, p.Description, p.Icon, p.Color, p.RepoPath, string(repoPathsBytes), p.GitRemoteUrl, p.LinearTeam, p.GithubRepo, p.JiraProject, p.IssueTracker, p.TrackerUrl, NormalizeProjectType(p.ProjectType), isDefInt, string(stageMappingBytes), string(skillOverridesBytes), p.AIProvider, p.AICommandTemplate, p.SpecFramework, p.UpdatedAt, p.ID)
	if err != nil {
		return nil, err
	}

	return d.getProjectByIDUnsafe(p.ID)
}

func (d *DB) DeleteProject(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	p, err := d.getProjectByIDUnsafe(id)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("projet non trouvé")
	}
	if p.IsDefault {
		return fmt.Errorf("impossible de supprimer le projet par défaut")
	}

	// Reassign tasks to default project
	var defaultProjID string
	_ = d.conn.QueryRow("SELECT id FROM projects WHERE is_default = 1 LIMIT 1").Scan(&defaultProjID)
	if defaultProjID == "" {
		defaultProjID = "default"
	}
	_, _ = d.conn.Exec("UPDATE tasks SET project_id = ? WHERE project_id = ?", defaultProjID, p.ID)
	_, err = d.conn.Exec("DELETE FROM projects WHERE id = ?", p.ID)
	return err
}

// -------------------------------------------------------------
// PROJECT SKILLS MANAGEMENT & PROVISIONING
// -------------------------------------------------------------

type ProjectSkillTemplate struct {
	ID          string
	Name        string
	DirName     string
	Description string
	Content     string
}

var DefaultProjectSkills = []ProjectSkillTemplate{
	{
		ID:          "clarify",
		Name:        "Clarify Issue",
		DirName:     "clarify-issue",
		Description: "Analyse les ambiguïtés techniques et produit 3 à 5 questions de cadrage.",
		Content: `---
name: clarify-issue
description: Analyse une story ou un ticket, identifie les zones d'ombre et formule les questions précises de cadrage.
---
# Skill : Clarify Issue

## Objectif
Analyser l'issue ou la story spécifiée, identifier les ambiguïtés techniques et fonctionnelles, et produire une synthèse claire avec des questions ciblées de clarification.

## Instructions
1. Lire la description du ticket et inspecter le code source existant dans le workspace.
2. Détecter les zones d'ombre : architecture, stockage, sécurité, dépendances et cas limites.
3. Formuler 3 à 5 questions concises et structurées pour aligner le produit et l'ingénierie.
4. Structurer la sortie avec :
   - Analyse des ambiguïtés
   - Dépendances critiques
   - Questions d'alignement numérotées
`,
	},
	{
		ID:          "specify",
		Name:        "Specify Issue (Speckit)",
		DirName:     "specify-issue",
		Description: "Rédige la spec technique Speckit, définit les critères d'acceptation Gherkin et prépare la branche Git.",
		Content: `---
name: specify-issue
description: Rédige la spécification technique Speckit, définit les critères d'acceptation Gherkin et prépare la branche Git.
---
# Skill : Specify Issue (Speckit)

## Objectif
Générer une spécification technique exhaustive et actionnable (Speckit) prête pour l'implémentation.

## Instructions
1. Vérifier les réponses de clarification et le contexte du projet.
2. Créer ou basculer sur la branche Git de travail au format <KEY>-<titre-slug>.
3. Rédiger la spec technique complète incluant :
   - Contexte et Objectifs
   - User Stories et Scénarios d'Acceptation (Given / When / Then)
   - Architecture & Diagrammes de flux (Mermaid)
   - Contrats d'API et Schémas de Données
   - Plan de tests et critères de validation
`,
	},
	{
		ID:          "implement",
		Name:        "Implement Code",
		DirName:     "code-issue",
		Description: "Implémente le code conformément à la spécification technique, exécute les tests et valide le build.",
		Content: `---
name: code-issue
description: Implémente le code conformément à la spécification technique, exécute les tests et valide le linting.
---
# Skill : Code Issue (Implement)

## Objectif
Implémenter les changements de code de manière rigoureuse et testée selon la spécification technique.

## Instructions
1. Inspecter la spécification technique et la branche courante.
2. Écrire ou mettre à jour les tests unitaires et d'intégration nécessaires.
3. Implémenter les modifications de code nécessaires dans le respect des conventions du projet.
4. Exécuter la suite de tests et les vérifications de linting / build.
5. Résumer les fichiers modifiés et les résultats des validations.
`,
	},
	{
		ID:          "create_pr",
		Name:        "Review & Pull Request / Merge",
		DirName:     "create-pr",
		Description: "Revue finale, commit conventionnel et création de la Pull Request (ou fusion locale si aucun remote n'est configuré).",
		Content: `---
name: create-pr
description: Effectue la revue finale, génère des commits conventionnels et publie la Pull Request sur GitHub (ou fusionne localement si aucun remote n'est configuré).
---
# Skill : Review, Pull Request & Local Merge

## Objectif
Revoir les changements, valider la qualité du code, commiter avec des messages conventionnels et publier la Pull Request (ou fusionner localement si aucun remote n'est configuré).

## Instructions
1. Effectuer un 'git status' et 'git diff' pour inspecter tous les changements sur la branche.
2. S'assurer que les modifications sont commitées (ex: feat(scope): ... ou fix(scope): ...).
3. Vérifier les remotes via 'git remote' :
   - **Si remote présent** : Pousser la branche ('git push -u origin <branch>') et créer la Pull Request ('gh pr create' ou 'glab mr create').
   - **Si aucun remote configuré** : Basculer sur la branche principale ('git checkout main') et fusionner la branche ('git merge --no-ff <branch>').
4. Produire un compte-rendu clair des actions réalisées.
`,
	},
	{
		ID:          "pick",
		Name:        "Auto-Pilot Orchestrator",
		DirName:     "pick-issue",
		Description: "Sélectionne la tâche prioritaire, exécute les compétences requises jusqu'à la PR ou la clôture.",
		Content: `---
name: pick-issue
description: Sélectionne la tâche prioritaire, avance son statut à travers le cycle de développement complet et crée la PR.
---
# Skill : Auto-Pilot Orchestrator (Pick Issue)

## Objectif
Prendre en charge une tâche depuis la file d'attente, exécuter de manière autonome le cycle de clarification, spécification, codage et ouverture de PR.

## Instructions
1. Identifier la tâche cible spécifiée ou prendre la plus prioritaire.
2. Basculer ou créer la branche Git dédiée dans le worktree.
3. Exécuter séquentiellement :
   - /clarify-issue (si non cadrée)
   - /specify-issue (si non spécifiée)
   - /code-issue (implémentation et tests)
   - /create-pr (validation finale et soumission)
4. Mettre à jour le statut et consigner les artefacts d'exécution.
`,
	},
}

func getGitWorktreePaths(repoPath string) []string {
	var paths []string
	cleanRepo, err := filepath.Abs(filepath.Clean(repoPath))
	if err != nil {
		cleanRepo = filepath.Clean(repoPath)
	}
	paths = append(paths, cleanRepo)

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = cleanRepo
	out, err := cmd.Output()
	if err != nil {
		return paths
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			wtPath := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			if abs, err := filepath.Abs(filepath.Clean(wtPath)); err == nil {
				wtPath = abs
			}
			if wtPath != "" {
				already := false
				for _, p := range paths {
					if p == wtPath {
						already = true
						break
					}
				}
				if !already {
					paths = append(paths, wtPath)
				}
			}
		}
	}
	return paths
}

func (d *DB) GetProjectSkillsStatus(projectIDOrPath string) (*models.ProjectSkillsStatus, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	projectID := projectIDOrPath
	projectName := projectIDOrPath
	specFramework := "speckit"

	if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil {
		projectID = proj.ID
		projectName = proj.Name
		if proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
		if proj.SpecFramework != "" {
			specFramework = proj.SpecFramework
		}
	}
	d.mu.RUnlock()

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		repoPath = "."
	}

	worktreePaths := getGitWorktreePaths(repoPath)

	res := &models.ProjectSkillsStatus{
		ProjectID:      projectID,
		ProjectName:    projectName,
		RepoPath:       repoPath,
		PathExists:     false,
		IsGitRepo:      false,
		InstalledAll:   true,
		SpecFramework:  specFramework,
		WorktreesCount: len(worktreePaths),
		WorktreePaths:  worktreePaths,
		Skills:         []models.InstalledSkillInfo{},
	}

	fi, err := os.Stat(repoPath)
	if err != nil || !fi.IsDir() {
		res.InstalledAll = false
		return res, nil
	}
	res.PathExists = true

	// Check git repo
	gitDir := filepath.Join(repoPath, ".git")
	if gfi, gErr := os.Stat(gitDir); gErr == nil && gfi.IsDir() {
		res.IsGitRepo = true
		gitCheck := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		gitCheck.Dir = repoPath
		if err := gitCheck.Run(); err == nil {
			branchCmd := exec.Command("git", "branch", "--show-current")
			branchCmd.Dir = repoPath
			if out, err := branchCmd.Output(); err == nil {
				b := strings.TrimSpace(string(out))
				if b != "" && b != "HEAD" {
					res.GitBranch = b
				}
			}
		}
	} else {
		res.InstalledAll = false
	}

	for _, s := range DefaultProjectSkills {
		p0 := filepath.Join(repoPath, ".agents", "skills", s.DirName, "SKILL.md")
		p1 := filepath.Join(repoPath, ".gemini", "skills", s.DirName, "SKILL.md")
		p2 := filepath.Join(repoPath, ".agy", "skills", s.DirName, "SKILL.md")
		p3 := filepath.Join(repoPath, ".skills", s.DirName, "SKILL.md")

		installed := false
		targetPath := p0
		if _, err := os.Stat(p0); err == nil {
			installed = true
			targetPath = p0
		} else if _, err := os.Stat(p1); err == nil {
			installed = true
			targetPath = p1
		} else if _, err := os.Stat(p2); err == nil {
			installed = true
			targetPath = p2
		} else if _, err := os.Stat(p3); err == nil {
			installed = true
			targetPath = p3
		} else {
			res.InstalledAll = false
		}

		skillName := s.Name
		if s.ID == "specify" {
			if specFramework == "openspec" {
				skillName = "Specify Issue (OpenSpec SDD)"
			} else {
				skillName = "Specify Issue (Spec Kit SDD)"
			}
		}

		res.Skills = append(res.Skills, models.InstalledSkillInfo{
			ID:          s.ID,
			Name:        skillName,
			Installed:   installed,
			Path:        targetPath,
			Description: s.Description,
		})
	}

	return res, nil
}

func (d *DB) InstallProjectSkills(projectIDOrPath string, overrides ...string) (*models.ProjectSkillsStatus, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	projectID := projectIDOrPath
	projectName := projectIDOrPath
	linearTeam := ""
	githubRepo := ""
	issueTracker := "local"
	specFramework := "speckit"
	aiProvider := "agy"
	aiCommandTemplate := ""

	if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil {
		projectID = proj.ID
		projectName = proj.Name
		if proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
		if proj.LinearTeam != "" {
			linearTeam = proj.LinearTeam
		}
		if proj.GithubRepo != "" {
			githubRepo = proj.GithubRepo
		}
		if proj.IssueTracker != "" {
			issueTracker = proj.IssueTracker
		}
		if proj.SpecFramework != "" {
			specFramework = proj.SpecFramework
		}
		if proj.AIProvider != "" {
			aiProvider = proj.AIProvider
		}
		if proj.AICommandTemplate != "" {
			aiCommandTemplate = proj.AICommandTemplate
		}
	}
	d.mu.RUnlock()

	if len(overrides) > 0 && strings.TrimSpace(overrides[0]) != "" {
		specFramework = overrides[0]
	}
	specFramework = runner.NormalizeSpecFramework(specFramework)
	if len(overrides) > 1 && strings.TrimSpace(overrides[1]) != "" {
		aiProvider = strings.TrimSpace(overrides[1])
	}
	if len(overrides) > 2 && strings.TrimSpace(overrides[2]) != "" {
		aiCommandTemplate = strings.TrimSpace(overrides[2])
	}

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("le chemin du dossier de travail (CWD) est obligatoire")
	}

	// Create CWD if doesn't exist
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("impossible de créer le répertoire %s: %w", repoPath, err)
	}

	// Get all worktree paths to scaffold skills into every worktree directory
	targetPaths := getGitWorktreePaths(repoPath)

	// Build skills with framework-specific content for specify-issue
	skillsToInstall := make([]ProjectSkillTemplate, len(DefaultProjectSkills))
	copy(skillsToInstall, DefaultProjectSkills)

	for i, s := range skillsToInstall {
		if s.ID == "specify" {
			if specFramework == "openspec" {
				skillsToInstall[i].Name = "Specify Issue (OpenSpec SDD)"
				skillsToInstall[i].Content = `---
name: specify-issue
description: Rédige la spécification selon la convention OpenSpec (proposition de changement sous openspec/changes/, deltas de specs et checklist de tâches).
---
# Skill : Specify Issue (OpenSpec SDD)

## Objectif
Produire une proposition de changement OpenSpec revue avant écriture de code, dans le
répertoire ` + "`openspec/`" + ` du projet.

## Pré-requis
Le projet doit être initialisé avec OpenSpec (répertoire ` + "`openspec/`" + ` présent).
Sinon, lancer l'installation OpenSpec depuis la configuration du projet Taskacao,
ou exécuter ` + "`openspec init`" + ` à la racine du dépôt.

## Instructions
1. Lire ` + "`openspec/project.md`" + ` et les specs existantes sous ` + "`openspec/specs/`" + `
   pour connaître les capacités déjà décrites.
2. Créer ou basculer sur la branche Git de travail au format <KEY>-<titre-slug>.
3. Créer le dossier de changement ` + "`openspec/changes/<KEY>-<titre-slug>/`" + ` et y écrire :
   - ` + "`proposal.md`" + ` : le pourquoi (problème, valeur, périmètre inclus et exclu).
   - ` + "`design.md`" + ` : les décisions techniques et les alternatives écartées.
   - ` + "`tasks.md`" + ` : la checklist ordonnée et vérifiable de mise en œuvre.
   - ` + "`specs/<capability>/spec.md`" + ` : les deltas de comportement, sous forme
     d'exigences ` + "`## ADDED`" + ` / ` + "`## MODIFIED`" + ` / ` + "`## REMOVED`" + ` avec des
     scénarios Given / When / Then.
4. Valider la proposition avec ` + "`openspec validate <change-id> --strict`" + ` et corriger
   les erreurs signalées.
5. Publier le résumé de la proposition en commentaire du ticket, puis attendre la revue
   avant d'implémenter.
`
			} else {
				skillsToInstall[i].Name = "Specify Issue (Spec Kit SDD)"
				skillsToInstall[i].Content = `---
name: specify-issue
description: Rédige la spécification technique selon GitHub Spec Kit (spec.md, plan.md, tasks.md sous specs/), avec user stories, architecture et critères d'acceptation Gherkin.
---
# Skill : Specify Issue (GitHub Spec Kit SDD)

## Objectif
Produire la spécification exécutable d'un ticket selon GitHub Spec Kit, dans le
répertoire ` + "`specs/`" + ` du projet.

## Pré-requis
Le projet doit être initialisé avec Spec Kit (répertoire ` + "`.specify/`" + ` présent).
Sinon, lancer l'installation Spec Kit depuis la configuration du projet Taskacao,
ou exécuter ` + "`specify init --here`" + ` à la racine du dépôt.

## Instructions
1. Lire ` + "`.specify/memory/constitution.md`" + ` pour respecter les principes du projet.
2. Créer ou basculer sur la branche Git de travail au format <KEY>-<titre-slug>.
3. Écrire ` + "`specs/<KEY>-<titre-slug>/spec.md`" + ` (le quoi et le pourquoi, sans
   choix d'implémentation) :
   - Contexte, User Stories priorisées et périmètre exclu
   - Exigences fonctionnelles numérotées et critères d'acceptation Given / When / Then
   - Points à clarifier marqués explicitement plutôt que devinés
4. Écrire ` + "`plan.md`" + ` (le comment) : pile technique, architecture, contrats de données
   et diagrammes de flux Mermaid.
5. Écrire ` + "`tasks.md`" + ` : la checklist ordonnée et vérifiable de mise en œuvre.
6. Si les commandes Spec Kit sont disponibles dans l'agent, utiliser
   ` + "`/speckit.specify`" + `, ` + "`/speckit.plan`" + ` puis ` + "`/speckit.tasks`" + `
   au lieu de rédiger les fichiers à la main.
7. Publier le résumé de la spécification en commentaire du ticket.
`
			}
		}
	}

	// Install skills into each target path (root repo and all worktrees)
	for _, targetDir := range targetPaths {
		for _, s := range skillsToInstall {
			dirs := []string{
				filepath.Join(targetDir, ".agents", "skills", s.DirName),
				filepath.Join(targetDir, ".gemini", "skills", s.DirName),
				filepath.Join(targetDir, ".agy", "skills", s.DirName),
				filepath.Join(targetDir, ".skills", s.DirName),
			}

			for _, dir := range dirs {
				if err := os.MkdirAll(dir, 0755); err != nil {
					continue
				}
				filePath := filepath.Join(dir, "SKILL.md")
				_ = os.WriteFile(filePath, []byte(s.Content), 0644)
			}
		}

		// Create .taskacao/config.json in each worktree/root
		taskacaoDir := filepath.Join(targetDir, ".taskacao")
		_ = os.MkdirAll(taskacaoDir, 0755)
		configFile := filepath.Join(taskacaoDir, "config.json")
		cfgData := map[string]interface{}{
			"projectId":         projectID,
			"projectName":       projectName,
			"linearTeam":        linearTeam,
			"githubRepo":        githubRepo,
			"issueTracker":      issueTracker,
			"specFramework":     specFramework,
			"aiProvider":        aiProvider,
			"aiCommandTemplate": aiCommandTemplate,
			"skills": []string{
				"clarify-issue",
				"specify-issue",
				"code-issue",
				"create-pr",
				"pick-issue",
			},
			"updatedAt": time.Now().Format(time.RFC3339),
		}
		if bytes, err := json.MarshalIndent(cfgData, "", "  "); err == nil {
			_ = os.WriteFile(configFile, bytes, 0644)
		}
	}

	return d.GetProjectSkillsStatus(projectID)
}

func (d *DB) InitProjectGit(projectIDOrPath string) (*models.ProjectGitInitResult, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil {
		if proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	d.mu.RUnlock()

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("le chemin du dossier de travail (CWD) est obligatoire")
	}

	// Create directory if not exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("impossible de créer le répertoire %s: %w", repoPath, err)
	}

	// Check if already git repo
	gitDir := filepath.Join(repoPath, ".git")
	isAlreadyGit := false
	if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
		isAlreadyGit = true
	}

	// Run git init -b main
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = repoPath
	if _, err := cmd.CombinedOutput(); err != nil {
		// Fallback for older git without -b flag: git init
		cmd2 := exec.Command("git", "init")
		cmd2.Dir = repoPath
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return nil, fmt.Errorf("erreur git init: %s (%w)", string(out2), err2)
		}
	}

	// Create a standard .gitignore if none exists
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		defaultGitignore := `# Node / Dependencies
node_modules/
dist/
build/
.env
.env.local

# OS
.DS_Store
Thumbs.db

# Logs & Temp
*.log
tmp/
`
		_ = os.WriteFile(gitignorePath, []byte(defaultGitignore), 0644)
	}

	// Detect current branch
	branchName := "main"
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoPath
	if out, err := branchCmd.Output(); err == nil {
		b := strings.TrimSpace(string(out))
		if b != "" && b != "HEAD" {
			branchName = b
		}
	}

	msg := "Dépôt Git initialisé avec succès (branche main)"
	if isAlreadyGit {
		msg = "Dépôt Git existant validé et synchronisé"
	}

	return &models.ProjectGitInitResult{
		RepoPath:    repoPath,
		IsGitRepo:   true,
		Branch:      branchName,
		Message:     msg,
		Initialized: true,
	}, nil
}

func (d *DB) DetectTrackerStatuses(projectID, tracker, linearTeam, githubRepo string) ([]models.DetectedStatus, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var results []models.DetectedStatus
	seen := make(map[string]bool)
	jiraProject := ""
	jiraRepoPath := ""

	addStatus := func(name, sType, color, source string) {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return
		}
		lower := strings.ToLower(trimmed)
		if seen[lower] {
			return
		}
		seen[lower] = true
		results = append(results, models.DetectedStatus{
			ID:     trimmed,
			Name:   trimmed,
			Type:   sType,
			Color:  color,
			Source: source,
		})
	}

	// 1. If projectID provided, load project info
	if projectID != "" && projectID != "detect-statuses" {
		if proj, _ := d.getProjectByIDUnsafe(projectID); proj != nil {
			if tracker == "" {
				tracker = proj.IssueTracker
			}
			if linearTeam == "" {
				linearTeam = proj.LinearTeam
			}
			if githubRepo == "" {
				githubRepo = proj.GithubRepo
			}
			if jiraProject == "" {
				jiraProject = jiraProjectKeyFor(proj)
			}
			if jiraRepoPath == "" {
				jiraRepoPath = proj.RepoPath
			}
		}
	}

	// 2. Try Linear API if tracker is linear (or team provided)
	if tracker == "linear" || linearTeam != "" {
		linearPath, err := exec.LookPath("linear")
		if err != nil {
			linearPath = "/opt/homebrew/bin/linear"
		}
		if _, err := os.Stat(linearPath); err == nil {
			query := `query { teams { nodes { key name states { nodes { id name color type position } } } } }`
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			cmd := exec.CommandContext(ctx, linearPath, "api", query)
			out, err := cmd.Output()
			cancel()
			if err == nil && len(out) > 0 {
				var gqlResp struct {
					Data struct {
						Teams struct {
							Nodes []struct {
								Key    string `json:"key"`
								Name   string `json:"name"`
								States struct {
									Nodes []struct {
										ID       string  `json:"id"`
										Name     string  `json:"name"`
										Color    string  `json:"color"`
										Type     string  `json:"type"`
										Position float64 `json:"position"`
									} `json:"nodes"`
								} `json:"states"`
							} `json:"nodes"`
						} `json:"teams"`
					} `json:"data"`
				}
				if jErr := json.Unmarshal(out, &gqlResp); jErr == nil {
					for _, t := range gqlResp.Data.Teams.Nodes {
						if linearTeam == "" || strings.EqualFold(t.Key, linearTeam) {
							for _, st := range t.States.Nodes {
								addStatus(st.Name, st.Type, st.Color, "linear")
							}
						}
					}
				}
			}
		}
	}

	// 3. Scan existing tasks in SQLite database for project / tracker
	query := "SELECT DISTINCT status FROM tasks WHERE 1=1"
	var args []interface{}
	if projectID != "" && projectID != "detect-statuses" {
		query += " AND project_id = ?"
		args = append(args, projectID)
	}
	if rows, err := d.conn.Query(query, args...); err == nil {
		for rows.Next() {
			var st string
			if sErr := rows.Scan(&st); sErr == nil && st != "" {
				addStatus(st, "db", "", "db")
			}
		}
		rows.Close()
	}

	// 4. If tracker is github, add github states
	if tracker == "github" {
		addStatus("open", "unstarted", "#3fb950", "github")
		addStatus("closed", "completed", "#8250df", "github")
	}

	// 4b. If tracker is jira, read the real workflow statuses from acli when it
	// is reachable, then fall back to the default software-project workflow.
	if tracker == "jira" {
		if acliPath, err := runner.FindCliTool("acli"); err == nil && acliPath != "" && jiraProject != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			cmd := exec.CommandContext(ctx, acliPath, "jira", "workitem", "search",
				"--jql", fmt.Sprintf("project = %s", jiraProject), "--fields", "status", "--limit", "100", "--output", "json")
			if jiraRepoPath != "" {
				cmd.Dir = jiraRepoPath
			}
			out, cErr := cmd.Output()
			cancel()
			if cErr == nil && len(out) > 0 {
				var items []struct {
					Fields struct {
						Status struct {
							Name           string `json:"name"`
							StatusCategory struct {
								Key string `json:"key"`
							} `json:"statusCategory"`
						} `json:"status"`
					} `json:"fields"`
				}
				if jErr := json.Unmarshal(out, &items); jErr == nil {
					for _, it := range items {
						sType := "custom"
						switch it.Fields.Status.StatusCategory.Key {
						case "new":
							sType = "unstarted"
						case "indeterminate":
							sType = "started"
						case "done":
							sType = "completed"
						}
						addStatus(it.Fields.Status.Name, sType, "", "jira")
					}
				}
			}
		}
		addStatus("To Do", "unstarted", "#42526e", "jira")
		addStatus("In Progress", "started", "#0052cc", "jira")
		addStatus("In Review", "started", "#5243aa", "jira")
		addStatus("Done", "completed", "#00875a", "jira")
	}

	// 5. Standard fallback presets if list is short or empty
	for _, def := range []struct {
		name  string
		sType string
		color string
	}{
		{"Backlog", "backlog", "#bec2c8"},
		{"Todo", "unstarted", "#e2e2e2"},
		{"In Progress", "started", "#f2c94c"},
		{"In Review", "started", "#5e6ad2"},
		{"Done", "completed", "#27ae60"},
		{"Canceled", "canceled", "#eb5757"},
		{"to_clarify", "backlog", "#06b6d4"},
		{"to_specify", "unstarted", "#f59e0b"},
		{"to_implement", "started", "#3b82f6"},
		{"to_test", "started", "#6366f1"},
		{"to_close", "completed", "#10b981"},
	} {
		addStatus(def.name, def.sType, def.color, "preset")
	}

	return results, nil
}


