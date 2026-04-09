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

// setupPatchTestServer creates an in-memory DB with all handlers needed for PATCH tests.
func setupPatchTestServer(t *testing.T) *http.ServeMux {
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
	api.NewProjectHandler(projectSvc, areaSvc).RegisterRoutes(mux)
	api.NewAreaHandler(areaSvc).RegisterRoutes(mux)
	api.NewSectionHandler(sectionSvc).RegisterRoutes(mux)
	return mux
}

// createSection is a helper that POSTs a section under the given project and returns its ID.
func createSection(t *testing.T, mux *http.ServeMux, projectID, title string) string {
	t.Helper()
	body := bytes.NewBufferString(`{"title":"` + title + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sections", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create section %q: expected 201, got %d: %s", title, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create section response: %v", err)
	}
	return resp.Data.ID
}

// createTask is a helper that POSTs a task and returns its ID.
func createTask(t *testing.T, mux *http.ServeMux, title string) string {
	t.Helper()
	body := bytes.NewBufferString(`{"title":"` + title + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create task %q: expected 201, got %d: %s", title, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create task response: %v", err)
	}
	return resp.Data.ID
}

// createProject is a helper that POSTs a project and returns its ID.
func createProject(t *testing.T, mux *http.ServeMux, title string) string {
	t.Helper()
	body := bytes.NewBufferString(`{"title":"` + title + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/projects", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project %q: expected 201, got %d: %s", title, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create project response: %v", err)
	}
	return resp.Data.ID
}

// createArea is a helper that POSTs an area and returns its ID.
func createArea(t *testing.T, mux *http.ServeMux, title string) string {
	t.Helper()
	body := bytes.NewBufferString(`{"title":"` + title + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/areas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create area %q: expected 201, got %d: %s", title, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create area response: %v", err)
	}
	return resp.Data.ID
}

