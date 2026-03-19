package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// setupViewTestServer creates an in-memory DB with migrations, wires up the ViewHandler
// and TaskService, and returns the mux, TaskService, and DB for use in tests.
func setupViewTestServer(t *testing.T) (*http.ServeMux, *service.TaskService, *store.DB) {
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
	viewHandler := api.NewViewHandler(db)

	mux := http.NewServeMux()
	viewHandler.RegisterRoutes(mux)

	return mux, taskSvc, db
}

func TestViewHandler_Inbox(t *testing.T) {
	mux, taskSvc, _ := setupViewTestServer(t)
	ctx := httptest.NewRequest(http.MethodGet, "/views/inbox", nil).Context()

	// Create 2 inbox tasks (inbox is default)
	_, err := taskSvc.Create(ctx, "Inbox Task 1", "test")
	if err != nil {
		t.Fatalf("create task 1: %v", err)
	}
	_, err = taskSvc.Create(ctx, "Inbox Task 2", "test")
	if err != nil {
		t.Fatalf("create task 2: %v", err)
	}

	// Create 1 task and move it to anytime (schedule=1)
	task3, err := taskSvc.Create(ctx, "Anytime Task", "test")
	if err != nil {
		t.Fatalf("create task 3: %v", err)
	}
	if err := taskSvc.UpdateSchedule(ctx, task3.ID, 1 /* ScheduleAnytime */, "test"); err != nil {
		t.Fatalf("update schedule: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/views/inbox", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var tasks []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("expected 2 inbox tasks, got %d", len(tasks))
	}
}

func TestViewHandler_Today(t *testing.T) {
	mux, taskSvc, _ := setupViewTestServer(t)
	ctx := httptest.NewRequest(http.MethodGet, "/views/today", nil).Context()

	// Create a task and set schedule to anytime (schedule=1, which is "today" view)
	task, err := taskSvc.Create(ctx, "Today Task", "test")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := taskSvc.UpdateSchedule(ctx, task.ID, 1 /* ScheduleAnytime */, "test"); err != nil {
		t.Fatalf("update schedule to anytime: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/views/today", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var tasks []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 today task, got %d", len(tasks))
	}
}
