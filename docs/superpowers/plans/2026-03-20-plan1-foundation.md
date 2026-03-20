# Plan 1: Foundation — Scaffold, API Client, Test Harness, Today View

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working Dioxus desktop app that displays Today view with real API data, verified end-to-end by Playwright tests.

**Architecture:** Dioxus 0.7 WebView desktop app with a dual-build setup: `desktop` feature for the real app, `web` feature for Playwright testing. API client wraps reqwest. State managed via Dioxus signals using newtype wrappers for cross-component reactivity. CSS-only styling from `theme.css`.

**Tech Stack:** Rust, Dioxus 0.7 (desktop + web), reqwest, serde, chrono, tokio, Playwright (TypeScript)

**Design Spec:** `docs/design_specs/DESIGN.md`
**Design Rules:** `docs/design_specs/CLAUDE.md`
**Theme CSS:** `docs/design_specs/theme.css`
**Reference Mockup:** `docs/design_specs/atask-screens-validation.html`
**V1 reference branch:** `feature/native-client` — verified API client and types to copy from

**Prerequisites:** The Go API on `main` already has these endpoints (added earlier this session):
- `PUT /tasks/{id}/today-index` — set/clear today ordering
- `POST /tasks/{id}/reopen` — reopen completed/cancelled tasks
- `PUT /projects/{id}/color` — set project color
- `PUT /projects/{id}/sections/{sid}/reorder` — section reordering
All verified working against live server.

**Lessons from v1 (MUST follow):**
1. Every task ends with a verification step — either a test or a manual check with exact expected output
2. Signal reads MUST be inside `rsx!` for Dioxus reactivity
3. Use newtype wrappers for signals shared via context (e.g., `SelectedTask(Signal<Option<String>>)`)
4. Writes from `spawn` async blocks may not trigger parent re-renders — use newtypes
5. No parallel subagent dispatches for UI work
6. `[profile.dev.package."*"] opt-level = 2` in Cargo.toml for fast dev builds
7. API status integers: 0=pending, 1=completed, 2=cancelled. Schedule: 0=inbox, 1=anytime, 2=someday.
8. GET endpoints return bare JSON arrays. Mutations return `{"event":"...","data":{...}}`
9. No inline styles. No animations. CSS classes from theme.css only.

---

## File Structure

```
atask-app/
├── Cargo.toml                       ← dual features: desktop (default) + web
├── Dioxus.toml
├── CLAUDE.md                        ← Dioxus-specific rules (reactivity, API, styling)
├── assets/
│   ├── theme.css                    ← copied from docs/design_specs/theme.css
│   └── fonts/Atkinson_Hyperlegible/ ← TTF font files
├── src/
│   ├── main.rs                      ← launch, App component, context providers
│   ├── lib.rs                       ← re-exports for integration tests
│   ├── api/
│   │   ├── mod.rs
│   │   ├── client.rs                ← ApiClient (reqwest), all HTTP methods
│   │   └── types.rs                 ← serde structs matching Go API
│   ├── state/
│   │   ├── mod.rs
│   │   ├── app.rs                   ← AppState: newtype wrappers for all shared signals
│   │   ├── credentials.rs           ← token persistence (~/.config/atask/)
│   │   └── date_fmt.rs              ← relative date formatting helpers
│   ├── components/
│   │   ├── mod.rs
│   │   ├── sidebar.rs
│   │   ├── toolbar.rs
│   │   ├── checkbox.rs
│   │   ├── task_item.rs
│   │   ├── task_meta.rs
│   │   ├── tag_pill.rs
│   │   ├── section_header.rs
│   │   └── new_task_inline.rs
│   └── views/
│       ├── mod.rs
│       ├── today.rs
│       └── login.rs
├── tests/
│   └── api_integration.rs           ← Rust API integration tests
└── e2e/
    ├── package.json
    ├── playwright.config.ts
    └── tests/
        ├── helpers.ts
        └── today-view.spec.ts        ← Playwright tests for Today view
```

**Key design decision: Single `AppState` struct with newtypes.**

Instead of providing 6+ individual signals via context (which caused reactivity bugs in v1), we use a single `AppState` struct with newtype-wrapped signals:

```rust
// src/state/app.rs
#[derive(Clone, Copy)]
pub struct AppState {
    pub api: Signal<ApiClient>,
    pub token: TokenSignal,
    pub active_view: ViewSignal,
    pub selected_task: SelectedTaskSignal,
    pub tasks: TasksSignal,
    pub projects: ProjectsSignal,
    pub areas: AreasSignal,
}

// Newtypes for cross-component reactivity
#[derive(Clone, Copy)]
pub struct TokenSignal(pub Signal<Option<String>>);

#[derive(Clone, Copy)]
pub struct ViewSignal(pub Signal<ActiveView>);

#[derive(Clone, Copy)]
pub struct SelectedTaskSignal(pub Signal<Option<String>>);

#[derive(Clone, Copy)]
pub struct TasksSignal(pub Signal<TaskStore>);
// etc.
```

Each component gets `AppState` from context and accesses what it needs. All signal reads happen inside `rsx!`.

---

## Task 1: Project Scaffold + Build Verification

**Files:**
- Create: `atask-app/Cargo.toml`
- Create: `atask-app/Dioxus.toml`
- Create: `atask-app/CLAUDE.md`
- Create: `atask-app/src/main.rs`
- Create: `atask-app/src/lib.rs`
- Create: `atask-app/src/api/mod.rs`
- Create: `atask-app/src/api/types.rs`
- Create: `atask-app/src/state/mod.rs`
- Create: `atask-app/src/components/mod.rs`
- Create: `atask-app/src/views/mod.rs`
- Copy: `docs/design_specs/theme.css` → `atask-app/assets/theme.css`
- Copy: font files → `atask-app/assets/fonts/`

- [ ] **Step 1: Create Cargo.toml with dual-target features**

```toml
[package]
name = "atask"
version = "0.1.0"
edition = "2021"

[profile.dev.package."*"]
opt-level = 2

[profile.release]
lto = "thin"
codegen-units = 1

[features]
default = ["desktop"]
desktop = ["dioxus/desktop"]
web = ["dioxus/web"]

[dependencies]
dioxus = "0.7"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
chrono = { version = "0.4", features = ["serde"] }
reqwest = { version = "0.12", features = ["json", "stream"] }
tokio = { version = "1", features = ["full"] }
futures-util = "0.3"

[dev-dependencies]
uuid = { version = "1", features = ["v4"] }
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

- [ ] **Step 3: Copy assets**

```bash
cp docs/design_specs/theme.css atask-app/assets/theme.css
cp -r docs/design_specs/fonts/ atask-app/assets/fonts/ 2>/dev/null || true
# If fonts aren't in design_specs, copy from v1:
cp -r .worktrees/native-client/atask-app/assets/fonts/ atask-app/assets/fonts/ 2>/dev/null || true
```

Update font paths in `assets/theme.css` to use TTF:
```css
src: url('./fonts/Atkinson_Hyperlegible/AtkinsonHyperlegible-Regular.ttf') format('truetype');
```

- [ ] **Step 4: Create api/types.rs**

All serde structs matching the Go API. Copy the verified types from v1 (`../.worktrees/native-client/atask-app/src/api/types.rs`) — those were tested against the live API and all 7 integration tests passed.

Critical rules:
- Each field uses `#[serde(rename = "ID")]`, `#[serde(rename = "Title")]` etc. — explicit per-field renames matching Go's PascalCase field names
- All ID fields are `String` (not `Uuid`) — Go serializes UUIDs as strings
- Status and Schedule are `i64` (Go serializes `type Status int` as bare integers: 0, 1, 2)
- Go's embedded `Timestamps` struct flattens to top-level `CreatedAt`, `UpdatedAt` — handle with explicit fields
- Go's embedded `SoftDelete` flattens to `Deleted` (integer), `DeletedAt` — include with `#[serde(default)]`
- `Notes` is `String` not `Option<String>` (Go sends `""` not `null`)
- `Tags` is `Option<Vec<String>>` with `#[serde(default)]` (only hydrated on GET by ID)
- The v1 types.rs (at `.worktrees/native-client/atask-app/src/api/types.rs`) handles all of this correctly and passed 7 integration tests — copy it

- [ ] **Step 5: Create minimal src/main.rs**

