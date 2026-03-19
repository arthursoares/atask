package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
)

// TaskService implements business logic for Tasks.
type TaskService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewTaskService constructs a TaskService backed by the given DB, EventStore, and Bus.
func NewTaskService(db *store.DB, es *event.EventStore, bus *event.Bus) *TaskService {
	return &TaskService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// taskFromRow converts a sqlc Task row to a domain.Task.
func taskFromRow(row sqlc.Task) *domain.Task {
	t := &domain.Task{
		ID:       row.ID,
		Notes:    row.Notes,
		Status:   domain.Status(row.Status),
		Schedule: domain.Schedule(row.Schedule),
		Index:    int(row.Index),
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}

	if row.Title.Valid {
		t.Title = row.Title.String
	}

	if row.StartDate.Valid {
		parsed, err := time.Parse("2006-01-02", row.StartDate.String)
		if err == nil {
			t.StartDate = &parsed
		}
	}

	if row.Deadline.Valid {
		parsed, err := time.Parse("2006-01-02", row.Deadline.String)
		if err == nil {
			t.Deadline = &parsed
		}
	}

	if row.CompletedAt.Valid {
		ca := row.CompletedAt.Time
		t.CompletedAt = &ca
	}

	if row.TodayIndex.Valid {
		ti := int(row.TodayIndex.Int64)
		t.TodayIndex = &ti
	}

	if row.ProjectID.Valid {
		pid := row.ProjectID.String
		t.ProjectID = &pid
	}

	if row.SectionID.Valid {
		sid := row.SectionID.String
		t.SectionID = &sid
	}

	if row.AreaID.Valid {
		aid := row.AreaID.String
		t.AreaID = &aid
	}

	if row.LocationID.Valid {
		lid := row.LocationID.String
		t.LocationID = &lid
	}

	if row.RecurrenceRule.Valid {
		var rule domain.RecurrenceRule
		if err := json.Unmarshal([]byte(row.RecurrenceRule.String), &rule); err == nil {
			t.RecurrenceRule = &rule
		}
	}

	if row.Deleted != 0 && row.DeletedAt.Valid {
		da := row.DeletedAt.Time
		t.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &da,
		}
	}

	return t
}

// publishEvent emits a delta event, domain event, and publishes to the bus.
func (s *TaskService) publishEvent(
	ctx context.Context,
	eventType domain.EventType,
	taskID, actorID string,
	now time.Time,
	payload map[string]any,
	deltaAction domain.DeltaAction,
	field *string,
	newValue json.RawMessage,
) error {
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "task",
		EntityID:   taskID,
		Action:     deltaAction,
		Field:      field,
		NewValue:   newValue,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, eventType, "task", taskID, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       eventType,
		EntityType: "task",
		EntityID:   taskID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Create creates a new task in inbox with the given title.
func (s *TaskService) Create(ctx context.Context, title, actorID string) (*domain.Task, error) {
	if title == "" {
		return nil, errors.New("task title must not be empty")
	}

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateTask(ctx, sqlc.CreateTaskParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	task := taskFromRow(row)

	payload := map[string]any{"title": title}
	if err := s.publishEvent(ctx, domain.TaskCreated, task.ID, actorID, now, payload, domain.DeltaCreated, nil, nil); err != nil {
		return nil, err
	}

	return task, nil
}

// Get fetches a task by ID.
func (s *TaskService) Get(ctx context.Context, id string) (*domain.Task, error) {
	row, err := s.queries.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	return taskFromRow(row), nil
}

// List returns all non-deleted tasks.
func (s *TaskService) List(ctx context.Context) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = taskFromRow(row)
	}
	return tasks, nil
}

