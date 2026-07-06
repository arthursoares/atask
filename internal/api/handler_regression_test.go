package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// buildFullTestMux wires every handler exercised by the "full" test server
// (locations, links, etc.) but does NOT pin the mux to any user context.
// Shared by setupFullTestServer (single fixed test user, for the bulk of
// this file's tests) and setupCrossUserTestServer (Task 7), which needs to
// drive the same mux as two different authenticated users.
func buildFullTestMux(t *testing.T) *http.ServeMux {
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
	locationSvc := service.NewLocationService(db, es, bus)

	mux := http.NewServeMux()
	api.NewTaskHandler(taskSvc, projectSvc, sectionSvc, areaSvc).RegisterRoutes(mux)
	api.NewProjectHandler(projectSvc, areaSvc).RegisterRoutes(mux)
	api.NewAreaHandler(areaSvc).RegisterRoutes(mux)
	api.NewSectionHandler(sectionSvc).RegisterRoutes(mux)
	api.NewLocationHandler(locationSvc).RegisterRoutes(mux)
	return mux
}

// setupFullTestServer is like setupPatchTestServer but registers every
// handler we want to exercise — locations, links, etc. Lives in this file
// rather than patch_test.go so the existing PATCH-focused setup stays
// minimal.
func setupFullTestServer(t *testing.T) http.Handler {
	t.Helper()
	return api.WithTestUser(testUserID)(buildFullTestMux(t))
}

// setupCrossUserTestServer is like setupFullTestServer but does not wrap the
// mux with a single fixed test user. Task 7's cross-user isolation tests
// need to authenticate as two different users against the same mux, so the
// user context is injected per-request by doJSONAsUser instead.
func setupCrossUserTestServer(t *testing.T) http.Handler {
	t.Helper()
	return buildFullTestMux(t)
}

