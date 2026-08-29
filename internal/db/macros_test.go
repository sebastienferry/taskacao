package db

import (
	"path/filepath"
	"testing"

	"tasks/internal/models"
)

func TestProjectCompatibility(t *testing.T) {
	p1 := &models.Project{ID: "p1", IssueTracker: "github", GithubRepo: "owner/repo1"}
	p2 := &models.Project{ID: "p2", IssueTracker: "github", GithubRepo: "owner/repo2"}
	pLocal := &models.Project{ID: "p3", IssueTracker: "local"}
	pJira := &models.Project{ID: "p4", IssueTracker: "jira"}

	if !IsProjectCompatible(p1, p2) {
		t.Errorf("expected GitHub projects to be compatible")
	}
	if !IsProjectCompatible(p1, pLocal) {
		t.Errorf("expected GitHub and local project to be compatible")
	}
	if IsProjectCompatible(p1, pJira) {
		t.Errorf("expected GitHub and Jira project not to be compatible")
	}
}

func TestMigrateMacroAndTasks(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create two local projects
	p1, err := database.CreateProject(models.CreateProjectRequest{
		Name:         "Project A",
		Slug:         "project-a",
		IssueTracker: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create project A: %v", err)
	}

	p2, err := database.CreateProject(models.CreateProjectRequest{
		Name:         "Project B",
		Slug:         "project-b",
		IssueTracker: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create project B: %v", err)
	}

	// Create a macro in project A
	h := "now"
	desc := "Macro in Project A"
	todos := []models.MacroTodo{{ID: "t1", Text: "Do something", Done: false}}
	macroA, err := database.SaveMacroMeta(p1.ID, "M-1", &h, &desc, &todos)
	if err != nil {
		t.Fatalf("Failed to save macro: %v", err)
	}
	if macroA.Key != "M-1" {
		t.Fatalf("Expected macro key M-1, got %s", macroA.Key)
	}

	// Create task under macro in project A
	taskA, err := database.CreateTask(models.CreateTaskRequest{
		ProjectID: p1.ID,
		Title:     "Task under M-1",
		Status:    models.StatusToClarify,
		Source:    "local",
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Set parent on task
	err = database.writeTaskParentLocally(taskA, "M-1")
	if err != nil {
		t.Fatalf("Failed to update parent: %v", err)
	}

	// Migrate macro and attached tasks from p1 to p2
	migratedMacro, taskCount, err := database.MigrateMacro(p1.ID, "M-1", p2.ID, true)
	if err != nil {
		t.Fatalf("MigrateMacro failed: %v", err)
	}
	if migratedMacro.ProjectID != p2.ID {
		t.Errorf("Expected migrated macro project ID %s, got %s", p2.ID, migratedMacro.ProjectID)
	}
	if taskCount != 1 {
		t.Errorf("Expected 1 migrated task, got %d", taskCount)
	}

	// Verify task in DB is now in p2
	updatedTask, err := database.GetTaskByID(taskA.ID)
	if err != nil || updatedTask == nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}
	if updatedTask.ProjectID != p2.ID {
		t.Errorf("Expected task project ID %s, got %s", p2.ID, updatedTask.ProjectID)
	}
}

func TestMigrateSingleTask(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	p1, err := database.CreateProject(models.CreateProjectRequest{
		Name:         "Project 1",
		Slug:         "project-1",
		IssueTracker: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}
	p2, err := database.CreateProject(models.CreateProjectRequest{
		Name:         "Project 2",
		Slug:         "project-2",
		IssueTracker: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	task, err := database.CreateTask(models.CreateTaskRequest{
		ProjectID: p1.ID,
		Title:     "Standalone Task",
		Status:    models.StatusToClarify,
		Source:    "local",
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	count, err := database.MigrateTasks([]string{task.ID}, p2.ID)
	if err != nil {
		t.Fatalf("MigrateTasks failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 task migrated, got %d", count)
	}

	migratedTask, err := database.GetTaskByID(task.ID)
	if err != nil || migratedTask == nil {
		t.Fatalf("Failed to get migrated task: %v", err)
	}
	if migratedTask.ProjectID != p2.ID {
		t.Errorf("Expected task project ID to be %s, got %s", p2.ID, migratedTask.ProjectID)
	}
}
