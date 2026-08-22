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

func normalizeLinearLabel(label string) string {
	switch strings.ToLower(label) {
	case "new":
		return "New"
	case "clarified":
		return "clarified"
	case "specified":
		return "specified"
	case "implemented":
		return "Implemented"
	case "reviewed":
		return "Reviewed"
	default:
		return label
	}
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

	args := []string{"issue", "create", "--team", teamKey, "--title", title, "-p", priorityNum, "--no-interactive"}
	if description != "" {
		args = append(args, "-d", description)
	}
	for _, l := range labels {
		if l != "" {
			args = append(args, "-l", normalizeLinearLabel(l))
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

	// Linear CLI output format:
	// Creating issue in FRE
	//
	// https://linear.app/fretzee/issue/FRE-74/test-dry-issue
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
	id := keyMatch
	if extURL != nil {
		id = *extURL
	}

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
	for _, l := range labels {
		if l != "" {
			args = append(args, "-l", normalizeLinearLabel(l))
		}
	}

	output, err := r.runCommand(ctx, "", linearPath, args...)
	if err != nil {
		log.Printf("[CLI] linear issue update %s failed: %v (output: %s)", issueKey, err, output)
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
			Key:         fmt.Sprintf("GH-#%d", item.Number),
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
	key := fmt.Sprintf("GH-#%s", issueNum)
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

func (r *Runner) UpdateGithubIssueState(repo string, repoPath string, keyOrNumber string, status models.Status) error {
	if repo == "" {
		repo = "sebastienferry/fretzee-studio"
	}
	if repoPath == "" {
		repoPath = "/Users/sferry/Sources/fretzee-studio"
	}

	cleanNum := strings.TrimPrefix(keyOrNumber, "GH-#")
	cleanNum = strings.TrimPrefix(cleanNum, "gh-")
	num, err := strconv.Atoi(cleanNum)
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

func (r *Runner) UpdateGithubIssue(repo string, repoPath string, keyOrNumber string, title *string, description *string, status *models.Status, labels []string) error {
	if repo == "" {
		repo = "sebastienferry/fretzee-studio"
	}
	if repoPath == "" {
		repoPath = "/Users/sferry/Sources/fretzee-studio"
	}

	cleanNum := strings.TrimPrefix(keyOrNumber, "GH-#")
	cleanNum = strings.TrimPrefix(cleanNum, "gh-")
	num, err := strconv.Atoi(cleanNum)
	if err != nil {
		return fmt.Errorf("invalid github issue number: %s", keyOrNumber)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		ghPath = "/opt/homebrew/bin/gh"
	}

	// Edit title, body & labels
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
	for _, l := range labels {
		args = append(args, "--add-label", l)
		needEdit = true
	}

	if needEdit {
		_, _ = r.runCommand(ctx, repoPath, ghPath, args...)
	}

	// State update (close / reopen)
	if status != nil {
		if *status == models.StatusDone {
			_, _ = r.runCommand(ctx, repoPath, ghPath, "issue", "close", strconv.Itoa(num), "-R", repo)
		} else {
			_, _ = r.runCommand(ctx, repoPath, ghPath, "issue", "reopen", strconv.Itoa(num), "-R", repo)
		}
	}
	return nil
}

func (r *Runner) AddIssueComment(source string, repo string, repoPath string, key string, body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if source == "linear" || strings.HasPrefix(key, "FRE-") {
		linearPath, err := exec.LookPath("linear")
		if err != nil {
			linearPath = "/opt/homebrew/bin/linear"
		}
		_, err = r.runCommand(ctx, "", linearPath, "issue", "comment", "add", key, "--body", body)
		return err
	} else if source == "github" || strings.HasPrefix(key, "GH-#") || strings.HasPrefix(key, "gh-") {
		ghPath, err := exec.LookPath("gh")
		if err != nil {
			ghPath = "/opt/homebrew/bin/gh"
		}
		if repo == "" {
			repo = "sebastienferry/fretzee-studio"
		}
		cleanNum := strings.TrimPrefix(key, "GH-#")
		cleanNum = strings.TrimPrefix(cleanNum, "gh-")
		_, err = r.runCommand(ctx, repoPath, ghPath, "issue", "comment", cleanNum, "-R", repo, "--body", body)
		return err
	}
	return nil
}

func (r *Runner) RunAI(settings *models.Settings, skillID string, task *models.Task, customPrompt string) (string, []string, error) {
	repoDir := settings.RepoPath
	if repoDir == "" {
		repoDir = "/Users/sferry/Sources/fretzee-studio"
	}
	if _, err := os.Stat(repoDir); err != nil {
		repoDir, _ = os.Getwd()
	}

	var promptTemplate string
	switch skillID {
	case "clarify":
		promptTemplate = settings.PromptClarify
		if promptTemplate == "" {
			promptTemplate = "Tu es l'agent de clarification technique pour Fretzee Studio. Analyse la tâche suivante :\nClé : {issueKey}\nTitre : {issueTitle}\nDescription : {issueDesc}\n\nIdentifie les ambiguïtés, les dépendances critiques, et formule 3 à 5 questions d'alignement précises et concises à destination de l'équipe produit / tech pour cadrer le développement."
		}
	case "specify":
		promptTemplate = settings.PromptSpecify
		if promptTemplate == "" {
			promptTemplate = "Tu es le Product Owner & Architecte technique pour Fretzee Studio. Rédige une spécification technique Speckit complète pour la tâche :\nClé : {issueKey}\nTitre : {issueTitle}\nDescription : {issueDesc}\n\nInclus :\n1. Objectifs & User Stories\n2. Architecture & Composants touchés (Backend Go / Frontend React)\n3. Contrat d'interface / API / Base de données\n4. Critères d'acceptation et plan de tests."
		}
	case "implement":
		promptTemplate = settings.PromptImplement
		if promptTemplate == "" {
			promptTemplate = "Tu es le développeur pour Fretzee Studio. Prépare le plan d'implémentation et la stratégie de modifications de code pour la tâche :\nClé : {issueKey}\nTitre : {issueTitle}\nDescription : {issueDesc}\nBranche : {branchName}\n\nIndique les fichiers à créer/éditer, les tests unitaires à lancer (go test ./...), et les règles de validation à respecter."
		}
	case "create_pr":
		promptTemplate = settings.PromptCreatePR
		if promptTemplate == "" {
			promptTemplate = "Tu es l'agent release pour Fretzee Studio. Rédige la Pull Request pour la tâche :\nClé : {issueKey}\nTitre : {issueTitle}\nDescription : {issueDesc}\nBranche : {branchName}\n\nFournis :\n1. Message de commit sémantique (ex: feat({issueKey}): description)\n2. Titre de la PR\n3. Description complète en Markdown avec le résumé des changements et la checklist de validation."
		}
	case "pick":
		promptTemplate = settings.PromptPick
		if promptTemplate == "" {
			promptTemplate = "Tu es le routeur d'orchestration pour Fretzee Studio. Analyse l'état de la tâche {issueKey} ({issueTitle}) et détermine la prochaine action requise dans le cycle SDLC."
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

	var steps []string
	steps = append(steps, fmt.Sprintf("Moteur IA sélectionné : %s", strings.ToUpper(provider)))
	steps = append(steps, fmt.Sprintf("Répertoire de travail : %s", repoDir))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
