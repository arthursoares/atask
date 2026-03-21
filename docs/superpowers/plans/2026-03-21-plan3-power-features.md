# Plan 3: Power Features — Command Palette, Keyboard Shortcuts, Checklist Counts

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add command palette (⌘K), global keyboard shortcuts, and checklist count indicators to make the app keyboard-driven and information-dense.

**Architecture:** Command palette is an overlay component with fuzzy search over a static command registry + dynamic task search. Keyboard shortcuts are handled by a single `onkeydown` on the app frame. Checklist counts require adding `checklist_count` and `open_checklist_count` fields to the Task API response.

**Tech Stack:** Same as Plans 1-2 (Rust, Dioxus 0.7, reqwest). Go API changes for checklist counts.

**Working Directory:** `/Users/arthur.soares/Github/openthings/.worktrees/native-client-v2`

**Prerequisite:** Plans 1-2 complete. 46 tests passing.

**Process rules (from v1 lessons):**
1. Sequential — one task at a time, verified before moving on
2. Signal reads inside `rsx!` — newtypes from `state::app`
3. Test each feature by running the app — not just `cargo build`
4. Playwright test for any API-side changes

---

## File Structure (new/modified)

```
# Go API (checklist counts)
internal/store/queries/tasks.sql         ← add checklist count subquery
internal/domain/task.go                  ← add ChecklistCount, OpenChecklistCount fields
internal/service/task_service.go         ← populate checklist counts

# Dioxus client
atask-app/src/
├── state/
│   └── app.rs                           ← add CommandOpen newtype
├── components/
│   ├── command_palette.rs               ← NEW: overlay with search + commands
│   └── task_meta.rs                     ← add checklist count display
├── api/
│   └── types.rs                         ← add checklist count fields to Task
└── main.rs                              ← add keyboard handler + command palette render
```

---

## Task 1: Checklist Count in API (Go backend)

Add `ChecklistCount` and `OpenChecklistCount` to the Task response so the client can show "3/5" without fetching per-task.

**Files:**
- Modify: `internal/domain/task.go`
- Modify: `internal/service/task_service.go`
- Modify: `internal/store/queries/tasks.sql` (add helper query)
- Test: `internal/service/task_service_test.go`

- [ ] **Step 1: Add fields to domain Task**

In `internal/domain/task.go`, add to the Task struct:
```go
ChecklistTotal int `json:"ChecklistTotal"`
ChecklistDone  int `json:"ChecklistDone"`
```

- [ ] **Step 2: Add sqlc query for checklist counts**

In `internal/store/queries/checklist_items.sql`, add:
```sql
-- name: CountChecklistByTask :one
SELECT
    COUNT(*) AS total,
    COUNT(CASE WHEN status = 1 THEN 1 END) AS done
FROM checklist_items
WHERE task_id = ? AND deleted = 0;
```

Run `make sqlc`.

- [ ] **Step 3: Hydrate counts in TaskService.Get**

In `taskFromRow`, the counts won't be available from the row (they come from a different table). Instead, hydrate in the `Get` method alongside tag hydration:

```go
func (s *TaskService) Get(ctx context.Context, id string) (*domain.Task, error) {
    row, err := s.queries.GetTask(ctx, id)
    if err != nil { return nil, err }
    task := taskFromRow(row)
    s.hydrateTags(ctx, task)
    // Hydrate checklist counts
    counts, err := s.queries.CountChecklistByTask(ctx, id)
    if err == nil {
        task.ChecklistTotal = int(counts.Total)
        task.ChecklistDone = int(counts.Done)
    }
    return task, nil
}
```

For list endpoints (views), hydrating per-task is expensive. Instead, add a batch approach — or accept that list endpoints don't include checklist counts. The client can show counts only in the detail panel, or fetch them lazily.

**Decision: Hydrate in Get only.** The client already fetches checklist items when opening the detail panel. For the task list metadata ("3/5"), we'll fetch counts lazily when the view loads (one query per task). This is acceptable for <100 visible tasks.

Actually, simpler: add a new endpoint `GET /tasks/{id}/checklist-count` that returns `{"total": 5, "done": 3}`. The client calls it per visible task.

Even simpler: **skip the API change entirely.** The client already fetches `list_checklist(task_id)` when you select a task. For the task list metadata, the client can batch-fetch counts or just show the indicator for tasks that have been viewed (cached).

Let's go with the simplest approach: **add ChecklistTotal/ChecklistDone to the GET /tasks/{id} response only.** The task list won't show counts — that's a future optimization.

- [ ] **Step 4: Run Go tests**

```bash
make sqlc && make test
```

- [ ] **Step 5: Add Playwright test**

```typescript
test('task has checklist counts after adding items', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Count test' } })).json();
    await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 1' } });
    await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 2' } });
    const { data: item } = await (await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 3' } })).json();
    await request.post(`/tasks/${task.ID}/checklist/${item.ID}/complete`, { headers });

    const resp = await request.get(`/tasks/${task.ID}`, { headers });
    const fetched = await resp.json();
    expect(fetched.ChecklistTotal).toBe(3);
    expect(fetched.ChecklistDone).toBe(1);
});
```

