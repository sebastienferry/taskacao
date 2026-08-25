package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"tasks/internal/models"
)

// Sprint and Team are Jira custom fields, and acli refuses them: its search
// --fields flag only accepts a fixed allow-list (issuetype, key, assignee,
// priority, status, summary, description, labels), and workitem view returns an
// even smaller set. So the values are fetched over the Jira REST API with a
// dedicated API token, in a single paginated pass that mirrors the JQL the acli
// sync already uses.

const (
	jiraRESTTimeout  = 90 * time.Second
	jiraRESTPageSize = 100
	jiraRESTMaxPages = 40 // 4000 work items, well past the largest project here
)

// JiraFieldValues carries the tracker fields a ticket belongs to.
type JiraFieldValues struct {
	Sprint string
	Team   string
}

// JiraRESTClient talks to the Jira Cloud REST API with Basic auth (email plus
// API token). It is created per sync run and caches the discovered field ids.
type JiraRESTClient struct {
	baseURL string
	email   string
	token   string
	http    *http.Client

	sprintFieldID string
	teamFieldID   string
}

// NewJiraRESTClient builds a client from the settings, or returns nil when the
// credentials are missing. A nil client is not an error: the sync then simply
// leaves sprint and team untouched instead of failing.
func NewJiraRESTClient(settings *models.Settings, trackerURL string) *JiraRESTClient {
	if settings == nil {
		return nil
	}
	email := strings.TrimSpace(settings.JiraEmail)
	token := JiraTokenFromEnv()
	if token == "" {
		token = strings.TrimSpace(settings.JiraAPIToken)
	}
	if email == "" || token == "" {
		return nil
	}

	base := normalizeJiraBaseURL(trackerURL)
	if base == "" {
		base = normalizeJiraBaseURL(settings.JiraUrl)
	}
	if base == "" {
		return nil
	}

	return &JiraRESTClient{
		baseURL: base,
		email:   email,
		token:   token,
		http:    &http.Client{Timeout: jiraRESTTimeout},
	}
}

// ListBoards returns the tracker boards attached to a project, for the picker in
// the project settings.
func (c *JiraRESTClient) ListBoards(ctx context.Context, projectKey string) ([]models.TrackerBoard, error) {
	query := url.Values{}
	query.Set("projectKeyOrId", strings.ToUpper(strings.TrimSpace(projectKey)))
	query.Set("maxResults", "50")

	body, err := c.get(ctx, "/rest/agile/1.0/board", query)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Values []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("liste des boards illisible: %w", err)
	}

	boards := make([]models.TrackerBoard, 0, len(payload.Values))
	for _, b := range payload.Values {
		boards = append(boards, models.TrackerBoard{
			ID:   fmt.Sprintf("%d", b.ID),
			Name: b.Name,
			Type: b.Type,
		})
	}
	return boards, nil
}

// ListBoardSprints returns the board's sprints with their state. The state is
// what separates the operational horizons: active means the work is in flight,
// future means it is planned but not started.
func (c *JiraRESTClient) ListBoardSprints(ctx context.Context, boardID string) ([]models.TrackerSprint, error) {
	boardID = strings.TrimSpace(boardID)
	if boardID == "" {
		return nil, fmt.Errorf("identifiant de board manquant")
	}

	out := []models.TrackerSprint{}
	startAt := 0
	for page := 0; page < 20; page++ {
		query := url.Values{}
		query.Set("state", "active,future")
		query.Set("maxResults", "50")
		query.Set("startAt", fmt.Sprintf("%d", startAt))

		body, err := c.get(ctx, "/rest/agile/1.0/board/"+url.PathEscape(boardID)+"/sprint", query)
		if err != nil {
			return out, err
		}
		var payload struct {
			Values []struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"values"`
			IsLast bool `json:"isLast"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return out, fmt.Errorf("sprints du board illisibles: %w", err)
		}
		for _, sp := range payload.Values {
			if strings.TrimSpace(sp.Name) == "" {
				continue
			}
			out = append(out, models.TrackerSprint{Name: sp.Name, State: strings.ToLower(sp.State)})
		}
		if payload.IsLast || len(payload.Values) == 0 {
			break
		}
		startAt += len(payload.Values)
	}
	return out, nil
}

