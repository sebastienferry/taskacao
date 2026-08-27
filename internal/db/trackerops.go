package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tasks/internal/models"
	"tasks/internal/runner"
)

// Every write on an existing work item goes through the activity queue, like the
// field sync already did: a tracker call takes between one and several seconds,
// and a batch of them (splitting an epic over twenty stories) blocks an HTTP
// request long past what a UI can wait for. Queueing them also gives each write
// the one thing a synchronous call never had: a trace of what was attempted, and
// a readable failure when the tracker refuses.
//
// The local state is written first, so the board shows the intent immediately,
// and the queued job is what mirrors it onto the tracker.

type TrackerOpKind string

const (
	// TrackerOpAssign sets or clears the assignee of a work item.
	TrackerOpAssign TrackerOpKind = "assign"
	// TrackerOpSetParent attaches one work item to an epic, or detaches it.
	TrackerOpSetParent TrackerOpKind = "set_parent"
	// TrackerOpMoveToEpic moves a batch of work items to an epic, created on the
	// fly when only a title is given. This is the epic split.
	TrackerOpMoveToEpic TrackerOpKind = "move_to_epic"
	// TrackerOpEpicHorizon mirrors the roadmap horizon of one epic as a label.
	TrackerOpEpicHorizon TrackerOpKind = "epic_horizon"
	// TrackerOpPushHorizons mirrors every locally classified epic whose label is
	// missing or stale.
	TrackerOpPushHorizons TrackerOpKind = "push_horizons"
	// TrackerOpTransition moves a work item to a status named as the tracker
	// spells it, which is what dropping a card in a board column does.
	TrackerOpTransition TrackerOpKind = "transition"
	// TrackerOpSetTeam writes the team of a work item, or clears it.
	TrackerOpSetTeam TrackerOpKind = "set_team"
	// TrackerOpSetSprint moves work items into a sprint, or back to the backlog.
	TrackerOpSetSprint TrackerOpKind = "set_sprint"
)

// TrackerOp is one queued write on the tracker. Only the fields its Kind uses
// are set.
type TrackerOp struct {
	Kind      TrackerOpKind
	ProjectID string
	// TaskID / TaskKey identify the single work item of an assign or set_parent.
	TaskID  string
	TaskKey string
	// TaskIDs is the batch of a move_to_epic.
	TaskIDs []string
	// AccountID / AssigneeName describe the target of an assign. An empty
	// AccountID unassigns.
	AccountID    string
	AssigneeName string
	// EpicKey / NewEpicTitle / Fields describe the target of a set_parent or a
	// move_to_epic. An empty EpicKey on a set_parent detaches the work item.
	EpicKey      string
	NewEpicTitle string
	Fields       map[string]string
	// Horizon is the roadmap horizon of an epic_horizon.
	Horizon string
	// TargetStatus is the tracker status of a transition, in the tracker's own
	// spelling ("Dev Test", "To Merge").
	TargetStatus string
	// TeamID / TeamName describe the target of a set_team. An empty TeamID clears
	// the field, which is legitimate: the team is never mandatory.
	TeamID   string
	TeamName string
	// SprintID / SprintName describe the target of a set_sprint. An empty
	// SprintID sends the work items back to the backlog.
	SprintID   string
	SprintName string
}

// EnqueueTrackerOp records the activity and hands the write to the worker. The
// returned activity is what the caller shows: the operation itself has not run
// yet.
func (d *DB) EnqueueTrackerOp(op TrackerOp) (*models.TaskActivity, error) {
	act, job, err := buildTrackerOpJob(op)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	err = d.addTaskActivityDirect(*act)
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}

	d.pushTrackerOpJob(job)
	return act, nil
}

// enqueueTrackerOpUnsafe is the same, for a caller already holding the write
// lock. UpdateTask is one: it writes the local state and queues the tracker
// write in the same critical section.
func (d *DB) enqueueTrackerOpUnsafe(op TrackerOp) (*models.TaskActivity, error) {
	act, job, err := buildTrackerOpJob(op)
	if err != nil {
		return nil, err
	}
	if err := d.addTaskActivityDirect(*act); err != nil {
		return nil, err
	}
	d.pushTrackerOpJob(job)
	return act, nil
}

func (d *DB) pushTrackerOpJob(job SkillJob) {
	select {
	case d.jobQueue <- job:
	default:
		// File pleine : la remise en file dans une goroutine évite de bloquer la
		// requête HTTP, comme le fait déjà la synchro des champs.
		go func() {
			d.jobQueue <- job
		}()
	}
}

