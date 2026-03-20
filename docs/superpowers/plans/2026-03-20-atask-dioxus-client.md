# atask Dioxus Desktop Client — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the atask desktop client as a Dioxus 0.7 WebView app, connecting to the Go REST API with real-time SSE updates.

**Architecture:** Three-pane layout (sidebar 240px, main content flex, detail panel 340px). State managed via Dioxus signals with optimistic updates. API client wraps reqwest; SSE handled via `use_coroutine`. All styling via CSS custom properties in `theme.css` — no inline styles, no animations except 80ms hover smoothing.

**Tech Stack:** Rust, Dioxus 0.7 (desktop/WebView), reqwest, serde, chrono, uuid, tokio

**Design Spec:** `docs/design_specs/DESIGN.md` — read before implementing any UI.
**Design CLAUDE.md:** `docs/design_specs/CLAUDE.md` — architectural rules and don'ts.
**Reference Mockup:** `docs/design_specs/atask-screens-validation.html` — open in browser to see target.
**CSS:** `docs/design_specs/theme.css` — copy to `atask-app/assets/theme.css` before starting.

---

## File Structure

```
atask-app/
├── Cargo.toml
├── Dioxus.toml
├── assets/
│   ├── theme.css                    ← copied from docs/design_specs/theme.css
│   └── fonts/
│       ├── AtkinsonHyperlegible-Regular.woff2
│       ├── AtkinsonHyperlegible-Bold.woff2
│       ├── AtkinsonHyperlegible-Italic.woff2
│       └── AtkinsonHyperlegible-BoldItalic.woff2
├── src/
│   ├── main.rs                      ← launch, context providers, app shell
│   ├── api/
│   │   ├── mod.rs                   ← re-exports
│   │   ├── client.rs                ← ApiClient struct, all HTTP methods
│   │   ├── types.rs                 ← Task, Project, Area, Tag, etc. (serde)
│   │   └── sse.rs                   ← SSE subscription via use_coroutine
│   ├── state/
│   │   ├── mod.rs                   ← re-exports, AppState context struct
│   │   ├── auth.rs                  ← token storage, login state
│   │   ├── tasks.rs                 ← task signals, optimistic update helpers
│   │   ├── projects.rs              ← project + section signals
│   │   ├── navigation.rs            ← active view, selected task, detail panel
│   │   └── command.rs               ← command palette open/query/results
│   ├── components/
│   │   ├── mod.rs                   ← re-exports
│   │   ├── sidebar.rs               ← sidebar with nav items, projects, areas
│   │   ├── toolbar.rs               ← toolbar with title, date, action buttons
│   │   ├── checkbox.rs              ← circular task checkbox, square checklist checkbox
│   │   ├── task_item.rs             ← single-line 32px task row
│   │   ├── task_meta.rs             ← right-aligned metadata pills
│   │   ├── task_detail.rs           ← 340px right panel with all fields
│   │   ├── section_header.rs        ← collapsible section divider
│   │   ├── new_task_inline.rs       ← "+ New Task" row that expands to input
│   │   ├── tag_pill.rs              ← colored badge (today, deadline, agent, etc.)
│   │   ├── checklist_item.rs        ← square checkbox + title for checklists
│   │   ├── activity_entry.rs        ← avatar + author + content for activity stream
│   │   ├── button.rs                ← primary/secondary/ghost/danger variants
│   │   ├── text_input.rs            ← standard + ghost variants
│   │   └── command_palette.rs       ← overlay with search + grouped commands
│   └── views/
│       ├── mod.rs                   ← re-exports
│       ├── today.rs                 ← Today view (amber checkboxes, evening section)
│       ├── inbox.rs                 ← Inbox view (triage actions on hover)
│       ├── upcoming.rs              ← Upcoming view (date-grouped sections)
│       ├── someday.rs               ← Someday view (flat list)
│       ├── logbook.rs               ← Logbook view (completed/cancelled, date-grouped)
│       ├── project.rs               ← Project view (sections, progress bar)
│       └── login.rs                 ← Login/register screen
```

---

## Phase 1: Scaffold + Static Shell (Tasks 1–4)

Produces a running app with the three-pane layout, hardcoded data, and all visual components. No API calls yet.

---

### Task 1: Project Scaffold

**Files:**
- Create: `atask-app/Cargo.toml`
- Create: `atask-app/Dioxus.toml`
- Create: `atask-app/src/main.rs`
- Create: `atask-app/src/api/mod.rs`
- Create: `atask-app/src/api/types.rs`
- Create: `atask-app/src/state/mod.rs`
- Create: `atask-app/src/components/mod.rs`
- Create: `atask-app/src/views/mod.rs`
- Copy: `docs/design_specs/theme.css` → `atask-app/assets/theme.css`

- [ ] **Step 1: Create Cargo.toml**

