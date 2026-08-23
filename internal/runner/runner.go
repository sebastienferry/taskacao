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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tasks/internal/models"
)

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

// Helper to execute command with timeout and full environment
func (r *Runner) runCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// Inherit and extend PATH to include ~/.local/bin and Homebrew paths
	env := os.Environ()
	customPath := "/Users/sferry/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
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
	tools := []string{"git", "gh", "linear", "agy", "vibe", "claude"}
	var results []models.CliStatus

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tool := range tools {
		path, err := exec.LookPath(tool)
		if err != nil {
			fallbackPaths := []string{
				"/Users/sferry/.local/bin/" + tool,
				"/opt/homebrew/bin/" + tool,
				"/usr/local/bin/" + tool,
				"/usr/bin/" + tool,
			}
			for _, fb := range fallbackPaths {
				if _, statErr := os.Stat(fb); statErr == nil {
					path = fb
					err = nil
					break
				}
			}
		}

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
					status.Details = "GitHub CLI connecté"
				} else {
					status.AuthStatus = "Not Authenticated"
					status.Details = "Exécutez 'gh auth login'"
				}
			case "linear":
				out, aErr := r.runCommand(ctx, repoPath, path, "auth", "whoami")
				if aErr == nil && strings.Contains(out, "Workspace:") {
					status.AuthStatus = "Authenticated"
					status.Details = "Linear connecté"
				} else {
					status.AuthStatus = "Not Authenticated"
					status.Details = "Exécutez 'linear auth login'"
				}
			case "git":
				status.AuthStatus = "Ready"
				status.Details = "Git disponible"
			case "agy":
				status.AuthStatus = "Ready"
				status.Details = "Antigravity CLI Agent prêt"
			case "vibe":
				status.AuthStatus = "Ready"
				status.Details = "Mistral Vibe CLI Agent prêt"
			case "claude":
				status.AuthStatus = "Ready"
				status.Details = "Claude Code CLI Agent prêt"
			}
		} else {
			status.AuthStatus = "Not Installed"
			status.Details = "Binaire introuvable dans le PATH"
		}

		results = append(results, status)
	}

	return results
}

