package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Project represents a collection of tasks grouped under a common goal.
type Project struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Notes        string     `json:"notes"`
	Status       Status     `json:"status"`
	Schedule     Schedule   `json:"schedule"`
	StartDate    *time.Time `json:"startDate,omitempty"`
	Deadline     *time.Time `json:"deadline,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	Index        int        `json:"index"`
	AreaID       *string    `json:"areaId,omitempty"`
	Tags         []string   `json:"tags"`
	AutoComplete bool       `json:"autoComplete"`
	Color        string     `json:"color"`
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
