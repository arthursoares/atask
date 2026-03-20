# Plan 2: All Views + Detail Panel

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete all 6 views and the task detail panel with editable fields, so every screen in the app is functional with real API data.

**Architecture:** Each view reads its newtype signal from context, renders TaskItems, handles create/complete via API. The detail panel opens on task selection and provides inline editing for all task fields. Picker components (schedule, project, date, tag) are built as needed by the detail panel.

**Tech Stack:** Rust, Dioxus 0.7, reqwest, chrono (same as Plan 1)

**Prerequisite:** Plan 1 complete — scaffold, API client, AppState newtypes, sidebar, toolbar, Today view all working.

**Working Directory:** `/Users/arthur.soares/Github/openthings/.worktrees/native-client-v2/atask-app`

**Lessons from v1 (STILL apply):**
1. Every task verified by running the app — not just `cargo build`
2. Signal reads inside `rsx!` only — use newtypes from `state::app`
3. Sequential execution — no parallel agents
4. Each view follows the Today view pattern exactly

---

## File Structure (new files only)

```
src/
├── views/
│   ├── inbox.rs                 ← Task 1
│   ├── upcoming.rs              ← Task 2
│   ├── someday.rs               ← Task 2
│   ├── logbook.rs               ← Task 3
│   └── project.rs               ← Task 4
├── components/
│   ├── task_detail.rs           ← Task 5
│   ├── schedule_picker.rs       ← Task 5 (inline in detail)
│   ├── project_picker.rs        ← Task 6
│   ├── date_picker.rs           ← Task 6
│   ├── tag_picker.rs            ← Task 6
│   └── checklist_item.rs        ← Task 5
└── state/
    └── app.rs                   ← Task 4 (add ProjectTasks newtype)
```

---

## Task 1: Inbox View

**Files:**
- Create: `src/views/inbox.rs`
- Modify: `src/views/mod.rs`
- Modify: `src/main.rs` (add match arm)

- [ ] **Step 1: Create views/inbox.rs**

Same pattern as Today view but:
- Reads `InboxTasks` from context (not `TodayTasks`)
- Standard checkboxes (`today_view: false`)
- Task metadata shows relative time ("Added 2 hours ago") — use `format_relative` from `date_fmt` on `created_at` field if available, or show nothing
- NewTaskInline creates task (default schedule is inbox, no need to change schedule)
- Empty state: "Inbox Zero ✓" with class `empty-state-success` (success green)
- On complete: mark completed + refetch via `api.list_inbox()`

Follow the exact TodayView pattern from `src/views/today.rs`:
```rust
let api: ApiSignal = use_context();
let mut inbox: InboxTasks = use_context();
let mut selected: SelectedTaskSignal = use_context();
// ... render tasks from inbox.0.read() INSIDE rsx!
```

- [ ] **Step 2: Wire into main.rs**

Add `ActiveView::Inbox => rsx! { views::inbox::InboxView {} }` to the match.

- [ ] **Step 3: Update views/mod.rs**

Add `pub mod inbox;`

- [ ] **Step 4: Ensure data loader fetches inbox**

Check `main.rs` data loader effect — it should already fetch inbox. Verify.

- [ ] **Step 5: Verify**

```bash
cargo run
```

1. Click "Inbox" in sidebar → Inbox view renders
2. If tasks exist: shown with standard checkboxes
3. If empty: "Inbox Zero ✓" in green
4. "+ New Task" creates a task that appears in the list
5. Completing a task removes it + sidebar badge decrements

- [ ] **Step 6: Commit**

```bash
git add src/views/
git commit -m "feat: add Inbox view with real data"
```

---

## Task 2: Upcoming + Someday Views

**Files:**
- Create: `src/views/upcoming.rs`
- Create: `src/views/someday.rs`
- Modify: `src/views/mod.rs`
- Modify: `src/main.rs`

- [ ] **Step 1: Create views/upcoming.rs**

Per DESIGN.md §6.3. Tasks grouped by `start_date`. Use `format_section_date` from `date_fmt` for section headers.

Pattern:
```rust
let api: ApiSignal = use_context();
let mut upcoming: UpcomingTasks = use_context();
let mut selected: SelectedTaskSignal = use_context();

// Group tasks by start_date
// For each date group: SectionHeader + tasks
```

Group tasks by `start_date` field. Tasks with no start_date shouldn't appear (the API view filters for this).

Empty state: "Nothing scheduled ahead."