```rust
use dioxus::prelude::*;
mod api;
mod state;
mod components;
mod views;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    rsx! {
        document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
        div { class: "app-frame",
            p { "atask v2 — scaffold works" }
        }
    }
}
```

With stub mod.rs files for api, state, components, views.

- [ ] **Step 6: Create src/lib.rs**

```rust
pub mod api;
pub mod state;
```

- [ ] **Step 7: Verify desktop build**

```bash
cd atask-app && cargo build
```

Expected: Compiles with warnings only (dead code). No errors.

- [ ] **Step 8: Verify the app runs**

```bash
cargo run
```

Expected: Window opens showing "atask v2 — scaffold works" with Bone-ivory background from theme.css.

- [ ] **Step 9: Commit**

```bash
git add atask-app/
git commit -m "feat: scaffold Dioxus app with dual desktop/web build targets"
```

---

## Task 2: API Client + Integration Tests

**Files:**
- Create: `atask-app/src/api/client.rs`
- Modify: `atask-app/src/api/mod.rs`
- Create: `atask-app/tests/api_integration.rs`

- [ ] **Step 1: Create api/client.rs**

Copy the verified client from v1 (`.worktrees/native-client/atask-app/src/api/client.rs`). It has all endpoints and passed 7 integration tests. Ensure it includes:

- `create_project`, `create_area`, `create_tag` methods
- `reorder_task`, `set_today_index`, `reopen_task` methods
- `add_task_tag`, `remove_task_tag` methods
- Helper methods: `get_json`, `post_json`, `post_action`, `put_json`, `delete_action`

- [ ] **Step 2: Update api/mod.rs**

```rust
pub mod client;
pub mod types;
```

- [ ] **Step 3: Create tests/api_integration.rs**

Copy from v1 — these tests are verified passing. Add tests for new endpoints:

```rust
#[tokio::test]
async fn test_today_index() {
    let client = setup_client().await;
    let task = client.create_task("Today index test").await.unwrap();
    client.set_today_index(&task.id, Some(3)).await.unwrap();
    // Verify task appears in today view
    client.update_task_schedule(&task.id, "anytime").await.unwrap();
    let today = client.list_today().await.unwrap();
    assert!(today.iter().any(|t| t.id == task.id));
    client.delete_task(&task.id).await.unwrap();
}

#[tokio::test]
async fn test_reopen_task() {
    let client = setup_client().await;
    let task = client.create_task("Reopen test").await.unwrap();
    client.complete_task(&task.id).await.unwrap();
    client.reopen_task(&task.id).await.unwrap();
    // Should be back in inbox
    let inbox = client.list_inbox().await.unwrap();
    assert!(inbox.iter().any(|t| t.id == task.id));
    client.delete_task(&task.id).await.unwrap();
}

#[tokio::test]
async fn test_project_color() {
    let client = setup_client().await;
    let project = client.create_project("Color test").await.unwrap();
    client.update_project_color(&project.id, "#4670a0").await.unwrap();
    let fetched = client.get_project(&project.id).await.unwrap();
    assert_eq!(fetched.color, "#4670a0");
}
```

- [ ] **Step 4: Start the Go server and run tests**

```bash
# Terminal 1: start Go server
cd /Users/arthur.soares/Github/openthings && go build -o /tmp/atask-server ./cmd/atask && /tmp/atask-server serve

# Terminal 2: run integration tests
cd atask-app && cargo test --test api_integration -- --test-threads=1
```

Expected: All tests pass (original 7 + new ones).

- [ ] **Step 5: Commit**

```bash
git add src/api/ tests/
git commit -m "feat: add API client with integration tests (all pass)"
```

---

## Task 3: AppState + Login + Credential Persistence

**Files:**
- Create: `atask-app/src/state/app.rs`
- Create: `atask-app/src/state/credentials.rs`
- Create: `atask-app/src/views/login.rs`
- Modify: `atask-app/src/state/mod.rs`
- Modify: `atask-app/src/views/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create state/app.rs with newtype wrappers**

Define ALL shared signals as newtypes upfront. This prevents the reactivity bugs from v1:

```rust
use dioxus::prelude::*;
use crate::api::client::ApiClient;
use crate::api::types::{Task, Project, Area, Tag};

