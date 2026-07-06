package service

import (
	"context"
	"testing"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
)

// newTestTaskService creates an in-memory DB, runs migrations, and returns a TaskService.
func newTestTaskService(t *testing.T) *TaskService {
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
	return NewTaskService(db, es, bus)
}

func TestTaskService_Create(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Buy groceries", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Title != "Buy groceries" {
		t.Errorf("expected title %q, got %q", "Buy groceries", task.Title)
	}
	if task.Status != domain.StatusPending {
		t.Errorf("expected status=pending, got %v", task.Status)
	}
	if task.Schedule != domain.ScheduleInbox {
		t.Errorf("expected schedule=inbox, got %v", task.Schedule)
	}

	// Verify it can be retrieved
	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Buy groceries" {
		t.Errorf("expected persisted title %q, got %q", "Buy groceries", got.Title)
	}
	if got.ID != task.ID {
		t.Errorf("expected ID %q, got %q", task.ID, got.ID)
	}
}

func TestTaskService_Create_EmptyTitle(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, testUserID, "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestTaskService_Complete(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Finish report", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Complete(ctx, testUserID, task.ID, "user-1"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after Complete: %v", err)
	}
	if got.Status != domain.StatusCompleted {
		t.Errorf("expected status=completed, got %v", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestTaskService_Cancel(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Draft email", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Cancel(ctx, testUserID, task.ID, "user-1"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after Cancel: %v", err)
	}
	if got.Status != domain.StatusCancelled {
		t.Errorf("expected status=cancelled, got %v", got.Status)
	}
}

func TestTaskService_UpdateTitle(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Original title", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.UpdateTitle(ctx, testUserID, task.ID, "Updated title", "user-1"); err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after UpdateTitle: %v", err)
	}
	if got.Title != "Updated title" {
		t.Errorf("expected title %q, got %q", "Updated title", got.Title)
	}
}

func TestTaskService_UpdateTitle_Empty(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Some task", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = svc.UpdateTitle(ctx, testUserID, task.ID, "", "user-1")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestTaskService_UpdateSchedule(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Plan week", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.UpdateSchedule(ctx, testUserID, task.ID, domain.ScheduleAnytime, "user-1"); err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after UpdateSchedule: %v", err)
	}
	if got.Schedule != domain.ScheduleAnytime {
		t.Errorf("expected schedule=anytime, got %v", got.Schedule)
	}
}

func TestTaskService_UpdateNotes(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Take notes", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.UpdateNotes(ctx, testUserID, task.ID, "Some important notes", "user-1"); err != nil {
		t.Fatalf("UpdateNotes: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after UpdateNotes: %v", err)
	}
	if got.Notes != "Some important notes" {
		t.Errorf("expected notes %q, got %q", "Some important notes", got.Notes)
	}
}