```toml
[package]
name = "atask"
version = "0.1.0"
edition = "2024"

[dependencies]
dioxus = { version = "0.7", features = ["desktop"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
chrono = { version = "0.4", features = ["serde"] }
uuid = { version = "1", features = ["v4", "serde"] }
reqwest = { version = "0.12", features = ["json"] }
tokio = { version = "1", features = ["full"] }
```

- [ ] **Step 2: Create Dioxus.toml**

```toml
[application]
name = "atask"
default_platform = "desktop"

[web.app]
title = "atask"

[[desktop.window]]
title = "atask"
width = 1080
height = 720
min_width = 640
min_height = 480
transparent = true
decorations = true
```

- [ ] **Step 3: Copy theme.css and download fonts**

```bash
cp docs/design_specs/theme.css atask-app/assets/theme.css
mkdir -p atask-app/assets/fonts
# Download Atkinson Hyperlegible woff2 files from Google Fonts or fontsource
# Place in atask-app/assets/fonts/
# For now, the CSS fallback (system-ui) will work without the font files
```

- [ ] **Step 4: Create api/types.rs with domain types**

All types from DESIGN.md §10. These are the API response types used throughout the app.

```rust
use chrono::{NaiveDate, NaiveDateTime};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TaskStatus {
    Pending,
    Completed,
    Cancelled,
}

impl Default for TaskStatus {
    fn default() -> Self {
        Self::Pending
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Schedule {
    Inbox,
    Anytime,
    Someday,
}

impl Default for Schedule {
    fn default() -> Self {
        Self::Inbox
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ActorType {
    Human,
    Agent,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ActivityType {
    Comment,
    ContextRequest,
    Reply,
    Artifact,
    StatusChange,
    Decomposition,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Task {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Notes")]
    pub notes: String,
    #[serde(rename = "Status")]
    pub status: i64,
    #[serde(rename = "Schedule")]
    pub schedule: i64,
    #[serde(rename = "StartDate")]
    pub start_date: Option<String>,
    #[serde(rename = "Deadline")]
    pub deadline: Option<String>,
    #[serde(rename = "CompletedAt")]
    pub completed_at: Option<String>,
    #[serde(rename = "CreatedAt")]
    pub created_at: String,
    #[serde(rename = "UpdatedAt")]
    pub updated_at: String,
    #[serde(rename = "Index")]
    pub index: i64,
    #[serde(rename = "TodayIndex")]
    pub today_index: Option<i64>,
    #[serde(rename = "ProjectID")]
    pub project_id: Option<String>,
    #[serde(rename = "SectionID")]
    pub section_id: Option<String>,
    #[serde(rename = "AreaID")]
    pub area_id: Option<String>,
    #[serde(rename = "LocationID")]
    pub location_id: Option<String>,
    #[serde(rename = "RecurrenceRule")]
    pub recurrence_rule: Option<RecurrenceRule>,
    #[serde(rename = "Tags", default)]
    pub tags: Option<Vec<String>>,
    #[serde(rename = "Deleted", default)]
    pub deleted: bool,
    #[serde(rename = "DeletedAt")]
    pub deleted_at: Option<String>,
}

// Go domain: StatusPending=0, StatusCompleted=1, StatusCancelled=2
// Go domain: ScheduleInbox=0, ScheduleAnytime=1, ScheduleSomeday=2
impl Task {
    pub fn is_completed(&self) -> bool {
        self.status == 1
    }

    pub fn is_cancelled(&self) -> bool {
        self.status == 2
    }

    pub fn is_today(&self) -> bool {
        self.today_index.is_some()
    }

    pub fn schedule_name(&self) -> &str {
        match self.schedule {
            0 => "Inbox",
            1 => "Anytime",
            2 => "Someday",
            _ => "Unknown",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RecurrenceRule {
    pub mode: String,
    pub interval: u32,
    pub unit: String,
    pub end: Option<RecurrenceEnd>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RecurrenceEnd {
    pub date: Option<String>,
    pub count: Option<u32>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Project {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Notes")]
    pub notes: Option<String>,
    #[serde(rename = "Status")]
    pub status: i64,
    #[serde(rename = "Schedule", default)]
    pub schedule: i64,
    #[serde(rename = "StartDate")]
    pub start_date: Option<String>,
    #[serde(rename = "Deadline")]
    pub deadline: Option<String>,
    #[serde(rename = "CompletedAt")]
    pub completed_at: Option<String>,
    #[serde(rename = "CreatedAt", default)]
    pub created_at: String,
    #[serde(rename = "UpdatedAt", default)]
    pub updated_at: String,
    #[serde(rename = "Index", default)]
    pub index: i64,
    #[serde(rename = "AreaID")]
    pub area_id: Option<String>,
    #[serde(rename = "Tags", default)]
    pub tags: Option<Vec<String>>,
    #[serde(rename = "AutoComplete", default)]
    pub auto_complete: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Section {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "ProjectID")]
    pub project_id: String,
    #[serde(rename = "Index")]
    pub index: i64,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Area {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Index")]
    pub index: i64,
    #[serde(rename = "Archived")]
    pub archived: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Tag {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Index")]
    pub index: i64,
    #[serde(rename = "ParentID")]
    pub parent_id: Option<String>,
    #[serde(rename = "Shortcut")]
    pub shortcut: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ChecklistItem {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "Title")]
    pub title: String,
    #[serde(rename = "Status")]
    pub status: i64,
    #[serde(rename = "TaskID")]
    pub task_id: String,
    #[serde(rename = "Index")]
    pub index: i64,
}

// Go domain: ChecklistPending=0, ChecklistCompleted=1
impl ChecklistItem {
    pub fn is_completed(&self) -> bool {
        self.status == 1
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Activity {
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "TaskID")]
    pub task_id: String,
    #[serde(rename = "ActorID")]
    pub actor_id: String,
    #[serde(rename = "ActorType")]
    pub actor_type: String,
    #[serde(rename = "Type")]
    pub activity_type: String,
    #[serde(rename = "Content")]
    pub content: String,
    #[serde(rename = "CreatedAt")]
    pub created_at: String,
}

/// SSE event from the server's /events/stream endpoint.
#[derive(Debug, Clone, Deserialize)]
pub struct SseEvent {
    pub entity_type: String,
    pub entity_id: String,
    pub actor_id: String,
    #[serde(flatten)]
    pub extra: serde_json::Value,
}

/// Wrapper for mutation responses: { "event": "task.created", "data": {...} }
#[derive(Debug, Clone, Deserialize)]
pub struct EventEnvelope<T> {
    pub event: String,
    pub data: T,
}
```

