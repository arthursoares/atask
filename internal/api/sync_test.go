package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
)

func setupSyncServer(t *testing.T) (*http.ServeMux, *event.EventStore) {
	t.Helper()

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	es := event.NewEventStore(db)
	syncHandler := api.NewSyncHandler(es)

	mux := http.NewServeMux()
	syncHandler.RegisterRoutes(mux)
	return mux, es
}

func TestSyncHandler_Deltas(t *testing.T) {
	mux, es := setupSyncServer(t)

	// Insert a delta event into the store.
	field := "title"
	err := es.AppendDelta(t.Context(), domain.DeltaEvent{
		EntityType: "task",
		EntityID:   "task-123",
		Action:     domain.DeltaCreated,
		Field:      &field,
		NewValue:   []byte(`"Buy milk"`),
		ActorID:    "user-1",
		Timestamp:  time.Now(),
	})
	if err != nil {
		t.Fatalf("AppendDelta: %v", err)
	}

	// Fetch deltas since cursor 0.
	req := httptest.NewRequest(http.MethodGet, "/sync/deltas?since=0", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var deltas []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&deltas); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(deltas))
	}

	row := deltas[0]

	entityType, _ := row["entity_type"].(map[string]any)
	if entityType["String"] != "task" {
		t.Errorf("expected entity_type %q, got %v", "task", entityType)
	}

	entityID, _ := row["entity_id"].(map[string]any)
	if entityID["String"] != "task-123" {
		t.Errorf("expected entity_id %q, got %v", "task-123", entityID)
	}
}
