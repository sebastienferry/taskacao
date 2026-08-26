package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"tasks/internal/models"
)

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func GetDynamicCustomPath() string {
	homeDir, _ := os.UserHomeDir()
	var parts []string
	if homeDir != "" {
		parts = append(parts, filepath.Join(homeDir, ".local", "bin"))
	}
	parts = append(parts, "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin")
	return strings.Join(parts, ":")
}

func FindCliTool(tool string) (string, error) {
	if p, err := exec.LookPath(tool); err == nil {
		return p, nil
	}
	homeDir, _ := os.UserHomeDir()
	var candidates []string
	if homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".local", "bin", tool))
	}
	candidates = append(candidates, "/opt/homebrew/bin/"+tool, "/usr/local/bin/"+tool, "/usr/bin/"+tool)
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return tool, fmt.Errorf("tool %s not found", tool)
}

// Helper to execute command with timeout and full environment
func (r *Runner) runCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// Inherit and extend PATH dynamically to include ~/.local/bin and Homebrew paths
	env := os.Environ()
	customPath := GetDynamicCustomPath()
	foundPath := false
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + customPath + ":" + strings.TrimPrefix(e, "PATH=")
			foundPath = true
			break
		}
	}
	if !foundPath {
		env = append(env, "PATH="+customPath)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	errOutput := stderr.String()

	if err != nil {
		combined := strings.TrimSpace(output + "\n" + errOutput)
		if combined == "" {
			combined = err.Error()
		}
		return combined, fmt.Errorf("command '%s %s' failed: %w (output: %s)", name, strings.Join(args, " "), err, combined)
	}

	if output == "" && errOutput != "" {
		return errOutput, nil
	}
	return output, nil
}

func (r *Runner) CheckCliTools(repoPath string) []models.CliStatus {
	tools := []string{"git", "gh", "linear", "acli", "agy", "vibe", "claude", "gemini", "codex", "uv", "specify", "openspec"}
	var results []models.CliStatus

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tool := range tools {
		path, err := FindCliTool(tool)

		status := models.CliStatus{
			Tool:      tool,
			Available: err == nil,
			Path:      path,
		}

		if status.Available {
			switch tool {
			case "gh":
				out, aErr := r.runCommand(ctx, repoPath, path, "auth", "status")
				if aErr == nil || strings.Contains(out, "Logged in to") {
					status.AuthStatus = "Authenticated"
					status.Details = "GitHub CLI connected"
				} else {
					status.AuthStatus = "Not Authenticated"
					status.Details = "Run 'gh auth login'"
				}
			case "linear":
				out, aErr := r.runCommand(ctx, repoPath, path, "auth", "whoami")
				if aErr == nil && strings.Contains(out, "Workspace:") {
					status.AuthStatus = "Authenticated"
					status.Details = "Linear connected"
				} else {
					status.AuthStatus = "Not Authenticated"
					status.Details = "Run 'linear auth login'"
				}
			case "acli":
				out, aErr := r.runCommand(ctx, repoPath, path, "jira", "auth", "status")
				if aErr != nil {
					out, aErr = r.runCommand(ctx, repoPath, path, "auth", "status")
				}
				if aErr == nil || strings.Contains(out, "Logged in") || strings.Contains(out, "Active account") {
					status.AuthStatus = "Authenticated"
					status.Details = "Atlassian CLI (Jira) connected"
				} else {
					status.AuthStatus = "Not Authenticated"
					status.Details = "Run 'acli jira auth login'"
				}
			case "uv":
				status.AuthStatus = "Ready"
				status.Details = "uv available (required to install GitHub Spec Kit)"
			case "specify":
				status.AuthStatus = "Ready"
				status.Details = "GitHub Spec Kit CLI installed"
			case "openspec":
				status.AuthStatus = "Ready"
				status.Details = "OpenSpec CLI installed"
			case "git":
				status.AuthStatus = "Ready"
				status.Details = "Git available"
			case "agy":
				status.AuthStatus = "Ready"
				status.Details = "Antigravity CLI Agent ready"
			case "vibe":
				status.AuthStatus = "Ready"
				status.Details = "Mistral Vibe CLI Agent ready"
			case "claude":
				status.AuthStatus = "Ready"
				status.Details = "Claude Code CLI Agent ready"
			case "gemini":
				status.AuthStatus = "Ready"
				status.Details = "Gemini CLI Agent ready"
			case "codex":
				status.AuthStatus = "Ready"
				status.Details = "Codex CLI Agent ready"
			}
		} else {
			status.AuthStatus = "Not Installed"
			switch tool {
			case "acli":
				status.Details = "Atlassian CLI missing — see https://developer.atlassian.com/cloud/acli/"
			case "uv":
				status.Details = "uv missing — curl -LsSf https://astral.sh/uv/install.sh | sh"
			case "specify":
				status.Details = "GitHub Spec Kit missing — install it from a project (Spec Kit / OpenSpec panel)"
			case "openspec":
				status.Details = "OpenSpec missing — install it from a project (Spec Kit / OpenSpec panel)"
			default:
				status.Details = fmt.Sprintf("Tool '%s' not found in PATH", tool)
			}
		}

		results = append(results, status)
	}

	return results
}

// Linear API Data Models
type LinearIssueNode struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Priority    int    `json:"priority"`
	State       struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"`
		Color string `json:"color"`
	} `json:"state"`
	Assignee *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		AvatarUrl   string `json:"avatarUrl"`
	} `json:"assignee"`
	Labels *struct {
		Nodes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type LinearQueryResponse struct {
	Nodes []LinearIssueNode `json:"nodes"`
}

func (r *Runner) SyncFromLinear(teamKey string) ([]models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	linearPath, _ := FindCliTool("linear")
	if linearPath == "" {
		linearPath = "linear"
	}

	var args []string
	if teamKey != "" {
		args = []string{"issue", "query", "--team", teamKey, "--json"}
	} else {
		args = []string{"issue", "query", "--json"}
	}

	output, err := r.runCommand(ctx, "", linearPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query Linear: %w (output: %s)", err, output)
	}

	var resp LinearQueryResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse Linear JSON response: %w", err)
	}

	var tasks []models.Task
	for i, node := range resp.Nodes {
		var priority models.Priority
		switch node.Priority {
		case 1:
			priority = models.PriorityUrgent
		case 2:
			priority = models.PriorityHigh
		case 3:
			priority = models.PriorityMedium
		default:
			priority = models.PriorityLow
		}

		var status models.Status
		stateName := strings.ToLower(node.State.Name)
		switch {
		case strings.Contains(stateName, "clarif") || strings.Contains(stateName, "triage") || strings.Contains(stateName, "backlog"):
			status = models.StatusToClarify
		case strings.Contains(stateName, "specif") || strings.Contains(stateName, "todo") || strings.Contains(stateName, "unstarted"):
			status = models.StatusToSpecify
		case strings.Contains(stateName, "progress") || strings.Contains(stateName, "started") || strings.Contains(stateName, "implem"):
			status = models.StatusToImplement
		case strings.Contains(stateName, "test") || strings.Contains(stateName, "valid") || strings.Contains(stateName, "review"):
			status = models.StatusToTest
		case strings.Contains(stateName, "done") || strings.Contains(stateName, "completed") || strings.Contains(stateName, "close") || strings.Contains(stateName, "cancel"):
			status = models.StatusToClose
		default:
			status = models.StatusToClarify
		}

		var labels []string
		if node.Labels != nil {
			for _, l := range node.Labels.Nodes {
				labels = append(labels, l.Name)
			}
		}

		assignee := ""
		if node.Assignee != nil {
			assignee = node.Assignee.Name
			if assignee == "" {
				assignee = node.Assignee.DisplayName
			}
		}

		cTime, _ := time.Parse(time.RFC3339, node.CreatedAt)
		uTime, _ := time.Parse(time.RFC3339, node.UpdatedAt)
		if cTime.IsZero() {
			cTime = time.Now()
		}
		if uTime.IsZero() {
			uTime = time.Now()
		}

		extURL := node.URL

		tasks = append(tasks, models.Task{
			ID:          node.ID,
			Key:         node.Identifier,
			Title:       node.Title,
			Description: node.Description,
			Status:      status,
			Priority:    priority,
			Labels:      labels,
			Assignee:    assignee,
			Position:    i,
			Source:      "linear",
			ExternalURL: &extURL,
			CreatedAt:   cTime,
			UpdatedAt:   uTime,
		})
	}

	return tasks, nil
}

type LinearCreateOutput struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

var validLinearLabels = map[string]string{
	"new":          "New",
	"to-clarify":   "to-clarify",
	"toclarify":    "to-clarify",
	"to_clarify":   "to-clarify",
	"clarified":    "clarified",
	"clarify":      "clarified",
	"specified":    "specified",
	"specify":      "specified",
	"to-specify":   "specified",
	"to_specify":   "specified",
	"implemented":  "Implemented",
	"implement":    "Implemented",
	"to-implement": "Implemented",
	"to_implement": "Implemented",
	"handoff":      "finished",
	"finished":     "finished",
	"reviewed":     "Reviewed",
	"review":       "Reviewed",
	"to-review":    "Reviewed",
	"to_review":    "Reviewed",
	"to-test":      "Reviewed",
	"to_test":      "Reviewed",
	"to-close":     "Reviewed",
	"to_close":     "Reviewed",
	"validate":     "validate",
	"validated":    "validated",
	"design":       "design",
	"enhancement":  "enhancement",
	"migrated":     "Migrated",
	"milestone":    "milestone",
	"improvement":  "Improvement",
	"feature":      "Feature",
	"bug":          "Bug",
}

func filterLinearLabels(labels []string) []string {
	var valid []string
	seen := make(map[string]bool)
	for _, l := range labels {
		cleaned := strings.ToLower(strings.TrimSpace(l))
		if target, ok := validLinearLabels[cleaned]; ok {
			if !seen[target] {
				seen[target] = true
				valid = append(valid, target)
			}
		}
	}
	return valid
}

