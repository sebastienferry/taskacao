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
	"sort"
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
	// jiraErrorBodyLimit is how much of a refusal body is kept. Jira names the
	// offending field in it ("customfield_NNNNN: Epic Type is required"), and
	// that sentence is what the activity has to show, so the old 200 characters
	// cut exactly where it mattered.
	jiraErrorBodyLimit = 800
)

// JiraFieldValues carries the tracker fields a ticket belongs to.
type JiraFieldValues struct {
	Sprint string
	Team   string
	// TeamID is the Atlassian team id. The members endpoint is keyed by id, so
	// the label alone is not enough to read who is in the team.
	TeamID string
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
	// cloudID identifies the site for the team endpoints, which live outside the
	// Jira API and take it as siteId. Read once per client.
	cloudID string
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

// UpdateIssueLabels adds and removes labels on a work item over REST. Spawning
// acli costs about 1.3 s per call, most of it process start-up and its own
// authentication; the REST round-trip is roughly six times faster, which matters
// when triaging a roadmap epic by epic.
func (c *JiraRESTClient) UpdateIssueLabels(ctx context.Context, issueKey string, add []string, remove []string) error {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return fmt.Errorf("clé de ticket manquante")
	}

	ops := make([]map[string]string, 0, len(add)+len(remove))
	for _, label := range add {
		if label = strings.TrimSpace(label); label != "" {
			ops = append(ops, map[string]string{"add": label})
		}
	}
	for _, label := range remove {
		if label = strings.TrimSpace(label); label != "" {
			ops = append(ops, map[string]string{"remove": label})
		}
	}
	if len(ops) == 0 {
		return nil
	}

	// Un PUT partiel : seuls les labels bougent, la description riche n'est
	// jamais renvoyée.
	payload := map[string]interface{}{"update": map[string]interface{}{"labels": ops}}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/rest/api/3/issue/"+url.PathEscape(issueKey), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.email+":"+c.token)))

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(respBody))
		if len(snippet) > jiraErrorBodyLimit {
			snippet = snippet[:jiraErrorBodyLimit]
		}
		return fmt.Errorf("Jira a refusé la mise à jour des labels de %s (%d): %s", issueKey, resp.StatusCode, snippet)
	}
	return nil
}

// SetIssueParent attaches a work item to an epic, or detaches it when parentKey
// is empty. acli's edit command exposes no --parent, so this only works over
// REST — the operation is what lets a roadmap epic be prototyped by pulling
// existing tickets into it.
func (c *JiraRESTClient) SetIssueParent(ctx context.Context, issueKey string, parentKey string) error {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return fmt.Errorf("clé de ticket manquante")
	}

	var parent interface{}
	if key := strings.TrimSpace(parentKey); key != "" {
		parent = map[string]string{"key": strings.ToUpper(key)}
	} else {
		// null détache : le ticket n'a plus d'épic.
		parent = nil
	}

	body, err := json.Marshal(map[string]interface{}{"fields": map[string]interface{}{"parent": parent}})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/rest/api/3/issue/"+url.PathEscape(issueKey), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.email+":"+c.token)))

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(respBody))
		if len(snippet) > jiraErrorBodyLimit {
			snippet = snippet[:jiraErrorBodyLimit]
		}
		return fmt.Errorf("Jira a refusé le rattachement de %s (%d): %s", issueKey, resp.StatusCode, snippet)
	}
	return nil
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
				ID        int    `json:"id"`
				Name      string `json:"name"`
				State     string `json:"state"`
				StartDate string `json:"startDate"`
				EndDate   string `json:"endDate"`
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
			out = append(out, models.TrackerSprint{
				ID:        fmt.Sprintf("%d", sp.ID),
				Name:      sp.Name,
				State:     strings.ToLower(sp.State),
				StartDate: sp.StartDate,
				EndDate:   sp.EndDate,
			})
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
		if len(snippet) > jiraErrorBodyLimit {
			snippet = snippet[:jiraErrorBodyLimit]
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
		if len(snippet) > jiraErrorBodyLimit {
			snippet = snippet[:jiraErrorBodyLimit]
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
// under one of several keys depending on the instance. The id is returned as
// well: it is the only key the team members endpoint accepts, and it must be
// kept verbatim — teams scoped to a site carry a suffix ("<uuid>-67") that is
// part of the id, not noise to trim.
func parseJiraTeamValue(raw json.RawMessage) (id string, name string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if v, ok := obj["id"].(string); ok {
			id = strings.TrimSpace(v)
		}
		for _, key := range []string{"name", "title", "value", "displayName"} {
			if v, ok := obj[key].(string); ok && strings.TrimSpace(v) != "" {
				return id, strings.TrimSpace(v)
			}
		}
		return id, ""
	}

	var asText string
	if err := json.Unmarshal(raw, &asText); err == nil {
		return "", strings.TrimSpace(asText)
	}
	return "", ""
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
// SearchProjectIssues reads a project's work items. sinceMinutes limits the read
// to what changed recently, which is what makes a background loop affordable: a
// full pass on a 1400 ticket project is fourteen paginated requests, an
// incremental one is a single request that usually answers nothing at all.
//
// The window is expressed in minutes rather than as a date, on purpose: JQL
// interprets a written date in the user's own time zone, and a background loop
// has no business guessing it.
func (c *JiraRESTClient) SearchProjectIssues(ctx context.Context, projectKey string, issueTypes []string, sinceMinutes int) (*JiraRESTSearchResult, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante")
	}
	if c.sprintFieldID == "" && c.teamFieldID == "" {
		// A missing Sprint or Team field is not fatal here: the rest of the
		// payload is still worth importing.
		_ = c.DiscoverFields(ctx)
	}

	quoted := make([]string, 0, len(issueTypes))
	for _, t := range NormalizeIssueTypes(issueTypes) {
		quoted = append(quoted, fmt.Sprintf("%q", t))
	}
	jql := fmt.Sprintf("project = %s AND issuetype IN (%s)", projectKey, strings.Join(quoted, ", "))
	if sinceMinutes > 0 {
		jql += fmt.Sprintf(" AND updated >= -%dm", sinceMinutes)
	}
	jql += " ORDER BY updated DESC"

	// created et updated sont demandés ici et nulle part ailleurs : acli refuse ces
	// deux champs, et sans eux le digest ne peut dire ni « ouvert depuis N jours »
	// ni « fermé récemment », faute de savoir quand quoi que ce soit s'est passé.
	// statuscategorychangedate dit depuis quand le ticket est dans sa catégorie de
	// statut. C'est ce qui répond à « en cours depuis combien de temps », et cela
	// voyage dans la même requête : le changelog, lui, coûterait un appel par
	// ticket, donc quinze cents pour un projet comme celui-ci.
	fields := []string{"key", "summary", "description", "status", "priority", "assignee", "labels", "issuetype", "parent", "created", "updated", "statuscategorychangedate"}
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
				values.TeamID, values.Team = parseJiraTeamValue(dynamic.Fields[c.teamFieldID])
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

// CreateIssue creates an issue and returns its key.
//
// La création passait par acli, qui sort en code zéro même quand Jira refuse :
// le motif du refus était perdu. Le REST renvoie le corps d'erreur de Jira, ce
// qui rend un champ obligatoire manquant immédiatement lisible.
func (c *JiraRESTClient) CreateIssue(ctx context.Context, projectKey, issueType, summary, parentKey string, extraFields map[string]string) (string, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	summary = strings.TrimSpace(summary)
	if projectKey == "" {
		return "", fmt.Errorf("clé de projet Jira manquante")
	}
	if summary == "" {
		return "", fmt.Errorf("intitulé manquant")
	}
	if strings.TrimSpace(issueType) == "" {
		issueType = "Epic"
	}

	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"issuetype": map[string]string{"name": issueType},
		"summary":   summary,
	}
	if strings.TrimSpace(parentKey) != "" {
		fields["parent"] = map[string]string{"key": strings.ToUpper(strings.TrimSpace(parentKey))}
	}

	// Champs imposés par l'instance, passés par identifiant d'option : c'est la
	// forme que Jira accepte pour une liste de choix.
	for id, optionID := range extraFields {
		id = strings.TrimSpace(id)
		optionID = strings.TrimSpace(optionID)
		if id == "" || optionID == "" {
			continue
		}
		fields[id] = map[string]string{"id": optionID}
	}

	body, err := c.post(ctx, "/rest/api/3/issue", map[string]interface{}{"fields": fields})
	if err != nil {
		return "", err
	}
	var created struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &created); err != nil || strings.TrimSpace(created.Key) == "" {
		return "", fmt.Errorf("clé du ticket créé introuvable dans la réponse de Jira")
	}
	return created.Key, nil
}

