package db

import (
	"fmt"
	"strings"
	"time"

	"tasks/internal/models"
)

// Tickets épinglés, pour basculer vite entre les chantiers en cours.
//
// Une table à part plutôt qu'une colonne sur tasks : la liste des colonnes de
// tasks est reprise dans plusieurs requêtes, et en oublier une donnerait un
// décalage silencieux des valeurs lues. Ici la fonctionnalité vit dans son coin.
func (d *DB) ensurePinsTable() {
	_, _ = d.conn.Exec(`CREATE TABLE IF NOT EXISTS pinned_tasks (
		task_id   TEXT PRIMARY KEY,
		pinned_at TEXT NOT NULL
	)`)
}

// PinnedTasks returns the pinned tickets, most recently pinned first.
//
// Les tâches sont renvoyées entières, pas seulement leurs identifiants : la
// barre d'épingles doit s'afficher même quand un ticket est filtré ou appartient
// à un autre projet que celui ouvert.
func (d *DB) PinnedTasks() ([]models.Task, error) {
	d.ensurePinsTable()

	d.mu.RLock()
	rows, err := d.conn.Query(`SELECT task_id FROM pinned_tasks ORDER BY pinned_at DESC`)
	if err != nil {
		d.mu.RUnlock()
		return []models.Task{}, nil
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	d.mu.RUnlock()

	out := make([]models.Task, 0, len(ids))
	stale := []string{}
	for _, id := range ids {
		task, err := d.GetTaskByID(id)
		if err != nil || task == nil {
			// Ticket disparu d'une synchro à l'autre : l'épingle n'a plus d'objet.
			stale = append(stale, id)
			continue
		}
		out = append(out, *task)
	}
	for _, id := range stale {
		_ = d.SetTaskPinned(id, false)
	}
	return out, nil
}

// SetTaskPinned pins or unpins a ticket.
func (d *DB) SetTaskPinned(taskID string, pinned bool) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("identifiant de tâche manquant")
	}
	d.ensurePinsTable()

	d.mu.Lock()
	defer d.mu.Unlock()
	if !pinned {
		_, err := d.conn.Exec(`DELETE FROM pinned_tasks WHERE task_id = ?`, taskID)
		return err
	}
	_, err := d.conn.Exec(`
		INSERT INTO pinned_tasks (task_id, pinned_at) VALUES (?, ?)
		ON CONFLICT(task_id) DO UPDATE SET pinned_at = excluded.pinned_at
	`, taskID, time.Now().Format(time.RFC3339))
	return err
}

// ToggleTaskPinned flips the pin and says what the new state is.
func (d *DB) ToggleTaskPinned(taskID string) (bool, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false, fmt.Errorf("identifiant de tâche manquant")
	}
	task, err := d.GetTaskByID(taskID)
	if err != nil || task == nil {
		return false, fmt.Errorf("tâche %s non trouvée", taskID)
	}
	d.ensurePinsTable()

	d.mu.RLock()
	var existing string
	_ = d.conn.QueryRow(`SELECT task_id FROM pinned_tasks WHERE task_id = ?`, task.ID).Scan(&existing)
	d.mu.RUnlock()

	pinned := existing == ""
	if err := d.SetTaskPinned(task.ID, pinned); err != nil {
		return false, err
	}
	return pinned, nil
}