No NewTaskInline (upcoming tasks come from setting start dates, not direct creation).

- [ ] **Step 2: Create views/someday.rs**

Per DESIGN.md §6.4. Flat list, same as Today but:
- Reads `SomedayTasks` from context
- Standard checkboxes (`today_view: false`)
- NewTaskInline: creates task, then sets schedule to "someday"
- Empty state: "No someday tasks. Everything is decided."

- [ ] **Step 3: Wire both into main.rs and views/mod.rs**

- [ ] **Step 4: Ensure data loader fetches upcoming + someday**

Check main.rs effect — add `list_upcoming()` and `list_someday()` if missing.

- [ ] **Step 5: Verify**

```bash
cargo run
```

1. Click "Upcoming" → shows date-grouped tasks or empty state
2. Click "Someday" → shows flat task list or empty state
3. Creating a task in Someday sets schedule correctly

- [ ] **Step 6: Commit**

```bash
git add src/
git commit -m "feat: add Upcoming and Someday views"
```

---

## Task 3: Logbook View

**Files:**
- Create: `src/views/logbook.rs`
- Modify: `src/views/mod.rs`
- Modify: `src/main.rs`

- [ ] **Step 1: Create views/logbook.rs**

Per DESIGN.md §6.5. Completed and cancelled tasks grouped by completion date.

Key differences from other views:
- Tasks are already completed — checkboxes shown as checked
- Completed tasks: checked checkbox (accent) + strikethrough title
- Cancelled tasks (status=2): "✕" icon instead of checkbox + strikethrough in tertiary color
- Grouped by `completed_at` date using `format_section_date`
- Clicking a completed checkbox calls `api.reopen_task(id)` to reopen it
- Empty state: "Nothing completed yet. Get started!"

Group by date:
```rust
// Parse completed_at, group into date buckets
// Sort dates descending (newest first)
// Render SectionHeader + tasks per group
```

- [ ] **Step 2: Wire into main.rs**

Add `ActiveView::Logbook => rsx! { views::logbook::LogbookView {} }`

- [ ] **Step 3: Ensure data loader fetches logbook**

Add `api_clone.list_logbook()` to the data loader in main.rs if missing.

- [ ] **Step 4: Verify**

```bash
cargo run
```

1. Click "Logbook" → shows completed/cancelled tasks grouped by date
2. Completed tasks have checked checkboxes + strikethrough
3. Cancelled tasks show ✕ icon
4. Empty state shows if no completed tasks
5. Clicking a checked checkbox reopens the task (it disappears from logbook)

- [ ] **Step 5: Commit**

```bash
git add src/
git commit -m "feat: add Logbook view with reopen support"
```

---

## Task 4: Project View

**Files:**
- Create: `src/views/project.rs`
- Modify: `src/views/mod.rs`
- Modify: `src/main.rs`
- Modify: `src/state/app.rs` (add ProjectTasks + ProjectSections newtypes)

- [ ] **Step 1: Add newtypes to state/app.rs**

```rust
use std::collections::HashMap;
use crate::api::types::Section;

#[derive(Clone, Copy)]
pub struct ProjectTasks(pub Signal<HashMap<String, Vec<Task>>>);

#[derive(Clone, Copy)]
pub struct ProjectSections(pub Signal<HashMap<String, Vec<Section>>>);
```

Add to main.rs context providers.

- [ ] **Step 2: Create views/project.rs**

Per DESIGN.md §6.6. Receives project ID from `ActiveView::Project(id)`.

On render (or via `use_effect` on project_id change):
- Fetch tasks: `api.list_tasks_by_project(project_id)`
- Fetch sections: `api.list_sections(project_id)`
- Store in `ProjectTasks` and `ProjectSections` hashmaps by project ID

Layout:
1. Sectionless tasks (no section_id) at top + NewTaskInline
2. For each section: SectionHeader + section tasks + NewTaskInline
3. Toolbar shows project name + progress ("4 / 12") — read from ProjectList

Task creation in project: create task, then `move_task_to_project(task_id, project_id)`. For section-specific NewTaskInline, also `move_task_to_section`.

Standard checkboxes. `show_project: false` on TaskItems (redundant).

- [ ] **Step 3: Wire into main.rs**

```rust
ActiveView::Project(ref id) => rsx! { views::project::ProjectView { project_id: id.clone() } },
```

- [ ] **Step 4: Update toolbar for project view**

