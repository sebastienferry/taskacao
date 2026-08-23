# API & Data Specifications

This document defines the SQLite schema, domain data models, REST endpoints, and WebSocket streaming protocols for **Taskacao**.

---

## 1. SQLite Database Schema

The database file is located at `tasks.db` in the server root. Foreign keys and WAL journal mode are enabled on connection.

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

-- Projects Table
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    icon TEXT DEFAULT 'folder',
    color TEXT DEFAULT 'indigo',
    repo_path TEXT NOT NULL DEFAULT '.',
    git_remote_url TEXT DEFAULT '',
    linear_team TEXT DEFAULT 'TASK',
    github_repo TEXT DEFAULT '',
    issue_tracker TEXT NOT NULL DEFAULT 'local',
    tracker_url TEXT DEFAULT '',
    is_default INTEGER DEFAULT 0,
    stage_mapping TEXT DEFAULT '{}',
    skill_overrides TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tasks Table
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL DEFAULT 'default',
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'to_clarify',
    priority TEXT NOT NULL DEFAULT 'medium',
    labels TEXT DEFAULT '[]',
    assignee TEXT DEFAULT '',
    assignee_avatar TEXT DEFAULT '',
    position INTEGER DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'local',
    external_id TEXT DEFAULT '',
    external_url TEXT DEFAULT '',
    branch_name TEXT,
    pr_url TEXT,
    worktree_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    due_date DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Task Activities Table (Background Jobs & Skill Runs)
CREATE TABLE IF NOT EXISTS task_activities (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    skill_id TEXT NOT NULL,
    skill_name TEXT NOT NULL,
    action TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    summary TEXT DEFAULT '',
    output TEXT DEFAULT '',
    steps TEXT DEFAULT '[]',
    prompt TEXT DEFAULT '',
    error TEXT DEFAULT '',
    duration TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_activities_task_id ON task_activities(task_id);
CREATE INDEX IF NOT EXISTS idx_activities_status ON task_activities(status);

-- Task Messages Table (Chat History)
CREATE TABLE IF NOT EXISTS task_messages (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    role TEXT NOT NULL, -- 'user' | 'assistant' | 'system'
    content TEXT NOT NULL,
    activity_id TEXT,
    skill_id TEXT,
    steps TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_task_id ON task_messages(task_id);

-- Global Settings Table
CREATE TABLE IF NOT EXISTS settings (
    id TEXT PRIMARY KEY,
    ai_provider TEXT DEFAULT 'antigravity',
    linear_api_key TEXT DEFAULT '',
    linear_team TEXT DEFAULT 'TASK',
    github_token TEXT DEFAULT '',
    github_repo TEXT DEFAULT '',
    repo_path TEXT DEFAULT '.',
    auto_create_branch INTEGER DEFAULT 1,
    auto_create_worktree INTEGER DEFAULT 1,
    theme TEXT DEFAULT 'dark',
    accent_color TEXT DEFAULT 'indigo',
    language TEXT DEFAULT 'fr',
    detail_mode TEXT DEFAULT 'panel',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 2. REST API Endpoints

### 2.1 Tasks API

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/tasks` | Returns array of all tasks (supports `?projectId=...`). |
| `POST` | `/api/tasks` | Creates a new task bound strictly to `projectId`. |
| `GET` | `/api/tasks/{id}` | Fetches task detail with messages and activities. |
| `PUT` | `/api/tasks/{id}` | Updates task fields (status, title, description, priority, etc.). |
| `DELETE` | `/api/tasks/{id}` | Deletes task and prunes associated Git worktree. |
| `POST` | `/api/tasks/{id}/skills/{skillId}` | Enqueues or immediately executes an AI skill on the task. |
| `POST` | `/api/tasks/{id}/chat` | Posts a user message and streams the AI agent's response. |
| `POST` | `/api/tasks/{id}/comment` | Publishes a comment to Linear or GitHub issue tracker. |
| `GET` | `/api/tasks/{id}/diff` | Computes and returns the Git diff of the task branch vs `main`. |

### 2.2 Activities API

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/activities` | Lists recent activities (supports `?taskId=...&status=...`). |
| `GET` | `/api/activities/stats` | Returns aggregate counts (`total`, `queued`, `running`, `completed`, `failed`). |
| `POST` | `/api/activities/{id}/retry` | Re-enqueues a failed activity. |
| `POST` | `/api/activities/{id}/cancel` | Cancels a running or queued activity. |
| `DELETE` | `/api/activities/{id}` | Deletes an activity entry. |
| `DELETE` | `/api/activities` | Clears all completed and canceled activities. |

### 2.3 Projects API

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/projects` | Lists all projects with their task counters. |
| `POST` | `/api/projects` | Creates a new workspace project. |
| `PUT` | `/api/projects/{id}` | Updates project configuration, tracker binding, and paths. |
| `DELETE` | `/api/projects/{id}` | Deletes project and its associated tasks. |
| `POST` | `/api/projects/{id}/skills/install` | Installs default skills into `.gemini/` and `.agents/`. |

---

## 3. WebSocket Terminal Protocol (`/ws/terminal`)

### Connection Handshake
- **URL**: `ws://<host>:<port>/ws/terminal?taskId=<taskId>`
- Automatically resolves worktree directory (`.tasks/worktrees/<taskKey>`) or project `repo_path`.
- Starts login shell `/bin/zsh -l` with PTY attached.

### Frame Formats

#### Client to Server
1. **Raw Keystrokes**: Standard text or binary bytes representing user keystrokes (e.g. `ls -la\n`, `agy\n`, `\x03` for Ctrl+C).
2. **Control Message (Window Resize)**:
   ```json
   {
     "type": "resize",
     "cols": 120,
     "rows": 36
   }
   ```

#### Server to Client
- Raw ANSI streaming output chunk (text or binary).
- Output is simultaneously appended to the 64KB circular replay buffer.
