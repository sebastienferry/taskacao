package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tasks/internal/db"
	"tasks/internal/handlers"
	"tasks/internal/models"
)

func TestHandleGitStatus(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	h := handlers.NewHandler(database)

	cwd, _ := os.Getwd()
	req, err := http.NewRequest(http.MethodGet, "/api/git-status?path="+cwd, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	h.HandleGitStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var status models.GitStatusInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("Failed to decode response JSON: %v", err)
	}

	if !status.IsGitRepo {
		t.Errorf("Expected isGitRepo=true for current directory")
	}

	if status.Branch == "" {
		t.Errorf("Expected non-empty branch name")
	}
}

func TestCreateTaskWithCustomTrackerSource(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	h := handlers.NewHandler(database)

	// Create a test project with issueTracker="linear"
	_, _ = database.CreateProject(models.CreateProjectRequest{
		Name:         "Test Project",
		Slug:         "test-proj",
		IssueTracker: "linear",
		LinearTeam:   "TEST",
		RepoPath:     ".",
	})

	// 1. Create a task with explicitly specified source="local"
	taskBody := `{"title": "Test Local Task", "source": "local", "projectId": "test-proj"}`
	req, err := http.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(taskBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.HandleTasks(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var task models.Task
	if err := json.Unmarshal(rr.Body.Bytes(), &task); err != nil {
		t.Fatalf("Failed to decode response JSON: %v", err)
	}

	if task.Source != "local" {
		t.Errorf("Expected task.Source='local', got '%s'", task.Source)
	}

	if task.Title != "Test Local Task" {
		t.Errorf("Expected task.Title='Test Local Task', got '%s'", task.Title)
	}

	// 2. Create another task where source is omitted (should fallback to project tracker "linear")
	taskBodyDefault := `{"title": "Default Project Tracker Task", "projectId": "test-proj"}`
	req2, err := http.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(taskBodyDefault))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")

	rr2 := httptest.NewRecorder()
	h.HandleTasks(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var task2 models.Task
	if err := json.Unmarshal(rr2.Body.Bytes(), &task2); err != nil {
		t.Fatalf("Failed to decode response JSON: %v", err)
	}

	if task2.Source != "linear" {
		t.Errorf("Expected task2.Source='linear' (from project test-proj), got '%s'", task2.Source)
	}
}

