package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Project represents a collection of tasks grouped under a common goal.
type Project struct {
	ID           string
	Title        string
	Notes        string
	Status       Status
	Schedule     Schedule
	StartDate    *time.Time
	Deadline     *time.Time
	CompletedAt  *time.Time
	Index        int
	AreaID       *string
	Tags         []string
	AutoComplete bool
	Color        string
	Timestamps
	SoftDelete
}

// NewProject creates a new Project with the given title.
// It returns an error if the title is empty.
func NewProject(title string) (*Project, error) {
	if title == "" {
		return nil, errors.New("project title must not be empty")
	}

	now := time.Now().UTC()

	return &Project{
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

// Validate checks that the project's fields are consistent.
func (p *Project) Validate() error {
	if p.Title == "" {
		return errors.New("project title must not be empty")
	}
	return nil
}
