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

// LocationService implements business logic for Locations.
type LocationService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewLocationService constructs a LocationService backed by the given DB, EventStore, and Bus.
func NewLocationService(db *store.DB, es *event.EventStore, bus *event.Bus) *LocationService {
	return &LocationService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// locationFromRow converts a sqlc Location row to a domain.Location.
func locationFromRow(row sqlc.Location) *domain.Location {
	loc := &domain.Location{
		ID: row.ID,
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}
	if row.Name.Valid {
		loc.Name = row.Name.String
	}
	if row.Latitude.Valid {
		lat := row.Latitude.Float64
		loc.Latitude = &lat
	}
	if row.Longitude.Valid {
		lon := row.Longitude.Float64
		loc.Longitude = &lon
	}
	if row.Radius.Valid {
		r := int(row.Radius.Int64)
		loc.Radius = &r
	}
	if row.Address.Valid {
		addr := row.Address.String
		loc.Address = &addr
	}
	if row.Deleted != 0 && row.DeletedAt.Valid {
		da := row.DeletedAt.Time
		loc.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &da,
		}
	}
	return loc
}

// publishLocationEvent emits a delta event, domain event, and publishes to the bus.
func (s *LocationService) publishLocationEvent(
	ctx context.Context,
	eventType domain.EventType,
	locationID, actorID string,
	now time.Time,
	payload map[string]any,
	deltaAction domain.DeltaAction,
	field *string,
	newValue json.RawMessage,
) error {
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "location",
		EntityID:   locationID,
		Action:     deltaAction,
		Field:      field,
		NewValue:   newValue,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, eventType, "location", locationID, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       eventType,
		EntityType: "location",
		EntityID:   locationID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Create validates, persists, emits events, then publishes to the bus.
// The variadic opts accepts a single optional client-provided ID (matches the
// pattern used by TaskService.Create / ProjectService.Create / AreaService.Create).
// Offline clients rely on the server preserving their UUID so that subsequent
// references (e.g. PUT /tasks/{id}/location with the client id) resolve correctly.
func (s *LocationService) Create(ctx context.Context, name, actorID string, opts ...string) (*domain.Location, error) {
	if name == "" {
		return nil, errors.New("location name must not be empty")
	}

	now := timeNow()
	id := ""
	if len(opts) > 0 && opts[0] != "" {
		id = opts[0] // Client-provided ID for sync
	} else {
		id = uuid.New().String()
	}

	row, err := s.queries.CreateLocation(ctx, sqlc.CreateLocationParams{
		ID:        id,
		Name:      sql.NullString{String: name, Valid: true},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	loc := locationFromRow(row)

	payload := map[string]any{"name": name}
	if err := s.publishLocationEvent(ctx, domain.LocationCreated, loc.ID, actorID, now, payload, domain.DeltaCreated, nil, nil); err != nil {
		return nil, err
	}

	return loc, nil
}

// Get fetches a location by ID.
func (s *LocationService) Get(ctx context.Context, id string) (*domain.Location, error) {
	row, err := s.queries.GetLocation(ctx, id)
	if err != nil {
		return nil, err
	}
	return locationFromRow(row), nil
}

// List returns all non-deleted locations.
func (s *LocationService) List(ctx context.Context) ([]*domain.Location, error) {
	rows, err := s.queries.ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	locs := make([]*domain.Location, len(rows))
	for i, row := range rows {
		locs[i] = locationFromRow(row)
	}
	return locs, nil
}

// Rename validates and updates the location name, then emits location.renamed.
func (s *LocationService) Rename(ctx context.Context, id, name, actorID string) error {
	if name == "" {
		return errors.New("location name must not be empty")
	}

	now := timeNow()
	_, err := s.queries.UpdateLocationName(ctx, sqlc.UpdateLocationNameParams{
		Name:      sql.NullString{String: name, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"name": name}
	nameJSON, _ := json.Marshal(name)
	return s.publishLocationEvent(ctx, domain.LocationRenamed, id, actorID, now, payload, domain.DeltaModified, strPtr("name"), nameJSON)
}

// Delete clears the location from all tasks, soft-deletes the location, and emits location.deleted.
func (s *LocationService) Delete(ctx context.Context, id, actorID string) error {
	now := timeNow()

	// Clear location from all tasks
	if err := s.queries.ClearLocationFromTasks(ctx, sqlc.ClearLocationFromTasksParams{
		UpdatedAt:  now,
		LocationID: sql.NullString{String: id, Valid: true},
	}); err != nil {
		return err
	}

	// Soft-delete the location itself
	if err := s.queries.SoftDeleteLocation(ctx, sqlc.SoftDeleteLocationParams{
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishLocationEvent(ctx, domain.LocationDeleted, id, actorID, now, payload, domain.DeltaDeleted, nil, nil)
}