- [ ] **Step 6: Update Rust Task type**

In `atask-app/src/api/types.rs`, add:
```rust
#[serde(rename = "ChecklistTotal", default)]
pub checklist_total: i64,
#[serde(rename = "ChecklistDone", default)]
pub checklist_done: i64,
```

- [ ] **Step 7: Show in detail panel**

In `task_detail.rs`, show "3/5" next to the CHECKLIST section title when counts > 0:
```rust
div { class: "detail-section-title",
    "CHECKLIST"
    if task.checklist_total > 0 {
        span { class: "checklist-count-badge",
            " {task.checklist_done}/{task.checklist_total}"
        }
    }
}
```

- [ ] **Step 8: Commit**

```bash
git add internal/ atask-app/src/api/types.rs atask-app/src/components/task_detail.rs atask-app/e2e/
git commit -m "feat: add checklist counts to task API response and detail panel"
```

---

## Task 2: Command Palette State + Component

**Files:**
- Modify: `atask-app/src/state/app.rs`
- Create: `atask-app/src/components/command_palette.rs`
- Modify: `atask-app/src/components/mod.rs`

- [ ] **Step 1: Add CommandOpen newtype to state/app.rs**

```rust
#[derive(Clone, Copy)]
pub struct CommandOpen(pub Signal<bool>);

#[derive(Clone, Copy)]
pub struct CommandQuery(pub Signal<String>);
```

- [ ] **Step 2: Create command_palette.rs**

Per DESIGN.md §7. Overlay centered on screen, 560px wide.

**Command registry:**
```rust
struct Command {
    id: &'static str,
    label: &'static str,
    shortcut: Option<&'static str>,
    category: &'static str, // "Navigation", "Task Actions", "Creation"
}

const COMMANDS: &[Command] = &[
    Command { id: "nav-inbox",    label: "Go to Inbox",       shortcut: Some("⌘1"), category: "Navigation" },
    Command { id: "nav-today",    label: "Go to Today",       shortcut: Some("⌘2"), category: "Navigation" },
    Command { id: "nav-upcoming", label: "Go to Upcoming",    shortcut: Some("⌘3"), category: "Navigation" },
    Command { id: "nav-someday",  label: "Go to Someday",     shortcut: Some("⌘4"), category: "Navigation" },
    Command { id: "nav-logbook",  label: "Go to Logbook",     shortcut: Some("⌘5"), category: "Navigation" },
    Command { id: "new-task",     label: "New Task",          shortcut: Some("⌘N"), category: "Creation" },
    Command { id: "complete",     label: "Complete Task",     shortcut: Some("⌘⇧C"), category: "Task Actions" },
    Command { id: "schedule-today", label: "Schedule for Today", shortcut: Some("⌘T"), category: "Task Actions" },
    Command { id: "defer-someday", label: "Defer to Someday",  shortcut: None, category: "Task Actions" },
    Command { id: "move-inbox",   label: "Move to Inbox",     shortcut: None, category: "Task Actions" },
    Command { id: "delete",       label: "Delete Task",       shortcut: Some("⌫"), category: "Task Actions" },
];
```

**Filtering:** Case-insensitive substring match on label.

**Keyboard within palette:**
- `↑↓` — move selection
- `Enter` — execute selected command
- `Escape` — close

**Context-aware:** Task Actions only show when a task is selected (`SelectedTaskSignal.0.read().is_some()`).

**Execution:** Each command maps to an action:
- Navigation: `active_view.0.set(ActiveView::Inbox)` etc.
- Task actions: API calls via `spawn`
- After execution: close palette

CSS classes from DESIGN.md §7: `.command-backdrop`, `.command-palette`, `.command-input`, `.command-results`, `.command-group-label`, `.command-item`, `.command-item.active`, `.command-item-shortcut`. Check theme.css, add if missing.

- [ ] **Step 3: Add to components/mod.rs**

```rust
pub mod command_palette;
```

- [ ] **Step 4: Verify compile**

```bash
cargo build
```

- [ ] **Step 5: Commit**

```bash
git add atask-app/src/
git commit -m "feat: add command palette component with command registry"
```

---

## Task 3: Wire Command Palette + Global Keyboard Shortcuts

**Files:**
- Modify: `atask-app/src/main.rs`
- Modify: `atask-app/assets/theme.css` (command palette CSS if missing)

- [ ] **Step 1: Add CommandOpen/CommandQuery to context providers in main.rs**

```rust
let command_open = CommandOpen(use_signal(|| false));
let command_query = CommandQuery(use_signal(|| String::new()));
use_context_provider(|| command_open);
use_context_provider(|| command_query);
```

- [ ] **Step 2: Add global onkeydown handler**

On the `app-frame` div, add `onkeydown`:

```rust
div {
    class: "app-frame",
    tabindex: 0,
    onkeydown: move |evt: Event<KeyboardData>| {
        handle_keydown(evt, command_open, command_query, active_view, selected_task, api, /* ... */);
    },
    // ... rest of app
}
```

