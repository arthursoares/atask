# atask TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a three-pane Bubbletea v2 TUI for atask with sidebar navigation, task lists, detail view, command palette, and SSE live updates — all consuming the atask REST API.

**Architecture:** Thin client pattern. An HTTP client (`internal/client/`) wraps the REST API. The TUI (`internal/tui/`) uses Bubbletea v2 with a root model coordinating three sub-models (sidebar, list, detail) plus overlay models (command palette, search, help). SSE events arrive as `tea.Msg` for live updates. CLI entry point uses cobra for `serve` vs TUI subcommands.

**Tech Stack:** Go 1.25, Bubbletea v2, Bubbles v2, Lipgloss v2, Cobra, atask REST API

**Spec:** `docs/superpowers/specs/2026-03-19-atask-tui-design.md`

---

## File Map

```
cmd/atask/main.go                        → modify: cobra CLI with serve/tui subcommands
internal/
  client/
    client.go                            → HTTP API client (auth, CRUD, views)
    client_test.go                       → client integration tests
    sse.go                               → SSE subscription client
    sse_test.go                          → SSE client tests
  tui/
    app.go                               → root model, pane coordination, window sizing
    app_test.go                          → root model tests
    sidebar.go                           → sidebar model (views, areas, projects, tags)
    list.go                              → list pane model (task rows, inline edit)
    detail.go                            → detail pane model (header, tabs, content)
    palette.go                           → command palette overlay (fuzzy search)
    search.go                            → search/filter overlay
    help.go                              → help overlay
    keys.go                              → key binding definitions
    styles.go                            → lipgloss styles (colors, borders, layout)
    messages.go                          → custom tea.Msg types for API responses and SSE
    confirm.go                           → confirmation dialog model (delete, etc.)
    picker.go                            → fuzzy picker for projects/tags/locations
    statusbar.go                         → bottom status bar (errors, flash messages, key hints)
```

---

## Phase 1: HTTP Client

### Task 1: Create HTTP API client

**Files:**
- Create: `internal/client/client.go`
- Create: `internal/client/client_test.go`

The client is the TUI's only interface to atask. It wraps every API endpoint.

- [ ] **Step 1: Write tests for basic operations**

Create `internal/client/client_test.go`:
```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// setupTestServer creates a full atask server for integration testing.
func setupTestServer(t *testing.T) (*httptest.Server, *Client) {
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
	sm := event.NewStreamManager(bus)

	authSvc := service.NewAuthService(db, "test-secret")
	areaSvc := service.NewAreaService(db, es, bus)
	taskSvc := service.NewTaskService(db, es, bus)
	projectSvc := service.NewProjectService(db, es, bus)
	sectionSvc := service.NewSectionService(db, es, bus)
	tagSvc := service.NewTagService(db, es, bus)
	locationSvc := service.NewLocationService(db, es, bus)
	checklistSvc := service.NewChecklistService(db, es, bus)
	activitySvc := service.NewActivityService(db, es, bus)

	handler := api.NewRouter(
		api.NewAreaHandler(areaSvc),
		api.NewTaskHandler(taskSvc),
		api.NewProjectHandler(projectSvc),
		api.NewSectionHandler(sectionSvc),
		api.NewTagHandler(tagSvc),
		api.NewLocationHandler(locationSvc),
		api.NewChecklistHandler(checklistSvc),
		api.NewActivityHandler(activitySvc),
		api.NewViewHandler(db),
		api.NewEventsHandler(sm),
		api.NewSyncHandler(es),
		api.NewAuthHandler(authSvc),
		authSvc,
	)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// Register and login to get a token
	c := New(srv.URL, "")
	ctx := context.Background()
	token, err := c.Register(ctx, "test@test.com", "password", "Test User")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	c.token = token

	return srv, c
}

func TestClient_Areas(t *testing.T) {
	_, c := setupTestServer(t)
	ctx := context.Background()

	// Create
	area, err := c.CreateArea(ctx, "Work")
	if err != nil {
		t.Fatalf("CreateArea: %v", err)
	}
	if area.Title != "Work" {
		t.Errorf("Title = %q, want %q", area.Title, "Work")
	}

	// List
	areas, err := c.ListAreas(ctx)
	if err != nil {
		t.Fatalf("ListAreas: %v", err)
	}
	if len(areas) != 1 {
		t.Errorf("got %d areas, want 1", len(areas))
	}
}

func TestClient_Tasks(t *testing.T) {
	_, c := setupTestServer(t)
	ctx := context.Background()

	// Create
	task, err := c.CreateTask(ctx, "Buy groceries")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", task.Title, "Buy groceries")
	}

	// Complete
	if err := c.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// View inbox (should be empty after completion)
	inbox, err := c.ListInbox(ctx)
	if err != nil {
		t.Fatalf("ListInbox: %v", err)
	}
	if len(inbox) != 0 {
		t.Errorf("inbox has %d tasks, want 0", len(inbox))
	}
}

func TestClient_Projects(t *testing.T) {
	_, c := setupTestServer(t)
	ctx := context.Background()

	project, err := c.CreateProject(ctx, "Q2 Launch")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.Title != "Q2 Launch" {
		t.Errorf("Title = %q, want %q", project.Title, "Q2 Launch")
	}

	projects, err := c.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("got %d projects, want 1", len(projects))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/client/ -v`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Implement client.go**