// Linear JSON structures
type LinearIssueNode struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Priority    int    `json:"priority"`
	State       struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Assignee *struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"assignee"`
	Labels *struct {
		Nodes []struct {
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
	if teamKey == "" {
		teamKey = "FRE"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	linearPath, err := exec.LookPath("linear")
	if err != nil {
		linearPath = "/opt/homebrew/bin/linear"
	}

	output, err := r.runCommand(ctx, "", linearPath, "issue", "query", "--team", teamKey, "--json")
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

func (r *Runner) SyncFromGithub(repo string, repoPath string) ([]models.Task, error) {
	if repo == "" {
		repo = "sebastienferry/fretzee-studio"
	}
	if repoPath == "" {
		repoPath = "/Users/sferry/Sources/fretzee-studio"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		ghPath = "/opt/homebrew/bin/gh"
	}

	output, err := r.runCommand(ctx, repoPath, ghPath, "issue", "list", "-R", repo, "--limit", "100", "--json", "number,title,labels,body,url,state,assignees")
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
	if repo == "" {
		repo = "sebastienferry/fretzee-studio"
	}
	if repoPath == "" {
		repoPath = "/Users/sferry/Sources/fretzee-studio"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		ghPath = "/opt/homebrew/bin/gh"
	}

	args := []string{"issue", "create", "-R", repo, "--title", title}
	if description != "" {
		args = append(args, "--body", description)
	} else {
		args = append(args, "--body", "")
	}
	for _, l := range labels {
		args = append(args, "--label", l)
	}

	output, err := r.runCommand(ctx, repoPath, ghPath, args...)
	if err != nil {
		return nil, fmt.Errorf("gh issue create failed: %w (output: %s)", err, output)
	}

	// Output is typically the issue URL: https://github.com/owner/repo/issues/123
	issueURL := strings.TrimSpace(output)
	parts := strings.Split(issueURL, "/")
	issueNum := parts[len(parts)-1]
	key := fmt.Sprintf("#%s", issueNum)
	id := fmt.Sprintf("gh-%s", issueNum)

	now := time.Now()
	return &models.Task{
		ID:          id,
		Key:         key,
		Title:       title,
		Description: description,
		Status:      models.StatusBacklog,
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
	if repo == "" {
		repo = "sebastienferry/fretzee-studio"
	}
	if repoPath == "" {
		repoPath = "/Users/sferry/Sources/fretzee-studio"
	}

	num, err := cleanGithubIssueNum(keyOrNumber)
	if err != nil {
		return fmt.Errorf("invalid github issue number: %s", keyOrNumber)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		ghPath = "/opt/homebrew/bin/gh"
	}

	if status == models.StatusDone {
		_, err = r.runCommand(ctx, repoPath, ghPath, "issue", "close", strconv.Itoa(num), "-R", repo)
	} else {
		_, err = r.runCommand(ctx, repoPath, ghPath, "issue", "reopen", strconv.Itoa(num), "-R", repo)
	}
	return err
}

func (r *Runner) UpdateGithubIssue(repo string, repoPath string, keyOrNumber string, title *string, description *string, status *models.Status, labels []string, removedLabels []string) error {
	if repo == "" {
		repo = "sebastienferry/fretzee-studio"
	}
	if repoPath == "" {
		repoPath = "/Users/sferry/Sources/fretzee-studio"
	}

	num, err := cleanGithubIssueNum(keyOrNumber)
	if err != nil {
		return fmt.Errorf("invalid github issue number: %s", keyOrNumber)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		ghPath = "/opt/homebrew/bin/gh"
	}

	// 1. Update State (close / reopen)
	if status != nil {
		if *status == models.StatusFinished || *status == models.StatusDone || *status == models.StatusToClose {
			output, err := r.runCommand(ctx, repoPath, ghPath, "issue", "close", strconv.Itoa(num), "-R", repo)
			if err != nil {
				log.Printf("[CLI] gh issue close %d failed: %v (output: %s)", num, err, output)
			} else {
				log.Printf("[CLI] Closed GitHub issue #%d in %s", num, repo)
			}
		} else {
			output, err := r.runCommand(ctx, repoPath, ghPath, "issue", "reopen", strconv.Itoa(num), "-R", repo)
			if err != nil {
				log.Printf("[CLI] gh issue reopen %d failed: %v (output: %s)", num, err, output)
			} else {
				log.Printf("[CLI] Reopened GitHub issue #%d in %s", num, repo)
			}
		}
	}

	// 2. Edit title, body & labels
	args := []string{"issue", "edit", strconv.Itoa(num), "-R", repo}
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
	if len(labels) > 0 {
		var activeStage string
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
				if st != activeStage {
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

	// Remove explicitly removed labels
	for _, rl := range removedLabels {
		cleanRl := strings.TrimPrefix(strings.ToLower(rl), "#")
		if cleanRl != "" {
			args = append(args, "--remove-label", cleanRl)
			needEdit = true
		}
	}

	if needEdit {
		output, err := r.runCommand(ctx, repoPath, ghPath, args...)
		if err != nil {
			log.Printf("[CLI] gh issue edit #%d failed: %v (output: %s)", num, err, output)
		} else {
			log.Printf("[CLI] Successfully updated GitHub issue #%d in %s", num, repo)
		}
	}

	return nil
}

func (r *Runner) AddIssueComment(source string, repo string, repoPath string, key string, body string) error {
	if body == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if source == "linear" || strings.HasPrefix(key, "FRE-") {
		linPath, err := exec.LookPath("linear")
		if err != nil {
			linPath = "/opt/homebrew/bin/linear"
		}
		output, err := r.runCommand(ctx, repoPath, linPath, "issue", "comment", "add", key, "--body", body)
		if err != nil {
			log.Printf("[CLI] linear issue comment add %s failed: %v (output: %s)", key, err, output)
		}
		return err

	} else if source == "github" || strings.HasPrefix(key, "#") || strings.HasPrefix(key, "GH-#") || strings.HasPrefix(key, "gh-") {
		ghPath, err := exec.LookPath("gh")
		if err != nil {
			ghPath = "/opt/homebrew/bin/gh"
		}
		if repo == "" {
			repo = "sebastienferry/fretzee-studio"
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

			output, err := r.runCommand(ctx, repoPath, ghPath, "issue", "comment", cleanNum, "-R", repo, "--body-file", tmpPath)
			if err == nil {
				log.Printf("[CLI] Comment added to GitHub %s: %s", key, strings.TrimSpace(output))
				return nil
			}
			log.Printf("[CLI] gh issue comment --body-file failed: %v (output: %s), trying direct --body flag...", err, output)
		}

		output, err := r.runCommand(ctx, repoPath, ghPath, "issue", "comment", cleanNum, "-R", repo, "--body", body)
		if err != nil {
			log.Printf("[CLI] gh issue comment %s failed: %v (output: %s)", key, err, output)
		}
		return err
	}
	return nil
}

func (r *Runner) RunAI(settings *models.Settings, skillID string, task *models.Task, customPrompt string) (string, []string, error) {
	repoDir := ""
	if settings != nil && settings.RepoPath != "" {
		repoDir = strings.TrimSpace(settings.RepoPath)
	}
	if repoDir == "" {
		repoDir = "/Users/sferry/Sources/fretzee-studio"
	}
	repoDir = filepath.Clean(repoDir)
	if abs, err := filepath.Abs(repoDir); err == nil {
		repoDir = abs
	}

	var steps []string
	if stat, err := os.Stat(repoDir); err != nil || !stat.IsDir() {
		cwd, _ := os.Getwd()
		steps = append(steps, fmt.Sprintf("⚠️ Répertoire projet '%s' introuvable, repli sur : %s", repoDir, cwd))
		repoDir = cwd
	} else {
		steps = append(steps, fmt.Sprintf("📁 Dossier du projet analysé (CWD) : %s", repoDir))
	}

	var promptTemplate string
	switch skillID {
	case "clarify":
		promptTemplate = settings.PromptClarify
		if promptTemplate == "" {
			promptTemplate = "Tu es l'agent de cadrage technique pour Taskacao. Analyse la tâche suivante :\nClé : {issueKey}\nTitre : {issueTitle}\nDescription : {issueDesc}\n\nIdentifie les ambiguïtés, les dépendances critiques, et formule 3 à 5 questions d'alignement précises et concises à destination de l'équipe produit / tech pour cadrer le développement."
		}
	case "specify":
		promptTemplate = settings.PromptSpecify
		if promptTemplate == "" {
			promptTemplate = `Tu es le Product Owner & Architecte technique pour Taskacao. Rédige une spécification technique Speckit complète pour la tâche :
