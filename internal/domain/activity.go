package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Activity records an action taken on a task by a human or agent actor.
type Activity struct {
	ID        string
	TaskID    string
	ActorID   string
	ActorType ActorType
	Type      ActivityType
	Content   string
	CreatedAt time.Time
}

// NewActivity creates a new Activity for the given task and actor.
// It returns an error if taskID, actorID, or content is empty.
func NewActivity(taskID, actorID string, actorType ActorType, activityType ActivityType, content string) (*Activity, error) {
	if taskID == "" {
		return nil, errors.New("activity taskID must not be empty")
	}
	if actorID == "" {
		return nil, errors.New("activity actorID must not be empty")
	}
	if actorType != ActorHuman && actorType != ActorAgent {
		return nil, fmt.Errorf("invalid actor type: %q", actorType)
	}
	if !isValidActivityType(activityType) {
		return nil, fmt.Errorf("invalid activity type: %q", activityType)
	}
	if content == "" {
		return nil, errors.New("activity content must not be empty")
	}

	return &Activity{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		ActorID:   actorID,
		ActorType: actorType,
		Type:      activityType,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func isValidActivityType(t ActivityType) bool {
	switch t {
	case ActivityComment, ActivityContextRequest, ActivityReply,
		ActivityArtifact, ActivityStatusChange, ActivityDecomposition:
		return true
	default:
		return false
	}
}