Create `internal/client/client.go` with:

```go
package client

// Client wraps the atask REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) *Client
```

**Response types** (used for JSON decoding — mirror API responses):
```go
type Task struct {
	ID             string     `json:"ID"`
	Title          string     `json:"Title"`
	Notes          string     `json:"Notes"`
	Status         int        `json:"Status"`
	Schedule       int        `json:"Schedule"`
	StartDate      *string    `json:"StartDate"`
	Deadline       *string    `json:"Deadline"`
	CompletedAt    *string    `json:"CompletedAt"`
	Index          int        `json:"Index"`
	TodayIndex     *int       `json:"TodayIndex"`
	ProjectID      *string    `json:"ProjectID"`
	SectionID      *string    `json:"SectionID"`
	AreaID         *string    `json:"AreaID"`
	LocationID     *string    `json:"LocationID"`
	Tags           []string   `json:"Tags"`
	CreatedAt      string     `json:"CreatedAt"`
	UpdatedAt      string     `json:"UpdatedAt"`
}

type Project struct { /* mirrors API response */ }
type Area struct { /* mirrors API response */ }
type Tag struct { /* mirrors API response */ }
type Location struct { /* mirrors API response */ }
type ChecklistItem struct { /* mirrors API response */ }
type Activity struct { /* mirrors API response */ }
type Section struct { /* mirrors API response */ }
```

**Methods** — each follows the same pattern: build request, add auth header, do request, decode response:

Auth:
- `Register(ctx, email, password, name) (token string, err error)`
- `Login(ctx, email, password) (token string, err error)`
- `CreateAPIKey(ctx, name) (key string, err error)`

Views:
- `ListInbox(ctx) ([]Task, error)`
- `ListToday(ctx) ([]Task, error)`
- `ListUpcoming(ctx) ([]Task, error)`
- `ListSomeday(ctx) ([]Task, error)`
- `ListLogbook(ctx) ([]Task, error)`

Tasks:
- `CreateTask(ctx, title) (*Task, error)`
- `GetTask(ctx, id) (*Task, error)`
- `CompleteTask(ctx, id) error`
- `CancelTask(ctx, id) error`
- `DeleteTask(ctx, id) error`
- `UpdateTaskTitle(ctx, id, title) error`
- `UpdateTaskNotes(ctx, id, notes) error`
- `UpdateTaskSchedule(ctx, id, schedule string) error`
- `SetTaskStartDate(ctx, id, date *string) error`
- `SetTaskDeadline(ctx, id, date *string) error`
- `MoveTaskToProject(ctx, id, projectID *string) error`
- `MoveTaskToSection(ctx, id, sectionID *string) error`
- `MoveTaskToArea(ctx, id, areaID *string) error`
- `SetTaskLocation(ctx, id, locationID *string) error`
- `AddTaskTag(ctx, taskID, tagID) error`
- `RemoveTaskTag(ctx, taskID, tagID) error`

Projects:
- `CreateProject(ctx, title) (*Project, error)`
- `ListProjects(ctx) ([]Project, error)`
- `GetProject(ctx, id) (*Project, error)`
- `CompleteProject(ctx, id) error`
- `CancelProject(ctx, id) error`
- `DeleteProject(ctx, id) error`
- `UpdateProjectTitle(ctx, id, title) error`

Areas:
- `CreateArea(ctx, title) (*Area, error)`
- `ListAreas(ctx) ([]Area, error)`
- `ArchiveArea(ctx, id) error`
- `UnarchiveArea(ctx, id) error`
- `DeleteArea(ctx, id, cascade bool) error`
- `RenameArea(ctx, id, title) error`

Tags:
- `CreateTag(ctx, title) (*Tag, error)`
- `ListTags(ctx) ([]Tag, error)`
- `RenameTag(ctx, id, title) error`
- `DeleteTag(ctx, id) error`

