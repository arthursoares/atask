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

// ChecklistService implements business logic for ChecklistItems.
type ChecklistService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewChecklistService constructs a ChecklistService backed by the given DB, EventStore, and Bus.
func NewChecklistService(db *store.DB, es *event.EventStore, bus *event.Bus) *ChecklistService {
	return &ChecklistService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// checklistItemFromRow converts a sqlc ChecklistItem row to a domain.ChecklistItem.
func checklistItemFromRow(row sqlc.ChecklistItem) *domain.ChecklistItem {
	item := &domain.ChecklistItem{
		ID:     row.ID,
		Status: domain.ChecklistStatus(row.Status),
		Index:  int(row.Index),
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}
	if row.Title.Valid {
		item.Title = row.Title.String
	}
	if row.TaskID.Valid {
		item.TaskID = row.TaskID.String
	}
	if row.Deleted != 0 && row.DeletedAt.Valid {
		da := row.DeletedAt.Time
		item.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &da,
		}
	}
	return item
}

// publishChecklistEvent emits a delta event, domain event, and publishes to the bus.
func (s *ChecklistService) publishChecklistEvent(
	ctx context.Context,
	eventType domain.EventType,
	itemID, actorID string,
	now time.Time,
	payload map[string]any,
	deltaAction domain.DeltaAction,
	field *string,
	newValue json.RawMessage,
) error {
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "checklist_item",
		EntityID:   itemID,
		Action:     deltaAction,
		Field:      field,
		NewValue:   newValue,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, eventType, "checklist_item", itemID, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       eventType,
		EntityType: "checklist_item",
		EntityID:   itemID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// AddItem creates a new checklist item for a task and emits checklist.item_added.
func (s *ChecklistService) AddItem(ctx context.Context, title, taskID, actorID string) (*domain.ChecklistItem, error) {
	if title == "" {
		return nil, errors.New("checklist item title must not be empty")
	}
	if taskID == "" {
		return nil, errors.New("checklist item taskID must not be empty")
	}

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateChecklistItem(ctx, sqlc.CreateChecklistItemParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: true},
		Status:    int64(domain.ChecklistPending),
		TaskID:    sql.NullString{String: taskID, Valid: true},
		Index:     0,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	item := checklistItemFromRow(row)

	payload := map[string]any{"title": title, "task_id": taskID}
	if err := s.publishChecklistEvent(ctx, domain.ChecklistItemAdded, item.ID, actorID, now, payload, domain.DeltaCreated, nil, nil); err != nil {
		return nil, err
	}

	return item, nil
}

// GetItem fetches a checklist item by ID.
func (s *ChecklistService) GetItem(ctx context.Context, id string) (*domain.ChecklistItem, error) {
	row, err := s.queries.GetChecklistItem(ctx, id)
	if err != nil {
		return nil, err
	}
	return checklistItemFromRow(row), nil
}

// ListByTask returns all non-deleted checklist items for a task.
func (s *ChecklistService) ListByTask(ctx context.Context, taskID string) ([]*domain.ChecklistItem, error) {
	rows, err := s.queries.ListChecklistItemsByTask(ctx, sql.NullString{String: taskID, Valid: true})
	if err != nil {
		return nil, err
	}
	items := make([]*domain.ChecklistItem, len(rows))
	for i, row := range rows {
		items[i] = checklistItemFromRow(row)
	}
	return items, nil
}

// UpdateTitle validates and updates the checklist item title, then emits checklist.item_title_changed.
func (s *ChecklistService) UpdateTitle(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return errors.New("checklist item title must not be empty")
	}

	now := timeNow()
	_, err := s.queries.UpdateChecklistItemTitle(ctx, sqlc.UpdateChecklistItemTitleParams{
		Title:     sql.NullString{String: title, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"title": title}
	titleJSON, _ := json.Marshal(title)
	return s.publishChecklistEvent(ctx, domain.ChecklistItemTitleChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("title"), titleJSON)
}

// CompleteItem marks a checklist item as completed and emits checklist.item_completed.
func (s *ChecklistService) CompleteItem(ctx context.Context, id, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateChecklistItemStatus(ctx, sqlc.UpdateChecklistItemStatusParams{
		Status:    int64(domain.ChecklistCompleted),
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishChecklistEvent(ctx, domain.ChecklistItemCompleted, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"completed"`))
}

// UncompleteItem marks a checklist item as pending and emits checklist.item_uncompleted.
func (s *ChecklistService) UncompleteItem(ctx context.Context, id, actorID string) error {
	now := timeNow()
	_, err := s.queries.UpdateChecklistItemStatus(ctx, sqlc.UpdateChecklistItemStatusParams{
		Status:    int64(domain.ChecklistPending),
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishChecklistEvent(ctx, domain.ChecklistItemUncompleted, id, actorID, now, payload, domain.DeltaModified, strPtr("status"), json.RawMessage(`"pending"`))
}

// RemoveItem soft-deletes a checklist item and emits checklist.item_removed.
func (s *ChecklistService) RemoveItem(ctx context.Context, id, actorID string) error {
	now := timeNow()

	if err := s.queries.SoftDeleteChecklistItem(ctx, sqlc.SoftDeleteChecklistItemParams{
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishChecklistEvent(ctx, domain.ChecklistItemRemoved, id, actorID, now, payload, domain.DeltaDeleted, nil, nil)
}