// patchJSON sends a PATCH request and returns the recorder.
func patchJSON(t *testing.T, mux *http.ServeMux, path string, jsonBody string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// --- PATCH /tasks/{id} ---

func TestPatchTask_TitleOnly(t *testing.T) {
	mux := setupPatchTestServer(t)
	id := createTask(t, mux, "Original Title")

	w := patchJSON(t, mux, "/tasks/"+id, `{"title":"Updated Title"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var task struct {
		Title    string `json:"title"`
		Schedule int    `json:"schedule"`
	}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.Title != "Updated Title" {
		t.Errorf("expected title %q, got %q", "Updated Title", task.Title)
	}
	// schedule should remain default (0 = inbox)
	if task.Schedule != 0 {
		t.Errorf("expected schedule 0 (inbox), got %d", task.Schedule)
	}
}

func TestPatchTask_Schedule(t *testing.T) {
	mux := setupPatchTestServer(t)
	id := createTask(t, mux, "Schedule Test")

	w := patchJSON(t, mux, "/tasks/"+id, `{"schedule":1}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var task struct {
		Title    string `json:"title"`
		Schedule int    `json:"schedule"`
	}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.Title != "Schedule Test" {
		t.Errorf("expected title unchanged %q, got %q", "Schedule Test", task.Title)
	}
	if task.Schedule != 1 {
		t.Errorf("expected schedule 1, got %d", task.Schedule)
	}
}

func TestPatchTask_EmptyBody(t *testing.T) {
	mux := setupPatchTestServer(t)
	id := createTask(t, mux, "No Change")

	w := patchJSON(t, mux, "/tasks/"+id, `{}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var task struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.Title != "No Change" {
		t.Errorf("expected title %q unchanged, got %q", "No Change", task.Title)
	}
}

func TestPatchTask_ClearProjectId(t *testing.T) {
	mux := setupPatchTestServer(t)
	taskID := createTask(t, mux, "Has Project")
	projectID := createProject(t, mux, "My Project")

	// First assign to project
	w := patchJSON(t, mux, "/tasks/"+taskID, `{"projectId":"`+projectID+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("assign to project: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var assigned struct {
		ProjectID *string `json:"projectId"`
	}
	if err := json.NewDecoder(w.Body).Decode(&assigned); err != nil {
		t.Fatalf("decode assign response: %v", err)
	}
	if assigned.ProjectID == nil || *assigned.ProjectID != projectID {
		t.Fatalf("expected projectId %q after assign, got %v", projectID, assigned.ProjectID)
	}

	// Now clear projectId by patching with empty string
	w = patchJSON(t, mux, "/tasks/"+taskID, `{"projectId":""}`)
	if w.Code != http.StatusOK {
		t.Fatalf("clear project: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var cleared struct {
		ProjectID *string `json:"projectId"`
	}
	if err := json.NewDecoder(w.Body).Decode(&cleared); err != nil {
		t.Fatalf("decode clear response: %v", err)
	}
	if cleared.ProjectID != nil {
		t.Errorf("expected projectId nil after clear, got %q", *cleared.ProjectID)
	}
}

func TestPatchTask_NotFound(t *testing.T) {
	mux := setupPatchTestServer(t)

	w := patchJSON(t, mux, "/tasks/nonexistent-id", `{"title":"Nope"}`)
	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 or 422 for nonexistent task, got %d: %s", w.Code, w.Body.String())
	}
}

// --- PATCH /projects/{id} ---

func TestPatchProject_Title(t *testing.T) {
	mux := setupPatchTestServer(t)
	id := createProject(t, mux, "Old Project")

	w := patchJSON(t, mux, "/projects/"+id, `{"title":"New Project"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var project struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if project.Title != "New Project" {
		t.Errorf("expected title %q, got %q", "New Project", project.Title)
	}
}

func TestPatchProject_Color(t *testing.T) {
	mux := setupPatchTestServer(t)
	id := createProject(t, mux, "Colorful")

	w := patchJSON(t, mux, "/projects/"+id, `{"color":"blue"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var project struct {
		Title string `json:"title"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if project.Color != "blue" {
		t.Errorf("expected color %q, got %q", "blue", project.Color)
	}
	if project.Title != "Colorful" {
		t.Errorf("expected title unchanged %q, got %q", "Colorful", project.Title)
	}
}

func TestPatchProject_ClearAreaId(t *testing.T) {
	mux := setupPatchTestServer(t)
	projectID := createProject(t, mux, "In Area")
	areaID := createArea(t, mux, "Work Area")

	// Assign to area
	w := patchJSON(t, mux, "/projects/"+projectID, `{"areaId":"`+areaID+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("assign to area: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var assigned struct {
		AreaID *string `json:"areaId"`
	}
	if err := json.NewDecoder(w.Body).Decode(&assigned); err != nil {
		t.Fatalf("decode assign response: %v", err)
	}
	if assigned.AreaID == nil || *assigned.AreaID != areaID {
		t.Fatalf("expected areaId %q after assign, got %v", areaID, assigned.AreaID)
	}

	// Clear areaId
	w = patchJSON(t, mux, "/projects/"+projectID, `{"areaId":""}`)
	if w.Code != http.StatusOK {
		t.Fatalf("clear area: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var cleared struct {
		AreaID *string `json:"areaId"`
	}
	if err := json.NewDecoder(w.Body).Decode(&cleared); err != nil {
		t.Fatalf("decode clear response: %v", err)
	}
	if cleared.AreaID != nil {
		t.Errorf("expected areaId nil after clear, got %q", *cleared.AreaID)
	}
}

// --- PATCH /areas/{id} ---

func TestPatchArea_Title(t *testing.T) {
	mux := setupPatchTestServer(t)
	id := createArea(t, mux, "Old Area")

	w := patchJSON(t, mux, "/areas/"+id, `{"title":"New Area"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var area struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(w.Body).Decode(&area); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if area.Title != "New Area" {
		t.Errorf("expected title %q, got %q", "New Area", area.Title)
	}
}

// TestPatchTask_SectionWithoutProject asserts that setting a sectionId on a
// task that has no projectId is rejected with 422 (merged-state validation).
// Without this check, the handler validates each patch field in isolation
// and happily applies the section move, leaving the task in an invalid state
// that domain.Task.Validate() explicitly forbids.
func TestPatchTask_SectionWithoutProject(t *testing.T) {
	mux := setupPatchTestServer(t)

	// Create a task with no project assigned.
	taskID := createTask(t, mux, "Homeless task")

	// Create a project + section so the section ID is real (passes existence
	// check) but the task itself has no project.
	projectID := createProject(t, mux, "Some project")
	sectionID := createSection(t, mux, projectID, "Some section")

	// PATCH only sectionId — must be rejected because the merged task would
	// have sectionId without projectId.
	body := bytes.NewBufferString(`{"sectionId":"` + sectionID + `"}`)
	req := httptest.NewRequest(http.MethodPatch, "/tasks/"+taskID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PATCH setting sectionId on projectless task: expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

// TestPatchTask_SectionWithProject_Succeeds asserts the happy path: setting
// both projectId and sectionId in the same PATCH is valid as long as the
// section belongs to the project.
func TestPatchTask_SectionWithProject_Succeeds(t *testing.T) {
	mux := setupPatchTestServer(t)

	taskID := createTask(t, mux, "Task")
	projectID := createProject(t, mux, "Project")
	sectionID := createSection(t, mux, projectID, "Section")

	body := bytes.NewBufferString(`{"projectId":"` + projectID + `","sectionId":"` + sectionID + `"}`)
	req := httptest.NewRequest(http.MethodPatch, "/tasks/"+taskID, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PATCH with projectId+sectionId: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