// Navigation
#[derive(Debug, Clone, PartialEq)]
pub enum ActiveView {
    Inbox, Today, Upcoming, Someday, Logbook, Project(String),
}
impl Default for ActiveView { fn default() -> Self { Self::Today } }

// Newtype wrappers — these ensure Dioxus reactivity works across components
#[derive(Clone, Copy)]
pub struct TokenSignal(pub Signal<Option<String>>);

#[derive(Clone, Copy)]
pub struct ViewSignal(pub Signal<ActiveView>);

#[derive(Clone, Copy)]
pub struct SelectedTaskSignal(pub Signal<Option<String>>);

#[derive(Clone, Copy)]
pub struct InboxTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct TodayTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct UpcomingTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct SomedayTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct LogbookTasks(pub Signal<Vec<Task>>);

#[derive(Clone, Copy)]
pub struct ProjectList(pub Signal<Vec<Project>>);

#[derive(Clone, Copy)]
pub struct AreaList(pub Signal<Vec<Area>>);

#[derive(Clone, Copy)]
pub struct TagList(pub Signal<Vec<Tag>>);

#[derive(Clone, Copy)]
pub struct ApiSignal(pub Signal<ApiClient>);

#[derive(Clone, Copy)]
pub struct LoadingSignal(pub Signal<bool>);
```

- [ ] **Step 2: Create state/credentials.rs**

Copy from v1 — this was working. Saves/loads `~/.config/atask/credentials.json`.

- [ ] **Step 3: Create views/login.rs**

Login form. Key pattern from v1 — write to `TokenSignal` newtype, NOT a bare `Signal<Option<String>>`:

```rust
use crate::state::app::{TokenSignal, ApiSignal};

#[component]
pub fn LoginView() -> Element {
    let mut api: ApiSignal = use_context();
    let mut token: TokenSignal = use_context();
    // ... login form ...
    // On success:
    // api.0.write().set_token(tok.clone());
    // token.0.set(Some(tok));
}
```

- [ ] **Step 4: Update main.rs with AppState context providers**

```rust
fn App() -> Element {
    let saved = state::credentials::load();
    let mut initial_api = ApiClient::new(&api_url);
    if let Some(ref tok) = saved.token {
        initial_api.set_token(tok.clone());
    }

    let api = ApiSignal(use_signal(|| initial_api));
    let token = TokenSignal(use_signal(|| saved.token));
    let active_view = ViewSignal(use_signal(|| ActiveView::Today));
    let selected_task = SelectedTaskSignal(use_signal(|| None));
    let today_tasks = TodayTasks(use_signal(|| Vec::new()));
    let inbox_tasks = InboxTasks(use_signal(|| Vec::new()));
    // ... etc for all signals ...
    let loading = LoadingSignal(use_signal(|| false));

    // Provide ALL via context
    use_context_provider(|| api);
    use_context_provider(|| token);
    use_context_provider(|| active_view);
    use_context_provider(|| selected_task);
    use_context_provider(|| today_tasks);
    use_context_provider(|| inbox_tasks);
    use_context_provider(|| loading);
    // ... etc ...

    // Data loader effect
    use_effect(move || {
        if token.0.read().is_some() {
            let api_clone = api.0.read().clone();
            spawn(async move {
                loading.0.set(true);
                // fetch data, set signals
                loading.0.set(false);
            });
        }
    });

    rsx! {
        document::Link { rel: "stylesheet", href: asset!("/assets/theme.css") }
        // Token read INSIDE rsx!
        if token.0.read().is_none() {
            views::login::LoginView {}
        } else {
            div { class: "app-frame",
                p { "Logged in. Today view coming next." }
            }
        }
    }
}
```

- [ ] **Step 5: Verify login flow**

```bash
cargo run
```

1. Start the Go server
2. Run the app
3. Login screen appears
4. Register + login
5. See "Logged in. Today view coming next."
6. Quit, reopen — skips login (credential persistence)

- [ ] **Step 6: Commit**

```bash
git add src/
git commit -m "feat: add AppState with newtypes, login, credential persistence"
```

---

## Task 4: Sidebar + Toolbar + View Routing

**Files:**
- Create: `atask-app/src/components/sidebar.rs`
- Create: `atask-app/src/components/toolbar.rs`
- Modify: `atask-app/src/components/mod.rs`
- Modify: `atask-app/src/main.rs`

- [ ] **Step 1: Create sidebar.rs**

Read the sidebar spec in DESIGN.md §5.4. Build it with real data from `ProjectList` and `AreaList` context signals. Badge counts from `InboxTasks` and `TodayTasks`.

Key: all signal reads inside `rsx!`. Use newtypes from AppState.

```rust
let projects: ProjectList = use_context();
let areas: AreaList = use_context();
let active_view: ViewSignal = use_context();
let inbox_tasks: InboxTasks = use_context();
let today_tasks: TodayTasks = use_context();
```

Include "+ Project" and "+ Area" inline creation from the start.

- [ ] **Step 2: Create toolbar.rs**

Read DESIGN.md §5.5. Shows view-appropriate title/icon. For Today: ★ + "Today" + formatted date.

- [ ] **Step 3: Wire into main.rs**

Replace the "Logged in" placeholder with sidebar + toolbar + view placeholder:

```rust
div { class: "app-frame",
    Sidebar {}
    div { class: "app-main",
        Toolbar {}
        div { class: "app-content",
            p { "Today view placeholder" }
        }
    }
}
```

- [ ] **Step 4: Verify sidebar navigation**

```bash
cargo run
```

1. Login
2. Sidebar shows nav items with real project/area data from API
3. Clicking nav items changes toolbar title
4. Badge counts show real inbox/today counts
5. "+ Project" creates a project via API and appears in sidebar

- [ ] **Step 5: Commit**

```bash
git add src/
git commit -m "feat: add sidebar with real data, toolbar, view routing"
```

---

## Task 5: Core Task Components

**Files:**
- Create: `atask-app/src/components/checkbox.rs`
- Create: `atask-app/src/components/task_item.rs`
- Create: `atask-app/src/components/task_meta.rs`
- Create: `atask-app/src/components/tag_pill.rs`
- Create: `atask-app/src/components/section_header.rs`
- Create: `atask-app/src/components/new_task_inline.rs`
- Create: `atask-app/src/state/date_fmt.rs`
- Modify: `atask-app/src/components/mod.rs`
- Modify: `atask-app/src/state/mod.rs`

- [ ] **Step 1: Create state/date_fmt.rs**

Implement the date formatting rules from DESIGN.md §9.6:

```rust
pub fn format_relative_date(date_str: &str) -> String {
    // Parse YYYY-MM-DD, return:
    // "Today", "Tomorrow", "Yesterday", "Friday", "Last Monday", "Mar 25", "Mar 25, 2027"
}

