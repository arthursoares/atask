package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
)

// newTestProjectService creates an in-memory DB, runs migrations, and returns a ProjectService.
func newTestProjectService(t *testing.T) (*ProjectService, *store.DB) {
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
	return NewProjectService(db, es, bus), db
}

// createTaskInProject inserts a task directly into the DB with the given project_id.
func createTaskInProject(t *testing.T, db *store.DB, projectID string) string {
	t.Helper()
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	id := uuid.New().String()
	_, err := q.CreateTask(context.Background(), sqlc.CreateTaskParams{
		ID:        id,
		Title:     sql.NullString{String: "Test task", Valid: true},
		Notes:     "",
		Status:    int64(domain.StatusPending),
		Schedule:  int64(domain.ScheduleInbox),
		ProjectID: sql.NullString{String: projectID, Valid: true},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("createTaskInProject: %v", err)
	}
	return id
}

// createSectionInProject inserts a section directly into the DB for the given project.
func createSectionInProject(t *testing.T, db *store.DB, projectID string) string {
	t.Helper()
	q := sqlc.New(db.DB)
	now := time.Now().UTC()
	id := uuid.New().String()
	_, err := q.CreateSection(context.Background(), sqlc.CreateSectionParams{
		ID:        id,
		Title:     sql.NullString{String: "Test section", Valid: true},
		ProjectID: sql.NullString{String: projectID, Valid: true},
		Index:     0,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("createSectionInProject: %v", err)
	}
	return id
}

func TestProjectService_Create(t *testing.T) {
	svc, _ := newTestProjectService(t)
	ctx := context.Background()

	project, err := svc.Create(ctx, "Launch website", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if project.ID == "" {
		t.Error("expected non-empty ID")
	}
	if project.Title != "Launch website" {
		t.Errorf("expected title %q, got %q", "Launch website", project.Title)
	}
	if project.Status != domain.StatusPending {
		t.Errorf("expected status=pending, got %v", project.Status)
	}
	if project.Schedule != domain.ScheduleInbox {
		t.Errorf("expected schedule=inbox, got %v", project.Schedule)
	}

	// Verify it can be retrieved
	got, err := svc.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Launch website" {
		t.Errorf("expected persisted title %q, got %q", "Launch website", got.Title)
	}
	if got.ID != project.ID {
		t.Errorf("expected ID %q, got %q", project.ID, got.ID)
	}
}

func TestProjectService_Create_EmptyTitle(t *testing.T) {
	svc, _ := newTestProjectService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestProjectService_List(t *testing.T) {
	svc, _ := newTestProjectService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "Project 1", "user-1")
	if err != nil {
		t.Fatalf("Create project 1: %v", err)
	}
	_, err = svc.Create(ctx, "Project 2", "user-1")
	if err != nil {
		t.Fatalf("Create project 2: %v", err)
	}

	projects, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestProjectService_Complete_CascadeTasks(t *testing.T) {
	svc, db := newTestProjectService(t)
	ctx := context.Background()

	project, err := svc.Create(ctx, "Q4 Goals", "user-1")
	if err != nil {
		t.Fatalf("Create project: %v", err)
	}

	taskID1 := createTaskInProject(t, db, project.ID)
	taskID2 := createTaskInProject(t, db, project.ID)

	if err := svc.Complete(ctx, project.ID, "user-1"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Verify project status
	got, err := svc.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get project after Complete: %v", err)
	}
	if got.Status != domain.StatusCompleted {
		t.Errorf("expected project status=completed, got %v", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("expected project CompletedAt to be set")
	}

	// Verify cascade: both tasks should be completed
	q := sqlc.New(db.DB)
	task1, err := q.GetTask(ctx, taskID1)
	if err != nil {
		t.Fatalf("GetTask 1: %v", err)
	}
	if domain.Status(task1.Status) != domain.StatusCompleted {
		t.Errorf("expected task1 status=completed, got %v", task1.Status)
	}

	task2, err := q.GetTask(ctx, taskID2)
	if err != nil {
		t.Fatalf("GetTask 2: %v", err)
	}
	if domain.Status(task2.Status) != domain.StatusCompleted {
		t.Errorf("expected task2 status=completed, got %v", task2.Status)
	}
}

func TestProjectService_Cancel_CascadeTasks(t *testing.T) {
	svc, db := newTestProjectService(t)
	ctx := context.Background()

	project, err := svc.Create(ctx, "Abandoned project", "user-1")
	if err != nil {
		t.Fatalf("Create project: %v", err)
	}

	taskID1 := createTaskInProject(t, db, project.ID)
	taskID2 := createTaskInProject(t, db, project.ID)

	if err := svc.Cancel(ctx, project.ID, "user-1"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	// Verify project status
	got, err := svc.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get project after Cancel: %v", err)
	}
	if got.Status != domain.StatusCancelled {
		t.Errorf("expected project status=cancelled, got %v", got.Status)
	}

	// Verify cascade: both tasks should be cancelled
	q := sqlc.New(db.DB)
	task1, err := q.GetTask(ctx, taskID1)
	if err != nil {
		t.Fatalf("GetTask 1: %v", err)
	}
	if domain.Status(task1.Status) != domain.StatusCancelled {
		t.Errorf("expected task1 status=cancelled, got %v", task1.Status)
	}

	task2, err := q.GetTask(ctx, taskID2)
	if err != nil {
		t.Fatalf("GetTask 2: %v", err)
	}
	if domain.Status(task2.Status) != domain.StatusCancelled {
		t.Errorf("expected task2 status=cancelled, got %v", task2.Status)
	}
}

func TestProjectService_Delete_CascadeAll(t *testing.T) {
	svc, db := newTestProjectService(t)
	ctx := context.Background()

	project, err := svc.Create(ctx, "Big project", "user-1")
	if err != nil {
		t.Fatalf("Create project: %v", err)
	}

	taskID1 := createTaskInProject(t, db, project.ID)
	taskID2 := createTaskInProject(t, db, project.ID)
	sectionID := createSectionInProject(t, db, project.ID)

	if err := svc.Delete(ctx, project.ID, "user-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify project is tombstoned (Get should return error)
	_, err = svc.Get(ctx, project.ID)
	if err == nil {
		t.Fatal("expected error for deleted project, got nil")
	}

	// Verify tasks are tombstoned
	q := sqlc.New(db.DB)
	_, err = q.GetTask(ctx, taskID1)
	if err == nil {
		t.Error("expected error for deleted task1, got nil")
	}
	_, err = q.GetTask(ctx, taskID2)
	if err == nil {
		t.Error("expected error for deleted task2, got nil")
	}

	// Verify section is tombstoned
	_, err = q.GetSection(ctx, sectionID)
	if err == nil {
		t.Error("expected error for deleted section, got nil")
	}
}

func TestProjectService_UpdateTitle(t *testing.T) {
	svc, _ := newTestProjectService(t)
	ctx := context.Background()

	project, err := svc.Create(ctx, "Old title", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.UpdateTitle(ctx, project.ID, "New title", "user-1"); err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}

	got, err := svc.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get after UpdateTitle: %v", err)
	}
	if got.Title != "New title" {
		t.Errorf("expected title %q, got %q", "New title", got.Title)
	}
}

func TestProjectService_SetDeadline(t *testing.T) {
	svc, _ := newTestProjectService(t)
	ctx := context.Background()

	project, err := svc.Create(ctx, "Deadline project", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	deadline := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := svc.SetDeadline(ctx, project.ID, &deadline, "user-1"); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}

	got, err := svc.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get after SetDeadline: %v", err)
	}
	if got.Deadline == nil {
		t.Fatal("expected Deadline to be set")
	}
	if got.Deadline.Format("2006-01-02") != "2026-12-31" {
		t.Errorf("expected deadline %q, got %q", "2026-12-31", got.Deadline.Format("2006-01-02"))
	}

	// Remove deadline
	if err := svc.SetDeadline(ctx, project.ID, nil, "user-1"); err != nil {
		t.Fatalf("SetDeadline (remove): %v", err)
	}

	got, err = svc.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Get after SetDeadline (remove): %v", err)
	}
	if got.Deadline != nil {
		t.Error("expected Deadline to be nil after removal")
	}
}
