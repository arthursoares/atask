package domain

import (
	"encoding/json"
	"time"
)

// DeltaAction represents the type of change recorded in a delta event.
type DeltaAction int

const (
	DeltaCreated  DeltaAction = 0
	DeltaModified DeltaAction = 1
	DeltaDeleted  DeltaAction = 2
)

// String returns the string representation of a DeltaAction.
func (d DeltaAction) String() string {
	switch d {
	case DeltaCreated:
		return "created"
	case DeltaModified:
		return "modified"
	case DeltaDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// DeltaEvent records a fine-grained change to a single field of an entity.
type DeltaEvent struct {
	ID         int64           `json:"id"`
	EntityType string          `json:"entityType"`
	EntityID   string          `json:"entityId"`
	Action     DeltaAction     `json:"action"`
	Field      *string         `json:"field,omitempty"`
	OldValue   json.RawMessage `json:"oldValue,omitempty"`
	NewValue   json.RawMessage `json:"newValue,omitempty"`
	ActorID    string          `json:"actorId"`
	Timestamp  time.Time       `json:"timestamp"`
}

// EventType identifies the kind of domain event that occurred.
type EventType string

// Task events.
const (
	TaskCreated            EventType = "task.created"
	TaskDeleted            EventType = "task.deleted"
	TaskCompleted          EventType = "task.completed"
	TaskCancelled          EventType = "task.cancelled"
	TaskTitleChanged       EventType = "task.title_changed"
	TaskNotesChanged       EventType = "task.notes_changed"
	TaskScheduledToday     EventType = "task.scheduled_today"
	TaskDeferred           EventType = "task.deferred"
	TaskMovedToInbox       EventType = "task.moved_to_inbox"
	TaskStartDateSet       EventType = "task.start_date_set"
	TaskDeadlineSet        EventType = "task.deadline_set"
	TaskDeadlineRemoved    EventType = "task.deadline_removed"
	TaskMovedToProject     EventType = "task.moved_to_project"
	TaskRemovedFromProject EventType = "task.removed_from_project"
	TaskMovedToSection     EventType = "task.moved_to_section"
	TaskRemovedFromSection EventType = "task.removed_from_section"
	TaskMovedToArea        EventType = "task.moved_to_area"
	TaskRemovedFromArea    EventType = "task.removed_from_area"
	TaskTagAdded           EventType = "task.tag_added"
	TaskTagRemoved         EventType = "task.tag_removed"
	TaskLocationSet        EventType = "task.location_set"
	TaskLocationRemoved    EventType = "task.location_removed"
	TaskLinkAdded          EventType = "task.link_added"
	TaskLinkRemoved        EventType = "task.link_removed"
	TaskRecurrenceSet      EventType = "task.recurrence_set"
	TaskRecurrenceRemoved  EventType = "task.recurrence_removed"
	TaskReordered          EventType = "task.reordered"
	TaskTodayIndexSet      EventType = "task.today_index_set"
	TaskReopened           EventType = "task.reopened"
)

// Project events.
const (
	ProjectCreated         EventType = "project.created"
	ProjectDeleted         EventType = "project.deleted"
	ProjectCompleted       EventType = "project.completed"
	ProjectCancelled       EventType = "project.cancelled"
	ProjectTitleChanged    EventType = "project.title_changed"
	ProjectNotesChanged    EventType = "project.notes_changed"
	ProjectTagAdded        EventType = "project.tag_added"
	ProjectTagRemoved      EventType = "project.tag_removed"
	ProjectMovedToArea     EventType = "project.moved_to_area"
	ProjectRemovedFromArea EventType = "project.removed_from_area"
	ProjectDeadlineSet     EventType = "project.deadline_set"
	ProjectDeadlineRemoved EventType = "project.deadline_removed"
	ProjectColorChanged    EventType = "project.color_changed"
)

// Checklist events.
const (
	ChecklistItemAdded        EventType = "checklist.item_added"
	ChecklistItemRemoved      EventType = "checklist.item_removed"
	ChecklistItemCompleted    EventType = "checklist.item_completed"
	ChecklistItemUncompleted  EventType = "checklist.item_uncompleted"
	ChecklistItemTitleChanged EventType = "checklist.item_title_changed"
)

// Activity events.
const (
	ActivityAdded EventType = "activity.added"
)

// Section events.
const (
	SectionCreated   EventType = "section.created"
	SectionDeleted   EventType = "section.deleted"
	SectionRenamed   EventType = "section.renamed"
	SectionReordered EventType = "section.reordered"
)

// Area events.
const (
	AreaCreated    EventType = "area.created"
	AreaDeleted    EventType = "area.deleted"
	AreaRenamed    EventType = "area.renamed"
	AreaArchived   EventType = "area.archived"
	AreaUnarchived EventType = "area.unarchived"
)

// Tag events.
const (
	TagCreated         EventType = "tag.created"
	TagDeleted         EventType = "tag.deleted"
	TagRenamed         EventType = "tag.renamed"
	TagShortcutChanged EventType = "tag.shortcut_changed"
)

// Location events.
const (
	LocationCreated EventType = "location.created"
	LocationDeleted EventType = "location.deleted"
	LocationRenamed EventType = "location.renamed"
)

// DomainEvent represents a high-level business event that occurred in the system.
type DomainEvent struct {
	ID         int64          `json:"id"`
	Type       EventType      `json:"type"`
	EntityType string         `json:"entityType"`
	EntityID   string         `json:"entityId"`
	ActorID    string         `json:"actorId"`
	Payload    map[string]any `json:"payload,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

// NewDomainEvent constructs a DomainEvent with the current timestamp.
func NewDomainEvent(eventType EventType, entityType, entityID, actorID string, payload map[string]any) DomainEvent {
	return DomainEvent{
		Type:       eventType,
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  time.Now(),
	}
}
