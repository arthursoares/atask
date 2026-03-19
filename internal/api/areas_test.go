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

// setupTestServer creates an in-memory DB with migrations, wires up the AreaHandler,
// and returns the mux and DB for use in tests.
func setupTestServer(t *testing.T) (*http.ServeMux, *store.DB) {
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

	areaSvc := service.NewAreaService(db, es, bus)
	areaHandler := api.NewAreaHandler(areaSvc)

	mux := http.NewServeMux()
	areaHandler.RegisterRoutes(mux)
	return mux, db
}

func TestAreaHandler_Create(t *testing.T) {
	mux, _ := setupTestServer(t)

	body := bytes.NewBufferString(`{"title": "Work"}`)
	req := httptest.NewRequest(http.MethodPost, "/areas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Event string `json:"event"`
		Data  struct {
			ID    string `json:"ID"`
			Title string `json:"Title"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Event != "area.created" {
		t.Errorf("expected event %q, got %q", "area.created", resp.Event)
	}
	if resp.Data.ID == "" {
		t.Error("expected non-empty ID in response")
	}
	if resp.Data.Title != "Work" {
		t.Errorf("expected title %q, got %q", "Work", resp.Data.Title)
	}
}

func TestAreaHandler_Create_EmptyTitle(t *testing.T) {
	mux, _ := setupTestServer(t)

	body := bytes.NewBufferString(`{"title": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/areas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", w.Code)
	}
}

func TestAreaHandler_List(t *testing.T) {
	mux, _ := setupTestServer(t)

	// Create two areas first
	for _, title := range []string{"Work", "Personal"} {
		body := bytes.NewBufferString(`{"title": "` + title + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/areas", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %q: expected 201, got %d", title, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/areas", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var areas []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&areas); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(areas) != 2 {
		t.Errorf("expected 2 areas, got %d", len(areas))
	}
}

func TestAreaHandler_Get(t *testing.T) {
	mux, _ := setupTestServer(t)

	// Create an area
	body := bytes.NewBufferString(`{"title": "Health"}`)
	req := httptest.NewRequest(http.MethodPost, "/areas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created struct {
		Data struct {
			ID string `json:"ID"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := created.Data.ID

	// Fetch it
	req = httptest.NewRequest(http.MethodGet, "/areas/"+id, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var area struct {
		ID    string `json:"ID"`
		Title string `json:"Title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&area); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if area.ID != id {
		t.Errorf("expected ID %q, got %q", id, area.ID)
	}
	if area.Title != "Health" {
		t.Errorf("expected title %q, got %q", "Health", area.Title)
	}
}

func TestAreaHandler_Get_NotFound(t *testing.T) {
	mux, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/areas/nonexistent-id", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestAreaHandler_Rename(t *testing.T) {
	mux, _ := setupTestServer(t)

	// Create an area
	body := bytes.NewBufferString(`{"title": "Old Name"}`)
	req := httptest.NewRequest(http.MethodPost, "/areas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created struct {
		Data struct {
			ID string `json:"ID"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := created.Data.ID

	// Rename it
	body = bytes.NewBufferString(`{"title": "New Name"}`)
	req = httptest.NewRequest(http.MethodPut, "/areas/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Event string `json:"event"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode rename response: %v", err)
	}
	if resp.Event != "area.renamed" {
		t.Errorf("expected event %q, got %q", "area.renamed", resp.Event)
	}

	// Verify the rename persisted
	req = httptest.NewRequest(http.MethodGet, "/areas/"+id, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var area struct {
		Title string `json:"Title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&area); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if area.Title != "New Name" {
		t.Errorf("expected title %q, got %q", "New Name", area.Title)
	}
}

func TestAreaHandler_Delete(t *testing.T) {
	mux, _ := setupTestServer(t)

	// Create an area
	body := bytes.NewBufferString(`{"title": "To Delete"}`)
	req := httptest.NewRequest(http.MethodPost, "/areas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created struct {
		Data struct {
			ID string `json:"ID"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := created.Data.ID

	// Delete it
	req = httptest.NewRequest(http.MethodDelete, "/areas/"+id, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Event string `json:"event"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if resp.Event != "area.deleted" {
		t.Errorf("expected event %q, got %q", "area.deleted", resp.Event)
	}

	// Verify list no longer includes it
	req = httptest.NewRequest(http.MethodGet, "/areas", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var areas []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&areas); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(areas) != 0 {
		t.Errorf("expected 0 areas after delete, got %d", len(areas))
	}
}
