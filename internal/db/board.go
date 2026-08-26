package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tasks/internal/models"
	"tasks/internal/runner"
)

// Board columns are Jira-like: a column is a name plus the tracker statuses it
// groups, and the project owns them. Importing a tracker board is only a
// starting point — columns can then be created, renamed, reordered and have
// statuses reassigned freely, and each agentic workflow stage is assigned to one
// or several of them.

const boardAPITimeout = 60 * time.Second

func (d *DB) jiraRESTClientForProject(projectID string) (*runner.JiraRESTClient, *models.Project, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, nil, fmt.Errorf("projet non trouvé")
	}
	settings, _ := d.GetSettings()
	client := runner.NewJiraRESTClient(settings, proj.TrackerUrl)
	if client == nil {
		return nil, proj, fmt.Errorf("accès à l'API Jira non configuré : renseignez le site, l'e-mail et le jeton dans les réglages")
	}
	return client, proj, nil
}

// ListProjectTrackerBoards returns the tracker boards attached to a project.
func (d *DB) ListProjectTrackerBoards(projectID string) ([]models.TrackerBoard, error) {
	client, proj, err := d.jiraRESTClientForProject(projectID)
	if err != nil {
		return nil, err
	}
	key := jiraProjectKeyFor(proj)
	if key == "" {
		if settings, _ := d.GetSettings(); settings != nil {
			key = settings.JiraProject
		}
	}
	if key == "" {
		return nil, fmt.Errorf("clé de projet Jira absente de la configuration du projet")
	}

	ctx, cancel := context.WithTimeout(context.Background(), boardAPITimeout)
	defer cancel()
	return client.ListBoards(ctx, key)
}

// ImportProjectBoardColumns retains a board for the project then refreshes from
// it. Le bouton « Détecter » et la synchro doivent donner le même résultat, donc
// les deux passent par la même fusion : elle préserve les statuts affectés à la
// main et les colonnes masquées, et ramène les sprints avec leur état.
func (d *DB) ImportProjectBoardColumns(projectID string, boardID string) (*models.Project, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}

	boardID = strings.TrimSpace(boardID)
	if boardID == "" {
		boardID = proj.BoardID
	}
	if boardID == "" {
		return nil, fmt.Errorf("aucun board sélectionné")
	}

	if boardID != proj.BoardID {
		if _, err := d.UpdateProject(proj.ID, models.UpdateProjectRequest{BoardID: &boardID}); err != nil {
			return nil, err
		}
	}

	if _, err := d.SyncProjectBoardColumns(proj.ID); err != nil {
		return nil, err
	}
	return d.GetProjectByID(proj.ID)
}

