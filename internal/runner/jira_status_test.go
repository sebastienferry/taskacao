package runner

import (
	"testing"

	"tasks/internal/models"
)

// TestJiraItemsToTasksStatusMapping pins the import of the statuses whose
// meaning is not obvious from Jira's own status category.
func TestJiraItemsToTasksStatusMapping(t *testing.T) {
	cases := []struct {
		statusName  string
		categoryKey string
		wantStatus  models.Status
	}{
		// The SFE workflow leaves WONTDO in the "In Progress" category, so the
		// category fallback used to import abandoned tickets as work still to
		// do — and the daily digest listed them under "à traiter aujourd'hui".
		{"WONTDO", "indeterminate", models.StatusFinished},
		{"Won't Do", "done", models.StatusFinished},
		{"Closed", "indeterminate", models.StatusFinished},
		{"Cancelled", "indeterminate", models.StatusFinished},
		{"Done", "done", models.StatusFinished},
		{"To Do", "new", models.StatusToSpecify},
		{"In Progress", "indeterminate", models.StatusToImplement},
		{"In Review", "indeterminate", models.StatusToTest},
	}

	r := &Runner{}
	for _, tc := range cases {
		item := JiraIssueItem{Key: "SFE-1"}
		item.Fields.Summary = "un ticket"
		item.Fields.IssueType.Name = "Task"
		item.Fields.Status.Name = tc.statusName
		item.Fields.Status.StatusCategory.Key = tc.categoryKey

		tasks := r.jiraItemsToTasks([]JiraIssueItem{item}, "")
		if len(tasks) != 1 {
			t.Fatalf("statut %q : %d tâche(s) importée(s), 1 attendue", tc.statusName, len(tasks))
		}
		if tasks[0].Status != tc.wantStatus {
			t.Errorf("statut Jira %q (catégorie %q) importé en %q, %q attendu",
				tc.statusName, tc.categoryKey, tasks[0].Status, tc.wantStatus)
		}
	}
}