// FetchBoardColumns returns the columns of a board in their board order, with
// the status names each one groups. The board configuration only carries status
// ids, so they are resolved against the instance's status list.
func (c *JiraRESTClient) FetchBoardColumns(ctx context.Context, boardID string) ([]models.TrackerColumn, error) {
	boardID = strings.TrimSpace(boardID)
	if boardID == "" {
		return nil, fmt.Errorf("identifiant de board manquant")
	}

	body, err := c.get(ctx, "/rest/agile/1.0/board/"+url.PathEscape(boardID)+"/configuration", nil)
	if err != nil {
		return nil, err
	}

	var cfg struct {
		ColumnConfig struct {
			Columns []struct {
				Name     string `json:"name"`
				Statuses []struct {
					ID string `json:"id"`
				} `json:"statuses"`
			} `json:"columns"`
		} `json:"columnConfig"`
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("configuration du board illisible: %w", err)
	}

	statusNames, err := c.statusNamesByID(ctx)
	if err != nil {
		return nil, err
	}

	columns := make([]models.TrackerColumn, 0, len(cfg.ColumnConfig.Columns))
	for _, col := range cfg.ColumnConfig.Columns {
		names := make([]string, 0, len(col.Statuses))
		for _, st := range col.Statuses {
			if name, ok := statusNames[st.ID]; ok && name != "" {
				names = append(names, name)
			}
		}
		// A column with no status holds no card; keeping it would only add an
		// empty lane to the board.
		if len(names) == 0 {
			continue
		}
		columns = append(columns, models.TrackerColumn{Name: col.Name, Statuses: names})
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("aucune colonne exploitable sur le board %s", boardID)
	}
	return columns, nil
}

func (c *JiraRESTClient) statusNamesByID(ctx context.Context) (map[string]string, error) {
	body, err := c.get(ctx, "/rest/api/3/status", nil)
	if err != nil {
		return nil, err
	}
	var statuses []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &statuses); err != nil {
		return nil, fmt.Errorf("liste des statuts Jira illisible: %w", err)
	}
	out := make(map[string]string, len(statuses))
	for _, st := range statuses {
		out[st.ID] = st.Name
	}
	return out, nil
}

// JiraReadSource names the path the sync will read through, for the activity log.
func JiraReadSource(settings *models.Settings, trackerURL string) string {
	if NewJiraRESTClient(settings, trackerURL) != nil {
		return "API REST"
	}
	return "CLI acli"
}

// JiraTokenEnvVar lets the token stay out of the local database entirely.
const JiraTokenEnvVar = "TASKACAO_JIRA_API_TOKEN"

// JiraTokenFromEnv returns the token from the environment, which takes
// precedence over the stored one.
func JiraTokenFromEnv() string {
	return strings.TrimSpace(os.Getenv(JiraTokenEnvVar))
}

// normalizeJiraBaseURL accepts what a user realistically types — a bare site,
// a full URL, or a deep link — and reduces it to the scheme plus host the REST
// API lives on.
func normalizeJiraBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return "https://" + parsed.Host
}

func (c *JiraRESTClient) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.email+":"+c.token)))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("Jira a refusé l'authentification (%d): vérifiez l'e-mail et le jeton d'API dans les réglages", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("Jira %s a répondu %d: %s", path, resp.StatusCode, snippet)
	}
	if readErr != nil {
		return nil, readErr
	}
	return body, nil
}

func (c *JiraRESTClient) post(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.email+":"+c.token)))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("Jira a refusé l'authentification (%d)", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(respBody))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return nil, fmt.Errorf("Jira %s a répondu %d: %s", path, resp.StatusCode, snippet)
	}
	return respBody, nil
}

// JiraTransition is one workflow transition available from the current status.
// Its name is what the workflow calls the action ("Close Issue"), which is not
// the name of the status it leads to ("Closed") — the distinction is exactly
// what made acli refuse a valid move.
type JiraTransition struct {
	ID       string
	Name     string
	ToStatus string
}

