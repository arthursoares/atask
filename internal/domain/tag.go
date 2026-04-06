package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Tag represents a label that can be applied to tasks and projects.
type Tag struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	ParentID *string `json:"parentId,omitempty"`
	Shortcut *string `json:"shortcut,omitempty"`
	Index    int     `json:"index"`
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
