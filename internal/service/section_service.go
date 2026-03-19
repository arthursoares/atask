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

// SectionService implements business logic for Sections.
type SectionService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

// NewSectionService constructs a SectionService backed by the given DB, EventStore, and Bus.
func NewSectionService(db *store.DB, es *event.EventStore, bus *event.Bus) *SectionService {
	return &SectionService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

// sectionFromRow converts a sqlc Section row to a domain.Section.
func sectionFromRow(row sqlc.Section) *domain.Section {
	s := &domain.Section{
		ID:    row.ID,
		Index: int(row.Index),
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}
	if row.Title.Valid {
		s.Title = row.Title.String
	}
	if row.ProjectID.Valid {
		s.ProjectID = row.ProjectID.String
	}
	if row.Deleted != 0 && row.DeletedAt.Valid {
		da := row.DeletedAt.Time
		s.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &da,
		}
	}
	return s
}

// publishSectionEvent emits a delta event, domain event, and publishes to the bus.
func (s *SectionService) publishSectionEvent(
	ctx context.Context,
	eventType domain.EventType,
	sectionID, actorID string,
	now time.Time,
	payload map[string]any,
	deltaAction domain.DeltaAction,
	field *string,
	newValue json.RawMessage,
) error {
	if err := s.events.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "section",
		EntityID:   sectionID,
		Action:     deltaAction,
		Field:      field,
		NewValue:   newValue,
		ActorID:    actorID,
		Timestamp:  now,
	}); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(payload)
	eventID, err := s.events.AppendDomainEvent(ctx, eventType, "section", sectionID, actorID, payloadJSON)
	if err != nil {
		return err
	}

	s.bus.Publish(&domain.DomainEvent{
		ID:         eventID,
		Type:       eventType,
		EntityType: "section",
		EntityID:   sectionID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  now,
	})

	return nil
}

// Create validates, persists, emits events, then publishes to the bus.
func (s *SectionService) Create(ctx context.Context, title, projectID, actorID string) (*domain.Section, error) {
	if title == "" {
		return nil, errors.New("section title must not be empty")
	}
	if projectID == "" {
		return nil, errors.New("section projectID must not be empty")
	}

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateSection(ctx, sqlc.CreateSectionParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: true},
		ProjectID: sql.NullString{String: projectID, Valid: true},
		Index:     0,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	section := sectionFromRow(row)

	payload := map[string]any{"title": title, "project_id": projectID}
	if err := s.publishSectionEvent(ctx, domain.SectionCreated, section.ID, actorID, now, payload, domain.DeltaCreated, nil, nil); err != nil {
		return nil, err
	}

	return section, nil
}

// Get fetches a section by ID.
func (s *SectionService) Get(ctx context.Context, id string) (*domain.Section, error) {
	row, err := s.queries.GetSection(ctx, id)
	if err != nil {
		return nil, err
	}
	return sectionFromRow(row), nil
}

// ListByProject returns all non-deleted sections for a project.
func (s *SectionService) ListByProject(ctx context.Context, projectID string) ([]*domain.Section, error) {
	rows, err := s.queries.ListSectionsByProject(ctx, sql.NullString{String: projectID, Valid: true})
	if err != nil {
		return nil, err
	}
	sections := make([]*domain.Section, len(rows))
	for i, row := range rows {
		sections[i] = sectionFromRow(row)
	}
	return sections, nil
}

// Rename validates and updates the section title, then emits section.renamed.
func (s *SectionService) Rename(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return errors.New("section title must not be empty")
	}

	now := timeNow()
	_, err := s.queries.UpdateSectionTitle(ctx, sqlc.UpdateSectionTitleParams{
		Title:     sql.NullString{String: title, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{"title": title}
	titleJSON, _ := json.Marshal(title)
	return s.publishSectionEvent(ctx, domain.SectionRenamed, id, actorID, now, payload, domain.DeltaModified, strPtr("title"), titleJSON)
}

// Delete soft-deletes the section. If cascade is true, it tombstones all tasks in the section;
// otherwise it orphans them. Emits section.deleted.
func (s *SectionService) Delete(ctx context.Context, id, actorID string, cascade bool) error {
	now := timeNow()
	deletedAt := sql.NullTime{Time: now, Valid: true}
	sectionIDNull := sql.NullString{String: id, Valid: true}

	if cascade {
		// Tombstone all tasks in the section
		if err := s.queries.CascadeDeleteTasksBySection(ctx, sqlc.CascadeDeleteTasksBySectionParams{
			DeletedAt: deletedAt,
			UpdatedAt: now,
			SectionID: sectionIDNull,
		}); err != nil {
			return err
		}
	} else {
		// Orphan tasks (set section_id = NULL)
		if err := s.queries.OrphanTasksBySection(ctx, sqlc.OrphanTasksBySectionParams{
			UpdatedAt: now,
			SectionID: sectionIDNull,
		}); err != nil {
			return err
		}
	}

	// Soft-delete the section itself
	if err := s.queries.SoftDeleteSection(ctx, sqlc.SoftDeleteSectionParams{
		DeletedAt: deletedAt,
		UpdatedAt: now,
		ID:        id,
	}); err != nil {
		return err
	}

	payload := map[string]any{"cascade": cascade}
	return s.publishSectionEvent(ctx, domain.SectionDeleted, id, actorID, now, payload, domain.DeltaDeleted, nil, nil)
}