pub fn format_deadline(date_str: &str) -> (String, &'static str) {
    // Returns (label, css_class):
    // ("Due Tomorrow", "deadline-normal")
    // ("Due Today", "deadline-today")
    // ("Overdue · Mar 18", "deadline-overdue")
}
```

Write unit tests for this — it's pure logic, easy to test:

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_today() { ... }
    #[test]
    fn test_format_tomorrow() { ... }
    #[test]
    fn test_format_overdue() { ... }
}
```

- [ ] **Step 2: Create checkbox.rs, tag_pill.rs**

Per DESIGN.md §5.1 and §5.8. These are simple, stateless display components.

- [ ] **Step 3: Create task_meta.rs**

Per DESIGN.md §5.2 metadata section. Uses `format_deadline` from date_fmt. Shows project pill, deadline, today badge, checklist indicator.

- [ ] **Step 4: Create task_item.rs**

Per DESIGN.md §5.2. 32px row, checkbox + title + meta. Includes grip handle for drag (visible on hover). Uses all the components above.

Important: `on_select` sets `SelectedTaskSignal`, `on_complete` calls API.

- [ ] **Step 5: Create section_header.rs and new_task_inline.rs**

Per DESIGN.md §5.6 and §5.7.

NewTaskInline: on Enter, calls `on_create` EventHandler. The parent view handles the API call.

- [ ] **Step 6: Verify all components compile**

```bash
cargo build
```

- [ ] **Step 7: Commit**

```bash
git add src/
git commit -m "feat: add core task components with date formatting"
```

---

## Task 6: Today View — End-to-End with Real Data

**Files:**
- Create: `atask-app/src/views/today.rs`
- Modify: `atask-app/src/views/mod.rs`
- Modify: `atask-app/src/main.rs`

