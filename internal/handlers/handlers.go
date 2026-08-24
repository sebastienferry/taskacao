package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"tasks/internal/db"
	"tasks/internal/models"
	"tasks/internal/runner"
	"tasks/internal/terminal"
)

type Handler struct {
	db          *db.DB
	terminalMgr *terminal.Manager
}

func NewHandler(database *db.DB) *Handler {
	return &Handler{
		db:          database,
		terminalMgr: terminal.NewManager(),
	}
}

func (h *Handler) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskacao-api"})
}

func (h *Handler) HandleCliStatus(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.db.GetSettings()
	repoPath := ""
	if settings != nil {
		repoPath = settings.RepoPath
	}
	statuses := h.db.GetRunner().CheckCliTools(repoPath)
	writeJSON(w, http.StatusOK, statuses)
}

func (h *Handler) HandleGitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	target := r.URL.Query().Get("path")
	if target == "" {
		target = r.URL.Query().Get("projectId")
	}
	if target == "" {
		target = r.URL.Query().Get("repoPath")
	}

	status, err := h.db.GetGitStatus(target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) HandleGitBranches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	target := r.URL.Query().Get("path")
	if target == "" {
		target = r.URL.Query().Get("projectId")
	}
	if target == "" {
		target = r.URL.Query().Get("repoPath")
	}

	info, err := h.db.GetGitBranches(target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) HandleGitCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Branch    string `json:"branch"`
		Path      string `json:"path"`
		ProjectID string `json:"projectId"`
		Create    bool   `json:"create"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	target := req.Path
	if target == "" {
		target = req.ProjectID
	}
	if target == "" {
		target = r.URL.Query().Get("projectId")
	}

	status, err := h.db.SwitchGitBranch(target, req.Branch, req.Create)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Bascule effectuée sur la branche '%s'", req.Branch),
		"status":  status,
		"branch":  req.Branch,
	})
}

func (h *Handler) HandleGitCleanBranches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path      string `json:"path"`
		ProjectID string `json:"projectId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	target := req.Path
	if target == "" {
		target = req.ProjectID
	}
	if target == "" {
		target = r.URL.Query().Get("projectId")
	}

	res, err := h.db.CleanAllLocalBranches(target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) HandleGitDeleteBranch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Branch       string `json:"branch"`
		Path         string `json:"path"`
		ProjectID    string `json:"projectId"`
		DeleteRemote bool   `json:"deleteRemote"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	branchName := req.Branch
	if branchName == "" {
		branchName = r.URL.Query().Get("branch")
	}
	if branchName == "" {
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) > 0 {
			last := pathParts[len(pathParts)-1]
			if last != "delete" && last != "branches" {
				branchName = last
			}
		}
	}

	if branchName == "" {
		writeError(w, http.StatusBadRequest, "Nom de branche requis")
		return
	}

	target := req.Path
	if target == "" {
		target = req.ProjectID
	}
	if target == "" {
		target = r.URL.Query().Get("projectId")
	}

	err := h.db.DeleteGitBranch(target, branchName, req.DeleteRemote)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"branch":  branchName,
		"message": fmt.Sprintf("Branche '%s' supprimée avec succès.", branchName),
	})
}

func (h *Handler) HandleSyncAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	activity, err := h.db.EnqueueSync("all", "", req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Synchronisation globale ajoutée à la file d'attente",
		"activity": activity,
	})
}

func (h *Handler) HandleSyncLinear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Team      string `json:"team"`
		ProjectID string `json:"projectId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	activity, err := h.db.EnqueueSync("linear", req.Team, req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Synchronisation Linear ajoutée à la file d'attente",
		"activity": activity,
	})
}

func (h *Handler) HandleSyncGithub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Repo      string `json:"repo"`
		ProjectID string `json:"projectId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	activity, err := h.db.EnqueueSync("github", req.Repo, req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Synchronisation GitHub ajoutée à la file d'attente",
		"activity": activity,
	})
}

func (h *Handler) HandleSyncJira(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ProjectKey string `json:"projectKey"`
		ProjectID  string `json:"projectId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	activity, err := h.db.EnqueueSync("jira", req.ProjectKey, req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Synchronisation Jira ajoutée à la file d'attente",
		"activity": activity,
	})
}