The toolbar already handles `ActiveView::Project` from Plan 1 Task 4. Verify it shows the project name and consider adding progress info.

- [ ] **Step 5: Verify**

```bash
cargo run
```

1. Click a project in sidebar → project view renders
2. Shows tasks grouped by sections
3. Sectionless tasks at top
4. Each section is collapsible
5. NewTaskInline creates task in the project
6. Completing a task works

- [ ] **Step 6: Commit**

```bash
git add src/
git commit -m "feat: add Project view with sections and task management"
```

---

## Task 5: Task Detail Panel (Display + Core Editing)

**Files:**
- Create: `src/components/task_detail.rs`
- Create: `src/components/checklist_item.rs`
- Create: `src/components/schedule_picker.rs`
- Modify: `src/components/mod.rs`
- Modify: `src/main.rs`

This is the largest task. The detail panel must:
1. Display all task fields
2. Allow editing title, notes, schedule
3. Show checklist with add/toggle
4. Open when a task is selected, close on Escape or ✕

- [ ] **Step 1: Create checklist_item.rs**

Square checkbox (16x16, 4px radius) + title. Props: `title: String`, `checked: bool`, `on_toggle: EventHandler<()>`.

CSS: `.checklist-item`, `.checklist-check` (square variant). Check theme.css.

- [ ] **Step 2: Create schedule_picker.rs**

Three inline pills: Inbox | Today | Someday. Active one highlighted.

Props: `current: i64` (0/1/2), `on_change: EventHandler<String>` (emits "inbox"/"anytime"/"someday").

CSS: `.schedule-picker`, `.schedule-option`, `.schedule-option.active`. Add to theme.css if missing.

- [ ] **Step 3: Create task_detail.rs**

340px panel on the right side. Reads `SelectedTaskSignal` from context.

**Key pattern:** Track `last_loaded_id` to know when to reinitialize drafts:

```rust
let mut last_loaded_id: Signal<Option<String>> = use_signal(|| None);
let mut title_draft: Signal<String> = use_signal(|| String::new());
let mut notes_draft: Signal<String> = use_signal(|| String::new());
let mut checklist: Signal<Vec<ChecklistItem>> = use_signal(|| Vec::new());
let mut checklist_input: Signal<String> = use_signal(|| String::new());
```

**Finding the task:** Search across ALL task signals (inbox, today, upcoming, someday, logbook, project_tasks) to find the selected task by ID.

**Data fetching:** `use_effect` that fires when selected task ID changes — fetches checklist via `api.list_checklist(task_id)`.

**Sections (top to bottom):**
1. ✕ close button — sets `selected.0.set(None)`
2. Title — `input` with class `input input-ghost detail-title-input`, on blur saves via `api.update_task_title`
3. Schedule — `SchedulePicker` component, on change calls `api.update_task_schedule`
4. Fields (display only for now — pickers in Task 6):
   - PROJECT: show name or "None"
   - START DATE: show formatted date or "None"
   - DEADLINE: show formatted date or "None"
   - TAGS: show tag pills or empty
5. Notes — `textarea` with class `detail-notes-input`, on blur saves via `api.update_task_notes`
6. Checklist — list of `ChecklistItem` components + add input
7. After toggling checklist or saving, refetch to stay in sync

**CSS classes:** `.detail-panel`, `.detail-header`, `.detail-close`, `.detail-field`, `.detail-field-label`, `.detail-field-value`, `.detail-section`, `.detail-section-title`. Check theme.css.

- [ ] **Step 4: Wire into main.rs**

After the app-main div, conditionally render:
```rust
// Inside rsx!, read signal here:
if selected_task.0.read().is_some() {
    components::task_detail::TaskDetail {}
}
```

- [ ] **Step 5: Verify**

```bash
cargo run
```

1. Click a task in Today/Inbox → detail panel appears on right (340px)
2. Title is editable — change it, click away, it saves
3. Notes textarea — type, click away, saves
4. Schedule picker — click "Someday" → task moves to someday view
5. Checklist shows items (if any)
6. Add checklist item via input at bottom
7. Toggle checklist checkbox works
8. ✕ closes the panel

- [ ] **Step 6: Commit**

```bash
git add src/ assets/theme.css
git commit -m "feat: add task detail panel with title, notes, schedule, checklist editing"
```

---

## Task 6: Picker Components + Detail Panel Full Editing

