package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ChecklistItem represents a sub-task within a task's checklist.
type ChecklistItem struct {
	ID     string
	Title  string
	Status ChecklistStatus
	TaskID string
	Index  int
	Timestamps
	SoftDelete
}

// NewChecklistItem creates a new ChecklistItem with the given title and task ID.
// It returns an error if either title or taskID is empty.
func NewChecklistItem(title, taskID string) (*ChecklistItem, error) {
	if title == "" {
		return nil, errors.New("checklist item title must not be empty")
	}
	if taskID == "" {
		return nil, errors.New("checklist item taskID must not be empty")
	}

	now := time.Now().UTC()

	return &ChecklistItem{
		ID:     uuid.New().String(),
		Title:  title,
		TaskID: taskID,
		Status: ChecklistPending,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
