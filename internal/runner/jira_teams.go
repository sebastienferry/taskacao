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
	"sort"
	"strings"

	"tasks/internal/models"
)

// Atlassian teams do not live in the Jira API: a work item carries a team id in
// its Team field, but the people behind that id are served by the teams service,
// under /gateway/api/v4/teams. It takes the site id (the cloud id) as a query
// parameter and answers with account ids only, so the display names, e-mails and
// avatars come from a second call to the Jira user API.
//
// The whole chain works with the same Basic auth (e-mail plus API token) as the
// rest of the sync, which is what makes it usable here at all: the public
// organisation-scoped teams API would need an org admin token.

const (
	// jiraTeamMembersPageSize is the page the teams service serves; it answers
	// with a cursor when more members remain.
	jiraTeamMembersPageSize = 50
	// jiraTeamMembersMaxPages caps a runaway cursor. 20 pages is 1000 people in
	// a single team, far past anything real.
	jiraTeamMembersMaxPages = 20
	// jiraUserBulkBatch is the number of account ids asked per user lookup. The
	// Jira user bulk endpoint takes them as repeated query parameters, so the
	// batch also keeps the URL a sane length.
	jiraUserBulkBatch = 40
)

// CloudID returns the site id of the instance, needed by the teams endpoints.
func (c *JiraRESTClient) CloudID(ctx context.Context) (string, error) {
	if c.cloudID != "" {
		return c.cloudID, nil
	}

	body, err := c.get(ctx, "/_edge/tenant_info", nil)
	if err != nil {
		return "", fmt.Errorf("identifiant du site Jira illisible: %w", err)
	}

	var payload struct {
		CloudID string `json:"cloudId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("réponse tenant_info illisible: %w", err)
	}
	if strings.TrimSpace(payload.CloudID) == "" {
		return "", fmt.Errorf("le site Jira n'a pas renvoyé de cloudId")
	}
	c.cloudID = strings.TrimSpace(payload.CloudID)
	return c.cloudID, nil
}

// ListTeamMemberAccountIDs returns the account ids of a team's members. Only
// full members are kept: an invited-but-not-joined entry is not somebody work
// can be assigned to.
func (c *JiraRESTClient) ListTeamMemberAccountIDs(ctx context.Context, teamID string) ([]string, error) {
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return nil, fmt.Errorf("identifiant d'équipe manquant")
	}
	siteID, err := c.CloudID(ctx)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]string, 0, jiraTeamMembersPageSize)
	seen := map[string]bool{}
	cursor := ""

	for page := 0; page < jiraTeamMembersMaxPages; page++ {
		query := url.Values{}
		query.Set("siteId", siteID)
		query.Set("first", fmt.Sprintf("%d", jiraTeamMembersPageSize))
		if cursor != "" {
			query.Set("after", cursor)
		}

		body, err := c.get(ctx, "/gateway/api/v4/teams/"+url.PathEscape(teamID)+"/members", query)
		if err != nil {
			return accountIDs, err
		}

		var payload struct {
			Entities []struct {
				MembershipID struct {
					MemberID string `json:"memberId"`
				} `json:"membershipId"`
				State string `json:"state"`
			} `json:"entities"`
			Cursor *string `json:"cursor"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return accountIDs, fmt.Errorf("liste des membres de l'équipe illisible: %w", err)
		}

		for _, e := range payload.Entities {
			id := strings.TrimSpace(e.MembershipID.MemberID)
			if id == "" || seen[id] {
				continue
			}
			if e.State != "" && !strings.EqualFold(e.State, "FULL_MEMBER") {
				continue
			}
			seen[id] = true
			accountIDs = append(accountIDs, id)
		}

		if payload.Cursor == nil || strings.TrimSpace(*payload.Cursor) == "" || len(payload.Entities) == 0 {
			break
		}
		cursor = strings.TrimSpace(*payload.Cursor)
	}

	return accountIDs, nil
}

