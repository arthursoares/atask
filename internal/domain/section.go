package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Section represents a named grouping of tasks within a project.
type Section struct {
	ID        string
	Title     string
	ProjectID string
	Index     int
	Timestamps
	SoftDelete
}

// NewSection creates a new Section with the given title and project ID.
// It returns an error if either title or projectID is empty.
func NewSection(title, projectID string) (*Section, error) {
	if title == "" {
		return nil, errors.New("section title must not be empty")
	}
	if projectID == "" {
		return nil, errors.New("section projectID must not be empty")
	}

	now := time.Now().UTC()

	return &Section{
		ID:        uuid.New().String(),
		Title:     title,
		ProjectID: projectID,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