func doJSON(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// doJSONAsUser is like doJSON but authenticates the request as userID via
// api.WithTestUser, wrapped fresh per call. mux must be an *unwrapped* mux
// (see setupCrossUserTestServer) — wrapping an already-fixed-user handler
// here would just have the outer fixed user win again.
func doJSONAsUser(t *testing.T, mux http.Handler, userID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.WithTestUser(userID)(mux).ServeHTTP(w, req)
	return w
}

// ─── Location HTTP round-trip (Fix #3) ──────────────────────────────────────

// TestPostLocations_PreservesClientID exercises the HTTP layer of T3 (Fix #3
// from the original review). The user posts a location with a client UUID
// and the response must echo that exact id back.
func TestPostLocations_PreservesClientID(t *testing.T) {
	mux := setupFullTestServer(t)

	const clientID = "client-loc-uuid-001"
	w := doJSON(t, mux, http.MethodPost, "/locations",
		`{"id":"`+clientID+`","name":"Office"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.ID != clientID {
		t.Errorf("expected id %q, got %q", clientID, resp.Data.ID)
	}
	if resp.Data.Name != "Office" {
		t.Errorf("expected name Office, got %q", resp.Data.Name)
	}

	// And a follow-up GET must round-trip the same id.
	w = doJSON(t, mux, http.MethodGet, "/locations/"+clientID, "")
	if w.Code != http.StatusOK {
		t.Errorf("GET /locations/%s: expected 200, got %d", clientID, w.Code)
	}
}

func TestPostLocations_GeneratesIDWhenOmitted(t *testing.T) {
	mux := setupFullTestServer(t)

	w := doJSON(t, mux, http.MethodPost, "/locations", `{"name":"Home"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.ID == "" {
		t.Error("expected server-generated id, got empty")
	}
}

// ─── Project tag hydration via HTTP (Fix #6) ───────────────────────────────

// TestGetProject_HydratesTags asserts the HTTP layer of T6 — that
// GET /projects/{id} returns a non-nil Tags array. This is the wire
// contract the Rust client depends on for projectTags inbound sync.
func TestGetProject_HydratesTags(t *testing.T) {
	mux := setupFullTestServer(t)

	// Create a project + tag, then attach.
	w := doJSON(t, mux, http.MethodPost, "/projects", `{"title":"Proj"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project: %d", w.Code)
	}
	var pResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&pResp)
	projectID := pResp.Data.ID

	// Tags don't have a public POST handler in the test server, but
	// we can attach one via the project handler's tag route. The
	// service-layer test already covers full hydration; here we just
	// verify the wire format includes a "tags" key (not null).
	w = doJSON(t, mux, http.MethodGet, "/projects/"+projectID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET project: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"tags":[`) && !strings.Contains(body, `"tags": [`) {
		t.Errorf("expected GET /projects/%s to include a non-nil tags array; got body: %s", projectID, body)
	}
}

// ─── PATCH decode error paths ────────────────────────────────────────────

// TestPatchTask_RejectsUnknownFields confirms DisallowUnknownFields is
// active. This is the contract the Rust client's narrow-body PATCH builders
// (Fix #1, patch_body.rs) depend on — if Go silently accepted server-only
// fields like `id` or `status`, the Rust narrow-body fix would be
// unnecessary and we'd never have caught the original bug.
func TestPatchTask_RejectsUnknownFields(t *testing.T) {
	mux := setupPatchTestServer(t)
	taskID := createTask(t, mux, "Task")

	// `status` is a real field on the Task struct but NOT on the PATCH
	// body struct in tasks.go's Patch handler.
	w := doJSON(t, mux, http.MethodPatch, "/tasks/"+taskID, `{"status":1}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown field, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "unknown field") {
		t.Errorf("expected error message to mention 'unknown field'; got: %s", w.Body.String())
	}
}

func TestPatchTask_RejectsMalformedJSON(t *testing.T) {
	mux := setupPatchTestServer(t)
	taskID := createTask(t, mux, "Task")

	w := doJSON(t, mux, http.MethodPatch, "/tasks/"+taskID, `{"title":`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

func TestPatchTask_RejectsEmptyBody(t *testing.T) {
	mux := setupPatchTestServer(t)
	taskID := createTask(t, mux, "Task")

	w := doJSON(t, mux, http.MethodPatch, "/tasks/"+taskID, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", w.Code)
	}
}

func TestPatchTask_NotFoundReturns404Strict(t *testing.T) {
	// Strict 404 assertion. The existing TestPatchTask_NotFound in
	// patch_test.go accepts both 404 and 422 — see the HIGH-3 finding
	// in the original code review. This test is the strict version
	// that locks in the correct behavior.
	mux := setupPatchTestServer(t)
	w := doJSON(t, mux, http.MethodPatch, "/tasks/does-not-exist", `{"title":"X"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing task, got %d", w.Code)
	}
}

func TestPatchTask_InvalidProjectReturns422(t *testing.T) {
	mux := setupPatchTestServer(t)
	taskID := createTask(t, mux, "Task")

	w := doJSON(t, mux, http.MethodPatch, "/tasks/"+taskID,
		`{"projectId":"this-project-does-not-exist"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid projectId, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPatchProject_RejectsUnknownFields(t *testing.T) {
	mux := setupPatchTestServer(t)
	projectID := createProject(t, mux, "Project")

	w := doJSON(t, mux, http.MethodPatch, "/projects/"+projectID, `{"index":99}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown field, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Task link bidirectionality via HTTP (Fix #2) ───────────────────────

// TestAddTaskLink_RejectsSelfLink confirms the HTTP layer enforces the
// self-link guard added in T2 (Fix #2). The service-layer test in
// internal/service/task_service_test.go covers the same invariant; this
// is the wire-level regression guard.
func TestAddTaskLink_RejectsSelfLink(t *testing.T) {
	mux := setupPatchTestServer(t)
	taskID := createTask(t, mux, "Lonely")

	w := doJSON(t, mux, http.MethodPost, "/tasks/"+taskID+"/links/"+taskID, "")
	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Errorf("expected 422 (or 400) for self-link, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Cross-user isolation (Task 7) ──────────────────────────────────────

// TestCrossUserIsolation confirms that per-user scoping (Tasks 5-6) actually
// isolates data at the HTTP layer: user A's task list must not include user
// B's tasks, and user B must not be able to GET or PATCH user A's task by ID
// (both should look like the resource doesn't exist — 404 — rather than
// leaking a 200/403 that would confirm the ID's existence).
func TestCrossUserIsolation(t *testing.T) {
	mux := setupCrossUserTestServer(t)

	// Create tasks as two different users.
	w := doJSONAsUser(t, mux, "user-a", http.MethodPost, "/tasks", `{"title":"A's task"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task A: %d: %s", w.Code, w.Body.String())
	}
	var respA struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&respA); err != nil {
		t.Fatalf("decode create task A response: %v", err)
	}

	w = doJSONAsUser(t, mux, "user-b", http.MethodPost, "/tasks", `{"title":"B's task"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task B: %d: %s", w.Code, w.Body.String())
	}

	// User A lists tasks — should only see their own.
	// GET /tasks returns the task array directly (no envelope) — see
	// TaskHandler.List's RespondJSON(w, http.StatusOK, tasks) call.
	w = doJSONAsUser(t, mux, "user-a", http.MethodGet, "/tasks", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list tasks A: %d: %s", w.Code, w.Body.String())
	}
	var listA []struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&listA); err != nil {
		t.Fatalf("decode list A response: %v", err)
	}
	if len(listA) != 1 || listA[0].Title != "A's task" {
		t.Errorf("user A should see 1 task, got %d: %+v", len(listA), listA)
	}

	// User B cannot GET user A's task by ID.
	w = doJSONAsUser(t, mux, "user-b", http.MethodGet, "/tasks/"+respA.Data.ID, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("user B accessing A's task: expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// User B cannot PATCH user A's task.
	w = doJSONAsUser(t, mux, "user-b", http.MethodPatch, "/tasks/"+respA.Data.ID, `{"title":"hacked"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("user B patching A's task: expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// Confirm the title truly wasn't touched by the rejected PATCH.
	w = doJSONAsUser(t, mux, "user-a", http.MethodGet, "/tasks/"+respA.Data.ID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("re-fetch task A: %d: %s", w.Code, w.Body.String())
	}
	var refetched struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&refetched); err != nil {
		t.Fatalf("decode re-fetch response: %v", err)
	}
	if refetched.Title != "A's task" {
		t.Errorf("task A title should be unchanged, got %q", refetched.Title)
	}
}

// TestCrossUserHorizontalEscalation_TaskProject is the horizontal-escalation
// case: User A tries to attach User B's project to their own task. SQL
// scoping alone would pass (the task being patched is A's), so this guards
// the service/handler-level ownership validation (Task 5 Step 10) that
// checks the referenced project also belongs to the requesting user.
func TestCrossUserHorizontalEscalation_TaskProject(t *testing.T) {
	mux := setupCrossUserTestServer(t)

	// User A creates a task.
	w := doJSONAsUser(t, mux, "user-a", http.MethodPost, "/tasks", `{"title":"A's task"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task A: %d: %s", w.Code, w.Body.String())
	}
	var respA struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&respA); err != nil {
		t.Fatalf("decode create task A response: %v", err)
	}

	// User B creates a project.
	w = doJSONAsUser(t, mux, "user-b", http.MethodPost, "/projects", `{"title":"B's project"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project B: %d: %s", w.Code, w.Body.String())
	}
	var respB struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&respB); err != nil {
		t.Fatalf("decode create project B response: %v", err)
	}

	// User A tries to PATCH their own task to point at User B's project.
	body := fmt.Sprintf(`{"projectId":%q}`, respB.Data.ID)
	w = doJSONAsUser(t, mux, "user-a", http.MethodPatch, "/tasks/"+respA.Data.ID, body)
	if w.Code != http.StatusNotFound && w.Code != http.StatusUnprocessableEntity {
		t.Errorf("horizontal escalation via projectId: expected 404 or 422, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the task was NOT modified.
	// GET /tasks/{id} returns the task object directly (no envelope) — see
	// TaskHandler.Get's RespondJSON(w, http.StatusOK, task) call.
	w = doJSONAsUser(t, mux, "user-a", http.MethodGet, "/tasks/"+respA.Data.ID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("re-fetch task A: %d: %s", w.Code, w.Body.String())
	}
	var task struct {
		ProjectID *string `json:"projectId"`
	}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode re-fetch response: %v", err)
	}
	if task.ProjectID != nil {
		t.Errorf("task projectId should remain unset after rejected escalation, got %v", *task.ProjectID)
	}
}
