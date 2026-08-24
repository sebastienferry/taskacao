package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	tools := []string{"git", "gh", "linear", "acli", "agy", "vibe", "claude", "gemini", "codex"}
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
				out, aErr := r.runCommand(ctx, repoPath, path, "auth", "status")
				if aErr == nil || strings.Contains(out, "Logged in") {
					status.AuthStatus = "Authenticated"
					status.Details = "Atlassian CLI connected"
				} else {
					status.AuthStatus = "Ready"
					status.Details = "Atlassian CLI available"
				}
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
			status.Details = fmt.Sprintf("Tool '%s' not found in PATH", tool)
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

type JiraIssueField struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Status      struct {
		Name string `json:"name"`
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

func (r *Runner) SyncFromJira(projectKey string, repoPath string, trackerUrl string) ([]models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	var args []string
	if projectKey != "" {
		args = []string{"jira", "workitem", "list", "--project", projectKey, "--output", "json"}
	} else {
		args = []string{"jira", "workitem", "list", "--output", "json"}
	}

	output, err := r.runCommand(ctx, repoPath, acliPath, args...)
	if err != nil {
		args = []string{"jira", "issue", "list", "--output", "json"}
		if projectKey != "" {
			args = append(args, "--project", projectKey)
		}
		var err2 error
		output, err2 = r.runCommand(ctx, repoPath, acliPath, args...)
		if err2 != nil {
			return nil, fmt.Errorf("failed to query Jira via acli: %w (output: %s)", err, output)
		}
	}

	var items []JiraIssueItem
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		var resp JiraQueryResponse
		if err2 := json.Unmarshal([]byte(output), &resp); err2 == nil {
			items = resp.Issues
		} else {
			return nil, fmt.Errorf("failed to parse Jira JSON response: %w", err)
		}
	}

	var tasks []models.Task
	for i, item := range items {
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
		case strings.Contains(stName, "clarif") || strings.Contains(stName, "triage") || strings.Contains(stName, "backlog") || strings.Contains(stName, "open"):
			status = models.StatusToClarify
		case strings.Contains(stName, "specif") || strings.Contains(stName, "to do") || strings.Contains(stName, "todo") || strings.Contains(stName, "selected"):
			status = models.StatusToSpecify
		case strings.Contains(stName, "progress") || strings.Contains(stName, "in dev") || strings.Contains(stName, "implement"):
			status = models.StatusToImplement
		case strings.Contains(stName, "test") || strings.Contains(stName, "qa") || strings.Contains(stName, "validation") || strings.Contains(stName, "review") || strings.Contains(stName, "pr"):
			status = models.StatusToTest
		case strings.Contains(stName, "done") || strings.Contains(stName, "closed") || strings.Contains(stName, "resolved") || strings.Contains(stName, "finish"):
			status = models.StatusFinished
		default:
			status = models.StatusToClarify
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
			Description:    item.Fields.Description,
			Status:         status,
			Priority:       priority,
			Labels:         item.Fields.Labels,
			Assignee:       item.Fields.Assignee.DisplayName,
			AssigneeAvatar: avatar,
			Position:       i + 1,
			Source:         "jira",
			ExternalURL:    extURL,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
	}
	return tasks, nil
}

