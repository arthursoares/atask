package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
)

// AreaService implements business logic for Areas.
type AreaService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewAreaService constructs an AreaService backed by the given DB, EventStore, and Bus.
func NewAreaService(db *store.DB, es *event.EventStore, bus *event.Bus) *AreaService {
	return &AreaService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// areaFromRow converts a sqlc Area row to a domain.Area.
func areaFromRow(row sqlc.Area) *domain.Area {
	a := &domain.Area{
		ID:       row.ID,
		Index:    int(row.Index),
		Archived: row.Archived != 0,
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}
	if row.Title.Valid {
		a.Title = row.Title.String
	}
	if row.Deleted != 0 && row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		a.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &t,
		}
	}
	return a
}

// Create validates, persists, emits delta and domain events, then publishes to the bus.
func (s *AreaService) Create(ctx context.Context, title, actorID string) (*domain.Area, error) {
	if title == "" {
		return nil, errors.New("area title must not be empty")
	}

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateArea(ctx, sqlc.CreateAreaParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: true},
		Index:     0,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	area := areaFromRow(row)

	// Emit delta event
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "area",
		EntityID:   area.ID,
		Action:     domain.DeltaCreated,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return nil, err
	}

	// Emit domain event
	payload := map[string]any{"title": title}
	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, domain.AreaCreated, "area", area.ID, actorID, payloadJSON)
	if err != nil {
		return nil, err
	}

	// Publish to bus
	de := &domain.DomainEvent{
		ID:         eventID,
		Type:       domain.AreaCreated,
		EntityType: "area",
		EntityID:   area.ID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	}
	s.bus.Publish(de)

	return area, nil
}

// Get fetches an area by ID.
func (s *AreaService) Get(ctx context.Context, id string) (*domain.Area, error) {
	row, err := s.queries.GetArea(ctx, id)
	if err != nil {
		return nil, err
	}
	return areaFromRow(row), nil
}

// List returns all non-archived, non-deleted areas.
func (s *AreaService) List(ctx context.Context) ([]*domain.Area, error) {
	rows, err := s.queries.ListAreas(ctx)
	if err != nil {
		return nil, err
	}
	areas := make([]*domain.Area, len(rows))
	for i, row := range rows {
		areas[i] = areaFromRow(row)
	}
	return areas, nil
}

// Rename validates and updates the area title, then emits events.
func (s *AreaService) Rename(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return errors.New("area title must not be empty")
	}

	now := timeNow()
	row, err := s.queries.UpdateAreaTitle(ctx, sqlc.UpdateAreaTitleParams{
		Title:     sql.NullString{String: title, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	// Emit delta event
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "area",
		EntityID:   row.ID,
		Action:     domain.DeltaModified,
		Field:      strPtr("title"),
		NewValue:   json.RawMessage(`"` + title + `"`),
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	// Emit domain event
	payload := map[string]any{"title": title}
	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, domain.AreaRenamed, "area", id, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       domain.AreaRenamed,
		EntityType: "area",
		EntityID:   id,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Archive sets archived=1 on the area and emits events.
func (s *AreaService) Archive(ctx context.Context, id, actorID string) error {
	now := timeNow()
	row, err := s.queries.UpdateAreaArchived(ctx, sqlc.UpdateAreaArchivedParams{
		Archived:  1,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "area",
		EntityID:   row.ID,
		Action:     domain.DeltaModified,
		Field:      strPtr("archived"),
		NewValue:   json.RawMessage(`true`),
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, domain.AreaArchived, "area", id, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       domain.AreaArchived,
		EntityType: "area",
		EntityID:   id,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Unarchive sets archived=0 on the area and emits events.
func (s *AreaService) Unarchive(ctx context.Context, id, actorID string) error {
	now := timeNow()
	row, err := s.queries.UpdateAreaArchived(ctx, sqlc.UpdateAreaArchivedParams{
		Archived:  0,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "area",
		EntityID:   row.ID,
		Action:     domain.DeltaModified,
		Field:      strPtr("archived"),
		NewValue:   json.RawMessage(`false`),
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, domain.AreaUnarchived, "area", id, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       domain.AreaUnarchived,
		EntityType: "area",
		EntityID:   id,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Delete soft-deletes the area. If cascade is true, it also tombstones all projects and tasks
// in the area. Otherwise it orphans them. Emits area.deleted.
func (s *AreaService) Delete(ctx context.Context, id, actorID string, cascade bool) error {
	now := timeNow()
	deletedAt := sql.NullTime{Time: now, Valid: true}
	areaIDNull := sql.NullString{String: id, Valid: true}

	if cascade {
		// Tombstone all tasks in the area
		if err := s.queries.CascadeDeleteTasksByArea(ctx, sqlc.CascadeDeleteTasksByAreaParams{
			DeletedAt: deletedAt,
			UpdatedAt: now,
			AreaID:    areaIDNull,
		}); err != nil {
			return err
		}
		// Tombstone all projects in the area
		if err := s.queries.CascadeDeleteProjectsByArea(ctx, sqlc.CascadeDeleteProjectsByAreaParams{
			DeletedAt: deletedAt,
			UpdatedAt: now,
			AreaID:    areaIDNull,
		}); err != nil {
			return err
		}
	} else {
		// Orphan tasks (set area_id = NULL)
		if err := s.queries.OrphanTasksByArea(ctx, sqlc.OrphanTasksByAreaParams{
			UpdatedAt: now,
			AreaID:    areaIDNull,
		}); err != nil {
			return err
		}
		// Orphan projects (set area_id = NULL)
		if err := s.queries.OrphanProjectsByArea(ctx, sqlc.OrphanProjectsByAreaParams{
			UpdatedAt: now,
			AreaID:    areaIDNull,
		}); err != nil {
			return err
		}
	}

	// Soft-delete the area itself
	if err := s.queries.SoftDeleteArea(ctx, sqlc.SoftDeleteAreaParams{
		DeletedAt: deletedAt,
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	// Emit delta event
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "area",
		EntityID:   id,
		Action:     domain.DeltaDeleted,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	// Emit domain event
	payload := map[string]any{"cascade": cascade}
	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, domain.AreaDeleted, "area", id, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       domain.AreaDeleted,
		EntityType: "area",
		EntityID:   id,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// strPtr is a convenience helper to get a pointer to a string literal.
func strPtr(s string) *string {
	return &s
}
