package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Location represents a geographic location that can be associated with a task.
type Location struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Radius    *int     `json:"radius,omitempty"`
	Address   *string  `json:"address,omitempty"`
	Timestamps
	SoftDelete
}

// NewLocation creates a new Location with the given name.
// It returns an error if the name is empty.
func NewLocation(name string) (*Location, error) {
	if name == "" {
		return nil, errors.New("location name must not be empty")
	}

	now := time.Now().UTC()

	return &Location{
		ID:   uuid.New().String(),
		Name: name,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