**API notes:**
- The Go API returns PascalCase JSON keys (e.g. `"ID"`, `"Title"`). The serde `rename` attributes handle this.
- Status integers: `0=pending, 1=completed, 2=cancelled`. Schedule integers: `0=inbox, 1=anytime, 2=someday`.
- **View/list endpoints** (`GET /views/*`, `GET /tasks`, `GET /projects`) return bare JSON arrays.
- **Mutation endpoints** (`POST /tasks`, `POST /tasks/{id}/complete`, etc.) return an event envelope: `{"event": "task.created", "data": {...}}`. Parse with `EventEnvelope<T>`.
- The `Notes` field is `String` (not `Option<String>`) — the Go API sends `""` for empty notes, not `null`.

**Deviations from DESIGN.md file structure:**
- `src/api/events.rs` renamed to `src/api/sse.rs` for clarity.
- `src/views/login.rs` and `src/state/auth.rs` added — not in DESIGN.md but required for authentication.

**Checkbox styling note:** DESIGN.md §5.1 shows an inline `style:` attribute for border-color. Per CLAUDE.md rules, use CSS class variants instead (`.checkbox.today`, `.checkbox.checked`) — no inline styles.

- [ ] **Step 5: Create module files (mod.rs stubs)**

`src/api/mod.rs`:
```rust
pub mod types;
```

`src/state/mod.rs`:
```rust
pub mod navigation;
```

`src/components/mod.rs`:
```rust
// Components will be added as they are built
```

`src/views/mod.rs`:
```rust
// Views will be added as they are built
```

- [ ] **Step 6: Create src/state/navigation.rs**

```rust
#[derive(Debug, Clone, PartialEq)]
pub enum ActiveView {
    Inbox,
    Today,
    Upcoming,
    Someday,
    Logbook,
    Project(String), // project ID
}

impl Default for ActiveView {
    fn default() -> Self {
        Self::Today
    }
}
```

- [ ] **Step 7: Create src/main.rs with app shell**

```rust
use dioxus::prelude::*;

mod api;
mod state;
mod components;
mod views;

use state::navigation::ActiveView;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    let active_view = use_signal(|| ActiveView::Today);
    let selected_task_id: Signal<Option<String>> = use_signal(|| None);

    // Provide global state via context
    use_context_provider(|| active_view);
    use_context_provider(|| selected_task_id);

    rsx! {
        document::Link { rel: "stylesheet", href: asset!("./assets/theme.css") }
        div { class: "app-frame",
            div { class: "sidebar",
                div { class: "sidebar-drag-region" }
                p { class: "sidebar-group-label", "atask" }
            }
            div { class: "app-main",
                div { class: "app-toolbar",
                    span { class: "app-view-title", "Today" }
                }
                div { class: "app-content",
                    p { "Tasks will appear here." }
                }
            }
        }
    }
}
```

- [ ] **Step 8: Verify it compiles and runs**

```bash
cd atask-app
cargo run
```

Expected: A window opens showing the Bone-ivory background with a sidebar area and "Tasks will appear here." in the main content. The warm neutral palette from `theme.css` should be visible.

- [ ] **Step 9: Commit**

