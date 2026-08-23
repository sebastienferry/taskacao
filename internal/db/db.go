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

	conn.SetMaxOpenConns(1) // SQLite single-writer safety

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
			linear_team TEXT NOT NULL DEFAULT '',
			github_repo TEXT NOT NULL DEFAULT '',
			issue_tracker TEXT NOT NULL DEFAULT 'local',
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
		`CREATE TABLE IF NOT EXISTS task_messages (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			activity_id TEXT NOT NULL DEFAULT '',
			skill_id TEXT NOT NULL DEFAULT '',
			steps TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(status, position);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_task ON task_activities(task_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_status ON task_activities(status);`,
		`CREATE INDEX IF NOT EXISTS idx_activities_created ON task_activities(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_task_messages_task ON task_messages(task_id, created_at ASC);`,
	}

	for _, query := range queries {
		if _, err := d.conn.Exec(query); err != nil {
			return err
		}
	}

	// Migrations for existing database instances
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN stage_mapping TEXT NOT NULL DEFAULT '{}';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN git_remote_url TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN tracker_url TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE projects ADD COLUMN skill_overrides TEXT NOT NULL DEFAULT '{}';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN project_id TEXT NOT NULL DEFAULT 'default';")
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN branch_name TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN pr_url TEXT;")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN source TEXT NOT NULL DEFAULT 'local';")
	_, _ = d.conn.Exec("ALTER TABLE tasks ADD COLUMN external_url TEXT;")

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
	_, _ = d.conn.Exec(`CREATE TABLE IF NOT EXISTS task_messages (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		activity_id TEXT NOT NULL DEFAULT '',
		skill_id TEXT NOT NULL DEFAULT '',
		steps TEXT NOT NULL DEFAULT '[]',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	);`)
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_task_messages_task ON task_messages(task_id, created_at ASC);")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_clarify TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_specify TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_implement TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_create_pr TEXT NOT NULL DEFAULT '';")
	_, _ = d.conn.Exec("ALTER TABLE settings ADD COLUMN prompt_pick TEXT NOT NULL DEFAULT '';")

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
			VALUES ('default', 'Workspace', 'default', 'Espace de travail principal', 'Folder', 'emerald', '.', '', '', 'local', 1);
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
			VALUES (1, 'dark', 'indigo', 'fr', 'standard', 'board', 'Developer', 'dev@example.com', '', 'agy', 'agy -p "{prompt}"', '.', 'local', '', '')
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
		VALUES (1, 'dark', 'indigo', 'fr', 'standard', 'board', 'Developer', 'dev@example.com', '', 'agy', 'agy -p "{prompt}"', '.', 'local', '', '')
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

	// 1. Seed the 6 local tasks for project 'tasks' (Taskacao)
	now := time.Now()
	localTasks := []struct {
		ID, Key, Title, Desc, Status, Priority, Branch, Labels string
		Pos                                                    int
	}{
		{"task-1", "TASK-1", "Create a github repo for Taskacao and configure repository metadata", "Initialisation et configuration du dépôt GitHub public pour Taskacao avec métadonnées et intégration continue.", "finished", "high", "TASK-1-create-a-github-repo-for-taska", `["github", "devops", "repo"]`, 1},
		{"task-2", "TASK-2", "Clarifier le titre de la side bar", "Améliorer la clarté de la barre latérale pour la gestion multi-projets et le statut des compétences.", "finished", "medium", "TASK-2-clarifier-le-titre-de-la-side-", `["ui", "sidebar", "ux"]`, 2},
		{"task-3", "TASK-3", "Task board flickering", "Résoudre le clignotement / scintillement intempestif des colonnes lors du rafraîchissement des activités et du drag & drop.", "finished", "high", "TASK-3-task-board-flickering", `["bug", "ui", "kanban"]`, 3},
		{"task-4", "TASK-4", "Add current branch in Ux", "Afficher la branche Git courante dans la barre d'état et le badge de tâche pour faciliter le repérage de travail.", "finished", "medium", "TASK-4-add-current-branch-in-ux", `["git", "ux", "statusbar"]`, 4},
		{"task-5", "TASK-5", "Ne pas pouvoir changer le tracker lors de la creation d'une tache", "Verrouiller le tracker cible (Linear, GitHub, Local) lors de la création pour respecter la configuration du projet actif.", "finished", "medium", "TASK-2-ne-pas-pouvoir-changer-le-trac", `["tracker", "quick-add", "ui"]`, 5},
		{"task-6", "TASK-6", "Wrong filter switching to/from activities", "Corriger les filtres de statut et de projet lors du passage entre la vue Activités et le tableau Kanban.", "finished", "high", "TASK-4-wrong-filter-switching-to/from", `["filters", "navigation", "bug"]`, 6},
	}

	for _, lt := range localTasks {
		_, _ = d.conn.Exec(`
			INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, position, branch_name, source, created_at, updated_at)
			VALUES (?, 'tasks', ?, ?, ?, ?, ?, ?, 'Developer', ?, ?, 'local', ?, ?)
		`, lt.ID, lt.Key, lt.Title, lt.Desc, lt.Status, lt.Priority, lt.Labels, lt.Pos, lt.Branch, now.Format(time.RFC3339), now)
	}

	// 2. Live sync from Linear for fretzee-studio
	linTasks, err := d.runner.SyncFromLinear("FRE")
	if err == nil && len(linTasks) > 0 {
		for _, t := range linTasks {
			labelsJSON, _ := json.Marshal(t.Labels)
			_, _ = d.conn.Exec(`
				INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at)
				VALUES (?, 'fretzee-studio', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, t.ID, t.Key, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.AssigneeAvatar, t.Position, t.DueDate, t.BranchName, t.PrURL, t.Source, t.ExternalURL, t.CreatedAt, now)
		}
	}

	// 3. Live sync from GitHub for fretzee-studio
	ghTasks, err := d.runner.SyncFromGithub("sebastienferry/fretzee-studio", "/Users/sferry/Sources/fretzee-studio")
	if err == nil && len(ghTasks) > 0 {
		for _, t := range ghTasks {
			labelsJSON, _ := json.Marshal(t.Labels)
			_, _ = d.conn.Exec(`
				INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at)
				VALUES (?, 'fretzee-studio', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

		src := t.Source
		if src == "" {
			if strings.HasPrefix(t.Key, "FRE-") {
				src = "linear"
			} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
				src = "github"
			} else {
				src = "local"
			}
		}

		projID := t.ProjectID
		if projID == "" {
			projID = "fretzee-studio"
		}

		if err == sql.ErrNoRows {
			// Insert new task
			newID := t.ID
			if newID == "" {
				newID = uuid.New().String()
			}
			_, _ = d.conn.Exec(`
				INSERT INTO tasks (id, project_id, key, title, description, status, priority, labels, assignee, position, due_date, source, external_url, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newID, projID, t.Key, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, t.Position, t.DueDate, src, t.ExternalURL, t.CreatedAt, now)
		} else if err == nil {
			// Update existing task title/desc/status/labels/source
			_, _ = d.conn.Exec(`
				UPDATE tasks
				SET title = ?, description = ?, status = ?, priority = ?, labels = ?, assignee = ?, source = ?, external_url = ?, updated_at = ?
				WHERE id = ?
			`, t.Title, t.Description, string(t.Status), string(t.Priority), string(labelsJSON), t.Assignee, src, t.ExternalURL, now, existingID)
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

	sqlQuery := "SELECT id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at FROM tasks"
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
		if source.Valid && source.String != "" {
			t.Source = source.String
		} else if strings.HasPrefix(t.Key, "FRE-") {
			t.Source = "linear"
		} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
			t.Source = "github"
		} else {
			t.Source = "local"
		}

		if extURL.Valid && extURL.String != "" {
			t.ExternalURL = &extURL.String
		} else if t.Source == "linear" {
			url := fmt.Sprintf("https://linear.app/fretzee/issue/%s", t.Key)
			t.ExternalURL = &url
		} else if t.Source == "github" {
			cleanNum := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(t.Key, "GH-#"), "gh-"), "#")
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
		SELECT id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at
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
	if source.Valid && source.String != "" {
		t.Source = source.String
	} else if strings.HasPrefix(t.Key, "FRE-") {
		t.Source = "linear"
	} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
		t.Source = "github"
	} else {
		t.Source = "local"
	}

	if extURL.Valid && extURL.String != "" {
		t.ExternalURL = &extURL.String
	} else if t.Source == "linear" {
		url := fmt.Sprintf("https://linear.app/fretzee/issue/%s", t.Key)
		t.ExternalURL = &url
	} else if t.Source == "github" {
		cleanNum := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(t.Key, "GH-#"), "gh-"), "#")
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

	mainRepoPath := ""
	if task.ProjectID != "" {
		if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil && proj.RepoPath != "" {
			mainRepoPath = proj.RepoPath
		}
	}
	if mainRepoPath == "" {
		if settings, _ := d.GetSettings(); settings != nil && settings.RepoPath != "" {
			mainRepoPath = settings.RepoPath
		}
	}

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

	repoPath := ""
	if task.ProjectID != "" {
		proj, _ := d.GetProjectByID(task.ProjectID)
		if proj != nil && proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}
	if repoPath == "" {
		settings, _ := d.GetSettings()
		if settings != nil && settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}

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
			LinearTeam: "FRE",
			GithubRepo: "sebastienferry/fretzee-studio",
			RepoPath:   "/Users/sferry/Sources/fretzee-studio",
		}
	}

	projID := req.ProjectID
	if projID == "" {
		projID = "fretzee-studio"
	}

	proj, _ := d.getProjectByIDUnsafe(projID)
	if proj == nil {
		projects, _ := d.GetProjects()
		if len(projects) > 0 {
			proj = &projects[0]
			projID = proj.ID
		}
	}

	linearTeam := settings.LinearTeam
	githubRepo := settings.GithubRepo
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
		}
		if proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
		if proj.LinearTeam != "" {
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
			// Fallback key with project prefix if CLI fails
			key, _ = d.getNextTaskKey(prefix)
		}
	} else if req.Source == "github" {
		created, err := d.runner.CreateGithubIssue(githubRepo, repoPath, req.Title, req.Description, req.Labels)
		if err == nil && created != nil {
			id = created.ID
			key = created.Key
			extURL = created.ExternalURL
		} else {
			key, _ = d.getNextTaskKey("GH")
		}
	} else {
		// Local project tracker
		key, _ = d.getNextTaskKey(prefix)
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
		SET project_id = ?, title = ?, description = ?, status = ?, priority = ?, labels = ?, assignee = ?, assignee_avatar = ?, position = ?, due_date = ?, branch_name = ?, pr_url = ?, source = ?, external_url = ?, updated_at = ?
		WHERE id = ? OR key = ?
	`, existing.ProjectID, existing.Title, existing.Description, string(existing.Status), string(existing.Priority), string(labelsJSON), existing.Assignee, existing.AssigneeAvatar, existing.Position, existing.DueDate, existing.BranchName, existing.PrURL, existing.Source, existing.ExternalURL, existing.UpdatedAt, existing.ID, existing.Key)

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
	var dueDate, branchName, prURL, source, extURL sql.NullString
	var statusStr, priorityStr string

	err := d.conn.QueryRow(`
		SELECT id, project_id, key, title, description, status, priority, labels, assignee, assignee_avatar, position, due_date, branch_name, pr_url, source, external_url, created_at, updated_at
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
	if source.Valid && source.String != "" {
		t.Source = source.String
	} else if strings.HasPrefix(t.Key, "FRE-") {
		t.Source = "linear"
	} else if strings.HasPrefix(t.Key, "#") || strings.HasPrefix(t.Key, "GH-#") || strings.HasPrefix(t.Key, "gh-") {
		t.Source = "github"
	} else {
		t.Source = "local"
	}

	if extURL.Valid && extURL.String != "" {
		t.ExternalURL = &extURL.String
	} else if t.Source == "linear" {
		url := fmt.Sprintf("https://linear.app/fretzee/issue/%s", t.Key)
		t.ExternalURL = &url
	} else if t.Source == "github" {
		cleanNum := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(t.Key, "GH-#"), "gh-"), "#")
		url := fmt.Sprintf("https://github.com/sebastienferry/fretzee-studio/issues/%s", cleanNum)
		t.ExternalURL = &url
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

func (d *DB) getTaskMessagesUnsafe(taskID string) ([]models.TaskMessage, error) {
	rows, err := d.conn.Query(`
		SELECT id, task_id, role, content, activity_id, skill_id, steps, created_at
		FROM task_messages
		WHERE task_id = ?
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return []models.TaskMessage{}, err
	}
	defer rows.Close()

	var list []models.TaskMessage
	for rows.Next() {
		var m models.TaskMessage
		var actID, skillID, stepsJSON sql.NullString
		if err := rows.Scan(&m.ID, &m.TaskID, &m.Role, &m.Content, &actID, &skillID, &stepsJSON, &m.CreatedAt); err != nil {
			continue
		}
		if actID.Valid {
			m.ActivityID = actID.String
		}
		if skillID.Valid {
			m.SkillID = skillID.String
		}
		if stepsJSON.Valid && stepsJSON.String != "" {
			_ = json.Unmarshal([]byte(stepsJSON.String), &m.Steps)
		}
		list = append(list, m)
	}
	if list == nil {
		list = []models.TaskMessage{}
	}
	return list, nil
}

func (d *DB) GetTaskMessages(taskID string) ([]models.TaskMessage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getTaskMessagesUnsafe(taskID)
}

func (d *DB) SaveTaskMessage(msg models.TaskMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	stepsJSON, _ := json.Marshal(msg.Steps)

	_, err := d.conn.Exec(`
		INSERT INTO task_messages (id, task_id, role, content, activity_id, skill_id, steps, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.TaskID, msg.Role, msg.Content, msg.ActivityID, msg.SkillID, string(stepsJSON), msg.CreatedAt)
	return err
}

func (d *DB) ClearTaskMessages(taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec("DELETE FROM task_messages WHERE task_id = ?", taskID)
	return err
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
				UserName:           "Developer",
				UserEmail:          "dev@example.com",
				UserAvatar:         "",
				AIProvider:         "agy",
				AICommandTemplate:  "agy -p \"{prompt}\"",
				RepoPath:           ".",
				IssueTracker:       "local",
				LinearTeam:         "",
				GithubRepo:         "",
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
	if s.UserName == "" { s.UserName = "Developer" }
	if s.UserEmail == "" { s.UserEmail = "dev@example.com" }
	if s.AIProvider == "" { s.AIProvider = "agy" }
	if s.AICommandTemplate == "" { s.AICommandTemplate = "agy -p \"{prompt}\"" }
	if s.RepoPath == "" { s.RepoPath = "." }
	if s.IssueTracker == "" { s.IssueTracker = "local" }
	if s.LinearTeam == "" { s.LinearTeam = "" }
	if s.GithubRepo == "" { s.GithubRepo = "" }

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
				LinearTeam: "FRE",
				GithubRepo: "sebastienferry/fretzee-studio",
				RepoPath:   "/Users/sferry/Sources/fretzee-studio",
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
			RepoPath:   "/Users/sferry/Sources/fretzee-studio",
			LinearTeam: "FRE",
			GithubRepo: "sebastienferry/fretzee-studio",
		}
	}

	// Dynamic per-project configuration override (CWD repository, Linear team, GitHub repo)
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
			if proj.IssueTracker != "" {
				settings.IssueTracker = proj.IssueTracker
			}
		}
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

	// Background state, label, and report comment sync with Linear / GitHub CLI
	if task.Source == "linear" || task.Source == "github" || strings.HasPrefix(task.Key, "FRE-") || strings.HasPrefix(task.Key, "#") || strings.HasPrefix(task.Key, "gh-") || strings.HasPrefix(task.Key, "GH-#") {
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
		team := settings.LinearTeam
		if job.Prompt != "" {
			team = job.Prompt
		}
		if team == "" {
			team = "FRE"
		}
		steps = append(steps, fmt.Sprintf("1. Connexion à la CLI Linear pour l'équipe %s...", team))
		outputLines = append(outputLines, fmt.Sprintf("### 🔄 Synchronisation Linear (Team: %s)\n", team))

		tasks, err := d.runner.SyncFromLinear(team)
		if err != nil {
			hasError = true
			errMsg := fmt.Sprintf("Échec de synchronisation Linear : %v", err)
			steps = append(steps, "⚠️ "+errMsg)
			outputLines = append(outputLines, "**Erreur :** "+errMsg)
			summary = "Erreur lors de la synchronisation Linear"
		} else {
			steps = append(steps, fmt.Sprintf("2. %d tickets récupérés depuis l'API Linear", len(tasks)))
			_ = d.ImportOrUpdateTasks(tasks)
			steps = append(steps, "3. Base de données locale mise à jour avec succès")
			totalImported = len(tasks)
			summary = fmt.Sprintf("%d issues Linear synchronisées avec succès", len(tasks))

			outputLines = append(outputLines, fmt.Sprintf("✅ **%d tickets importés / mis à jour depuis Linear :**\n", len(tasks)))
			for _, t := range tasks {
				outputLines = append(outputLines, fmt.Sprintf("- **[%s]** %s *(Statut: %s, Priorité: %s)*", t.Key, t.Title, t.Status, t.Priority))
			}
		}

	case "sync_github":
		repo := settings.GithubRepo
		if job.Prompt != "" {
			repo = job.Prompt
		}
		if repo == "" {
			repo = "sebastienferry/fretzee-studio"
		}
		repoPath := settings.RepoPath
		if repoPath == "" {
			repoPath = "/Users/sferry/Sources/fretzee-studio"
		}
		steps = append(steps, fmt.Sprintf("1. Connexion à GitHub CLI pour le repository %s...", repo))
		outputLines = append(outputLines, fmt.Sprintf("### 🐙 Synchronisation GitHub (%s)\n", repo))

		tasks, err := d.runner.SyncFromGithub(repo, repoPath)
		if err != nil {
			hasError = true
			errMsg := fmt.Sprintf("Échec de synchronisation GitHub : %v", err)
			steps = append(steps, "⚠️ "+errMsg)
			outputLines = append(outputLines, "**Erreur :** "+errMsg)
			summary = "Erreur lors de la synchronisation GitHub"
		} else {
			steps = append(steps, fmt.Sprintf("2. %d tickets récupérés depuis GitHub Issues", len(tasks)))
			_ = d.ImportOrUpdateTasks(tasks)
			steps = append(steps, "3. Base de données locale mise à jour avec succès")
			totalImported = len(tasks)
			summary = fmt.Sprintf("%d issues GitHub synchronisées avec succès", len(tasks))

			outputLines = append(outputLines, fmt.Sprintf("✅ **%d tickets importés / mis à jour depuis GitHub :**\n", len(tasks)))
			for _, t := range tasks {
				outputLines = append(outputLines, fmt.Sprintf("- **[%s]** %s *(Statut: %s, Priorité: %s)*", t.Key, t.Title, t.Status, t.Priority))
			}
		}

	case "sync_all":
		steps = append(steps, "1. Démarrage de la synchronisation globale...")
		outputLines = append(outputLines, "### 🌐 Synchronisation Globale (Linear + GitHub)\n")

		// 1. Linear
		team := settings.LinearTeam
		if team == "" {
			team = "FRE"
		}
		linTasks, linErr := d.runner.SyncFromLinear(team)
		if linErr != nil {
			steps = append(steps, fmt.Sprintf("⚠️ Linear (%s) : %v", team, linErr))
			outputLines = append(outputLines, fmt.Sprintf("❌ Linear (%s) : %v", team, linErr))
		} else {
			_ = d.ImportOrUpdateTasks(linTasks)
			steps = append(steps, fmt.Sprintf("2. Linear : %d issues importées", len(linTasks)))
			outputLines = append(outputLines, fmt.Sprintf("✅ Linear (%s) : %d issues synchronisées", team, len(linTasks)))
			totalImported += len(linTasks)
		}

		// 2. GitHub
		repo := settings.GithubRepo
		if repo == "" {
			repo = "sebastienferry/fretzee-studio"
		}
		repoPath := settings.RepoPath
		if repoPath == "" {
			repoPath = "/Users/sferry/Sources/fretzee-studio"
		}
		ghTasks, ghErr := d.runner.SyncFromGithub(repo, repoPath)
		if ghErr != nil {
			steps = append(steps, fmt.Sprintf("⚠️ GitHub (%s) : %v", repo, ghErr))
			outputLines = append(outputLines, fmt.Sprintf("❌ GitHub (%s) : %v", repo, ghErr))
		} else {
			_ = d.ImportOrUpdateTasks(ghTasks)
			steps = append(steps, fmt.Sprintf("3. GitHub : %d issues importées", len(ghTasks)))
			outputLines = append(outputLines, fmt.Sprintf("✅ GitHub (%s) : %d issues synchronisées", repo, len(ghTasks)))
			totalImported += len(ghTasks)
		}

		if linErr != nil && ghErr != nil {
			hasError = true
			summary = "Échec de synchronisation globale"
		} else {
			steps = append(steps, "4. Synchronisation globale terminée avec succès")
			summary = fmt.Sprintf("Synchronisation globale terminée (%d tickets mis à jour)", totalImported)
		}
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

	if !isLinear && !isGithub {
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
	default:
		syncType = "sync_all"
		skillName = "Sync Globale"
		summary = "Synchronisation globale (Linear + GitHub) en file d'attente"
		steps = []string{
			fmt.Sprintf("Cibles : Linear (Team: %s) & GitHub (%s)", linearTeam, githubRepo),
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

	act := models.TaskActivity{
		ID:        activityID,
		TaskID:    task.ID,
		SkillID:   targetSkill.ID,
		SkillName: targetSkill.Name,
		Action:    fmt.Sprintf("Exécution de la compétence %s", targetSkill.Name),
		Status:    string(models.ActivityStatusQueued),
		Summary:   fmt.Sprintf("Compétence %s en file d'attente pour la tâche %s", targetSkill.Name, task.Key),
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
	d.mu.RUnlock()

	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

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
			_ = d.runner.UpdateGithubIssue(repo, repoPath, newKey, nil, nil, &task.Status, task.Labels, nil)
		}

	default:
		return nil, fmt.Errorf("tracker distant non supporté: %s (choisir 'linear' ou 'github')", target)
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

func (d *DB) GetProjects() ([]models.Project, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT p.id, p.name, p.slug, p.description, p.icon, p.color, p.repo_path, p.git_remote_url, p.linear_team, p.github_repo, p.issue_tracker, p.tracker_url, p.is_default, p.stage_mapping, p.skill_overrides, p.created_at, p.updated_at,
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
		var stageMappingJSON, skillOverridesJSON string
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Icon, &p.Color, &p.RepoPath, &p.GitRemoteUrl, &p.LinearTeam, &p.GithubRepo, &p.IssueTracker, &p.TrackerUrl, &isDefault, &stageMappingJSON, &skillOverridesJSON, &p.CreatedAt, &p.UpdatedAt, &p.TaskCount,
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
		projects = append(projects, p)
	}
	if projects == nil {
		projects = []models.Project{}
	}
	return projects, nil
}

func (d *DB) GetProjectByID(id string) (*models.Project, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getProjectByIDUnsafe(id)
}

func (d *DB) getProjectByIDUnsafe(id string) (*models.Project, error) {
	var p models.Project
	var isDefault int
	var stageMappingJSON, skillOverridesJSON string
	err := d.conn.QueryRow(`
		SELECT p.id, p.name, p.slug, p.description, p.icon, p.color, p.repo_path, p.git_remote_url, p.linear_team, p.github_repo, p.issue_tracker, p.tracker_url, p.is_default, p.stage_mapping, p.skill_overrides, p.created_at, p.updated_at,
		       (SELECT COUNT(*) FROM tasks WHERE project_id = p.id) as task_count
		FROM projects p
		WHERE p.id = ? OR p.slug = ?
	`, id, id).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Icon, &p.Color, &p.RepoPath, &p.GitRemoteUrl, &p.LinearTeam, &p.GithubRepo, &p.IssueTracker, &p.TrackerUrl, &isDefault, &stageMappingJSON, &skillOverridesJSON, &p.CreatedAt, &p.UpdatedAt, &p.TaskCount,
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

	_, err := d.conn.Exec(`
		INSERT INTO projects (id, name, slug, description, icon, color, repo_path, git_remote_url, linear_team, github_repo, issue_tracker, tracker_url, is_default, stage_mapping, skill_overrides, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, name, slug, req.Description, icon, color, req.RepoPath, gitRemote, req.LinearTeam, githubRepo, issueTracker, req.TrackerUrl, isDefInt, string(stageMappingBytes), string(skillOverridesBytes), now, now)
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

	_, err = d.conn.Exec(`
		UPDATE projects
		SET name = ?, slug = ?, description = ?, icon = ?, color = ?, repo_path = ?, git_remote_url = ?, linear_team = ?, github_repo = ?, issue_tracker = ?, tracker_url = ?, is_default = ?, stage_mapping = ?, skill_overrides = ?, updated_at = ?
		WHERE id = ?
	`, p.Name, p.Slug, p.Description, p.Icon, p.Color, p.RepoPath, p.GitRemoteUrl, p.LinearTeam, p.GithubRepo, p.IssueTracker, p.TrackerUrl, isDefInt, string(stageMappingBytes), string(skillOverridesBytes), p.UpdatedAt, p.ID)
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
	if p.IsDefault || p.ID == "fretzee-studio" {
		return fmt.Errorf("impossible de supprimer le projet par défaut")
	}

	// Reassign tasks to default project
	_, _ = d.conn.Exec("UPDATE tasks SET project_id = 'fretzee-studio' WHERE project_id = ?", p.ID)
	_, err = d.conn.Exec("DELETE FROM projects WHERE id = ?", p.ID)
	return err
}

// -------------------------------------------------------------
// PROJECT SKILLS MANAGEMENT & PROVISIONING
// -------------------------------------------------------------

var DefaultProjectSkills = []struct {
	ID          string
	Name        string
	DirName     string
	Description string
	Content     string
}{
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
		Description: "Routeur intelligent orchestrant automatiquement le cycle de vie complet de la tâche.",
		Content: `---
name: pick-issue
description: Auto-pilote intelligent orchestrant automatiquement le cycle de vie de la tâche.
---
# Skill : Auto-Pilot Orchestrator

## Objectif
Déterminer automatiquement l'étape actuelle du ticket et enchaîner l'action optimale (Clarify -> Specify -> Implement -> PR).
`,
	},
}

func (d *DB) GetProjectSkillsStatus(projectIDOrPath string) (*models.ProjectSkillsStatus, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	repoPath := projectIDOrPath
	projectID := projectIDOrPath
	projectName := projectIDOrPath

	if proj, _ := d.getProjectByIDUnsafe(projectIDOrPath); proj != nil {
		projectID = proj.ID
		projectName = proj.Name
		if proj.RepoPath != "" {
			repoPath = proj.RepoPath
		}
	}

	res := &models.ProjectSkillsStatus{
		ProjectID:    projectID,
		ProjectName:  projectName,
		RepoPath:     repoPath,
		PathExists:   false,
		InstalledAll: true,
		Skills:       []models.InstalledSkillInfo{},
	}

	if repoPath == "" {
		res.InstalledAll = false
		return res, nil
	}

	if fi, err := os.Stat(repoPath); err == nil && fi.IsDir() {
		res.PathExists = true
		gitDir := filepath.Join(repoPath, ".git")
		if gfi, gerr := os.Stat(gitDir); gerr == nil && gfi.IsDir() {
			res.IsGitRepo = true
			branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
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
		// Check both .gemini/skills and .agy/skills and .skills
		p1 := filepath.Join(repoPath, ".gemini", "skills", s.DirName, "SKILL.md")
		p2 := filepath.Join(repoPath, ".agy", "skills", s.DirName, "SKILL.md")
		p3 := filepath.Join(repoPath, ".skills", s.DirName, "SKILL.md")

		installed := false
		targetPath := p1
		if _, err := os.Stat(p1); err == nil {
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

		res.Skills = append(res.Skills, models.InstalledSkillInfo{
			ID:          s.ID,
			Name:        s.Name,
			Installed:   installed,
			Path:        targetPath,
			Description: s.Description,
		})
	}

	return res, nil
}

func (d *DB) InstallProjectSkills(projectIDOrPath string) (*models.ProjectSkillsStatus, error) {
	d.mu.RLock()
	repoPath := projectIDOrPath
	projectID := projectIDOrPath
	projectName := projectIDOrPath
	linearTeam := "FRE"
	githubRepo := ""
	issueTracker := "local"

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
	}
	d.mu.RUnlock()

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("le chemin du dossier de travail (CWD) est obligatoire")
	}

	// Create CWD if doesn't exist
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("impossible de créer le répertoire %s: %w", repoPath, err)
	}

	// Install skills into .gemini/skills/ and .agy/skills/
	for _, s := range DefaultProjectSkills {
		dirs := []string{
			filepath.Join(repoPath, ".gemini", "skills", s.DirName),
			filepath.Join(repoPath, ".agy", "skills", s.DirName),
		}

		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("erreur création dossier skill %s: %w", dir, err)
			}
			filePath := filepath.Join(dir, "SKILL.md")
			if err := os.WriteFile(filePath, []byte(s.Content), 0644); err != nil {
				return nil, fmt.Errorf("erreur écriture skill %s: %w", filePath, err)
			}
		}
	}

	// Create .taskacao/config.json in project root
	taskacaoDir := filepath.Join(repoPath, ".taskacao")
	_ = os.MkdirAll(taskacaoDir, 0755)
	configFile := filepath.Join(taskacaoDir, "config.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		cfgData := map[string]interface{}{
			"projectId":    projectID,
			"projectName":  projectName,
			"linearTeam":   linearTeam,
			"githubRepo":   githubRepo,
			"issueTracker": issueTracker,
			"skills": []string{
				"clarify-issue",
				"specify-issue",
				"code-issue",
				"create-pr",
				"pick-issue",
			},
			"createdAt": time.Now().Format(time.RFC3339),
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


