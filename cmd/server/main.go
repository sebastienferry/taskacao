package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tasks/internal/db"
	"tasks/internal/handlers"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "tasks.db"
	}

	database, err := db.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal database error: %v", err)
	}
	defer database.Close()

	h := handlers.NewHandler(database)

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/health", h.HandleHealth)
	mux.HandleFunc("/api/cli-status", h.HandleCliStatus)
	mux.HandleFunc("/api/git-status", h.HandleGitStatus)
	mux.HandleFunc("/api/git/status", h.HandleGitStatus)
	mux.HandleFunc("/api/git/branches", h.HandleGitBranches)
	mux.HandleFunc("/api/git-branches", h.HandleGitBranches)
	mux.HandleFunc("/api/git/branches/clean", h.HandleGitCleanBranches)
	mux.HandleFunc("/api/git/branches/delete", h.HandleGitDeleteBranch)
	mux.HandleFunc("/api/git/checkout", h.HandleGitCheckout)
	mux.HandleFunc("/api/git-checkout", h.HandleGitCheckout)
	mux.HandleFunc("/api/sync/all", h.HandleSyncAll)
	mux.HandleFunc("/api/sync/linear", h.HandleSyncLinear)
	mux.HandleFunc("/api/sync/github", h.HandleSyncGithub)
	mux.HandleFunc("/api/sync/jira", h.HandleSyncJira)
	mux.HandleFunc("/api/skills", h.HandleSkills)
	mux.HandleFunc("/api/spec-framework/status", h.HandleSpecFrameworkStatus)
	mux.HandleFunc("/api/spec-framework/install", h.HandleSpecFrameworkInstall)
	mux.HandleFunc("/api/projects", h.HandleProjects)
	mux.HandleFunc("/api/projects/", h.HandleProjectDetail)
	mux.HandleFunc("/api/tasks", h.HandleTasks)
	mux.HandleFunc("/api/tasks/", h.HandleTaskDetail)
	mux.HandleFunc("/api/activities", h.HandleActivities)
	mux.HandleFunc("/api/activities/", h.HandleActivityDetail)
	mux.HandleFunc("/api/settings", h.HandleSettings)
	mux.HandleFunc("/api/seed", h.HandleSeed)
	mux.HandleFunc("/api/open-editor", h.HandleOpenEditor)
	mux.HandleFunc("/api/editor/open", h.HandleOpenEditor)

	// Interactive PTY Terminal Routes & WebSocket
	mux.HandleFunc("/ws/terminal", h.HandleTerminalWs)
	mux.HandleFunc("/api/terminal/send", h.HandleTerminalSend)
	mux.HandleFunc("/api/terminal/reset", h.HandleTerminalReset)

	// Static Web Assets / SPA fallback
	webDistDir := "./web/dist"
	if _, err := os.Stat(webDistDir); err == nil {
		fs := http.FileServer(http.Dir(webDistDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Route API non trouvée: %s", r.URL.Path)})
				return
			}
			path := filepath.Join(webDistDir, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(webDistDir, "index.html"))
				return
			}
			fs.ServeHTTP(w, r)
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Route API non trouvée: %s", r.URL.Path)})
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Taskacao API</title></head>
<body style="font-family: system-ui; padding: 2rem; background: #0f172a; color: #f8fafc;">
  <h2>Taskacao Go API is running!</h2>
  <p>To run the frontend with hot-reloading, run <code>cd web && npm run dev</code></p>
  <p>Or build static bundle via <code>cd web && npm run build</code> and restart this Go server.</p>
  <p>API endpoints available at <a href="/api/tasks" style="color: #818cf8;">/api/tasks</a>, <a href="/api/skills" style="color: #818cf8;">/api/skills</a> and <a href="/api/settings" style="color: #818cf8;">/api/settings</a>.</p>
</body>
</html>`)
		})
	}

	handlerWithCORS := h.EnableCORS(mux)

	addr := ":" + port
	log.Printf("🚀 Taskacao Server listening on http://localhost%s (DB: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, handlerWithCORS); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
