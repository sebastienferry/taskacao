package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tasks/internal/db"
	"tasks/internal/models"
)

type Handler struct {
	db *db.DB
}

func NewHandler(database *db.DB) *Handler {
	return &Handler{db: database}
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "fretzee-tasks-api"})
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

func (h *Handler) HandleSyncAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	settings, _ := h.db.GetSettings()
	teamKey := "FRE"
	repo := "sebastienferry/fretzee-studio"
	repoPath := "/Users/sferry/Sources/fretzee-studio"
	if settings != nil {
		if settings.LinearTeam != "" {
			teamKey = settings.LinearTeam
		}
		if settings.GithubRepo != "" {
			repo = settings.GithubRepo
		}
		if settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}

	var totalSynced int
	var syncMsgs []string

	linearTasks, lErr := h.db.GetRunner().SyncFromLinear(teamKey)
	if lErr == nil {
		_ = h.db.ImportOrUpdateTasks(linearTasks)
		totalSynced += len(linearTasks)
		syncMsgs = append(syncMsgs, fmt.Sprintf("%d Linear", len(linearTasks)))
	}

	ghTasks, gErr := h.db.GetRunner().SyncFromGithub(repo, repoPath)
	if gErr == nil {
		_ = h.db.ImportOrUpdateTasks(ghTasks)
		totalSynced += len(ghTasks)
		syncMsgs = append(syncMsgs, fmt.Sprintf("%d GitHub", len(ghTasks)))
	}

	allTasks, _ := h.db.GetTasks("", "", "", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("%d issues synchronisées (%s)", totalSynced, strings.Join(syncMsgs, " + ")),
		"count":   totalSynced,
		"tasks":   allTasks,
	})
}

func (h *Handler) HandleSyncLinear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	settings, _ := h.db.GetSettings()
	teamKey := "FRE"
	if settings != nil && settings.LinearTeam != "" {
		teamKey = settings.LinearTeam
	}

	tasks, err := h.db.GetRunner().SyncFromLinear(teamKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Linear sync failed: "+err.Error())
		return
	}

	if err := h.db.ImportOrUpdateTasks(tasks); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to persist synced tasks: "+err.Error())
		return
	}

	allTasks, _ := h.db.GetTasks("", "", "", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("%d issues synchronisées depuis Linear (Team: %s)", len(tasks), teamKey),
		"count":   len(tasks),
		"tasks":   allTasks,
	})
}

func (h *Handler) HandleSyncGithub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	settings, _ := h.db.GetSettings()
	repo := "sebastienferry/fretzee-studio"
	repoPath := "/Users/sferry/Sources/fretzee-studio"
	if settings != nil {
		if settings.GithubRepo != "" {
			repo = settings.GithubRepo
		}
		if settings.RepoPath != "" {
			repoPath = settings.RepoPath
		}
	}

	tasks, err := h.db.GetRunner().SyncFromGithub(repo, repoPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GitHub sync failed: "+err.Error())
		return
	}

	if err := h.db.ImportOrUpdateTasks(tasks); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to persist synced GitHub issues: "+err.Error())
		return
	}

	allTasks, _ := h.db.GetTasks("", "", "", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("%d issues synchronisées depuis GitHub CLI (%s)", len(tasks), repo),
		"count":   len(tasks),
		"tasks":   allTasks,
	})
}

func (h *Handler) HandleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	skills := h.db.GetAvailableSkills()
	writeJSON(w, http.StatusOK, skills)
}

func (h *Handler) HandleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("q")
		status := r.URL.Query().Get("status")
		priority := r.URL.Query().Get("priority")
		label := r.URL.Query().Get("label")

		tasks, err := h.db.GetTasks(q, status, priority, label)
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

func (h *Handler) HandleTaskDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "Task ID is required")
		return
	}
	id := parts[0]

	// Sub-action: /api/tasks/{id}/move
	if len(parts) >= 2 && parts[1] == "move" && (r.Method == http.MethodPatch || r.Method == http.MethodPost) {
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
	if len(parts) >= 2 && parts[1] == "run-skill" && r.Method == http.MethodPost {
		var req models.RunSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid skill request: "+err.Error())
			return
		}
		if req.SkillID == "" {
			writeError(w, http.StatusBadRequest, "Skill ID is required")
			return
		}

		task, activity, err := h.db.RunSkillOnTask(id, req.SkillID, req.Prompt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, models.RunSkillResponse{
			Task:     *task,
			Activity: *activity,
			Message:  "Skill " + req.SkillID + " executed successfully",
		})
		return
	}

	// Sub-action: /api/tasks/{id}/activities
	if len(parts) >= 2 && parts[1] == "activities" && r.Method == http.MethodGet {
		activities, err := h.db.GetTaskActivities(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, activities)
		return
	}

	// Sub-action: /api/tasks/{id}/comment
	if len(parts) >= 2 && parts[1] == "comment" && r.Method == http.MethodPost {
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
	if len(parts) >= 2 && parts[1] == "convert" && r.Method == http.MethodPost {
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

func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := h.db.GetSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, settings)

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
		writeJSON(w, http.StatusOK, saved)

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

	tasks, _ := h.db.GetTasks("", "", "", "")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Database reset & demo tasks reseeded successfully",
		"tasks":   tasks,
	})
}