// JiraFieldOption is one allowed value of a mandatory creation field.
type JiraFieldOption struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// JiraRequiredField is a field an instance makes mandatory on creation, beyond
// the universal project, type and summary.
type JiraRequiredField struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Options []JiraFieldOption `json:"options"`
}

// RequiredCreateFields lists the mandatory creation fields of an issue type that
// Taskacao cannot guess.
//
// Une instance ajoute ce qu'elle veut : PE impose « Epic Type » sur ses épics, et
// la création échouait en 400 sans que rien ne le dise. Les champs à valeur par
// défaut sont écartés, Jira les remplit lui-même.
func (c *JiraRESTClient) RequiredCreateFields(ctx context.Context, projectKey, issueType string) ([]JiraRequiredField, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante")
	}
	if strings.TrimSpace(issueType) == "" {
		issueType = "Epic"
	}

	query := url.Values{}
	query.Set("projectKeys", projectKey)
	query.Set("issuetypeNames", issueType)
	query.Set("expand", "projects.issuetypes.fields")

	body, err := c.get(ctx, "/rest/api/3/issue/createmeta", query)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Projects []struct {
			IssueTypes []struct {
				Fields map[string]struct {
					Name            string `json:"name"`
					Required        bool   `json:"required"`
					HasDefaultValue bool   `json:"hasDefaultValue"`
					AllowedValues   []struct {
						ID    string `json:"id"`
						Value string `json:"value"`
						Name  string `json:"name"`
					} `json:"allowedValues"`
				} `json:"fields"`
			} `json:"issuetypes"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("métadonnées de création illisibles: %w", err)
	}

	// Ces trois là sont toujours fournis par l'appelant.
	known := map[string]bool{"project": true, "issuetype": true, "summary": true, "parent": true}

	out := []JiraRequiredField{}
	for _, proj := range payload.Projects {
		for _, it := range proj.IssueTypes {
			for id, f := range it.Fields {
				if !f.Required || known[id] || f.HasDefaultValue {
					continue
				}
				field := JiraRequiredField{ID: id, Name: f.Name}
				for _, av := range f.AllowedValues {
					label := av.Value
					if label == "" {
						label = av.Name
					}
					field.Options = append(field.Options, JiraFieldOption{ID: av.ID, Value: label})
				}
				out = append(out, field)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