// HandleSpecFrameworkStatus reports whether GitHub Spec Kit / OpenSpec are
// installed on the host and initialized in a project working directory.
// GET /api/spec-framework/status?projectId=…&repoPath=…&framework=speckit|openspec
func (h *Handler) HandleSpecFrameworkStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	target := r.URL.Query().Get("projectId")
	if target == "" {
		target = r.URL.Query().Get("repoPath")
	}

	statuses := h.db.GetSpecFrameworkStatus(target, r.URL.Query().Get("framework"))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"frameworks": statuses,
	})
}

// HandleSpecFrameworkInstall bootstraps a Spec-Driven Design toolchain
// (GitHub Spec Kit or OpenSpec) in a project working directory.
// POST /api/spec-framework/install
func (h *Handler) HandleSpecFrameworkInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.SpecFrameworkInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête JSON invalide: "+err.Error())
		return
	}

	res, err := h.db.InstallSpecFramework(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// The install itself may have failed while the request was well-formed; the
	// result carries the per-command detail, so return 200 with installed=false
	// rather than an opaque 500.
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) HandleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	skills := h.db.GetAvailableSkills()
	writeJSON(w, http.StatusOK, skills)
}

func (h *Handler) HandleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := h.db.GetProjects()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, projects)

	case http.MethodPost:
		var req models.CreateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Payload invalide: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(w, http.StatusBadRequest, "Le nom du projet est obligatoire")
			return
		}

		project, err := h.db.CreateProject(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, project)

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handler) HandleProjectDetail(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	rawPath = strings.Trim(rawPath, "/")
	if rawPath == "" {
		writeError(w, http.StatusBadRequest, "ID du projet obligatoire")
		return
	}

	parts := strings.Split(rawPath, "/")
	id, err := url.PathUnescape(parts[0])
	if err != nil {
		id = parts[0]
	}

	// Sub-action: /api/projects/{id}/skills-status, /api/projects/{id}/skills, or query repoPath
	if (len(parts) >= 2 && (parts[1] == "skills-status" || parts[1] == "skills")) || id == "skills-status" || id == "skills" {
		target := id
		if len(parts) >= 2 {
			target = id
		} else if qPath := r.URL.Query().Get("repoPath"); qPath != "" {
			target = qPath
		}
		status, err := h.db.GetProjectSkillsStatus(target)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	// Sub-action: /api/projects/{id}/install-skills or /api/projects/install-skills
	if ((len(parts) >= 2 && parts[1] == "install-skills") || id == "install-skills") && r.Method == http.MethodPost {
		target := id
		var payload struct {
			RepoPath          string `json:"repoPath"`
			ProjectID         string `json:"projectId"`
			SpecFramework     string `json:"specFramework"`
			AIProvider        string `json:"aiProvider"`
			AICommandTemplate string `json:"aiCommandTemplate"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload.RepoPath != "" {
			target = payload.RepoPath
		} else if payload.ProjectID != "" {
			target = payload.ProjectID
		}

		status, err := h.db.InstallProjectSkills(target, payload.SpecFramework, payload.AIProvider, payload.AICommandTemplate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Skills IA installées avec succès dans le projet",
			"status":  status,
		})
		return
	}

	// Sub-action: /api/projects/{id}/daily-digest
	//   GET  → compute the task sections and merge any stored agenda
	//   POST → same, then persist; with {"enrich": true} also runs the agenda pass
	if len(parts) >= 2 && parts[1] == "daily-digest" {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("history") == "1" {
				dates, err := h.db.ListDigestDates(id, 30)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				writeJSON(w, http.StatusOK, map[string]interface{}{"dates": dates})
				return
			}
			digest, err := h.db.ComputeDailyDigest(id, r.URL.Query().Get("date"), r.URL.Query().Get("assignee"))
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, digest)
			return

		case http.MethodPost:
			var payload models.DailyDigestRequest
			_ = json.NewDecoder(r.Body).Decode(&payload)

			if payload.Enrich {
				// Runs the agent; can take a while, and reports its own failure
				// inside the digest rather than as an HTTP error.
				digest, err := h.db.EnqueueDigestAgenda(id, payload.Date, payload.Assignee)
				if err != nil {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				writeJSON(w, http.StatusOK, digest)
				return
			}

			digest, err := h.db.ComputeDailyDigest(id, payload.Date, payload.Assignee)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := h.db.SaveDailyDigest(digest); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, digest)
			return

		default:
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
	}

	// Sub-action: /api/projects/{id}/spec-framework-status
	if len(parts) >= 2 && parts[1] == "spec-framework-status" {
		statuses := h.db.GetSpecFrameworkStatus(id, r.URL.Query().Get("framework"))
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"frameworks": statuses,
		})
		return
	}

	// Sub-action: /api/projects/{id}/install-spec-framework
	if len(parts) >= 2 && parts[1] == "install-spec-framework" && r.Method == http.MethodPost {
		var payload models.SpecFrameworkInstallRequest
		_ = json.NewDecoder(r.Body).Decode(&payload)
		payload.ProjectID = id

		res, err := h.db.InstallSpecFramework(payload)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, res)
		return
	}

	// Sub-action: /api/projects/{id}/init-git or /api/projects/init-git
	if ((len(parts) >= 2 && parts[1] == "init-git") || id == "init-git") && r.Method == http.MethodPost {
		target := id
		var payload struct {
			RepoPath  string `json:"repoPath"`
			ProjectID string `json:"projectId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload.RepoPath != "" {
			target = payload.RepoPath
		} else if payload.ProjectID != "" {
			target = payload.ProjectID
		}

		res, err := h.db.InitProjectGit(target)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, res)
		return
	}

	// Sub-action: /api/projects/{id}/detect-statuses or /api/projects/detect-statuses
	if (len(parts) >= 2 && parts[1] == "detect-statuses") || id == "detect-statuses" {
		target := id
		if len(parts) >= 2 {
			target = id
		}
		tracker := r.URL.Query().Get("tracker")
		team := r.URL.Query().Get("team")
		repo := r.URL.Query().Get("repo")

		if r.Method == http.MethodPost {
			var body struct {
				ProjectID    string `json:"projectId"`
				IssueTracker string `json:"issueTracker"`
				LinearTeam   string `json:"linearTeam"`
				GithubRepo   string `json:"githubRepo"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				if body.ProjectID != "" {
					target = body.ProjectID
				}
				if body.IssueTracker != "" {
					tracker = body.IssueTracker
				}
				if body.LinearTeam != "" {
					team = body.LinearTeam
				}
				if body.GithubRepo != "" {
					repo = body.GithubRepo
				}
			}
		}

		statuses, err := h.db.DetectTrackerStatuses(target, tracker, team, repo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, statuses)
		return
	}

	switch r.Method {
	case http.MethodGet:
		project, err := h.db.GetProjectByID(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if project == nil {
			writeError(w, http.StatusNotFound, "Projet non trouvé")
			return
		}
		writeJSON(w, http.StatusOK, project)

	case http.MethodPut, http.MethodPatch:
		var req models.UpdateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Payload invalide: "+err.Error())
			return
		}

		project, err := h.db.UpdateProject(id, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, project)

	case http.MethodDelete:
		if err := h.db.DeleteProject(id); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Projet supprimé avec succès"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handler) HandleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("q")
		status := r.URL.Query().Get("status")
		priority := r.URL.Query().Get("priority")
		label := r.URL.Query().Get("label")
		projectID := r.URL.Query().Get("projectId")
		sprint := r.URL.Query().Get("sprint")
		team := r.URL.Query().Get("team")

		tasks, err := h.db.GetTasks(q, status, priority, label, projectID, sprint, team)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)

	case http.MethodPost:
		var req models.CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			writeError(w, http.StatusBadRequest, "Task title is required")
			return
		}

		task, err := h.db.CreateTask(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleTaskFacets serves the distinct sprint and team values present on the
// board, so the UI shows those filters only for trackers that feed them.
func (h *Handler) HandleTaskFacets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	facets, err := h.db.GetTaskFacets(r.URL.Query().Get("projectId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, facets)
}

func (h *Handler) HandleTaskDetail(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	rawPath = strings.Trim(rawPath, "/")
	if rawPath == "" {
		writeError(w, http.StatusBadRequest, "Task ID is required")
		return
	}

	var subAction string
	var id string

	switch {
	case strings.HasSuffix(rawPath, "/run-skill"):
		subAction = "run-skill"
		id = strings.TrimSuffix(rawPath, "/run-skill")
	case strings.HasSuffix(rawPath, "/move"):
		subAction = "move"
		id = strings.TrimSuffix(rawPath, "/move")
	case strings.HasSuffix(rawPath, "/activities"):
		subAction = "activities"
		id = strings.TrimSuffix(rawPath, "/activities")
	case strings.HasSuffix(rawPath, "/comment"):
		subAction = "comment"
		id = strings.TrimSuffix(rawPath, "/comment")
	case strings.HasSuffix(rawPath, "/convert"):
		subAction = "convert"
		id = strings.TrimSuffix(rawPath, "/convert")
	case strings.HasSuffix(rawPath, "/git-diff"):
		subAction = "git-diff"
		id = strings.TrimSuffix(rawPath, "/git-diff")
	case strings.HasSuffix(rawPath, "/diff"):
		subAction = "git-diff"
		id = strings.TrimSuffix(rawPath, "/diff")
	case strings.HasSuffix(rawPath, "/checkout-branch"):
		subAction = "checkout-branch"
		id = strings.TrimSuffix(rawPath, "/checkout-branch")
	case strings.HasSuffix(rawPath, "/checkout"):
		subAction = "checkout-branch"
		id = strings.TrimSuffix(rawPath, "/checkout")
	case strings.HasSuffix(rawPath, "/worktree"):
		subAction = "worktree"
		id = strings.TrimSuffix(rawPath, "/worktree")
	case strings.HasSuffix(rawPath, "/messages"):
		subAction = "messages"
		id = strings.TrimSuffix(rawPath, "/messages")
	case strings.HasSuffix(rawPath, "/chat/stream"):
		subAction = "chat"
		id = strings.TrimSuffix(rawPath, "/chat/stream")
	case strings.HasSuffix(rawPath, "/chat"):
		subAction = "chat"
		id = strings.TrimSuffix(rawPath, "/chat")
	default:
		id = rawPath
	}

	if unescaped, err := url.PathUnescape(id); err == nil && unescaped != "" {
		id = unescaped
	}
	id = strings.TrimSpace(id)

	// Sub-action: /api/tasks/{id}/move
	if subAction == "move" && (r.Method == http.MethodPatch || r.Method == http.MethodPost) {
		var req models.MoveTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid move payload: "+err.Error())
			return
		}
		task, err := h.db.MoveTask(id, req.Status, req.Position)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, task)
		return
	}

	// Sub-action: /api/tasks/{id}/run-skill
	if subAction == "run-skill" && r.Method == http.MethodPost {
		var req models.RunSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid skill request: "+err.Error())
			return
		}
		if req.SkillID == "" {
			writeError(w, http.StatusBadRequest, "Skill ID is required")
			return
		}

		task, activity, err := h.db.EnqueueSkillOnTask(id, req.SkillID, req.Prompt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, models.RunSkillResponse{
			Task:     *task,
			Activity: *activity,
			Message:  "Skill " + req.SkillID + " ajoutée à la file d'exécution",
		})
		return
	}

	// Sub-action: /api/tasks/{id}/activities
	if subAction == "activities" && r.Method == http.MethodGet {
		activities, err := h.db.GetTaskActivities(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, activities)
		return
	}

	// Sub-action: /api/tasks/{id}/comment
	if subAction == "comment" && r.Method == http.MethodPost {
		var req struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid comment payload: "+err.Error())
			return
		}
		if req.Body == "" {
			writeError(w, http.StatusBadRequest, "Comment body is required")
			return
		}
		if err := h.db.AddTaskComment(id, req.Body); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	// Sub-action: /api/tasks/{id}/convert
	if subAction == "convert" && r.Method == http.MethodPost {
		var req models.ConvertTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid convert payload: "+err.Error())
			return
		}
		if req.Target == "" {
			writeError(w, http.StatusBadRequest, "Target tracker is required ('linear' or 'github')")
			return
		}
		task, err := h.db.ConvertTaskToRemote(id, req.Target)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, task)
		return
	}

	// Sub-action: /api/tasks/{id}/git-diff
	if subAction == "git-diff" && r.Method == http.MethodGet {
		diffRes, err := h.db.GetTaskGitDiff(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, diffRes)
		return
	}

	// Sub-action: /api/tasks/{id}/checkout-branch
	if subAction == "checkout-branch" && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
		task, err := h.db.GetTaskByID(id)
		if err != nil || task == nil {
			writeError(w, http.StatusNotFound, "Tâche non trouvée")
			return
		}
		repoPath := h.db.ResolveTaskRepoPath(task)
		branch, err := h.db.EnsureTaskGitBranch(repoPath, task)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":  fmt.Sprintf("Bascule effectuée avec succès sur la branche '%s'", branch),
			"branch":   branch,
			"repoPath": repoPath,
		})
		return
	}

	// Sub-action: /api/tasks/{id}/worktree
	if subAction == "worktree" {
		if r.Method == http.MethodGet {
			info, err := h.db.GetTaskWorktreeInfo(id)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, info)
			return
		} else if r.Method == http.MethodDelete {
			task, err := h.db.GetTaskByID(id)
			if err != nil || task == nil {
				writeError(w, http.StatusNotFound, "Tâche non trouvée")
				return
			}
			repoPath := h.db.ResolveTaskRepoPath(task)
			if err := h.db.RemoveTaskWorktree(repoPath, task.Key); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"message": "Worktree supprimé avec succès"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		task, err := h.db.GetTaskByID(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if task == nil {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeJSON(w, http.StatusOK, task)

	case http.MethodPut, http.MethodPatch:
		var req models.UpdateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		task, err := h.db.UpdateTask(id, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, task)

	case http.MethodDelete:
		if err := h.db.DeleteTask(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Task deleted successfully"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// maskJiraToken strips the Jira API token from anything sent to the client and
// replaces it with two flags: whether a token is configured at all, and whether
// it comes from the environment rather than the database. An empty incoming
// token means "keep the stored one", so the UI can leave its field blank.
func maskJiraToken(s *models.Settings) *models.Settings {
	if s == nil {
		return nil
	}
	copied := *s
	fromEnv := runner.JiraTokenFromEnv() != ""
	copied.JiraAPITokenSet = fromEnv || strings.TrimSpace(copied.JiraAPIToken) != ""
	copied.JiraAPITokenFromEnv = fromEnv
	copied.JiraAPIToken = ""
	return &copied
}

func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := h.db.GetSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, maskJiraToken(settings))

	case http.MethodPost, http.MethodPut:
		var req models.Settings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid settings payload: "+err.Error())
			return
		}
		saved, err := h.db.UpdateSettings(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, maskJiraToken(saved))

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handler) HandleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := h.db.SeedDemoData(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks, _ := h.db.GetTasks("", "", "", "", "", "", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Database reset & demo tasks reseeded successfully",
		"tasks":   tasks,
	})
}

func (h *Handler) HandleActivities(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projectID := r.URL.Query().Get("projectId")

		// Check for /api/activities/stats or general query
		if strings.HasSuffix(r.URL.Path, "/stats") {
			stats, err := h.db.GetActivityStats(projectID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, stats)
			return
		}

		status := r.URL.Query().Get("status")
		skillID := r.URL.Query().Get("skillId")
		taskID := r.URL.Query().Get("taskId")
		q := r.URL.Query().Get("q")
		limit := 100
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		activities, err := h.db.GetActivities(projectID, status, skillID, taskID, q, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, activities)

	case http.MethodDelete:
		count, err := h.db.ClearCompletedActivities()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("%d activités terminées supprimées", count),
			"count":   count,
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handler) HandleActivityDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/activities/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "Activity ID or action required")
		return
	}

	// Sub-action: /api/activities/stats
	if parts[0] == "stats" && r.Method == http.MethodGet {
		projectID := r.URL.Query().Get("projectId")
		stats, err := h.db.GetActivityStats(projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, stats)
		return
	}

	// Sub-action: /api/activities/clear
	if parts[0] == "clear" && (r.Method == http.MethodPost || r.Method == http.MethodDelete) {
		count, err := h.db.ClearCompletedActivities()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("%d activités terminées supprimées", count),
			"count":   count,
		})
		return
	}

	id := parts[0]

	// Sub-action: /api/activities/{id}/retry
	if len(parts) >= 2 && parts[1] == "retry" && r.Method == http.MethodPost {
		act, err := h.db.RetryActivity(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, act)
		return
	}

	// Sub-action: /api/activities/{id}/cancel
	if len(parts) >= 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		if err := h.db.CancelActivity(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Activité annulée avec succès"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		act, err := h.db.GetActivityByID(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if act == nil {
			writeError(w, http.StatusNotFound, "Activity not found")
			return
		}
		writeJSON(w, http.StatusOK, act)

	case http.MethodDelete:
		if err := h.db.DeleteActivity(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Activité supprimée avec succès"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleTerminalWs upgrades the connection to WebSocket and streams the interactive PTY session
func (h *Handler) HandleTerminalWs(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("taskId")
	sessionID := r.URL.Query().Get("sessionId")
	customCwd := r.URL.Query().Get("cwd")

	if sessionID == "" {
		if taskID != "" {
			sessionID = "task-" + taskID
		} else {
			sessionID = "global-workspace"
		}
	}

	workDir := customCwd
	envVars := make(map[string]string)

	if taskID != "" {
		task, _ := h.db.GetTaskByID(taskID)
		if task != nil {
			envVars["TASKACAO_TASK_ID"] = task.ID
			envVars["TASKACAO_TASK_KEY"] = task.Key
			envVars["TASKACAO_TASK_TITLE"] = task.Title
			envVars["TASKACAO_TASK_PROJECT"] = task.ProjectID

			baseRepo := h.db.ResolveTaskRepoPath(task)
			if baseRepo == "" {
				baseRepo = "."
			}
			if task.ProjectID != "" {
				if proj, _ := h.db.GetProjectByID(task.ProjectID); proj != nil {
					envVars["TASKACAO_PROJECT_NAME"] = proj.Name
					envVars["TASKACAO_GITHUB_REPO"] = proj.GithubRepo
					envVars["TASKACAO_LINEAR_TEAM"] = proj.LinearTeam
				}
			}

			// Ensure the task worktree, unless the project opted out: then the
			// shell simply opens in the clone, and TASKACAO_TASK_WORKTREE stays
			// unset so a script can tell the two situations apart.
			if baseRepo != "" {
				wtPath, branch, err := h.db.EnsureTaskWorktree(baseRepo, task)
				switch {
				case err == nil && wtPath != "" && wtPath != baseRepo:
					workDir = wtPath
					envVars["TASKACAO_TASK_WORKTREE"] = wtPath
					envVars["TASKACAO_TASK_BRANCH"] = branch
				case workDir == "":
					workDir = baseRepo
				}
			}
		}
	}

	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	h.terminalMgr.HandleWebSocket(w, r, sessionID, workDir, envVars)
}

// HandleTerminalSend allows sending command strings / keystrokes into a running terminal session
func (h *Handler) HandleTerminalSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		TaskID    string `json:"taskId"`
		SessionID string `json:"sessionId"`
		Input     string `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		if req.TaskID != "" {
			sessionID = "task-" + req.TaskID
		} else {
			sessionID = "global-workspace"
		}
	}

	if err := h.terminalMgr.SendInput(sessionID, req.Input); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Input sent successfully"})
}

// HandleTerminalReset terminates the running PTY session so a clean shell can spawn
func (h *Handler) HandleTerminalReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		TaskID    string `json:"taskId"`
		SessionID string `json:"sessionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		if req.TaskID != "" {
			sessionID = "task-" + req.TaskID
		} else {
			sessionID = "global-workspace"
		}
	}

	_ = h.terminalMgr.CloseSession(sessionID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Terminal session reset successfully"})
}

// HandleOpenEditor opens a workspace, worktree, or path in the user's code editor (default: code)
func (h *Handler) HandleOpenEditor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Path          string `json:"path"`
		TaskID        string `json:"taskId"`
		ProjectID     string `json:"projectId"`
		EditorCommand string `json:"editorCommand"`
	}

	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	settings, _ := h.db.GetSettings()
	editorCmd := req.EditorCommand
	if editorCmd == "" && settings != nil && settings.EditorCommand != "" {
		editorCmd = settings.EditorCommand
	}
	if editorCmd == "" {
		editorCmd = "code"
	}

	targetPath := req.Path
	if targetPath == "" && req.TaskID != "" {
		task, err := h.db.GetTaskByID(req.TaskID)
		if err == nil && task != nil {
			if task.WorktreePath != nil && *task.WorktreePath != "" {
				if _, statErr := os.Stat(*task.WorktreePath); statErr == nil {
					targetPath = *task.WorktreePath
				}
			}
			repoPath := h.db.ResolveTaskRepoPath(task)
			if repoPath == "" {
				repoPath = "."
			}

			if targetPath == "" {
				wtPath, _, wtErr := h.db.EnsureTaskWorktree(repoPath, task)
				if wtErr == nil && wtPath != "" {
					targetPath = wtPath
				}
			}
			if targetPath == "" {
				targetPath = repoPath
			}
		}
	}

	if targetPath == "" && req.ProjectID != "" {
		proj, err := h.db.GetProjectByID(req.ProjectID)
		if err == nil && proj != nil && proj.RepoPath != "" {
			targetPath = proj.RepoPath
		}
	}

	if targetPath == "" && settings != nil && settings.RepoPath != "" {
		targetPath = settings.RepoPath
	}

	if targetPath == "" {
		targetPath, _ = os.Getwd()
	}

	if err := h.db.GetRunner().OpenInEditor(editorCmd, targetPath); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to open '%s' in '%s': %v", targetPath, editorCmd, err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"path":    targetPath,
		"editor":  editorCmd,
		"message": fmt.Sprintf("Opened %s in %s", targetPath, editorCmd),
	})
}
