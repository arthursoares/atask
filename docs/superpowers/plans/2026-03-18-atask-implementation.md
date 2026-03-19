# atask Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an event-sourced Go API server for an AI-first task manager with semantic operations, dual-stream events, SSE subscriptions, and JWT/API-key auth.

**Architecture:** Monolith Go binary with clean layer separation: domain types → event store → SQLite persistence → HTTP API. Every mutation flows through the event system, producing both delta events (sync) and domain events (agent intelligence). SSE delivers domain events to subscribers.

**Tech Stack:** Go 1.22+, SQLite (modernc.org/sqlite), sqlc, goose, log/slog, net/http, Docker

**Spec:** `docs/superpowers/specs/2026-03-18-atask-design.md`

---

## File Map

```
atask/
├── cmd/atask/main.go                    → entry point, wiring, server startup
├── internal/
│   ├── domain/
│   │   ├── types.go                          → enums (Status, Schedule, ActorType, ActivityType, etc.)
│   │   ├── task.go                           → Task entity + validation
│   │   ├── project.go                        → Project entity + validation
│   │   ├── area.go                           → Area entity + validation
│   │   ├── tag.go                            → Tag entity + validation
│   │   ├── section.go                        → Section entity + validation
│   │   ├── checklist.go                      → ChecklistItem entity + validation
│   │   ├── location.go                       → Location entity + validation
│   │   ├── activity.go                       → Activity entity + validation
│   │   ├── recurrence.go                     → RecurrenceRule type + logic
│   │   └── event.go                          → DomainEvent + DeltaEvent types, event catalog constants
│   ├── store/
│   │   ├── db.go                             → database connection, transaction helpers
│   │   ├── migrations/
│   │   │   └── 001_initial_schema.sql        → all tables, indexes, triggers
│   │   ├── queries/
│   │   │   ├── tasks.sql                     → task CRUD queries
│   │   │   ├── projects.sql                  → project CRUD queries
│   │   │   ├── areas.sql                     → area CRUD queries
│   │   │   ├── tags.sql                      → tag CRUD queries
│   │   │   ├── sections.sql                  → section CRUD queries
│   │   │   ├── checklist_items.sql           → checklist item CRUD queries
│   │   │   ├── locations.sql                 → location CRUD queries
│   │   │   ├── activities.sql                → activity CRUD queries
│   │   │   ├── events.sql                    → delta + domain event queries
│   │   │   ├── task_tags.sql                 → task-tag join queries
│   │   │   ├── task_links.sql                → task-task soft link queries
│   │   │   ├── auth.sql                      → user + api key queries
│   │   │   └── views.sql                     → view queries (inbox, today, upcoming, someday, logbook)
│   │   └── sqlc/                             → generated Go code (do not edit)
│   ├── event/
│   │   ├── store.go                          → delta event persistence (append, query since cursor)
│   │   ├── bus.go                            → in-process pub/sub for domain events
│   │   └── stream.go                         → SSE subscription manager + HTTP handler
│   ├── service/
│   │   ├── task_service.go                   → task operations (create, complete, schedule, etc.)
│   │   ├── project_service.go                → project operations (create, complete, cancel + cascade)
│   │   ├── area_service.go                   → area operations
│   │   ├── tag_service.go                    → tag operations
│   │   ├── section_service.go                → section operations
│   │   ├── checklist_service.go              → checklist item operations
│   │   ├── location_service.go               → location operations
│   │   ├── activity_service.go               → activity operations
│   │   └── auth_service.go                   → auth + API key operations
│   └── api/
│       ├── router.go                         → route registration, middleware stack
│       ├── middleware.go                      → auth middleware, logging, request ID
│       ├── response.go                       → response envelope helpers
│       ├── tasks.go                          → task HTTP handlers
│       ├── projects.go                       → project HTTP handlers
│       ├── areas.go                          → area HTTP handlers
│       ├── tags.go                           → tag HTTP handlers
│       ├── sections.go                       → section HTTP handlers
│       ├── checklist.go                      → checklist HTTP handlers
│       ├── locations.go                      → location HTTP handlers
│       ├── activities.go                     → activity HTTP handlers
│       ├── views.go                          → view HTTP handlers
│       ├── events.go                         → SSE stream HTTP handler
│       ├── sync.go                           → sync/deltas HTTP handler
│       └── auth.go                           → auth HTTP handlers
├── sqlc.yaml                                 → sqlc configuration
├── go.mod
├── go.sum
├── Makefile                                  → build, test, lint, migrate, docker targets
├── Dockerfile                                → multi-stage build
├── docker-compose.yml                        → single service + volume
└── .golangci.yml                             → linter configuration
```

**Note:** The spec listed a flat `api → domain → event → store` flow. This plan adds a `service` layer between `api` and `domain`/`store`/`event` — this is where the orchestration lives (validate domain rules, persist via store, emit events). The API handlers stay thin (parse request → call service → write response). This keeps the domain pure and the API dumb.

---

## Phase 1: Project Scaffolding

### Task 1: Initialize Go module and project skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/atask/main.go`
- Create: `Makefile`
- Create: `.golangci.yml`
- Create: `Dockerfile`
- Create: `docker-compose.yml`

- [ ] **Step 1: Initialize Go module**

Run: `go mod init github.com/atask/atask`

- [ ] **Step 2: Create directory structure**

Run:
```bash
mkdir -p cmd/atask internal/domain internal/store/migrations internal/store/queries internal/store/sqlc internal/event internal/service internal/api
```

- [ ] **Step 3: Create minimal main.go**

Create `cmd/atask/main.go`:
```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
```

- [ ] **Step 4: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build run test lint fmt vet migrate docker-build docker-up

BINARY=atask
BUILD_DIR=./cmd/atask

build:
	go build -o $(BINARY) $(BUILD_DIR)

run:
	go run $(BUILD_DIR)

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

fmt:
	goimports -w .
	go fmt ./...

vet:
	go vet ./...

migrate:
	goose -dir internal/store/migrations sqlite3 atask.db up

migrate-down:
	goose -dir internal/store/migrations sqlite3 atask.db down

sqlc:
	sqlc generate

docker-build:
	docker build -t atask .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

check: fmt vet lint test
```

- [ ] **Step 5: Create .golangci.yml**

Create `.golangci.yml`:
```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - gocritic
    - gosimple
    - ineffassign
    - unused
    - misspell
    - gofmt
    - goimports

linters-settings:
  gocritic:
    enabled-tags:
      - diagnostic
      - style
      - performance

run:
  timeout: 5m

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

- [ ] **Step 6: Create Dockerfile**

Create `Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o atask ./cmd/atask

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/atask .
COPY --from=builder /app/internal/store/migrations ./migrations

EXPOSE 8080

CMD ["./atask"]
```

- [ ] **Step 7: Create docker-compose.yml**

Create `docker-compose.yml`:
```yaml
services:
  api:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - atask-data:/app/data
    environment:
      - PORT=8080
      - DB_PATH=/app/data/atask.db
    restart: unless-stopped

volumes:
  atask-data:
```

- [ ] **Step 8: Verify build compiles**

Run: `go build ./cmd/atask`
Expected: no errors

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: initialize project skeleton with Go module, Makefile, Docker"
```

---

## Phase 2: Domain Types

### Task 2: Define enums and shared types

**Files:**
- Create: `internal/domain/types.go`

- [ ] **Step 1: Write the test**

Create `internal/domain/types_test.go`:
```go
package domain

import "testing"

