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

// TagService implements business logic for Tags.
type TagService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewTagService constructs a TagService backed by the given DB, EventStore, and Bus.
func NewTagService(db *store.DB, es *event.EventStore, bus *event.Bus) *TagService {
	return &TagService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// tagFromRow converts a sqlc Tag row to a domain.Tag.
func tagFromRow(row sqlc.Tag) *domain.Tag {
	t := &domain.Tag{
		ID:    row.ID,
		Index: int(row.Index),
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}
	if row.Title.Valid {
		t.Title = row.Title.String
	}
	if row.ParentID.Valid {
		pid := row.ParentID.String
		t.ParentID = &pid
	}
	if row.Shortcut.Valid {
		s := row.Shortcut.String
		t.Shortcut = &s
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

// publishTagEvent emits a delta event, domain event, and publishes to the bus.
func (s *TagService) publishTagEvent(
	ctx context.Context,
	eventType domain.EventType,
	tagID, actorID string,
	now time.Time,
	payload map[string]any,
	deltaAction domain.DeltaAction,
	field *string,
	newValue json.RawMessage,
) error {
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "tag",
		EntityID:   tagID,
		Action:     deltaAction,
		Field:      field,
		NewValue:   newValue,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, eventType, "tag", tagID, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       eventType,
		EntityType: "tag",
		EntityID:   tagID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Create validates, persists, emits events, then publishes to the bus.
func (s *TagService) Create(ctx context.Context, title, actorID string) (*domain.Tag, error) {
	if title == "" {
		return nil, errors.New("tag title must not be empty")
	}

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateTag(ctx, sqlc.CreateTagParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: true},
		Index:     0,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	tag := tagFromRow(row)

	payload := map[string]any{"title": title}
	if err := s.publishTagEvent(ctx, domain.TagCreated, tag.ID, actorID, now, payload, domain.DeltaCreated, nil, nil); err != nil {
		return nil, err
	}

	return tag, nil
}

// Get fetches a tag by ID.
func (s *TagService) Get(ctx context.Context, id string) (*domain.Tag, error) {
	row, err := s.queries.GetTag(ctx, id)
	if err != nil {
		return nil, err
	}
	return tagFromRow(row), nil
}

// List returns all non-deleted tags.
func (s *TagService) List(ctx context.Context) ([]*domain.Tag, error) {
	rows, err := s.queries.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	tags := make([]*domain.Tag, len(rows))
	for i, row := range rows {
		tags[i] = tagFromRow(row)
	}
	return tags, nil
}

// Rename validates and updates the tag title, then emits tag.renamed.
func (s *TagService) Rename(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return errors.New("tag title must not be empty")
	}

	now := timeNow()
	_, err := s.queries.UpdateTagTitle(ctx, sqlc.UpdateTagTitleParams{
		Title:     sql.NullString{String: title, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"title": title}
	titleJSON, _ := json.Marshal(title)
	return s.publishTagEvent(ctx, domain.TagRenamed, id, actorID, now, payload, domain.DeltaModified, strPtr("title"), titleJSON)
}

// UpdateShortcut sets or clears the tag shortcut, then emits tag.shortcut_changed.
func (s *TagService) UpdateShortcut(ctx context.Context, id string, shortcut *string, actorID string) error {
	now := timeNow()

	var shortcutNull sql.NullString
	if shortcut != nil {
		shortcutNull = sql.NullString{String: *shortcut, Valid: true}
	}

	_, err := s.queries.UpdateTagShortcut(ctx, sqlc.UpdateTagShortcutParams{
		Shortcut:  shortcutNull,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{}
	if shortcut != nil {
		payload["shortcut"] = *shortcut
	}
	return s.publishTagEvent(ctx, domain.TagShortcutChanged, id, actorID, now, payload, domain.DeltaModified, strPtr("shortcut"), nil)
}

// Delete removes all tag references, soft-deletes the tag, and emits tag.deleted.
func (s *TagService) Delete(ctx context.Context, id, actorID string) error {
	now := timeNow()

	// Remove all task tag references
	if err := s.queries.RemoveAllTagReferences(ctx, id); err != nil {
		return err
	}

	// Remove all project tag references
	if err := s.queries.RemoveAllProjectTagReferences(ctx, id); err != nil {
		return err
	}

	// Soft-delete the tag itself
	if err := s.queries.SoftDeleteTag(ctx, sqlc.SoftDeleteTagParams{
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	payload := map[string]any{}
	return s.publishTagEvent(ctx, domain.TagDeleted, id, actorID, now, payload, domain.DeltaDeleted, nil, nil)
}