// GetProjectTrackerStatuses lists the tracker statuses actually seen on the
// project's tickets, plus those already assigned to a column. The point is to
// offer real values in the assignment UI rather than the instance's full status
// list, most of which never appears on this project.
func (d *DB) GetProjectTrackerStatuses(projectID string) ([]string, error) {
	proj, err := d.GetProjectByID(projectID)
	if err != nil || proj == nil {
		return nil, fmt.Errorf("projet non trouvé")
	}

	d.mu.RLock()
	rows, err := d.conn.Query(`
		SELECT DISTINCT tracker_status FROM tasks
		WHERE tracker_status != ''
		  AND (project_id = ? OR project_id = (SELECT slug FROM projects WHERE id = ?) OR project_id = (SELECT id FROM projects WHERE slug = ?))
		ORDER BY tracker_status ASC
	`, proj.ID, proj.ID, proj.ID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}

	for _, col := range proj.TrackerColumns {
		for _, name := range col.Statuses {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}

// MoveTaskToTrackerStatus transitions a ticket to a status named as the tracker
// spells it, then records it locally. Synchronous: the caller moved the card
// optimistically and needs to know whether the tracker accepted the move.
func (d *DB) MoveTaskToTrackerStatus(taskIDOrKey string, statusName string) (*models.Task, error) {
	statusName = strings.TrimSpace(statusName)
	if statusName == "" {
		return nil, fmt.Errorf("statut cible manquant")
	}

	task, err := d.GetTaskByID(taskIDOrKey)
	if err != nil || task == nil {
		return nil, fmt.Errorf("tâche non trouvée")
	}

	source := task.Source
	if source == "" && task.ProjectID != "" {
		if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil {
			source = proj.IssueTracker
		}
	}
	if source != "jira" {
		return nil, fmt.Errorf("le déplacement par colonne de tracker n'est disponible que sur un ticket Jira")
	}

	repoPath := d.ResolveTaskRepoPath(task)
	settings, _ := d.GetSettings()
	trackerURL := ""
	if task.ProjectID != "" {
		if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil {
			trackerURL = proj.TrackerUrl
		}
	}
	if err := d.runner.TransitionJiraIssueToStatus(settings, trackerURL, task.Key, statusName, repoPath); err != nil {
		return nil, err
	}

	d.mu.Lock()
	_, execErr := d.conn.Exec("UPDATE tasks SET tracker_status = ?, updated_at = ? WHERE id = ?", statusName, time.Now(), task.ID)
	d.mu.Unlock()
	if execErr != nil {
		return nil, execErr
	}

	return d.GetTaskByID(task.ID)
}

// SyncProjectBoardColumns refreshes a project's columns from its tracker board,
// merging rather than overwriting: the column list and their order come from the
// tracker, while the statuses a user assigned by hand to a column of the same
// name are kept — as long as the tracker does not claim them elsewhere. A status
// the tracker maps nowhere, such as a workflow status absent from the board,
// therefore stays where the user put it.
//
// Called at the end of a Jira sync, so the board follows the tracker without a
// manual import.
func (d *DB) SyncProjectBoardColumns(projectID string) (string, error) {
	client, proj, err := d.jiraRESTClientForProject(projectID)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), boardAPITimeout)
	defer cancel()

	boardID := strings.TrimSpace(proj.BoardID)
	if boardID == "" {
		// No board chosen yet: the first scrum board of the project is the one
		// that carries columns worth mirroring.
		key := jiraProjectKeyFor(proj)
		if key == "" {
			if settings, _ := d.GetSettings(); settings != nil {
				key = settings.JiraProject
			}
		}
		if key == "" {
			return "", fmt.Errorf("clé de projet Jira absente")
		}
		boards, err := client.ListBoards(ctx, key)
		if err != nil {
			return "", err
		}
		for _, b := range boards {
			if strings.EqualFold(b.Type, "scrum") {
				boardID = b.ID
				break
			}
		}
		if boardID == "" && len(boards) > 0 {
			boardID = boards[0].ID
		}
		if boardID == "" {
			return "", fmt.Errorf("aucun board sur le projet %s", key)
		}
	}

	remote, err := client.FetchBoardColumns(ctx, boardID)
	if err != nil {
		return "", err
	}

	// Statuses the tracker assigns, so a manual assignment cannot duplicate one.
	claimed := map[string]bool{}
	for _, col := range remote {
		for _, st := range col.Statuses {
			claimed[strings.ToLower(st)] = true
		}
	}

	previousByName := map[string][]string{}
	// Le masquage est un choix d'affichage de l'utilisateur : il doit survivre à
	// une relecture du board, qui ne connaît que noms, ordre et statuts.
	previousHidden := map[string]bool{}
	for _, col := range proj.TrackerColumns {
		previousByName[strings.ToLower(col.Name)] = col.Statuses
		previousHidden[strings.ToLower(col.Name)] = col.Hidden
	}

	merged := make([]models.TrackerColumn, 0, len(remote))
	for _, col := range remote {
		statuses := append([]string{}, col.Statuses...)
		seen := map[string]bool{}
		for _, st := range statuses {
			seen[strings.ToLower(st)] = true
		}
		for _, st := range previousByName[strings.ToLower(col.Name)] {
			key := strings.ToLower(st)
			if seen[key] || claimed[key] {
				continue
			}
			statuses = append(statuses, st)
			seen[key] = true
		}
		merged = append(merged, models.TrackerColumn{Name: col.Name, Statuses: statuses, Hidden: previousHidden[strings.ToLower(col.Name)]})
	}

	// Columns the user created and the tracker does not know are kept at the end
	// rather than silently dropped: they may hold statuses nothing else claims.
	remoteNames := map[string]bool{}
	for _, col := range remote {
		remoteNames[strings.ToLower(col.Name)] = true
	}
	for _, col := range proj.TrackerColumns {
		if remoteNames[strings.ToLower(col.Name)] {
			continue
		}
		kept := []string{}
		for _, st := range col.Statuses {
			if !claimed[strings.ToLower(st)] {
				kept = append(kept, st)
			}
		}
		if len(kept) > 0 {
			merged = append(merged, models.TrackerColumn{Name: col.Name, Statuses: kept, Hidden: col.Hidden})
		}
	}

	// Stage assignments are keyed by column name: drop the ones whose column
	// disappeared, keep the rest untouched.
	names := map[string]bool{}
	for _, col := range merged {
		names[col.Name] = true
	}
	stages := map[string][]string{}
	for stage, cols := range proj.StageColumns {
		kept := []string{}
		for _, c := range cols {
			if names[c] {
				kept = append(kept, c)
			}
		}
		if len(kept) > 0 {
			stages[stage] = kept
		}
	}

	// Les sprints suivent au même moment : leur état est ce qui distingue NOW
	// (sprint actif) de NEXT (sprint futur) dans la roadmap.
	sprints, sprintErr := client.ListBoardSprints(ctx, boardID)
	if sprintErr != nil {
		sprints = proj.Sprints
	}

	if _, err := d.UpdateProject(proj.ID, models.UpdateProjectRequest{
		BoardID:        &boardID,
		TrackerColumns: &merged,
		StageColumns:   &stages,
		Sprints:        &sprints,
	}); err != nil {
		return "", err
	}

	note := fmt.Sprintf("%d colonnes du board %s reprises", len(merged), boardID)
	if sprintErr == nil {
		active := 0
		for _, sp := range sprints {
			if sp.State == "active" {
				active++
			}
		}
		note = fmt.Sprintf("%s, %d sprints (%d actifs)", note, len(sprints), active)
	}
	return note, nil
}