```bash
git add atask-app/
git commit -m "feat: scaffold Dioxus desktop app with types and theme"
```

---

### Task 2: Sidebar Component

**Files:**
- Create: `atask-app/src/components/sidebar.rs`
- Modify: `atask-app/src/components/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create sidebar.rs**

Build the sidebar per DESIGN.md §5.4: 240px, translucent bg, nav items (Inbox/Today/Upcoming/Someday/Logbook), projects section, areas section. Clicking a nav item updates the `ActiveView` signal.

Refer to `docs/design_specs/atask-screens-validation.html` for exact visual structure. Use CSS classes from `theme.css` — the sidebar classes (`.sidebar`, `.sidebar-nav-item`, `.sidebar-group-label`, etc.) are already defined.

Key behaviors:
- Nav items show icon (as inline SVG) + label + optional badge count
- Active item gets `.active` class (bold + tinted background)
- Projects show colored dot (8×8 circle) + name + badge
- Areas show folder icon + name
- Groups separated by `.sidebar-separator` dividers

For this task, hardcode 3 projects and 2 areas. Badge counts can be hardcoded (e.g., Inbox: 3, Today: 5).

- [ ] **Step 2: Update components/mod.rs**

```rust
pub mod sidebar;
```

- [ ] **Step 3: Wire sidebar into main.rs App component**

Replace the placeholder sidebar div with the `Sidebar` component. The sidebar should read from the `ActiveView` context signal and call `active_view.set(...)` on click.

- [ ] **Step 4: Verify sidebar renders and navigation works**

```bash
cargo run
```

Expected: Sidebar shows all 5 nav items with icons, 3 projects with colored dots, 2 areas. Clicking "Inbox" highlights it and changes the view title in the toolbar. Clicking "Today" switches back.

- [ ] **Step 5: Commit**

```bash
git add atask-app/src/components/sidebar.rs atask-app/src/components/mod.rs atask-app/src/main.rs
git commit -m "feat: add sidebar with nav items, projects, and areas"
```

---

### Task 3: Core Components (Checkbox, TaskItem, TagPill, Button)

**Files:**
- Create: `atask-app/src/components/checkbox.rs`
- Create: `atask-app/src/components/task_item.rs`
- Create: `atask-app/src/components/task_meta.rs`
- Create: `atask-app/src/components/tag_pill.rs`
- Create: `atask-app/src/components/section_header.rs`
- Create: `atask-app/src/components/new_task_inline.rs`
- Create: `atask-app/src/components/button.rs`
- Modify: `atask-app/src/components/mod.rs`

- [ ] **Step 1: Create checkbox.rs**

Per DESIGN.md §5.1. Circular, 20×20px. Props: `checked: bool`, `today: bool` (amber border variant), `on_toggle: EventHandler<()>`. When checked: filled with accent color + SVG checkmark. Instant state change, no animation.

CSS classes are in `theme.css`: `.checkbox`, `.checkbox.checked`, `.checkbox.today`.

- [ ] **Step 2: Create tag_pill.rs**

Per DESIGN.md §5.8. Props: `label: String`, `variant: TagVariant` (default/accent/today/deadline/agent/success). Small inline badge with rounded full radius.

CSS classes: `.tag-pill`, `.tag-pill--today`, `.tag-pill--deadline`, `.tag-pill--agent`, etc.

- [ ] **Step 3: Create task_meta.rs**

Right-aligned metadata for a task row. Shows project pill, deadline, today badge, agent indicator. Props: `task: Task`, `projects: Vec<Project>` (to resolve project name from ID). Truncate after 3 items.

CSS: `.task-meta` is `flex-shrink: 0; margin-left: auto; white-space: nowrap;`

- [ ] **Step 4: Create task_item.rs**

Per DESIGN.md §5.2. Single-line 32px row. Props: `task: Task`, `selected: bool`, `today_view: bool`, `on_select: EventHandler<String>`, `on_complete: EventHandler<String>`.

Layout: checkbox + title (truncates with ellipsis) + task_meta (pinned right). All on one horizontal axis.

CSS classes: `.task-item`, `.task-item.selected`, `.task-title`, `.task-title.completed`.

- [ ] **Step 5: Create section_header.rs**

Per DESIGN.md §5.6. Props: `title: String`, `count: usize`, `collapsed: bool`, `on_toggle: EventHandler<()>`. Shows chevron + title + count + horizontal line.

- [ ] **Step 6: Create new_task_inline.rs**

Per DESIGN.md §5.7. Dashed circle + "New Task" text. On click, transforms to a text input. Enter creates task (emits `on_create: EventHandler<String>`), Escape cancels.

- [ ] **Step 6b: Create button.rs**

Per DESIGN.md §5.10. Four variants: primary, secondary, ghost, danger. Three sizes: sm, default, lg. Props: `label: String`, `variant: ButtonVariant`, `size: ButtonSize`, `on_click: EventHandler<()>`. Active state: `transform: scale(0.97)`.

CSS classes: `.btn`, `.btn--primary`, `.btn--ghost`, `.btn--danger`, `.btn--sm`, `.btn--lg`.

- [ ] **Step 7: Update components/mod.rs with all new modules**

```rust
pub mod sidebar;
pub mod checkbox;
pub mod task_item;
pub mod task_meta;
pub mod tag_pill;
pub mod section_header;
pub mod new_task_inline;
pub mod button;
```

- [ ] **Step 8: Verify all components compile**

```bash
cargo run
```

Expected: App still runs. Components aren't wired to views yet but compile without errors.

- [ ] **Step 9: Commit**

```bash
git add atask-app/src/components/
git commit -m "feat: add core components (checkbox, task_item, tag_pill, section_header)"
```

---

### Task 4: Today View + Toolbar

**Files:**
- Create: `atask-app/src/components/toolbar.rs`
- Create: `atask-app/src/views/today.rs`
- Modify: `atask-app/src/views/mod.rs`
- Modify: `atask-app/src/components/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create toolbar.rs**