func TestTaskService_List(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, testUserID, "Task 1", "user-1")
	if err != nil {
		t.Fatalf("Create task 1: %v", err)
	}
	_, err = svc.Create(ctx, testUserID, "Task 2", "user-1")
	if err != nil {
		t.Fatalf("Create task 2: %v", err)
	}

	tasks, err := svc.List(ctx, testUserID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestTaskService_Delete(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Task to delete", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, testUserID, task.ID, "user-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get should return error since the task is soft-deleted
	_, err = svc.Get(ctx, testUserID, task.ID)
	if err == nil {
		t.Fatal("expected error for deleted task, got nil")
	}
}

func TestTaskService_Delete_NotInList(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Task to delete", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, testUserID, task.ID, "user-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	tasks, err := svc.List(ctx, testUserID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, t2 := range tasks {
		if t2.ID == task.ID {
			t.Error("deleted task should not appear in List")
		}
	}
}

func TestTaskService_MoveToArea(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Area task", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	areaSvc := &AreaService{queries: svc.queries, events: svc.events, bus: svc.bus}
	area, err := areaSvc.Create(ctx, testUserID, "Test area", "user-1")
	if err != nil {
		t.Fatalf("Create area: %v", err)
	}
	areaID := area.ID
	if err := svc.MoveToArea(ctx, testUserID, task.ID, &areaID, "user-1"); err != nil {
		t.Fatalf("MoveToArea: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after MoveToArea: %v", err)
	}
	if got.AreaID == nil || *got.AreaID != areaID {
		t.Errorf("expected area_id %q, got %v", areaID, got.AreaID)
	}

	// Remove from area
	if err := svc.MoveToArea(ctx, testUserID, task.ID, nil, "user-1"); err != nil {
		t.Fatalf("MoveToArea (remove): %v", err)
	}

	got, err = svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after MoveToArea (remove): %v", err)
	}
	if got.AreaID != nil {
		t.Errorf("expected nil area_id, got %v", *got.AreaID)
	}
}

func TestTaskService_SetRecurrence(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Recurring task", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rule := &domain.RecurrenceRule{
		Mode:     domain.RecurrenceModeFixed,
		Interval: 1,
		Unit:     domain.RecurrenceUnitWeek,
	}

	if err := svc.SetRecurrence(ctx, testUserID, task.ID, rule, "user-1"); err != nil {
		t.Fatalf("SetRecurrence: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after SetRecurrence: %v", err)
	}
	if got.RecurrenceRule == nil {
		t.Fatal("expected RecurrenceRule to be set")
	}
	if got.RecurrenceRule.Interval != 1 {
		t.Errorf("expected interval 1, got %d", got.RecurrenceRule.Interval)
	}
	if got.RecurrenceRule.Unit != domain.RecurrenceUnitWeek {
		t.Errorf("expected unit=week, got %v", got.RecurrenceRule.Unit)
	}

	// Remove recurrence
	if err := svc.SetRecurrence(ctx, testUserID, task.ID, nil, "user-1"); err != nil {
		t.Fatalf("SetRecurrence (remove): %v", err)
	}

	got, err = svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after SetRecurrence (remove): %v", err)
	}
	if got.RecurrenceRule != nil {
		t.Error("expected RecurrenceRule to be nil after removal")
	}
}

