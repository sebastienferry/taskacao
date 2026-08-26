package db

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"tasks/internal/models"
	"tasks/internal/runner"
)

// CompleteInteractiveStep closes a workflow step that ran in a TTY session.
//
// A headless skill is moved by the worker, which is the only place that posts
// the stage label and transitions the ticket. An interactive step has no
// worker: the agent answers in the terminal and the repo skill only produces
// text, it never touches the tracker. Without this call the ticket stayed where
// it was, whatever happened in the session. So the user confirms the session is
// over, and Taskacao applies the same move the worker would have applied.
//
// note, when not empty, is published as a comment on the ticket: the summary of
// what the session concluded.
func (d *DB) CompleteInteractiveStep(taskID, skillID, note string) (*models.Task, *models.TaskActivity, error) {
	skillID = strings.TrimSpace(skillID)
	stageLabel := skillStageLabel[skillID]
	if stageLabel == "" {
		return nil, nil, fmt.Errorf("skill %q sans étape de workflow", skillID)
	}

	task, err := d.GetTaskByID(taskID)
	if err != nil || task == nil {
		return nil, nil, fmt.Errorf("tâche %s non trouvée", taskID)
	}

	// Statut interne : la correspondance du projet d'abord, le repli générique
	// ensuite, comme le fait le worker.
	newStatus := task.Status
	var proj *models.Project
	if task.ProjectID != "" {
		proj, _ = d.GetProjectByID(task.ProjectID)
	}
	if proj != nil {
		if mapped, ok := proj.StageMapping[stageLabel]; ok && mapped != "" {
			newStatus = models.Status(mapped)
		}
	}
	if newStatus == task.Status {
		if st, ok := InternalStatusForStage(stageLabel); ok {
			newStatus = st
		}
	}

	trackerStatusTarget := ""
	trackerURL := ""
	if proj != nil {
		trackerURL = proj.TrackerUrl
		trackerStatusTarget = TrackerStatusForStage(proj, stageLabel)
	}

	// Une session interactive peut avoir ouvert la MR : on la rattache au ticket
	// à la confirmation, sinon elle resterait dans le terminal.
	mrURL := ""
	if task.PrURL == nil || strings.TrimSpace(*task.PrURL) == "" {
		branchForMR := ""
		if task.BranchName != nil {
			branchForMR = *task.BranchName
		}
		if url, _ := d.runner.MergeRequestForStep(d.ResolveTaskRepoPath(task), branchForMR, note); url != "" {
			mrURL = url
		}
	}

	newLabels := SetWorkflowLabel(task.Labels, stageLabel)
	nowT := time.Now()
	now := nowT.Format("2006-01-02 15:04:05")

	trackerStatus := task.TrackerStatus
	if trackerStatusTarget != "" {
		trackerStatus = trackerStatusTarget
	}

	d.mu.Lock()
	labelsJSON, _ := json.Marshal(newLabels)
	var execErr error
	if mrURL != "" {
		_, execErr = d.conn.Exec(`
			UPDATE tasks SET status = ?, labels = ?, tracker_status = ?, pr_url = ?, updated_at = ?
			WHERE id = ?
		`, string(newStatus), string(labelsJSON), trackerStatus, mrURL, now, task.ID)
	} else {
		_, execErr = d.conn.Exec(`
			UPDATE tasks SET status = ?, labels = ?, tracker_status = ?, updated_at = ?
			WHERE id = ?
		`, string(newStatus), string(labelsJSON), trackerStatus, now, task.ID)
	}
	d.mu.Unlock()
	if execErr != nil {
		return nil, nil, execErr
	}

	task.Status = newStatus
	task.Labels = newLabels
	if mrURL != "" {
		task.PrURL = &mrURL
	}
	task.TrackerStatus = trackerStatus
	task.UpdatedAt = nowT

	skillName := skillID
	for _, s := range d.GetAvailableSkills() {
		if s.ID == skillID {
			skillName = s.Name
			break
		}
	}
	act := models.TaskActivity{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		SkillID:   skillID,
		SkillName: skillName,
		Action:    fmt.Sprintf("%s confirmée en session interactive", skillName),
		Status:    "completed",
		Summary:   fmt.Sprintf("Session TTY clôturée par l'utilisateur ➔ Étape: %s [Label: #%s]", newStatus, stageLabel),
		Output:    note,
		Steps: append(
			[]string{"Session interactive menée dans le terminal de la tâche", "Étape confirmée par l'utilisateur"},
			runner.MergeRequestStep(mrURL, "session interactive"),
		),
		StartedAt:   &nowT,
		CompletedAt: &nowT,
		CreatedAt:   nowT,
	}
	if err := d.AddTaskActivity(act); err != nil {
		log.Printf("[interactive] activité de %s non enregistrée: %v", task.Key, err)
	}

	d.pushStageToTracker(task, stageLabel, trackerStatusTarget, trackerURL, note)
	return task, &act, nil
}

// pushStageToTracker mirrors a stage change onto the tracker in the background:
// the stage label posted, the previous ones removed, and the transition to the
// status the project maps the stage to. The tracker does not replace labels on
// its own, unlike SetWorkflowLabel locally.
func (d *DB) pushStageToTracker(task *models.Task, stageLabel, statusTarget, trackerURL, note string) {
	isTracked := task.Source == "linear" || task.Source == "github" || task.Source == "jira" ||
		strings.HasPrefix(task.Key, "FRE-") || strings.HasPrefix(task.Key, "#") ||
		strings.HasPrefix(task.Key, "gh-") || strings.HasPrefix(task.Key, "GH-#")
	if !isTracked {
		return
	}

	settings, _ := d.GetSettings()
	if settings == nil {
		return
	}
	repoPath := d.ResolveTaskRepoPath(task)
	if repoPath == "" {
		repoPath = settings.RepoPath
	}
	stale := StaleWorkflowLabels(stageLabel)
	body := ""
	if strings.TrimSpace(note) != "" {
		body = "### 💬 [Taskacao] Rapport de session interactive\n\n" + note
	}

	go func(src, repo, rPath, key string, st models.Status, lbls, staleLbls []string, target, url, comment string) {
		switch {
		case src == "linear" || strings.HasPrefix(key, "FRE-"):
			_ = d.runner.UpdateLinearIssueState(key, st)
			_ = d.runner.UpdateLinearIssue(key, nil, nil, nil, &st, lbls)
		case src == "jira":
			if err := d.runner.UpdateJiraIssue(key, rPath, nil, nil, nil, nil, lbls, staleLbls); err != nil {
				log.Printf("[interactive] labels de %s non mis à jour: %v", key, err)
			}
			if target != "" {
				if err := d.runner.TransitionJiraIssueToStatus(settings, url, key, target, rPath); err != nil {
					log.Printf("[interactive] transition de %s vers %q impossible: %v", key, target, err)
				}
			}
		default:
			_ = d.runner.UpdateGithubIssueState(repo, rPath, key, st)
			_ = d.runner.UpdateGithubIssue(repo, rPath, key, nil, nil, &st, lbls, staleLbls)
		}
		if comment != "" {
			_ = d.runner.AddIssueComment(src, repo, rPath, key, comment)
		}
	}(task.Source, settings.GithubRepo, repoPath, task.Key, task.Status, task.Labels, stale, statusTarget, trackerURL, body)
}
