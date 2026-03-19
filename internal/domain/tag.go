package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Tag represents a label that can be applied to tasks and projects.
type Tag struct {
	ID       string
	Title    string
	ParentID *string
	Shortcut *string
	Index    int
	Timestamps
	SoftDelete
}

// NewTag creates a new Tag with the given title.
// It returns an error if the title is empty.
func NewTag(title string) (*Tag, error) {
	if title == "" {
		return nil, errors.New("tag title must not be empty")
	}

	now := time.Now().UTC()

	return &Tag{
		ID:    uuid.New().String(),
		Title: title,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