func TestTaskService_ListByProject(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task1, err := svc.Create(ctx, testUserID, "Task in project", "user-1")
	if err != nil {
		t.Fatalf("Create task 1: %v", err)
	}

	projectSvc := &ProjectService{queries: svc.queries, events: svc.events, bus: svc.bus}
	project, err := projectSvc.Create(ctx, testUserID, "Test project", "user-1")
	if err != nil {
		t.Fatalf("Create project: %v", err)
	}
	projectID := project.ID
	if err := svc.MoveToProject(ctx, testUserID, task1.ID, &projectID, "user-1"); err != nil {
		t.Fatalf("MoveToProject: %v", err)
	}

	// Create a task NOT in the project
	if _, err := svc.Create(ctx, testUserID, "Task without project", "user-1"); err != nil {
		t.Fatalf("Create task 2: %v", err)
	}

	tasks, err := svc.ListByProject(ctx, testUserID, projectID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != task1.ID {
		t.Errorf("expected task %s, got %s", task1.ID, tasks[0].ID)
	}
}

func TestTaskService_ListBySchedule(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task1, err := svc.Create(ctx, testUserID, "Someday task", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.UpdateSchedule(ctx, testUserID, task1.ID, domain.ScheduleSomeday, "user-1"); err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}

	// task2 stays in inbox (default)
	if _, err := svc.Create(ctx, testUserID, "Inbox task", "user-1"); err != nil {
		t.Fatalf("Create task 2: %v", err)
	}

	tasks, err := svc.ListBySchedule(ctx, testUserID, domain.ScheduleSomeday)
	if err != nil {
		t.Fatalf("ListBySchedule: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 someday task, got %d", len(tasks))
	}
}

func TestTaskService_Reorder(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	task, err := svc.Create(ctx, testUserID, "Reorderable task", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Reorder(ctx, testUserID, task.ID, 5, "user-1"); err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	got, err := svc.Get(ctx, testUserID, task.ID)
	if err != nil {
		t.Fatalf("Get after Reorder: %v", err)
	}
	if got.Index != 5 {
		t.Errorf("expected index=5, got %d", got.Index)
	}
}

// --- TaskLink tests ---

// Links must be symmetric: if A links to B, then B also sees the link to A.
// The bug was storing a single directed row and only hydrating the outgoing
// side, so task B never surfaced the link from its own GET.
func TestTaskService_AddLink_IsBidirectional(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	a, err := svc.Create(ctx, testUserID, "A", "user-1")
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	b, err := svc.Create(ctx, testUserID, "B", "user-1")
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	if err := svc.AddLink(ctx, testUserID, a.ID, b.ID, "user-1"); err != nil {
		t.Fatalf("AddLink: %v", err)
	}

	gotA, err := svc.Get(ctx, testUserID, a.ID)
	if err != nil {
		t.Fatalf("Get A: %v", err)
	}
	gotB, err := svc.Get(ctx, testUserID, b.ID)
	if err != nil {
		t.Fatalf("Get B: %v", err)
	}

	if len(gotA.LinkedTaskIDs) != 1 || gotA.LinkedTaskIDs[0] != b.ID {
		t.Errorf("A.LinkedTaskIDs = %v, want [%q]", gotA.LinkedTaskIDs, b.ID)
	}
	if len(gotB.LinkedTaskIDs) != 1 || gotB.LinkedTaskIDs[0] != a.ID {
		t.Errorf("B.LinkedTaskIDs = %v, want [%q]", gotB.LinkedTaskIDs, a.ID)
	}
}

// RemoveLink must clean up both directions.
func TestTaskService_RemoveLink_RemovesBothDirections(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	a, _ := svc.Create(ctx, testUserID, "A", "user-1")
	b, _ := svc.Create(ctx, testUserID, "B", "user-1")

	if err := svc.AddLink(ctx, testUserID, a.ID, b.ID, "user-1"); err != nil {
		t.Fatalf("AddLink: %v", err)
	}
	if err := svc.RemoveLink(ctx, testUserID, a.ID, b.ID, "user-1"); err != nil {
		t.Fatalf("RemoveLink: %v", err)
	}

	gotA, _ := svc.Get(ctx, testUserID, a.ID)
	gotB, _ := svc.Get(ctx, testUserID, b.ID)

	if len(gotA.LinkedTaskIDs) != 0 {
		t.Errorf("A.LinkedTaskIDs = %v, want []", gotA.LinkedTaskIDs)
	}
	if len(gotB.LinkedTaskIDs) != 0 {
		t.Errorf("B.LinkedTaskIDs = %v, want []", gotB.LinkedTaskIDs)
	}
}

// Linking a task to itself makes no semantic sense and caused the task to
// appear in its own linkedTaskIds. Must be rejected.
func TestTaskService_AddLink_RejectsSelfLink(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	a, _ := svc.Create(ctx, testUserID, "A", "user-1")
	if err := svc.AddLink(ctx, testUserID, a.ID, a.ID, "user-1"); err == nil {
		t.Fatal("expected error when linking task to itself, got nil")
	}
}

// Adding the same link twice is idempotent (INSERT OR IGNORE).
func TestTaskService_AddLink_Idempotent(t *testing.T) {
	svc := newTestTaskService(t)
	ctx := context.Background()

	a, _ := svc.Create(ctx, testUserID, "A", "user-1")
	b, _ := svc.Create(ctx, testUserID, "B", "user-1")

	if err := svc.AddLink(ctx, testUserID, a.ID, b.ID, "user-1"); err != nil {
		t.Fatalf("AddLink 1: %v", err)
	}
	if err := svc.AddLink(ctx, testUserID, a.ID, b.ID, "user-1"); err != nil {
		t.Fatalf("AddLink 2: %v", err)
	}

	gotA, _ := svc.Get(ctx, testUserID, a.ID)
	if len(gotA.LinkedTaskIDs) != 1 {
		t.Errorf("expected exactly 1 link after 2 adds, got %v", gotA.LinkedTaskIDs)
	}
}
