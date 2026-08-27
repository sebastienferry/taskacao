package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tasks/internal/models"
	"tasks/internal/runner"
)

// First run setup.
//
// Nothing works before the tracker credentials are in place, and until now the
// only way to find that out was to sync and get nothing. The check below answers
// the three questions someone actually has: are these credentials valid, whose
// are they, and which projects can I see.
//
// The token can be kept out of the database. A database is a file one copies,
// backs up and hands over; an environment file next to it is not more secret in
// theory, but it is the file one knows not to send.

// TrackerCheck is the answer to "are these credentials usable".
type TrackerCheck struct {
	OK       bool                  `json:"ok"`
	Error    string                `json:"error,omitempty"`
	Identity *runner.JiraIdentity  `json:"identity,omitempty"`
	Projects []models.TrackerBoard `json:"projects,omitempty"`
}

// CheckJiraCredentials validates a site, an e-mail and a token without storing
// anything. Called before saving, so a wrong value never reaches the settings.
func (d *DB) CheckJiraCredentials(siteURL, email, token string) *TrackerCheck {
	siteURL = strings.TrimSpace(siteURL)
	email = strings.TrimSpace(email)
	token = strings.TrimSpace(token)

	if siteURL == "" || email == "" {
		return &TrackerCheck{Error: "le site et l'e-mail sont requis"}
	}
	if token == "" {
		// Un jeton déjà enregistré peut servir à revalider le reste.
		if existing, _ := d.GetSettings(); existing != nil {
			token = strings.TrimSpace(existing.JiraAPIToken)
		}
		if token == "" {
			token = runner.JiraTokenFromEnv()
		}
		if token == "" {
			return &TrackerCheck{Error: "aucun jeton d'API fourni ni enregistré"}
		}
	}

	probe := &models.Settings{JiraEmail: email, JiraAPIToken: token, JiraUrl: siteURL}
	client := runner.NewJiraRESTClient(probe, siteURL)
	if client == nil {
		return &TrackerCheck{Error: "site, e-mail ou jeton inexploitable"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	identity, err := client.WhoAmI(ctx)
	if err != nil {
		return &TrackerCheck{Error: err.Error()}
	}

	check := &TrackerCheck{OK: true, Identity: identity}
	// Les projets visibles disent aussi ce que le compte peut lire, ce qui rend
	// une erreur de site immédiatement évidente.
	if projects, err := client.ListVisibleProjects(ctx, 20); err == nil {
		check.Projects = projects
	}
	return check
}

// SaveTrackerCredentials records the credentials once they have been checked.
// When storeTokenInFile is set, the token goes to <data dir>/.env instead of the
// database, and the settings keep none of it.
func (d *DB) SaveTrackerCredentials(siteURL, email, token string, storeTokenInFile bool, dataDir string) (*models.Settings, error) {
	check := d.CheckJiraCredentials(siteURL, email, token)
	if !check.OK {
		return nil, fmt.Errorf("%s", check.Error)
	}

	settings, err := d.GetSettings()
	if err != nil || settings == nil {
		return nil, fmt.Errorf("réglages illisibles")
	}

	settings.JiraUrl = strings.TrimSpace(siteURL)
	settings.JiraEmail = strings.TrimSpace(email)
	settings.IssueTracker = "jira"

	token = strings.TrimSpace(token)
	if token != "" {
		if storeTokenInFile {
			if err := writeTokenFile(dataDir, token); err != nil {
				return nil, err
			}
			// Le jeton vit dans le fichier : la base n'en garde pas de copie.
			settings.JiraAPIToken = "__clear__"
			_ = os.Setenv(runner.JiraTokenEnvVar, token)
		} else {
			settings.JiraAPIToken = token
		}
	}

	return d.UpdateSettings(*settings)
}

// writeTokenFile writes the token to <data dir>/.env, readable by its owner only.
func writeTokenFile(dataDir string, token string) error {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		return fmt.Errorf("dossier de données introuvable : le jeton ne peut pas être écrit hors de la base")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("dossier de données non créé: %w", err)
	}

	path := filepath.Join(dataDir, ".env")
	content := fmt.Sprintf("# Écrit par Taskacao. Ce fichier porte un secret : ne le partagez pas.\n%s=%s\n",
		runner.JiraTokenEnvVar, token)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("jeton non écrit dans %s: %w", path, err)
	}
	return nil
}
