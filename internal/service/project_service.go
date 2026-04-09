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

// ProjectService implements business logic for Projects.
type ProjectService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewProjectService constructs a ProjectService backed by the given DB, EventStore, and Bus.
func NewProjectService(db *store.DB, es *event.EventStore, bus *event.Bus) *ProjectService {
	return &ProjectService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// projectFromRow converts a sqlc Project row to a domain.Project.
func projectFromRow(row sqlc.Project) *domain.Project {
	p := &domain.Project{
		ID:           row.ID,
		Notes:        row.Notes,
		Status:       domain.Status(row.Status),
		Schedule:     domain.Schedule(row.Schedule),
		Index:        int(row.Index),
		AutoComplete: row.AutoComplete != 0,
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}

	if row.Title.Valid {
		p.Title = row.Title.String
	}

	if row.StartDate.Valid {
		parsed, err := time.Parse("2006-01-02", row.StartDate.String)
		if err == nil {
			p.StartDate = &parsed
		}
	}

	if row.Deadline.Valid {
		parsed, err := time.Parse("2006-01-02", row.Deadline.String)
		if err == nil {
			p.Deadline = &parsed
		}
	}

	if row.CompletedAt.Valid {
		ca := row.CompletedAt.Time
		p.CompletedAt = &ca
	}

	if row.AreaID.Valid {
		aid := row.AreaID.String
		p.AreaID = &aid
	}

	p.Color = row.Color

	if row.Deleted != 0 && row.DeletedAt.Valid {
		da := row.DeletedAt.Time
		p.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &da,
		}
	}

	return p
}

