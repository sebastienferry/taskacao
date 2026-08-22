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
	StatusToClarify   Status = "to_clarify"   // A clarifier (Label: New)
	StatusToSpecify   Status = "to_specify"   // A spécifier (Label: Clarified)
	StatusToImplement Status = "to_implement" // A implémenter (Label: Specified)
	StatusToTest      Status = "to_test"      // A tester (Label: Implemented)
	StatusToClose     Status = "to_close"     // A fermer (Label: Reviewed)

	// Compatibility aliases
	StatusBacklog    Status = "backlog"
	StatusSpecified  Status = "specified"
	StatusInProgress Status = "in_progress"
	StatusToValidate Status = "to_validate"
	StatusDone       Status = "done"
)

type TaskActivity struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	SkillID   string    `json:"skillId"`
	SkillName string    `json:"skillName"`
	Action    string    `json:"action"`
	Status    string    `json:"status"` // "running", "completed", "failed"
	Summary   string    `json:"summary"`
	Output    string    `json:"output"` // Real AI or CLI Markdown / logs
	Steps     []string  `json:"steps"`
	CreatedAt time.Time `json:"createdAt"`
}

type Task struct {
	ID             string         `json:"id"`
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
	Source         string         `json:"source,omitempty"` // "linear", "github", "local"
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
	RepoPath           string    `json:"repoPath"`          // e.g. '/Users/sferry/Sources/fretzee-studio'
	IssueTracker       string    `json:"issueTracker"`      // "linear", "github", "local"
	LinearTeam         string    `json:"linearTeam"`        // "FRE"
	GithubRepo         string    `json:"githubRepo"`        // "fretzee/studio"
	PromptClarify      string    `json:"promptClarify"`
	PromptSpecify      string    `json:"promptSpecify"`
	PromptImplement    string    `json:"promptImplement"`
	PromptCreatePR     string    `json:"promptCreatePr"`
	PromptPick         string    `json:"promptPick"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type MoveTaskRequest struct {
	Status   Status `json:"status"`
	Position int    `json:"position"`
}

type CreateTaskRequest struct {
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