func (r *Runner) CreateJiraIssue(projectKey string, repoPath string, trackerUrl string, title string, description string, priority models.Priority, labels []string) (*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	args := []string{"jira", "workitem", "create", "--project", projectKey, "--summary", title, "--description", description, "--output", "json"}
	output, err := r.runCommand(ctx, repoPath, acliPath, args...)
	if err != nil {
		args = []string{"jira", "issue", "create", "--project", projectKey, "--summary", title, "--description", description, "--output", "json"}
		var err2 error
		output, err2 = r.runCommand(ctx, repoPath, acliPath, args...)
		if err2 != nil {
			return nil, fmt.Errorf("failed to create Jira issue: %w (output: %s)", err, output)
		}
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

func (r *Runner) UpdateJiraIssueState(issueKey string, status models.Status, repoPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	stateName := "In Progress"
	switch status {
	case models.StatusToClarify, models.StatusBacklog:
		stateName = "To Do"
	case models.StatusToSpecify:
		stateName = "To Do"
	case models.StatusToImplement, models.StatusInProgress:
		stateName = "In Progress"
	case models.StatusToTest, models.StatusToValidate:
		stateName = "In Review"
	case models.StatusToClose, models.StatusFinished, models.StatusDone:
		stateName = "Done"
	}

	args := []string{"jira", "workitem", "transition", issueKey, "--state", stateName}
	_, err := r.runCommand(ctx, repoPath, acliPath, args...)
	return err
}

func (r *Runner) PostJiraComment(issueKey string, body string, repoPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	args := []string{"jira", "workitem", "comment", issueKey, "--body", body}
	_, err := r.runCommand(ctx, repoPath, acliPath, args...)
	return err
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

func (r *Runner) RunAI(settings *models.Settings, skillID string, task *models.Task, customPrompt string) (string, []string, error) {
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
			if settings.SpecFramework == "openfeature" {
				promptTemplate = `Tu es le Lead Architecte & Feature Engineer pour Taskacao. Rédige une spécification technique OpenFeature complète et standardisée pour la tâche :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git cible : {branchName}
Dossier du projet : {repoPath}

Contenu attendu (Framework Spec-Driven Design : OpenFeature) :
1. Définition des Feature Flags (Flag Key, Types: boolean/string/number/object, Valeurs par défaut, Variations)
2. Evaluation Context & Règles de ciblage (Attributs utilisateur, tenant, environnement)
3. Intégration OpenFeature SDK (Provider, Evaluation Hooks, Fallbacks de sécurité)
4. Cycle de vie du Flag (Création -> Rollout progressif -> Dépréciation & Nettoyage de code)
5. Plan de tests et de validation (Scénarios Given / When / Then).`
			} else {
				promptTemplate = `Tu es le Product Owner & Architecte technique pour Taskacao. Rédige une spécification technique SpecKit complète pour la tâche :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git cible : {branchName}
Dossier du projet : {repoPath}

Contenu attendu (Framework Spec-Driven Design : SpecKit) :
1. Contexte & User Stories
2. Architecture et composants cibles (fichiers à créer / modifier)
3. Critères d'acceptation détaillés (Scénarios Given / When / Then)
4. Plan de tests et de validation.`
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
	case "pick":
		promptTemplate = settings.PromptPick
		if promptTemplate == "" {
			promptTemplate = "Tu es le routeur d'orchestration pour Taskacao. Analyse l'état de la tâche {issueKey} ({issueTitle}) et détermine la prochaine action requise dans le cycle SDLC."
		}
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var output string
	var execErr error

	if settings.AICommandTemplate != "" && (provider == "custom" || strings.Contains(settings.AICommandTemplate, "{prompt}")) {
		cmdStr := settings.AICommandTemplate
		cmdToRun := strings.ReplaceAll(cmdStr, "{prompt}", finalPrompt)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{issueKey}", task.Key)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{issueTitle}", task.Title)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{issueDesc}", task.Description)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{branchName}", branchName)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{repoPath}", repoDir)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{tracker}", trackerName)
		cmdToRun = strings.ReplaceAll(cmdToRun, "{repo}", repoName)

		steps = append(steps, fmt.Sprintf("Exécution de la commande personnalisée : %s dans %s", cmdToRun, filepath.Base(repoDir)))
		output, execErr = r.runCommand(ctx, repoDir, "sh", "-c", cmdToRun)
	} else {
		switch provider {
		case "agy":
			agyPath, _ := FindCliTool("agy")
			steps = append(steps, fmt.Sprintf("Exécution de : agy -p \"...\" dans %s", filepath.Base(repoDir)))
			output, execErr = r.runCommand(ctx, repoDir, agyPath, "-p", finalPrompt, "--dangerously-skip-permissions")

		case "vibe":
			vibePath, _ := FindCliTool("vibe")
			steps = append(steps, fmt.Sprintf("Exécution de : vibe -p \"...\" dans %s", filepath.Base(repoDir)))
			output, execErr = r.runCommand(ctx, repoDir, vibePath, "-p", finalPrompt, "--auto-approve")

		case "claude":
			claudePath, _ := FindCliTool("claude")
			steps = append(steps, fmt.Sprintf("Exécution de : claude -p \"...\" dans %s", filepath.Base(repoDir)))
			output, execErr = r.runCommand(ctx, repoDir, claudePath, "-p", finalPrompt)

		case "gemini":
			geminiPath, _ := FindCliTool("gemini")
			steps = append(steps, fmt.Sprintf("Exécution de : gemini -p \"...\" dans %s", filepath.Base(repoDir)))
			output, execErr = r.runCommand(ctx, repoDir, geminiPath, "-p", finalPrompt)

		case "cursor":
			cursorPath, _ := FindCliTool("cursor")
			steps = append(steps, fmt.Sprintf("Exécution de : cursor agent -p \"...\" dans %s", filepath.Base(repoDir)))
			output, execErr = r.runCommand(ctx, repoDir, cursorPath, "agent", "-p", finalPrompt)

		default:
			cmdStr := settings.AICommandTemplate
			if cmdStr == "" {
				cmdStr = "agy -p \"{prompt}\""
			}
			cmdToRun := strings.ReplaceAll(cmdStr, "{prompt}", finalPrompt)
			steps = append(steps, fmt.Sprintf("Exécution de la commande personnalisée dans %s", filepath.Base(repoDir)))
			output, execErr = r.runCommand(ctx, repoDir, "sh", "-c", cmdToRun)
		}
	}

	if execErr != nil {
		steps = append(steps, fmt.Sprintf("⚠️ Erreur d'exécution : %v", execErr))
		return fmt.Sprintf("### ⚠️ Erreur lors de l'exécution de la commande IA (%s)\n\n```text\n%s\n```\n\n*Vérifiez que le binaire '%s' est bien accessible et authentifié.*", provider, output, provider), steps, nil
	}

	steps = append(steps, "✅ Réponse générée par le modèle IA avec succès")
	return output, steps, nil
}

// RunAIChatStream executes the configured AI CLI with live streaming of output chunks and steps
func (r *Runner) RunAIChatStream(
	ctx context.Context,
	settings *models.Settings,
	task *models.Task,
	skillID string,
	userPrompt string,
	history []models.TaskMessage,
	onChunk func(chunk string),
	onStep func(step string),
) (string, []string, error) {
	repoDir := ""
	if settings != nil && settings.RepoPath != "" {
		repoDir = strings.TrimSpace(settings.RepoPath)
	}
	if repoDir == "" {
		cwd, _ := os.Getwd()
		repoDir = cwd
	}
	repoDir = filepath.Clean(repoDir)
	if abs, err := filepath.Abs(repoDir); err == nil {
		repoDir = abs
	}

	var steps []string
	addStep := func(step string) {
		steps = append(steps, step)
		if onStep != nil {
			onStep(step)
		}
	}

	if stat, err := os.Stat(repoDir); err != nil || !stat.IsDir() {
		cwd, _ := os.Getwd()
		addStep(fmt.Sprintf("⚠️ Répertoire projet '%s' introuvable, repli sur : %s", repoDir, cwd))
		repoDir = cwd
	} else {
		addStep(fmt.Sprintf("📁 Dossier de travail actif (CWD) : %s", repoDir))
	}

	branchName := ""
	if task.BranchName != nil && *task.BranchName != "" {
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

	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("Tu es l'agent Copilot autonome pour Taskacao. Tu es en discussion directe avec le développeur concernant la tâche suivante :\n"))
	promptBuilder.WriteString(fmt.Sprintf("- Clé : %s\n- Titre : %s\n- Description : %s\n- Branche Git : %s\n- Dossier du projet (CWD) : %s\n\n", task.Key, task.Title, task.Description, branchName, repoDir))

	if skillID != "" {
		switch skillID {
		case "clarify":
			promptBuilder.WriteString("Objectif de l'action : Analyse les ambiguïtés techniques et pose des questions précises de cadrage.\n\n")
		case "specify":
			promptBuilder.WriteString("Objectif de l'action : Rédige ou affine la spécification technique (Speckit) et les critères d'acceptation.\n\n")
		case "implement":
			promptBuilder.WriteString("Objectif de l'action : Implémente et écris directement les modifications de code dans le projet, puis teste le build.\n\n")
		case "create_pr":
			promptBuilder.WriteString("Objectif de l'action : Vérifie l'état Git, commite les changements et prépare/crée la PR.\n\n")
		}
	}

	// Include recent conversation history (up to last 10 messages)
	if len(history) > 0 {
		promptBuilder.WriteString("--- Historique récent des échanges ---\n")
		startIdx := 0
		if len(history) > 10 {
			startIdx = len(history) - 10
		}
		for _, msg := range history[startIdx:] {
			roleLabel := "Développeur"
			if msg.Role == "assistant" {
				roleLabel = "Agent Copilot"
			} else if msg.Role == "system" {
				roleLabel = "Système"
			}
			promptBuilder.WriteString(fmt.Sprintf("%s : %s\n\n", roleLabel, msg.Content))
		}
		promptBuilder.WriteString("--- Fin de l'historique ---\n\n")
	}

	promptBuilder.WriteString(fmt.Sprintf("Dernier message / consigne du développeur :\n%s\n\n", userPrompt))
	promptBuilder.WriteString("Consignes de réponse :\n1. Réponds de manière concise, structurée (en Markdown) et actionnable.\n2. Si une action de code ou de commande est demandée, réalise-la directement dans le CWD et résume ce qui a été fait.\n")

	finalPrompt := promptBuilder.String()

	provider := strings.ToLower(settings.AIProvider)
	if provider == "" {
		provider = "agy"
	}
	addStep(fmt.Sprintf("🤖 Moteur IA : %s", strings.ToUpper(provider)))

	var cmd *exec.Cmd
	if settings.AICommandTemplate != "" && (provider == "custom" || strings.Contains(settings.AICommandTemplate, "{prompt}")) {
		cmdStr := settings.AICommandTemplate
		cmdToRun := strings.ReplaceAll(cmdStr, "{prompt}", finalPrompt)
		addStep(fmt.Sprintf("Exécution de la commande personnalisée dans %s", filepath.Base(repoDir)))
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdToRun)
	} else {
		switch provider {
		case "agy":
			agyPath, _ := FindCliTool("agy")
			addStep(fmt.Sprintf("Exécution en direct : agy -p \"...\" dans %s", filepath.Base(repoDir)))
			cmd = exec.CommandContext(ctx, agyPath, "-p", finalPrompt, "--dangerously-skip-permissions")
		case "vibe":
			vibePath, _ := FindCliTool("vibe")
			addStep(fmt.Sprintf("Exécution en direct : vibe -p \"...\" dans %s", filepath.Base(repoDir)))
			cmd = exec.CommandContext(ctx, vibePath, "-p", finalPrompt, "--auto-approve")
		case "claude":
			claudePath, _ := FindCliTool("claude")
			addStep(fmt.Sprintf("Exécution en direct : claude -p \"...\" dans %s", filepath.Base(repoDir)))
			cmd = exec.CommandContext(ctx, claudePath, "-p", finalPrompt)
		case "gemini":
			geminiPath, _ := FindCliTool("gemini")
			addStep(fmt.Sprintf("Exécution en direct : gemini -p \"...\" dans %s", filepath.Base(repoDir)))
			cmd = exec.CommandContext(ctx, geminiPath, "-p", finalPrompt)
		case "cursor":
			cursorPath, _ := FindCliTool("cursor")
			addStep(fmt.Sprintf("Exécution en direct : cursor agent -p \"...\" dans %s", filepath.Base(repoDir)))
			cmd = exec.CommandContext(ctx, cursorPath, "agent", "-p", finalPrompt)
		default:
			cmdStr := settings.AICommandTemplate
			if cmdStr == "" {
				cmdStr = "agy -p \"{prompt}\""
			}
			cmdToRun := strings.ReplaceAll(cmdStr, "{prompt}", finalPrompt)
			addStep(fmt.Sprintf("Exécution de la commande personnalisée dans %s", filepath.Base(repoDir)))
			cmd = exec.CommandContext(ctx, "sh", "-c", cmdToRun)
		}
	}

	cmd.Dir = repoDir

	// Inherit and extend PATH
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

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", steps, fmt.Errorf("failed to open stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", steps, fmt.Errorf("failed to open stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		addStep(fmt.Sprintf("⚠️ Échec du démarrage : %v", err))
		return "", steps, fmt.Errorf("failed to start command: %w", err)
	}

	var totalOutput bytes.Buffer
	var outMu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(2)

	// Stream Stdout
	go func() {
		defer wg.Done()
		buf := make([]byte, 512)
		for {
			n, readErr := stdoutPipe.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				outMu.Lock()
				totalOutput.Write(buf[:n])
				outMu.Unlock()
				if onChunk != nil {
					onChunk(chunk)
				}
			}
			if readErr != nil {
				break
			}
		}
	}()

	// Stream Stderr
	go func() {
		defer wg.Done()
		buf := make([]byte, 512)
		for {
			n, readErr := stderrPipe.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				outMu.Lock()
				totalOutput.Write(buf[:n])
				outMu.Unlock()
				if onChunk != nil {
					onChunk(chunk)
				}
			}
			if readErr != nil {
				break
			}
		}
	}()

	wg.Wait()
	cmdErr := cmd.Wait()

	result := totalOutput.String()
	if cmdErr != nil {
		addStep(fmt.Sprintf("⚠️ Fin avec avertissement/code retour : %v", cmdErr))
	} else {
		addStep("✅ Réponse générée et transmise en direct avec succès")
	}

	return result, steps, nil
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

