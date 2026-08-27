package db

import (
	"tasks/internal/models"
)

// TrackerCheck is the answer to "are these credentials usable".
type TrackerCheck struct {
	OK       bool                  `json:"ok"`
	Error    string                `json:"error,omitempty"`
	Projects []models.TrackerBoard `json:"projects,omitempty"`
}