// ListTransitions returns the transitions currently available on a work item.
func (c *JiraRESTClient) ListTransitions(ctx context.Context, issueKey string) ([]JiraTransition, error) {
	body, err := c.get(ctx, "/rest/api/3/issue/"+url.PathEscape(strings.TrimSpace(issueKey))+"/transitions", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("transitions Jira illisibles: %w", err)
	}
	out := make([]JiraTransition, 0, len(payload.Transitions))
	for _, tr := range payload.Transitions {
		out = append(out, JiraTransition{ID: tr.ID, Name: tr.Name, ToStatus: tr.To.Name})
	}
	return out, nil
}

// TransitionToStatus moves a work item to a target status by finding the
// transition that leads there and posting its id. Deterministic: no name
// matching, and a refusal comes back as a real error instead of a success.
func (c *JiraRESTClient) TransitionToStatus(ctx context.Context, issueKey string, statusName string) error {
	statusName = strings.TrimSpace(statusName)
	transitions, err := c.ListTransitions(ctx, issueKey)
	if err != nil {
		return err
	}

	for _, tr := range transitions {
		if strings.EqualFold(tr.ToStatus, statusName) {
			_, err := c.post(ctx, "/rest/api/3/issue/"+url.PathEscape(issueKey)+"/transitions",
				map[string]interface{}{"transition": map[string]string{"id": tr.ID}})
			return err
		}
	}

	available := make([]string, 0, len(transitions))
	for _, tr := range transitions {
		available = append(available, tr.ToStatus)
	}
	return fmt.Errorf("aucune transition de %s ne mène à « %s » depuis son statut actuel (possibles : %s)",
		issueKey, statusName, strings.Join(available, ", "))
}

type jiraFieldMeta struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Custom bool   `json:"custom"`
	Schema struct {
		Type   string `json:"type"`
		Custom string `json:"custom"`
	} `json:"schema"`
}

// DiscoverFields resolves the Sprint and Team field ids. They are custom field
// ids that differ per Jira instance, so they are looked up by schema then by
// name rather than hardcoded.
func (c *JiraRESTClient) DiscoverFields(ctx context.Context) error {
	body, err := c.get(ctx, "/rest/api/3/field", nil)
	if err != nil {
		return err
	}

	var fields []jiraFieldMeta
	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("liste des champs Jira illisible: %w", err)
	}

	for _, f := range fields {
		switch {
		case c.sprintFieldID == "" && strings.HasSuffix(f.Schema.Custom, ":gh-sprint"):
			c.sprintFieldID = f.ID
		case c.teamFieldID == "" && (f.Schema.Type == "team" || strings.HasSuffix(f.Schema.Custom, ":atlassian-team")):
			c.teamFieldID = f.ID
		}
	}

	// Fall back on the display name when the schema does not identify the field
	// (older instances, or a locally defined "Team" field).
	for _, f := range fields {
		name := strings.ToLower(strings.TrimSpace(f.Name))
		if c.sprintFieldID == "" && name == "sprint" {
			c.sprintFieldID = f.ID
		}
		if c.teamFieldID == "" && (name == "team" || name == "équipe" || name == "equipe") {
			c.teamFieldID = f.ID
		}
	}

	if c.sprintFieldID == "" && c.teamFieldID == "" {
		return fmt.Errorf("ni le champ Sprint ni le champ Team n'existent sur cette instance Jira")
	}
	return nil
}

// FieldsFound reports which of the two fields were resolved, for the sync log.
func (c *JiraRESTClient) FieldsFound() (sprint bool, team bool) {
	return c.sprintFieldID != "", c.teamFieldID != ""
}

type jiraSearchPage struct {
	Issues []struct {
		Key    string                     `json:"key"`
		Fields map[string]json.RawMessage `json:"fields"`
	} `json:"issues"`
	NextPageToken string `json:"nextPageToken"`
	IsLast        bool   `json:"isLast"`
}

type jiraSprintValue struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// parseJiraSprintValue picks the sprint to display. A ticket carried across
// sprints holds several: the active one wins, otherwise the last of the list,
// which is the most recent one Jira reports.
func parseJiraSprintValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var sprints []jiraSprintValue
	if err := json.Unmarshal(raw, &sprints); err == nil {
		chosen := ""
		for _, s := range sprints {
			if strings.TrimSpace(s.Name) == "" {
				continue
			}
			if strings.EqualFold(s.State, "active") {
				return s.Name
			}
			chosen = s.Name
		}
		return chosen
	}

	// Some instances expose the sprint as a single object or a plain string.
	var single jiraSprintValue
	if err := json.Unmarshal(raw, &single); err == nil && single.Name != "" {
		return single.Name
	}
	var asText string
	if err := json.Unmarshal(raw, &asText); err == nil {
		return strings.TrimSpace(asText)
	}
	return ""
}