Locations:
- `CreateLocation(ctx, name) (*Location, error)`
- `ListLocations(ctx) ([]Location, error)`
- `RenameLocation(ctx, id, name) error`
- `DeleteLocation(ctx, id) error`

Sections:
- `CreateSection(ctx, projectID, title) (*Section, error)`
- `ListSections(ctx, projectID) ([]Section, error)`
- `RenameSection(ctx, projectID, sectionID, title) error`
- `DeleteSection(ctx, projectID, sectionID, cascade bool) error`

Checklist:
- `AddChecklistItem(ctx, taskID, title) (*ChecklistItem, error)`
- `ListChecklistItems(ctx, taskID) ([]ChecklistItem, error)`
- `CompleteChecklistItem(ctx, taskID, itemID) error`
- `UncompleteChecklistItem(ctx, taskID, itemID) error`
- `DeleteChecklistItem(ctx, taskID, itemID) error`

Activity:
- `AddActivity(ctx, taskID, actorType, activityType, content) (*Activity, error)`
- `ListActivities(ctx, taskID) ([]Activity, error)`

Tasks by relationship:
- `ListTasksByProject(ctx, projectID) ([]Task, error)`
- `ListTasksByArea(ctx, areaID) ([]Task, error)`

Use private helpers:
```go
func (c *Client) doJSON(ctx, method, path string, body, result any) error
func (c *Client) doAction(ctx, method, path string, body any) error  // for mutations that return event envelope
```

`doJSON` — builds request, sets `Authorization: Bearer {token}`, Content-Type, encodes body, decodes response into result.

`doAction` — same but decodes the event envelope `{"event": ..., "data": ...}` and extracts data into result if provided.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/client/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/client/client.go internal/client/client_test.go
git commit -m "feat: add HTTP API client for TUI"
```

### Task 2: Create SSE subscription client

**Files:**
- Create: `internal/client/sse.go`
- Create: `internal/client/sse_test.go`

- [ ] **Step 1: Write test**

Create `internal/client/sse_test.go`:
```go
func TestClient_SubscribeEvents(t *testing.T) {
	srv, c := setupTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := c.SubscribeEvents(ctx, "*")
	if err != nil {
		t.Fatalf("SubscribeEvents: %v", err)
	}

	// Create a task — should trigger a domain event
	_, err = c.CreateTask(ctx, "Test SSE")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Read event from channel
	select {
	case evt := <-events:
		if evt.Type != "task.created" {
			t.Errorf("Type = %q, want task.created", evt.Type)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for SSE event")
	}
}
```

- [ ] **Step 2: Implement sse.go**

```go
package client

// DomainEvent represents an SSE domain event.
type DomainEvent struct {
	ID       int64                  `json:"id"`
	Type     string                 `json:"type"`
	EntityID string                 `json:"entity_id"`
	Payload  map[string]any         `json:"payload"`
}

// SubscribeEvents connects to the SSE endpoint and returns a channel
// of domain events. Closes the channel when ctx is cancelled.
// Handles reconnection with Last-Event-ID.
func (c *Client) SubscribeEvents(ctx context.Context, topics string) (<-chan DomainEvent, error)
```

Implementation:
1. `GET {baseURL}/events/stream?topics={topics}` with auth header and `Accept: text/event-stream`
2. Parse SSE format line by line: `event:`, `data:`, `id:` fields
3. Unmarshal data JSON into DomainEvent, set Type from event line, set ID from id line
4. Send to buffered channel (cap 256)
5. On connection drop, reconnect with `Last-Event-ID` header after 1s delay
6. Close channel when ctx is cancelled

- [ ] **Step 3: Run tests**

Run: `go test ./internal/client/ -v -run TestClient_SubscribeEvents`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/client/sse.go internal/client/sse_test.go
git commit -m "feat: add SSE subscription client"
```

---

## Phase 2: CLI and Styles Foundation

### Task 3: Add cobra CLI with serve and TUI subcommands

**Files:**
- Modify: `cmd/atask/main.go`

- [ ] **Step 1: Add cobra dependency**

Run: `go get github.com/spf13/cobra`

- [ ] **Step 2: Rewrite main.go with cobra**

The current `main.go` starts the server directly. Refactor to:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "atask",
	Short: "AI-first task manager",
	// Default: run TUI
	RunE: runTUI,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the headless API server",
	RunE:  runServe,
}

var (
	flagServer    string
	flagAddr      string
	flagDBPath    string
	flagJWTSecret string
)