// publishProjectEvent emits a delta event, domain event, and publishes to the bus.
func (s *ProjectService) publishProjectEvent(
	ctx context.Context,
	eventType domain.EventType,
	projectID, actorID string,
	now time.Time,
	payload map[string]any,
	deltaAction domain.DeltaAction,
	field *string,
	newValue json.RawMessage,
) error {
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "project",
		EntityID:   projectID,
		Action:     deltaAction,
		Field:      field,
		NewValue:   newValue,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, eventType, "project", projectID, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       eventType,
		EntityType: "project",
		EntityID:   projectID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Create creates a new project with the given title.
// An optional client-provided ID can be passed as opts[0]; if empty or omitted, a new UUID is generated.
func (s *ProjectService) Create(ctx context.Context, title, actorID string, opts ...string) (*domain.Project, error) {
	if title == "" {
		return nil, errors.New("project title must not be empty")
	}

	now := timeNow()
	id := uuid.New().String()
	if len(opts) > 0 && opts[0] != "" {
		id = opts[0]
	}

	row, err := s.queries.CreateProject(ctx, sqlc.CreateProjectParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		Color:     "",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	project := projectFromRow(row)

	payload := map[string]any{"title": title}
	if err := s.publishProjectEvent(ctx, domain.ProjectCreated, project.ID, actorID, now, payload, domain.DeltaCreated, nil, nil); err != nil {
		return nil, err
	}

	return project, nil
}

// hydrateTags queries and populates the Tags field on a project. Always sets
// Tags to a non-nil slice so the JSON wire format emits "tags": [] (never
// null), which the Rust client uses as the signal to reconcile local
// projectTags against the server's authoritative list.
func (s *ProjectService) hydrateTags(ctx context.Context, project *domain.Project) error {
	tags, err := s.queries.ListProjectTags(ctx, project.ID)
	if err != nil {
		return err
	}
	project.Tags = make([]string, len(tags))
	for i, t := range tags {
		project.Tags[i] = t.ID
	}
	return nil
}

// Get fetches a project by ID, including its tags.
func (s *ProjectService) Get(ctx context.Context, id string) (*domain.Project, error) {
	row, err := s.queries.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}
	project := projectFromRow(row)
	if err := s.hydrateTags(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

// List returns all non-deleted projects.
func (s *ProjectService) List(ctx context.Context) ([]*domain.Project, error) {
	rows, err := s.queries.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	projects := make([]*domain.Project, len(rows))
	for i, row := range rows {
		projects[i] = projectFromRow(row)
	}
	return projects, nil
}

// Complete sets the project status to completed, cascades to mark all pending tasks
// as completed, and emits project.completed.
func (s *ProjectService) Complete(ctx context.Context, id, actorID string) error {
	now := timeNow()

	// CASCADE: mark all pending tasks in the project as completed
	if err := s.queries.CompleteTasksByProject(ctx, sqlc.CompleteTasksByProjectParams{
		CompletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ProjectID:   sql.NullString{String: id, Valid: true},
	}); err != nil {
		return err
	}

	_, err := s.queries.UpdateProjectStatus(ctx, sqlc.UpdateProjectStatusParams{
		Status:      int64(domain.StatusCompleted),
		CompletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ID:          id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishProjectEvent(ctx, domain.ProjectCompleted, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"completed"`))
}

// Cancel sets the project status to cancelled, cascades to mark all pending tasks
// as cancelled, and emits project.cancelled.
func (s *ProjectService) Cancel(ctx context.Context, id, actorID string) error {
	now := timeNow()

	// CASCADE: mark all pending tasks in the project as cancelled
	if err := s.queries.CancelTasksByProject(ctx, sqlc.CancelTasksByProjectParams{
		UpdatedAt: now,
		ProjectID: sql.NullString{String: id, Valid: true},
	}); err != nil {
		return err
	}

	_, err := s.queries.UpdateProjectStatus(ctx, sqlc.UpdateProjectStatusParams{
		Status:      int64(domain.StatusCancelled),
		CompletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ID:          id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishProjectEvent(ctx, domain.ProjectCancelled, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"cancelled"`))
}

// UpdateTitle validates and updates the project title, then emits project.title_changed.
func (s *ProjectService) UpdateTitle(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return errors.New("project title must not be empty")
	}

	now := timeNow()
	_, err := s.queries.UpdateProjectTitle(ctx, sqlc.UpdateProjectTitleParams{
		Title:     sql.NullString{String: title, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"title": title}
	titleJSON, _ := json.Marshal(title)
	return s.publishProjectEvent(ctx, domain.ProjectTitleChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("title"), titleJSON)
}

// UpdateNotes updates the project notes and emits project.notes_changed.
func (s *ProjectService) UpdateNotes(ctx context.Context, id, notes, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateProjectNotes(ctx, sqlc.UpdateProjectNotesParams{
		Notes:     notes,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"notes": notes}
	notesJSON, _ := json.Marshal(notes)
	return s.publishProjectEvent(ctx, domain.ProjectNotesChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("notes"), notesJSON)
}

// SetDeadline sets the project deadline and emits project.deadline_set or project.deadline_removed if nil.
func (s *ProjectService) SetDeadline(ctx context.Context, id string, date *time.Time, actorID string) error {
	now := timeNow()

	var deadline sql.NullString
	if date != nil {
		deadline = sql.NullString{String: date.Format("2006-01-02"), Valid: true}
	}

	_, err := s.queries.UpdateProjectDeadline(ctx, sqlc.UpdateProjectDeadlineParams{
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
		eventType = domain.ProjectDeadlineSet
		payload["deadline"] = date.Format("2006-01-02")
	} else {
		eventType = domain.ProjectDeadlineRemoved
	}
	return s.publishProjectEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("deadline"), nil)
}

// MoveToArea sets the project area and emits project.moved_to_area or project.removed_from_area.
func (s *ProjectService) MoveToArea(ctx context.Context, id string, areaID *string, actorID string) error {
	now := timeNow()

	var areaNullStr sql.NullString
	if areaID != nil {
		areaNullStr = sql.NullString{String: *areaID, Valid: true}
	}

	_, err := s.queries.UpdateProjectArea(ctx, sqlc.UpdateProjectAreaParams{
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
		eventType = domain.ProjectMovedToArea
		payload["area_id"] = *areaID
	} else {
		eventType = domain.ProjectRemovedFromArea
	}
	return s.publishProjectEvent(ctx, eventType, id, actorID, now, payload, domain.DeltaModified, strPtr("area_id"), nil)
}

// AddTag adds a tag to the project and emits project.tag_added.
func (s *ProjectService) AddTag(ctx context.Context, id, tagID, actorID string) error {
	now := timeNow()

	if err := s.queries.AddProjectTag(ctx, sqlc.AddProjectTagParams{
		ProjectID: id,
		TagID:     tagID,
	}); err != nil {
		return err
	}

	payload := map[string]any{"tag_id": tagID}
	return s.publishProjectEvent(ctx, domain.ProjectTagAdded, id, actorID, now, payload, domain.DeltaModified, strPtr("tags"), nil)
}

// RemoveTag removes a tag from the project and emits project.tag_removed.
func (s *ProjectService) RemoveTag(ctx context.Context, id, tagID, actorID string) error {
	now := timeNow()

	if err := s.queries.RemoveProjectTag(ctx, sqlc.RemoveProjectTagParams{
		ProjectID: id,
		TagID:     tagID,
	}); err != nil {
		return err
	}

	payload := map[string]any{"tag_id": tagID}
	return s.publishProjectEvent(ctx, domain.ProjectTagRemoved, id, actorID, now, payload, domain.DeltaModified, strPtr("tags"), nil)
}

// UpdateColor updates the project color and emits project.color_changed.
func (s *ProjectService) UpdateColor(ctx context.Context, id, color, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateProjectColor(ctx, sqlc.UpdateProjectColorParams{
		Color:     color,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"color": color}
	colorJSON, _ := json.Marshal(color)
	return s.publishProjectEvent(ctx, domain.ProjectColorChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("color"), colorJSON)
}

// Delete soft-deletes the project and cascades to tombstone all tasks and sections
// in the project, then tombstones the project itself. Emits project.deleted.
func (s *ProjectService) Delete(ctx context.Context, id, actorID string) error {
	now := timeNow()
	deletedAt := sql.NullTime{Time: now, Valid: true}
	projectIDNull := sql.NullString{String: id, Valid: true}

	// CASCADE: tombstone all tasks in the project
	if err := s.queries.SoftDeleteTasksByProject(ctx, sqlc.SoftDeleteTasksByProjectParams{
		DeletedAt: deletedAt,
		UpdatedAt: now,
		ProjectID: projectIDNull,
	}); err != nil {
		return err
	}

	// CASCADE: tombstone all sections in the project
	if err := s.queries.SoftDeleteSectionsByProject(ctx, sqlc.SoftDeleteSectionsByProjectParams{
		DeletedAt: deletedAt,
		UpdatedAt: now,
		ProjectID: projectIDNull,
	}); err != nil {
		return err
	}

	// Soft-delete the project itself
	if err := s.queries.SoftDeleteProject(ctx, sqlc.SoftDeleteProjectParams{
		DeletedAt: deletedAt,
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishProjectEvent(ctx, domain.ProjectDeleted, id, actorID, now, payload, domain.DeltaDeleted, nil, nil)
}