// parseJiraTeamValue reads the Team field, which is an object whose label lives
// under one of several keys depending on the instance.
func parseJiraTeamValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, key := range []string{"name", "title", "value", "displayName"} {
			if v, ok := obj[key].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	var asText string
	if err := json.Unmarshal(raw, &asText); err == nil {
		return strings.TrimSpace(asText)
	}
	return ""
}

// JiraRESTSearchResult is one paginated read of a project: the work items in the
// same shape acli returns (so the conversion code is shared), plus the custom
// fields acli cannot project.
type JiraRESTSearchResult struct {
	Items  []JiraIssueItem
	Fields map[string]JiraFieldValues
}

// SearchProjectIssues reads every synced work item of a project over REST. It
// projects parent as well, which removes the need for the epic-walking pass the
// acli path has to run because its --fields rejects 'parent'.
func (c *JiraRESTClient) SearchProjectIssues(ctx context.Context, projectKey string) (*JiraRESTSearchResult, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante")
	}
	if c.sprintFieldID == "" && c.teamFieldID == "" {
		// A missing Sprint or Team field is not fatal here: the rest of the
		// payload is still worth importing.
		_ = c.DiscoverFields(ctx)
	}

	quoted := make([]string, 0, len(jiraSyncedIssueTypes))
	for _, t := range jiraSyncedIssueTypes {
		quoted = append(quoted, fmt.Sprintf("%q", t))
	}
	jql := fmt.Sprintf("project = %s AND issuetype IN (%s) ORDER BY updated DESC",
		projectKey, strings.Join(quoted, ", "))

	fields := []string{"key", "summary", "description", "status", "priority", "assignee", "labels", "issuetype", "parent"}
	if c.sprintFieldID != "" {
		fields = append(fields, c.sprintFieldID)
	}
	if c.teamFieldID != "" {
		fields = append(fields, c.teamFieldID)
	}

	result := &JiraRESTSearchResult{Fields: map[string]JiraFieldValues{}}
	pageToken := ""

	for page := 0; page < jiraRESTMaxPages; page++ {
		query := url.Values{}
		query.Set("jql", jql)
		query.Set("fields", strings.Join(fields, ","))
		query.Set("maxResults", fmt.Sprintf("%d", jiraRESTPageSize))
		if pageToken != "" {
			query.Set("nextPageToken", pageToken)
		}

		body, err := c.get(ctx, "/rest/api/3/search/jql", query)
		if err != nil {
			return result, err
		}

		var raw struct {
			Issues        []json.RawMessage `json:"issues"`
			NextPageToken string            `json:"nextPageToken"`
			IsLast        bool              `json:"isLast"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return result, fmt.Errorf("réponse de recherche Jira illisible: %w", err)
		}

		for _, rawIssue := range raw.Issues {
			var item JiraIssueItem
			if err := json.Unmarshal(rawIssue, &item); err != nil || item.Key == "" {
				continue
			}
			result.Items = append(result.Items, item)

			// The custom field ids are instance-specific, so they are read from
			// the raw payload rather than through a typed struct.
			var dynamic struct {
				Fields map[string]json.RawMessage `json:"fields"`
			}
			if err := json.Unmarshal(rawIssue, &dynamic); err != nil {
				continue
			}
			values := JiraFieldValues{}
			if c.sprintFieldID != "" {
				values.Sprint = parseJiraSprintValue(dynamic.Fields[c.sprintFieldID])
			}
			if c.teamFieldID != "" {
				values.Team = parseJiraTeamValue(dynamic.Fields[c.teamFieldID])
			}
			if values.Sprint != "" || values.Team != "" {
				result.Fields[item.Key] = values
			}
		}

		if raw.IsLast || raw.NextPageToken == "" || len(raw.Issues) == 0 {
			break
		}
		pageToken = raw.NextPageToken
	}

	return result, nil
}
