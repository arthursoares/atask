package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Location represents a geographic location that can be associated with a task.
type Location struct {
	ID        string
	Name      string
	Latitude  *float64
	Longitude *float64
	Radius    *int
	Address   *string
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