Per DESIGN.md §5.5. 52px height, shows view icon + title + subtitle (date for Today). Right side: search and new task icon buttons. Read `ActiveView` from context to determine title/icon.

For Today: show ★ icon (amber) + "Today" + "Thursday, Mar 20" (use `chrono::Local::now()` for real date).
For Inbox: show tray icon + "Inbox".
For Project: show colored dot + project name + "4 / 12" progress.

- [ ] **Step 2: Create views/today.rs**

Per DESIGN.md §6.1. Renders a task list from a `Vec<Task>` signal. All checkboxes use amber variant (`today: true`). Includes an optional "This Evening" section divider. NewTaskInline at bottom.

For now, populate with 5 hardcoded tasks in a `use_signal`. Wire checkbox clicks to toggle the task's completed status in the signal.

Clicking a task sets the `selected_task_id` context signal.

- [ ] **Step 3: Update views/mod.rs**

```rust
pub mod today;
```

- [ ] **Step 4: Update components/mod.rs**

```rust
pub mod toolbar;
```

- [ ] **Step 5: Wire Today view and Toolbar into main.rs**

Replace placeholder content in `App` with `Toolbar` and a view switch:

```rust
div { class: "app-main",
    Toolbar {}
    div { class: "app-content",
        match *active_view.read() {
            ActiveView::Today => rsx! { TodayView {} },
            _ => rsx! { p { "View not implemented yet." } },
        }
    }
}
```

- [ ] **Step 6: Verify Today view renders with hardcoded tasks**

```bash
cargo run
```

Expected: Today view shows 5 tasks with amber circular checkboxes, "This Evening" section divider, "+ New Task" at bottom. Toolbar shows ★ Today with current date. Clicking a checkbox marks the task as completed (strikethrough). Clicking sidebar items switches the toolbar title.

- [ ] **Step 7: Commit**

```bash
git add atask-app/src/
git commit -m "feat: add Today view with toolbar and hardcoded tasks"
```

---

## Phase 2: Remaining Views (Tasks 5–8)

---

### Task 5: Inbox View

**Files:**
- Create: `atask-app/src/views/inbox.rs`
- Modify: `atask-app/src/views/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create views/inbox.rs**

Per DESIGN.md §6.2. Task list ordered by creation date (newest first). Standard checkboxes (no amber). Shows "Added 2 hours ago" style relative timestamps in task_meta. NewTaskInline at bottom.

Empty state: "Inbox Zero ✓" centered, success-green.

Hardcode 3 tasks for now.

- [ ] **Step 2: Wire into main.rs view switch**

Add `ActiveView::Inbox => rsx! { InboxView {} }` to the match.

- [ ] **Step 3: Verify**

```bash
cargo run
```

Expected: Click "Inbox" in sidebar → shows inbox tasks with relative timestamps. Click back to "Today" → shows Today view.

- [ ] **Step 4: Commit**

```bash
git add atask-app/src/views/inbox.rs atask-app/src/views/mod.rs atask-app/src/main.rs
git commit -m "feat: add Inbox view"
```

---

### Task 6: Upcoming + Someday + Logbook Views

**Files:**
- Create: `atask-app/src/views/upcoming.rs`
- Create: `atask-app/src/views/someday.rs`
- Create: `atask-app/src/views/logbook.rs`
- Modify: `atask-app/src/views/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create views/upcoming.rs**

Per DESIGN.md §6.3. Tasks grouped by `start_date` with date section headers. Follow the date formatting rules: "Tomorrow — Fri, Mar 21", "Saturday, Mar 22", etc. Empty state: "Nothing scheduled ahead."

Hardcode 4 tasks across 2-3 date groups.

- [ ] **Step 2: Create views/someday.rs**

