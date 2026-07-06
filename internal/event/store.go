package event

import (
	"context"
	"database/sql"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
)

// EventStore wraps sqlc-generated queries to provide event persistence.
type EventStore struct {
	queries *sqlc.Queries
}

// NewEventStore constructs an EventStore backed by the given DB.
func NewEventStore(db *store.DB) *EventStore {
	return &EventStore{
		queries: sqlc.New(db.DB),
	}
}

// AppendDelta inserts a delta event into the store. ev.UserID identifies the
// owner of the entity the delta describes and is required for scoped delta
// pulls (DeltasSince).
func (s *EventStore) AppendDelta(ctx context.Context, ev domain.DeltaEvent) error {
	var field sql.NullString
	if ev.Field != nil {
		field = sql.NullString{String: *ev.Field, Valid: true}
	}

	var oldValue sql.NullString
	if len(ev.OldValue) > 0 {
		oldValue = sql.NullString{String: string(ev.OldValue), Valid: true}
	}

	var newValue sql.NullString
	if len(ev.NewValue) > 0 {
		newValue = sql.NullString{String: string(ev.NewValue), Valid: true}
	}

	return s.queries.InsertDeltaEvent(ctx, sqlc.InsertDeltaEventParams{
		EntityType: sql.NullString{String: ev.EntityType, Valid: true},
		EntityID:   sql.NullString{String: ev.EntityID, Valid: true},
		Action:     sql.NullInt64{Int64: int64(ev.Action), Valid: true},
		Field:      field,
		OldValue:   oldValue,
		NewValue:   newValue,
		ActorID:    sql.NullString{String: ev.ActorID, Valid: true},
		Timestamp:  sql.NullTime{Time: ev.Timestamp, Valid: !ev.Timestamp.IsZero()},
		UserID:     ev.UserID,
	})
}

// DeltasSince returns all delta events owned by userID with ID > cursor, ordered by ID.
func (s *EventStore) DeltasSince(ctx context.Context, userID string, cursor int64) ([]sqlc.DeltaEvent, error) {
	return s.queries.ListDeltaEventsSince(ctx, sqlc.ListDeltaEventsSinceParams{
		ID:     cursor,
		UserID: userID,
	})
}

// AppendDomainEvent inserts a domain event owned by userID and returns its auto-generated ID.
func (s *EventStore) AppendDomainEvent(ctx context.Context, eventType domain.EventType, entityType, entityID, actorID, userID string, payload []byte) (int64, error) {
	return s.queries.InsertDomainEvent(ctx, sqlc.InsertDomainEventParams{
		Type:       sql.NullString{String: string(eventType), Valid: true},
		EntityType: sql.NullString{String: entityType, Valid: true},
		EntityID:   sql.NullString{String: entityID, Valid: true},
		ActorID:    sql.NullString{String: actorID, Valid: true},
		Payload:    string(payload),
		Timestamp:  sql.NullTime{Time: time.Now(), Valid: true},
		UserID:     userID,
	})
}

// DomainEventsSince returns all domain events owned by userID with ID > cursor, ordered by ID.
func (s *EventStore) DomainEventsSince(ctx context.Context, userID string, cursor int64) ([]sqlc.DomainEvent, error) {
	return s.queries.ListDomainEventsSince(ctx, sqlc.ListDomainEventsSinceParams{
		ID:     cursor,
		UserID: userID,
	})
}
