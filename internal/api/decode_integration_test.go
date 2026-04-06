package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

func setupTaskAndAuthTestServer(t *testing.T) *http.ServeMux {
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
	authSvc := service.NewAuthService(db, "test-secret")

	taskHandler := api.NewTaskHandler(taskSvc, projectSvc, sectionSvc, areaSvc)
	authHandler := api.NewAuthHandler(authSvc)

	mux := http.NewServeMux()
	taskHandler.RegisterRoutes(mux)
	authHandler.RegisterRoutes(mux)
	return mux
}

func TestTaskHandler_Create_UnknownField(t *testing.T) {
	mux := setupTaskAndAuthTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"Test","extra":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != `request body contains unknown field "extra"` {
		t.Fatalf("unexpected error message: %q", body["error"])
	}
}

func TestAuthHandler_Login_TrailingJSON(t *testing.T) {
	mux := setupTaskAndAuthTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"a@example.com","password":"secret"}{"email":"b@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "request body must contain a single JSON object" {
		t.Fatalf("unexpected error message: %q", body["error"])
	}
}
