package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/google/uuid"
)

// --- Setup helpers ---

func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

func newTestTagService(t *testing.T) (*TagService, *store.DB) {
	t.Helper()
	db := newTestDB(t)
	es := event.NewEventStore(db)
	bus := event.NewBus()
	return NewTagService(db, es, bus), db
}

func newTestSectionService(t *testing.T) (*SectionService, *store.DB) {
	t.Helper()
	db := newTestDB(t)
	es := event.NewEventStore(db)
	bus := event.NewBus()
	return NewSectionService(db, es, bus), db
}

func newTestChecklistService(t *testing.T) (*ChecklistService, *store.DB) {
	t.Helper()
	db := newTestDB(t)
	es := event.NewEventStore(db)
	bus := event.NewBus()
	return NewChecklistService(db, es, bus), db
}

func newTestLocationService(t *testing.T) (*LocationService, *store.DB) {
	t.Helper()
	db := newTestDB(t)
	es := event.NewEventStore(db)
	bus := event.NewBus()
	return NewLocationService(db, es, bus), db
}

func newTestActivityService(t *testing.T) (*ActivityService, *store.DB) {
	t.Helper()
	db := newTestDB(t)
	es := event.NewEventStore(db)
	bus := event.NewBus()
	return NewActivityService(db, es, bus), db
}

