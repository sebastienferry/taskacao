package models

import "time"

type Priority string

const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

type Status string

const (
	StatusToClarify   Status = "to_clarify"   // A clarifier (Label: #new)
	StatusToSpecify   Status = "to_specify"   // A spécifier (Label: #clarified)
	StatusToImplement Status = "to_implement" // A implémenter (Label: #specified)
	StatusToTest      Status = "to_test"      // A tester (Label: #implemented)
	StatusToClose     Status = "to_close"     // En revue / PR (Label: #reviewed)
	StatusFinished    Status = "finished"     // Terminé (Label: #finished)

	// Compatibility aliases
	StatusBacklog    Status = "backlog"
	StatusSpecified  Status = "specified"
	StatusInProgress Status = "in_progress"
	StatusToValidate Status = "to_validate"
	StatusDone       Status = "done"
)

type ActivityStatus string

const (
	ActivityStatusQueued    ActivityStatus = "queued"
	ActivityStatusPending   ActivityStatus = "pending"
	ActivityStatusRunning   ActivityStatus = "running"
	ActivityStatusCompleted ActivityStatus = "completed"
	ActivityStatusFailed    ActivityStatus = "failed"
	ActivityStatusCanceled  ActivityStatus = "canceled"
)

type TaskActivity struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"taskId"`
	ProjectID   string     `json:"projectId,omitempty"`
	TaskKey     string     `json:"taskKey,omitempty"`
	TaskTitle   string     `json:"taskTitle,omitempty"`
	SkillID     string     `json:"skillId"`
	SkillName   string     `json:"skillName"`
	Action      string     `json:"action"`
	Status      string     `json:"status"` // "queued", "pending", "running", "completed", "failed", "canceled"
	Summary     string     `json:"summary"`
	Output      string     `json:"output"` // Real AI or CLI Markdown / logs
	Steps       []string   `json:"steps"`
	Prompt      string     `json:"prompt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Error       string     `json:"error,omitempty"`
	Duration    string     `json:"duration,omitempty"`
}

type ActivityStats struct {
	Total     int `json:"total"`
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Canceled  int `json:"canceled"`
}

type Project struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Slug               string            `json:"slug"`
	Description        string            `json:"description"`
	Icon               string            `json:"icon"`         // "Folder", "Terminal", "Flame", "Zap", "Layers", "Code2", "Box", "Cpu"
	Color              string            `json:"color"`        // "indigo", "emerald", "purple", "amber", "blue", "rose", "cyan", "orange", "neon-cyan", etc.
	RepoPath           string            `json:"repoPath"`     // CWD for AI skills
	GitRemoteUrl       string            `json:"gitRemoteUrl"` // e.g. "git@github.com:owner/repo.git"
	LinearTeam         string            `json:"linearTeam"`   // Key / prefix (optionnel)
	GithubRepo         string            `json:"githubRepo"`   // e.g. "owner/repo"
	IssueTracker       string            `json:"issueTracker"` // "linear", "github", "local"
	TrackerUrl         string            `json:"trackerUrl"`   // e.g. "https://linear.app/my-team/project/xxx"
	IsDefault          bool              `json:"isDefault"`
	StageMapping       map[string]string `json:"stageMapping,omitempty"`   // mapping AI workflow labels to tracker statuses
	SkillOverrides     map[string]string `json:"skillOverrides,omitempty"` // skillId -> custom skill name override
	AIProvider         string            `json:"aiProvider,omitempty"`        // "agy", "claude", "vibe", "gemini", "cursor", "custom"
	AICommandTemplate  string            `json:"aiCommandTemplate,omitempty"` // e.g. 'agy -p "{prompt}"'
	SpecFramework      string            `json:"specFramework,omitempty"`      // "speckit", "openfeature"
	TaskCount          int               `json:"taskCount"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type CreateProjectRequest struct {
	Name               string            `json:"name"`
	Slug               string            `json:"slug,omitempty"`
	Description        string            `json:"description,omitempty"`
	Icon               string            `json:"icon,omitempty"`
	Color              string            `json:"color,omitempty"`
	RepoPath           string            `json:"repoPath,omitempty"`
	GitRemoteUrl       string            `json:"gitRemoteUrl,omitempty"`
	LinearTeam         string            `json:"linearTeam,omitempty"`
	GithubRepo         string            `json:"githubRepo,omitempty"`
	IssueTracker       string            `json:"issueTracker,omitempty"`
	TrackerUrl         string            `json:"trackerUrl,omitempty"`
	IsDefault          bool              `json:"isDefault,omitempty"`
	StageMapping       map[string]string `json:"stageMapping,omitempty"`
	SkillOverrides     map[string]string `json:"skillOverrides,omitempty"`
	AIProvider         string            `json:"aiProvider,omitempty"`
	AICommandTemplate  string            `json:"aiCommandTemplate,omitempty"`
	SpecFramework      string            `json:"specFramework,omitempty"`
}

type UpdateProjectRequest struct {
	Name               *string            `json:"name,omitempty"`
	Slug               *string            `json:"slug,omitempty"`
	Description        *string            `json:"description,omitempty"`
	Icon               *string            `json:"icon,omitempty"`
	Color              *string            `json:"color,omitempty"`
	RepoPath           *string            `json:"repoPath,omitempty"`
	GitRemoteUrl       *string            `json:"gitRemoteUrl,omitempty"`
	LinearTeam         *string            `json:"linearTeam,omitempty"`
	GithubRepo         *string            `json:"githubRepo,omitempty"`
	IssueTracker       *string            `json:"issueTracker,omitempty"`
	TrackerUrl         *string            `json:"trackerUrl,omitempty"`
	IsDefault          *bool              `json:"isDefault,omitempty"`
	StageMapping       *map[string]string `json:"stageMapping,omitempty"`
	SkillOverrides     *map[string]string `json:"skillOverrides,omitempty"`
	AIProvider         *string            `json:"aiProvider,omitempty"`
	AICommandTemplate  *string            `json:"aiCommandTemplate,omitempty"`
	SpecFramework      *string            `json:"specFramework,omitempty"`
}

type InstalledSkillInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Installed   bool   `json:"installed"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type ProjectSkillsStatus struct {
	ProjectID      string               `json:"projectId"`
	ProjectName    string               `json:"projectName"`
	RepoPath       string               `json:"repoPath"`
	PathExists     bool                 `json:"pathExists"`
	IsGitRepo      bool                 `json:"isGitRepo"`
	GitBranch      string               `json:"gitBranch,omitempty"`
	InstalledAll   bool                 `json:"installedAll"`
	SpecFramework  string               `json:"specFramework,omitempty"`
	WorktreesCount int                  `json:"worktreesCount,omitempty"`
	WorktreePaths  []string             `json:"worktreePaths,omitempty"`
	Skills         []InstalledSkillInfo `json:"skills"`
}

