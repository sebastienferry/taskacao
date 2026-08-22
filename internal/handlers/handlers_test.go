package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