This is the critical task — the first view that works end-to-end with real API data.

- [ ] **Step 1: Create views/today.rs**

Per DESIGN.md §6.1. Reads `TodayTasks` from context. Renders task list with amber checkboxes.

Key behaviors:
- Task completion: strikethrough instantly, stays visible (completed_at checked on next-day roll)
- NewTaskInline: creates task via API, sets schedule to "anytime", refreshes today list
- Task selection: sets `SelectedTaskSignal` (detail panel comes in Plan 2)

```rust
#[component]
pub fn TodayView() -> Element {
    let api: ApiSignal = use_context();
    let mut today: TodayTasks = use_context();
    let mut selected: SelectedTaskSignal = use_context();

    rsx! {
        div { class: "view-content",
            // Signal read INSIDE rsx!
            for task in today.0.read().iter() {
                TaskItem {
                    key: "{task.id}",
                    task: task.clone(),
                    selected: *selected.0.read() == Some(task.id.clone()),
                    today_view: true,
                    on_select: move |id: String| selected.0.set(Some(id)),
                    on_complete: move |id: String| {
                        let api_clone = api.0.read().clone();
                        spawn(async move {
                            let _ = api_clone.complete_task(&id).await;
                        });
                        // Refetch
                        let api_clone2 = api.0.read().clone();
                        spawn(async move {
                            if let Ok(tasks) = api_clone2.list_today().await {
                                today.0.set(tasks);
                            }
                        });
                    },
                }
            }
            NewTaskInline {
                on_create: move |title: String| {
                    let api_clone = api.0.read().clone();
                    spawn(async move {
                        if let Ok(task) = api_clone.create_task(&title).await {
                            let _ = api_clone.update_task_schedule(&task.id, "anytime").await;
                        }
                        // Refetch
                        if let Ok(tasks) = api_clone.list_today().await {
                            today.0.set(tasks);
                        }
                    });
                },
            }
        }
    }
}
```

- [ ] **Step 2: Wire into main.rs**

```rust
match *active_view.0.read() {
    ActiveView::Today => rsx! { views::today::TodayView {} },
    _ => rsx! { p { "View not implemented" } },
}
```

- [ ] **Step 3: Verify Today view manually**

```bash
cargo run
```

1. Login
2. Today view shows real tasks from API (or empty state if no today tasks)
3. Create a task via "+ New Task" — it appears in the list
4. Click checkbox — task shows as completed (strikethrough)
5. Click a task — it highlights as selected (detail panel comes later)
6. Sidebar today badge count matches

- [ ] **Step 4: Commit**

```bash
git add src/
git commit -m "feat: add Today view with real API data, task creation and completion"
```

---

## Task 7: Playwright API Workflow Tests

> Note: These test the Go API workflows that the UI depends on, NOT the Dioxus UI itself. True UI e2e tests (via Dioxus web build + DOM assertions) are deferred to Plan 2 once the web build is stable.

**Files:**
- Create: `atask-app/e2e/package.json`
- Create: `atask-app/e2e/playwright.config.ts`
- Create: `atask-app/e2e/tests/helpers.ts`
- Create: `atask-app/e2e/tests/today-view.spec.ts`
- Create: `atask-app/e2e/.gitignore`

- [ ] **Step 1: Create e2e infrastructure**

```json
// e2e/package.json
{
  "name": "atask-e2e",
  "private": true,
  "scripts": {
    "test": "npx playwright test",
    "test:ui": "npx playwright test --ui"
  },
  "devDependencies": {
    "@playwright/test": "^1.48.0"
  }
}
```

```typescript
// e2e/playwright.config.ts
import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './tests',
  timeout: 30000,
  use: { baseURL: 'http://localhost:8080' },
});
```

- [ ] **Step 2: Create helpers.ts**

```typescript
export async function registerAndLogin(request: any): Promise<string> {
  const email = `test-${Date.now()}@test.com`;
  await request.post('/auth/register', {
    data: { email, password: 'testpass', name: 'E2E Test' }
  });
  const resp = await request.post('/auth/login', {
    data: { email, password: 'testpass' }
  });
  const { token } = await resp.json();
  return token;
}

export function authHeaders(token: string) {
  return { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' };
}
```

