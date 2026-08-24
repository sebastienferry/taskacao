package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// The Sprint field cannot be read through acli's work item search: its --fields
// flag only accepts a fixed allow-list and rejects custom fields. The agile
// commands go the other way round though — they list the work items of a given
// sprint — so walking the project's scrum boards reconstructs the same mapping
// with no API token at all.

const (
	jiraSprintCLITimeout = 4 * time.Minute
	// jiraSprintCLIMaxSprints caps the walk: each sprint costs one acli call of
	// roughly six seconds, so an unbounded loop over a project with dozens of
	// backlog sprints would dwarf the sync itself. What gets skipped is logged.
	jiraSprintCLIMaxSprints = 8
)

type jiraBoardList struct {
	Values []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Location string `json:"location"`
	} `json:"values"`
}

type jiraSprintList struct {
	Sprints []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		State string `json:"state"`
	} `json:"sprints"`
}

type jiraSprintWorkItems struct {
	Issues []struct {
		Key string `json:"key"`
	} `json:"issues"`
}

// jiraSprintAssignment is one sprint and the work items it holds.
type jiraSprintAssignment struct {
	Name  string
	State string
	Keys  []string
}

// FetchSprintsViaCLI returns the sprint name of each work item of a project,
// using acli only. Active sprints come first, then future ones, because those
// are the ones a board is filtered on; closed sprints are left out.
func (r *Runner) FetchSprintsViaCLI(projectKey string, repoPath string) (map[string]string, []string, error) {
	projectKey = strings.ToUpper(strings.TrimSpace(projectKey))
	if projectKey == "" {
		return nil, nil, fmt.Errorf("clé de projet Jira manquante")
	}

	acliPath, _ := FindCliTool("acli")
	if acliPath == "" {
		acliPath = "acli"
	}

	ctx, cancel := context.WithTimeout(context.Background(), jiraSprintCLITimeout)
	defer cancel()

	notes := []string{}

	boardsOut, err := r.runCommand(ctx, repoPath, acliPath, "jira", "board", "search", "--project", projectKey, "--json")
	if err != nil {
		return nil, notes, fmt.Errorf("liste des boards Jira impossible: %w", err)
	}

	var boardValues []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := decodeJSONDocuments(boardsOut, func(dec *json.Decoder) error {
		var page jiraBoardList
		if err := dec.Decode(&page); err != nil {
			return err
		}
		for _, b := range page.Values {
			boardValues = append(boardValues, struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			}{ID: b.ID, Name: b.Name, Type: b.Type})
		}
		return nil
	}); err != nil {
		return nil, notes, fmt.Errorf("liste des boards illisible: %w", err)
	}

	scrumBoards := make([]int, 0, len(boardValues))
	for _, b := range boardValues {
		// Only a scrum board carries sprints; a kanban board has none.
		if strings.EqualFold(b.Type, "scrum") {
			scrumBoards = append(scrumBoards, b.ID)
		}
	}
	if len(scrumBoards) == 0 {
		return map[string]string{}, notes, fmt.Errorf("aucun board scrum sur le projet %s", projectKey)
	}

	// Collect the sprints of every scrum board, de-duplicated: the same sprint is
	// often shared by several boards of a project.
	seenSprint := make(map[int]bool)
	assignments := make([]jiraSprintAssignment, 0, jiraSprintCLIMaxSprints)
	type pendingSprint struct {
		boardID int
		id      int
		name    string
		state   string
	}
	pending := make([]pendingSprint, 0, 16)

	for _, boardID := range scrumBoards {
		out, sprintErr := r.runCommand(ctx, repoPath, acliPath, "jira", "board", "list-sprints",
			"--id", fmt.Sprintf("%d", boardID), "--state", "active,future", "--json")
		if sprintErr != nil {
			notes = append(notes, fmt.Sprintf("board %d ignoré: %v", boardID, sprintErr))
			continue
		}
		if err := decodeJSONDocuments(out, func(dec *json.Decoder) error {
			var page jiraSprintList
			if err := dec.Decode(&page); err != nil {
				return err
			}
			for _, sp := range page.Sprints {
				if seenSprint[sp.ID] || strings.TrimSpace(sp.Name) == "" {
					continue
				}
				seenSprint[sp.ID] = true
				pending = append(pending, pendingSprint{boardID: boardID, id: sp.ID, name: sp.Name, state: sp.State})
			}
			return nil
		}); err != nil {
			notes = append(notes, fmt.Sprintf("board %d: sprints illisibles (%v)", boardID, err))
			continue
		}
	}

	// Active sprints first: if the cap truncates the walk, the current sprint is
	// the one that must survive it.
	sort.SliceStable(pending, func(i, j int) bool {
		return jiraSprintStateRank(pending[i].state) < jiraSprintStateRank(pending[j].state)
	})

	skipped := 0
	for i, sp := range pending {
		if i >= jiraSprintCLIMaxSprints {
			skipped++
			continue
		}
		out, itemsErr := r.runCommand(ctx, repoPath, acliPath, "jira", "sprint", "list-workitems",
			"--sprint", fmt.Sprintf("%d", sp.id), "--board", fmt.Sprintf("%d", sp.boardID),
			"--fields", "key", "--paginate", "--json")
		if itemsErr != nil {
			notes = append(notes, fmt.Sprintf("sprint %q ignoré: %v", sp.name, itemsErr))
			continue
		}
		keys := make([]string, 0, 64)
		if err := decodeJSONDocuments(out, func(dec *json.Decoder) error {
			var page jiraSprintWorkItems
			if err := dec.Decode(&page); err != nil {
				return err
			}
			for _, issue := range page.Issues {
				if issue.Key != "" {
					keys = append(keys, issue.Key)
				}
			}
			return nil
		}); err != nil {
			notes = append(notes, fmt.Sprintf("sprint %q: tickets illisibles (%v)", sp.name, err))
			continue
		}
		assignments = append(assignments, jiraSprintAssignment{Name: sp.name, State: sp.state, Keys: keys})
	}

	if skipped > 0 {
		notes = append(notes, fmt.Sprintf("%d sprints au-delà des %d premiers non parcourus", skipped, jiraSprintCLIMaxSprints))
	}

	// An item can appear in several sprints (carried over). The active sprint
	// wins, which is what the sort above guarantees by writing it first.
	out := make(map[string]string)
	for _, a := range assignments {
		for _, key := range a.Keys {
			if _, already := out[key]; already {
				continue
			}
			out[key] = a.Name
		}
	}

	return out, notes, nil
}

func jiraSprintStateRank(state string) int {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return 0
	case "future":
		return 1
	default:
		return 2
	}
}

// decodeJSONDocuments walks every JSON document acli printed. With --paginate it
// emits one document per page, concatenated, so decoding only the first (or
// slicing to the last closing brace) silently loses pages or fails outright.
// Anything printed before the first document, such as an upgrade notice, is
// skipped. onDoc is called once per document; a document that fails to decode is
// reported and the walk continues.
func decodeJSONDocuments(raw string, onDoc func(dec *json.Decoder) error) error {
	trimmed := strings.TrimSpace(raw)
	start := strings.IndexAny(trimmed, "{[")
	if start < 0 {
		return fmt.Errorf("aucun document JSON dans la sortie d'acli")
	}

	dec := json.NewDecoder(strings.NewReader(trimmed[start:]))
	decoded := 0
	var firstErr error
	for {
		if err := onDoc(dec); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if firstErr == nil {
				firstErr = err
			}
			break
		}
		decoded++
	}

	if decoded == 0 {
		if firstErr != nil {
			return firstErr
		}
		return fmt.Errorf("sortie JSON d'acli vide")
	}
	return nil
}