type ProjectGitInitResult struct {
	RepoPath    string `json:"repoPath"`
	IsGitRepo   bool   `json:"isGitRepo"`
	Branch      string `json:"branch"`
	Message     string `json:"message"`
	Initialized bool   `json:"initialized"`
}

type DetectedStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type,omitempty"`   // "backlog", "unstarted", "started", "completed", "canceled", "triage", "custom"
	Color  string `json:"color,omitempty"`
	Source string `json:"source,omitempty"` // "linear", "github", "db", "preset"
}

type GitBranchItem struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"isCurrent"`
	IsRemote  bool   `json:"isRemote"`
	Commit    string `json:"commit,omitempty"`
	Message   string `json:"message,omitempty"`
}

type GitBranchesInfo struct {
	RepoPath      string          `json:"repoPath"`
	CurrentBranch string          `json:"currentBranch"`
	Branches      []GitBranchItem `json:"branches"`
}

type CleanBranchesResult struct {
	RepoPath        string   `json:"repoPath"`
	DefaultBranch   string   `json:"defaultBranch"`
	DeletedBranches []string `json:"deletedBranches"`
	Message         string   `json:"message"`
}

type GitDiffFile struct {
	Path      string `json:"path"`
	OldPath   string `json:"oldPath,omitempty"`
	Status    string `json:"status"` // "modified", "added", "deleted", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Diff      string `json:"diff"`
}

type GitDiffResult struct {
	TaskKey      string        `json:"taskKey"`
	Branch       string        `json:"branch"`
	BaseBranch   string        `json:"baseBranch"`
	RepoPath     string        `json:"repoPath"`
	WorktreePath string        `json:"worktreePath,omitempty"`
	IsClean      bool          `json:"isClean"`
	FilesChanged int           `json:"filesChanged"`
	Insertions   int           `json:"insertions"`
	Deletions    int           `json:"deletions"`
	Files        []GitDiffFile `json:"files"`
	RawDiff      string        `json:"rawDiff"`
	PrURL        *string       `json:"prUrl,omitempty"`
	Error        string        `json:"error,omitempty"`
}

type GitStatusInfo struct {
	RepoPath       string `json:"repoPath"`
	IsGitRepo      bool   `json:"isGitRepo"`
	Branch         string `json:"branch"`
	BaseBranch     string `json:"baseBranch,omitempty"`
	IsClean        bool   `json:"isClean"`
	ModifiedCount  int    `json:"modifiedCount"`
	UntrackedCount int    `json:"untrackedCount"`
	Ahead          int    `json:"ahead"`
	Behind         int    `json:"behind"`
	RemoteName     string `json:"remoteName,omitempty"`
	RemoteURL      string `json:"remoteUrl,omitempty"`
	LatestCommit   string `json:"latestCommit,omitempty"`
	Error          string `json:"error,omitempty"`
}

type WorktreeInfo struct {
	TaskKey      string `json:"taskKey"`
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktreePath"`
	Exists       bool   `json:"exists"`
	MainRepoPath string `json:"mainRepoPath"`
}