- [ ] **Step 3: Create today-view.spec.ts**

Test the complete Today view workflow through the API:

```typescript
import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Today View Workflows', () => {
  let token: string;
  let headers: Record<string, string>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('create task and schedule for today', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Today test' } });
    const { data: task } = await resp.json();

    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();
  });

  test('complete task in today view', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Complete me today' } });
    const { data: task } = await resp.json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    await request.post(`/tasks/${task.ID}/complete`, { headers });

    // Should still appear in today (completed today, rolls to logbook tomorrow)
    // But /views/today only returns pending tasks, so it should NOT be there
    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeFalsy();

    // Should be in logbook
    const logbook = await request.get('/views/logbook', { headers });
    const completed = await logbook.json();
    expect(completed.some((t: any) => t.ID === task.ID)).toBeTruthy();
  });

  test('reopen completed task', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Reopen me' } });
    const { data: task } = await resp.json();

    await request.post(`/tasks/${task.ID}/complete`, { headers });
    await request.post(`/tasks/${task.ID}/reopen`, { headers });

    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();
  });

  test('set today index for ordering', async ({ request }) => {
    const r1 = await request.post('/tasks', { headers, data: { title: 'First' } });
    const { data: t1 } = await r1.json();
    const r2 = await request.post('/tasks', { headers, data: { title: 'Second' } });
    const { data: t2 } = await r2.json();

    await request.put(`/tasks/${t1.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${t2.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${t1.ID}/today-index`, { headers, data: { index: 1 } });
    await request.put(`/tasks/${t2.ID}/today-index`, { headers, data: { index: 0 } });

    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    const idx1 = tasks.findIndex((t: any) => t.ID === t1.ID);
    const idx2 = tasks.findIndex((t: any) => t.ID === t2.ID);
    expect(idx2).toBeLessThan(idx1); // t2 has lower today_index, appears first
  });

  test('SSE delivers task.created event', async ({ request }) => {
    const controller = new AbortController();
    const events: any[] = [];

    const sseResp = await fetch('http://localhost:8080/events/stream?topics=task.created', {
      headers: { 'Authorization': `Bearer ${token}` },
      signal: controller.signal
    });

    const reader = sseResp.body!.getReader();
    const decoder = new TextDecoder();
    const readPromise = (async () => {
      while (events.length < 1) {
        const { value, done } = await reader.read();
        if (done) break;
        const text = decoder.decode(value);
        for (const line of text.split('\n')) {
          if (line.startsWith('event:')) events.push({ type: line.replace('event: ', '').trim() });
          if (line.startsWith('data:') && events.length > 0) {
            events[events.length - 1].data = JSON.parse(line.replace('data: ', '').trim());
          }
        }
      }
    })();

    await new Promise(r => setTimeout(r, 500));
    await request.post('/tasks', { headers, data: { title: 'SSE test' } });
    await Promise.race([readPromise, new Promise(r => setTimeout(r, 3000))]);
    controller.abort();

    expect(events.length).toBeGreaterThan(0);
    expect(events[0].type).toBe('task.created');
    expect(events[0].data.entity_type).toBe('task');
  });
});
```

- [ ] **Step 4: Install and run Playwright tests**

```bash
cd e2e && npm install && npx playwright install chromium
npx playwright test
```

Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add e2e/
git commit -m "test: add Playwright e2e tests for Today view workflows"
```

---

## Summary

| Task | What It Produces | Verification |
|------|-----------------|--------------|
| 1. Scaffold | Compiling Dioxus app with theme | `cargo build` + window opens |
| 2. API Client | All HTTP methods + integration tests | `cargo test` — all pass |
| 3. AppState + Login | Auth flow with credential persistence | Login → persist → auto-login |
| 4. Sidebar + Toolbar | Real data navigation | Projects/areas from API, badge counts |
| 5. Task Components | Reusable UI building blocks | `cargo build` + date_fmt unit tests |
| 6. Today View | First fully working view | Create, complete, select tasks via UI |
| 7. API Workflow Tests | Playwright tests for API workflows | `npx playwright test` — all pass |

**After Plan 1:** A working app with login, sidebar, toolbar, and Today view — all backed by real API data, all verified by tests. Plan 2 adds remaining views and the detail panel. Plan 3 adds power features.