// Le mapping du projet fait le lien entre statut du tracker, colonne et étape du
// workflow agentique. Ces deux fonctions le traversent dans les deux sens, pour
// que changer l'un des deux côtés dans la fiche mette l'autre à jour.

var stageToInternalStatus = map[string]models.Status{
	"new":         models.StatusToClarify,
	"clarified":   models.StatusToSpecify,
	"specified":   models.StatusToImplement,
	"implemented": models.StatusToTest,
	"reviewed":    models.StatusToClose,
	"finished":    models.StatusFinished,
}

var workflowStageOrder = []string{"new", "clarified", "specified", "implemented", "reviewed", "finished"}

// StageForTrackerStatus returns the workflow stage a tracker status belongs to,
// through the column that groups it. Empty when the project has no mapping for
// it, in which case the caller keeps whatever it had.
func StageForTrackerStatus(proj *models.Project, trackerStatus string) string {
	trackerStatus = strings.ToLower(strings.TrimSpace(trackerStatus))
	if proj == nil || trackerStatus == "" {
		return ""
	}
	column := ""
	for _, col := range proj.TrackerColumns {
		for _, st := range col.Statuses {
			if strings.ToLower(st) == trackerStatus {
				column = col.Name
				break
			}
		}
		if column != "" {
			break
		}
	}
	if column == "" {
		return ""
	}
	// Plusieurs étapes sur une colonne : la moins avancée, celle qui reste à
	// faire, comme côté interface.
	for _, stage := range workflowStageOrder {
		for _, name := range proj.StageColumns[stage] {
			if name == column {
				return stage
			}
		}
	}
	return ""
}

// TrackerStatusForStage returns the tracker status a workflow stage lands on:
// the first status of the first column that stage is assigned to.
func TrackerStatusForStage(proj *models.Project, stage string) string {
	stage = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(stage), "#"))
	if proj == nil || stage == "" {
		return ""
	}
	for _, columnName := range proj.StageColumns[stage] {
		for _, col := range proj.TrackerColumns {
			if col.Name == columnName && len(col.Statuses) > 0 {
				return col.Statuses[0]
			}
		}
	}
	return ""
}

// InternalStatusForStage folds a workflow stage onto the six internal statuses,
// which the generic views and the existing filters still use.
func InternalStatusForStage(stage string) (models.Status, bool) {
	stage = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(stage), "#"))
	st, ok := stageToInternalStatus[stage]
	return st, ok
}

// Le pas suivant du workflow agentique, côté serveur. Le front a la même
// résolution, mais la chaîne autonome tourne dans le worker : elle ne peut pas
// dépendre de l'interface, qui peut être fermée.

// StageOfTask returns a task's workflow stage: the column it sits in when the
// project maps stages to columns, its workflow label otherwise.
func (d *DB) StageOfTask(task *models.Task) string {
	if task == nil {
		return ""
	}
	if task.ProjectID != "" {
		if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil {
			if stage := StageForTrackerStatus(proj, task.TrackerStatus); stage != "" {
				return stage
			}
		}
	}
	for _, stage := range workflowStageOrder {
		for _, label := range task.Labels {
			if strings.EqualFold(strings.TrimPrefix(label, "#"), stage) {
				return stage
			}
		}
	}

	// Repli sur le statut interne, comme le fait l'interface. Sans lui, un
	// ticket clos sans label de workflow était rendu comme « new », et l'app
	// proposait de clarifier un ticket déjà terminé.
	switch task.Status {
	case models.StatusFinished, models.StatusDone:
		return "finished"
	case models.StatusToClose:
		return "reviewed"
	case models.StatusToTest, models.StatusToValidate:
		return "implemented"
	case models.StatusToImplement, models.StatusInProgress:
		return "specified"
	case models.StatusToSpecify, models.StatusSpecified:
		return "clarified"
	}
	return "new"
}

// StageStep describes what advancing one step from a stage means: which skill
// runs, and whether it is interactive. Clarification is the one step that needs a
// human in the loop, so it opens a TTY instead of running headless.
type StageStep struct {
	SkillID     string
	Interactive bool
	Label       string
}

var stageSteps = map[string]StageStep{
	"new":         {SkillID: "clarify", Interactive: true, Label: "Clarifier en session TTY"},
	"clarified":   {SkillID: "specify", Interactive: false, Label: "Spécifier en autonomie"},
	"specified":   {SkillID: "implement", Interactive: false, Label: "Implémenter en autonomie"},
	"implemented": {SkillID: "create_pr", Interactive: false, Label: "Créer la MR, la fusion reste manuelle"},
	"reviewed":    {SkillID: "handoff", Interactive: false, Label: "Handoff et nettoyage local"},
}

// NextStep returns the step to run from a stage, and whether there is one.
func NextStep(stage string) (StageStep, bool) {
	step, ok := stageSteps[strings.ToLower(strings.TrimSpace(stage))]
	return step, ok
}

// AutonomousStopStage is where an autonomous run stops on its own: the code is
// written and waiting for the user's review. Going further would create the MR
// and close the ticket without a human ever looking at the diff.
const AutonomousStopStage = "implemented"
