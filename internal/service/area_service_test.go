package service

import (
	"context"
	"testing"

	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
)

// newTestAreaService creates an in-memory DB, runs migrations, and returns an AreaService.
func newTestAreaService(t *testing.T) *AreaService {
	t.Helper()

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	es := event.NewEventStore(db)
	bus := event.NewBus()
	return NewAreaService(db, es, bus)
}

func TestAreaService_Create(t *testing.T) {
	svc := newTestAreaService(t)
	ctx := context.Background()

	area, err := svc.Create(ctx, "Work", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if area.Title != "Work" {
		t.Errorf("expected title %q, got %q", "Work", area.Title)
	}
	if area.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Verify it can be retrieved
	got, err := svc.Get(ctx, area.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Work" {
		t.Errorf("expected persisted title %q, got %q", "Work", got.Title)
	}
	if got.ID != area.ID {
		t.Errorf("expected ID %q, got %q", area.ID, got.ID)
	}
}

func TestAreaService_Create_EmptyTitle(t *testing.T) {
	svc := newTestAreaService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestAreaService_Archive(t *testing.T) {
	svc := newTestAreaService(t)
	ctx := context.Background()

	area, err := svc.Create(ctx, "Personal", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Archive(ctx, area.ID, "user-1"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	got, err := svc.Get(ctx, area.ID)
	if err != nil {
		t.Fatalf("Get after Archive: %v", err)
	}
	if !got.Archived {
		t.Error("expected Archived=true after Archive()")
	}
}