func buildTrackerOpJob(op TrackerOp) (*models.TaskActivity, SkillJob, error) {
	if strings.TrimSpace(string(op.Kind)) == "" {
		return nil, SkillJob{}, fmt.Errorf("opération tracker inconnue")
	}

	activityID := uuid.New().String()
	now := time.Now()

	taskID := strings.TrimSpace(op.TaskID)
	if taskID == "" {
		// Une opération de lot n'appartient à aucun ticket : elle est rattachée
		// au projet, comme le sont les synchros.
		taskID = "tracker-op-" + op.ProjectID
	}

	var action, summary string
	steps := []string{}

	switch op.Kind {
	case TrackerOpAssign:
		who := strings.TrimSpace(op.AssigneeName)
		if who == "" {
			who = "personne"
		}
		action = fmt.Sprintf("Assignation de %s", op.TaskKey)
		summary = fmt.Sprintf("Assignation de %s à %s en file d'attente", op.TaskKey, who)
		steps = append(steps, fmt.Sprintf("Cible : %s ➔ %s", op.TaskKey, who))
	case TrackerOpSetParent:
		if strings.TrimSpace(op.EpicKey) == "" {
			action = fmt.Sprintf("Détachement de %s de son épic", op.TaskKey)
			summary = fmt.Sprintf("Détachement de %s en file d'attente", op.TaskKey)
			steps = append(steps, fmt.Sprintf("Cible : %s ➔ aucun épic", op.TaskKey))
		} else {
			action = fmt.Sprintf("Rattachement de %s à %s", op.TaskKey, op.EpicKey)
			summary = fmt.Sprintf("Rattachement de %s à l'épic %s en file d'attente", op.TaskKey, op.EpicKey)
			steps = append(steps, fmt.Sprintf("Cible : %s ➔ %s", op.TaskKey, op.EpicKey))
		}
	case TrackerOpMoveToEpic:
		target := strings.TrimSpace(op.EpicKey)
		if target == "" {
			target = fmt.Sprintf("nouvel épic « %s »", strings.TrimSpace(op.NewEpicTitle))
		}
		action = fmt.Sprintf("Découpe d'épic : %d ticket(s) ➔ %s", len(op.TaskIDs), target)
		summary = fmt.Sprintf("Déplacement de %d ticket(s) vers %s en file d'attente", len(op.TaskIDs), target)
		steps = append(steps, fmt.Sprintf("Cible : %s", target), fmt.Sprintf("%d ticket(s) à déplacer", len(op.TaskIDs)))
	case TrackerOpEpicHorizon:
		action = fmt.Sprintf("Horizon de %s ➔ %s", op.EpicKey, op.Horizon)
		summary = fmt.Sprintf("Label d'horizon de %s en file d'attente", op.EpicKey)
		steps = append(steps, fmt.Sprintf("Cible : %s ➔ %s", op.EpicKey, op.Horizon))
	case TrackerOpTransition:
		action = fmt.Sprintf("Transition de %s ➔ %s", op.TaskKey, op.TargetStatus)
		summary = fmt.Sprintf("Transition de %s vers « %s » en file d'attente", op.TaskKey, op.TargetStatus)
		steps = append(steps, fmt.Sprintf("Cible : %s ➔ %s", op.TaskKey, op.TargetStatus))
	case TrackerOpSetTeam:
		label := op.TeamName
		if label == "" {
			label = op.TeamID
		}
		if strings.TrimSpace(op.TeamID) == "" {
			label = "aucune équipe"
		}
		count := len(op.TaskIDs)
		if count == 0 {
			count = 1
		}
		if count == 1 {
			action = fmt.Sprintf("Équipe de %s ➔ %s", op.TaskKey, label)
			summary = fmt.Sprintf("Changement d'équipe de %s en file d'attente", op.TaskKey)
		} else {
			action = fmt.Sprintf("Équipe de %d ticket(s) ➔ %s", count, label)
			summary = fmt.Sprintf("Changement d'équipe de %d ticket(s) en file d'attente", count)
		}
		steps = append(steps, fmt.Sprintf("Cible : %s", label))
	case TrackerOpSetSprint:
		target := strings.TrimSpace(op.SprintName)
		if strings.TrimSpace(op.SprintID) == "" {
			target = "backlog"
		} else if target == "" {
			target = op.SprintID
		}
		count := len(op.TaskIDs)
		if count == 0 {
			count = 1
		}
		if count == 1 {
			action = fmt.Sprintf("Sprint de %s ➔ %s", op.TaskKey, target)
			summary = fmt.Sprintf("Changement de sprint de %s en file d'attente", op.TaskKey)
		} else {
			action = fmt.Sprintf("Sprint de %d ticket(s) ➔ %s", count, target)
			summary = fmt.Sprintf("Changement de sprint de %d ticket(s) en file d'attente", count)
		}
		steps = append(steps, fmt.Sprintf("Cible : %s", target))
	case TrackerOpPushHorizons:
		action = "Horizons de roadmap ➔ labels Jira"
		summary = "Mise à jour des labels d'horizon en file d'attente"
		steps = append(steps, "Cible : tous les épics classés localement dont le label est absent ou obsolète")
	default:
		return nil, SkillJob{}, fmt.Errorf("opération tracker inconnue : %s", op.Kind)
	}
	steps = append(steps, "Poussée dans la file d'attente d'exécution...")

	act := models.TaskActivity{
		ID:        activityID,
		TaskID:    taskID,
		TaskKey:   op.TaskKey,
		SkillID:   "tracker_op",
		SkillName: "Écriture tracker",
		Action:    action,
		Status:    string(models.ActivityStatusQueued),
		Summary:   summary,
		Steps:     steps,
		CreatedAt: now,
	}

	opCopy := op
	job := SkillJob{
		ActivityID: activityID,
		TaskID:     taskID,
		ProjectID:  op.ProjectID,
		SkillID:    "tracker_op",
		Op:         &opCopy,
	}
	return &act, job, nil
}