Per DESIGN.md §6.4. Flat task list ordered by index. Standard checkboxes. Empty state: "No someday tasks. Everything is decided."

Hardcode 3 tasks.

- [ ] **Step 3: Create views/logbook.rs**

Per DESIGN.md §6.5. Completed and cancelled tasks grouped by completion date. Completed tasks: checked checkbox + strikethrough. Cancelled tasks: ✕ icon + strikethrough in tertiary color.

Hardcode 4 tasks (mix of completed and cancelled).

- [ ] **Step 4: Wire all into main.rs view switch**

- [ ] **Step 5: Verify all views render correctly**

```bash
cargo run
```

Expected: All 5 sidebar nav items produce the correct view with appropriate styling.

- [ ] **Step 6: Commit**

```bash
git add atask-app/src/views/
git commit -m "feat: add Upcoming, Someday, and Logbook views"
```

---

### Task 7: Task Detail Panel

**Files:**
- Create: `atask-app/src/components/task_detail.rs`
- Create: `atask-app/src/components/checklist_item.rs`
- Create: `atask-app/src/components/activity_entry.rs`
- Create: `atask-app/src/components/text_input.rs`
- Modify: `atask-app/src/components/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create text_input.rs**

Per DESIGN.md §5.11. Two variants: standard (bordered, shadow) and ghost (no border, large text for titles). Props: `value: String`, `placeholder: String`, `ghost: bool`, `on_change: EventHandler<String>`.

- [ ] **Step 2: Create checklist_item.rs**

Per DESIGN.md §5.1 (square variant). 16×16px, 4px radius. Props: `item: ChecklistItem`, `on_toggle: EventHandler<String>`.

- [ ] **Step 3: Create activity_entry.rs**

Per DESIGN.md §5.9. Avatar (28×28 circle) + author name + timestamp + content. Human avatar: accent bg + initial. Agent avatar: agent-tint bg + ✦ symbol.

- [ ] **Step 4: Create task_detail.rs**

Per DESIGN.md §5.3. 340px fixed-width panel. Sections top-to-bottom: title (ghost input), meta row (tag pills), fields (project, schedule, dates, tags), notes, checklist, activity stream.

For now, use hardcoded data to populate checklist items and activity entries matching the mockup.

Read `selected_task_id` from context. When it's `Some(id)`, find the task in the hardcoded list and display its details.

- [ ] **Step 5: Wire detail panel into main.rs**

Add the detail panel to the app shell, conditionally rendered:

```rust
div { class: "app-frame",
    Sidebar {}
    div { class: "app-main",
        Toolbar {}
        div { class: "app-content",
            // view switch here
        }
    }
    if selected_task_id.read().is_some() {
        TaskDetail {}
    }
}
```

CSS class `.detail-panel` in theme.css handles the 340px width and border-left.

- [ ] **Step 6: Verify detail panel opens on task click**

```bash
cargo run
```

Expected: Click a task → detail panel appears on the right (340px). Shows title, fields, notes, checklist items with square checkboxes, and activity entries. Press Escape or click ✕ to close.

- [ ] **Step 7: Commit**

```bash
git add atask-app/src/components/
git commit -m "feat: add task detail panel with checklist and activity"
```

---

### Task 8: Project View

**Files:**
- Create: `atask-app/src/views/project.rs`
- Modify: `atask-app/src/views/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create views/project.rs**

Per DESIGN.md §6.6. Shows tasks grouped by sections. Sectionless tasks at top. Each section is collapsible with SectionHeader. Each section has its own NewTaskInline. Toolbar shows project name + progress ("4 / 12") + thin progress bar.

Hardcode 3 sections with 2-4 tasks each, plus 2 sectionless tasks.

- [ ] **Step 2: Wire into main.rs**

```rust
ActiveView::Project(ref id) => rsx! { ProjectView { project_id: id.clone() } },
```

- [ ] **Step 3: Verify project view**

```bash
cargo run
```

Expected: Click a project in the sidebar → shows project view with sections, collapsible headers, task counts, progress bar. Detail panel still works when clicking tasks.

- [ ] **Step 4: Commit**

```bash
git add atask-app/src/views/project.rs atask-app/src/views/mod.rs atask-app/src/main.rs
git commit -m "feat: add Project view with sections and progress bar"
```

---

## Phase 3: API Integration (Tasks 9–11)

---

### Task 9: API Client

**Files:**
- Create: `atask-app/src/api/client.rs`
- Modify: `atask-app/src/api/mod.rs`

- [ ] **Step 1: Create api/client.rs**

Implement `ApiClient` struct wrapping `reqwest::Client`. Methods mirror the Go API:

```rust
impl ApiClient {
    pub fn new(base_url: &str) -> Self;
    pub fn set_token(&mut self, token: String);

    // Auth
    pub async fn login(&self, email: &str, password: &str) -> Result<String, ApiError>;
    pub async fn register(&self, email: &str, password: &str, name: &str) -> Result<(), ApiError>;

    // Views
    pub async fn list_inbox(&self) -> Result<Vec<Task>, ApiError>;
    pub async fn list_today(&self) -> Result<Vec<Task>, ApiError>;
    pub async fn list_upcoming(&self) -> Result<Vec<Task>, ApiError>;
    pub async fn list_someday(&self) -> Result<Vec<Task>, ApiError>;
    pub async fn list_logbook(&self) -> Result<Vec<Task>, ApiError>;

    // Tasks
    pub async fn create_task(&self, title: &str) -> Result<Task, ApiError>;
    pub async fn complete_task(&self, id: &str) -> Result<(), ApiError>;
    pub async fn cancel_task(&self, id: &str) -> Result<(), ApiError>;
    pub async fn delete_task(&self, id: &str) -> Result<(), ApiError>;
    pub async fn update_task_title(&self, id: &str, title: &str) -> Result<(), ApiError>;
    pub async fn update_task_notes(&self, id: &str, notes: &str) -> Result<(), ApiError>;
    pub async fn update_task_schedule(&self, id: &str, schedule: &str) -> Result<(), ApiError>;
    pub async fn move_task_to_project(&self, id: &str, project_id: Option<&str>) -> Result<(), ApiError>;

    // Projects
    pub async fn list_projects(&self) -> Result<Vec<Project>, ApiError>;
    pub async fn get_project(&self, id: &str) -> Result<Project, ApiError>;
    pub async fn list_tasks_by_project(&self, project_id: &str) -> Result<Vec<Task>, ApiError>;
    pub async fn list_sections(&self, project_id: &str) -> Result<Vec<Section>, ApiError>;

    // Areas
    pub async fn list_areas(&self) -> Result<Vec<Area>, ApiError>;

    // Tags
    pub async fn list_tags(&self) -> Result<Vec<Tag>, ApiError>;

    // Checklist
    pub async fn list_checklist(&self, task_id: &str) -> Result<Vec<ChecklistItem>, ApiError>;
    pub async fn complete_checklist_item(&self, task_id: &str, item_id: &str) -> Result<(), ApiError>;

    // Activity
    pub async fn list_activity(&self, task_id: &str) -> Result<Vec<Activity>, ApiError>;
}
```

Error type:
```rust
#[derive(Debug)]
pub enum ApiError {
    Network(reqwest::Error),
    Api { status: u16, message: String },
}
```

Mutation responses come wrapped in `{"event": "...", "data": {...}}`. Parse with `EventEnvelope<T>`.

- [ ] **Step 2: Update api/mod.rs**

```rust
pub mod client;
pub mod types;
```

- [ ] **Step 3: Verify compiles**

```bash
cargo run
```

- [ ] **Step 4: Commit**

```bash
git add atask-app/src/api/
git commit -m "feat: add API client with all endpoint methods"
```

---

### Task 10: State Layer + Data Loading

**Files:**
- Create: `atask-app/src/state/auth.rs`
- Create: `atask-app/src/state/tasks.rs`
- Create: `atask-app/src/state/projects.rs`
- Modify: `atask-app/src/state/mod.rs`
- Modify: `atask-app/src/main.rs`
- Modify: all view files to use real data

- [ ] **Step 1: Create state/auth.rs**

Signal for JWT token. On app start, check for stored token. Provide `ApiClient` via context.

```rust
pub struct AuthState {
    pub token: Option<String>,
    pub api: ApiClient,
}
```

- [ ] **Step 2: Create state/tasks.rs**

Task list signals + loading functions. One signal per view:

```rust
pub struct TaskState {
    pub inbox: Vec<Task>,
    pub today: Vec<Task>,
    pub upcoming: Vec<Task>,
    pub someday: Vec<Task>,
    pub logbook: Vec<Task>,
    pub loading: bool,
}
```

Optimistic update helpers per DESIGN.md §10:
```rust
pub fn optimistic_complete(&mut self, task_id: &str) { ... }
pub fn rollback_complete(&mut self, task_id: &str) { ... }
```

- [ ] **Step 3: Create state/projects.rs**

```rust
pub struct ProjectState {
    pub projects: Vec<Project>,
    pub sections: HashMap<String, Vec<Section>>,
    pub project_tasks: HashMap<String, Vec<Task>>,
}
```

- [ ] **Step 4: Update state/mod.rs**

```rust
pub mod auth;
pub mod navigation;
pub mod tasks;
pub mod projects;
```

- [ ] **Step 5: Wire data loading into main.rs**

On app start, fetch inbox/today counts and project list in parallel using `use_future`. Provide state signals via context. Replace hardcoded data in views with context-provided signals.

- [ ] **Step 6: Create views/login.rs**

Simple login form: email + password + submit. On success, store token and load data. Show this view when no token is present.