// LookupUsers resolves account ids into people. Deactivated accounts are kept
// and flagged rather than dropped: a ticket still shows the name of whoever owned
// it, and hiding the person would leave that assignee unexplained.
func (c *JiraRESTClient) LookupUsers(ctx context.Context, accountIDs []string) ([]models.TeamMember, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}

	members := make([]models.TeamMember, 0, len(accountIDs))
	for start := 0; start < len(accountIDs); start += jiraUserBulkBatch {
		end := start + jiraUserBulkBatch
		if end > len(accountIDs) {
			end = len(accountIDs)
		}

		query := url.Values{}
		for _, id := range accountIDs[start:end] {
			query.Add("accountId", id)
		}
		query.Set("maxResults", fmt.Sprintf("%d", jiraUserBulkBatch))

		body, err := c.get(ctx, "/rest/api/3/user/bulk", query)
		if err != nil {
			return members, err
		}

		var payload struct {
			Values []struct {
				AccountID    string            `json:"accountId"`
				DisplayName  string            `json:"displayName"`
				EmailAddress string            `json:"emailAddress"`
				Active       bool              `json:"active"`
				AccountType  string            `json:"accountType"`
				AvatarUrls   map[string]string `json:"avatarUrls"`
			} `json:"values"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return members, fmt.Errorf("réponse d'annuaire Jira illisible: %w", err)
		}

		for _, u := range payload.Values {
			if strings.TrimSpace(u.AccountID) == "" {
				continue
			}
			// An app account is a member of nothing a human would assign work to.
			if u.AccountType != "" && !strings.EqualFold(u.AccountType, "atlassian") {
				continue
			}
			avatar := u.AvatarUrls["48x48"]
			if avatar == "" {
				avatar = u.AvatarUrls["32x32"]
			}
			members = append(members, models.TeamMember{
				AccountID:   u.AccountID,
				DisplayName: strings.TrimSpace(u.DisplayName),
				Email:       strings.TrimSpace(u.EmailAddress),
				AvatarURL:   avatar,
				Active:      u.Active,
			})
		}
	}

	return members, nil
}

// FetchTeamMembers reads the people of one team, ready to be stored.
func (c *JiraRESTClient) FetchTeamMembers(ctx context.Context, team models.TrackerTeam) ([]models.TeamMember, error) {
	accountIDs, err := c.ListTeamMemberAccountIDs(ctx, team.ID)
	if err != nil {
		return nil, err
	}
	if len(accountIDs) == 0 {
		return nil, nil
	}

	members, err := c.LookupUsers(ctx, accountIDs)
	if err != nil {
		return members, err
	}
	for i := range members {
		members[i].TeamID = team.ID
		members[i].TeamName = team.Name
	}
	return members, nil
}

// AssignIssue sets the assignee of a work item, or clears it when accountID is
// empty. Jira assigns by account id only: a display name is rejected, which is
// why the member list has to be read before anybody can be picked.
func (c *JiraRESTClient) AssignIssue(ctx context.Context, issueKey string, accountID string) error {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return fmt.Errorf("clé de ticket manquante")
	}

	var payload map[string]interface{}
	if id := strings.TrimSpace(accountID); id != "" {
		payload = map[string]interface{}{"accountId": id}
	} else {
		// null désassigne, contrairement à la chaîne vide que Jira refuse.
		payload = map[string]interface{}{"accountId": nil}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/rest/api/3/issue/"+url.PathEscape(issueKey)+"/assignee", bytes.NewReader(body))
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
		return fmt.Errorf("Jira a refusé l'assignation de %s (%d): %s", issueKey, resp.StatusCode, snippet)
	}
	return nil
}

// SearchAssignableUsers looks up the people a work item can be assigned to. The
// endpoint is the very one Jira's own edit screen points at (the `assignee`
// field's autoCompleteUrl), so the answer is exactly who the instance accepts on
// that ticket, permissions included: a directory-wide search would offer people
// Jira then refuses.
func (c *JiraRESTClient) SearchAssignableUsers(ctx context.Context, issueKey string, query string, limit int) ([]models.TeamMember, error) {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil, fmt.Errorf("clé de ticket manquante")
	}
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{}
	params.Set("issueKey", issueKey)
	params.Set("query", strings.TrimSpace(query))
	params.Set("maxResults", fmt.Sprintf("%d", limit))

	body, err := c.get(ctx, "/rest/api/3/user/assignable/search", params)
	if err != nil {
		return nil, err
	}

	var payload []struct {
		AccountID    string            `json:"accountId"`
		DisplayName  string            `json:"displayName"`
		EmailAddress string            `json:"emailAddress"`
		Active       bool              `json:"active"`
		AccountType  string            `json:"accountType"`
		AvatarUrls   map[string]string `json:"avatarUrls"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("réponse de recherche d'utilisateurs illisible: %w", err)
	}

	out := make([]models.TeamMember, 0, len(payload))
	for _, u := range payload {
		if strings.TrimSpace(u.AccountID) == "" {
			continue
		}
		if u.AccountType != "" && !strings.EqualFold(u.AccountType, "atlassian") {
			continue
		}
		avatar := u.AvatarUrls["48x48"]
		if avatar == "" {
			avatar = u.AvatarUrls["32x32"]
		}
		out = append(out, models.TeamMember{
			AccountID:   u.AccountID,
			DisplayName: strings.TrimSpace(u.DisplayName),
			Email:       strings.TrimSpace(u.EmailAddress),
			AvatarURL:   avatar,
			Active:      u.Active,
		})
	}
	return out, nil
}