// processTrackerOpJob runs one queued tracker write.
func (d *DB) processTrackerOpJob(ctx context.Context, job SkillJob) {
	if job.Op == nil {
		d.finishTrackerOp(job.ActivityID, nil, "", fmt.Errorf("opération absente du job"))
		return
	}
	op := *job.Op
	steps := []string{}

	var output string
	var err error

	switch op.Kind {
	case TrackerOpAssign:
		output, err = d.runAssignOp(ctx, op, &steps)
	case TrackerOpSetParent:
		output, err = d.runSetParentOp(op, &steps)
	case TrackerOpMoveToEpic:
		output, err = d.runMoveToEpicOp(op, &steps)
	case TrackerOpEpicHorizon:
		output, err = d.runEpicHorizonOp(op, &steps)
	case TrackerOpPushHorizons:
		output, err = d.runPushHorizonsOp(op, &steps)
	case TrackerOpTransition:
		output, err = d.runTransitionOp(op, &steps)
	case TrackerOpSetTeam:
		output, err = d.runSetTeamOp(op, &steps)
	case TrackerOpSetSprint:
		output, err = d.runSetSprintOp(op, &steps)
	default:
		err = fmt.Errorf("opération tracker inconnue : %s", op.Kind)
	}

	d.finishTrackerOp(job.ActivityID, steps, output, err)
}

