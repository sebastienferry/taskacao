package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tasks/internal/models"
	"tasks/internal/runner"
)

// resolveSpecFrameworkTarget turns a project id, project slug or bare path into
// the working directory, framework and AI agent to use for a Spec-Driven Design
// toolchain operation. Explicit request fields always win over project defaults.
func (d *DB) resolveSpecFrameworkTarget(req models.SpecFrameworkInstallRequest) (repoPath string, framework string, aiAgent string) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	repoPath = strings.TrimSpace(req.RepoPath)
	framework = strings.TrimSpace(req.Framework)
	aiAgent = strings.TrimSpace(req.AIAgent)

	target := strings.TrimSpace(req.ProjectID)
	if target == "" {
		target = repoPath
	}

	if proj, _ := d.getProjectByIDUnsafe(target); proj != nil {
		if repoPath == "" {
			repoPath = proj.RepoPath
		}
		if framework == "" {
			framework = proj.SpecFramework
		}
		if aiAgent == "" {
			aiAgent = proj.AIProvider
		}
	}

	if s, _ := d.getSettingsUnsafe(); s != nil {
		if repoPath == "" {
			repoPath = s.RepoPath
		}
		if framework == "" {
			framework = s.SpecFramework
		}
		if aiAgent == "" {
			aiAgent = s.AIProvider
		}
	}

	if repoPath == "" {
		repoPath = "."
	}
	framework = runner.NormalizeSpecFramework(framework)

	return repoPath, framework, aiAgent
}

// GetSpecFrameworkStatus reports whether the SDD toolchain CLI is reachable and
// whether the project working directory has already been initialized. Passing an
// empty framework reports on both Spec Kit and OpenSpec.
func (d *DB) GetSpecFrameworkStatus(projectIDOrPath string, framework string) []models.SpecFrameworkStatus {
	repoPath, resolved, _ := d.resolveSpecFrameworkTarget(models.SpecFrameworkInstallRequest{
		ProjectID: projectIDOrPath,
		RepoPath:  projectIDOrPath,
		Framework: framework,
	})

	frameworks := []string{"speckit", "openspec"}
	if strings.TrimSpace(framework) != "" {
		frameworks = []string{resolved}
	}

	out := make([]models.SpecFrameworkStatus, 0, len(frameworks))
	for _, f := range frameworks {
		out = append(out, *d.runner.GetSpecFrameworkStatus(f, repoPath))
	}
	return out
}

// InstallSpecFramework bootstraps GitHub Spec Kit or OpenSpec in the project
// working directory and records the outcome as a task activity so the run shows
// up in the Activities view like any other Taskacao command.
func (d *DB) InstallSpecFramework(req models.SpecFrameworkInstallRequest) (*models.SpecFrameworkInstallResult, error) {
	repoPath, framework, aiAgent := d.resolveSpecFrameworkTarget(req)

	// An explicit but unrecognised framework is a caller mistake: report it
	// instead of silently installing Spec Kit.
	if strings.TrimSpace(req.Framework) != "" && !isKnownFrameworkAlias(req.Framework) {
		return nil, fmt.Errorf("framework SDD inconnu: %q (valeurs acceptées: 'speckit', 'openspec')", req.Framework)
	}

	res := d.runner.InstallSpecFramework(models.SpecFrameworkInstallRequest{
		Framework: framework,
		RepoPath:  repoPath,
		AIAgent:   aiAgent,
		Force:     req.Force,
	})

	d.recordSpecFrameworkActivity(req.ProjectID, res)

	return res, nil
}

// isKnownFrameworkAlias tells apart a recognised spelling of a supported
// framework from an outright unknown value, so a typo is reported instead of
// silently falling back to Spec Kit.
func isKnownFrameworkAlias(framework string) bool {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "speckit", "spec-kit", "spec kit", "specify", "openspec", "open-spec", "open spec":
		return true
	}
	return false
}

// recordSpecFrameworkActivity writes a standalone activity row (no parent task)
// describing the installation, so it is visible and auditable in the UI.
func (d *DB) recordSpecFrameworkActivity(projectID string, res *models.SpecFrameworkInstallResult) {
	if res == nil {
		return
	}

	var steps []string
	var outputLines []string

	outputLines = append(outputLines, fmt.Sprintf("### Installation %s\n", res.FrameworkLabel))
	outputLines = append(outputLines, fmt.Sprintf("- Répertoire de travail : `%s`", res.RepoPath))
	if res.Version != "" {
		outputLines = append(outputLines, fmt.Sprintf("- Version : `%s`", res.Version))
	}
	outputLines = append(outputLines, "")

	for _, st := range res.Steps {
		icon := "❌"
		switch {
		case st.Skipped:
			icon = "⏭️"
		case st.Success:
			icon = "✅"
		}
		steps = append(steps, fmt.Sprintf("%s %s", icon, st.Label))
		outputLines = append(outputLines, fmt.Sprintf("%s **%s**", icon, st.Label))
		outputLines = append(outputLines, fmt.Sprintf("```\n$ %s\n```", st.Command))
		if st.Output != "" {
			outputLines = append(outputLines, fmt.Sprintf("```\n%s\n```", st.Output))
		}
		if st.Error != "" {
			outputLines = append(outputLines, fmt.Sprintf("> Erreur : %s", st.Error))
		}
		outputLines = append(outputLines, "")
	}

	if len(res.MarkerPaths) > 0 {
		outputLines = append(outputLines, fmt.Sprintf("Fichiers détectés : %s", strings.Join(res.MarkerPaths, ", ")))
	}

	if res.AlreadyInit && len(res.Steps) == 0 {
		steps = append(steps, "⏭️ Déjà initialisé, aucune commande exécutée")
	}

	status := models.ActivityStatusCompleted
	if !res.Installed {
		status = models.ActivityStatusFailed
	}

	// Installations are project-scoped, not task-scoped: reuse the same
	// synthetic task_id convention as the tracker sync activities.
	targetTaskID := "spec-framework-" + res.Framework
	if strings.TrimSpace(projectID) != "" {
		targetTaskID = "spec-framework-" + strings.TrimSpace(projectID)
	}

	now := time.Now()
	act := models.TaskActivity{
		ID:          uuid.New().String(),
		TaskID:      targetTaskID,
		ProjectID:   projectID,
		SkillID:     "install_spec_framework",
		SkillName:   fmt.Sprintf("Install %s", res.FrameworkLabel),
		Action:      fmt.Sprintf("Installation de %s dans %s", res.FrameworkLabel, res.RepoPath),
		Status:      string(status),
		Summary:     res.Message,
		Output:      strings.Join(outputLines, "\n"),
		Steps:       steps,
		CreatedAt:   now,
		StartedAt:   &now,
		CompletedAt: &now,
		Error:       res.Error,
	}

	d.mu.Lock()
	_ = d.addTaskActivityDirect(act)
	d.mu.Unlock()
}