type Task struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	Key            string         `json:"key"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Status         Status         `json:"status"`
	Priority       Priority       `json:"priority"`
	Labels         []string       `json:"labels"`
	Assignee       string         `json:"assignee"`
	AssigneeAvatar string         `json:"assigneeAvatar"`
	Position       int            `json:"position"`
	DueDate        *string        `json:"dueDate"`
	BranchName     *string        `json:"branchName,omitempty"`
	PrURL          *string        `json:"prUrl,omitempty"`
	WorktreePath   *string        `json:"worktreePath,omitempty"`
	Source         string         `json:"source"` // "linear", "github", "jira", "local"
	ExternalURL    *string        `json:"externalUrl,omitempty"`
	Activities     []TaskActivity `json:"activities,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type Settings struct {
	ID                 int       `json:"id"`
	Theme              string    `json:"theme"`       // "dark", "light", "system"
	AccentColor        string    `json:"accentColor"` // "indigo", "violet", "emerald", "amber", "rose", "cyan", "blue", "orange"
	Language           string    `json:"language"`    // "fr", "en"
	Density            string    `json:"density"`     // "compact", "standard", "comfortable"
	DefaultView        string    `json:"defaultView"` // "board", "list"
	DetailMode         string    `json:"detailMode"`  // "panel", "modal"
	UserName           string    `json:"userName"`
	UserEmail          string    `json:"userEmail"`
	UserAvatar         string    `json:"userAvatar"`
	AIProvider         string    `json:"aiProvider"`        // "agy", "vibe", "claude", "custom"
	AICommandTemplate  string    `json:"aiCommandTemplate"` // e.g. 'agy -p "{prompt}"'
	RepoPath           string    `json:"repoPath"`          // e.g. '/path/to/project'
	IssueTracker       string    `json:"issueTracker"`      // "linear", "github", "local"
	LinearTeam         string    `json:"linearTeam"`        // e.g. "ENG"
	GithubRepo         string    `json:"githubRepo"`        // e.g. "owner/repo"
	PromptClarify      string    `json:"promptClarify"`
	PromptSpecify      string    `json:"promptSpecify"`
	PromptImplement    string    `json:"promptImplement"`
	PromptCreatePR     string    `json:"promptCreatePr"`
	PromptPick         string    `json:"promptPick"`
	EditorCommand      string    `json:"editorCommand"` // "code", "cursor", "zed", "subl", etc.
	SpecFramework      string    `json:"specFramework"` // "speckit", "openfeature"
	UpdatedAt          time.Time `json:"updatedAt"`
}

type MoveTaskRequest struct {
	Status   Status `json:"status"`
	Position int    `json:"position"`
}

type CreateTaskRequest struct {
	ProjectID      string   `json:"projectId,omitempty"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         Status   `json:"status"`
	Priority       Priority `json:"priority"`
	Labels         []string `json:"labels"`
	Assignee       string   `json:"assignee"`
	AssigneeAvatar string   `json:"assigneeAvatar"`
	DueDate        *string  `json:"dueDate"`
	Source         string   `json:"source,omitempty"`
	ExternalURL    *string  `json:"externalUrl,omitempty"`
}

type UpdateTaskRequest struct {
	ProjectID      *string   `json:"projectId,omitempty"`
	Title          *string   `json:"title,omitempty"`
	Description    *string   `json:"description,omitempty"`
	Status         *Status   `json:"status,omitempty"`
	Priority       *Priority `json:"priority,omitempty"`
	Labels         *[]string `json:"labels,omitempty"`
	Assignee       *string   `json:"assignee,omitempty"`
	AssigneeAvatar *string   `json:"assigneeAvatar,omitempty"`
	Position       *int      `json:"position,omitempty"`
	DueDate        *string   `json:"dueDate,omitempty"`
	BranchName     *string   `json:"branchName,omitempty"`
	PrURL          *string   `json:"prUrl,omitempty"`
	Source         *string   `json:"source,omitempty"`
	ExternalURL    *string   `json:"externalUrl,omitempty"`
}

type Skill struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Command      string   `json:"command"`
	Description  string   `json:"description"`
	InputStatus  Status   `json:"inputStatus"`
	OutputStatus Status   `json:"outputStatus"`
	Icon         string   `json:"icon"`
	Color        string   `json:"color"`
	Steps        []string `json:"steps"`
}

type RunSkillRequest struct {
	SkillID string `json:"skillId"`
	Prompt  string `json:"prompt,omitempty"`
}

type RunSkillResponse struct {
	Task     Task         `json:"task"`
	Activity TaskActivity `json:"activity"`
	Message  string       `json:"message"`
}

type CliStatus struct {
	Tool       string `json:"tool"`
	Available  bool   `json:"available"`
	Path       string `json:"path"`
	AuthStatus string `json:"authStatus"`
	Details    string `json:"details"`
}

type ConvertTaskRequest struct {
	Target string `json:"target"` // "linear" or "github"
}

type TaskMessage struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	Role       string    `json:"role"` // "user", "assistant", "system"
	Content    string    `json:"content"`
	ActivityID string    `json:"activityId,omitempty"`
	SkillID    string    `json:"skillId,omitempty"`
	Steps      []string  `json:"steps,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type TaskChatRequest struct {
	Message string `json:"message"`
	SkillID string `json:"skillId,omitempty"`
}