**Shortcut handler logic:**

```rust
fn handle_keydown(evt, command_open, ...) {
    let key = evt.key();
    let meta = evt.modifiers().contains(Modifiers::META);
    let shift = evt.modifiers().contains(Modifiers::SHIFT);

    // If palette is open, only handle Escape (palette handles its own keys)
    if *command_open.0.read() {
        if key == Key::Escape {
            command_open.0.set(false);
            command_query.0.set(String::new());
        }
        return;
    }

    // ⌘K — toggle command palette
    if meta && key == Key::Character("k".into()) {
        evt.prevent_default();
        command_open.0.set(true);
        return;
    }

    // ⌘N — new task
    if meta && !shift && key == Key::Character("n".into()) {
        evt.prevent_default();
        // Create task via API, refresh inbox
        return;
    }

    // ⌘1-5 — navigation
    if meta && !shift {
        let nav = match &key {
            Key::Character(c) if c == "1" => Some(ActiveView::Inbox),
            Key::Character(c) if c == "2" => Some(ActiveView::Today),
            Key::Character(c) if c == "3" => Some(ActiveView::Upcoming),
            Key::Character(c) if c == "4" => Some(ActiveView::Someday),
            Key::Character(c) if c == "5" => Some(ActiveView::Logbook),
            _ => None,
        };
        if let Some(view) = nav {
            evt.prevent_default();
            active_view.0.set(view);
            selected_task.0.set(None);
            return;
        }
    }

    // Escape — close detail panel
    if key == Key::Escape {
        if selected_task.0.read().is_some() {
            selected_task.0.set(None);
        }
        return;
    }

    // Task shortcuts (when selected)
    if let Some(ref tid) = *selected_task.0.read() {
        // ⌘⇧C — complete
        if meta && shift && key == Key::Character("c".into()) { ... }
        // ⌘T — schedule today
        if meta && !shift && key == Key::Character("t".into()) { ... }
        // Space — toggle completion
        if key == Key::Character(" ".into()) { ... }
        // Backspace — delete
        if key == Key::Backspace { ... }
    }

    // ↑↓ — arrow key navigation in task list
    // (move selection through current view's tasks)
}
```

- [ ] **Step 3: Render command palette overlay**

At the end of the `app-frame` div, inside `rsx!`:
```rust
if *command_open.0.read() {
    components::command_palette::CommandPalette {}
}
```

- [ ] **Step 4: Add command palette CSS if missing**

Check theme.css. Add:
```css
.command-backdrop { position:fixed; inset:0; background:rgba(0,0,0,0.15); z-index:1000; }
.command-palette { position:fixed; top:20%; left:50%; transform:translateX(-50%); width:560px; ... }
.command-input { width:100%; border:none; outline:none; font-size:var(--text-lg); padding:var(--sp-3); }
.command-results { max-height:400px; overflow-y:auto; }
.command-group-label { font-size:var(--text-xs); font-weight:700; color:var(--ink-tertiary); text-transform:uppercase; padding:var(--sp-2) var(--sp-3); }
.command-item { display:flex; align-items:center; justify-content:space-between; padding:var(--sp-2) var(--sp-3); cursor:pointer; }
.command-item:hover, .command-item.active { background:var(--accent-subtle); color:var(--accent); }
.command-item-shortcut { font-size:var(--text-xs); color:var(--ink-tertiary); font-family:'SF Mono',monospace; }
```

- [ ] **Step 5: Verify**

```bash
cargo run
```

1. ⌘K opens command palette, type to filter, Enter executes, Escape closes
2. ⌘1-5 navigates views
3. Arrow keys move task selection
4. Space completes selected task
5. Escape closes detail panel

- [ ] **Step 6: Commit**

```bash
git add atask-app/
git commit -m "feat: add command palette (⌘K) and global keyboard shortcuts"
```

---

## Task 4: Playwright Tests for New Features

**Files:**
- Create: `atask-app/e2e/tests/checklist-counts.spec.ts`
- Modify: existing test files if needed

- [ ] **Step 1: Checklist count test**

Already described in Task 1 Step 5. Verify it runs.

- [ ] **Step 2: Run all tests**

```bash
cd atask-app/e2e && npx playwright test
make test  # Go tests
cargo test --lib  # Rust unit tests
```

All must pass.

- [ ] **Step 3: Commit**

```bash
git add atask-app/e2e/
git commit -m "test: add checklist count Playwright test"
```

---

## Summary

| Task | What | Verification |
|------|------|-------------|
| 1. Checklist Counts | API returns counts, detail panel shows "3/5" | Playwright test + visual |
| 2. Command Palette | Overlay with search + commands | `cargo build` |
| 3. Keyboard Shortcuts | ⌘K, ⌘1-5, arrows, Space, Escape | Run app, test each shortcut |
| 4. Tests | Playwright for checklist counts | All 47+ tests pass |

**After Plan 3:** Keyboard-driven app with command palette. Plan 4: drag-and-drop, context menus, settings, local-first sync.