func init() {
	rootCmd.Flags().StringVar(&flagServer, "server", "http://localhost:8080", "API server URL")

	serveCmd.Flags().StringVar(&flagAddr, "addr", ":8080", "Server listen address")
	serveCmd.Flags().StringVar(&flagDBPath, "db", "atask.db", "Database file path")
	serveCmd.Flags().StringVar(&flagJWTSecret, "jwt-secret", "", "JWT signing secret")

	rootCmd.AddCommand(serveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runServe contains the server startup code.
// Move the entire body of the current main() function from cmd/atask/main.go
// into this function. Replace env var reads with flag values (with env fallback):
//   dbPath = flagDBPath (or DB_PATH env, or "atask.db")
//   jwtSecret = flagJWTSecret (or JWT_SECRET env, or "change-me-in-production")
//   addr = flagAddr (or ADDR env, or ":8080")
// The rest (DB open, migrate, create services, create handlers, router, server) stays the same.
func runServe(cmd *cobra.Command, args []string) error {
	// ... existing server startup code from current main() ...
}

// runTUI starts the Bubbletea TUI
func runTUI(cmd *cobra.Command, args []string) error {
	// Placeholder — will be implemented in later tasks
	fmt.Println("TUI not yet implemented. Use 'atask serve' to start the server.")
	return nil
}
```

`runServe` moves the existing server code into the cobra command. Env var fallbacks remain (DB_PATH, JWT_SECRET, ADDR) but flags take priority.

- [ ] **Step 3: Verify build and both commands work**

Run:
```bash
go build -o /tmp/atask ./cmd/atask
/tmp/atask --help
/tmp/atask serve --help
```

- [ ] **Step 4: Commit**

```bash
git add cmd/atask/main.go go.mod go.sum
git commit -m "feat: add cobra CLI with serve and TUI subcommands"
```

### Task 4: Create TUI styles and key bindings

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/keys.go`
- Create: `internal/tui/messages.go`

- [ ] **Step 1: Implement styles.go**

Define all lipgloss styles:
```go
package tui

import "github.com/charmbracelet/lipgloss/v2"

var (
	// Pane borders
	focusedBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("39"))  // cyan
	blurredBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))

	// Sidebar
	sidebarWidth     = 22
	sectionHeader    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true).MarginTop(1)
	selectedItem     = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	dimmedItem       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	countBadge       = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Align(lipgloss.Right)

	// List
	taskRow          = lipgloss.NewStyle()
	selectedTask     = lipgloss.NewStyle().Background(lipgloss.Color("17")).Foreground(lipgloss.Color("39"))
	completedTask    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)
	overdueDeadline  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))  // red

	// Detail
	detailTitle      = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	metadataLine     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	activeTab        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Underline(true)
	inactiveTab      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Activity
	agentActor       = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))  // purple
	humanActor       = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))   // cyan

	// Status bar
	statusBar        = lipgloss.NewStyle().Background(lipgloss.Color("235")).Padding(0, 1)
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	flashStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))  // yellow

	// Overlays
	overlayStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("39")).Padding(1, 2)
)
```

- [ ] **Step 2: Implement keys.go**

Define key bindings using bubbletea v2 key types:
```go
package tui

import tea "github.com/charmbracelet/bubbletea/v2"

// Key binding definitions — checked in Update methods
type KeyMap struct {
	// Navigation
	Up, Down, Enter, Escape   tea.KeyType
	Tab, ShiftTab             tea.KeyType
	Top, Bottom               tea.KeyType

	// Actions (checked as runes)
}

// Helper to check key matches
func isKey(msg tea.KeyMsg, keys ...string) bool
func isRune(msg tea.KeyMsg, r rune) bool
```

- [ ] **Step 3: Implement messages.go**

Define all custom tea.Msg types:
```go
package tui

import "github.com/atask/atask/internal/client"

// API response messages
type TasksLoadedMsg struct{ Tasks []client.Task }
type TaskCreatedMsg struct{ Task client.Task }
type TaskCompletedMsg struct{ ID string }
type TaskDeletedMsg struct{ ID string }
type ErrorMsg struct{ Err error }

// SSE event message
type SSEEventMsg struct{ Event client.DomainEvent }

// Navigation
type ViewSelectedMsg struct{ View string }
type ProjectSelectedMsg struct{ ID string }
type AreaSelectedMsg struct{ ID string }
type TagSelectedMsg struct{ ID string }
type TaskSelectedMsg struct{ ID string }

// UI state
type FocusPaneMsg struct{ Pane int }
type FlashMsg struct{ Message string }
type RefreshMsg struct{}

// Areas, Projects, Tags loaded
type AreasLoadedMsg struct{ Areas []client.Area }
type ProjectsLoadedMsg struct{ Projects []client.Project }
type TagsLoadedMsg struct{ Tags []client.Tag }
type LocationsLoadedMsg struct{ Locations []client.Location }
type SectionsLoadedMsg struct{ Sections []client.Section }
type ChecklistLoadedMsg struct{ Items []client.ChecklistItem }
type ActivitiesLoadedMsg struct{ Activities []client.Activity }
```

- [ ] **Step 4: Add bubbletea v2 dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea/v2
go get github.com/charmbracelet/bubbles/v2
go get github.com/charmbracelet/lipgloss/v2
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add TUI styles, key bindings, and message types"
```

---

## Phase 3: Core TUI Shell

### Task 5: Implement root app model with three-pane layout

**Files:**
- Create: `internal/tui/app.go`
- Create: `internal/tui/statusbar.go`

The root model holds the three panes, manages focus, handles global keys, and coordinates data flow.

- [ ] **Step 1: Implement app.go**

```go
package tui

import (
	"github.com/atask/atask/internal/client"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

const (
	PaneSidebar = iota
	PaneList
	PaneDetail
)

type App struct {
	client   *client.Client
	width    int
	height   int
	focus    int  // PaneSidebar, PaneList, PaneDetail

	sidebar  Sidebar
	list     List
	detail   Detail
	statusbar StatusBar

	// Overlay state
	palette  *Palette
	search   *Search
	help     *Help
	confirm  *Confirm

	// Cached data
	areas     []client.Area
	projects  []client.Project
	tags      []client.Tag
	locations []client.Location
}

func NewApp(c *client.Client) App

func (a App) Init() tea.Cmd
// Init loads initial data: areas, projects, tags, locations, inbox tasks
// Also starts SSE subscription in background

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd)
// Handles:
// - tea.WindowSizeMsg → resize all panes
// - tea.KeyMsg → global keys (Tab, q, /, :, ?, r) then delegate to focused pane
// - API response messages → update cached data, propagate to panes
// - SSEEventMsg → trigger targeted refresh
// - Overlay messages → show/hide overlays

func (a App) View() string
// Renders three panes side by side using lipgloss.JoinHorizontal
// with status bar at bottom using lipgloss.JoinVertical
// If overlay is active, render it centered over the layout
```

Key behaviors:
- `Tab` / `Shift+Tab` cycles focus (skip detail if collapsed)
- `Esc` moves focus back (detail→list, list→sidebar)
- `q` quits (or closes overlay if one is open)
- `r` triggers RefreshMsg
- `:` or `Ctrl+P` opens command palette
- `/` opens search
- `?` opens help
- Window size < 80 cols → collapse detail pane

- [ ] **Step 2: Implement statusbar.go**

Simple model rendering a single line at the bottom with:
- Left: current context ("Inbox", "Today — 5 tasks", "Project: Q2 Launch")
- Center: flash messages (auto-clear after 3 seconds)
- Right: key hints ("[Tab] panes  [?] help  [:] cmd")

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/`

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go internal/tui/statusbar.go
git commit -m "feat: add root TUI app model with three-pane layout"
```

### Task 6: Implement sidebar model

**Files:**
- Create: `internal/tui/sidebar.go`

- [ ] **Step 1: Implement sidebar.go**

```go
type SidebarItem struct {
	Label    string
	Kind     string  // "view", "area", "project", "tag", "section-header"
	ID       string
	Count    int
	Indent   int
	Expanded bool
}

type Sidebar struct {
	items    []SidebarItem
	cursor   int
	height   int
	offset   int  // scroll offset
}

func NewSidebar() Sidebar
func (s Sidebar) Update(msg tea.Msg) (Sidebar, tea.Cmd)
func (s Sidebar) View() string
func (s *Sidebar) SetData(areas []client.Area, projects []client.Project, tags []client.Tag, taskCounts map[string]int)
func (s Sidebar) SelectedItem() SidebarItem
```

Items are built from cached data:
```
[view]    📥 Inbox  (3)
[view]    ⭐ Today  (5)
[view]    📅 Upcoming (12)
[view]    💤 Someday (8)
[view]    📓 Logbook
[header]  AREAS
[area]    ▾ Work
[project]   📁 Q2 Launch
[project]   📁 Website
[area]    ▸ Personal
[header]  TAGS
[tag]     🏷 urgent
[tag]     🏷 waiting
```

Navigation:
- `j/k` moves cursor (skips section headers)
- `Enter` on view → emits ViewSelectedMsg
- `Enter` on area → toggle expanded, emits AreaSelectedMsg
- `Enter` on project → emits ProjectSelectedMsg
- `Enter` on tag → emits TagSelectedMsg
- `n` on area section → create new area
- `n` on expanded area → create new project in that area
- `e` → rename selected area/project
- `d` → delete confirmation
- `a` on area → archive/unarchive

- [ ] **Step 2: Commit**

```bash
git add internal/tui/sidebar.go
git commit -m "feat: add sidebar model with views, areas, projects, tags"
```

### Task 7: Implement list pane model

**Files:**
- Create: `internal/tui/list.go`

- [ ] **Step 1: Implement list.go**

```go
type List struct {
	tasks     []client.Task
	cursor    int
	height    int
	width     int
	offset    int
	editing   bool      // inline title edit mode
	editInput textinput.Model
	filter    string    // active search filter
	title     string    // e.g., "Today — 5 tasks"
}

func NewList() List
func (l List) Update(msg tea.Msg) (List, tea.Cmd)
func (l List) View() string
func (l *List) SetTasks(tasks []client.Task, title string)
func (l List) SelectedTask() *client.Task  // returns nil if no tasks or cursor out of range
```

Rendering each row:
```
☐ Buy groceries                    Mar 20
☐ Write email to Doctor          📁 Health
✓ Fix login bug                     (dim)
```

Inline edit:
- `e` enters edit mode — replaces the selected row with a text input
- `Enter` saves, `Esc` cancels
- The text input gets the current title as initial value

- [ ] **Step 2: Commit**

```bash
git add internal/tui/list.go
git commit -m "feat: add list pane model with task display and inline edit"
```

### Task 8: Implement detail pane model

**Files:**
- Create: `internal/tui/detail.go`

- [ ] **Step 1: Implement detail.go**

```go
type DetailTab int

const (
	TabNotes DetailTab = iota
	TabChecklist
	TabActivity
)

type Detail struct {
	task         *client.Task
	tab          DetailTab
	height       int
	width        int

	// Tab content
	notes        viewport.Model
	checklist    []client.ChecklistItem
	checkCursor  int
	activities   []client.Activity
	activityView viewport.Model
	latestActivity *client.Activity

	// Input mode
	addingComment bool
	commentInput  textinput.Model
	addingCheckItem bool
	checkItemInput  textinput.Model
}

func NewDetail() Detail
func (d Detail) Update(msg tea.Msg) (Detail, tea.Cmd)
func (d Detail) View() string
func (d *Detail) SetTask(task *client.Task)
func (d *Detail) SetChecklist(items []client.ChecklistItem)
func (d *Detail) SetActivities(activities []client.Activity)
```

View structure:
```
┌ Detail ─────────────────────────┐
│ Write email to Doctor           │
│ 📁 Health  🏷 waiting  📅 Mar 22│
│                                 │
│ [Notes]  [Checklist 2/4]  [Act.]│
│                                 │
│ (tab content here - viewport)   │
│                                 │
│ ─── Latest Activity ──────────  │
│ 🤖 agent — Draft ready.        │
└─────────────────────────────────┘
```

Tab switching: `1`, `2`, `3` keys or `Tab` within detail pane cycles tabs.

Notes tab: scrollable viewport rendering markdown-ish content (plain text for v0).
Checklist tab: navigable list with `j/k`, `x` toggles, `n` adds, `d` removes.
Activity tab: scrollable viewport with chronological entries. `a` opens comment input.

- [ ] **Step 2: Commit**

```bash
git add internal/tui/detail.go
git commit -m "feat: add detail pane model with notes, checklist, and activity tabs"
```

---

## Phase 4: Overlays

### Task 9: Implement command palette

**Files:**
- Create: `internal/tui/palette.go`

- [ ] **Step 1: Implement palette.go**

```go
type Command struct {
	Name     string
	Category string  // "Task", "Project", "Area", "Navigation", "System"
	Action   func() tea.Cmd
}

type Palette struct {
	input    textinput.Model
	commands []Command
	filtered []Command
	cursor   int
	height   int
}

func NewPalette(commands []Command) Palette
func (p Palette) Update(msg tea.Msg) (Palette, tea.Cmd)
func (p Palette) View() string
```

Fuzzy matching: filter commands by checking if input is a subsequence of command name (case-insensitive). Show matching commands with cursor selection.

Keys:
- Type to filter
- `j/k` or arrow keys to navigate
- `Enter` to execute selected command
- `Esc` to close

The root App model builds the command list based on current focus and selected item context.

- [ ] **Step 2: Commit**

```bash
git add internal/tui/palette.go
git commit -m "feat: add command palette with fuzzy search"
```

### Task 10: Implement picker, search, help, and confirm overlays

**Files:**
- Create: `internal/tui/picker.go`
- Create: `internal/tui/search.go`
- Create: `internal/tui/help.go`
- Create: `internal/tui/confirm.go`

- [ ] **Step 1: Implement picker.go**

Fuzzy picker for selecting projects, tags, or locations. Used by `m` (move to project), `t` (assign tag), `l` (set location).

```go
type PickerItem struct {
	ID    string
	Label string
}

type Picker struct {
	title    string
	input    textinput.Model
	items    []PickerItem
	filtered []PickerItem
	cursor   int
	onSelect func(id string) tea.Cmd
}
```

- [ ] **Step 2: Implement search.go**

```go
type Search struct {
	input  textinput.Model
	active bool
}
```

Renders at the top of the list pane. Updates filter on each keystroke. Esc clears.

- [ ] **Step 3: Implement help.go**

```go
type Help struct {
	viewport viewport.Model
}
```

Full-screen overlay showing the key binding matrix from the spec. Esc closes.

- [ ] **Step 4: Implement confirm.go**

```go
type Confirm struct {
	message string
	onYes   func() tea.Cmd
}
```

Simple yes/no dialog for destructive actions. `y` confirms, `n` or `Esc` cancels.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/picker.go internal/tui/search.go internal/tui/help.go internal/tui/confirm.go
git commit -m "feat: add picker, search, help, and confirm overlays"
```

---

## Phase 5: Wire TUI to CLI and SSE

### Task 11: Wire TUI startup in main.go

**Files:**
- Modify: `cmd/atask/main.go`

- [ ] **Step 1: Implement runTUI function**

```go
func runTUI(cmd *cobra.Command, args []string) error {
	c := client.New(flagServer, "")

	// Auth: check env var first, then prompt for login
	token := os.Getenv("ATASK_TOKEN")
	if token == "" {
		// Interactive login prompt before starting TUI
		fmt.Print("Email: ")
		var email string
		fmt.Scanln(&email)
		fmt.Print("Password: ")
		var password string
		fmt.Scanln(&password)
		var err error
		token, err = c.Login(context.Background(), email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}
	c.SetToken(token)

	app := tui.NewApp(c)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

Add `SetToken(token string)` method to client.

- [ ] **Step 2: Verify build**

Run: `go build ./cmd/atask`

- [ ] **Step 3: Commit**

```bash
git add cmd/atask/main.go internal/client/client.go
git commit -m "feat: wire TUI startup in cobra CLI"
```

### Task 12: Implement SSE integration as tea.Cmd

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/messages.go`

- [ ] **Step 1: Add SSE subscription command**

In app.go, add a tea.Cmd that starts the SSE subscription and converts events to tea.Msg:

```go
func (a App) subscribeSSE() tea.Msg {
	events, err := a.client.SubscribeEvents(context.Background(), "*")
	if err != nil {
		return ErrorMsg{Err: err}
	}
	// Return first event — bubbletea will call this cmd again for the next
	evt := <-events
	return SSEEventMsg{Event: evt}
}
```

In Bubbletea v2, use a long-running `tea.Cmd` pattern. The SSE goroutine reads from the events channel and returns one event at a time. After each event is processed, return another `tea.Cmd` that reads the next event. This avoids needing `p.Send()` or external goroutine coordination:

```go
func listenSSE(events <-chan client.DomainEvent) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-events
		if !ok {
			return SSEDisconnectedMsg{}
		}
		return SSEEventMsg{Event: evt}
	}
}
```

In Update, when receiving `SSEEventMsg`, return `listenSSE(a.sseEvents)` as the next command to keep listening. Initialize the channel in `Init()` via `SubscribeEvents`.

In `Update`, handle `SSEEventMsg`:
- Parse event type prefix (`task.*`, `project.*`, etc.)
- If it affects the current view, dispatch a refresh command for the relevant pane
- If it affects the selected task, refresh detail pane

- [ ] **Step 2: Commit**

```bash
git add internal/tui/app.go internal/tui/messages.go
git commit -m "feat: integrate SSE events into TUI update loop"
```

---

## Phase 6: Integration and Polish

### Task 13: Connect all pane interactions to API

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/sidebar.go`
- Modify: `internal/tui/list.go`
- Modify: `internal/tui/detail.go`

This is the wiring task — connect every user action to the appropriate API call and handle the response.

- [ ] **Step 1: Wire sidebar actions**

When sidebar emits a selection message, the App:
1. Calls the appropriate client method (e.g., `ListInbox()` for Inbox view)
2. Returns a tea.Cmd that produces a `TasksLoadedMsg`
3. On receiving `TasksLoadedMsg`, updates the list pane

Wire: create area, rename area, archive area, delete area, create project in area.

- [ ] **Step 2: Wire list actions**

When list emits an action:
- `n` → text input for title → `CreateTask` → `TaskCreatedMsg` → refresh list
- `x` → `CompleteTask` → `TaskCompletedMsg` → refresh list
- `d` → show Confirm → `DeleteTask` → `TaskDeletedMsg` → refresh list
- `e` → inline edit → `UpdateTaskTitle` → refresh list
- `s` → schedule picker (Inbox/Today/Someday) → `UpdateTaskSchedule` → refresh
- `m` → project picker → `MoveTaskToProject` → refresh
- `t` → tag picker → `AddTaskTag` / `RemoveTaskTag` → refresh

- [ ] **Step 3: Wire detail actions**

When detail emits an action:
- Notes `e` → open `$EDITOR` or inline → `UpdateTaskNotes`
- Checklist `x` → `CompleteChecklistItem` / `UncompleteChecklistItem`
- Checklist `n` → input → `AddChecklistItem`
- Checklist `d` → `DeleteChecklistItem`
- Activity `a` → input → `AddActivity`

- [ ] **Step 4: Wire command palette**

Build command list based on context:
```go
func (a App) buildCommands() []Command {
	var cmds []Command
	// Always available
	cmds = append(cmds, Command{Name: "Go to Inbox", Action: ...})
	// ... navigation commands ...

	// If task is selected
	if task := a.list.SelectedTask(); task != nil {
		cmds = append(cmds, Command{Name: "Complete Task", Action: ...})
		// ... task commands ...
	}
	return cmds
}
```

- [ ] **Step 5: Verify full interaction loop works**

Start server: `atask serve &`
Register + login, export ATASK_TOKEN
Start TUI: `atask --server http://localhost:8080`

Manual test:
- Navigate sidebar (inbox/today/areas)
- Create a task
- Complete a task
- Open command palette
- Open detail view
- Add a comment

- [ ] **Step 6: Commit**

```bash
git add internal/tui/
git commit -m "feat: wire all pane interactions to API calls"
```

### Task 14: Add schedule picker and date input

**Files:**
- Modify: `internal/tui/list.go` or create `internal/tui/schedule.go`

The `s` key opens a small picker:
```
Schedule:
  → Inbox
    Today
    Someday
  ──────────
  Start Date: [          ]
```

- [ ] **Step 1: Implement schedule picker**

Simple overlay with 3 options + optional date input. Selecting an option calls `UpdateTaskSchedule`. If "Today" or "Someday" is chosen with a start date, also call `SetTaskStartDate`.

- [ ] **Step 2: Commit**

```bash
git add internal/tui/
git commit -m "feat: add schedule picker overlay"
```

### Task 15: Final integration test and polish

**Files:**
- Modify: various TUI files for polish

- [ ] **Step 1: Run full test suite**

Run: `go test -race ./...`
Expected: All tests pass

- [ ] **Step 2: Build and manual smoke test**

```bash
go build -o /tmp/atask ./cmd/atask

# Terminal 1: start server
/tmp/atask serve &

# Register, login, get token
TOKEN=$(curl -s -X POST http://localhost:8080/auth/register -H 'Content-Type: application/json' -d '{"email":"demo@atask.dev","password":"secret","name":"Demo"}' > /dev/null && curl -s -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"demo@atask.dev","password":"secret"}' | jq -r .token)

# Terminal 2: start TUI
ATASK_TOKEN=$TOKEN /tmp/atask
```

- [ ] **Step 3: Polish pass**

- Verify all key bindings work
- Check pane resizing on terminal resize
- Verify SSE live updates (create a task via curl while TUI is open)
- Check error handling (stop server while TUI is running)

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: polish TUI interactions and error handling"
```

### Task 16: Update README and CLAUDE.md

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update README**

Add a "TUI" section after "Quick Start":
```markdown
### Terminal UI

```bash
# Start the server
atask serve &

# Register and get a token
TOKEN=$(curl ... | jq -r .token)

# Start the TUI
ATASK_TOKEN=$TOKEN atask
```

Key shortcuts: Tab (panes), j/k (navigate), n (new), x (complete), : (commands), / (search), ? (help)
```

- [ ] **Step 2: Update CLAUDE.md**

Add TUI section:
```markdown
## TUI (internal/tui/)

- Root model in app.go coordinates three sub-models (sidebar, list, detail)
- Each pane has its own Update/View, root routes messages to focused pane
- API calls return tea.Cmd that produce typed messages (TasksLoadedMsg, etc.)
- SSE events arrive as SSEEventMsg and trigger targeted refreshes
- Overlays (palette, search, help, confirm) render over the layout
- Styles in styles.go, key bindings in keys.go, messages in messages.go
```

- [ ] **Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: update README and CLAUDE.md with TUI documentation"
```