func (d *DB) runAssignOp(ctx context.Context, op TrackerOp, steps *[]string) (string, error) {
	task, err := d.GetTaskByID(op.TaskID)
	if err != nil || task == nil {
		return "", fmt.Errorf("tâche introuvable")
	}
	if task.Source != "jira" {
		return "Tâche non Jira : l'assignation reste locale", nil
	}

	client, err := d.jiraRESTClientForTask(task)
	if err != nil {
		return "", err
	}

	who := strings.TrimSpace(op.AssigneeName)
	accountID := strings.TrimSpace(op.AccountID)
	if accountID == "" && who != "" {
		// Nom choisi hors liste (saisie libre, ou membre d'une autre équipe) :
		// l'identifiant de compte se retrouve dans les équipes connues.
		accountID = d.AccountIDForAssignee(who, task.Team)
		if accountID == "" {
			return "", fmt.Errorf("aucun compte Jira connu pour « %s » : synchronisez l'équipe du ticket, ou choisissez une personne dans la liste", who)
		}
		*steps = append(*steps, fmt.Sprintf("Compte résolu depuis les équipes connues : %s", accountID))
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.AssignIssue(callCtx, task.Key, accountID); err != nil {
		return "", err
	}

	if accountID == "" {
		*steps = append(*steps, fmt.Sprintf("✅ %s désassigné sur Jira", task.Key))
		return fmt.Sprintf("%s n'a plus d'assigné sur Jira", task.Key), nil
	}
	*steps = append(*steps, fmt.Sprintf("✅ %s assigné à %s sur Jira", task.Key, who))
	return fmt.Sprintf("%s assigné à %s sur Jira", task.Key, who), nil
}

func (d *DB) runSetParentOp(op TrackerOp, steps *[]string) (string, error) {
	task, err := d.applyTaskEpic(op.TaskID, op.EpicKey, steps)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(op.EpicKey) == "" {
		return fmt.Sprintf("%s détaché de son épic", task.Key), nil
	}
	return fmt.Sprintf("%s rattaché à l'épic %s", task.Key, strings.ToUpper(strings.TrimSpace(op.EpicKey))), nil
}

func (d *DB) runMoveToEpicOp(op TrackerOp, steps *[]string) (string, error) {
	if len(op.TaskIDs) == 0 {
		return "", fmt.Errorf("aucun ticket sélectionné")
	}

	targetEpicKey := strings.ToUpper(strings.TrimSpace(op.EpicKey))
	if targetEpicKey == "" {
		if strings.TrimSpace(op.NewEpicTitle) == "" {
			return "", fmt.Errorf("épic cible ou intitulé du nouvel épic obligatoire")
		}
		created, err := d.CreateEpic(op.ProjectID, op.NewEpicTitle, "", op.Fields)
		if err != nil {
			return "", err
		}
		targetEpicKey = created.Key
		*steps = append(*steps, fmt.Sprintf("Épic créé : %s, %s", created.Key, created.Title))
	}

	moved := 0
	var failures []string
	for _, id := range op.TaskIDs {
		if _, err := d.applyTaskEpic(id, targetEpicKey, steps); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", id, err))
			*steps = append(*steps, fmt.Sprintf("❌ %s : %v", id, err))
			continue
		}
		moved++
	}

	output := fmt.Sprintf("%d ticket(s) déplacé(s) vers %s", moved, targetEpicKey)
	if len(failures) > 0 {
		output += fmt.Sprintf(", %d échec(s) : %s", len(failures), strings.Join(failures, " | "))
		if moved == 0 {
			return output, fmt.Errorf("aucun ticket déplacé : %s", strings.Join(failures, " | "))
		}
	}
	return output, nil
}

func (d *DB) runSetSprintOp(op TrackerOp, steps *[]string) (string, error) {
	ids := op.TaskIDs
	if len(ids) == 0 && strings.TrimSpace(op.TaskID) != "" {
		ids = []string{op.TaskID}
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("aucun ticket sélectionné")
	}

	var client *runner.JiraRESTClient
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		task, err := d.GetTaskByID(id)
		if err != nil || task == nil {
			*steps = append(*steps, fmt.Sprintf("❌ %s : ticket introuvable", id))
			continue
		}
		if task.Source != "jira" {
			*steps = append(*steps, fmt.Sprintf("ℹ️ %s : ticket non Jira, sprint gardé en local", task.Key))
			continue
		}
		if client == nil {
			client, err = d.jiraRESTClientForTask(task)
			if err != nil {
				return "", err
			}
		}
		keys = append(keys, task.Key)
	}
	if len(keys) == 0 || client == nil {
		return "", fmt.Errorf("aucun ticket Jira à déplacer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := client.MoveIssuesToSprint(ctx, op.SprintID, keys); err != nil {
		return "", err
	}

	target := strings.TrimSpace(op.SprintName)
	if strings.TrimSpace(op.SprintID) == "" {
		target = "backlog"
	} else if target == "" {
		target = op.SprintID
	}
	*steps = append(*steps, fmt.Sprintf("✅ %s ➔ %s", strings.Join(keys, ", "), target))
	return fmt.Sprintf("%d ticket(s) déplacé(s) vers %s", len(keys), target), nil
}

