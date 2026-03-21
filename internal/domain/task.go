package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Task represents a single actionable item in the system.
type Task struct {
	ID             string
	Title          string
	Notes          string
	Status         Status
	Schedule       Schedule
	StartDate      *time.Time
	Deadline       *time.Time
	CompletedAt    *time.Time
	Index          int
	TodayIndex     *int
	ProjectID      *string
	SectionID      *string
	AreaID         *string
	LocationID     *string
	RecurrenceRule *RecurrenceRule
	TimeSlot       *string // nil, "morning", "evening"
	ChecklistTotal int
	ChecklistDone  int
	Tags           []string
	Timestamps
	SoftDelete
}

// NewTask creates a new Task with the given title.
// It returns an error if the title is empty.
func NewTask(title string) (*Task, error) {
	if title == "" {
		return nil, errors.New("task title must not be empty")
	}

	now := time.Now().UTC()

	return &Task{
		ID:       uuid.New().String(),
		Title:    title,
		Status:   StatusPending,
		Schedule: ScheduleInbox,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

// Validate checks that the Task's fields are consistent.
// It returns an error if validation fails.
func (t *Task) Validate() error {
	if t.Title == "" {
		return errors.New("task title must not be empty")
	}

	if t.SectionID != nil && t.ProjectID == nil {
		return errors.New("task cannot have a section without a project")
	}

	return nil
}