// SearchTeams looks up the teams of the instance by name. The teams service has
// no search endpoint reachable with a user token, but JQL autocompletion does
// the job and is the only one that answers both the label and the id, which is
// what writing the field needs. Its labels come back with <b> markers around the
// matched part, so they are stripped here.
func (c *JiraRESTClient) SearchTeams(ctx context.Context, query string) ([]models.TrackerTeam, error) {
	if err := c.DiscoverFields(ctx); err != nil && c.teamFieldID == "" {
		return nil, fmt.Errorf("champ Team introuvable sur cette instance: %w", err)
	}

	params := url.Values{}
	params.Set("fieldName", "Team")
	params.Set("fieldValue", strings.TrimSpace(query))

	body, err := c.get(ctx, "/rest/api/3/jql/autocompletedata/suggestions", params)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Results []struct {
			Value       string `json:"value"`
			DisplayName string `json:"displayName"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("réponse de recherche d'équipes illisible: %w", err)
	}

	out := make([]models.TrackerTeam, 0, len(payload.Results))
	for _, r := range payload.Results {
		id := strings.TrimSpace(r.Value)
		name := stripAutocompleteMarkup(r.DisplayName)
		if id == "" || name == "" {
			continue
		}
		out = append(out, models.TrackerTeam{ID: id, Name: name})
	}
	return out, nil
}

// stripAutocompleteMarkup removes the <b> markers JQL autocompletion wraps the
// matched substring in.
func stripAutocompleteMarkup(label string) string {
	label = strings.ReplaceAll(label, "<b>", "")
	label = strings.ReplaceAll(label, "</b>", "")
	return strings.TrimSpace(label)
}

// SetIssueTeam writes the Team field of a work item, or clears it when teamID is
// empty. The field accepts the team id; instances differ on whether it wants the
// bare id or an object carrying it, so the object form is tried when the bare
// one is refused rather than guessing once and failing.
func (c *JiraRESTClient) SetIssueTeam(ctx context.Context, issueKey string, teamID string) error {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return fmt.Errorf("clé de ticket manquante")
	}
	if err := c.DiscoverFields(ctx); err != nil && c.teamFieldID == "" {
		return fmt.Errorf("champ Team introuvable sur cette instance: %w", err)
	}
	if c.teamFieldID == "" {
		return fmt.Errorf("champ Team introuvable sur cette instance")
	}

	teamID = strings.TrimSpace(teamID)
	var values []interface{}
	if teamID == "" {
		values = []interface{}{nil}
	} else {
		values = []interface{}{teamID, map[string]string{"id": teamID}}
	}

	var lastErr error
	for _, value := range values {
		err := c.putIssueFields(ctx, issueKey, map[string]interface{}{c.teamFieldID: value})
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

// putIssueFields sends a partial field update on a work item.
func (c *JiraRESTClient) putIssueFields(ctx context.Context, issueKey string, fields map[string]interface{}) error {
	body, err := json.Marshal(map[string]interface{}{"fields": fields})
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
		return fmt.Errorf("Jira a refusé la mise à jour de %s (%d): %s", issueKey, resp.StatusCode, snippet)
	}
	return nil
}

// MoveIssuesToSprint puts work items into a sprint, or back into the backlog when
// sprintID is empty. Sprint membership is not an ordinary field: on a Jira board
// it is owned by the Agile API, which takes the sprint id and moves up to fifty
// keys per call.
func (c *JiraRESTClient) MoveIssuesToSprint(ctx context.Context, sprintID string, issueKeys []string) error {
	keys := make([]string, 0, len(issueKeys))
	for _, k := range issueKeys {
		if k = strings.TrimSpace(strings.ToUpper(k)); k != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return fmt.Errorf("aucun ticket à déplacer")
	}

	path := "/rest/agile/1.0/backlog/issue"
	if id := strings.TrimSpace(sprintID); id != "" {
		path = "/rest/agile/1.0/sprint/" + url.PathEscape(id) + "/issue"
	}

	// Le lot est borné par l'API elle-même : cinquante clés par appel.
	const batch = 50
	for start := 0; start < len(keys); start += batch {
		end := start + batch
		if end > len(keys) {
			end = len(keys)
		}
		if _, err := c.post(ctx, path, map[string]interface{}{"issues": keys[start:end]}); err != nil {
			return err
		}
	}
	return nil
}

// ListProjectIssueTypes returns the work item types a Jira project exposes, so
// the project settings can offer them instead of asking the user to type a name.
// The list comes from the creation metadata, which is what the instance itself
// uses to build its own type picker.
func (c *JiraRESTClient) ListProjectIssueTypes(ctx context.Context, projectKey string) ([]string, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, fmt.Errorf("clé de projet Jira manquante")
	}

	query := url.Values{}
	query.Set("projectKeys", projectKey)

	body, err := c.get(ctx, "/rest/api/3/issue/createmeta", query)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Projects []struct {
			IssueTypes []struct {
				Name    string `json:"name"`
				Subtask bool   `json:"subtask"`
			} `json:"issuetypes"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("types de tickets illisibles: %w", err)
	}

	out := []string{}
	seen := map[string]bool{}
	for _, proj := range payload.Projects {
		for _, it := range proj.IssueTypes {
			name := strings.TrimSpace(it.Name)
			// Les sous-tâches ne sont pas des cartes de board : elles vivent sous
			// leur parent, qui est lui même importé.
			if name == "" || it.Subtask || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}
