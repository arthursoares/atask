package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

func setupSyncServer(t *testing.T) (http.Handler, *event.EventStore) {
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
	return api.WithTestUser(testUserID)(mux), es
}

func TestSyncHandler_Deltas(t *testing.T) {
	mux, es := setupSyncServer(t)

	// Insert a delta event into the store, owned by the same test user the
	// mux is authenticated as (see setupSyncServer / WithTestUser), so the
	// scoped GET /sync/deltas below actually returns it.
	field := "title"
	err := es.AppendDelta(t.Context(), domain.DeltaEvent{
		EntityType: "task",
		EntityID:   "task-123",
		Action:     domain.DeltaCreated,
		Field:      &field,
		NewValue:   []byte(`"Buy milk"`),
		ActorID:    "user-1",
		UserID:     testUserID,
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

// ─── Cross-user delta sync isolation (Task 13) ──────────────────────────────

// deltaRow mirrors the JSON shape of sqlc.DeltaEvent as returned by
// GET /sync/deltas, decoded with real field types (rather than the
// map[string]any style above) so the containment test below can read
// entity_id / user_id directly.
type deltaRow struct {
	ID       int64  `json:"id"`
	UserID   string `json:"user_id"`
	EntityID struct {
		String string `json:"String"`
		Valid  bool   `json:"Valid"`
	} `json:"entity_id"`
}

// setupSyncCrossUserServer wires TaskHandler and SyncHandler onto the same
// EventStore/DB, unwrapped (no fixed user), so tests can drive it as two
// different authenticated users via doJSONAsUser (defined in
// handler_regression_test.go, same api_test package).
func setupSyncCrossUserServer(t *testing.T) http.Handler {
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
	bus := event.NewBus()

	taskSvc := service.NewTaskService(db, es, bus)
	projectSvc := service.NewProjectService(db, es, bus)
	sectionSvc := service.NewSectionService(db, es, bus)
	areaSvc := service.NewAreaService(db, es, bus)

	mux := http.NewServeMux()
	api.NewTaskHandler(taskSvc, projectSvc, sectionSvc, areaSvc).RegisterRoutes(mux)
	api.NewSyncHandler(es).RegisterRoutes(mux)
	return mux
}

// pullDeltas GETs /sync/deltas?since=<cursor> as userID and decodes the
// response into []deltaRow.
func pullDeltas(t *testing.T, mux http.Handler, userID string, cursor int64) []deltaRow {
	t.Helper()
	w := doJSONAsUser(t, mux, userID, http.MethodGet, fmt.Sprintf("/sync/deltas?since=%d", cursor), "")
	if w.Code != http.StatusOK {
		t.Fatalf("pull deltas as %s: %d: %s", userID, w.Code, w.Body.String())
	}
	var rows []deltaRow
	if err := json.NewDecoder(w.Body).Decode(&rows); err != nil {
		t.Fatalf("decode deltas for %s: %v", userID, err)
	}
	return rows
}

// TestDeltaSyncIsolation is the Task 13 brief's core scenario: two users hit
// the real HTTP task-creation flow (not a hand-inserted delta row) and pull
// from the real /sync/deltas endpoint, confirming Tasks 5/6's user-scoping
// (EventStore.DeltasSince + ListDeltaEventsSince's `user_id = ?` filter)
// holds end-to-end through the handler layer.
func TestDeltaSyncIsolation(t *testing.T) {
	mux := setupSyncCrossUserServer(t)

	// User A creates a task -> exactly one delta (task.created), owned by A.
	w := doJSONAsUser(t, mux, "user-a", http.MethodPost, "/tasks", `{"title":"A's task"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task A: %d: %s", w.Code, w.Body.String())
	}

	// User B pulls deltas since 0 -> must see none of A's activity.
	deltasB := pullDeltas(t, mux, "user-b", 0)
	if len(deltasB) != 0 {
		t.Fatalf("user B should see 0 deltas after A's task creation, got %d: %+v", len(deltasB), deltasB)
	}

	// User A pulls deltas since 0 -> must see exactly the one delta from
	// their own create.
	deltasA := pullDeltas(t, mux, "user-a", 0)
	if len(deltasA) != 1 {
		t.Fatalf("user A should see 1 delta, got %d: %+v", len(deltasA), deltasA)
	}
	if deltasA[0].UserID != "user-a" {
		t.Errorf("delta user_id: expected user-a, got %q", deltasA[0].UserID)
	}
}

// TestDeltaSyncIsolation_NoOpCrossUserActionContainment guards a known sharp
// edge flagged in prior review: several service methods (TaskService.Delete,
// RemoveTag, RemoveLink; ChecklistService.RemoveItem; cascade :exec ops) run
// a user-scoped `WHERE id = ? AND user_id = ?` write and then unconditionally
// call publishEvent — they don't check rows-affected first. So when user B
// references user A's entity ID (e.g. DELETE /tasks/{A's task id}), the
// write matches zero rows but a delta is still recorded, with
// UserID = the CALLING user (B) and EntityID = A's task id.
//
// This test proves that behavior is contained: since DeltasSince filters by
// `user_id = ?`, the phantom delta (user_id=B) never appears in user A's
// pull, even though its entity_id points at A's task. The phantom delta does
// land in B's own stream (asserted below too) — that's noise in B's own
// sync, not a cross-user leak. If either assertion below failed, that would
// mean the containment doesn't hold and this task would need to stop and
// report BLOCKED.
func TestDeltaSyncIsolation_NoOpCrossUserActionContainment(t *testing.T) {
	mux := setupSyncCrossUserServer(t)

	// User A creates a task with a client-supplied ID so User B can
	// reference that exact ID (simulating an attacker/buggy client that
	// already knows or guesses an ID belonging to another user).
	const victimTaskID = "victim-task-id"
	w := doJSONAsUser(t, mux, "user-a", http.MethodPost, "/tasks",
		`{"id":"`+victimTaskID+`","title":"A's task"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task A: %d: %s", w.Code, w.Body.String())
	}

	// Drain A's stream so the create delta doesn't contaminate the
	// containment assertion below; remember the cursor.
	created := pullDeltas(t, mux, "user-a", 0)
	if len(created) != 1 {
		t.Fatalf("setup: expected 1 delta from A's create, got %d", len(created))
	}
	cursorA := created[len(created)-1].ID

	// User B attempts to delete User A's task by its known ID. Current
	// handler behavior (internal/api/tasks.go TaskHandler.Delete) returns
	// 200 regardless of whether SoftDeleteTask actually matched a row —
	// documented here, not fixed, since that's outside Task 13's scope.
	w = doJSONAsUser(t, mux, "user-b", http.MethodDelete, "/tasks/"+victimTaskID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("user B delete of A's task id: %d: %s", w.Code, w.Body.String())
	}

	// Containment: user A's delta stream must NOT surface anything from
	// B's no-op delete, even though it referenced A's entity_id.
	deltasA := pullDeltas(t, mux, "user-a", cursorA)
	if len(deltasA) != 0 {
		t.Fatalf("CONTAINMENT BREACH: user A received a delta triggered by user B's action referencing A's task id: %+v", deltasA)
	}

	// User A's task must still exist — B's delete was a no-op server-side.
	w = doJSONAsUser(t, mux, "user-a", http.MethodGet, "/tasks/"+victimTaskID, "")
	if w.Code != http.StatusOK {
		t.Errorf("user A's task should survive B's no-op delete attempt, got %d: %s", w.Code, w.Body.String())
	}

	// Document (not merely assert away) the phantom delta: it DOES land in
	// B's own stream, attributed to B, referencing A's entity id. This is
	// the known emit-on-no-op behavior — confirms it's confined to the
	// acting user's own stream rather than silently vanishing or leaking.
	deltasB := pullDeltas(t, mux, "user-b", 0)
	if len(deltasB) != 1 {
		t.Fatalf("expected exactly 1 phantom delta in B's own stream (documenting emit-on-no-op), got %d: %+v", len(deltasB), deltasB)
	}
	if deltasB[0].UserID != "user-b" {
		t.Errorf("phantom delta user_id: expected user-b, got %q", deltasB[0].UserID)
	}
	if deltasB[0].EntityID.String != victimTaskID {
		t.Errorf("phantom delta entity_id: expected %q, got %q", victimTaskID, deltasB[0].EntityID.String)
	}
}