func (r *Runner) CreateLinearIssue(teamKey string, title string, description string, priority models.Priority, labels []string) (*models.Task, error) {
	if teamKey == "" {
		teamKey = "FRE"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	linearPath, err := exec.LookPath("linear")
	if err != nil {
		linearPath = "/opt/homebrew/bin/linear"
	}

	priorityNum := "3"
	switch priority {
	case models.PriorityUrgent:
		priorityNum = "1"
	case models.PriorityHigh:
		priorityNum = "2"
	case models.PriorityMedium:
		priorityNum = "3"
	case models.PriorityLow:
		priorityNum = "4"
	}

	filteredLabels := filterLinearLabels(labels)

	args := []string{"issue", "create", "--team", teamKey, "--title", title, "-p", priorityNum, "--no-interactive"}
	if description != "" {
		args = append(args, "-d", description)
	}
	for _, l := range filteredLabels {
		if l != "" {
			args = append(args, "-l", l)
		}
	}

	output, err := r.runCommand(ctx, "", linearPath, args...)
	if err != nil {
		// Fallback retry without labels if a label doesn't exist on Linear workspace
		retryArgs := []string{"issue", "create", "--team", teamKey, "--title", title, "-p", priorityNum, "--no-interactive"}
		if description != "" {
			retryArgs = append(retryArgs, "-d", description)
		}
		retryOutput, retryErr := r.runCommand(ctx, "", linearPath, retryArgs...)
		if retryErr != nil {
			return nil, fmt.Errorf("linear issue create failed: %w (output: %s)", err, output)
		}
		output = retryOutput
	}

	urlRe := regexp.MustCompile(`https?://linear\.app/\S+`)
	urlMatch := urlRe.FindString(output)

	keyRe := regexp.MustCompile(`([A-Z0-9]+-\d+)`)
	keyMatch := keyRe.FindString(output)
	if keyMatch == "" {
		keyMatch = fmt.Sprintf("%s-new", teamKey)
	}

	var extURL *string
	if urlMatch != "" {
		cleanURL := strings.TrimSpace(urlMatch)
		extURL = &cleanURL
	}

	now := time.Now()
	id := uuid.New().String()

	return &models.Task{
		ID:          id,
		Key:         keyMatch,
		Title:       title,
		Description: description,
		Status:      models.StatusToClarify,
		Priority:    priority,
		Labels:      labels,
		Source:      "linear",
		ExternalURL: extURL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func mapStatusToLinearState(status models.Status) string {
	switch status {
	case models.StatusToClarify, models.StatusBacklog:
		return "Backlog"
	case models.StatusToSpecify, models.StatusSpecified:
		return "Todo"
	case models.StatusToImplement, models.StatusInProgress:
		return "In Progress"
	case models.StatusToTest, models.StatusToValidate:
		return "In Review"
	case models.StatusToClose, models.StatusDone:
		return "Done"
	default:
		return "Backlog"
	}
}

func (r *Runner) UpdateLinearIssueState(issueKey string, status models.Status) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	linearPath, err := exec.LookPath("linear")
	if err != nil {
		linearPath = "/opt/homebrew/bin/linear"
	}

	stateName := mapStatusToLinearState(status)
	output, err := r.runCommand(ctx, "", linearPath, "issue", "update", issueKey, "--state", stateName)
	if err != nil {
		log.Printf("[CLI] linear issue update %s --state '%s' failed: %v (output: %s)", issueKey, stateName, err, output)
	}
	return err
}

func (r *Runner) UpdateLinearIssue(issueKey string, title *string, description *string, priority *models.Priority, status *models.Status, labels []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	linearPath, err := exec.LookPath("linear")
	if err != nil {
		linearPath = "/opt/homebrew/bin/linear"
	}

	filteredLabels := filterLinearLabels(labels)

	buildArgs := func(withLabels bool) []string {
		args := []string{"issue", "update", issueKey}
		if title != nil && *title != "" {
			args = append(args, "--title", *title)
		}
		if description != nil {
			args = append(args, "--description", *description)
		}
		if priority != nil {
			switch *priority {
			case models.PriorityUrgent:
				args = append(args, "--priority", "1")
			case models.PriorityHigh:
				args = append(args, "--priority", "2")
			case models.PriorityMedium:
				args = append(args, "--priority", "3")
			case models.PriorityLow:
				args = append(args, "--priority", "4")
			}
		}
		if status != nil {
			args = append(args, "--state", mapStatusToLinearState(*status))
		}
		if withLabels {
			for _, l := range filteredLabels {
				if l != "" {
					args = append(args, "-l", l)
				}
			}
		}
		return args
	}

	args := buildArgs(len(filteredLabels) > 0)
	output, err := r.runCommand(ctx, "", linearPath, args...)
	if err != nil {
		log.Printf("[CLI] linear issue update %s with labels failed: %v (output: %s), retrying without labels...", issueKey, err, output)
		retryArgs := buildArgs(false)
		output, err = r.runCommand(ctx, "", linearPath, retryArgs...)
		if err != nil {
			log.Printf("[CLI] linear issue update %s without labels also failed: %v (output: %s)", issueKey, err, output)
		}
	}
	return err
}

// GitHub structures
type GithubIssueItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
}

// CleanGithubRepo extracts the owner/repo string from a Git URL or plain repo identifier.
func CleanGithubRepo(repo string) string {
	raw := strings.TrimSpace(repo)
	raw = strings.TrimSuffix(raw, ".git")
	if strings.Contains(raw, "github.com/") {
		parts := strings.Split(raw, "github.com/")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(raw, "github.com:") {
		parts := strings.Split(raw, "github.com:")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}
	if strings.Contains(raw, "/") && !strings.Contains(raw, ":") && !strings.Contains(raw, " ") {
		return raw
	}
	return raw
}

// ResolveGithubRepo attempts to determine the target repository and path dynamically.
func ResolveGithubRepo(repo string, repoPath string) (string, string) {
	if repoPath == "" {
		repoPath = "."
	}
	if repo != "" {
		return CleanGithubRepo(repo), repoPath
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "config", "--get", "remote.origin.url")
	if out, err := cmd.Output(); err == nil {
		raw := strings.TrimSpace(string(out))
		cleaned := CleanGithubRepo(raw)
		if cleaned != "" {
			return cleaned, repoPath
		}
	}
	return "", repoPath
}

func (r *Runner) SyncFromGithub(repo string, repoPath string) ([]models.Task, error) {
	repo, repoPath = ResolveGithubRepo(repo, repoPath)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ghPath, _ := FindCliTool("gh")
	if ghPath == "" {
		ghPath = "gh"
	}

	var args []string
	if repo != "" {
		args = []string{"issue", "list", "-R", repo, "--limit", "100", "--json", "number,title,labels,body,url,state,assignees"}
	} else {
		args = []string{"issue", "list", "--limit", "100", "--json", "number,title,labels,body,url,state,assignees"}
	}

	output, err := r.runCommand(ctx, repoPath, ghPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub issues: %w (output: %s)", err, output)
	}

	var items []GithubIssueItem
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub JSON: %w", err)
	}

	var tasks []models.Task
	for i, item := range items {
		var labels []string
		for _, l := range item.Labels {
			labels = append(labels, l.Name)
		}

		var status models.Status = models.StatusToClarify
		for _, l := range labels {
			low := strings.ToLower(l)
			if strings.Contains(low, "clarif") {
				status = models.StatusToSpecify
			} else if strings.Contains(low, "specif") {
				status = models.StatusToImplement
			} else if strings.Contains(low, "progress") || strings.Contains(low, "implem") {
				status = models.StatusToTest
			} else if strings.Contains(low, "valid") || strings.Contains(low, "review") || strings.Contains(low, "test") {
				status = models.StatusToClose
			}
		}
		if strings.ToUpper(item.State) == "CLOSED" {
			status = models.StatusToClose
		}

		assignee := ""
		if len(item.Assignees) > 0 {
			assignee = item.Assignees[0].Login
		}

		extURL := item.URL

		tasks = append(tasks, models.Task{
			ID:          fmt.Sprintf("gh-%d", item.Number),
			Key:         fmt.Sprintf("#%d", item.Number),
			Title:       item.Title,
			Description: item.Body,
			Status:      status,
			Priority:    models.PriorityMedium,
			Labels:      labels,
			Assignee:    assignee,
			Position:    i,
			Source:      "github",
			ExternalURL: &extURL,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return tasks, nil
}

func (r *Runner) CreateGithubIssue(repo string, repoPath string, title string, description string, labels []string) (*models.Task, error) {
	repo, repoPath = ResolveGithubRepo(repo, repoPath)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	ghPath, _ := FindCliTool("gh")
	if ghPath == "" {
		ghPath = "gh"
	}

	// Filter internal workflow labels (like "New", "untouched") that don't exist as standard labels on GitHub repos
	var filteredLabels []string
	for _, l := range labels {
		l = strings.TrimSpace(l)
		if l != "" && !strings.EqualFold(l, "new") && !strings.EqualFold(l, "untouched") {
			filteredLabels = append(filteredLabels, l)
		}
	}

	var args []string
	if repo != "" {
		args = []string{"issue", "create", "-R", repo, "--title", title}
	} else {
		args = []string{"issue", "create", "--title", title}
	}

	if description != "" {
		args = append(args, "--body", description)
	} else {
		args = append(args, "--body", "")
	}
	for _, l := range filteredLabels {
		args = append(args, "--label", l)
	}

	output, err := r.runCommand(ctx, repoPath, ghPath, args...)
	if err != nil {
		// Fallback retry without labels if a label doesn't exist on GitHub repo
		fallbackArgs := []string{"issue", "create"}
		if repo != "" {
			fallbackArgs = append(fallbackArgs, "-R", repo)
		}
		fallbackArgs = append(fallbackArgs, "--title", title)
		if description != "" {
			fallbackArgs = append(fallbackArgs, "--body", description)
		} else {
			fallbackArgs = append(fallbackArgs, "--body", "")
		}
		retryOutput, retryErr := r.runCommand(ctx, repoPath, ghPath, fallbackArgs...)
		if retryErr != nil {
			return nil, fmt.Errorf("gh issue create failed: %w (output: %s)", err, output)
		}
		output = retryOutput
	}

	// Extract issue URL and number via regex: https://github.com/owner/repo/issues/123
	re := regexp.MustCompile(`https://github\.com/[^/\s]+/[^/\s]+/issues/(\d+)`)
	matches := re.FindStringSubmatch(output)
	var issueURL string
	var issueNum string

	if len(matches) > 1 {
		issueURL = matches[0]
		issueNum = matches[1]
	} else {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		issueURL = lastLine
		parts := strings.Split(issueURL, "/")
		issueNum = parts[len(parts)-1]
	}

	key := fmt.Sprintf("#%s", issueNum)
	id := fmt.Sprintf("gh-%s", issueNum)

	now := time.Now()
	return &models.Task{
		ID:          id,
		Key:         key,
		Title:       title,
		Description: description,
		Status:      models.StatusToClarify,
		Priority:    models.PriorityMedium,
		Labels:      labels,
		Source:      "github",
		ExternalURL: &issueURL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func cleanGithubIssueNum(keyOrNumber string) (int, error) {
	s := strings.TrimSpace(keyOrNumber)
	s = strings.TrimPrefix(s, "GH-#")
	s = strings.TrimPrefix(s, "gh-")
	s = strings.TrimPrefix(s, "GH-")
	s = strings.TrimPrefix(s, "#")
	return strconv.Atoi(s)
}

func (r *Runner) UpdateGithubIssueState(repo string, repoPath string, keyOrNumber string, status models.Status) error {
	repo, repoPath = ResolveGithubRepo(repo, repoPath)

	num, err := cleanGithubIssueNum(keyOrNumber)
	if err != nil {
		return fmt.Errorf("invalid github issue number: %s", keyOrNumber)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ghPath, _ := FindCliTool("gh")
	if ghPath == "" {
		ghPath = "gh"
	}

	var args []string
	if status == models.StatusDone || status == models.StatusFinished || status == models.StatusToClose {
		if repo != "" {
			args = []string{"issue", "close", strconv.Itoa(num), "-R", repo}
		} else {
			args = []string{"issue", "close", strconv.Itoa(num)}
		}
	} else {
		if repo != "" {
			args = []string{"issue", "reopen", strconv.Itoa(num), "-R", repo}
		} else {
			args = []string{"issue", "reopen", strconv.Itoa(num)}
		}
	}

	_, err = r.runCommand(ctx, repoPath, ghPath, args...)
	return err
}

func (r *Runner) UpdateGithubIssue(repo string, repoPath string, keyOrNumber string, title *string, description *string, status *models.Status, labels []string, removedLabels []string) error {
	repo, repoPath = ResolveGithubRepo(repo, repoPath)

	num, err := cleanGithubIssueNum(keyOrNumber)
	if err != nil {
		return fmt.Errorf("invalid github issue number: %s", keyOrNumber)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ghPath, _ := FindCliTool("gh")
	if ghPath == "" {
		ghPath = "gh"
	}

	// 1. Update State (close / reopen)
	if status != nil {
		if *status == models.StatusFinished || *status == models.StatusDone || *status == models.StatusToClose {
			var args []string
			if repo != "" {
				args = []string{"issue", "close", strconv.Itoa(num), "-R", repo}
			} else {
				args = []string{"issue", "close", strconv.Itoa(num)}
			}
			output, err := r.runCommand(ctx, repoPath, ghPath, args...)
			if err != nil {
				log.Printf("[CLI] gh issue close %d failed: %v (output: %s)", num, err, output)
			} else {
				log.Printf("[CLI] Closed GitHub issue #%d in %s", num, repo)
			}
		} else {
			var args []string
			if repo != "" {
				args = []string{"issue", "reopen", strconv.Itoa(num), "-R", repo}
			} else {
				args = []string{"issue", "reopen", strconv.Itoa(num)}
			}
			output, err := r.runCommand(ctx, repoPath, ghPath, args...)
			if err != nil {
				log.Printf("[CLI] gh issue reopen %d failed: %v (output: %s)", num, err, output)
			} else {
				log.Printf("[CLI] Reopened GitHub issue #%d in %s", num, repo)
			}
		}
	}

	// 2. Query existing labels so we only remove labels that actually exist on GitHub
	existingLabelsMap := make(map[string]bool)
	var viewArgs []string
	if repo != "" {
		viewArgs = []string{"issue", "view", strconv.Itoa(num), "-R", repo, "--json", "labels"}
	} else {
		viewArgs = []string{"issue", "view", strconv.Itoa(num), "--json", "labels"}
	}
	if viewOut, err := r.runCommand(ctx, repoPath, ghPath, viewArgs...); err == nil {
		var viewRes struct {
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		}
		if err := json.Unmarshal([]byte(viewOut), &viewRes); err == nil {
			for _, l := range viewRes.Labels {
				existingLabelsMap[strings.ToLower(l.Name)] = true
			}
		}
	}

	// 3. Edit title, body & labels
	var args []string
	if repo != "" {
		args = []string{"issue", "edit", strconv.Itoa(num), "-R", repo}
	} else {
		args = []string{"issue", "edit", strconv.Itoa(num)}
	}
	needEdit := false
	if title != nil && *title != "" {
		args = append(args, "--title", *title)
		needEdit = true
	}
	if description != nil {
		args = append(args, "--body", *description)
		needEdit = true
	}

	// Handle workflow stage labels
	allStages := []string{"new", "untouched", "clarified", "specified", "implemented", "reviewed", "finished"}
	var activeStage string
	if len(labels) > 0 {
		for _, l := range labels {
			cleanL := strings.TrimPrefix(strings.ToLower(l), "#")
			for _, st := range allStages {
				if cleanL == st {
					activeStage = st
					break
				}
			}
		}

		if activeStage != "" {
			args = append(args, "--add-label", activeStage)
			for _, st := range allStages {
				if st != activeStage && existingLabelsMap[st] {
					args = append(args, "--remove-label", st)
				}
			}
			needEdit = true
		}

		// Also add other non-workflow labels
		for _, l := range labels {
			cleanL := strings.TrimPrefix(strings.ToLower(l), "#")
			isStage := false
			for _, st := range allStages {
				if cleanL == st {
					isStage = true
					break
				}
			}
			if !isStage && cleanL != "" {
				args = append(args, "--add-label", cleanL)
				needEdit = true
			}
		}
	}

	// Remove explicitly removed labels only if they exist on the issue
	for _, rl := range removedLabels {
		cleanRl := strings.TrimPrefix(strings.ToLower(rl), "#")
		if cleanRl != "" && (len(existingLabelsMap) == 0 || existingLabelsMap[cleanRl]) {
			args = append(args, "--remove-label", cleanRl)
			needEdit = true
		}
	}

	if needEdit {
		output, err := r.runCommand(ctx, repoPath, ghPath, args...)
		if err != nil {
			log.Printf("[CLI] gh issue edit #%d failed: %v (output: %s), retrying without --remove-label...", num, err, output)
			// Fallback: Retry without --remove-label flags
			var fallbackArgs []string
			if repo != "" {
				fallbackArgs = []string{"issue", "edit", strconv.Itoa(num), "-R", repo}
			} else {
				fallbackArgs = []string{"issue", "edit", strconv.Itoa(num)}
			}
			if title != nil && *title != "" {
				fallbackArgs = append(fallbackArgs, "--title", *title)
			}
			if description != nil {
				fallbackArgs = append(fallbackArgs, "--body", *description)
			}
			if activeStage != "" {
				fallbackArgs = append(fallbackArgs, "--add-label", activeStage)
			}
			if out2, err2 := r.runCommand(ctx, repoPath, ghPath, fallbackArgs...); err2 == nil {
				log.Printf("[CLI] Successfully updated GitHub issue #%d in %s (fallback)", num, repo)
			} else {
				log.Printf("[CLI] Fallback gh issue edit #%d also failed: %v (output: %s)", num, err2, out2)
				// Self-healing: if label was not found in repo, create it and retry
				if activeStage != "" && (strings.Contains(output, "not found") || strings.Contains(out2, "not found")) {
					labelColor := "1d76db"
					switch activeStage {
					case "new", "untouched":
						labelColor = "0075ca"
					case "clarified":
						labelColor = "fbca04"
					case "specified":
						labelColor = "1d76db"
					case "implemented":
						labelColor = "5319e7"
					case "reviewed":
						labelColor = "6f42c1"
					case "finished":
						labelColor = "0e8a16"
					}
					var createArgs []string
					if repo != "" {
						createArgs = []string{"label", "create", activeStage, "--color", labelColor, "-R", repo}
					} else {
						createArgs = []string{"label", "create", activeStage, "--color", labelColor}
					}
					_, _ = r.runCommand(ctx, repoPath, ghPath, createArgs...)
					if _, err3 := r.runCommand(ctx, repoPath, ghPath, fallbackArgs...); err3 == nil {
						log.Printf("[CLI] Successfully updated GitHub issue #%d in %s after creating label '%s'", num, repo, activeStage)
					} else {
						// Final fallback: update title & body only without labels
						var bodyOnlyArgs []string
						if repo != "" {
							bodyOnlyArgs = []string{"issue", "edit", strconv.Itoa(num), "-R", repo}
						} else {
							bodyOnlyArgs = []string{"issue", "edit", strconv.Itoa(num)}
						}
						if title != nil && *title != "" {
							bodyOnlyArgs = append(bodyOnlyArgs, "--title", *title)
						}
						if description != nil {
							bodyOnlyArgs = append(bodyOnlyArgs, "--body", *description)
						}
						_, _ = r.runCommand(ctx, repoPath, ghPath, bodyOnlyArgs...)
					}
				}
			}
		} else {
			log.Printf("[CLI] Successfully updated GitHub issue #%d in %s", num, repo)
		}
	}

	return nil
}

// -------------------------------------------------------------
// JIRA CLI (acli) INTEGRATION
// -------------------------------------------------------------

// jiraSearchFields is the exact set of fields acli accepts for
// 'jira workitem search --fields'. Notably 'created' and 'updated' are
// rejected by the CLI, so task timestamps fall back to the import time.
const jiraSearchFields = "key,summary,description,status,priority,assignee,labels,issuetype"

// jiraSyncedIssueTypes is the set of Jira work item types Taskacao imports as
// board cards. Epics are containers, and the other project-specific types
// (Vulnerability, Corrective Action, Technical debt…) are out of scope: they
// would flood the board without being part of the dev workflow.
var jiraSyncedIssueTypes = []string{"Task", "Story"}

// jiraSyncTimeout covers a full paginated fetch of a project's work items.
const jiraSyncTimeout = 4 * time.Minute

// jiraParentLookupWorkers bounds the concurrency of the parent enrichment pass.
// acli's 'workitem search' cannot project the parent field (its --fields
// allow-list rejects both 'parent' and any customfield), so the parent of each
// work item has to be read one by one with 'workitem view'. Each call costs
// roughly a second, hence the fan-out.
const jiraParentLookupWorkers = 8

// JiraIssueRef is a lightweight reference to another work item, used for the
// 'parent' field returned by 'acli jira workitem view'.
type JiraIssueRef struct {
	Key    string `json:"key"`
	Fields struct {
		Summary   string `json:"summary"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
	} `json:"fields"`
}

type JiraIssueField struct {
	Summary   string `json:"summary"`
	IssueType struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Parent *JiraIssueRef `json:"parent"`
	// Description is Atlassian Document Format: a nested JSON object on Jira
	// Cloud, but a plain string on older/Server payloads. Keep it raw and
	// flatten it with jiraDescriptionToText.
	Description json.RawMessage `json:"description"`
	Status      struct {
		Name           string `json:"name"`
		StatusCategory struct {
			Key string `json:"key"`
		} `json:"statusCategory"`
	} `json:"status"`
	Priority struct {
		Name string `json:"name"`
	} `json:"priority"`
	Assignee struct {
		DisplayName string            `json:"displayName"`
		AvatarUrls  map[string]string `json:"avatarUrls"`
	} `json:"assignee"`
	Labels  []string `json:"labels"`
	Created string   `json:"created"`
	Updated string   `json:"updated"`
}

type JiraIssueItem struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Self   string         `json:"self"`
	Fields JiraIssueField `json:"fields"`
}

type JiraQueryResponse struct {
	Issues []JiraIssueItem `json:"issues"`
}

// jiraDescriptionToText flattens a Jira description into plain text. It accepts
// both a JSON string and an Atlassian Document Format tree, walking the tree to
// collect text nodes and inserting line breaks at block boundaries.
func jiraDescriptionToText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Plain string payload (Jira Server, or an already-flattened value).
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString
	}

	var node interface{}
	if err := json.Unmarshal(raw, &node); err != nil {
		return ""
	}

	var b strings.Builder
	var walk func(n interface{})
	walk = func(n interface{}) {
		switch v := n.(type) {
		case map[string]interface{}:
			nodeType, _ := v["type"].(string)
			switch nodeType {
			case "hardBreak":
				b.WriteString("\n")
			case "text":
				if txt, ok := v["text"].(string); ok {
					b.WriteString(txt)
				}
			}
			if content, ok := v["content"].([]interface{}); ok {
				for _, child := range content {
					walk(child)
				}
			}
			// Close block-level nodes with a newline so paragraphs, headings and
			// list items do not run together.
			switch nodeType {
			case "paragraph", "heading", "listItem", "blockquote", "codeBlock", "rule", "tableRow":
				b.WriteString("\n")
			}
		case []interface{}:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(node)

	return strings.TrimSpace(b.String())
}

// parseJiraSearchOutput accepts either a bare array of work items (acli
// 'search --json') or a REST-shaped {"issues": [...]} envelope.
func parseJiraSearchOutput(output string) ([]JiraIssueItem, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, fmt.Errorf("réponse vide de acli")
	}

	var items []JiraIssueItem
	if err := json.Unmarshal([]byte(trimmed), &items); err == nil {
		return items, nil
	}

	var resp JiraQueryResponse
	if err := json.Unmarshal([]byte(trimmed), &resp); err == nil {
		return resp.Issues, nil
	}

	preview := trimmed
	if len(preview) > 300 {
		preview = preview[:300] + "…"
	}
	return nil, fmt.Errorf("réponse JSON Jira illisible: %s", preview)
}

// SyncFromJira reads a Jira project. Two sources, same conversion: the REST API
// when credentials are configured, which returns the parent, Sprint and Team in
// the same pass, and acli otherwise, which needs two extra passes because its
// --fields rejects those.
func (r *Runner) SyncFromJira(settings *models.Settings, projectKey string, repoPath string, trackerUrl string) ([]models.Task, error) {
	if client := NewJiraRESTClient(settings, trackerUrl); client != nil {
		tasks, err := r.syncFromJiraREST(client, projectKey, trackerUrl)
		if err == nil {
			return tasks, nil
		}
		// A REST failure must not lose the board: fall back on acli, which is
		// the path that worked before credentials existed.
		log.Printf("[SyncFromJira] REST read failed (%v), falling back on acli", err)
	}
	return r.syncFromJiraCLI(projectKey, repoPath, trackerUrl)
}

// syncFromJiraREST reads the project over the REST API, including the fields
// acli cannot project.
func (r *Runner) syncFromJiraREST(client *JiraRESTClient, projectKey string, trackerUrl string) ([]models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), jiraRESTTimeout)
	defer cancel()

	result, err := client.SearchProjectIssues(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("aucun ticket renvoyé par l'API Jira pour %s", projectKey)
	}

	tasks := r.jiraItemsToTasks(result.Items, trackerUrl)
	for i := range tasks {
		// The parent comes straight from the payload here, so the epic-walking
		// pass the acli path needs is not run at all.
		if item := findJiraItem(result.Items, tasks[i].Key); item != nil && item.Fields.Parent != nil {
			tasks[i].ParentKey = item.Fields.Parent.Key
			tasks[i].ParentTitle = item.Fields.Parent.Fields.Summary
			tasks[i].ParentType = item.Fields.Parent.Fields.IssueType.Name
		}
		if values, ok := result.Fields[tasks[i].Key]; ok {
			tasks[i].Sprint = values.Sprint
			tasks[i].Team = values.Team
		}
	}
	return tasks, nil
}

func findJiraItem(items []JiraIssueItem, key string) *JiraIssueItem {
	for i := range items {
		if items[i].Key == key {
			return &items[i]
		}
	}
	return nil
}

// jiraStatusIsClosed reports whether a Jira status name means the work item has
// left the board. Abandoned statuses count as closed just like "Done": WONTDO,
// Cancelled or Rejected tickets are not work waiting to be done. Recognising
// them by name matters because a workflow may leave such a status in Jira's
// "In Progress" (indeterminate) category — the SFE board does exactly that, and
// its 41 WONTDO tickets were showing up in the daily digest's "À traiter
// aujourd'hui" section.
func jiraStatusIsClosed(lowerStatusName string) bool {
	closedMarkers := []string{
		"done", "close", "resolved", "finish", "complete",
		"wontdo", "won't do", "wont do", "won t do", "cancel",
		"abandon", "annul", "reject", "rejet", "duplicate", "doublon",
		"obsolete", "invalid",
	}
	for _, marker := range closedMarkers {
		if strings.Contains(lowerStatusName, marker) {
			return true
		}
	}
	return false
}

// jiraItemsToTasks converts Jira work items into board cards. Shared by the
// REST and the acli read paths, which return the same payload shape.
func (r *Runner) jiraItemsToTasks(items []JiraIssueItem, trackerUrl string) []models.Task {
	var tasks []models.Task
	for i, item := range items {
		// Second line of defence: the legacy 'workitem list' fallback above
		// cannot filter by type, so enforce the allow-list here as well. An
		// empty type (older payloads) is kept rather than silently dropped.
		itemType := strings.TrimSpace(item.Fields.IssueType.Name)
		if itemType != "" && !jiraTypeIsSynced(itemType) {
			continue
		}

		var priority models.Priority
		pName := strings.ToLower(item.Fields.Priority.Name)
		switch {
		case strings.Contains(pName, "urgent") || strings.Contains(pName, "blocker") || strings.Contains(pName, "highest") || strings.Contains(pName, "critical"):
			priority = models.PriorityUrgent
		case strings.Contains(pName, "high") || strings.Contains(pName, "major"):
			priority = models.PriorityHigh
		case strings.Contains(pName, "medium") || strings.Contains(pName, "normal"):
			priority = models.PriorityMedium
		default:
			priority = models.PriorityLow
		}

		var status models.Status = models.StatusToClarify
		stName := strings.ToLower(item.Fields.Status.Name)
		switch {
		// Closed work is tested first, because a status such as "WONTDO" or
		// "Closed - Duplicate" is work that left the board and must not fall
		// through to the status-category fallback below.
		case jiraStatusIsClosed(stName):
			status = models.StatusFinished
		case strings.Contains(stName, "clarif") || strings.Contains(stName, "triage") || strings.Contains(stName, "backlog") || strings.Contains(stName, "open"):
			status = models.StatusToClarify
		case strings.Contains(stName, "specif") || strings.Contains(stName, "to do") || strings.Contains(stName, "todo") || strings.Contains(stName, "selected"):
			status = models.StatusToSpecify
		case strings.Contains(stName, "progress") || strings.Contains(stName, "in dev") || strings.Contains(stName, "implement"):
			status = models.StatusToImplement
		case strings.Contains(stName, "test") || strings.Contains(stName, "qa") || strings.Contains(stName, "validation") || strings.Contains(stName, "review") || strings.Contains(stName, "pr"):
			status = models.StatusToTest
		default:
			// The workflow uses a status name Taskacao does not recognise; fall
			// back to Jira's own status category, which every workflow sets.
			switch item.Fields.Status.StatusCategory.Key {
			case "indeterminate":
				status = models.StatusToImplement
			case "done":
				status = models.StatusFinished
			default:
				status = models.StatusToClarify
			}
		}

		var extURL *string
		if trackerUrl != "" {
			u := fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(trackerUrl, "/"), item.Key)
			extURL = &u
		}

		avatar := ""
		if item.Fields.Assignee.AvatarUrls != nil {
			avatar = item.Fields.Assignee.AvatarUrls["48x48"]
			if avatar == "" {
				avatar = item.Fields.Assignee.AvatarUrls["32x32"]
			}
		}

		tasks = append(tasks, models.Task{
			ID:             "jira-" + item.Key,
			Key:            item.Key,
			Title:          item.Fields.Summary,
			Description:    jiraDescriptionToText(item.Fields.Description),
			Status:         status,
			Priority:       priority,
			Labels:         item.Fields.Labels,
			Assignee:       item.Fields.Assignee.DisplayName,
			AssigneeAvatar: avatar,
			Position:       i + 1,
			Source:         "jira",
			TrackerStatus:  strings.TrimSpace(item.Fields.Status.Name),
			ExternalURL:    extURL,
			IssueType:      itemType,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
	}

	return tasks
}

func (r *Runner) syncFromJiraCLI(projectKey string, repoPath string, trackerUrl string) ([]models.Task, error) {
	// A full paginated fetch of a large project (1300+ work items) takes about
	// 40 seconds, so the old 20s budget silently truncated the import.
	ctx, cancel := context.WithTimeout(context.Background(), jiraSyncTimeout)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante: renseignez 'Projet Jira' dans la configuration du projet")
	}

	// Only Task and Story are imported. Epics are containers reattached to their
	// children as a property by enrichJiraParents below; the remaining types are
	// deliberately left out of the board.
	//
	// acli queries work items through JQL. 'search' is the current command;
	// older builds exposed 'list --project', kept here as a fallback.
	quoted := make([]string, 0, len(jiraSyncedIssueTypes))
	for _, t := range jiraSyncedIssueTypes {
		quoted = append(quoted, fmt.Sprintf("%q", t))
	}
	jql := fmt.Sprintf("project = %s AND issuetype IN (%s) ORDER BY updated DESC",
		projectKey, strings.Join(quoted, ", "))
	// --paginate walks every page: without it acli returns only the first page
	// and the board silently stops at 100 tickets.
	attempts := [][]string{
		{"jira", "workitem", "search", "--jql", jql, "--fields", jiraSearchFields, "--paginate", "--json"},
		{"jira", "workitem", "search", "--jql", jql, "--paginate", "--json"},
		{"jira", "workitem", "list", "--project", projectKey, "--output", "json"},
	}

	var output string
	var lastErr error
	for _, args := range attempts {
		out, err := r.runCommand(ctx, repoPath, acliPath, args...)
		if err == nil {
			output = out
			lastErr = nil
			break
		}
		lastErr = err
		output = out
	}
	if lastErr != nil {
		return nil, fmt.Errorf("interrogation de Jira via acli impossible: %w", lastErr)
	}

	items, err := parseJiraSearchOutput(output)
	if err != nil {
		return nil, err
	}

	tasks := r.jiraItemsToTasks(items, trackerUrl)

	// Attach the parent (epic, or parent story for a sub-task) as a property of
	// each task. Best-effort: a failure here leaves tasks without a parent
	// rather than failing the whole sync.
	r.enrichJiraParents(projectKey, tasks, repoPath)

	// Sprint, on this path, is reconstructed from the project's scrum boards:
	// acli's search cannot project the field, but its agile commands list the
	// work items of each sprint. Team has no such detour and stays empty.
	if sprintByKey, notes, err := r.FetchSprintsViaCLI(projectKey, repoPath); err == nil {
		for i := range tasks {
			if name, ok := sprintByKey[tasks[i].Key]; ok {
				tasks[i].Sprint = name
			}
		}
		for _, note := range notes {
			log.Printf("[SyncFromJira] sprint via acli: %s", note)
		}
	} else {
		log.Printf("[SyncFromJira] sprint via acli indisponible: %v", err)
	}

	return tasks, nil
}

// jiraTypeIsSynced reports whether a Jira work item type is imported as a card.
func jiraTypeIsSynced(issueType string) bool {
	for _, t := range jiraSyncedIssueTypes {
		if strings.EqualFold(strings.TrimSpace(issueType), t) {
			return true
		}
	}
	return false
}

// enrichJiraParents fills ParentKey / ParentTitle / ParentType on Jira tasks in
// place, by walking the project's epics and asking Jira for each epic's
// children.
//
// The obvious approach — one 'acli jira workitem view <KEY>' per task — is not
// viable at scale: acli's search command cannot project the parent field (its
// --fields allow-list rejects 'parent' and every customfield), and a project
// with 1300 work items would mean 1300 CLI calls. Epics are one to two orders
// of magnitude fewer than their children (140 epics for 1321 items on PE), so
// the mapping is built from the epic side instead: one query for the epic list,
// then one 'parent = <EPIC>' query per epic, run concurrently.
//
// Known limitation: this resolves epic parents only. A task that hangs under a
// parent *story* rather than an epic keeps an empty parent, because covering
// that case would put us back to one call per task.
func (r *Runner) enrichJiraParents(projectKey string, tasks []models.Task, repoPath string) {
	if len(tasks) == 0 || strings.TrimSpace(projectKey) == "" {
		return
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	ctx, cancel := context.WithTimeout(context.Background(), jiraSyncTimeout)
	defer cancel()

	// 1. The project's epics, with their titles.
	epicsOut, err := r.runCommand(ctx, repoPath, acliPath,
		"jira", "workitem", "search",
		"--jql", fmt.Sprintf("project = %s AND issuetype = Epic", projectKey),
		"--fields", "key,summary", "--paginate", "--json")
	if err != nil {
		log.Printf("[CLI] Jira parent enrichment skipped: epic list unavailable: %v", err)
		return
	}
	epics, err := parseJiraSearchOutput(epicsOut)
	if err != nil || len(epics) == 0 {
		log.Printf("[CLI] Jira parent enrichment: no epic found for project %s", projectKey)
		return
	}

	// 2. Children of each epic, concurrently.
	type link struct {
		childKey  string
		epicKey   string
		epicTitle string
	}

	jobs := make(chan JiraIssueItem)
	results := make(chan link, len(tasks)*2)
	var wg sync.WaitGroup

	for w := 0; w < jiraParentLookupWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for epic := range jobs {
				if ctx.Err() != nil {
					return
				}
				// acli returns an array of nulls when --fields asks for "key"
				// alone; a second field makes it emit real items.
				out, cErr := r.runCommand(ctx, repoPath, acliPath,
					"jira", "workitem", "search",
					"--jql", fmt.Sprintf("parent = %s", epic.Key),
					"--fields", "key,summary", "--paginate", "--json")
				if cErr != nil {
					continue
				}
				children, pErr := parseJiraSearchOutput(out)
				if pErr != nil {
					continue
				}
				for _, c := range children {
					if c.Key == "" {
						continue
					}
					results <- link{childKey: c.Key, epicKey: epic.Key, epicTitle: epic.Fields.Summary}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, e := range epics {
			if e.Key == "" {
				continue
			}
			select {
			case jobs <- e:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	close(results)

	byChild := make(map[string]link, len(tasks))
	for l := range results {
		byChild[l.childKey] = l
	}

	// 3. Apply the mapping to the imported tasks.
	found := 0
	for i := range tasks {
		if l, ok := byChild[tasks[i].Key]; ok {
			tasks[i].ParentKey = l.epicKey
			tasks[i].ParentTitle = l.epicTitle
			tasks[i].ParentType = "Epic"
			found++
		}
	}

	log.Printf("[CLI] Jira parent enrichment: %d/%d work items linked to an epic (%d epics scanned)",
		found, len(tasks), len(epics))
}

// jiraKeyPattern matches a work item key such as PE-1234 in acli's plain output.
var jiraKeyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-[0-9]+`)

// CreateJiraEpic creates an epic, the container a roadmap needs before stories
// can be moved into it. Distinct from CreateJiraChildIssue: an epic has no
// parent, and its type must be Epic.
func (r *Runner) CreateJiraEpic(projectKey string, repoPath string, title string) (string, error) {
	return r.CreateJiraEpicWith(nil, "", projectKey, repoPath, title)
}

// CreateJiraEpicWith creates an epic through the REST API when a token is
// available, and falls back to acli otherwise.
func (r *Runner) CreateJiraEpicWith(settings *models.Settings, trackerURL, projectKey, repoPath, title string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if client := NewJiraRESTClient(settings, trackerURL); client != nil {
		key, err := client.CreateIssue(ctx, projectKey, "Epic", title, "")
		if err == nil {
			return key, nil
		}
		log.Printf("[jira] création REST de l'épic refusée (%v), repli sur acli", err)
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	title = strings.TrimSpace(title)
	if projectKey == "" {
		return "", fmt.Errorf("clé de projet Jira manquante")
	}
	if title == "" {
		return "", fmt.Errorf("intitulé de l'épic manquant")
	}

	out, err := r.runCommand(ctx, repoPath, acliPath,
		"jira", "workitem", "create", "--project", projectKey, "--type", "Epic", "--summary", title, "--json")
	if err != nil {
		return "", fmt.Errorf("création de l'épic impossible: %w", err)
	}
	if failure := jiraCLIFailure(out); failure != "" {
		return "", fmt.Errorf("création refusée: %s", failure)
	}

	var created struct {
		Key string `json:"key"`
	}
	_ = decodeJSONDocuments(out, func(dec *json.Decoder) error {
		if created.Key != "" {
			return io.EOF
		}
		return dec.Decode(&created)
	})
	if created.Key == "" {
		if m := jiraKeyPattern.FindString(out); m != "" {
			return m, nil
		}
		return "", fmt.Errorf("clé de l'épic créé introuvable dans la réponse d'acli")
	}
	return created.Key, nil
}

// CreateJiraChildIssue creates a work item of a given type under a parent, which
// is how a line of epic shaping becomes a real story. Distinct from
// CreateJiraIssue: that one creates a standalone Task, with no parent and no
// choice of type.
func (r *Runner) CreateJiraChildIssue(projectKey string, repoPath string, parentKey string, issueType string, title string, description string, labels []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	parentKey = strings.ToUpper(strings.TrimSpace(parentKey))
	title = strings.TrimSpace(title)
	if projectKey == "" {
		return "", fmt.Errorf("clé de projet Jira manquante")
	}
	if parentKey == "" {
		return "", fmt.Errorf("épic parent manquant")
	}
	if title == "" {
		return "", fmt.Errorf("intitulé de la story manquant")
	}
	if strings.TrimSpace(issueType) == "" {
		issueType = "Story"
	}

	args := []string{
		"jira", "workitem", "create",
		"--project", projectKey,
		"--type", issueType,
		"--parent", parentKey,
		"--summary", title,
		"--json",
	}
	if strings.TrimSpace(description) != "" {
		args = append(args, "--description", description)
	}
	if jiraLabels := filterJiraLabels(labels); len(jiraLabels) > 0 {
		args = append(args, "--label", strings.Join(jiraLabels, ","))
	}

	out, err := r.runCommand(ctx, repoPath, acliPath, args...)
	if err != nil {
		return "", fmt.Errorf("création de la story sous %s impossible: %w", parentKey, err)
	}
	// acli signale un refus sans code de sortie non nul : on lit sa sortie.
	if failure := jiraCLIFailure(out); failure != "" {
		return "", fmt.Errorf("création refusée: %s", failure)
	}

	var created struct {
		Key string `json:"key"`
	}
	// La sortie peut porter un bandeau avant le JSON : on décode le premier
	// document, comme pour les autres commandes acli.
	decodeErr := decodeJSONDocuments(out, func(dec *json.Decoder) error {
		if created.Key != "" {
			return io.EOF
		}
		return dec.Decode(&created)
	})
	if decodeErr != nil || created.Key == "" {
		// Certaines versions n'impriment que « ✓ … PROJ-123 créé ».
		if m := jiraKeyPattern.FindString(out); m != "" {
			return m, nil
		}
		return "", fmt.Errorf("clé du ticket créé introuvable dans la réponse d'acli")
	}
	return created.Key, nil
}

func (r *Runner) CreateJiraIssue(projectKey string, repoPath string, trackerUrl string, title string, description string, priority models.Priority, labels []string) (*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante: renseignez 'Projet Jira' dans la configuration du projet")
	}

	// acli rejects labels containing whitespace, so sanitize them first. The
	// --label flag takes one comma-separated value, not a repeated flag.
	jiraLabels := filterJiraLabels(labels)

	// acli requires --type on creation and, unlike Linear or GitHub, exposes no
	// --priority flag on create: the priority is applied by the caller through
	// a follow-up edit where supported.
	base := []string{
		"jira", "workitem", "create",
		"--project", projectKey,
		"--type", "Task",
		"--summary", title,
		"--description", description,
		"--json",
	}
	withLabels := append([]string{}, base...)
	if len(jiraLabels) > 0 {
		withLabels = append(withLabels, "--label", strings.Join(jiraLabels, ","))
	}

	attempts := [][]string{
		withLabels,
		base,
	}

	var output string
	var err error
	for _, args := range attempts {
		output, err = r.runCommand(ctx, repoPath, acliPath, args...)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("création du ticket Jira impossible: %w", err)
	}

	var created struct {
		Key string `json:"key"`
		ID  string `json:"id"`
	}
	_ = json.Unmarshal([]byte(output), &created)
	key := created.Key
	if key == "" {
		re := regexp.MustCompile(`[A-Z][A-Z0-9_]+-[0-9]+`)
		key = re.FindString(output)
		if key == "" {
			key = fmt.Sprintf("%s-NEW", projectKey)
		}
	}

	var extURL *string
	if trackerUrl != "" {
		u := fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(trackerUrl, "/"), key)
		extURL = &u
	}

	return &models.Task{
		ID:          "jira-" + key,
		Key:         key,
		Title:       title,
		Description: description,
		Status:      models.StatusToClarify,
		Priority:    priority,
		Labels:      labels,
		Source:      "jira",
		ExternalURL: extURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// filterJiraLabels keeps only labels Jira accepts: no whitespace, no leading '#'.
func filterJiraLabels(labels []string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, l := range labels {
		clean := strings.TrimSpace(strings.TrimPrefix(l, "#"))
		if clean == "" {
			continue
		}
		clean = strings.ReplaceAll(clean, " ", "-")
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

// Note on priority: acli exposes no --priority flag on either
// 'jira workitem create' or 'jira workitem edit', so Taskacao reads a work
// item's priority from Jira but cannot write it back. Priority changes made on
// the board stay local for Jira-tracked tasks.

// mapStatusToJiraState translates a Taskacao workflow stage into the Jira
// status name used by the default software-project workflow.
func mapStatusToJiraState(status models.Status) string {
	switch status {
	case models.StatusToClarify, models.StatusBacklog, models.StatusToSpecify:
		return "To Do"
	case models.StatusToImplement, models.StatusInProgress, models.StatusSpecified:
		return "In Progress"
	case models.StatusToTest, models.StatusToValidate:
		return "In Review"
	case models.StatusToClose, models.StatusFinished, models.StatusDone:
		return "Done"
	default:
		return "In Progress"
	}
}

// TransitionJiraIssueToStatus moves a work item to a status named as the tracker
// spells it.
//
// Two traps here, both learned the hard way. acli takes the *transition* name,
// not the target status name — a workflow exposing "Close Issue" towards the
// status "Closed" refuses --status "Closed". And acli exits 0 even when it
// refuses, printing "✗ Failure: … No allowed transitions found", so the exit
// code cannot be trusted.
//
// So the REST API does it when credentials exist: it lists the transitions,
// picks the one whose target matches, and posts its id. acli remains the
// fallback, with its output inspected rather than its exit code.
func (r *Runner) TransitionJiraIssueToStatus(settings *models.Settings, trackerURL string, issueKey string, statusName string, repoPath string) error {
	issueKey = strings.TrimSpace(issueKey)
	statusName = strings.TrimSpace(statusName)
	if issueKey == "" || statusName == "" {
		return fmt.Errorf("clé de ticket ou statut cible manquant")
	}

	if client := NewJiraRESTClient(settings, trackerURL); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		return client.TransitionToStatus(ctx, issueKey, statusName)
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.runCommand(ctx, repoPath, acliPath, "jira", "workitem", "transition",
		"--key", issueKey, "--status", statusName, "--yes")
	if err != nil {
		return fmt.Errorf("transition de %s vers '%s' impossible: %w", issueKey, statusName, err)
	}
	if failure := jiraCLIFailure(out); failure != "" {
		return fmt.Errorf("transition de %s vers '%s' refusée: %s", issueKey, statusName, failure)
	}
	return nil
}

// jiraCLIFailure extracts the failure line acli prints while still exiting 0.
func jiraCLIFailure(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "✗") || strings.Contains(trimmed, "can't be transitioned") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "✗"))
		}
	}
	return ""
}

func (r *Runner) UpdateJiraIssueState(issueKey string, status models.Status, repoPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	stateName := mapStatusToJiraState(status)

	// acli addresses work items with --key (not positionally) and names the
	// target status --status. --yes skips the interactive confirmation.
	args := []string{"jira", "workitem", "transition", "--key", issueKey, "--status", stateName, "--yes"}
	if out, err := r.runCommand(ctx, repoPath, acliPath, args...); err == nil {
		// acli sort en zéro même quand il refuse : c'est sa sortie qui tranche.
		if failure := jiraCLIFailure(out); failure == "" {
			return nil
		}
	}

	// Older acli builds took the key positionally and called the flag --state.
	fallback := []string{"jira", "workitem", "transition", issueKey, "--state", stateName}
	out, err := r.runCommand(ctx, repoPath, acliPath, fallback...)
	if err != nil {
		return fmt.Errorf("transition de %s vers '%s' impossible: %w", issueKey, stateName, err)
	}
	if failure := jiraCLIFailure(out); failure != "" {
		return fmt.Errorf("transition de %s vers '%s' refusée: %s", issueKey, stateName, failure)
	}
	return nil
}

// UpdateJiraIssue pushes the full editable payload (summary, description,
// priority, labels) to Jira and then transitions the work item if a status is
// supplied. Every field is optional; nil means "leave untouched".
func (r *Runner) UpdateJiraIssue(issueKey string, repoPath string, title *string, description *string, priority *models.Priority, status *models.Status, labels []string, removedLabels []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	// acli 'workitem edit' exposes --summary, --description, --labels and
	// --remove-labels (comma-separated values, not repeated flags), and has no
	// --priority flag: the priority is intentionally left untouched here.
	var fieldArgs []string
	if title != nil && strings.TrimSpace(*title) != "" {
		fieldArgs = append(fieldArgs, "--summary", *title)
	}
	if description != nil {
		fieldArgs = append(fieldArgs, "--description", *description)
	}
	if added := filterJiraLabels(labels); len(added) > 0 {
		fieldArgs = append(fieldArgs, "--labels", strings.Join(added, ","))
	}
	if removed := filterJiraLabels(removedLabels); len(removed) > 0 {
		fieldArgs = append(fieldArgs, "--remove-labels", strings.Join(removed, ","))
	}

	var firstErr error
	if len(fieldArgs) > 0 {
		args := append([]string{"jira", "workitem", "edit", "--key", issueKey}, fieldArgs...)
		args = append(args, "--yes")
		out, err := r.runCommand(ctx, repoPath, acliPath, args...)
		if err == nil {
			// Même piège que la transition : acli signale un refus sans code de
			// sortie non nul.
			if failure := jiraCLIFailure(out); failure != "" {
				err = fmt.Errorf("%s", failure)
			}
		}
		if err != nil {
			// Retry without the label mutations: a label the Jira project does
			// not allow makes the whole edit fail, and losing the label update
			// is preferable to losing the summary and description update too.
			var narrowed []string
			for i := 0; i+1 < len(fieldArgs); i += 2 {
				if fieldArgs[i] == "--labels" || fieldArgs[i] == "--remove-labels" {
					continue
				}
				narrowed = append(narrowed, fieldArgs[i], fieldArgs[i+1])
			}
			if len(narrowed) > 0 {
				retry := append([]string{"jira", "workitem", "edit", "--key", issueKey}, narrowed...)
				retry = append(retry, "--yes")
				out2, err2 := r.runCommand(ctx, repoPath, acliPath, retry...)
				if err2 == nil {
					if failure := jiraCLIFailure(out2); failure != "" {
						err2 = fmt.Errorf("%s", failure)
					}
				}
				if err2 != nil {
					firstErr = fmt.Errorf("acli jira workitem edit %s a échoué: %w", issueKey, err2)
				}
			} else {
				firstErr = fmt.Errorf("acli jira workitem edit %s a échoué: %w", issueKey, err)
			}
		}
	}

	if status != nil {
		if err := r.UpdateJiraIssueState(issueKey, *status, repoPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (r *Runner) PostJiraComment(issueKey string, body string, repoPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	// 'comment' is a command group: the leaf that posts a comment is 'create',
	// and the work item is addressed with --key.
	args := []string{"jira", "workitem", "comment", "create", "--key", issueKey, "--body", body}
	if _, err := r.runCommand(ctx, repoPath, acliPath, args...); err == nil {
		return nil
	}

	// Older acli builds accepted 'comment' directly with a positional key.
	fallback := []string{"jira", "workitem", "comment", issueKey, "--body", body}
	_, err := r.runCommand(ctx, repoPath, acliPath, fallback...)
	if err != nil {
		return fmt.Errorf("publication du commentaire sur %s impossible: %w", issueKey, err)
	}
	return nil
}

func (r *Runner) AddIssueComment(source string, repo string, repoPath string, key string, body string) error {
	if body == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if source == "linear" {
		linPath, _ := FindCliTool("linear")
		if linPath == "" {
			linPath = "linear"
		}
		output, err := r.runCommand(ctx, repoPath, linPath, "issue", "comment", "add", key, "--body", body)
		if err != nil {
			log.Printf("[CLI] linear issue comment add %s failed: %v (output: %s)", key, err, output)
		}
		return err

	} else if source == "github" || strings.HasPrefix(key, "#") || strings.HasPrefix(key, "GH-#") || strings.HasPrefix(key, "gh-") {
		repo, repoPath = ResolveGithubRepo(repo, repoPath)

		ghPath, _ := FindCliTool("gh")
		if ghPath == "" {
			ghPath = "gh"
		}

		num, err := cleanGithubIssueNum(key)
		if err != nil {
			return fmt.Errorf("invalid github issue number: %s", key)
		}
		cleanNum := strconv.Itoa(num)

		tmpFile, tmpErr := os.CreateTemp("", "gh-comment-*.md")
		if tmpErr == nil {
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)
			_, _ = tmpFile.WriteString(body)
			_ = tmpFile.Close()

			var args []string
			if repo != "" {
				args = []string{"issue", "comment", cleanNum, "-R", repo, "--body-file", tmpPath}
			} else {
				args = []string{"issue", "comment", cleanNum, "--body-file", tmpPath}
			}
			output, err := r.runCommand(ctx, repoPath, ghPath, args...)
			if err == nil {
				log.Printf("[CLI] Comment added to GitHub %s: %s", key, strings.TrimSpace(output))
				return nil
			}
			log.Printf("[CLI] gh issue comment --body-file failed: %v (output: %s), trying direct --body flag...", err, output)
		}

		var args []string
		if repo != "" {
			args = []string{"issue", "comment", cleanNum, "-R", repo, "--body", body}
		} else {
			args = []string{"issue", "comment", cleanNum, "--body", body}
		}
		output, err := r.runCommand(ctx, repoPath, ghPath, args...)
		if err != nil {
			log.Printf("[CLI] gh issue comment %s failed: %v (output: %s)", key, err, output)
		}
		return err
	} else if source == "jira" {
		return r.PostJiraComment(key, body, repoPath)
	}
	return nil
}

// installedSkillPath returns the SKILL.md of a workflow skill inside a checkout,
// whichever agent directory holds it. Empty when the skill is not installed.
func installedSkillPath(repoDir, skillID string) string {
	dirName := models.SkillDirNames[skillID]
	if dirName == "" || repoDir == "" {
		return ""
	}

	// La commande slash d'abord : c'est elle qui rend « /clarify-issue »
	// invocable. Une skill seule est choisie par le modèle, jamais appelée par
	// son nom, et le prompt se contentait alors d'être recopié.
	cmdPath := filepath.Join(repoDir, ".claude", "commands", dirName+".md")
	if fi, err := os.Stat(cmdPath); err == nil && !fi.IsDir() {
		return cmdPath
	}

	for _, agent := range models.SkillAgentDirs {
		var p string
		if agent == "" {
			p = filepath.Join(repoDir, ".skills", dirName, "SKILL.md")
		} else {
			p = filepath.Join(repoDir, agent, "skills", dirName, "SKILL.md")
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// skillSlashPrompt is the unified invocation: the agent runs the skill installed
// in the repository, and the ticket context follows. The instructions then live
// in one place, the SKILL.md rendered from the in-app editor, instead of being
// written twice: once in a file, once in a prompt here.
func skillSlashPrompt(skillID string) string {
	dirName := models.SkillDirNames[skillID]
	return "/" + dirName + ` {issueKey}

Contexte du ticket
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git : {branchName}
Dossier du projet : {repoPath}
Tracker : {tracker}`
}

// AIInvocation is everything needed to run one workflow step: where, with which
// engine, and with which prompt. Splitting it out of RunAI is what lets the same
// invocation run either headless or inside a PTY session the user can open.
type AIInvocation struct {
	RepoDir  string
	Provider string
	Template string // modèle de commande du projet, placeholders déjà résolus
	Prompt   string
	Steps    []string
}

// RunAI keeps the headless path: pipes, no terminal to attach to. It stays the
// fallback for when no session is available.
func (r *Runner) RunAI(settings *models.Settings, skillID string, task *models.Task, customPrompt string) (string, []string, error) {
	inv, err := r.PrepareAI(settings, skillID, task, customPrompt)
	if err != nil {
		return "", nil, err
	}
	steps := inv.Steps

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, execSteps, execErr := r.execAgentCommand(ctx, inv.RepoDir, inv.Provider, inv.Template, inv.Prompt)
	steps = append(steps, execSteps...)

	if execErr != nil {
		steps = append(steps, fmt.Sprintf("⚠️ Erreur d'exécution : %v", execErr))
		return fmt.Sprintf("### ⚠️ Erreur lors de l'exécution de la commande IA (%s)\n\n```text\n%s\n```\n\n*Vérifiez que le binaire '%s' est bien accessible et authentifié.*", inv.Provider, output, inv.Provider), steps, nil
	}

	steps = append(steps, "✅ Réponse générée par le modèle IA avec succès")
	return output, steps, nil
}

// PrepareAI resolves the working directory, the engine and the prompt of one
// workflow step, without running anything.
func (r *Runner) PrepareAI(settings *models.Settings, skillID string, task *models.Task, customPrompt string) (*AIInvocation, error) {
	repoDir := ""
	if task != nil && task.WorktreePath != nil && *task.WorktreePath != "" {
		repoDir = *task.WorktreePath
	} else if settings != nil && settings.RepoPath != "" {
		repoDir = strings.TrimSpace(settings.RepoPath)
	}
	if repoDir == "" {
		repoDir = "."
	}
	repoDir = filepath.Clean(repoDir)
	if abs, err := filepath.Abs(repoDir); err == nil {
		repoDir = abs
	}

	var steps []string
	if stat, err := os.Stat(repoDir); err != nil || !stat.IsDir() {
		cwd, _ := os.Getwd()
		steps = append(steps, fmt.Sprintf("⚠️ Project directory '%s' not found, falling back to: %s", repoDir, cwd))
		repoDir = cwd
	} else {
		steps = append(steps, fmt.Sprintf("📁 Project working directory (CWD): %s", repoDir))
	}

	trackerName := "github"
	if task != nil && task.Source != "" {
		trackerName = strings.ToLower(task.Source)
	} else if settings != nil && settings.IssueTracker != "" {
		trackerName = strings.ToLower(settings.IssueTracker)
	}

	repoName := filepath.Base(repoDir)
	if settings != nil && settings.GithubRepo != "" {
		repoName = settings.GithubRepo
	}

	// La skill installée dans le dépôt fait référence quand elle est là : c'est
	// le fichier que l'éditeur de Taskacao produit. Les prompts ci-dessous ne
	// servent plus que de filet quand rien n'est installé.
	installedSkill := installedSkillPath(repoDir, skillID)
	if installedSkill != "" {
		steps = append(steps, fmt.Sprintf("📄 Skill du dépôt utilisée : %s", installedSkill))
	}

	var promptTemplate string
	switch skillID {
	case "clarify":
		promptTemplate = settings.PromptClarify
		if promptTemplate == "" {
			promptTemplate = "/clarify-issue {issueKey} tracked on {tracker} in {repo}"
		}
	case "specify":
		promptTemplate = settings.PromptSpecify
		if promptTemplate == "" {
			if NormalizeSpecFramework(settings.SpecFramework) == "openspec" {
				promptTemplate = `Tu es le Lead Architecte pour Taskacao. Rédige une proposition de changement OpenSpec complète pour la tâche :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git cible : {branchName}
Dossier du projet : {repoPath}

Contenu attendu (Framework Spec-Driven Design : OpenSpec). Écris les fichiers dans
openspec/changes/{issueKey}-<titre-slug>/ :
1. proposal.md : problème, valeur, périmètre inclus et exclu.
2. design.md : décisions techniques et alternatives écartées.
3. tasks.md : checklist ordonnée et vérifiable de mise en œuvre.
4. specs/<capability>/spec.md : deltas de comportement en sections ## ADDED / ## MODIFIED / ## REMOVED, avec des scénarios Given / When / Then.
5. Valide avec 'openspec validate <change-id> --strict' et corrige les erreurs signalées.

Si le répertoire openspec/ est absent, signale-le au lieu de deviner la structure.`
			} else {
				promptTemplate = `Tu es le Product Owner & Architecte technique pour Taskacao. Rédige une spécification GitHub Spec Kit complète pour la tâche :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git cible : {branchName}
Dossier du projet : {repoPath}

Contenu attendu (Framework Spec-Driven Design : GitHub Spec Kit). Écris les fichiers dans
specs/{issueKey}-<titre-slug>/ :
1. spec.md : contexte, user stories priorisées, périmètre exclu, exigences fonctionnelles numérotées et critères d'acceptation Given / When / Then. Pas de choix d'implémentation ici.
2. plan.md : pile technique, architecture, composants et fichiers cibles, diagrammes de flux Mermaid.
3. tasks.md : checklist ordonnée et vérifiable de mise en œuvre, plus le plan de tests.

Respecte .specify/memory/constitution.md s'il existe. Marque explicitement les points à
clarifier au lieu de les deviner.`
			}
		}
	case "implement":
		promptTemplate = settings.PromptImplement
		if promptTemplate == "" {
			promptTemplate = `Tu es le développeur senior autonome pour Taskacao. Tu dois IMPLÉMENTER ET ÉCRIRE DIRECTEMENT les modifications de code dans le projet ({repoPath}) pour accomplir cette tâche.

Contexte de la tâche :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git : {branchName}
Dossier du projet : {repoPath}

INSTRUCTIONS D'EXÉCUTION OBLIGATOIRES :
1. Vérifie le code existant et assure-toi d'être sur la branche Git '{branchName}'.
2. Écris et modifie concrètement les fichiers nécessaires dans le projet pour implémenter complètement la fonctionnalité ou résoudre le bug.
3. Exécute les commandes de test et de build du projet (ex: npm run build ou go test ./... selon la stack) pour vérifier que le code compile et fonctionne parfaitement sans régression.
4. Fournis un compte-rendu clair des fichiers modifiés/créés et des résultats des validations.`
		}
	case "create_pr":
		promptTemplate = settings.PromptCreatePR
		if promptTemplate == "" {
			promptTemplate = `Tu es l'ingénieur DevOps & Release pour Taskacao. Tu dois finaliser la tâche, commiter et créer la Pull Request ou effectuer la fusion (merge) locale :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git : {branchName}
Dossier du projet : {repoPath}

INSTRUCTIONS D'EXÉCUTION OBLIGATOIRES :
1. Vérifie l'état Git dans '{repoPath}' ('git status' et 'git remote').
2. Assure-toi que toutes les modifications sur la branche '{branchName}' sont commitées proprement avec un message conventionnel (ex: 'feat({issueKey}): {issueTitle}').
3. CAS A : Si un dépôt distant (remote 'origin' ou GitHub/GitLab) est configuré :
   - Pousse la branche vers le remote : 'git push -u origin {branchName}'
   - Crée la Pull Request via 'gh pr create' ou 'glab mr create' si disponible.
4. CAS B : Si AUCUN remote distant n'est configuré (dépôt local uniquement) :
   - Bascule sur la branche principale : 'git checkout main' (ou 'git checkout master' selon la branche par défaut).
   - Fusionne la branche de la tâche : 'git merge --no-ff {branchName} -m "Merge branch \'{branchName}\' for {issueKey}: {issueTitle}"'
5. Fournis un compte-rendu clair de l'action réalisée (Pull Request créée ou Merge local effectué sur la branche principale).`
		}
	case "handoff":
		promptTemplate = `Tu es responsable de la clôture propre de la tâche pour Taskacao. Le code a été revu et fusionné : il reste à documenter le handoff et à nettoyer.

Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git : {branchName}
Dossier du projet : {repoPath}

INSTRUCTIONS D'EXÉCUTION OBLIGATOIRES :
1. Vérifie que la branche '{branchName}' est bien fusionnée dans la branche principale ('git log --oneline main..{branchName}' doit être vide). Si ce n'est pas le cas, ARRÊTE-TOI et dis-le, sans rien nettoyer.
2. Rédige le compte-rendu de handoff : ce qui a été livré, ce qui a été laissé de côté, ce qu'un lecteur doit savoir pour reprendre. Mets à jour la documentation du dépôt si le changement l'exige (README, CHANGELOG, docs).
3. Nettoie l'espace de travail local : retire le worktree de la tâche s'il existe et supprime la branche locale fusionnée.
4. Termine par un compte-rendu court : ce qui a été documenté, ce qui a été nettoyé, ce qui reste à faire côté humain.`

	case "pick":
		promptTemplate = settings.PromptPick
		if promptTemplate == "" {
			promptTemplate = "Tu es le routeur d'orchestration pour Taskacao. Analyse l'état de la tâche {issueKey} ({issueTitle}) et détermine la prochaine action requise dans le cycle SDLC."
		}
	}

	// Ordre de priorité : le prompt surchargé dans les réglages, puis la skill
	// installée, puis le filet codé au-dessus.
	if installedSkill != "" && !settingsPromptOverridden(settings, skillID) {
		promptTemplate = skillSlashPrompt(skillID)
	}

	if customPrompt != "" {
		promptTemplate += "\n\nInstructions supplémentaires fournies par l'utilisateur :\n" + customPrompt
	}

	branchName := ""
	if task.BranchName != nil {
		branchName = *task.BranchName
	} else {
		cleanTitle := strings.ToLower(task.Title)
		cleanTitle = strings.ReplaceAll(cleanTitle, " ", "-")
		cleanTitle = strings.ReplaceAll(cleanTitle, "'", "-")
		if len(cleanTitle) > 30 {
			cleanTitle = cleanTitle[:30]
		}
		branchName = fmt.Sprintf("%s-%s", task.Key, cleanTitle)
	}

	finalPrompt := promptTemplate
	finalPrompt = strings.ReplaceAll(finalPrompt, "{issueKey}", task.Key)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{issueTitle}", task.Title)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{issueDesc}", task.Description)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{branchName}", branchName)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{repoPath}", repoDir)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{tracker}", trackerName)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{repo}", repoName)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{prompt}", customPrompt)

	provider := strings.ToLower(settings.AIProvider)
	if provider == "" {
		provider = "agy"
	}

	steps = append(steps, fmt.Sprintf("🤖 Moteur IA : %s", strings.ToUpper(provider)))

	// The custom-template branch substitutes the task placeholders first, then
	// hands the resolved template to the shared dispatcher.
	resolvedTemplate := settings.AICommandTemplate
	if resolvedTemplate != "" {
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{issueKey}", task.Key)
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{issueTitle}", task.Title)
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{issueDesc}", task.Description)
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{branchName}", branchName)
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{repoPath}", repoDir)
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{tracker}", trackerName)
		resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{repo}", repoName)
	}

	return &AIInvocation{
		RepoDir:  repoDir,
		Provider: provider,
		Template: resolvedTemplate,
		Prompt:   finalPrompt,
		Steps:    steps,
	}, nil
}

// escapeForDoubleQuotes makes a string safe to interpolate inside a
// double-quoted shell word. Inside double quotes sh only treats \, ", $ and `
// specially, so backslash-escaping exactly those four yields the literal text.
//
// This matters for security, not just for quoting: the AI command template is
// executed through 'sh -c', and the prompt embeds task titles and descriptions
// that come straight from Jira, GitHub or Linear. Without escaping, a ticket
// titled `"; rm -rf ~ #` would run as a shell command.
func escapeForDoubleQuotes(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '\\', '"', '$', '`':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// execAgentCommand runs the configured AI CLI for an already-resolved prompt.
// cmdTemplate must have every placeholder other than {prompt} already
// substituted by the caller; it is used when it is non-empty and either the
// provider is "custom" or the template carries a {prompt} slot.
func (r *Runner) execAgentCommand(ctx context.Context, repoDir string, provider string, cmdTemplate string, finalPrompt string) (string, []string, error) {
	var steps []string

	if cmdTemplate != "" && (provider == "custom" || strings.Contains(cmdTemplate, "{prompt}")) {
		cmdToRun := strings.ReplaceAll(cmdTemplate, "{prompt}", escapeForDoubleQuotes(finalPrompt))
		steps = append(steps, fmt.Sprintf("Exécution de la commande personnalisée : %s dans %s", cmdToRun, filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, "sh", "-c", cmdToRun)
		return out, steps, err
	}

	switch provider {
	case "agy":
		agyPath, _ := FindCliTool("agy")
		steps = append(steps, fmt.Sprintf("Exécution de : agy -p \"...\" dans %s", filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, agyPath, "-p", finalPrompt, "--dangerously-skip-permissions")
		return out, steps, err

	case "vibe":
		vibePath, _ := FindCliTool("vibe")
		steps = append(steps, fmt.Sprintf("Exécution de : vibe -p \"...\" dans %s", filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, vibePath, "-p", finalPrompt, "--auto-approve")
		return out, steps, err

	case "claude":
		claudePath, _ := FindCliTool("claude")
		steps = append(steps, fmt.Sprintf("Exécution de : claude -p \"...\" dans %s", filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, claudePath, "-p", finalPrompt)
		return out, steps, err

	case "gemini":
		geminiPath, _ := FindCliTool("gemini")
		steps = append(steps, fmt.Sprintf("Exécution de : gemini -p \"...\" dans %s", filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, geminiPath, "-p", finalPrompt)
		return out, steps, err

	case "cursor":
		cursorPath, _ := FindCliTool("cursor")
		steps = append(steps, fmt.Sprintf("Exécution de : cursor agent -p \"...\" dans %s", filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, cursorPath, "agent", "-p", finalPrompt)
		return out, steps, err

	default:
		cmdStr := cmdTemplate
		if cmdStr == "" {
			cmdStr = "agy -p \"{prompt}\""
		}
		cmdToRun := strings.ReplaceAll(cmdStr, "{prompt}", escapeForDoubleQuotes(finalPrompt))
		steps = append(steps, fmt.Sprintf("Exécution de la commande personnalisée dans %s", filepath.Base(repoDir)))
		out, err := r.runCommand(ctx, repoDir, "sh", "-c", cmdToRun)
		return out, steps, err
	}
}

// RunAgentPrompt executes the configured AI CLI on a free-form prompt with no
// task context. Used by features that are not tied to a single work item, such
// as the daily digest agenda.
func (r *Runner) RunAgentPrompt(ctx context.Context, settings *models.Settings, prompt string) (string, []string, error) {
	if settings == nil {
		return "", nil, fmt.Errorf("réglages IA indisponibles")
	}

	provider := settings.AIProvider
	if provider == "" {
		provider = "agy"
	}

	repoDir := strings.TrimSpace(settings.RepoPath)
	if repoDir == "" {
		repoDir = "."
	}
	if abs, err := filepath.Abs(repoDir); err == nil {
		repoDir = abs
	}
	if stat, err := os.Stat(repoDir); err != nil || !stat.IsDir() {
		cwd, _ := os.Getwd()
		repoDir = cwd
	}

	out, steps, err := r.execAgentCommand(ctx, repoDir, provider, settings.AICommandTemplate, prompt)
	if err != nil {
		return out, steps, fmt.Errorf("exécution de l'agent %s impossible: %w", provider, err)
	}
	return out, steps, nil
}

// GetGitDiff computes git diff for a task branch or working directory
func (r *Runner) GetGitDiff(repoDir string, branchName string, taskKey string, prURL *string) (*models.GitDiffResult, error) {
	if repoDir == "" {
		return nil, fmt.Errorf("répertoire de projet non configuré")
	}

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		return &models.GitDiffResult{
			TaskKey:  taskKey,
			Branch:   branchName,
			RepoPath: repoDir,
			PrURL:    prURL,
			Error:    fmt.Sprintf("Le répertoire '%s' n'existe pas", repoDir),
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify it's a git repo
	if _, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return &models.GitDiffResult{
			TaskKey:  taskKey,
			Branch:   branchName,
			RepoPath: repoDir,
			PrURL:    prURL,
			Error:    fmt.Sprintf("Le répertoire '%s' n'est pas un dépôt Git valide", repoDir),
		}, nil
	}

	// Detect base branch (main or master)
	baseBranch := "main"
	if _, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--verify", "main"); err != nil {
		if _, errMaster := r.runCommand(ctx, repoDir, "git", "rev-parse", "--verify", "master"); errMaster == nil {
			baseBranch = "master"
		} else {
			baseBranch = "HEAD~1"
		}
	}

	// Get current active branch
	currBranch, _ := r.runCommand(ctx, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	currBranch = strings.TrimSpace(currBranch)

	targetBranch := strings.TrimSpace(branchName)
	if targetBranch == "" {
		targetBranch = currBranch
	}

	// Try diffing between baseBranch and targetBranch
	var rawDiff string
	var numstat string
	var diffErr error

	// Check if targetBranch exists
	hasBranch := false
	if _, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--verify", targetBranch); err == nil {
		hasBranch = true
	}

	if hasBranch && targetBranch != baseBranch {
		// Three-dot diff against base
		rawDiff, diffErr = r.runCommand(ctx, repoDir, "git", "diff", fmt.Sprintf("%s...%s", baseBranch, targetBranch))
		numstat, _ = r.runCommand(ctx, repoDir, "git", "diff", "--numstat", fmt.Sprintf("%s...%s", baseBranch, targetBranch))
		if strings.TrimSpace(rawDiff) == "" {
			// Fallback to two-dot diff
			rawDiff, diffErr = r.runCommand(ctx, repoDir, "git", "diff", fmt.Sprintf("%s..%s", baseBranch, targetBranch))
			numstat, _ = r.runCommand(ctx, repoDir, "git", "diff", "--numstat", fmt.Sprintf("%s..%s", baseBranch, targetBranch))
		}
	}

	// If current branch is target branch and there are uncommitted working tree changes, include them
	if currBranch == targetBranch || strings.TrimSpace(rawDiff) == "" {
		workDiff, _ := r.runCommand(ctx, repoDir, "git", "diff", "HEAD")
		workNumstat, _ := r.runCommand(ctx, repoDir, "git", "diff", "--numstat", "HEAD")
		if strings.TrimSpace(workDiff) != "" {
			if rawDiff != "" {
				rawDiff = rawDiff + "\n" + workDiff
				numstat = numstat + "\n" + workNumstat
			} else {
				rawDiff = workDiff
				numstat = workNumstat
			}
		}
	}

	// If still empty and on base branch or no commits between, try diff of last commit
	if strings.TrimSpace(rawDiff) == "" {
		lastCommitDiff, _ := r.runCommand(ctx, repoDir, "git", "diff", "HEAD~1..HEAD")
		lastCommitNumstat, _ := r.runCommand(ctx, repoDir, "git", "diff", "--numstat", "HEAD~1..HEAD")
		if strings.TrimSpace(lastCommitDiff) != "" {
			rawDiff = lastCommitDiff
			numstat = lastCommitNumstat
		}
	}

	if diffErr != nil && strings.TrimSpace(rawDiff) == "" {
		return &models.GitDiffResult{
			TaskKey:    taskKey,
			Branch:     targetBranch,
			BaseBranch: baseBranch,
			RepoPath:   repoDir,
			PrURL:      prURL,
			Error:      fmt.Sprintf("Erreur lors de la récupération du diff : %v", diffErr),
		}, nil
	}

	// Parse numstat and split rawDiff into file blocks
	fileMap := make(map[string]*models.GitDiffFile)
	totalInsertions := 0
	totalDeletions := 0

	for _, line := range strings.Split(numstat, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			adds, _ := strconv.Atoi(parts[0])
			dels, _ := strconv.Atoi(parts[1])
			path := parts[2]
			totalInsertions += adds
			totalDeletions += dels

			fileMap[path] = &models.GitDiffFile{
				Path:      path,
				Status:    "modified",
				Additions: adds,
				Deletions: dels,
			}
		}
	}

	// Parse rawDiff chunks by file
	diffChunks := strings.Split(rawDiff, "diff --git ")
	var files []models.GitDiffFile

	for _, chunk := range diffChunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		fullChunk := "diff --git " + chunk
		firstLine := strings.SplitN(chunk, "\n", 2)[0]
		parts := strings.Fields(firstLine)

		filePath := ""
		if len(parts) >= 2 {
			filePath = strings.TrimPrefix(parts[1], "b/")
		}

		status := "modified"
		if strings.Contains(chunk, "new file mode") {
			status = "added"
		} else if strings.Contains(chunk, "deleted file mode") {
			status = "deleted"
		} else if strings.Contains(chunk, "similarity index") || strings.Contains(chunk, "rename from") {
			status = "renamed"
		}

		adds := 0
		dels := 0
		if existingFile, ok := fileMap[filePath]; ok {
			adds = existingFile.Additions
			dels = existingFile.Deletions
			existingFile.Diff = fullChunk
			existingFile.Status = status
			files = append(files, *existingFile)
			delete(fileMap, filePath)
		} else {
			// Count manually
			for _, l := range strings.Split(chunk, "\n") {
				if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
					adds++
				} else if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
					dels++
				}
			}
			totalInsertions += adds
			totalDeletions += dels
			files = append(files, models.GitDiffFile{
				Path:      filePath,
				Status:    status,
				Additions: adds,
				Deletions: dels,
				Diff:      fullChunk,
			})
		}
	}

	return &models.GitDiffResult{
		TaskKey:      taskKey,
		Branch:       targetBranch,
		BaseBranch:   baseBranch,
		RepoPath:     repoDir,
		IsClean:      len(files) == 0,
		FilesChanged: len(files),
		Insertions:   totalInsertions,
		Deletions:    totalDeletions,
		Files:        files,
		RawDiff:      rawDiff,
		PrURL:        prURL,
	}, nil
}

// GetCwdGitStatus returns git status, active branch, and repo state for a directory
func (r *Runner) GetCwdGitStatus(repoDir string) (*models.GitStatusInfo, error) {
	if repoDir == "" {
		cwd, err := os.Getwd()
		if err == nil && cwd != "" {
			repoDir = cwd
		}
	}

	if repoDir == "" {
		return &models.GitStatusInfo{
			Error: "Répertoire de travail non spécifié",
		}, nil
	}

	if fi, err := os.Stat(repoDir); err != nil || !fi.IsDir() {
		return &models.GitStatusInfo{
			RepoPath:  repoDir,
			IsGitRepo: false,
			Error:     fmt.Sprintf("Le répertoire '%s' n'existe pas", repoDir),
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if git repo
	if _, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return &models.GitStatusInfo{
			RepoPath:  repoDir,
			IsGitRepo: false,
			Error:     "Ce dossier n'est pas un dépôt Git",
		}, nil
	}

	// Real repo top level path
	topLevel, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--show-toplevel")
	if err == nil && strings.TrimSpace(topLevel) != "" {
		repoDir = strings.TrimSpace(topLevel)
	}

	// Current active branch
	branch, _ := r.runCommand(ctx, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	branch = strings.TrimSpace(branch)
	if branch == "HEAD" || branch == "" {
		// Detached HEAD or tag
		if tag, err := r.runCommand(ctx, repoDir, "git", "describe", "--tags", "--always"); err == nil && strings.TrimSpace(tag) != "" {
			branch = strings.TrimSpace(tag)
		} else if shortHash, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--short", "HEAD"); err == nil {
			branch = strings.TrimSpace(shortHash)
		}
	}

	// Base branch detection
	baseBranch := "main"
	if _, err := r.runCommand(ctx, repoDir, "git", "rev-parse", "--verify", "main"); err != nil {
		if _, errMaster := r.runCommand(ctx, repoDir, "git", "rev-parse", "--verify", "master"); errMaster == nil {
			baseBranch = "master"
		}
	}

	// Status porcelain
	porcelainOut, _ := r.runCommand(ctx, repoDir, "git", "status", "--porcelain")
	modifiedCount := 0
	untrackedCount := 0
	for _, line := range strings.Split(porcelainOut, "\n") {
		line = strings.TrimRight(line, "\r\n")
		if len(line) < 2 {
			continue
		}
		if strings.HasPrefix(line, "??") {
			untrackedCount++
		} else {
			modifiedCount++
		}
	}

	isClean := (modifiedCount == 0 && untrackedCount == 0)

	// Remote info
	remoteName, _ := r.runCommand(ctx, repoDir, "git", "remote")
	remoteName = strings.TrimSpace(strings.Split(remoteName, "\n")[0])
	remoteURL := ""
	if remoteName != "" {
		if rURL, err := r.runCommand(ctx, repoDir, "git", "remote", "get-url", remoteName); err == nil {
			remoteURL = strings.TrimSpace(rURL)
		}
	}

	// Ahead / Behind counts against upstream if configured
	ahead := 0
	behind := 0
	if countsOut, err := r.runCommand(ctx, repoDir, "git", "rev-list", "--left-right", "--count", "HEAD...@{u}"); err == nil {
		parts := strings.Fields(countsOut)
		if len(parts) >= 2 {
			ahead, _ = strconv.Atoi(parts[0])
			behind, _ = strconv.Atoi(parts[1])
		}
	}

	// Latest commit message & short hash
	latestCommit, _ := r.runCommand(ctx, repoDir, "git", "log", "-1", "--format=%h %s (%cr)")
	latestCommit = strings.TrimSpace(latestCommit)

	return &models.GitStatusInfo{
		RepoPath:       repoDir,
		IsGitRepo:      true,
		Branch:         branch,
		BaseBranch:     baseBranch,
		IsClean:        isClean,
		ModifiedCount:  modifiedCount,
		UntrackedCount: untrackedCount,
		Ahead:          ahead,
		Behind:         behind,
		RemoteName:     remoteName,
		RemoteURL:      remoteURL,
		LatestCommit:   latestCommit,
	}, nil
}

// FindEditorBinary locates the binary or execution command for a given code editor on the host OS
func FindEditorBinary(editor string) (string, []string) {
	editor = strings.TrimSpace(editor)
	base := strings.ToLower(editor)

	// 1. Direct LookPath or standard CLI path
	if p, err := FindCliTool(editor); err == nil && p != "" {
		return p, nil
	}

	homeDir, _ := os.UserHomeDir()

	// 2. Known application paths & fallbacks on macOS
	if runtime.GOOS == "darwin" {
		switch base {
		case "code", "vscode":
			macPaths := []string{
				"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
				"/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code",
				filepath.Join(homeDir, "Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"),
			}
			for _, mp := range macPaths {
				if _, err := os.Stat(mp); err == nil {
					return mp, nil
				}
			}
			return "open", []string{"-a", "Visual Studio Code"}

		case "cursor":
			macPaths := []string{
				"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
				filepath.Join(homeDir, "Applications/Cursor.app/Contents/Resources/app/bin/cursor"),
			}
			for _, mp := range macPaths {
				if _, err := os.Stat(mp); err == nil {
					return mp, nil
				}
			}
			return "open", []string{"-a", "Cursor"}

		case "zed":
			macPaths := []string{
				"/Applications/Zed.app/Contents/MacOS/cli",
				filepath.Join(homeDir, "Applications/Zed.app/Contents/MacOS/cli"),
			}
			for _, mp := range macPaths {
				if _, err := os.Stat(mp); err == nil {
					return mp, nil
				}
			}
			return "open", []string{"-a", "Zed"}

		case "subl", "sublime":
			macPaths := []string{
				"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl",
				filepath.Join(homeDir, "Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"),
			}
			for _, mp := range macPaths {
				if _, err := os.Stat(mp); err == nil {
					return mp, nil
				}
			}
			return "open", []string{"-a", "Sublime Text"}

		case "idea", "intellij":
			return "open", []string{"-a", "IntelliJ IDEA"}

		case "webstorm":
			return "open", []string{"-a", "WebStorm"}
		}
	}

	return editor, nil
}

// OpenInEditor opens the specified directory or file in a code editor (defaults to 'code' for VS Code)
func (r *Runner) OpenInEditor(editorCmd string, targetPath string) error {
	if strings.TrimSpace(editorCmd) == "" {
		editorCmd = "code"
	}
	editorCmd = strings.TrimSpace(editorCmd)

	if targetPath == "" {
		targetPath = "."
	}
	targetPath = filepath.Clean(targetPath)
	if abs, err := filepath.Abs(targetPath); err == nil {
		targetPath = abs
	}

	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		parts = []string{"code"}
	}

	bin, prefixArgs := FindEditorBinary(parts[0])
	var args []string
	if len(prefixArgs) > 0 {
		args = append(args, prefixArgs...)
		args = append(args, parts[1:]...)
		args = append(args, targetPath)
	} else {
		args = append(args, parts[1:]...)
		args = append(args, targetPath)
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "PATH="+GetDynamicCustomPath()+":"+os.Getenv("PATH"))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open in '%s': %w", editorCmd, err)
	}
	return nil
}

// runCommandForTest runs a shell snippet and returns its raw stdout. It exists
// so the escaping tests can assert what the shell actually parses.
func (r *Runner) runCommandForTest(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.runCommand(ctx, "", "sh", "-c", script)
}

// settingsPromptOverridden says whether the user deliberately wrote their own
// prompt for a step in the settings. That choice keeps priority over the skill
// installed in the repository.
func settingsPromptOverridden(settings *models.Settings, skillID string) bool {
	if settings == nil {
		return false
	}
	switch skillID {
	case "clarify":
		return strings.TrimSpace(settings.PromptClarify) != ""
	case "specify":
		return strings.TrimSpace(settings.PromptSpecify) != ""
	case "implement":
		return strings.TrimSpace(settings.PromptImplement) != ""
	case "create_pr", "review":
		return strings.TrimSpace(settings.PromptCreatePR) != ""
	}
	return false
}

// SessionCommandLine turns an invocation into a shell command line that can be
// injected into a PTY session, plus the cleanup of the temporary file it uses.
//
// The prompt goes through a file rather than the command line on purpose: it is
// several thousand characters of markdown carrying ticket text, and pushing that
// through a terminal line editor would truncate it and mangle the quoting. The
// file is read with a command substitution, so the agent still receives it as a
// single argument.
func (r *Runner) SessionCommandLine(inv *AIInvocation) (string, func(), error) {
	if inv == nil {
		return "", func() {}, fmt.Errorf("invocation vide")
	}

	f, err := os.CreateTemp("", "taskacao-prompt-*.md")
	if err != nil {
		return "", func() {}, err
	}
	if _, err := f.WriteString(inv.Prompt); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", func() {}, err
	}
	_ = f.Close()
	promptFile := f.Name()
	cleanup := func() { _ = os.Remove(promptFile) }

	// $(cat 'fichier') : le chemin est un temporaire que nous fabriquons, donc
	// sans apostrophe, et le prompt n'est jamais relu par le shell.
	promptRef := fmt.Sprintf(`"$(cat '%s')"`, promptFile)

	if inv.Template != "" && (inv.Provider == "custom" || strings.Contains(inv.Template, "{prompt}")) {
		return strings.ReplaceAll(inv.Template, "{prompt}", "$(cat '"+promptFile+"')"), cleanup, nil
	}

	switch inv.Provider {
	case "agy":
		bin, _ := FindCliTool("agy")
		return fmt.Sprintf("%s -p %s --dangerously-skip-permissions", shellQuote(bin), promptRef), cleanup, nil
	case "vibe":
		bin, _ := FindCliTool("vibe")
		return fmt.Sprintf("%s -p %s --auto-approve", shellQuote(bin), promptRef), cleanup, nil
	case "claude":
		bin, _ := FindCliTool("claude")
		return fmt.Sprintf("%s -p %s", shellQuote(bin), promptRef), cleanup, nil
	case "gemini":
		bin, _ := FindCliTool("gemini")
		return fmt.Sprintf("%s -p %s", shellQuote(bin), promptRef), cleanup, nil
	case "cursor":
		bin, _ := FindCliTool("cursor")
		return fmt.Sprintf("%s agent -p %s", shellQuote(bin), promptRef), cleanup, nil
	}

	template := inv.Template
	if template == "" {
		template = `agy -p "{prompt}"`
	}
	return strings.ReplaceAll(template, "{prompt}", "$(cat '"+promptFile+"')"), cleanup, nil
}

// shellQuote wraps a path in single quotes so a space in it cannot split the
// command. The binary paths come from FindCliTool, not from user input.
func shellQuote(s string) string {
	if s == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// InteractiveAgentLaunch is the command that opens the agent CLI as a live
// session, as opposed to the one-shot print mode used for headless steps.
//
// Aucun drapeau de contournement des permissions ici, contrairement au mode
// non interactif : un humain regarde la session, il peut répondre aux demandes
// de permission, et c'est précisément l'intérêt de ce mode.
func InteractiveAgentLaunch(settings *models.Settings) (string, error) {
	provider := "agy"
	if settings != nil && strings.TrimSpace(settings.AIProvider) != "" {
		provider = strings.ToLower(strings.TrimSpace(settings.AIProvider))
	}

	switch provider {
	case "agy", "vibe", "claude", "gemini":
		return resolveAgentBinary(provider, "")
	case "cursor":
		line, err := resolveAgentBinary("cursor", "")
		if err != nil {
			return "", err
		}
		return line + " agent", nil
	case "custom":
		// Un moteur personnalisé n'a que son modèle de commande : son premier mot
		// est le binaire, et c'est lui qu'on ouvre en interactif.
		return resolveAgentBinary(firstWord(settings.AICommandTemplate), provider)
	}
	return "", fmt.Errorf("le moteur %q n'a pas de mode interactif connu : configure un moteur agy, claude, gemini, cursor ou vibe sur le projet", provider)
}

// resolveAgentBinary finds an engine binary and says where it looked when it
// fails, because "binaire introuvable" alone leaves nothing to act on.
func resolveAgentBinary(tool, provider string) (string, error) {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return "", fmt.Errorf("aucun binaire à lancer pour le moteur %q : renseigne son modèle de commande", provider)
	}
	bin, err := FindCliTool(tool)
	if err != nil || bin == "" || bin == tool {
		if _, statErr := os.Stat(bin); statErr != nil {
			home, _ := os.UserHomeDir()
			return "", fmt.Errorf(
				"moteur %q introuvable : %s absent du PATH, de %s/.local/bin, /opt/homebrew/bin, /usr/local/bin et /usr/bin",
				strings.TrimSpace(provider+" "+tool), tool, home,
			)
		}
	}
	return shellQuote(bin), nil
}

// firstWord extracts the binary from a command template such as
// `claude --dangerously-skip-permissions -p "{prompt}"`.
func firstWord(template string) string {
	fields := strings.Fields(strings.TrimSpace(template))
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "'\"")
}

// SkillCallLine is what gets typed into the running agent: the skill's slash
// command, the ticket key, and just enough context to place the work.
//
// C'est la skill qui porte les instructions, pas ce message : le SKILL.md est
// rendu depuis l'éditeur de Taskacao, l'agent le lit en exécutant la commande.
func SkillCallLine(skillID string, task *models.Task, trackerName string) string {
	dirName := models.SkillDirNames[skillID]
	if dirName == "" {
		dirName = skillID
	}
	return SkillCallLineWithCommand("/"+dirName, task, trackerName)
}

// SkillCallLineWithCommand builds the same line for an explicit slash command,
// so a project that renamed its skills keeps its own command.
func SkillCallLineWithCommand(command string, task *models.Task, trackerName string) string {
	command = "/" + strings.TrimPrefix(strings.TrimSpace(command), "/")
	if task == nil {
		return command
	}

	line := fmt.Sprintf("%s %s", command, task.Key)
	title := strings.TrimSpace(task.Title)
	if title != "" {
		line += fmt.Sprintf(" (%s)", collapseSpaces(title))
	}
	if trackerName != "" && trackerName != "local" {
		line += fmt.Sprintf(" suivi dans %s", trackerName)
	}
	return line
}

// collapseSpaces flattens a title to a single line: a newline in the injected
// text would be read as a validation by the agent's prompt.
func collapseSpaces(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
