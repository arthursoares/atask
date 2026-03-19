package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/store"
)

func setupTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEventStore_AppendAndQuery(t *testing.T) {
	db := setupTestDB(t)
	es := NewEventStore(db)
	ctx := context.Background()

	field := "title"
	ev := domain.DeltaEvent{
		EntityType: "task",
		EntityID:   "task-abc",
		Action:     domain.DeltaCreated,
		Field:      &field,
		OldValue:   nil,
		NewValue:   json.RawMessage(`"Buy milk"`),
		ActorID:    "user-1",
		Timestamp:  time.Now(),
	}

	if err := es.AppendDelta(ctx, ev); err != nil {
		t.Fatalf("AppendDelta error: %v", err)
	}

	events, err := es.DeltasSince(ctx, 0)
	if err != nil {
		t.Fatalf("DeltasSince error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if !events[0].EntityID.Valid || events[0].EntityID.String != "task-abc" {
		t.Errorf("expected EntityID %q, got %v", "task-abc", events[0].EntityID)
	}
}

func TestEventStore_AppendDomainEvent(t *testing.T) {
	db := setupTestDB(t)
	es := NewEventStore(db)
	ctx := context.Background()

	payload, err := json.Marshal(map[string]any{"title": "Buy milk"})
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	id, err := es.AppendDomainEvent(ctx, domain.TaskCreated, "task", "task-xyz", "user-1", payload)
	if err != nil {
		t.Fatalf("AppendDomainEvent error: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID from AppendDomainEvent")
	}

	events, err := es.DomainEventsSince(ctx, 0)
	if err != nil {
		t.Fatalf("DomainEventsSince error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if !events[0].Type.Valid || events[0].Type.String != string(domain.TaskCreated) {
		t.Errorf("expected Type %q, got %v", domain.TaskCreated, events[0].Type)
	}
}
