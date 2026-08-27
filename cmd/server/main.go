package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"io/fs"
	"os/exec"
	"runtime"
	"time"

	"tasks/internal/db"
	"tasks/internal/handlers"
	"tasks/internal/webui"
)

// loadDotEnv reads KEY=VALUE lines from a .env file next to the binary's working
// directory. A real environment variable always wins, so exporting a value in
// the shell overrides the file. Secrets such as TASKACAO_JIRA_API_TOKEN can then
// live outside the database and outside git, .env being already gitignored.
func loadDotEnv(paths ...string) {
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			// Quotes are what a user naturally types around a secret.
			if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
			if key == "" {
				continue
			}
			if _, already := os.LookupEnv(key); already {
				continue
			}
			_ = os.Setenv(key, value)
		}
		log.Printf("Loaded environment from %s", path)
	}
}

// resolveDBPath decides where the database lives.
//
// Three cases, in this order. An explicit DB_PATH wins, always. Then a database
// already sitting in the working directory is kept: someone who has been running
// the program from a checkout must not silently start from an empty board. Only
// otherwise does the database go to the user's data directory, which is what a
// distributed binary needs, since it may be launched from anywhere.
func resolveDBPath(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if _, err := os.Stat("tasks.db"); err == nil {
		return "tasks.db"
	}

	dir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		// Pas de dossier utilisateur lisible : le répertoire courant reste un
		// endroit valable, et vaut mieux qu'un démarrage refusé.
		return "tasks.db"
	}
	appDir := filepath.Join(dir, "taskacao")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "tasks.db"
	}
	return filepath.Join(appDir, "tasks.db")
}

// openBrowser opens the interface once the server listens. It is best effort by
// design: a machine without a browser, or a headless run, must not turn a
// cosmetic step into a failure to start.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", ""}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Printf("Navigateur non ouvert (%v). Ouvrez %s à la main.", err, url)
	}
}

func main() {
	// .env.local last: it overrides nothing already exported, but is the usual
	// place for a machine-specific secret.
	loadDotEnv(".env", ".env.local")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	dbPath := resolveDBPath(os.Getenv("DB_PATH"))

	database, err := db.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal database error: %v", err)
	}
	defer database.Close()

	h := handlers.NewHandler(database)

	// Boucle de synchronisation de fond. Elle ne fait rien tant que le réglage
	// est éteint, et ne lit ensuite que ce qui a changé depuis sa passe
	// précédente.
	database.StartAutoSync()

	// Les pas du workflow tournent dans la session PTY de leur tâche : visibles
	// pendant qu'ils travaillent, ouvrables d'un clic, et interrogeables.
	database.SetTerminalRunner(h.TerminalRunner())

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
	mux.HandleFunc("/api/sync/auto", h.HandleAutoSyncStatus)
	mux.HandleFunc("/api/skills", h.HandleSkills)
	mux.HandleFunc("/api/spec-framework/status", h.HandleSpecFrameworkStatus)
	mux.HandleFunc("/api/spec-framework/install", h.HandleSpecFrameworkInstall)
	mux.HandleFunc("/api/projects", h.HandleProjects)
	mux.HandleFunc("/api/projects/", h.HandleProjectDetail)
	mux.HandleFunc("/api/tasks", h.HandleTasks)
	mux.HandleFunc("/api/tasks/facets", h.HandleTaskFacets)
	mux.HandleFunc("/api/teams", h.HandleTeams)
	mux.HandleFunc("/api/teams/", h.HandleTeams)
	mux.HandleFunc("/api/tasks/pins", h.HandleTaskPins)
	mux.HandleFunc("/api/tasks/", h.HandleTaskDetail)
	mux.HandleFunc("/api/activities", h.HandleActivities)
	mux.HandleFunc("/api/activities/", h.HandleActivityDetail)
	mux.HandleFunc("/api/settings", h.HandleSettings)
	mux.HandleFunc("/api/open-editor", h.HandleOpenEditor)
	mux.HandleFunc("/api/editor/open", h.HandleOpenEditor)

	// Interactive PTY Terminal Routes & WebSocket
	mux.HandleFunc("/ws/terminal", h.HandleTerminalWs)
	mux.HandleFunc("/api/terminal/sessions", h.HandleTerminalSessions)
	mux.HandleFunc("/api/terminal/send", h.HandleTerminalSend)
	mux.HandleFunc("/api/terminal/reset", h.HandleTerminalReset)

	// Interface : la copie embarquée d'abord, le dossier de build ensuite.
	//
	// L'embarqué est ce qui fait tenir l'application dans un fichier. Le repli
	// disque sert la boucle de développement, où l'on rebuild le front sans
	// recompiler le serveur.
	uiFS, uiEmbedded := webui.FS()
	webDistDir := "./internal/webui/dist"
	if !uiEmbedded {
		if _, err := os.Stat(webDistDir); err == nil {
			uiFS = os.DirFS(webDistDir)
			uiEmbedded = true
			log.Printf("Interface servie depuis %s", webDistDir)
		}
	}

	if uiEmbedded {
		fileServer := http.FileServer(http.FS(uiFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Route API non trouvée: %s", r.URL.Path)})
				return
			}
			// index.html must never be cached: it is the file that names the
			// hashed bundle, so a stale copy keeps serving the previous build and
			// the app looks unchanged after a rebuild. The assets themselves are
			// content-hashed, so they can be cached hard.
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-store, must-revalidate")
			}
			name := strings.TrimPrefix(r.URL.Path, "/")
			if name == "" {
				name = "index.html"
			}
			if _, err := fs.Stat(uiFS, name); err != nil {
				// Route de l'application : c'est index.html qui la résout.
				index, err := fs.ReadFile(uiFS, "index.html")
				if err != nil {
					http.Error(w, "interface indisponible", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(index)
				return
			}
			fileServer.ServeHTTP(w, r)
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
  <p>Or build the bundle via <code>make build</code>, which compiles the interface into the binary.</p>
  <p>API endpoints available at <a href="/api/tasks" style="color: #818cf8;">/api/tasks</a>, <a href="/api/skills" style="color: #818cf8;">/api/skills</a> and <a href="/api/settings" style="color: #818cf8;">/api/settings</a>.</p>
</body>
</html>`)
		})
	}

	handlerWithCORS := h.EnableCORS(mux)

	addr := ":" + port
	url := fmt.Sprintf("http://localhost%s", addr)
	log.Printf("🚀 Taskacao Server listening on %s (DB: %s)", url, dbPath)

	// Ouverture du navigateur : ce que fait une application quand on la lance.
	// TASKACAO_NO_BROWSER=1 l'en empêche, pour un serveur ou un lancement par un
	// gestionnaire de services.
	if os.Getenv("TASKACAO_NO_BROWSER") == "" && uiEmbedded {
		go func() {
			// Laisser le port s'ouvrir : un navigateur plus rapide que le
			// serveur afficherait une erreur de connexion.
			time.Sleep(400 * time.Millisecond)
			openBrowser(url)
		}()
	}

	if err := http.ListenAndServe(addr, handlerWithCORS); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