Clé : {issueKey}
Titre : {issueTitle}
Description : {issueDesc}
Branche Git cible : {branchName}
Dossier du projet : {repoPath}

Contenu attendu :
1. Contexte & User Stories
2. Architecture et composants cibles (fichiers à créer / modifier)
3. Critères d'acceptation détaillés (Scénarios Given / When / Then)
4. Plan de tests et de validation.`
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

	switch provider {
	case "agy":
		agyPath, err := exec.LookPath("agy")
		if err != nil {
			agyPath = "/Users/sferry/.local/bin/agy"
		}
		steps = append(steps, fmt.Sprintf("Exécution de : agy -p \"...\" dans %s", filepath.Base(repoDir)))
		output, execErr = r.runCommand(ctx, repoDir, agyPath, "-p", finalPrompt, "--dangerously-skip-permissions")

	case "vibe":
		vibePath, err := exec.LookPath("vibe")
		if err != nil {
			vibePath = "/Users/sferry/.local/bin/vibe"
		}
		steps = append(steps, fmt.Sprintf("Exécution de : vibe -p \"...\" dans %s", filepath.Base(repoDir)))
		output, execErr = r.runCommand(ctx, repoDir, vibePath, "-p", finalPrompt, "--auto-approve")

	case "claude":
		claudePath, err := exec.LookPath("claude")
		if err != nil {
			claudePath = "claude"
		}
		steps = append(steps, fmt.Sprintf("Exécution de : claude -p \"...\" dans %s", filepath.Base(repoDir)))
		output, execErr = r.runCommand(ctx, repoDir, claudePath, "-p", finalPrompt)

	default: // Custom command
		cmdStr := settings.AICommandTemplate
		if cmdStr == "" {
			cmdStr = "agy -p \"{prompt}\""
		}
		cmdToRun := strings.ReplaceAll(cmdStr, "{prompt}", finalPrompt)
		steps = append(steps, fmt.Sprintf("Exécution de la commande personnalisée dans %s", filepath.Base(repoDir)))
		output, execErr = r.runCommand(ctx, repoDir, "sh", "-c", cmdToRun)
	}

	if execErr != nil {
		steps = append(steps, fmt.Sprintf("⚠️ Erreur d'exécution : %v", execErr))
		return fmt.Sprintf("### ⚠️ Erreur lors de l'exécution de la commande IA (%s)\n\n```text\n%s\n```\n\n*Vérifiez que le binaire '%s' est bien accessible et authentifié.*", provider, output, provider), steps, nil
	}

	steps = append(steps, "✅ Réponse générée par le modèle IA avec succès")
	return output, steps, nil
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

