package tracker

import (
	"context"

	"tasks/internal/runner"
)

// JiraWriter adapts the Jira REST client to the common interface. It holds no
// logic of its own: the calls it forwards are the ones that were already there,
// and the point is that the caller stops naming Jira.
type JiraWriter struct {
	client *runner.JiraRESTClient
	// transition goes through the runner rather than the REST client: moving a
	// work item means finding the workflow transition that leads to the wanted
	// status, which the runner already resolves, acli fallback included.
	transition func(ctx context.Context, key string, status string) error
}

// NewJiraWriter returns nil when Jira is not reachable, which is how the caller
// learns that no write is possible: no token, no site, nothing to talk to.
func NewJiraWriter(client *runner.JiraRESTClient, transition func(ctx context.Context, key, status string) error) *JiraWriter {
	if client == nil {
		return nil
	}
	return &JiraWriter{client: client, transition: transition}
}

func (w *JiraWriter) Name() string { return "jira" }

// jiraCapabilities is everything a Jira Cloud instance answers to, given a
// token. Sprints and teams are exactly what other trackers lack.
var jiraCapabilities = []Capability{
	CapAssign, CapTransition, CapSprint, CapTeam, CapEpic, CapLabels, CapComment,
}

func (w *JiraWriter) Supports(c Capability) bool { return Has(jiraCapabilities, c) }

func (w *JiraWriter) Assign(ctx context.Context, key string, personID string) error {
	return w.client.AssignIssue(ctx, key, personID)
}

func (w *JiraWriter) Transition(ctx context.Context, key string, status string) error {
	if w.transition == nil {
		return Unsupported(w.Name(), CapTransition)
	}
	return w.transition(ctx, key, status)
}

func (w *JiraWriter) SetSprint(ctx context.Context, sprintID string, keys []string) error {
	return w.client.MoveIssuesToSprint(ctx, sprintID, keys)
}

func (w *JiraWriter) SetTeam(ctx context.Context, key string, teamID string) error {
	return w.client.SetIssueTeam(ctx, key, teamID)
}

func (w *JiraWriter) SetParent(ctx context.Context, key string, parentKey string) error {
	return w.client.SetIssueParent(ctx, key, parentKey)
}

func (w *JiraWriter) UpdateLabels(ctx context.Context, key string, add []string, remove []string) error {
	return w.client.UpdateIssueLabels(ctx, key, add, remove)
}

func (w *JiraWriter) SearchAssignable(ctx context.Context, key string, query string, limit int) ([]Person, error) {
	members, err := w.client.SearchAssignableUsers(ctx, key, query, limit)
	if err != nil {
		return nil, err
	}
	people := make([]Person, 0, len(members))
	for _, m := range members {
		people = append(people, Person{
			ID:          m.AccountID,
			DisplayName: m.DisplayName,
			Email:       m.Email,
			AvatarURL:   m.AvatarURL,
			Active:      m.Active,
		})
	}
	return people, nil
}
