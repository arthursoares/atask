package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/google/uuid"
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

	if row.TimeSlot.Valid {
		ts := row.TimeSlot.String
		t.TimeSlot = &ts
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
func (s *TaskService) Create(ctx context.Context, title, actorID string, opts ...string) (*domain.Task, error) {
	if title == "" {
		return nil, errors.New("task title must not be empty")
	}

	now := timeNow()
	id := ""
	if len(opts) > 0 && opts[0] != "" {
		id = opts[0] // Client-provided ID for sync
	} else {
		id = uuid.New().String()
	}

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

// hydrateTags queries and populates the Tags field on a task.
func (s *TaskService) hydrateTags(ctx context.Context, task *domain.Task) error {
	tags, err := s.queries.ListTaskTags(ctx, task.ID)
	if err != nil {
		return err
	}
	task.Tags = make([]string, len(tags))
	for i, t := range tags {
		task.Tags[i] = t.ID
	}
	return nil
}

// hydrateLinks queries and populates the LinkedTaskIDs field on a task.
func (s *TaskService) hydrateLinks(ctx context.Context, task *domain.Task) error {
	links, err := s.queries.ListTaskLinks(ctx, task.ID)
	if err != nil {
		return err
	}
	task.LinkedTaskIDs = make([]string, len(links))
	for i, l := range links {
		task.LinkedTaskIDs[i] = l.RelatedTaskID
	}
	return nil
}

// Get fetches a task by ID, including its tags and links.
func (s *TaskService) Get(ctx context.Context, id string) (*domain.Task, error) {
	row, err := s.queries.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	task := taskFromRow(row)
	if err := s.hydrateTags(ctx, task); err != nil {
		return nil, err
	}
	if err := s.hydrateLinks(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// List returns all non-deleted tasks.
func (s *TaskService) List(ctx context.Context) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

// ListByProject returns non-deleted tasks belonging to the given project.
func (s *TaskService) ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasksByProject(ctx, sql.NullString{String: projectID, Valid: true})
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

// ListByArea returns non-deleted tasks belonging to the given area.
func (s *TaskService) ListByArea(ctx context.Context, areaID string) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasksByArea(ctx, sql.NullString{String: areaID, Valid: true})
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

// ListBySection returns non-deleted tasks belonging to the given section.
func (s *TaskService) ListBySection(ctx context.Context, sectionID string) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasksBySection(ctx, sql.NullString{String: sectionID, Valid: true})
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

// ListBySchedule returns non-deleted tasks with the given schedule.
func (s *TaskService) ListBySchedule(ctx context.Context, schedule domain.Schedule) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasksBySchedule(ctx, int64(schedule))
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

// ListByLocation returns non-deleted tasks at the given location.
func (s *TaskService) ListByLocation(ctx context.Context, locationID string) ([]*domain.Task, error) {
	rows, err := s.queries.ListTasksByLocation(ctx, sql.NullString{String: locationID, Valid: true})
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

func tasksFromRows(rows []sqlc.Task) []*domain.Task {
	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = taskFromRow(row)
	}
	return tasks
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

// ErrSelfLink is returned by AddLink when a task is asked to link to
// itself. Exposed as a sentinel so the HTTP handler can distinguish
// "user input was invalid" (-> 422) from "infrastructure failed"
// (-> 500). Caller code should use errors.Is(err, ErrSelfLink).
var ErrSelfLink = errors.New("task cannot link to itself")

// AddLink adds a bidirectional link between two tasks and emits
// task.link_added for both tasks so clients viewing either task receive the
// delta. The link is stored as two mirrored rows in task_links so that
// hydrateLinks (which reads only outgoing) works symmetrically.
func (s *TaskService) AddLink(ctx context.Context, id, relatedTaskID, actorID string) error {
	if id == relatedTaskID {
		return ErrSelfLink
	}

	now := timeNow()

	// Insert both directions. task_links uses INSERT OR IGNORE so re-adds are
	// idempotent, which is required because clients may retry pending ops.
	if err := s.queries.AddTaskLink(ctx, sqlc.AddTaskLinkParams{
		TaskID:           id,
		RelatedTaskID:    relatedTaskID,
		RelationshipType: "related",
		CreatedAt:        sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return err
	}
	if err := s.queries.AddTaskLink(ctx, sqlc.AddTaskLinkParams{
		TaskID:           relatedTaskID,
		RelatedTaskID:    id,
		RelationshipType: "related",
		CreatedAt:        sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return err
	}

	payload := map[string]any{"related_task_id": relatedTaskID}
	if err := s.publishEvent(ctx, domain.TaskLinkAdded, id, actorID, now, payload, domain.DeltaModified, strPtr("links"), nil); err != nil {
		return err
	}
	// Mirror delta for the peer so a client viewing the peer also refreshes.
	peerPayload := map[string]any{"related_task_id": id}
	return s.publishEvent(ctx, domain.TaskLinkAdded, relatedTaskID, actorID, now, peerPayload, domain.DeltaModified, strPtr("links"), nil)
}

// RemoveLink removes both directions of a link between two tasks and emits
// task.link_removed for both peers.
func (s *TaskService) RemoveLink(ctx context.Context, id, relatedTaskID, actorID string) error {
	now := timeNow()

	if err := s.queries.RemoveTaskLink(ctx, sqlc.RemoveTaskLinkParams{
		TaskID:        id,
		RelatedTaskID: relatedTaskID,
	}); err != nil {
		return err
	}
	if err := s.queries.RemoveTaskLink(ctx, sqlc.RemoveTaskLinkParams{
		TaskID:        relatedTaskID,
		RelatedTaskID: id,
	}); err != nil {
		return err
	}

	payload := map[string]any{"related_task_id": relatedTaskID}
	if err := s.publishEvent(ctx, domain.TaskLinkRemoved, id, actorID, now, payload, domain.DeltaModified, strPtr("links"), nil); err != nil {
		return err
	}
	peerPayload := map[string]any{"related_task_id": id}
	return s.publishEvent(ctx, domain.TaskLinkRemoved, relatedTaskID, actorID, now, peerPayload, domain.DeltaModified, strPtr("links"), nil)
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

// SetTodayIndex sets or clears the today_index for a task and emits task.today_index_set.
func (s *TaskService) SetTodayIndex(ctx context.Context, id string, todayIndex *int, actorID string) error {
	now := timeNow()

	var todayIndexNull sql.NullInt64
	if todayIndex != nil {
		todayIndexNull = sql.NullInt64{Int64: int64(*todayIndex), Valid: true}
	}

	_, err := s.queries.UpdateTaskTodayIndex(ctx, sqlc.UpdateTaskTodayIndexParams{
		TodayIndex: todayIndexNull,
		UpdatedAt:  now,
		ID:         id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	if todayIndex != nil {
		payload["today_index"] = *todayIndex
	}
	idxJSON, _ := json.Marshal(todayIndex)
	return s.publishEvent(ctx, domain.TaskTodayIndexSet, id, actorID, now, payload, domain.DeltaModified, strPtr("today_index"), idxJSON)
}

// Reopen sets a completed or cancelled task back to pending and emits task.reopened.
func (s *TaskService) Reopen(ctx context.Context, id, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateTaskStatus(ctx, sqlc.UpdateTaskStatusParams{
		Status:      int64(domain.StatusPending),
		CompletedAt: sql.NullTime{},
		UpdatedAt:   now,
		ID:          id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishEvent(ctx, domain.TaskReopened, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"pending"`))
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