// seedProject inserts a project directly for use in section/task tests.
func seedProject(t *testing.T, db *store.DB) string {
	t.Helper()
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	id := uuid.New().String()
	_, err := q.CreateProject(context.Background(), sqlc.CreateProjectParams{
		ID:        id,
		Title:     sql.NullString{String: "Test Project", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seedProject: %v", err)
	}
	return id
}

// seedTask inserts a task directly for use in checklist/activity tests.
func seedTask(t *testing.T, db *store.DB) string {
	t.Helper()
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	id := uuid.New().String()
	_, err := q.CreateTask(context.Background(), sqlc.CreateTaskParams{
		ID:        id,
		Title:     sql.NullString{String: "Test Task", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seedTask: %v", err)
	}
	return id
}

// --- TagService tests ---

func TestTagService_Create(t *testing.T) {
	svc, _ := newTestTagService(t)
	ctx := context.Background()

	tag, err := svc.Create(ctx, "Work", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tag.ID == "" {
		t.Error("expected non-empty ID")
	}
	if tag.Title != "Work" {
		t.Errorf("expected title %q, got %q", "Work", tag.Title)
	}

	// Verify it can be retrieved
	got, err := svc.Get(ctx, tag.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Work" {
		t.Errorf("expected persisted title %q, got %q", "Work", got.Title)
	}
}

func TestTagService_Create_EmptyTitle(t *testing.T) {
	svc, _ := newTestTagService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestTagService_Delete(t *testing.T) {
	svc, _ := newTestTagService(t)
	ctx := context.Background()

	tag, err := svc.Create(ctx, "ToDelete", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, tag.ID, "user-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should no longer be retrievable
	_, err = svc.Get(ctx, tag.ID)
	if err == nil {
		t.Fatal("expected error for deleted tag, got nil")
	}
}

func TestTagService_Rename(t *testing.T) {
	svc, _ := newTestTagService(t)
	ctx := context.Background()

	tag, err := svc.Create(ctx, "Old", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Rename(ctx, tag.ID, "New", "user-1"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, err := svc.Get(ctx, tag.ID)
	if err != nil {
		t.Fatalf("Get after Rename: %v", err)
	}
	if got.Title != "New" {
		t.Errorf("expected title %q, got %q", "New", got.Title)
	}
}

func TestTagService_UpdateShortcut(t *testing.T) {
	svc, _ := newTestTagService(t)
	ctx := context.Background()

	tag, err := svc.Create(ctx, "Shortcut Tag", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	sc := "ctrl+w"
	if err := svc.UpdateShortcut(ctx, tag.ID, &sc, "user-1"); err != nil {
		t.Fatalf("UpdateShortcut: %v", err)
	}

	got, err := svc.Get(ctx, tag.ID)
	if err != nil {
		t.Fatalf("Get after UpdateShortcut: %v", err)
	}
	if got.Shortcut == nil || *got.Shortcut != sc {
		t.Errorf("expected shortcut %q, got %v", sc, got.Shortcut)
	}

	// Clear shortcut
	if err := svc.UpdateShortcut(ctx, tag.ID, nil, "user-1"); err != nil {
		t.Fatalf("UpdateShortcut (clear): %v", err)
	}
	got, err = svc.Get(ctx, tag.ID)
	if err != nil {
		t.Fatalf("Get after clear shortcut: %v", err)
	}
	if got.Shortcut != nil {
		t.Errorf("expected nil shortcut after clear, got %q", *got.Shortcut)
	}
}

// --- SectionService tests ---

func TestSectionService_Create(t *testing.T) {
	svc, db := newTestSectionService(t)
	ctx := context.Background()

	projectID := seedProject(t, db)

	section, err := svc.Create(ctx, "Sprint 1", projectID, "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if section.ID == "" {
		t.Error("expected non-empty ID")
	}
	if section.Title != "Sprint 1" {
		t.Errorf("expected title %q, got %q", "Sprint 1", section.Title)
	}
	if section.ProjectID != projectID {
		t.Errorf("expected projectID %q, got %q", projectID, section.ProjectID)
	}

	// Verify it can be retrieved
	got, err := svc.Get(ctx, section.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Sprint 1" {
		t.Errorf("expected persisted title %q, got %q", "Sprint 1", got.Title)
	}
}

func TestSectionService_Create_EmptyTitle(t *testing.T) {
	svc, db := newTestSectionService(t)
	ctx := context.Background()

	projectID := seedProject(t, db)

	_, err := svc.Create(ctx, "", projectID, "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestSectionService_Create_EmptyProjectID(t *testing.T) {
	svc, _ := newTestSectionService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "Section", "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty projectID, got nil")
	}
}

func TestSectionService_ListByProject(t *testing.T) {
	svc, db := newTestSectionService(t)
	ctx := context.Background()

	projectID := seedProject(t, db)

	_, err := svc.Create(ctx, "Section A", projectID, "user-1")
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	_, err = svc.Create(ctx, "Section B", projectID, "user-1")
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	sections, err := svc.ListByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sections))
	}
}

func TestSectionService_Delete_Cascade(t *testing.T) {
	svc, db := newTestSectionService(t)
	ctx := context.Background()

	projectID := seedProject(t, db)
	section, err := svc.Create(ctx, "Sprint 2", projectID, "user-1")
	if err != nil {
		t.Fatalf("Create section: %v", err)
	}

	// Create a task in the section
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	taskID := uuid.New().String()
	_, err = q.CreateTask(ctx, sqlc.CreateTaskParams{
		ID:        taskID,
		Title:     sql.NullString{String: "Task in section", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		SectionID: sql.NullString{String: section.ID, Valid: true},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Delete with cascade=true
	if err := svc.Delete(ctx, section.ID, "user-1", true); err != nil {
		t.Fatalf("Delete (cascade): %v", err)
	}

	// Section should be deleted
	_, err = svc.Get(ctx, section.ID)
	if err == nil {
		t.Fatal("expected error for deleted section, got nil")
	}

	// Task should be tombstoned
	_, err = q.GetTask(ctx, taskID)
	if err == nil {
		t.Error("expected error for tombstoned task, got nil")
	}
}

func TestSectionService_Delete_Orphan(t *testing.T) {
	svc, db := newTestSectionService(t)
	ctx := context.Background()

	projectID := seedProject(t, db)
	section, err := svc.Create(ctx, "Orphan Section", projectID, "user-1")
	if err != nil {
		t.Fatalf("Create section: %v", err)
	}

	// Create a task in the section
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	taskID := uuid.New().String()
	_, err = q.CreateTask(ctx, sqlc.CreateTaskParams{
		ID:        taskID,
		Title:     sql.NullString{String: "Orphan task", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		SectionID: sql.NullString{String: section.ID, Valid: true},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Delete with cascade=false (orphan)
	if err := svc.Delete(ctx, section.ID, "user-1", false); err != nil {
		t.Fatalf("Delete (orphan): %v", err)
	}

	// Section should be deleted
	_, err = svc.Get(ctx, section.ID)
	if err == nil {
		t.Fatal("expected error for deleted section, got nil")
	}

	// Task should still exist (orphaned - section_id = NULL)
	task, err := q.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask after orphan: %v", err)
	}
	if task.SectionID.Valid {
		t.Errorf("expected task section_id to be NULL after orphan, got %q", task.SectionID.String)
	}
}

// --- ChecklistService tests ---

func TestChecklistService_AddItem(t *testing.T) {
	svc, db := newTestChecklistService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	item, err := svc.AddItem(ctx, "Buy groceries", taskID, "user-1")
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if item.ID == "" {
		t.Error("expected non-empty ID")
	}
	if item.Title != "Buy groceries" {
		t.Errorf("expected title %q, got %q", "Buy groceries", item.Title)
	}
	if item.TaskID != taskID {
		t.Errorf("expected taskID %q, got %q", taskID, item.TaskID)
	}
	if item.Status != domain.ChecklistPending {
		t.Errorf("expected status=pending, got %v", item.Status)
	}

	// Verify retrieval
	got, err := svc.GetItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if got.Title != "Buy groceries" {
		t.Errorf("expected persisted title %q, got %q", "Buy groceries", got.Title)
	}
}

func TestChecklistService_AddItem_EmptyTitle(t *testing.T) {
	svc, db := newTestChecklistService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	_, err := svc.AddItem(ctx, "", taskID, "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestChecklistService_CompleteItem(t *testing.T) {
	svc, db := newTestChecklistService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	item, err := svc.AddItem(ctx, "Complete me", taskID, "user-1")
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := svc.CompleteItem(ctx, item.ID, "user-1"); err != nil {
		t.Fatalf("CompleteItem: %v", err)
	}

	got, err := svc.GetItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetItem after CompleteItem: %v", err)
	}
	if got.Status != domain.ChecklistCompleted {
		t.Errorf("expected status=completed, got %v", got.Status)
	}
}

func TestChecklistService_UncompleteItem(t *testing.T) {
	svc, db := newTestChecklistService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	item, err := svc.AddItem(ctx, "Uncomplete me", taskID, "user-1")
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := svc.CompleteItem(ctx, item.ID, "user-1"); err != nil {
		t.Fatalf("CompleteItem: %v", err)
	}

	if err := svc.UncompleteItem(ctx, item.ID, "user-1"); err != nil {
		t.Fatalf("UncompleteItem: %v", err)
	}

	got, err := svc.GetItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetItem after UncompleteItem: %v", err)
	}
	if got.Status != domain.ChecklistPending {
		t.Errorf("expected status=pending, got %v", got.Status)
	}
}

func TestChecklistService_ListByTask(t *testing.T) {
	svc, db := newTestChecklistService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	_, err := svc.AddItem(ctx, "Item 1", taskID, "user-1")
	if err != nil {
		t.Fatalf("AddItem 1: %v", err)
	}
	_, err = svc.AddItem(ctx, "Item 2", taskID, "user-1")
	if err != nil {
		t.Fatalf("AddItem 2: %v", err)
	}

	items, err := svc.ListByTask(ctx, taskID)
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestChecklistService_RemoveItem(t *testing.T) {
	svc, db := newTestChecklistService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	item, err := svc.AddItem(ctx, "Remove me", taskID, "user-1")
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := svc.RemoveItem(ctx, item.ID, "user-1"); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}

	_, err = svc.GetItem(ctx, item.ID)
	if err == nil {
		t.Fatal("expected error for removed item, got nil")
	}
}

// --- LocationService tests ---

func TestLocationService_Create(t *testing.T) {
	svc, _ := newTestLocationService(t)
	ctx := context.Background()

	loc, err := svc.Create(ctx, "Office", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if loc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if loc.Name != "Office" {
		t.Errorf("expected name %q, got %q", "Office", loc.Name)
	}

	// Verify retrieval
	got, err := svc.Get(ctx, loc.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Office" {
		t.Errorf("expected persisted name %q, got %q", "Office", got.Name)
	}
}

func TestLocationService_Create_EmptyName(t *testing.T) {
	svc, _ := newTestLocationService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

// Locations must honor client-provided IDs, matching the contract used by
// Task/Project/Area Create. The desktop app generates a UUID for an offline
// location and then references it from tasks — if the server renames the id
// on create, the task references dangle forever. Regression guard for the
// silent sync break that would otherwise happen.
func TestLocationService_Create_PreservesClientID(t *testing.T) {
	svc, _ := newTestLocationService(t)
	ctx := context.Background()

	const clientID = "client-generated-uuid-1234"
	loc, err := svc.Create(ctx, "Office", "user-1", clientID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if loc.ID != clientID {
		t.Errorf("expected ID %q, got %q", clientID, loc.ID)
	}

	// Verify retrieval round-trips the client id.
	got, err := svc.Get(ctx, clientID)
	if err != nil {
		t.Fatalf("Get by client id: %v", err)
	}
	if got.ID != clientID {
		t.Errorf("Get returned ID %q, want %q", got.ID, clientID)
	}
}

// Passing an empty id (or no id at all) must still generate a server-side
// UUID, preserving the existing behavior for API-only callers.
func TestLocationService_Create_EmptyIDGeneratesUUID(t *testing.T) {
	svc, _ := newTestLocationService(t)
	ctx := context.Background()

	loc, err := svc.Create(ctx, "Home", "user-1", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if loc.ID == "" {
		t.Error("expected server-generated ID, got empty")
	}
}

func TestLocationService_Delete(t *testing.T) {
	svc, db := newTestLocationService(t)
	ctx := context.Background()

	loc, err := svc.Create(ctx, "Home", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create a task with this location
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	taskID := uuid.New().String()
	_, err = q.CreateTask(ctx, sqlc.CreateTaskParams{
		ID:         taskID,
		Title:      sql.NullString{String: "Task with location", Valid: true},
		Notes:      "",
		Status:     int64(domain.StatusPending),
		Schedule:   int64(domain.ScheduleInbox),
		LocationID: sql.NullString{String: loc.ID, Valid: true},
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := svc.Delete(ctx, loc.ID, "user-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Location should be deleted
	_, err = svc.Get(ctx, loc.ID)
	if err == nil {
		t.Fatal("expected error for deleted location, got nil")
	}

	// Task's location_id should be NULL
	task, err := q.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask after location delete: %v", err)
	}
	if task.LocationID.Valid {
		t.Errorf("expected task location_id to be NULL after location delete, got %q", task.LocationID.String)
	}
}

func TestLocationService_Rename(t *testing.T) {
	svc, _ := newTestLocationService(t)
	ctx := context.Background()

	loc, err := svc.Create(ctx, "Old Name", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Rename(ctx, loc.ID, "New Name", "user-1"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, err := svc.Get(ctx, loc.ID)
	if err != nil {
		t.Fatalf("Get after Rename: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("expected name %q, got %q", "New Name", got.Name)
	}
}

// --- ActivityService tests ---

func TestActivityService_Add(t *testing.T) {
	svc, db := newTestActivityService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	activity, err := svc.Add(ctx, taskID, "user-1", domain.ActorHuman, domain.ActivityComment, "Great progress!")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if activity.ID == "" {
		t.Error("expected non-empty ID")
	}
	if activity.TaskID != taskID {
		t.Errorf("expected taskID %q, got %q", taskID, activity.TaskID)
	}
	if activity.ActorID != "user-1" {
		t.Errorf("expected actorID %q, got %q", "user-1", activity.ActorID)
	}
	if activity.ActorType != domain.ActorHuman {
		t.Errorf("expected actorType %q, got %q", domain.ActorHuman, activity.ActorType)
	}
	if activity.Type != domain.ActivityComment {
		t.Errorf("expected type %q, got %q", domain.ActivityComment, activity.Type)
	}
	if activity.Content != "Great progress!" {
		t.Errorf("expected content %q, got %q", "Great progress!", activity.Content)
	}
}

func TestActivityService_ListByTask(t *testing.T) {
	svc, db := newTestActivityService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	_, err := svc.Add(ctx, taskID, "user-1", domain.ActorHuman, domain.ActivityComment, "First comment")
	if err != nil {
		t.Fatalf("Add 1: %v", err)
	}
	_, err = svc.Add(ctx, taskID, "agent-1", domain.ActorAgent, domain.ActivityContextRequest, "Need more info")
	if err != nil {
		t.Fatalf("Add 2: %v", err)
	}

	activities, err := svc.ListByTask(ctx, taskID)
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("expected 2 activities, got %d", len(activities))
	}
}

func TestActivityService_ListByTask_Empty(t *testing.T) {
	svc, db := newTestActivityService(t)
	ctx := context.Background()

	taskID := seedTask(t, db)

	activities, err := svc.ListByTask(ctx, taskID)
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(activities) != 0 {
		t.Errorf("expected 0 activities, got %d", len(activities))
	}
}

// Project tag inbound sync depends on ProjectService.Get() returning a
// fully populated Tags array. If the hydration is missing, the Rust client's
// upsert_project sees tags as absent and never reconciles projectTags, so
// project tags silently never sync. This regression test locks in that the
// Get path always hydrates tags (as [] when empty, not nil).
func TestProjectService_Get_HydratesTags(t *testing.T) {
	svc, db := newTestProjectService(t)
	ctx := context.Background()

	// Seed a project and a tag directly so this test does not depend on a
	// tag service constructor.
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	projectID := uuid.New().String()
	_, err := q.CreateProject(ctx, sqlc.CreateProjectParams{
		ID:        projectID,
		Title:     sql.NullString{String: "Proj", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	tagID := uuid.New().String()
	_, err = q.CreateTag(ctx, sqlc.CreateTagParams{
		ID:        tagID,
		Title:     sql.NullString{String: "urgent", Valid: true},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if err := q.AddProjectTag(ctx, sqlc.AddProjectTagParams{
		ProjectID: projectID,
		TagID:     tagID,
	}); err != nil {
		t.Fatalf("AddProjectTag: %v", err)
	}

	got, err := svc.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != tagID {
		t.Errorf("expected Tags = [%q], got %v", tagID, got.Tags)
	}
}

// Even when the project has no tags, Tags must be a non-nil empty slice so
// the JSON wire format emits "tags": [] (not "tags": null). The Rust client
// inspects tags.as_array() before rewriting projectTags; a null tags field
// silently disables the sync path.
func TestProjectService_Get_EmptyTagsIsNotNil(t *testing.T) {
	svc, db := newTestProjectService(t)
	ctx := context.Background()

	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	projectID := uuid.New().String()
	_, err := q.CreateProject(ctx, sqlc.CreateProjectParams{
		ID:        projectID,
		Title:     sql.NullString{String: "No tags", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	got, err := svc.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Tags == nil {
		t.Error("expected Tags to be non-nil empty slice, got nil")
	}
	if len(got.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(got.Tags))
	}
}
