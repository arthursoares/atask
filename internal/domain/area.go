package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Area represents a broad category or responsibility under which projects and tasks can be grouped.
type Area struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Index    int    `json:"index"`
	Archived bool   `json:"archived"`
	Timestamps
	SoftDelete
}

// NewArea creates a new Area with the given title.
// It returns an error if the title is empty.
func NewArea(title string) (*Area, error) {
	if title == "" {
		return nil, errors.New("area title must not be empty")
	}

	now := time.Now().UTC()

	return &Area{
		ID:    uuid.New().String(),
		Title: title,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