func TestStatusString(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusPending, "pending"},
		{StatusCompleted, "completed"},
		{StatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Status.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input string
		want  Status
		err   bool
	}{
		{"pending", StatusPending, false},
		{"completed", StatusCompleted, false},
		{"cancelled", StatusCancelled, false},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseStatus(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ParseStatus(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestScheduleString(t *testing.T) {
	tests := []struct {
		s    Schedule
		want string
	}{
		{ScheduleInbox, "inbox"},
		{ScheduleAnytime, "anytime"},
		{ScheduleSomeday, "someday"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Schedule.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input string
		want  Schedule
		err   bool
	}{
		{"inbox", ScheduleInbox, false},
		{"anytime", ScheduleAnytime, false},
		{"someday", ScheduleSomeday, false},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseSchedule(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ParseSchedule(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSchedule(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/ -run TestStatus -v`
Expected: FAIL — types not defined yet

- [ ] **Step 3: Implement types.go**

Create `internal/domain/types.go`:
```go
package domain

import (
	"fmt"
	"time"
)

// Status represents the lifecycle state of a task or project.
type Status int

const (
	StatusPending   Status = iota // 0
	StatusCompleted               // 1
	StatusCancelled               // 2
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusCompleted:
		return "completed"
	case StatusCancelled:
		return "cancelled"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

func ParseStatus(s string) (Status, error) {
	switch s {
	case "pending":
		return StatusPending, nil
	case "completed":
		return StatusCompleted, nil
	case "cancelled":
		return StatusCancelled, nil
	default:
		return 0, fmt.Errorf("invalid status: %q", s)
	}
}

// Schedule represents the attention bucket for a task.
type Schedule int

const (
	ScheduleInbox   Schedule = iota // 0
	ScheduleAnytime                 // 1
	ScheduleSomeday                 // 2
)

func (s Schedule) String() string {
	switch s {
	case ScheduleInbox:
		return "inbox"
	case ScheduleAnytime:
		return "anytime"
	case ScheduleSomeday:
		return "someday"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

func ParseSchedule(s string) (Schedule, error) {
	switch s {
	case "inbox":
		return ScheduleInbox, nil
	case "anytime":
		return ScheduleAnytime, nil
	case "someday":
		return ScheduleSomeday, nil
	default:
		return 0, fmt.Errorf("invalid schedule: %q", s)
	}
}

// ChecklistStatus represents the state of a checklist item.
type ChecklistStatus int

const (
	ChecklistPending   ChecklistStatus = iota
	ChecklistCompleted
)

func (s ChecklistStatus) String() string {
	switch s {
	case ChecklistPending:
		return "pending"
	case ChecklistCompleted:
		return "completed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// ActorType identifies whether an action was performed by a human or agent.
type ActorType string

const (
	ActorHuman ActorType = "human"
	ActorAgent ActorType = "agent"
)

// ActivityType categorizes the kind of collaboration entry.
type ActivityType string

const (
	ActivityComment        ActivityType = "comment"
	ActivityContextRequest ActivityType = "context_request"
	ActivityReply          ActivityType = "reply"
	ActivityArtifact       ActivityType = "artifact"
	ActivityStatusChange   ActivityType = "status_change"
	ActivityDecomposition  ActivityType = "decomposition"
)

// Timestamps is embedded in entities that track creation/modification times.
type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SoftDelete is embedded in entities that support tombstone deletion.
type SoftDelete struct {
	Deleted   bool
	DeletedAt *time.Time
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/types.go internal/domain/types_test.go
git commit -m "feat: add domain enum types (Status, Schedule, ActorType, ActivityType)"
```

### Task 3: Define Task entity

**Files:**
- Create: `internal/domain/task.go`

- [ ] **Step 1: Write the test**

Create `internal/domain/task_test.go`:
```go
package domain

import "testing"

func TestNewTask(t *testing.T) {
	task, err := NewTask("Buy groceries")
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", task.Title, "Buy groceries")
	}
	if task.Status != StatusPending {
		t.Errorf("Status = %v, want %v", task.Status, StatusPending)
	}
	if task.Schedule != ScheduleInbox {
		t.Errorf("Schedule = %v, want %v", task.Schedule, ScheduleInbox)
	}
	if task.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestNewTask_EmptyTitle(t *testing.T) {
	_, err := NewTask("")
	if err == nil {
		t.Error("NewTask(\"\") should return an error")
	}
}

func TestTask_Validate(t *testing.T) {
	task, _ := NewTask("Test")

	// Valid: standalone task (no project, no area)
	if err := task.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	// Valid: task in a project
	task.ProjectID = ptrStr("proj-1")
	if err := task.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	// Invalid: section without project
	task.ProjectID = nil
	task.SectionID = ptrStr("sec-1")
	if err := task.Validate(); err == nil {
		t.Error("Validate() should fail when SectionID set without ProjectID")
	}
}

func ptrStr(s string) *string { return &s }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/ -run TestNewTask -v`
Expected: FAIL

- [ ] **Step 3: Implement task.go**

Create `internal/domain/task.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID             string
	Title          string
	Notes          string
	Status         Status
	Schedule       Schedule
	StartDate      *time.Time
	Deadline       *time.Time
	CompletedAt    *time.Time
	Index          int
	TodayIndex     *int
	ProjectID      *string
	SectionID      *string
	AreaID         *string
	LocationID     *string
	RecurrenceRule *RecurrenceRule
	Tags           []string

	Timestamps
	SoftDelete
}

func NewTask(title string) (*Task, error) {
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	now := time.Now().UTC()
	return &Task{
		ID:       uuid.New().String(),
		Title:    title,
		Status:   StatusPending,
		Schedule: ScheduleInbox,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func (t *Task) Validate() error {
	if t.Title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if t.SectionID != nil && t.ProjectID == nil {
		return fmt.Errorf("task with a section must also have a project")
	}
	return nil
}
```

- [ ] **Step 4: Add uuid dependency**

Run: `go get github.com/google/uuid`

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/domain/ -run TestNewTask -v && go test ./internal/domain/ -run TestTask_Validate -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/domain/task.go internal/domain/task_test.go go.mod go.sum
git commit -m "feat: add Task entity with validation"
```

### Task 4: Define remaining entities (Project, Area, Section, Tag, ChecklistItem, Location, Activity)

**Files:**
- Create: `internal/domain/project.go`, `internal/domain/area.go`, `internal/domain/section.go`, `internal/domain/tag.go`, `internal/domain/checklist.go`, `internal/domain/location.go`, `internal/domain/activity.go`, `internal/domain/recurrence.go`

- [ ] **Step 1: Write tests for all entities**

Create `internal/domain/project_test.go`:
```go
package domain

import "testing"

func TestNewProject(t *testing.T) {
	p, err := NewProject("Q2 Launch")
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.Title != "Q2 Launch" {
		t.Errorf("Title = %q, want %q", p.Title, "Q2 Launch")
	}
	if p.Status != StatusPending {
		t.Errorf("Status = %v, want %v", p.Status, StatusPending)
	}
}

func TestNewProject_EmptyTitle(t *testing.T) {
	_, err := NewProject("")
	if err == nil {
		t.Error("expected error for empty title")
	}
}
```

Create `internal/domain/area_test.go`:
```go
package domain

import "testing"

func TestNewArea(t *testing.T) {
	a, err := NewArea("Work")
	if err != nil {
		t.Fatalf("NewArea() error = %v", err)
	}
	if a.Title != "Work" {
		t.Errorf("Title = %q, want %q", a.Title, "Work")
	}
	if a.Archived {
		t.Error("new area should not be archived")
	}
}
```

Create `internal/domain/recurrence_test.go`:
```go
package domain

import "testing"

func TestRecurrenceRule_Validate(t *testing.T) {
	tests := []struct {
		name string
		rule RecurrenceRule
		err  bool
	}{
		{"valid fixed daily", RecurrenceRule{Mode: RecurrenceFixed, Interval: 1, Unit: RecurrenceDay}, false},
		{"valid after completion weekly", RecurrenceRule{Mode: RecurrenceAfterCompletion, Interval: 2, Unit: RecurrenceWeek}, false},
		{"zero interval", RecurrenceRule{Mode: RecurrenceFixed, Interval: 0, Unit: RecurrenceDay}, true},
		{"negative interval", RecurrenceRule{Mode: RecurrenceFixed, Interval: -1, Unit: RecurrenceDay}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.err {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/ -v`
Expected: FAIL — entities not yet implemented

- [ ] **Step 3: Implement all entity files**

Create `internal/domain/project.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID           string
	Title        string
	Notes        string
	Status       Status
	Schedule     Schedule
	StartDate    *time.Time
	Deadline     *time.Time
	CompletedAt  *time.Time
	Index        int
	AreaID       *string
	Tags         []string
	AutoComplete bool

	Timestamps
	SoftDelete
}

func NewProject(title string) (*Project, error) {
	if title == "" {
		return nil, fmt.Errorf("project title cannot be empty")
	}
	now := time.Now().UTC()
	return &Project{
		ID:       uuid.New().String(),
		Title:    title,
		Status:   StatusPending,
		Schedule: ScheduleInbox,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func (p *Project) Validate() error {
	if p.Title == "" {
		return fmt.Errorf("project title cannot be empty")
	}
	return nil
}
```

Create `internal/domain/area.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Area struct {
	ID       string
	Title    string
	Index    int
	Archived bool

	Timestamps
	SoftDelete
}

func NewArea(title string) (*Area, error) {
	if title == "" {
		return nil, fmt.Errorf("area title cannot be empty")
	}
	now := time.Now().UTC()
	return &Area{
		ID:    uuid.New().String(),
		Title: title,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
```

Create `internal/domain/section.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Section struct {
	ID        string
	Title     string
	ProjectID string
	Index     int

	Timestamps
	SoftDelete
}

func NewSection(title, projectID string) (*Section, error) {
	if title == "" {
		return nil, fmt.Errorf("section title cannot be empty")
	}
	if projectID == "" {
		return nil, fmt.Errorf("section must belong to a project")
	}
	now := time.Now().UTC()
	return &Section{
		ID:        uuid.New().String(),
		Title:     title,
		ProjectID: projectID,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
```

Create `internal/domain/tag.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID       string
	Title    string
	ParentID *string
	Shortcut *string
	Index    int

	Timestamps
	SoftDelete
}

func NewTag(title string) (*Tag, error) {
	if title == "" {
		return nil, fmt.Errorf("tag title cannot be empty")
	}
	now := time.Now().UTC()
	return &Tag{
		ID:    uuid.New().String(),
		Title: title,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
```

Create `internal/domain/checklist.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ChecklistItem struct {
	ID     string
	Title  string
	Status ChecklistStatus
	TaskID string
	Index  int

	Timestamps
	SoftDelete
}

func NewChecklistItem(title, taskID string) (*ChecklistItem, error) {
	if title == "" {
		return nil, fmt.Errorf("checklist item title cannot be empty")
	}
	if taskID == "" {
		return nil, fmt.Errorf("checklist item must belong to a task")
	}
	now := time.Now().UTC()
	return &ChecklistItem{
		ID:     uuid.New().String(),
		Title:  title,
		Status: ChecklistPending,
		TaskID: taskID,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
```

Create `internal/domain/location.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Location struct {
	ID        string
	Name      string
	Latitude  *float64
	Longitude *float64
	Radius    *int
	Address   *string

	Timestamps
	SoftDelete
}

func NewLocation(name string) (*Location, error) {
	if name == "" {
		return nil, fmt.Errorf("location name cannot be empty")
	}
	now := time.Now().UTC()
	return &Location{
		ID:   uuid.New().String(),
		Name: name,
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
```

Create `internal/domain/activity.go`:
```go
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Activity struct {
	ID        string
	TaskID    string
	ActorID   string
	ActorType ActorType
	Type      ActivityType
	Content   string
	CreatedAt time.Time
}

func NewActivity(taskID, actorID string, actorType ActorType, activityType ActivityType, content string) (*Activity, error) {
	if taskID == "" {
		return nil, fmt.Errorf("activity must belong to a task")
	}
	if actorID == "" {
		return nil, fmt.Errorf("activity must have an actor")
	}
	if content == "" {
		return nil, fmt.Errorf("activity content cannot be empty")
	}
	return &Activity{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		ActorID:   actorID,
		ActorType: actorType,
		Type:      activityType,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}, nil
}
```

Create `internal/domain/recurrence.go`:
```go
package domain

import "fmt"

type RecurrenceMode string

const (
	RecurrenceFixed           RecurrenceMode = "fixed"
	RecurrenceAfterCompletion RecurrenceMode = "after_completion"
)

type RecurrenceUnit string

const (
	RecurrenceDay   RecurrenceUnit = "day"
	RecurrenceWeek  RecurrenceUnit = "week"
	RecurrenceMonth RecurrenceUnit = "month"
)

type RecurrenceEnd struct {
	Date  *string `json:"date,omitempty"`
	Count *int    `json:"count,omitempty"`
}

type RecurrenceRule struct {
	Mode     RecurrenceMode  `json:"mode"`
	Interval int             `json:"interval"`
	Unit     RecurrenceUnit  `json:"unit"`
	End      *RecurrenceEnd  `json:"end,omitempty"`
}

func (r *RecurrenceRule) Validate() error {
	if r.Interval <= 0 {
		return fmt.Errorf("recurrence interval must be positive, got %d", r.Interval)
	}
	switch r.Mode {
	case RecurrenceFixed, RecurrenceAfterCompletion:
	default:
		return fmt.Errorf("invalid recurrence mode: %q", r.Mode)
	}
	switch r.Unit {
	case RecurrenceDay, RecurrenceWeek, RecurrenceMonth:
	default:
		return fmt.Errorf("invalid recurrence unit: %q", r.Unit)
	}
	return nil
}
```

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/domain/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/
git commit -m "feat: add all domain entities (Project, Area, Section, Tag, ChecklistItem, Location, Activity, Recurrence)"
```

### Task 5: Define domain event types and catalog

**Files:**
- Create: `internal/domain/event.go`

- [ ] **Step 1: Write the test**

Create `internal/domain/event_test.go`:
```go
package domain

import "testing"

func TestNewDomainEvent(t *testing.T) {
	e := NewDomainEvent(EventTaskCompleted, "task", "task-123", "user-1", map[string]any{
		"title": "Buy groceries",
	})
	if e.Type != EventTaskCompleted {
		t.Errorf("Type = %q, want %q", e.Type, EventTaskCompleted)
	}
	if e.EntityID != "task-123" {
		t.Errorf("EntityID = %q, want %q", e.EntityID, "task-123")
	}
}

func TestDeltaAction_String(t *testing.T) {
	tests := []struct {
		a    DeltaAction
		want string
	}{
		{DeltaCreated, "created"},
		{DeltaModified, "modified"},
		{DeltaDeleted, "deleted"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("DeltaAction.String() = %q, want %q", got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/ -run TestNewDomainEvent -v`
Expected: FAIL

- [ ] **Step 3: Implement event.go**

Create `internal/domain/event.go`:
```go
package domain

import (
	"encoding/json"
	"time"
)

// DeltaAction represents what happened to an entity.
type DeltaAction int

const (
	DeltaCreated  DeltaAction = iota
	DeltaModified
	DeltaDeleted
)

func (a DeltaAction) String() string {
	switch a {
	case DeltaCreated:
		return "created"
	case DeltaModified:
		return "modified"
	case DeltaDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// DeltaEvent is an immutable record of a field-level change for sync.
type DeltaEvent struct {
	ID         int64
	EntityType string
	EntityID   string
	Action     DeltaAction
	Field      *string
	OldValue   json.RawMessage
	NewValue   json.RawMessage
	ActorID    string
	Timestamp  time.Time
}

// EventType is a semantic domain event identifier.
type EventType string

// Task events
const (
	EventTaskCreated          EventType = "task.created"
	EventTaskDeleted          EventType = "task.deleted"
	EventTaskCompleted        EventType = "task.completed"
	EventTaskCancelled        EventType = "task.cancelled"
	EventTaskTitleChanged     EventType = "task.title_changed"
	EventTaskNotesChanged     EventType = "task.notes_changed"
	EventTaskScheduledToday   EventType = "task.scheduled_today"
	EventTaskDeferred         EventType = "task.deferred"
	EventTaskMovedToInbox     EventType = "task.moved_to_inbox"
	EventTaskStartDateSet     EventType = "task.start_date_set"
	EventTaskDeadlineSet      EventType = "task.deadline_set"
	EventTaskDeadlineRemoved  EventType = "task.deadline_removed"
	EventTaskMovedToProject   EventType = "task.moved_to_project"
	EventTaskRemovedFromProject EventType = "task.removed_from_project"
	EventTaskMovedToSection   EventType = "task.moved_to_section"
	EventTaskRemovedFromSection EventType = "task.removed_from_section"
	EventTaskMovedToArea      EventType = "task.moved_to_area"
	EventTaskRemovedFromArea  EventType = "task.removed_from_area"
	EventTaskTagAdded         EventType = "task.tag_added"
	EventTaskTagRemoved       EventType = "task.tag_removed"
	EventTaskLocationSet      EventType = "task.location_set"
	EventTaskLocationRemoved  EventType = "task.location_removed"
	EventTaskLinkAdded        EventType = "task.link_added"
	EventTaskLinkRemoved      EventType = "task.link_removed"
	EventTaskRecurrenceSet    EventType = "task.recurrence_set"
	EventTaskRecurrenceRemoved EventType = "task.recurrence_removed"
	EventTaskReordered        EventType = "task.reordered"
)

// Project events
const (
	EventProjectCreated          EventType = "project.created"
	EventProjectDeleted          EventType = "project.deleted"
	EventProjectCompleted        EventType = "project.completed"
	EventProjectCancelled        EventType = "project.cancelled"
	EventProjectTitleChanged     EventType = "project.title_changed"
	EventProjectNotesChanged     EventType = "project.notes_changed"
	EventProjectTagAdded         EventType = "project.tag_added"
	EventProjectTagRemoved       EventType = "project.tag_removed"
	EventProjectMovedToArea      EventType = "project.moved_to_area"
	EventProjectRemovedFromArea  EventType = "project.removed_from_area"
	EventProjectDeadlineSet      EventType = "project.deadline_set"
	EventProjectDeadlineRemoved  EventType = "project.deadline_removed"
)

// Checklist events
const (
	EventChecklistItemAdded       EventType = "checklist.item_added"
	EventChecklistItemRemoved     EventType = "checklist.item_removed"
	EventChecklistItemCompleted   EventType = "checklist.item_completed"
	EventChecklistItemUncompleted EventType = "checklist.item_uncompleted"
	EventChecklistItemTitleChanged EventType = "checklist.item_title_changed"
)

// Activity events
const (
	EventActivityAdded EventType = "activity.added"
)

// Section events
const (
	EventSectionCreated EventType = "section.created"
	EventSectionDeleted EventType = "section.deleted"
	EventSectionRenamed EventType = "section.renamed"
)

// Area events
const (
	EventAreaCreated    EventType = "area.created"
	EventAreaDeleted    EventType = "area.deleted"
	EventAreaRenamed    EventType = "area.renamed"
	EventAreaArchived   EventType = "area.archived"
	EventAreaUnarchived EventType = "area.unarchived"
)

// Tag events
const (
	EventTagCreated         EventType = "tag.created"
	EventTagDeleted         EventType = "tag.deleted"
	EventTagRenamed         EventType = "tag.renamed"
	EventTagShortcutChanged EventType = "tag.shortcut_changed"
)

// Location events
const (
	EventLocationCreated EventType = "location.created"
	EventLocationDeleted EventType = "location.deleted"
	EventLocationRenamed EventType = "location.renamed"
)

// DomainEvent is a semantic event for agent intelligence.
type DomainEvent struct {
	ID         int64
	Type       EventType
	EntityType string
	EntityID   string
	ActorID    string
	Payload    map[string]any
	Timestamp  time.Time
}

func NewDomainEvent(eventType EventType, entityType, entityID, actorID string, payload map[string]any) *DomainEvent {
	return &DomainEvent{
		Type:       eventType,
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
	}
}
```

- [ ] **Step 4: Run all domain tests**

Run: `go test ./internal/domain/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/event.go internal/domain/event_test.go
git commit -m "feat: add domain event types and full event catalog"
```

---

## Phase 3: Database Schema and Persistence

### Task 6: Create SQLite migration

**Files:**
- Create: `internal/store/migrations/001_initial_schema.sql`

- [ ] **Step 1: Write the migration**

Create `internal/store/migrations/001_initial_schema.sql`:
```sql
-- +goose Up

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    permissions TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

CREATE TABLE areas (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    "index" INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    parent_id TEXT REFERENCES tags(id),
    shortcut TEXT,
    "index" INTEGER NOT NULL DEFAULT 0,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE locations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    latitude REAL,
    longitude REAL,
    radius INTEGER,
    address TEXT,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    schedule INTEGER NOT NULL DEFAULT 0,
    start_date TEXT,
    deadline TEXT,
    completed_at DATETIME,
    "index" INTEGER NOT NULL DEFAULT 0,
    area_id TEXT REFERENCES areas(id),
    auto_complete INTEGER NOT NULL DEFAULT 0,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sections (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    schedule INTEGER NOT NULL DEFAULT 0,
    start_date TEXT,
    deadline TEXT,
    completed_at DATETIME,
    "index" INTEGER NOT NULL DEFAULT 0,
    today_index INTEGER,
    project_id TEXT REFERENCES projects(id),
    section_id TEXT REFERENCES sections(id),
    area_id TEXT REFERENCES areas(id),
    location_id TEXT REFERENCES locations(id),
    recurrence_rule TEXT,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE checklist_items (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    deleted INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_tags (
    task_id TEXT NOT NULL REFERENCES tasks(id),
    tag_id TEXT NOT NULL REFERENCES tags(id),
    PRIMARY KEY (task_id, tag_id)
);

CREATE TABLE project_tags (
    project_id TEXT NOT NULL REFERENCES projects(id),
    tag_id TEXT NOT NULL REFERENCES tags(id),
    PRIMARY KEY (project_id, tag_id)
);

CREATE TABLE task_links (
    task_id TEXT NOT NULL REFERENCES tasks(id),
    related_task_id TEXT NOT NULL REFERENCES tasks(id),
    relationship_type TEXT NOT NULL DEFAULT 'related',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, related_task_id)
);

CREATE TABLE activities (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE delta_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    action INTEGER NOT NULL,
    field TEXT,
    old_value TEXT,
    new_value TEXT,
    actor_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE domain_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}',
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_tasks_project_id ON tasks(project_id) WHERE deleted = 0;
CREATE INDEX idx_tasks_section_id ON tasks(section_id) WHERE deleted = 0;
CREATE INDEX idx_tasks_area_id ON tasks(area_id) WHERE deleted = 0;
CREATE INDEX idx_tasks_location_id ON tasks(location_id) WHERE deleted = 0;
CREATE INDEX idx_tasks_schedule ON tasks(schedule) WHERE deleted = 0;
CREATE INDEX idx_tasks_status ON tasks(status) WHERE deleted = 0;
CREATE INDEX idx_tasks_start_date ON tasks(start_date) WHERE deleted = 0;
CREATE INDEX idx_tasks_deadline ON tasks(deadline) WHERE deleted = 0;

CREATE INDEX idx_sections_project_id ON sections(project_id) WHERE deleted = 0;
CREATE INDEX idx_projects_area_id ON projects(area_id) WHERE deleted = 0;
CREATE INDEX idx_checklist_items_task_id ON checklist_items(task_id) WHERE deleted = 0;
CREATE INDEX idx_activities_task_id ON activities(task_id);

CREATE INDEX idx_delta_events_entity ON delta_events(entity_type, entity_id);
CREATE INDEX idx_domain_events_type ON domain_events(type);
CREATE INDEX idx_domain_events_entity ON domain_events(entity_type, entity_id);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- +goose Down
DROP TABLE IF EXISTS domain_events;
DROP TABLE IF EXISTS delta_events;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS task_links;
DROP TABLE IF EXISTS project_tags;
DROP TABLE IF EXISTS task_tags;
DROP TABLE IF EXISTS checklist_items;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS sections;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS areas;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 2: Install goose and run migration**

Run:
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir internal/store/migrations sqlite3 atask.db up
```
Expected: migration applies successfully

- [ ] **Step 3: Verify schema**

Run: `sqlite3 atask.db ".tables"`
Expected: all tables listed

- [ ] **Step 4: Commit**

```bash
git add internal/store/migrations/001_initial_schema.sql
git commit -m "feat: add initial SQLite schema migration"
```

### Task 7: Configure sqlc and write core queries

**Files:**
- Create: `sqlc.yaml`
- Create: `internal/store/queries/tasks.sql`
- Create: `internal/store/queries/areas.sql`
- Create: `internal/store/queries/projects.sql`
- Create: `internal/store/queries/events.sql`
- Create: `internal/store/queries/views.sql`

- [ ] **Step 1: Create sqlc.yaml**

Create `sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "internal/store/queries"
    schema: "internal/store/migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/store/sqlc"
        emit_json_tags: true
        emit_empty_slices: true
```

- [ ] **Step 2: Write task queries**

Create `internal/store/queries/tasks.sql`:
```sql
-- name: CreateTask :exec
INSERT INTO tasks (id, title, notes, status, schedule, start_date, deadline, "index", today_index, project_id, section_id, area_id, location_id, recurrence_rule, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? AND deleted = 0;

-- name: ListTasks :many
SELECT * FROM tasks WHERE deleted = 0 ORDER BY "index";

-- name: ListTasksByProject :many
SELECT * FROM tasks WHERE project_id = ? AND deleted = 0 ORDER BY "index";

-- name: ListTasksByArea :many
SELECT * FROM tasks WHERE area_id = ? AND project_id IS NULL AND deleted = 0 ORDER BY "index";

-- name: ListTasksBySection :many
SELECT * FROM tasks WHERE section_id = ? AND deleted = 0 ORDER BY "index";

-- name: ListTasksByLocation :many
SELECT * FROM tasks WHERE location_id = ? AND deleted = 0 ORDER BY "index";

-- name: ListTasksBySchedule :many
SELECT * FROM tasks WHERE schedule = ? AND deleted = 0 ORDER BY "index";

-- name: UpdateTaskTitle :exec
UPDATE tasks SET title = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskNotes :exec
UPDATE tasks SET notes = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskStatus :exec
UPDATE tasks SET status = ?, completed_at = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskSchedule :exec
UPDATE tasks SET schedule = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskStartDate :exec
UPDATE tasks SET start_date = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskDeadline :exec
UPDATE tasks SET deadline = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskProject :exec
UPDATE tasks SET project_id = ?, section_id = NULL, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskSection :exec
UPDATE tasks SET section_id = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskArea :exec
UPDATE tasks SET area_id = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskLocation :exec
UPDATE tasks SET location_id = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskRecurrence :exec
UPDATE tasks SET recurrence_rule = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskIndex :exec
UPDATE tasks SET "index" = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTaskTodayIndex :exec
UPDATE tasks SET today_index = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteTask :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;

-- name: SoftDeleteTasksByProject :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ? WHERE project_id = ? AND deleted = 0;

-- name: OrphanTasksByArea :exec
UPDATE tasks SET area_id = NULL, updated_at = ? WHERE area_id = ? AND deleted = 0;

-- name: OrphanTasksBySection :exec
UPDATE tasks SET section_id = NULL, updated_at = ? WHERE section_id = ? AND deleted = 0;

-- name: CascadeDeleteTasksByArea :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ? WHERE area_id = ? AND deleted = 0;

-- name: CascadeDeleteTasksBySection :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ? WHERE section_id = ? AND deleted = 0;

-- name: CompleteTasksByProject :exec
UPDATE tasks SET status = 1, completed_at = ?, updated_at = ? WHERE project_id = ? AND status = 0 AND deleted = 0;

-- name: CancelTasksByProject :exec
UPDATE tasks SET status = 2, completed_at = ?, updated_at = ? WHERE project_id = ? AND status = 0 AND deleted = 0;
```

- [ ] **Step 3: Write remaining query files**

Create `internal/store/queries/areas.sql`:
```sql
-- name: CreateArea :exec
INSERT INTO areas (id, title, "index", created_at, updated_at) VALUES (?, ?, ?, ?, ?);

-- name: GetArea :one
SELECT * FROM areas WHERE id = ? AND deleted = 0;

-- name: ListAreas :many
SELECT * FROM areas WHERE deleted = 0 AND archived = 0 ORDER BY "index";

-- name: ListAllAreas :many
SELECT * FROM areas WHERE deleted = 0 ORDER BY "index";

-- name: UpdateAreaTitle :exec
UPDATE areas SET title = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateAreaArchived :exec
UPDATE areas SET archived = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteArea :exec
UPDATE areas SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;
```

Create `internal/store/queries/projects.sql`:
```sql
-- name: CreateProject :exec
INSERT INTO projects (id, title, notes, status, schedule, start_date, deadline, "index", area_id, auto_complete, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProject :one
SELECT * FROM projects WHERE id = ? AND deleted = 0;

-- name: ListProjects :many
SELECT * FROM projects WHERE deleted = 0 ORDER BY "index";

-- name: ListProjectsByArea :many
SELECT * FROM projects WHERE area_id = ? AND deleted = 0 ORDER BY "index";

-- name: UpdateProjectTitle :exec
UPDATE projects SET title = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateProjectNotes :exec
UPDATE projects SET notes = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateProjectStatus :exec
UPDATE projects SET status = ?, completed_at = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateProjectDeadline :exec
UPDATE projects SET deadline = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateProjectArea :exec
UPDATE projects SET area_id = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteProject :exec
UPDATE projects SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;

-- name: OrphanProjectsByArea :exec
UPDATE projects SET area_id = NULL, updated_at = ? WHERE area_id = ? AND deleted = 0;

-- name: CascadeDeleteProjectsByArea :exec
UPDATE projects SET deleted = 1, deleted_at = ?, updated_at = ? WHERE area_id = ? AND deleted = 0;
```

Create `internal/store/queries/events.sql`:
```sql
-- name: InsertDeltaEvent :exec
INSERT INTO delta_events (entity_type, entity_id, action, field, old_value, new_value, actor_id, timestamp)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListDeltaEventsSince :many
SELECT * FROM delta_events WHERE id > ? ORDER BY id;

-- name: InsertDomainEvent :one
INSERT INTO domain_events (type, entity_type, entity_id, actor_id, payload, timestamp)
VALUES (?, ?, ?, ?, ?, ?) RETURNING id;

-- name: ListDomainEventsSince :many
SELECT * FROM domain_events WHERE id > ? ORDER BY id;

-- name: ListDomainEventsByTypeSince :many
SELECT * FROM domain_events WHERE type = ? AND id > ? ORDER BY id;

-- name: ListDomainEventsByEntitySince :many
SELECT * FROM domain_events WHERE entity_type = ? AND entity_id = ? AND id > ? ORDER BY id;
```

Create `internal/store/queries/views.sql`:
```sql
-- name: ViewInbox :many
SELECT * FROM tasks WHERE schedule = 0 AND status = 0 AND deleted = 0 ORDER BY "index";

-- name: ViewToday :many
SELECT * FROM tasks
WHERE schedule = 1 AND status = 0 AND deleted = 0
AND (start_date IS NULL OR start_date <= ?)
ORDER BY COALESCE(today_index, 999999), "index";

-- name: ViewUpcoming :many
SELECT * FROM tasks
WHERE start_date > ? AND status = 0 AND deleted = 0
ORDER BY start_date, "index";

-- name: ViewSomeday :many
SELECT * FROM tasks WHERE schedule = 2 AND status = 0 AND deleted = 0 ORDER BY "index";

-- name: ViewLogbook :many
SELECT * FROM tasks WHERE status IN (1, 2) AND deleted = 0 ORDER BY completed_at DESC;
```

Create `internal/store/queries/sections.sql`:
```sql
-- name: CreateSection :exec
INSERT INTO sections (id, title, project_id, "index", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetSection :one
SELECT * FROM sections WHERE id = ? AND deleted = 0;

-- name: ListSectionsByProject :many
SELECT * FROM sections WHERE project_id = ? AND deleted = 0 ORDER BY "index";

-- name: UpdateSectionTitle :exec
UPDATE sections SET title = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteSection :exec
UPDATE sections SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;

-- name: SoftDeleteSectionsByProject :exec
UPDATE sections SET deleted = 1, deleted_at = ?, updated_at = ? WHERE project_id = ? AND deleted = 0;
```

Create `internal/store/queries/tags.sql`:
```sql
-- name: CreateTag :exec
INSERT INTO tags (id, title, parent_id, shortcut, "index", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetTag :one
SELECT * FROM tags WHERE id = ? AND deleted = 0;

-- name: ListTags :many
SELECT * FROM tags WHERE deleted = 0 ORDER BY "index";

-- name: UpdateTagTitle :exec
UPDATE tags SET title = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateTagShortcut :exec
UPDATE tags SET shortcut = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteTag :exec
UPDATE tags SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;
```

Create `internal/store/queries/checklist_items.sql`:
```sql
-- name: CreateChecklistItem :exec
INSERT INTO checklist_items (id, title, status, task_id, "index", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetChecklistItem :one
SELECT * FROM checklist_items WHERE id = ? AND deleted = 0;

-- name: ListChecklistItemsByTask :many
SELECT * FROM checklist_items WHERE task_id = ? AND deleted = 0 ORDER BY "index";

-- name: UpdateChecklistItemTitle :exec
UPDATE checklist_items SET title = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: UpdateChecklistItemStatus :exec
UPDATE checklist_items SET status = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteChecklistItem :exec
UPDATE checklist_items SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;
```

Create `internal/store/queries/locations.sql`:
```sql
-- name: CreateLocation :exec
INSERT INTO locations (id, name, latitude, longitude, radius, address, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetLocation :one
SELECT * FROM locations WHERE id = ? AND deleted = 0;

-- name: ListLocations :many
SELECT * FROM locations WHERE deleted = 0 ORDER BY name;

-- name: UpdateLocationName :exec
UPDATE locations SET name = ?, updated_at = ? WHERE id = ? AND deleted = 0;

-- name: SoftDeleteLocation :exec
UPDATE locations SET deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?;

-- name: ClearLocationFromTasks :exec
UPDATE tasks SET location_id = NULL, updated_at = ? WHERE location_id = ? AND deleted = 0;
```

Create `internal/store/queries/activities.sql`:
```sql
-- name: CreateActivity :exec
INSERT INTO activities (id, task_id, actor_id, actor_type, type, content, created_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListActivitiesByTask :many
SELECT * FROM activities WHERE task_id = ? ORDER BY created_at;
```

Create `internal/store/queries/task_tags.sql`:
```sql
-- name: AddTaskTag :exec
INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?);

-- name: RemoveTaskTag :exec
DELETE FROM task_tags WHERE task_id = ? AND tag_id = ?;

-- name: ListTaskTags :many
SELECT tag_id FROM task_tags WHERE task_id = ?;

-- name: RemoveAllTagReferences :exec
DELETE FROM task_tags WHERE tag_id = ?;

-- name: AddProjectTag :exec
INSERT OR IGNORE INTO project_tags (project_id, tag_id) VALUES (?, ?);

-- name: RemoveProjectTag :exec
DELETE FROM project_tags WHERE project_id = ? AND tag_id = ?;

-- name: ListProjectTags :many
SELECT tag_id FROM project_tags WHERE project_id = ?;

-- name: RemoveAllProjectTagReferences :exec
DELETE FROM project_tags WHERE tag_id = ?;
```

Create `internal/store/queries/task_links.sql`:
```sql
-- name: AddTaskLink :exec
INSERT OR IGNORE INTO task_links (task_id, related_task_id, relationship_type, created_at) VALUES (?, ?, ?, ?);

-- name: RemoveTaskLink :exec
DELETE FROM task_links WHERE task_id = ? AND related_task_id = ?;

-- name: ListTaskLinks :many
SELECT * FROM task_links WHERE task_id = ? OR related_task_id = ?;
```

Create `internal/store/queries/auth.sql`:
```sql
-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: UpdateUser :exec
UPDATE users SET name = ?, updated_at = ? WHERE id = ?;

-- name: CreateAPIKey :exec
INSERT INTO api_keys (id, user_id, name, key_hash, permissions, created_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = ?;

-- name: ListAPIKeysByUser :many
SELECT id, user_id, name, permissions, created_at, last_used_at FROM api_keys WHERE user_id = ?;

-- name: UpdateAPIKeyName :exec
UPDATE api_keys SET name = ? WHERE id = ? AND user_id = ?;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = ? WHERE id = ?;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = ? AND user_id = ?;
```

- [ ] **Step 4: Install sqlc and generate code**

Run:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate
```
Expected: Go files generated in `internal/store/sqlc/`

- [ ] **Step 5: Add dependencies and verify build**

Run:
```bash
go get modernc.org/sqlite
go build ./...
```
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add sqlc.yaml internal/store/
git commit -m "feat: add sqlc queries and generate store layer"
```

### Task 8: Create database connection helper

**Files:**
- Create: `internal/store/db.go`

- [ ] **Step 1: Write the test**

Create `internal/store/db_test.go`:
```go
package store

import (
	"context"
	"testing"
)

func TestNewDB_InMemory(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestNewDB_RunMigrations(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Verify tables exist
	var count int
	err = db.DB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&count)
	if err != nil {
		t.Fatalf("query error = %v", err)
	}
	if count != 1 {
		t.Errorf("tasks table not found")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -v`
Expected: FAIL

- [ ] **Step 3: Implement db.go**

Create `internal/store/db.go`:
```go
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	DB *sql.DB
}

func NewDB(path string) (*DB, error) {
	dsn := path
	if path != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path)
	} else {
		dsn = ":memory:?_foreign_keys=on"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite is single-writer

	return &DB{DB: db}, nil
}

func (d *DB) Migrate() error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(d.DB, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (d *DB) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

func (d *DB) Close() error {
	return d.DB.Close()
}
```

- [ ] **Step 4: Add goose dependency**

Run: `go get github.com/pressly/goose/v3`

- [ ] **Step 5: Run tests**

Run: `go test ./internal/store/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/db.go internal/store/db_test.go go.mod go.sum
git commit -m "feat: add database connection with embedded migrations"
```

---

## Phase 4: Event Store and Bus

### Task 9: Implement delta event store

**Files:**
- Create: `internal/event/store.go`

- [ ] **Step 1: Write the test**

Create `internal/event/store_test.go`:
```go
package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/store"
)

func setupTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEventStore_AppendAndQuery(t *testing.T) {
	db := setupTestDB(t)
	es := NewEventStore(db)
	ctx := context.Background()

	// Append a delta event
	err := es.AppendDelta(ctx, domain.DeltaEvent{
		EntityType: "task",
		EntityID:   "task-1",
		Action:     domain.DeltaCreated,
		ActorID:    "user-1",
	})
	if err != nil {
		t.Fatalf("AppendDelta: %v", err)
	}

	// Query since cursor 0
	events, err := es.DeltasSince(ctx, 0)
	if err != nil {
		t.Fatalf("DeltasSince: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].EntityID != "task-1" {
		t.Errorf("EntityID = %q, want %q", events[0].EntityID, "task-1")
	}
}

func TestEventStore_AppendDomainEvent(t *testing.T) {
	db := setupTestDB(t)
	es := NewEventStore(db)
	ctx := context.Background()

	payload := map[string]any{"title": "Buy groceries"}
	payloadJSON, _ := json.Marshal(payload)

	id, err := es.AppendDomainEvent(ctx, domain.EventTaskCreated, "task", "task-1", "user-1", payloadJSON)
	if err != nil {
		t.Fatalf("AppendDomainEvent: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero event ID")
	}

	events, err := es.DomainEventsSince(ctx, 0)
	if err != nil {
		t.Fatalf("DomainEventsSince: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != string(domain.EventTaskCreated) {
		t.Errorf("Type = %q, want %q", events[0].Type, domain.EventTaskCreated)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/event/ -v`
Expected: FAIL

- [ ] **Step 3: Implement store.go**

Create `internal/event/store.go`:
```go
package event

import (
	"context"
	"fmt"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/store"
	"github.com/atask/atask/internal/store/sqlc"
)

type EventStore struct {
	queries *sqlc.Queries
}

func NewEventStore(db *store.DB) *EventStore {
	return &EventStore{
		queries: sqlc.New(db.DB),
	}
}

func (s *EventStore) AppendDelta(ctx context.Context, e domain.DeltaEvent) error {
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return s.queries.InsertDeltaEvent(ctx, sqlc.InsertDeltaEventParams{
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Action:     int64(e.Action),
		Field:      ptrToNullStr(e.Field),
		OldValue:   rawToNullStr(e.OldValue),
		NewValue:   rawToNullStr(e.NewValue),
		ActorID:    e.ActorID,
		Timestamp:  ts,
	})
}

func (s *EventStore) DeltasSince(ctx context.Context, cursor int64) ([]sqlc.DeltaEvent, error) {
	return s.queries.ListDeltaEventsSince(ctx, cursor)
}

func (s *EventStore) AppendDomainEvent(ctx context.Context, eventType domain.EventType, entityType, entityID, actorID string, payload []byte) (int64, error) {
	if payload == nil {
		payload = []byte("{}")
	}
	return s.queries.InsertDomainEvent(ctx, sqlc.InsertDomainEventParams{
		Type:       string(eventType),
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actorID,
		Payload:    string(payload),
		Timestamp:  time.Now().UTC(),
	})
}

func (s *EventStore) DomainEventsSince(ctx context.Context, cursor int64) ([]sqlc.DomainEvent, error) {
	return s.queries.ListDomainEventsSince(ctx, cursor)
}

func ptrToNullStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func rawToNullStr(b json.RawMessage) sql.NullString {
	if b == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}
```

Note: the exact `sqlc.InsertDomainEventParams` and `sqlc.InsertDeltaEventParams` types depend on what sqlc generates. The implementer should adapt the field names to match the generated code (e.g., `sql.NullString` vs `*string` depending on sqlc's output mode). Check `internal/store/sqlc/` after generation. This file needs `"database/sql"` and `"encoding/json"` imports — add them.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/event/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/event/store.go internal/event/store_test.go
git commit -m "feat: add event store for delta and domain events"
```

### Task 10: Implement domain event bus

**Files:**
- Create: `internal/event/bus.go`

- [ ] **Step 1: Write the test**

Create `internal/event/bus_test.go`:
```go
package event

import (
	"sync"
	"testing"
	"time"

	"github.com/atask/atask/internal/domain"
)

func TestBus_SubscribeAndPublish(t *testing.T) {
	bus := NewBus()

	var received *domain.DomainEvent
	var mu sync.Mutex

	bus.Subscribe("task.*", func(e *domain.DomainEvent) {
		mu.Lock()
		received = e
		mu.Unlock()
	})

	event := domain.NewDomainEvent(domain.EventTaskCompleted, "task", "task-1", "user-1", nil)
	bus.Publish(event)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("subscriber did not receive event")
	}
	if received.Type != domain.EventTaskCompleted {
		t.Errorf("Type = %q, want %q", received.Type, domain.EventTaskCompleted)
	}
}

func TestBus_WildcardMatching(t *testing.T) {
	bus := NewBus()

	count := 0
	var mu sync.Mutex

	bus.Subscribe("task.*", func(e *domain.DomainEvent) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	bus.Publish(domain.NewDomainEvent(domain.EventTaskCreated, "task", "t1", "u1", nil))
	bus.Publish(domain.NewDomainEvent(domain.EventProjectCreated, "project", "p1", "u1", nil))
	bus.Publish(domain.NewDomainEvent(domain.EventTaskCompleted, "task", "t2", "u1", nil))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 2 {
		t.Errorf("got %d events, want 2 (only task.*)", count)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/event/ -run TestBus -v`
Expected: FAIL

- [ ] **Step 3: Implement bus.go**

Create `internal/event/bus.go`:
```go
package event

import (
	"strings"
	"sync"

	"github.com/atask/atask/internal/domain"
)

type Handler func(e *domain.DomainEvent)

type subscription struct {
	id      int
	pattern string
	handler Handler
}

type Bus struct {
	mu          sync.RWMutex
	subscribers []subscription
	nextID      int
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) Subscribe(pattern string, handler Handler) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	b.subscribers = append(b.subscribers, subscription{id: b.nextID, pattern: pattern, handler: handler})
	return b.nextID
}

func (b *Bus) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, sub := range b.subscribers {
		if sub.id == id {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			return
		}
	}
}

func (b *Bus) Publish(event *domain.DomainEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscribers {
		if matchPattern(sub.pattern, string(event.Type)) {
			go sub.handler(event)
		}
	}
}

func matchPattern(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(eventType, prefix+".")
	}
	return pattern == eventType
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/event/ -run TestBus -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/event/bus.go internal/event/bus_test.go
git commit -m "feat: add in-process domain event bus with wildcard topic matching"
```

### Task 11: Implement SSE stream manager

**Files:**
- Create: `internal/event/stream.go`

- [ ] **Step 1: Write the test**

Create `internal/event/stream_test.go`:
```go
package event

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/atask/atask/internal/domain"
)

func TestStreamManager_SSE(t *testing.T) {
	bus := NewBus()
	sm := NewStreamManager(bus)

	req := httptest.NewRequest("GET", "/events/stream?topics=task.*", nil)
	rec := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(req.Context(), 200*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	go func() {
		time.Sleep(50 * time.Millisecond)
		event := domain.NewDomainEvent(domain.EventTaskCompleted, "task", "task-1", "user-1", map[string]any{"title": "Test"})
		event.ID = 42
		bus.Publish(event)
	}()

	sm.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "event: task.completed") {
		t.Errorf("response body should contain SSE event, got: %s", body)
	}
	if !strings.Contains(body, "id: 42") {
		t.Errorf("response body should contain event ID, got: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/event/ -run TestStreamManager -v`
Expected: FAIL

- [ ] **Step 3: Implement stream.go**

Create `internal/event/stream.go`:
```go
package event

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/atask/atask/internal/domain"
)

type StreamManager struct {
	bus *Bus
}

func NewStreamManager(bus *Bus) *StreamManager {
	return &StreamManager{bus: bus}
}

func (sm *StreamManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	topicsParam := r.URL.Query().Get("topics")
	if topicsParam == "" {
		topicsParam = "*"
	}
	topics := strings.Split(topicsParam, ",")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events := make(chan *domain.DomainEvent, 64)

	// Subscribe to requested topics. Track subscription IDs for cleanup.
	var subIDs []int
	for _, topic := range topics {
		topic := strings.TrimSpace(topic)
		id := sm.bus.Subscribe(topic, func(e *domain.DomainEvent) {
			select {
			case events <- e:
			default:
				slog.Warn("SSE client buffer full, dropping event", "type", e.Type)
			}
		})
		subIDs = append(subIDs, id)
	}
	defer func() {
		for _, id := range subIDs {
			sm.bus.Unsubscribe(id)
		}
	}()

	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-events:
			payload, err := json.Marshal(event.Payload)
			if err != nil {
				slog.Error("marshal SSE payload", "error", err)
				continue
			}
			fmt.Fprintf(w, "event: %s\n", event.Type)
			fmt.Fprintf(w, "data: %s\n", payload)
			fmt.Fprintf(w, "id: %d\n\n", event.ID)
			flusher.Flush()
		}
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/event/ -run TestStreamManager -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/event/stream.go internal/event/stream_test.go
git commit -m "feat: add SSE stream manager with topic-based subscriptions"
```

---

## Phase 5: Service Layer (Core Operations)

### Task 12: Implement Area service

**Files:**
- Create: `internal/service/area_service.go`

This is the simplest entity — use it to establish the service layer pattern that all others follow: validate → persist → emit delta events → emit domain event → return.

- [ ] **Step 1: Write the test**

Create `internal/service/area_service_test.go`:
```go
package service

import (
	"context"
	"testing"

	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
)

func setupTest(t *testing.T) (*store.DB, *event.EventStore, *event.Bus) {
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
	return db, es, bus
}

func TestAreaService_Create(t *testing.T) {
	db, es, bus := setupTest(t)
	svc := NewAreaService(db, es, bus)
	ctx := context.Background()

	area, err := svc.Create(ctx, "Work", "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if area.Title != "Work" {
		t.Errorf("Title = %q, want %q", area.Title, "Work")
	}

	// Verify it was persisted
	got, err := svc.Get(ctx, area.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Work" {
		t.Errorf("persisted Title = %q, want %q", got.Title, "Work")
	}
}

func TestAreaService_Create_EmptyTitle(t *testing.T) {
	db, es, bus := setupTest(t)
	svc := NewAreaService(db, es, bus)

	_, err := svc.Create(context.Background(), "", "user-1")
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestAreaService_Archive(t *testing.T) {
	db, es, bus := setupTest(t)
	svc := NewAreaService(db, es, bus)
	ctx := context.Background()

	area, _ := svc.Create(ctx, "Old Work", "user-1")
	if err := svc.Archive(ctx, area.ID, "user-1"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	got, err := svc.Get(ctx, area.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Archived {
		t.Error("expected area to be archived")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -v`
Expected: FAIL

- [ ] **Step 3: Implement area_service.go**

Create `internal/service/area_service.go`:
```go
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/store"
	"github.com/atask/atask/internal/store/sqlc"
)

type AreaService struct {
	queries *sqlc.Queries
	events  *event.EventStore
	bus     *event.Bus
}

func NewAreaService(db *store.DB, es *event.EventStore, bus *event.Bus) *AreaService {
	return &AreaService{
		queries: sqlc.New(db.DB),
		events:  es,
		bus:     bus,
	}
}

func (s *AreaService) Create(ctx context.Context, title, actorID string) (*domain.Area, error) {
	area, err := domain.NewArea(title)
	if err != nil {
		return nil, err
	}

	if err := s.queries.CreateArea(ctx, sqlc.CreateAreaParams{
		ID:        area.ID,
		Title:     area.Title,
		Index:     int64(area.Index),
		CreatedAt: area.CreatedAt,
		UpdatedAt: area.UpdatedAt,
	}); err != nil {
		return nil, fmt.Errorf("persist area: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{"title": area.Title})
	s.events.AppendDomainEvent(ctx, domain.EventAreaCreated, "area", area.ID, actorID, payload)
	s.bus.Publish(domain.NewDomainEvent(domain.EventAreaCreated, "area", area.ID, actorID, map[string]any{"title": area.Title}))

	return area, nil
}

func (s *AreaService) Get(ctx context.Context, id string) (*domain.Area, error) {
	row, err := s.queries.GetArea(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get area: %w", err)
	}
	return areaFromRow(row), nil
}

func (s *AreaService) List(ctx context.Context) ([]*domain.Area, error) {
	rows, err := s.queries.ListAreas(ctx)
	if err != nil {
		return nil, fmt.Errorf("list areas: %w", err)
	}
	areas := make([]*domain.Area, len(rows))
	for i, row := range rows {
		areas[i] = areaFromRow(row)
	}
	return areas, nil
}

func (s *AreaService) Rename(ctx context.Context, id, title, actorID string) error {
	if title == "" {
		return fmt.Errorf("area title cannot be empty")
	}
	now := timeNow()
	if err := s.queries.UpdateAreaTitle(ctx, sqlc.UpdateAreaTitleParams{
		Title: title, UpdatedAt: now, ID: id,
	}); err != nil {
		return fmt.Errorf("rename area: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{"title": title})
	s.events.AppendDomainEvent(ctx, domain.EventAreaRenamed, "area", id, actorID, payload)
	s.bus.Publish(domain.NewDomainEvent(domain.EventAreaRenamed, "area", id, actorID, map[string]any{"title": title}))
	return nil
}

func (s *AreaService) Archive(ctx context.Context, id, actorID string) error {
	now := timeNow()
	if err := s.queries.UpdateAreaArchived(ctx, sqlc.UpdateAreaArchivedParams{
		Archived: 1, UpdatedAt: now, ID: id,
	}); err != nil {
		return fmt.Errorf("archive area: %w", err)
	}
	s.events.AppendDomainEvent(ctx, domain.EventAreaArchived, "area", id, actorID, nil)
	s.bus.Publish(domain.NewDomainEvent(domain.EventAreaArchived, "area", id, actorID, nil))
	return nil
}

func (s *AreaService) Unarchive(ctx context.Context, id, actorID string) error {
	now := timeNow()
	if err := s.queries.UpdateAreaArchived(ctx, sqlc.UpdateAreaArchivedParams{
		Archived: 0, UpdatedAt: now, ID: id,
	}); err != nil {
		return fmt.Errorf("unarchive area: %w", err)
	}
	s.events.AppendDomainEvent(ctx, domain.EventAreaUnarchived, "area", id, actorID, nil)
	s.bus.Publish(domain.NewDomainEvent(domain.EventAreaUnarchived, "area", id, actorID, nil))
	return nil
}

func (s *AreaService) Delete(ctx context.Context, id, actorID string, cascade bool) error {
	now := timeNow()

	if cascade {
		s.queries.CascadeDeleteProjectsByArea(ctx, sqlc.CascadeDeleteProjectsByAreaParams{DeletedAt: &now, UpdatedAt: now, AreaID: &id})
		s.queries.CascadeDeleteTasksByArea(ctx, sqlc.CascadeDeleteTasksByAreaParams{DeletedAt: &now, UpdatedAt: now, AreaID: &id})
	} else {
		s.queries.OrphanProjectsByArea(ctx, sqlc.OrphanProjectsByAreaParams{UpdatedAt: now, AreaID: &id})
		s.queries.OrphanTasksByArea(ctx, sqlc.OrphanTasksByAreaParams{UpdatedAt: now, AreaID: &id})
	}

	if err := s.queries.SoftDeleteArea(ctx, sqlc.SoftDeleteAreaParams{
		DeletedAt: &now, UpdatedAt: now, ID: id,
	}); err != nil {
		return fmt.Errorf("delete area: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{"cascade": cascade})
	s.events.AppendDomainEvent(ctx, domain.EventAreaDeleted, "area", id, actorID, payload)
	s.bus.Publish(domain.NewDomainEvent(domain.EventAreaDeleted, "area", id, actorID, map[string]any{"cascade": cascade}))
	return nil
}

func areaFromRow(row sqlc.Area) *domain.Area {
	return &domain.Area{
		ID:       row.ID,
		Title:    row.Title,
		Index:    int(row.Index),
		Archived: row.Archived != 0,
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
		SoftDelete: domain.SoftDelete{
			Deleted: row.Deleted != 0,
		},
	}
}
```

Note: `timeNow()` is a package-level helper — add to a `internal/service/helpers.go`:
```go
package service

import "time"

func timeNow() time.Time {
	return time.Now().UTC()
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/service/ -v`
Expected: PASS (may need to adjust sqlc generated types — adapt as needed)

- [ ] **Step 5: Commit**

```bash
git add internal/service/
git commit -m "feat: add area service with CRUD, archive, and event emission"
```

### Task 13: Implement Task service

**Files:**
- Create: `internal/service/task_service.go`

This is the largest service — covers all task operations. Follow the same pattern as Area service: validate → persist → emit events.

- [ ] **Step 1: Write tests for core task operations**

Create `internal/service/task_service_test.go` with tests for: `Create`, `Get`, `Complete`, `Cancel`, `UpdateTitle`, `UpdateSchedule`, `Delete`. Follow the same pattern as `area_service_test.go` using `setupTest`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run TestTaskService -v`
Expected: FAIL

- [ ] **Step 3: Implement task_service.go**

Implement `NewTaskService`, `Create`, `Get`, `List`, `Complete`, `Cancel`, `UpdateTitle`, `UpdateNotes`, `UpdateSchedule`, `SetStartDate`, `SetDeadline`, `MoveToProject`, `MoveToSection`, `MoveToArea`, `SetLocation`, `SetRecurrence`, `AddTag`, `RemoveTag`, `AddLink`, `RemoveLink`, `Reorder`, `Delete` — each following the same pattern as AreaService methods.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/service/ -run TestTaskService -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/task_service.go internal/service/task_service_test.go
git commit -m "feat: add task service with all semantic operations"
```

### Task 14: Implement Project service (with cascade)

**Files:**
- Create: `internal/service/project_service.go`

Key behavior: `Complete` cascades to mark all open tasks as completed. `Cancel` cascades to mark all open tasks as cancelled. `Delete` cascades to tombstone all tasks and sections.

- [ ] **Step 1: Write tests**

Create `internal/service/project_service_test.go` with tests for: `Create`, `Complete` (verify tasks cascade), `Cancel` (verify tasks cascade), `Delete` (verify sections + tasks cascade).

- [ ] **Step 2: Run tests to verify they fail**
- [ ] **Step 3: Implement project_service.go**
- [ ] **Step 4: Run tests**

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/project_service.go internal/service/project_service_test.go
git commit -m "feat: add project service with cascade complete/cancel/delete"
```

### Task 15: Implement remaining services (Tag, Section, ChecklistItem, Location, Activity)

**Files:**
- Create: `internal/service/tag_service.go`, `section_service.go`, `checklist_service.go`, `location_service.go`, `activity_service.go`

These follow the established pattern. Each is straightforward CRUD + event emission.

- [ ] **Step 1: Write tests for each service**
- [ ] **Step 2: Run tests to verify they fail**
- [ ] **Step 3: Implement all services**
- [ ] **Step 4: Run all tests**

Run: `go test ./internal/service/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/
git commit -m "feat: add tag, section, checklist, location, and activity services"
```

---

## Phase 6: API Layer

### Task 16: Implement API response helpers and middleware

**Files:**
- Create: `internal/api/response.go`
- Create: `internal/api/middleware.go`

- [ ] **Step 1: Write the test**

Create `internal/api/response_test.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondJSON(rec, http.StatusOK, map[string]any{"data": "test"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["data"] != "test" {
		t.Errorf("body = %v, want data=test", body)
	}
}

func TestRespondEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondEvent(rec, http.StatusCreated, "task.created", map[string]any{"id": "123"})

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["event"] != "task.created" {
		t.Errorf("event = %v, want task.created", body["event"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Implement response.go and middleware.go**

`response.go` — helpers: `RespondJSON(w, status, data)`, `RespondEvent(w, status, eventType, data)`, `RespondError(w, status, message)`

`middleware.go` — logging middleware (request ID, duration, status code using `log/slog`)

- [ ] **Step 4: Run tests**
- [ ] **Step 5: Commit**

```bash
git add internal/api/response.go internal/api/response_test.go internal/api/middleware.go
git commit -m "feat: add API response helpers and logging middleware"
```

### Task 17: Implement Area API handlers

**Files:**
- Create: `internal/api/areas.go`

- [ ] **Step 1: Write integration test**

Create `internal/api/areas_test.go` — test `POST /areas`, `GET /areas`, `GET /areas/{id}`, `PUT /areas/{id}`, `DELETE /areas/{id}`, `POST /areas/{id}/archive`. Use `httptest` with a real in-memory database.

- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Implement areas.go handlers**

Each handler: parse request → call service → respond with event envelope.

- [ ] **Step 4: Run tests**
- [ ] **Step 5: Commit**

```bash
git add internal/api/areas.go internal/api/areas_test.go
git commit -m "feat: add area API handlers"
```

### Task 18: Implement Task API handlers

**Files:**
- Create: `internal/api/tasks.go`

- [ ] **Step 1: Write integration tests for core operations**
- [ ] **Step 2: Implement tasks.go** — all task endpoints
- [ ] **Step 3: Run tests**
- [ ] **Step 4: Commit**

```bash
git add internal/api/tasks.go internal/api/tasks_test.go
git commit -m "feat: add task API handlers with all semantic operations"
```

### Task 19: Implement Project, Section, Checklist, Tag, Location, Activity API handlers

**Files:**
- Create: `internal/api/projects.go`, `sections.go`, `checklist.go`, `tags.go`, `locations.go`, `activities.go`

- [ ] **Step 1: Write integration tests**
- [ ] **Step 2: Implement all handlers**
- [ ] **Step 3: Run tests**
- [ ] **Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat: add project, section, checklist, tag, location, and activity API handlers"
```

### Task 20: Implement Views API handlers

**Files:**
- Create: `internal/api/views.go`

- [ ] **Step 1: Write tests**

Test each view returns the correct subset of tasks: inbox (schedule=inbox), today (schedule=anytime, start_date <= today), upcoming (start_date > today), someday (schedule=someday), logbook (completed/cancelled).

- [ ] **Step 2: Implement views.go**
- [ ] **Step 3: Run tests**
- [ ] **Step 4: Commit**

```bash
git add internal/api/views.go internal/api/views_test.go
git commit -m "feat: add views API (inbox, today, upcoming, someday, logbook)"
```

### Task 21: Implement SSE and Sync endpoints

**Files:**
- Create: `internal/api/events.go`
- Create: `internal/api/sync.go`

- [ ] **Step 1: Write tests for SSE endpoint and sync endpoint**
- [ ] **Step 2: Implement events.go** — wire SSE StreamManager to `GET /events/stream`
- [ ] **Step 3: Implement sync.go** — `GET /sync/deltas?since={cursor}` returning delta events
- [ ] **Step 4: Run tests**
- [ ] **Step 5: Commit**

```bash
git add internal/api/events.go internal/api/sync.go internal/api/events_test.go internal/api/sync_test.go
git commit -m "feat: add SSE event stream and delta sync endpoints"
```

---

## Phase 7: Auth

### Task 22: Implement Auth service

**Files:**
- Create: `internal/service/auth_service.go`

- [ ] **Step 1: Write tests for user creation, login (password hashing), API key creation/verification**
- [ ] **Step 2: Implement auth_service.go** — `CreateUser`, `Login` (bcrypt), `CreateAPIKey` (generate + hash), `ValidateAPIKey`, `ListAPIKeys`, `DeleteAPIKey`
- [ ] **Step 3: Add dependency:** `go get golang.org/x/crypto/bcrypt`
- [ ] **Step 4: Run tests**
- [ ] **Step 5: Commit**

```bash
git add internal/service/auth_service.go internal/service/auth_service_test.go go.mod go.sum
git commit -m "feat: add auth service with JWT and API key management"
```

### Task 23: Implement Auth middleware and API handlers

**Files:**
- Create: `internal/api/auth.go`
- Modify: `internal/api/middleware.go` — add auth middleware

- [ ] **Step 1: Write tests for auth endpoints and middleware (valid/invalid token, API key auth)**
- [ ] **Step 2: Implement auth.go handlers** — `POST /auth/login`, `POST /auth/refresh`, `GET /auth/me`, etc.
- [ ] **Step 3: Add JWT middleware** — extract Bearer token or API key, validate, set actor in context
- [ ] **Step 4: Add dependency:** `go get github.com/golang-jwt/jwt/v5`
- [ ] **Step 5: Run tests**
- [ ] **Step 6: Commit**

```bash
git add internal/api/auth.go internal/api/middleware.go go.mod go.sum
git commit -m "feat: add auth API handlers and JWT/API-key middleware"
```

---

## Phase 8: Router and Wiring

### Task 24: Wire everything together in router and main

**Files:**
- Create: `internal/api/router.go`
- Modify: `cmd/atask/main.go`

- [ ] **Step 1: Implement router.go**

Register all routes with their handlers:
```go
func NewRouter(/* all services, event bus, stream manager */) http.Handler {
    mux := http.NewServeMux()
    // Register all routes...
    // Wrap with middleware
    return middleware(mux)
}
```

- [ ] **Step 2: Update main.go**

Wire: open DB → migrate → create services → create bus + stream manager → create router → start server.

- [ ] **Step 3: Run full test suite**

Run: `go test -race ./...`
Expected: PASS

- [ ] **Step 4: Build and smoke test**

Run:
```bash
go build -o atask ./cmd/atask
./atask &
curl http://localhost:8080/health
curl -X POST http://localhost:8080/areas -H 'Content-Type: application/json' -d '{"title":"Work"}'
kill %1
```
Expected: health returns ok, area creation returns event envelope

- [ ] **Step 5: Commit**

```bash
git add internal/api/router.go cmd/atask/main.go
git commit -m "feat: wire all services, routes, and middleware in main"
```

---

## Phase 9: Docker and Final Verification

### Task 25: Build and verify Docker deployment

- [ ] **Step 1: Build Docker image**

Run: `docker build -t atask .`
Expected: build succeeds

- [ ] **Step 2: Run with docker compose**

Run: `docker compose up -d`
Expected: container starts, health endpoint responds

- [ ] **Step 3: Smoke test against Docker container**

Run:
```bash
curl http://localhost:8080/health
curl -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"test@test.com","password":"test"}'
```

- [ ] **Step 4: Run full test suite one final time**

Run: `go test -race -count=1 ./...`
Expected: all tests PASS

- [ ] **Step 5: Docker compose down and commit**

```bash
docker compose down
git add Dockerfile docker-compose.yml
git commit -m "feat: verify Docker build and deployment"
```
