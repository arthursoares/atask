package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Task represents a single actionable item in the system.
type Task struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Notes          string          `json:"notes"`
	Status         Status          `json:"status"`
	Schedule       Schedule        `json:"schedule"`
	StartDate      *time.Time      `json:"startDate,omitempty"`
	Deadline       *time.Time      `json:"deadline,omitempty"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
	Index          int             `json:"index"`
	TodayIndex     *int            `json:"todayIndex,omitempty"`
	ProjectID      *string         `json:"projectId,omitempty"`
	SectionID      *string         `json:"sectionId,omitempty"`
	AreaID         *string         `json:"areaId,omitempty"`
	LocationID     *string         `json:"locationId,omitempty"`
	RecurrenceRule *RecurrenceRule `json:"repeatRule,omitempty"`
	TimeSlot       *string         `json:"timeSlot,omitempty"`
	Tags           []string        `json:"tags"`
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