**Files:**
- Create: `src/components/project_picker.rs`
- Create: `src/components/date_picker.rs`
- Create: `src/components/tag_picker.rs`
- Modify: `src/components/task_detail.rs`
- Modify: `src/components/mod.rs`

- [ ] **Step 1: Create project_picker.rs**

Dropdown list of projects from `ProjectList` context. Props: `current_project_id: Option<String>`, `on_select: EventHandler<Option<String>>`.

Shows all projects with colored dots. "None" option at top. Current highlighted. Click selects and closes.

CSS: `.picker-dropdown`, `.picker-item`, `.picker-item.active`. Add to theme.css if missing.

- [ ] **Step 2: Create date_picker.rs**

Native HTML date input. Props: `value: Option<String>`, `label: String`, `on_change: EventHandler<Option<String>>`.

Shows `input type="date"` + "Clear" link when set. Styled to match theme.

- [ ] **Step 3: Create tag_picker.rs**

Toggle list of tags from `TagList` context. Props: `task_id: String`, `current_tags: Vec<String>`, `on_add: EventHandler<String>`, `on_remove: EventHandler<String>`.

Shows all tags as pills, active ones highlighted. Click toggles.

- [ ] **Step 4: Wire pickers into task_detail.rs**

Update the detail panel to make fields interactive:
- PROJECT field: click opens `ProjectPicker`, on select calls `api.move_task_to_project`
- START DATE field: `DatePicker` component, on change calls `api.set_task_start_date`
- DEADLINE field: `DatePicker`, on change calls `api.set_task_deadline`
- TAGS field: show pills + "Add" button toggles `TagPicker`, on add/remove calls `api.add_task_tag`/`api.remove_task_tag`

Use `position: relative` on the field container (via CSS class `.detail-field-picker`) so picker dropdowns position correctly.

- [ ] **Step 5: Verify**

```bash
cargo run
```

1. Open detail panel for a task
2. Click PROJECT field → picker shows projects → select one → API called, field updates
3. Click START DATE → date picker → select date → saves
4. Click DEADLINE → same
5. Click "+ Add" on tags → tag picker → toggle tags → saves
6. All fields reflect real data after page refresh

- [ ] **Step 6: Commit**

```bash
git add src/ assets/theme.css
git commit -m "feat: add project, date, tag pickers — detail panel fully editable"
```

---

## Task 7: SSE Integration + View Refresh

**Files:**
- Create: `src/api/sse.rs`
- Modify: `src/api/mod.rs`
- Modify: `src/main.rs`

- [ ] **Step 1: Create api/sse.rs**

SSE stream parser using reqwest streaming response. Copy the verified structure from v1 if applicable.

```rust
pub struct SseParsedEvent {
    pub event_type: String,
    pub data: String,
    pub id: Option<String>,
}
```

Connect to `/events/stream?topics=task.*,project.*`. Parse SSE wire protocol (event/data/id lines, blank line = emit).

- [ ] **Step 2: Wire SSE into main.rs**

`use_coroutine` that:
1. Waits for token
2. Connects to SSE
3. On any task event: refetches the active view's data
4. On project event: refetches project list
5. Reconnects with 2s delay on disconnect

Simple approach: on any event, refetch the current view. Don't try surgical updates.

- [ ] **Step 3: Verify**

```bash
cargo run
```

1. Open the app showing Today view
2. In another terminal: `curl -X POST localhost:8080/tasks -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"title":"SSE test"}'` then schedule it for today
3. The task should appear in the app without manual refresh

- [ ] **Step 4: Commit**

```bash
git add src/api/ src/main.rs
git commit -m "feat: add SSE for real-time updates"
```

---

## Summary

| Task | What | Verification |
|------|------|-------------|
| 1. Inbox | Inbox view with create/complete | Navigate, create, complete tasks |
| 2. Upcoming + Someday | Date-grouped + flat views | Navigate, verify grouping |
| 3. Logbook | Completed tasks + reopen | View completed, reopen works |
| 4. Project | Section-grouped tasks | Navigate to project, sections work |
| 5. Detail Panel | Title, notes, schedule, checklist editing | Edit all fields, verify saves |
| 6. Pickers | Project, date, tag pickers in detail | All pickers open, select, save |
| 7. SSE | Real-time updates | External change appears without refresh |

**After Plan 2:** All 6 views working with real data, detail panel fully editable, SSE real-time sync. Plan 3 adds power features (command palette, keyboard shortcuts, drag-and-drop, context menus, settings).
