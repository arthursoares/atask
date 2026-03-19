package client_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/client"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// setupTestServer creates an in-memory DB, migrates it, wires all services and handlers,
// spins up an httptest.Server, registers a test user, logs in, and returns an authenticated client.
func setupTestServer(t *testing.T) (*httptest.Server, *client.Client) {
	t.Helper()

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	bus := event.NewBus()
	es := event.NewEventStore(db)
	streamManager := event.NewStreamManager(bus)

	jwtSecret := "test-secret"

	authSvc := service.NewAuthService(db, jwtSecret)
	areaSvc := service.NewAreaService(db, es, bus)
	taskSvc := service.NewTaskService(db, es, bus)
	projectSvc := service.NewProjectService(db, es, bus)
	sectionSvc := service.NewSectionService(db, es, bus)
	tagSvc := service.NewTagService(db, es, bus)
	locationSvc := service.NewLocationService(db, es, bus)
	checklistSvc := service.NewChecklistService(db, es, bus)
	activitySvc := service.NewActivityService(db, es, bus)

	authHandler := api.NewAuthHandler(authSvc)
	areaHandler := api.NewAreaHandler(areaSvc)
	taskHandler := api.NewTaskHandler(taskSvc)
	projectHandler := api.NewProjectHandler(projectSvc)
	sectionHandler := api.NewSectionHandler(sectionSvc)
	tagHandler := api.NewTagHandler(tagSvc)
	locationHandler := api.NewLocationHandler(locationSvc)
	checklistHandler := api.NewChecklistHandler(checklistSvc)
	activityHandler := api.NewActivityHandler(activitySvc)
	viewHandler := api.NewViewHandler(db)
	eventsHandler := api.NewEventsHandler(streamManager)
	syncHandler := api.NewSyncHandler(es)

	handler := api.NewRouter(
		areaHandler,
		taskHandler,
		projectHandler,
		sectionHandler,
		tagHandler,
		locationHandler,
		checklistHandler,
		activityHandler,
		viewHandler,
		eventsHandler,
		syncHandler,
		authHandler,
		authSvc,
	)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := client.New(srv.URL, "")

	ctx := context.Background()
	token, err := c.Register(ctx, "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	c.SetToken(token)

	return srv, c
}

func TestClient_Areas(t *testing.T) {
	_, c := setupTestServer(t)
	ctx := context.Background()

	// Create an area
	area, err := c.CreateArea(ctx, "Work")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	if area.ID == "" {
		t.Error("expected non-empty area ID")
	}
	if area.Title != "Work" {
		t.Errorf("expected title %q, got %q", "Work", area.Title)
	}

	// Create a second area
	area2, err := c.CreateArea(ctx, "Personal")
	if err != nil {
		t.Fatalf("CreateArea (second): %v", err)
	}
	if area2.Title != "Personal" {
		t.Errorf("expected title %q, got %q", "Personal", area2.Title)
	}

	// List areas
	areas, err := c.ListAreas(ctx)
	if err != nil {
		t.Fatalf("ListAreas: %v", err)
	}
	if len(areas) != 2 {
		t.Errorf("expected 2 areas, got %d", len(areas))
	}

	// Verify IDs are present
	ids := make(map[string]bool)
	for _, a := range areas {
		ids[a.ID] = true
	}
	if !ids[area.ID] {
		t.Errorf("expected area ID %q in list", area.ID)
	}
	if !ids[area2.ID] {
		t.Errorf("expected area2 ID %q in list", area2.ID)
	}
}

func TestClient_Tasks(t *testing.T) {
	_, c := setupTestServer(t)
	ctx := context.Background()

	// Create a task
	task, err := c.CreateTask(ctx, "Buy groceries")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ID == "" {
		t.Error("expected non-empty task ID")
	}
	if task.Title != "Buy groceries" {
		t.Errorf("expected title %q, got %q", "Buy groceries", task.Title)
	}

	// Inbox should have the task (schedule=0 by default)
	inbox, err := c.ListInbox(ctx)
	if err != nil {
		t.Fatalf("ListInbox: %v", err)
	}
	if len(inbox) != 1 {
		t.Errorf("expected 1 task in inbox, got %d", len(inbox))
	}

	// Complete the task
	if err := c.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// Inbox should now be empty
	inbox, err = c.ListInbox(ctx)
	if err != nil {
		t.Fatalf("ListInbox after complete: %v", err)
	}
	if len(inbox) != 0 {
		t.Errorf("expected 0 tasks in inbox after completion, got %d", len(inbox))
	}

	// Task should appear in logbook
	logbook, err := c.ListLogbook(ctx)
	if err != nil {
		t.Fatalf("ListLogbook: %v", err)
	}
	if len(logbook) != 1 {
		t.Errorf("expected 1 task in logbook, got %d", len(logbook))
	}
}

func TestClient_Projects(t *testing.T) {
	_, c := setupTestServer(t)
	ctx := context.Background()

	// Create a project
	project, err := c.CreateProject(ctx, "Launch Website")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == "" {
		t.Error("expected non-empty project ID")
	}
	if project.Title != "Launch Website" {
		t.Errorf("expected title %q, got %q", "Launch Website", project.Title)
	}

	// Create a second project
	project2, err := c.CreateProject(ctx, "Write Book")
	if err != nil {
		t.Fatalf("CreateProject (second): %v", err)
	}
	if project2.Title != "Write Book" {
		t.Errorf("expected title %q, got %q", "Write Book", project2.Title)
	}

	// List projects
	projects, err := c.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	// Verify IDs are present
	ids := make(map[string]bool)
	for _, p := range projects {
		ids[p.ID] = true
	}
	if !ids[project.ID] {
		t.Errorf("expected project ID %q in list", project.ID)
	}
	if !ids[project2.ID] {
		t.Errorf("expected project2 ID %q in list", project2.ID)
	}

	// Get a specific project
	fetched, err := c.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if fetched.ID != project.ID {
		t.Errorf("expected ID %q, got %q", project.ID, fetched.ID)
	}
	if fetched.Title != "Launch Website" {
		t.Errorf("expected title %q, got %q", "Launch Website", fetched.Title)
	}
}
