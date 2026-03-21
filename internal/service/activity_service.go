package service

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/google/uuid"
)

// ActivityService implements business logic for Activities.
type ActivityService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewActivityService constructs an ActivityService backed by the given DB, EventStore, and Bus.
func NewActivityService(db *store.DB, es *event.EventStore, bus *event.Bus) *ActivityService {
	return &ActivityService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// activityFromRow converts a sqlc Activity row to a domain.Activity.
func activityFromRow(row sqlc.Activity) *domain.Activity {
	a := &domain.Activity{
		ID: row.ID,
	}
	if row.TaskID.Valid {
		a.TaskID = row.TaskID.String
	}
	if row.ActorID.Valid {
		a.ActorID = row.ActorID.String
	}
	if row.ActorType.Valid {
		a.ActorType = domain.ActorType(row.ActorType.String)
	}
	if row.Type.Valid {
		a.Type = domain.ActivityType(row.Type.String)
	}
	if row.Content.Valid {
		a.Content = row.Content.String
	}
	if row.CreatedAt.Valid {
		a.CreatedAt = row.CreatedAt.Time
	}
	return a
}

// Add creates a new activity record for a task and emits activity.added.
func (s *ActivityService) Add(ctx context.Context, taskID, actorID string, actorType domain.ActorType, activityType domain.ActivityType, content string) (*domain.Activity, error) {
	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateActivity(ctx, sqlc.CreateActivityParams{
		ID:        id,
		TaskID:    sql.NullString{String: taskID, Valid: true},
		ActorID:   sql.NullString{String: actorID, Valid: true},
		ActorType: sql.NullString{String: string(actorType), Valid: true},
		Type:      sql.NullString{String: string(activityType), Valid: true},
		Content:   sql.NullString{String: content, Valid: true},
		CreatedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	activity := activityFromRow(row)

	// Emit delta event
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "activity",
		EntityID:   activity.ID,
		Action:     domain.DeltaCreated,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return nil, err
	}

	// Emit domain event
	payload := map[string]any{
		"task_id":    taskID,
		"actor_type": string(actorType),
		"type":       string(activityType),
		"content":    content,
	}
	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, domain.ActivityAdded, "activity", activity.ID, actorID, payloadJSON)
	if err != nil {
		return nil, err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       domain.ActivityAdded,
		EntityType: "activity",
		EntityID:   activity.ID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return activity, nil
}

// ListByTask returns all activities for a task ordered by created_at desc.
func (s *ActivityService) ListByTask(ctx context.Context, taskID string) ([]*domain.Activity, error) {
	rows, err := s.queries.ListActivitiesByTask(ctx, sql.NullString{String: taskID, Valid: true})
	if err != nil {
		return nil, err
	}
	activities := make([]*domain.Activity, len(rows))
	for i, row := range rows {
		activities[i] = activityFromRow(row)
	}
	return activities, nil
}
