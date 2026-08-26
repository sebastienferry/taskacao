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

// JiraEpic is what an epic carries for the roadmap: its labels (the roadmap
// horizon), its real title and its status. The sync imports only Task and Story,
// so none of this arrives otherwise.
type JiraEpic struct {
	Key            string
	Summary        string
	Labels         []string
	StatusName     string
	StatusCategory string
}

// FetchJiraEpics reads a project's epics. Both paths work: REST when a token
// exists, and acli otherwise — key, summary, labels and status are all in the
// fields its --fields flag accepts.
func (r *Runner) FetchJiraEpics(settings *models.Settings, trackerURL string, projectKey string, repoPath string) (map[string]JiraEpic, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante")
	}
	jql := fmt.Sprintf("project = %s AND issuetype = Epic ORDER BY key ASC", projectKey)

	if client := NewJiraRESTClient(settings, trackerURL); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if out, err := client.searchEpics(ctx, jql); err == nil {
			return out, nil
		}
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	raw, err := r.runCommand(ctx, repoPath, acliPath, "jira", "workitem", "search",
		"--jql", jql, "--fields", "key,summary,labels,status", "--paginate", "--json")
	if err != nil {
		return nil, fmt.Errorf("lecture des épics impossible: %w", err)
	}

	out := map[string]JiraEpic{}
	decodeErr := decodeJSONDocuments(raw, func(dec *json.Decoder) error {
		var page struct {
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string   `json:"summary"`
					Labels  []string `json:"labels"`
					Status  struct {
						Name           string `json:"name"`
						StatusCategory struct {
							Key string `json:"key"`
						} `json:"statusCategory"`
					} `json:"status"`
				} `json:"fields"`
			} `json:"issues"`
		}
		if err := dec.Decode(&page); err != nil {
			return err
		}
		for _, issue := range page.Issues {
			if issue.Key == "" {
				continue
			}
			out[issue.Key] = JiraEpic{
				Key:            issue.Key,
				Summary:        issue.Fields.Summary,
				Labels:         issue.Fields.Labels,
				StatusName:     issue.Fields.Status.Name,
				StatusCategory: issue.Fields.Status.StatusCategory.Key,
			}
		}
		return nil
	})
	if decodeErr != nil && len(out) == 0 {
		return nil, fmt.Errorf("réponse acli illisible: %w", decodeErr)
	}
	return out, nil
}

// FetchJiraIssuesByKeys reads specific work items by key. Needed for the epics
// that live in another Jira project: a "project = X AND issuetype = Epic" query
// never returns them, so comparing their labels required asking for them by name.
func (r *Runner) FetchJiraIssuesByKeys(settings *models.Settings, trackerURL string, keys []string, repoPath string) (map[string]JiraEpic, error) {
	quoted := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.ToUpper(strings.TrimSpace(key))
		if key != "" {
			quoted = append(quoted, key)
		}
	}
	if len(quoted) == 0 {
		return map[string]JiraEpic{}, nil
	}
	jql := fmt.Sprintf("key in (%s)", strings.Join(quoted, ", "))

	if client := NewJiraRESTClient(settings, trackerURL); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if out, err := client.searchEpics(ctx, jql); err == nil {
			return out, nil
		}
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	raw, err := r.runCommand(ctx, repoPath, acliPath, "jira", "workitem", "search",
		"--jql", jql, "--fields", "key,summary,labels,status", "--paginate", "--json")
	if err != nil {
		return nil, fmt.Errorf("lecture des épics hors projet impossible: %w", err)
	}

	out := map[string]JiraEpic{}
	_ = decodeJSONDocuments(raw, func(dec *json.Decoder) error {
		var page struct {
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string   `json:"summary"`
					Labels  []string `json:"labels"`
					Status  struct {
						Name           string `json:"name"`
						StatusCategory struct {
							Key string `json:"key"`
						} `json:"statusCategory"`
					} `json:"status"`
				} `json:"fields"`
			} `json:"issues"`
		}
		if err := dec.Decode(&page); err != nil {
			return err
		}
		for _, issue := range page.Issues {
			if issue.Key != "" {
				out[issue.Key] = JiraEpic{
					Key: issue.Key, Summary: issue.Fields.Summary, Labels: issue.Fields.Labels,
					StatusName: issue.Fields.Status.Name, StatusCategory: issue.Fields.Status.StatusCategory.Key,
				}
			}
		}
		return nil
	})
	return out, nil
}

func (c *JiraRESTClient) searchEpics(ctx context.Context, jql string) (map[string]JiraEpic, error) {
	out := map[string]JiraEpic{}
	pageToken := ""
	for page := 0; page < 20; page++ {
		query := url.Values{}
		query.Set("jql", jql)
		query.Set("fields", "summary,labels,status")
		query.Set("maxResults", "100")
		if pageToken != "" {
			query.Set("nextPageToken", pageToken)
		}
		body, err := c.get(ctx, "/rest/api/3/search/jql", query)
		if err != nil {
			return out, err
		}
		var payload struct {
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string   `json:"summary"`
					Labels  []string `json:"labels"`
					Status  struct {
						Name           string `json:"name"`
						StatusCategory struct {
							Key string `json:"key"`
						} `json:"statusCategory"`
					} `json:"status"`
				} `json:"fields"`
			} `json:"issues"`
			NextPageToken string `json:"nextPageToken"`
			IsLast        bool   `json:"isLast"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return out, err
		}
		for _, issue := range payload.Issues {
			if issue.Key == "" {
				continue
			}
			out[issue.Key] = JiraEpic{
				Key:            issue.Key,
				Summary:        issue.Fields.Summary,
				Labels:         issue.Fields.Labels,
				StatusName:     issue.Fields.Status.Name,
				StatusCategory: issue.Fields.Status.StatusCategory.Key,
			}
		}
		if payload.IsLast || payload.NextPageToken == "" || len(payload.Issues) == 0 {
			break
		}
		pageToken = payload.NextPageToken
	}
	return out, nil
}
