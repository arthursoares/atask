package domain

import (
	"fmt"
	"time"
)

// Status represents the completion state of a task.
type Status int

const (
	StatusPending   Status = 0
	StatusCompleted Status = 1
	StatusCancelled Status = 2
)

// String returns the string representation of a Status.
func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusCompleted:
		return "completed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// ParseStatus parses a string into a Status value.
func ParseStatus(s string) (Status, error) {
	switch s {
	case "pending":
		return StatusPending, nil
	case "completed":
		return StatusCompleted, nil
	case "cancelled":
		return StatusCancelled, nil
	default:
		return 0, fmt.Errorf("unknown status: %q", s)
	}
}

// Schedule represents when a task is intended to be worked on.
type Schedule int

const (
	ScheduleInbox   Schedule = 0
	ScheduleAnytime Schedule = 1
	ScheduleSomeday Schedule = 2
)

// String returns the string representation of a Schedule.
func (s Schedule) String() string {
	switch s {
	case ScheduleInbox:
		return "inbox"
	case ScheduleAnytime:
		return "anytime"
	case ScheduleSomeday:
		return "someday"
	default:
		return "unknown"
	}
}

// ParseSchedule parses a string into a Schedule value.
func ParseSchedule(s string) (Schedule, error) {
	switch s {
	case "inbox":
		return ScheduleInbox, nil
	case "anytime":
		return ScheduleAnytime, nil
	case "someday":
		return ScheduleSomeday, nil
	default:
		return 0, fmt.Errorf("unknown schedule: %q", s)
	}
}

// ChecklistStatus represents the completion state of a checklist item.
type ChecklistStatus int

const (
	ChecklistPending   ChecklistStatus = 0
	ChecklistCompleted ChecklistStatus = 1
)

// String returns the string representation of a ChecklistStatus.
func (c ChecklistStatus) String() string {
	switch c {
	case ChecklistPending:
		return "pending"
	case ChecklistCompleted:
		return "completed"
	default:
		return "unknown"
	}
}

// ActorType identifies who performed an action.
type ActorType string

const (
	ActorHuman ActorType = "human"
	ActorAgent ActorType = "agent"
)

// ActivityType identifies the kind of activity recorded.
type ActivityType string

const (
	ActivityComment        ActivityType = "comment"
	ActivityContextRequest ActivityType = "context_request"
	ActivityReply          ActivityType = "reply"
	ActivityArtifact       ActivityType = "artifact"
	ActivityStatusChange   ActivityType = "status_change"
	ActivityDecomposition  ActivityType = "decomposition"
)

// Timestamps holds standard creation and update timestamps.
type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SoftDelete holds soft-deletion state for an entity.
type SoftDelete struct {
	Deleted   bool
	DeletedAt *time.Time
}