- [ ] **Step 7: Update each view to read from state signals instead of hardcoded data**

Each view reads its data from the corresponding signal in `TaskState`. Loading state shows a subtle loading indicator.

- [ ] **Step 8: Verify app loads real data from the API**

Start the Go backend (`make run` in the main repo), then:

```bash
cd atask-app
cargo run
```

Expected: Login screen appears. After login, sidebar shows real project names and counts. Today view shows real tasks from the API. Completing a task calls the API.

- [ ] **Step 9: Commit**

```bash
git add atask-app/src/
git commit -m "feat: add state layer and wire views to live API data"
```

---

### Task 11: SSE Real-Time Updates

**Files:**
- Create: `atask-app/src/api/sse.rs`
- Modify: `atask-app/src/api/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create api/sse.rs**

Use `use_coroutine` to maintain a long-lived connection to `GET /events/stream?topics=task.*,project.*`. Parse SSE events (text lines: `event:`, `data:`, `id:`). On each event, update the appropriate state signal.

Use `reqwest` with streaming response body. Read line-by-line. Reconnect on disconnect with last event ID.

- [ ] **Step 2: Wire SSE into main.rs**

Start the SSE coroutine after successful login. On event:
- `task.created` → refetch the relevant view
- `task.completed` → remove from current view, refetch logbook count
- `task.scheduled_today` → refetch today + inbox
- `project.created` → refetch projects

- [ ] **Step 3: Verify real-time updates**

Open the app. In another terminal, create a task via curl. The app should show the new task without manual refresh.

- [ ] **Step 4: Commit**

```bash
git add atask-app/src/api/sse.rs atask-app/src/api/mod.rs atask-app/src/main.rs
git commit -m "feat: add SSE subscription for real-time updates"
```

---

## Phase 4: Command Palette + Keyboard Shortcuts (Tasks 12–13)

---

### Task 12: Command Palette

**Files:**
- Create: `atask-app/src/state/command.rs`
- Create: `atask-app/src/components/command_palette.rs`
- Modify: `atask-app/src/state/mod.rs`
- Modify: `atask-app/src/components/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create state/command.rs**

```rust
pub struct CommandState {
    pub open: bool,
    pub query: String,
    pub selected_index: usize,
}
```

Define command registry: list of `Command { label, shortcut, category, action }`. Filter by fuzzy match on query. Categories: Navigation, Task Actions (context-aware), Creation.

- [ ] **Step 2: Create components/command_palette.rs**

Per DESIGN.md §7. Overlay: 560px centered, offset 20% from top. Backdrop with `rgba(0,0,0,0.15)`. Ghost input + grouped results with keyboard shortcut hints. Arrow keys navigate, Enter executes, Escape closes.

Appears/disappears instantly (no animation per motion policy).

- [ ] **Step 3: Wire into main.rs**

Render command palette overlay when `command_state.open` is true:

```rust
if command_state.read().open {
    CommandPalette {}
}
```

- [ ] **Step 4: Verify**

```bash
cargo run
```

Expected: ⌘K opens command palette. Type to filter commands. Arrow keys + Enter to execute. Escape to close.

- [ ] **Step 5: Commit**

```bash
git add atask-app/src/
git commit -m "feat: add command palette with fuzzy search and keyboard navigation"
```

---

### Task 13: Global Keyboard Shortcuts

**Files:**
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Add global keydown handler**

Per DESIGN.md §8. Handle all keyboard shortcuts in a single `onkeydown` handler on the app frame:

- `⌘K` → toggle command palette
- `⌘N` → new task
- `⌘1-5` → navigate to views
- `↑↓` → move task selection
- `Space` → toggle completion
- `Enter` → open detail panel
- `Escape` → close detail/palette
- `⌘T` → schedule for today
- `⌘⇧C` → complete task
- `⌫` → delete (with confirmation)

Use `event.meta_key()`, `event.shift_key()`, `event.key()` to detect combos.

- [ ] **Step 2: Verify keyboard shortcuts work**

```bash
cargo run
```

Expected: ⌘K opens palette. ⌘1 goes to Inbox. Arrow keys navigate tasks. Space completes. Escape closes panels.

- [ ] **Step 3: Commit**

```bash
git add atask-app/src/main.rs
git commit -m "feat: add global keyboard shortcuts"
```

---

## Summary

| Phase | Tasks | Outcome |
|-------|-------|---------|
| 1: Scaffold + Static Shell | 1–4 | Running app with three-pane layout, all components, Today view with hardcoded data |
| 2: Remaining Views | 5–8 | All 6 views + detail panel working with hardcoded data |
| 3: API Integration | 9–11 | Live data from Go backend, login, optimistic updates, SSE real-time |
| 4: Command Palette + Keys | 12–13 | ⌘K palette, full keyboard shortcut map |

Each phase produces a working, testable application. Phase 1+2 can be validated visually against the mockup without a running backend.