// Complete sets a task's status to completed and emits task.completed.
func (s *TaskService) Complete(ctx context.Context, id, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateTaskStatus(ctx, sqlc.UpdateTaskStatusParams{
		Status:      int64(domain.StatusCompleted),
		CompletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ID:          id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishEvent(ctx, domain.TaskCompleted, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"completed"`))
}

// Cancel sets a task's status to cancelled and emits task.cancelled.
func (s *TaskService) Cancel(ctx context.Context, id, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateTaskStatus(ctx, sqlc.UpdateTaskStatusParams{
		Status:      int64(domain.StatusCancelled),
		CompletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ID:          id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishEvent(ctx, domain.TaskCancelled, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"cancelled"`))
}

// UpdateTitle validates and updates the task title, then emits task.title_changed.
func (s *TaskService) UpdateTitle(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return errors.New("task title must not be empty")
	}

	now := timeNow()
	_, err := s.queries.UpdateTaskTitle(ctx, sqlc.UpdateTaskTitleParams{
		Title:     sql.NullString{String: title, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"title": title}
	titleJSON, _ := json.Marshal(title)
	return s.publishEvent(ctx, domain.TaskTitleChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("title"), titleJSON)
}

// UpdateNotes updates the task notes and emits task.notes_changed.
func (s *TaskService) UpdateNotes(ctx context.Context, id, notes, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateTaskNotes(ctx, sqlc.UpdateTaskNotesParams{
		Notes:     notes,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"notes": notes}
	notesJSON, _ := json.Marshal(notes)
	return s.publishEvent(ctx, domain.TaskNotesChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("notes"), notesJSON)
}

// UpdateSchedule updates the task schedule and emits the appropriate event.
// Emits: task.scheduled_today (anytime), task.deferred (someday), task.moved_to_inbox (inbox).
func (s *TaskService) UpdateSchedule(ctx context.Context, id string, schedule domain.Schedule, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateTaskSchedule(ctx, sqlc.UpdateTaskScheduleParams{
		Schedule:  int64(schedule),
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	var eventType domain.EventType
	switch schedule {
	case domain.ScheduleAnytime:
		eventType = domain.TaskScheduledToday
	case domain.ScheduleSomeday:
		eventType = domain.TaskDeferred
	default:
		eventType = domain.TaskMovedToInbox
	}

	payload := map[string]any{"schedule": schedule.String()}
	schedJSON, _ := json.Marshal(schedule.String())
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("schedule"), schedJSON)
}

// SetStartDate sets the task start date and emits task.start_date_set.
func (s *TaskService) SetStartDate(ctx context.Context, id string, date *time.Time, actorID string) error {
	now := timeNow()

	var startDate sql.NullString
	if date != nil {
		startDate = sql.NullString{String: date.Format("2006-01-02"), Valid: true}
	}

	_, err := s.queries.UpdateTaskStartDate(ctx, sqlc.UpdateTaskStartDateParams{
		StartDate: startDate,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	if date != nil {
		payload["start_date"] = date.Format("2006-01-02")
	}
	return s.publishEvent(ctx, domain.TaskStartDateSet, id, actorID, now, payload, domain.DeltaModified, strPtr("start_date"), nil)
}

// SetDeadline sets the task deadline and emits task.deadline_set or task.deadline_removed if nil.
func (s *TaskService) SetDeadline(ctx context.Context, id string, date *time.Time, actorID string) error {
	now := timeNow()

	var deadline sql.NullString
	if date != nil {
		deadline = sql.NullString{String: date.Format("2006-01-02"), Valid: true}
	}

	_, err := s.queries.UpdateTaskDeadline(ctx, sqlc.UpdateTaskDeadlineParams{
		Deadline:  deadline,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	var eventType domain.EventType
	payload := map[string]any{}
	if date != nil {
		eventType = domain.TaskDeadlineSet
		payload["deadline"] = date.Format("2006-01-02")
	} else {
		eventType = domain.TaskDeadlineRemoved
	}
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("deadline"), nil)
}

// MoveToProject sets the task project. If nil, clears project and section and emits task.removed_from_project.
// If set, emits task.moved_to_project.
func (s *TaskService) MoveToProject(ctx context.Context, id string, projectID *string, actorID string) error {
	now := timeNow()

	var projectNullStr sql.NullString
	if projectID != nil {
		projectNullStr = sql.NullString{String: *projectID, Valid: true}
	}

	_, err := s.queries.UpdateTaskProject(ctx, sqlc.UpdateTaskProjectParams{
		ProjectID: projectNullStr,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	// If removing from project, also clear the section
	if projectID == nil {
		if _, err := s.queries.UpdateTaskSection(ctx, sqlc.UpdateTaskSectionParams{
			SectionID: sql.NullString{},
			UpdatedAt: now,
			ID:        id,
		}); err != nil {
			return err
		}
	}

	var eventType domain.EventType
	payload := map[string]any{}
	if projectID != nil {
		eventType = domain.TaskMovedToProject
		payload["project_id"] = *projectID
	} else {
		eventType = domain.TaskRemovedFromProject
	}
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("project_id"), nil)
}

// MoveToSection sets the task section and emits task.moved_to_section or task.removed_from_section.
func (s *TaskService) MoveToSection(ctx context.Context, id string, sectionID *string, actorID string) error {
	now := timeNow()

	var sectionNullStr sql.NullString
	if sectionID != nil {
		sectionNullStr = sql.NullString{String: *sectionID, Valid: true}
	}

	_, err := s.queries.UpdateTaskSection(ctx, sqlc.UpdateTaskSectionParams{
		SectionID: sectionNullStr,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	var eventType domain.EventType
	payload := map[string]any{}
	if sectionID != nil {
		eventType = domain.TaskMovedToSection
		payload["section_id"] = *sectionID
	} else {
		eventType = domain.TaskRemovedFromSection
	}
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("section_id"), nil)
}

// MoveToArea sets the task area and emits task.moved_to_area or task.removed_from_area.
func (s *TaskService) MoveToArea(ctx context.Context, id string, areaID *string, actorID string) error {
	now := timeNow()

	var areaNullStr sql.NullString
	if areaID != nil {
		areaNullStr = sql.NullString{String: *areaID, Valid: true}
	}

	_, err := s.queries.UpdateTaskArea(ctx, sqlc.UpdateTaskAreaParams{
		AreaID:    areaNullStr,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	var eventType domain.EventType
	payload := map[string]any{}
	if areaID != nil {
		eventType = domain.TaskMovedToArea
		payload["area_id"] = *areaID
	} else {
		eventType = domain.TaskRemovedFromArea
	}
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("area_id"), nil)
}

// SetLocation sets the task location and emits task.location_set or task.location_removed.
func (s *TaskService) SetLocation(ctx context.Context, id string, locationID *string, actorID string) error {
	now := timeNow()

	var locationNullStr sql.NullString
	if locationID != nil {
		locationNullStr = sql.NullString{String: *locationID, Valid: true}
	}

	_, err := s.queries.UpdateTaskLocation(ctx, sqlc.UpdateTaskLocationParams{
		LocationID: locationNullStr,
		UpdatedAt:  now,
		ID:         id,
	})
	if err != nil {
		return err
	}

	var eventType domain.EventType
	payload := map[string]any{}
	if locationID != nil {
		eventType = domain.TaskLocationSet
		payload["location_id"] = *locationID
	} else {
		eventType = domain.TaskLocationRemoved
	}
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("location_id"), nil)
}

// SetRecurrence validates and sets the recurrence rule, then emits task.recurrence_set or task.recurrence_removed.
func (s *TaskService) SetRecurrence(ctx context.Context, id string, rule *domain.RecurrenceRule, actorID string) error {
	now := timeNow()

	var recurrenceNullStr sql.NullString
	if rule != nil {
		if err := rule.Validate(); err != nil {
			return err
		}
		ruleJSON, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		recurrenceNullStr = sql.NullString{String: string(ruleJSON), Valid: true}
	}

	_, err := s.queries.UpdateTaskRecurrence(ctx, sqlc.UpdateTaskRecurrenceParams{
		RecurrenceRule: recurrenceNullStr,
		UpdatedAt:      now,
		ID:             id,
	})
	if err != nil {
		return err
	}

	var eventType domain.EventType
	payload := map[string]any{}
	if rule != nil {
		eventType = domain.TaskRecurrenceSet
		payload["rule"] = rule
	} else {
		eventType = domain.TaskRecurrenceRemoved
	}
	return s.publishEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("recurrence_rule"), nil)
}

// AddTag adds a tag to the task and emits task.tag_added.
func (s *TaskService) AddTag(ctx context.Context, id, tagID, actorID string) error {
	now := timeNow()

	if err := s.queries.AddTaskTag(ctx, sqlc.AddTaskTagParams{
		TaskID: id,
		TagID:  tagID,
	}); err != nil {
		return err
	}

	payload := map[string]any{"tag_id": tagID}
	return s.publishEvent(ctx, domain.TaskTagAdded, id, actorID, now, payload, domain.DeltaModified, strPtr("tags"), nil)
}

// RemoveTag removes a tag from the task and emits task.tag_removed.
func (s *TaskService) RemoveTag(ctx context.Context, id, tagID, actorID string) error {
	now := timeNow()

	if err := s.queries.RemoveTaskTag(ctx, sqlc.RemoveTaskTagParams{
		TaskID: id,
		TagID:  tagID,
	}); err != nil {
		return err
	}

	payload := map[string]any{"tag_id": tagID}
	return s.publishEvent(ctx, domain.TaskTagRemoved, id, actorID, now, payload, domain.DeltaModified, strPtr("tags"), nil)
}

// AddLink adds a link between two tasks and emits task.link_added.
func (s *TaskService) AddLink(ctx context.Context, id, relatedTaskID, actorID string) error {
	now := timeNow()

	if err := s.queries.AddTaskLink(ctx, sqlc.AddTaskLinkParams{
		TaskID:           id,
		RelatedTaskID:    relatedTaskID,
		RelationshipType: "related",
		CreatedAt:        sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return err
	}

	payload := map[string]any{"related_task_id": relatedTaskID}
	return s.publishEvent(ctx, domain.TaskLinkAdded, id, actorID, now, payload, domain.DeltaModified, strPtr("links"), nil)
}

// RemoveLink removes a link between two tasks and emits task.link_removed.
func (s *TaskService) RemoveLink(ctx context.Context, id, relatedTaskID, actorID string) error {
	now := timeNow()

	if err := s.queries.RemoveTaskLink(ctx, sqlc.RemoveTaskLinkParams{
		TaskID:        id,
		RelatedTaskID: relatedTaskID,
	}); err != nil {
		return err
	}

	payload := map[string]any{"related_task_id": relatedTaskID}
	return s.publishEvent(ctx, domain.TaskLinkRemoved, id, actorID, now, payload, domain.DeltaModified, strPtr("links"), nil)
}

// Reorder sets the task index and emits task.reordered.
func (s *TaskService) Reorder(ctx context.Context, id string, newIndex int, actorID string) error {
	now := timeNow()

	_, err := s.queries.UpdateTaskIndex(ctx, sqlc.UpdateTaskIndexParams{
		Index:     int64(newIndex),
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"index": newIndex}
	idxJSON, _ := json.Marshal(newIndex)
	return s.publishEvent(ctx, domain.TaskReordered, id, actorID, now, payload, domain.DeltaModified, strPtr("index"), idxJSON)
}

// Delete soft-deletes the task and emits task.deleted.
func (s *TaskService) Delete(ctx context.Context, id, actorID string) error {
	now := timeNow()
	deletedAt := sql.NullTime{Time: now, Valid: true}

	if err := s.queries.SoftDeleteTask(ctx, sqlc.SoftDeleteTaskParams{
		DeletedAt: deletedAt,
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishEvent(ctx, domain.TaskDeleted, id, actorID, now, payload, domain.DeltaDeleted, nil, nil)
}
