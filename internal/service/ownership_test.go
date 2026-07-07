package service

import (
	"context"
	"errors"
	"testing"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
)

// ownershipTestSetup wires up all 8 services against a single shared in-memory
// DB so cross-service ownership checks (spec §2.4) can be exercised the way
// they run in production: each service holds its own *sqlc.Queries, but all
// point at the same underlying database.
type ownershipTestSetup struct {
	tasks      *TaskService
	projects   *ProjectService
	areas      *AreaService
	sections   *SectionService
	tags       *TagService
	locations  *LocationService
	checklists *ChecklistService
	activities *ActivityService
}

func newOwnershipTestSetup(t *testing.T) *ownershipTestSetup {
	t.Helper()
	db := newTestDB(t)
	es := event.NewEventStore(db)
	bus := event.NewBus()
	return &ownershipTestSetup{
		tasks:      NewTaskService(db, es, bus),
		projects:   NewProjectService(db, es, bus),
		areas:      NewAreaService(db, es, bus),
		sections:   NewSectionService(db, es, bus),
		tags:       NewTagService(db, es, bus),
		locations:  NewLocationService(db, es, bus),
		checklists: NewChecklistService(db, es, bus),
		activities: NewActivityService(db, es, bus),
	}
}

// 1. TaskService.MoveToProject (spec: "Set task's project")
func TestOwnership_TaskMoveToProject_RejectsCrossUserProject(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	projB, err := s.projects.Create(ctx, "user-b", "project B", "actor-b")
	if err != nil {
		t.Fatalf("Create project B: %v", err)
	}

	err = s.tasks.MoveToProject(ctx, "user-a", taskA.ID, &projB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 2. TaskService.MoveToArea (spec: "Set task's area")
func TestOwnership_TaskMoveToArea_RejectsCrossUserArea(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	areaB, err := s.areas.Create(ctx, "user-b", "area B", "actor-b")
	if err != nil {
		t.Fatalf("Create area B: %v", err)
	}

	err = s.tasks.MoveToArea(ctx, "user-a", taskA.ID, &areaB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 3. TaskService.MoveToSection (spec: "Set task's section" — cross-user section)
func TestOwnership_TaskMoveToSection_RejectsCrossUserSection(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	projB, err := s.projects.Create(ctx, "user-b", "project B", "actor-b")
	if err != nil {
		t.Fatalf("Create project B: %v", err)
	}
	sectionB, err := s.sections.Create(ctx, "user-b", "section B", projB.ID, "actor-b")
	if err != nil {
		t.Fatalf("Create section B: %v", err)
	}

	err = s.tasks.MoveToSection(ctx, "user-a", taskA.ID, &sectionB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 3b. TaskService.MoveToSection — same-user section that belongs to a
// different project than the task's current project must also be rejected
// (spec §2.4: "confirm the section's project_id matches the task's project_id").
func TestOwnership_TaskMoveToSection_RejectsMismatchedProject(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	projA1, err := s.projects.Create(ctx, "user-a", "project A1", "actor-a")
	if err != nil {
		t.Fatalf("Create project A1: %v", err)
	}
	projA2, err := s.projects.Create(ctx, "user-a", "project A2", "actor-a")
	if err != nil {
		t.Fatalf("Create project A2: %v", err)
	}
	sectionA2, err := s.sections.Create(ctx, "user-a", "section in A2", projA2.ID, "actor-a")
	if err != nil {
		t.Fatalf("Create section in A2: %v", err)
	}

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	if err := s.tasks.MoveToProject(ctx, "user-a", taskA.ID, &projA1.ID, "actor-a"); err != nil {
		t.Fatalf("MoveToProject: %v", err)
	}

	// task is in projA1, section belongs to projA2 -> must be rejected even
	// though both are owned by user-a.
	err = s.tasks.MoveToSection(ctx, "user-a", taskA.ID, &sectionA2.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for mismatched project, got %v", err)
	}
}

// 4. TaskService.SetLocation (spec: "Set task's location")
func TestOwnership_TaskSetLocation_RejectsCrossUserLocation(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	locB, err := s.locations.Create(ctx, "user-b", "location B", "actor-b")
	if err != nil {
		t.Fatalf("Create location B: %v", err)
	}

	err = s.tasks.SetLocation(ctx, "user-a", taskA.ID, &locB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 5. TaskService.AddLink (spec: "Add task link")
func TestOwnership_TaskAddLink_RejectsCrossUserTask(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	taskB, err := s.tasks.Create(ctx, "user-b", "task B", "actor-b")
	if err != nil {
		t.Fatalf("Create task B: %v", err)
	}

	err = s.tasks.AddLink(ctx, "user-a", taskA.ID, taskB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 6. TaskService.AddTag (spec: "Add tag to task/project")
func TestOwnership_TaskAddTag_RejectsCrossUserTag(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	tagB, err := s.tags.Create(ctx, "user-b", "tag B", "actor-b")
	if err != nil {
		t.Fatalf("Create tag B: %v", err)
	}

	err = s.tasks.AddTag(ctx, "user-a", taskA.ID, tagB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 7. ProjectService.MoveToArea (spec: "Move project to area")
func TestOwnership_ProjectMoveToArea_RejectsCrossUserArea(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	projA, err := s.projects.Create(ctx, "user-a", "project A", "actor-a")
	if err != nil {
		t.Fatalf("Create project A: %v", err)
	}
	areaB, err := s.areas.Create(ctx, "user-b", "area B", "actor-b")
	if err != nil {
		t.Fatalf("Create area B: %v", err)
	}

	err = s.projects.MoveToArea(ctx, "user-a", projA.ID, &areaB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 8. ProjectService.AddTag (spec: "Add tag to task/project")
func TestOwnership_ProjectAddTag_RejectsCrossUserTag(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	projA, err := s.projects.Create(ctx, "user-a", "project A", "actor-a")
	if err != nil {
		t.Fatalf("Create project A: %v", err)
	}
	tagB, err := s.tags.Create(ctx, "user-b", "tag B", "actor-b")
	if err != nil {
		t.Fatalf("Create tag B: %v", err)
	}

	err = s.projects.AddTag(ctx, "user-a", projA.ID, tagB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 9. SectionService.Create (spec: "Create section in project")
func TestOwnership_SectionCreate_RejectsCrossUserProject(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	projB, err := s.projects.Create(ctx, "user-b", "project B", "actor-b")
	if err != nil {
		t.Fatalf("Create project B: %v", err)
	}

	_, err = s.sections.Create(ctx, "user-a", "section", projB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 10. ChecklistService.AddItem (spec: "Add checklist item to task")
func TestOwnership_ChecklistAddItem_RejectsCrossUserTask(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskB, err := s.tasks.Create(ctx, "user-b", "task B", "actor-b")
	if err != nil {
		t.Fatalf("Create task B: %v", err)
	}

	_, err = s.checklists.AddItem(ctx, "user-a", "buy milk", taskB.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 11. ActivityService.Add (batch review P2: Add() inserted an activity
// referencing taskID without verifying the caller owns that task, unlike its
// sibling ChecklistService.AddItem).
func TestOwnership_ActivityAdd_RejectsCrossUserTask(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskB, err := s.tasks.Create(ctx, "user-b", "task B", "actor-b")
	if err != nil {
		t.Fatalf("Create task B: %v", err)
	}

	_, err = s.activities.Add(ctx, "user-a", taskB.ID, "actor-a", domain.ActorHuman, domain.ActivityComment, "hi")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 12. TaskService.AddTag (batch review P2: AddTag verified the TAG belongs
// to the user but not the TASK, so a caller could tag another user's task
// using their own tag).
func TestOwnership_TaskAddTag_RejectsCrossUserTask(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskB, err := s.tasks.Create(ctx, "user-b", "task B", "actor-b")
	if err != nil {
		t.Fatalf("Create task B: %v", err)
	}
	tagA, err := s.tags.Create(ctx, "user-a", "tag A", "actor-a")
	if err != nil {
		t.Fatalf("Create tag A: %v", err)
	}

	err = s.tasks.AddTag(ctx, "user-a", taskB.ID, tagA.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// 13. TaskService.AddLink primary-task side (batch review P2 follow-up:
// AddLink only verified the related task's ownership, not the primary
// task's, so a caller could link another user's task to one of their own).
func TestOwnership_TaskAddLink_RejectsCrossUserPrimaryTask(t *testing.T) {
	s := newOwnershipTestSetup(t)
	ctx := context.Background()

	taskA, err := s.tasks.Create(ctx, "user-a", "task A", "actor-a")
	if err != nil {
		t.Fatalf("Create task A: %v", err)
	}
	taskB, err := s.tasks.Create(ctx, "user-b", "task B", "actor-b")
	if err != nil {
		t.Fatalf("Create task B: %v", err)
	}

	err = s.tasks.AddLink(ctx, "user-a", taskB.ID, taskA.ID, "actor-a")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
