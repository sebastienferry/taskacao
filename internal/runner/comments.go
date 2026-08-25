package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"tasks/internal/models"
)

// Reading comments has two paths, for the same reason the sync does. The REST
// API carries the timestamp and the full body; acli carries author and an
// already flattened body but no date, which is enough when no token is
// configured.

const jiraCommentPageSize = 100

// FetchJiraComments reads the comments of a work item over REST, newest last.
func (c *JiraRESTClient) FetchJiraComments(ctx context.Context, issueKey string) ([]models.TaskComment, error) {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil, fmt.Errorf("clé de ticket manquante")
	}

	query := url.Values{}
	query.Set("maxResults", fmt.Sprintf("%d", jiraCommentPageSize))
	query.Set("orderBy", "created")

	body, err := c.get(ctx, "/rest/api/3/issue/"+url.PathEscape(issueKey)+"/comment", query)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Comments []struct {
			ID     string `json:"id"`
			Author struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
			Body    json.RawMessage `json:"body"`
			Created string          `json:"created"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("commentaires Jira illisibles: %w", err)
	}

	out := make([]models.TaskComment, 0, len(payload.Comments))
	for _, cm := range payload.Comments {
		comment := models.TaskComment{
			ID:     cm.ID,
			Author: cm.Author.DisplayName,
			// The body is Atlassian Document Format, the same shape as a
			// description, so the same flattener applies.
			Body:   jiraDescriptionToText(cm.Body),
			Source: "jira",
		}
		if cm.Created != "" {
			// Jira stamps "2026-08-25T09:12:33.000+0200".
			for _, layout := range []string{"2006-01-02T15:04:05.000-0700", time.RFC3339} {
				if ts, err := time.Parse(layout, cm.Created); err == nil {
					comment.CreatedAt = &ts
					break
				}
			}
		}
		out = append(out, comment)
	}
	return out, nil
}

// FetchJiraCommentsViaCLI reads the comments with acli, which needs no token.
// The payload has no timestamp, so the order carries the chronology.
func (r *Runner) FetchJiraCommentsViaCLI(issueKey string, repoPath string) ([]models.TaskComment, error) {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil, fmt.Errorf("clé de ticket manquante")
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.runCommand(ctx, repoPath, acliPath, "jira", "workitem", "comment", "list",
		"--key", issueKey, "--order", "+created", "--json")
	if err != nil {
		return nil, fmt.Errorf("lecture des commentaires de %s impossible: %w", issueKey, err)
	}

	comments := []models.TaskComment{}
	err = decodeJSONDocuments(out, func(dec *json.Decoder) error {
		var page struct {
			Comments []struct {
				ID     string `json:"id"`
				Author string `json:"author"`
				Body   string `json:"body"`
			} `json:"comments"`
		}
		if err := dec.Decode(&page); err != nil {
			return err
		}
		for _, cm := range page.Comments {
			comments = append(comments, models.TaskComment{
				ID:     cm.ID,
				Author: cm.Author,
				Body:   cm.Body,
				Source: "jira",
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("commentaires acli illisibles: %w", err)
	}
	return comments, nil
}