func (d *DB) runSetTeamOp(op TrackerOp, steps *[]string) (string, error) {
	ids := op.TaskIDs
	if len(ids) == 0 && strings.TrimSpace(op.TaskID) != "" {
		ids = []string{op.TaskID}
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("aucun ticket sélectionné")
	}

	label := op.TeamName
	if label == "" {
		label = op.TeamID
	}
	if strings.TrimSpace(op.TeamID) == "" {
		label = "aucune équipe"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Le champ Team s'écrit ticket par ticket : contrairement au sprint, aucune
	// API ne prend un lot. Une seule activité les porte quand même, sinon trier
	// cinquante tickets en produirait cinquante.
	done := 0
	var failures []string
	for _, id := range ids {
		task, err := d.GetTaskByID(id)
		if err != nil || task == nil {
			failures = append(failures, fmt.Sprintf("%s: ticket introuvable", id))
			continue
		}
		if task.Source != "jira" {
			*steps = append(*steps, fmt.Sprintf("ℹ️ %s : ticket non Jira, équipe gardée en local", task.Key))
			continue
		}
		client, err := d.jiraRESTClientForTask(task)
		if err != nil {
			return "", err
		}
		if err := client.SetIssueTeam(ctx, task.Key, op.TeamID); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", task.Key, err))
			*steps = append(*steps, fmt.Sprintf("❌ %s : %v", task.Key, err))
			continue
		}
		*steps = append(*steps, fmt.Sprintf("✅ %s ➔ %s", task.Key, label))
		done++
	}

	output := fmt.Sprintf("%d ticket(s) ➔ %s", done, label)
	if len(failures) > 0 {
		output += fmt.Sprintf(", %d échec(s) : %s", len(failures), strings.Join(failures, " | "))
		if done == 0 {
			return output, fmt.Errorf("aucun ticket modifié : %s", strings.Join(failures, " | "))
		}
	}
	return output, nil
}

func (d *DB) runTransitionOp(op TrackerOp, steps *[]string) (string, error) {
	task, err := d.GetTaskByID(op.TaskID)
	if err != nil || task == nil {
		return "", fmt.Errorf("tâche introuvable")
	}

	settings, _ := d.GetSettings()
	trackerURL := ""
	if proj, _ := d.GetProjectByID(task.ProjectID); proj != nil {
		trackerURL = proj.TrackerUrl
	}
	repoPath := d.ResolveTaskRepoPath(task)

	if err := d.runner.TransitionJiraIssueToStatus(settings, trackerURL, task.Key, op.TargetStatus, repoPath); err != nil {
		return "", err
	}
	*steps = append(*steps, fmt.Sprintf("✅ %s transitionné vers « %s »", task.Key, op.TargetStatus))
	return fmt.Sprintf("%s transitionné vers « %s »", task.Key, op.TargetStatus), nil
}

func (d *DB) runEpicHorizonOp(op TrackerOp, steps *[]string) (string, error) {
	note, err := d.PushEpicHorizonLabel(op.ProjectID, op.EpicKey, op.Horizon)
	if err != nil {
		return "", err
	}
	*steps = append(*steps, "✅ "+note)
	return note, nil
}

func (d *DB) runPushHorizonsOp(op TrackerOp, steps *[]string) (string, error) {
	pushed, failures, err := d.PushPendingHorizons(op.ProjectID)
	if err != nil {
		return "", err
	}
	*steps = append(*steps, fmt.Sprintf("%d épic(s) mis à jour", pushed))
	output := fmt.Sprintf("%d épic(s) mis à jour", pushed)
	if len(failures) > 0 {
		for _, f := range failures {
			*steps = append(*steps, "❌ "+f)
		}
		output += fmt.Sprintf(", %d échec(s) : %s", len(failures), strings.Join(failures, " | "))
	}
	return output, nil
}

// finishTrackerOp closes the activity with what the write actually did.
func (d *DB) finishTrackerOp(activityID string, steps []string, output string, opErr error) {
	status := string(models.ActivityStatusCompleted)
	summary := output
	errText := ""
	if opErr != nil {
		status = string(models.ActivityStatusFailed)
		errText = opErr.Error()
		summary = "Échec de l'écriture sur le tracker"
		output = fmt.Sprintf("Erreur : %v", opErr)
		steps = append(steps, fmt.Sprintf("❌ Échec : %v", opErr))
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Les étapes déjà écrites à la mise en file sont conservées : elles disent
	// ce qui était demandé, ce que la trace d'exécution complète.
	existing := []string{}
	var raw string
	if err := d.conn.QueryRow("SELECT steps FROM task_activities WHERE id = ?", activityID).Scan(&raw); err == nil && strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &existing)
	}
	existing = append(existing, steps...)
	stepsJSON, _ := json.Marshal(existing)

	_, _ = d.conn.Exec(`
		UPDATE task_activities
		SET status = ?, summary = ?, output = ?, steps = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, status, summary, output, string(stepsJSON), errText, time.Now(), activityID)
}
