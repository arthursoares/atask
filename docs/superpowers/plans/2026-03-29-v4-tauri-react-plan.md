# atask v4 Tauri/React — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local-first macOS task manager with Tauri (Rust) backend and React frontend, faithful to the HTML validation design reference.

**Architecture:** Rust owns SQLite (rusqlite) + sync engine. React owns UI state (Zustand) + rendering. Communication via Tauri `invoke()` commands. CSS ported verbatim from the HTML validation file.

**Tech Stack:** Tauri 2, React 19, TypeScript, Zustand, rusqlite, plain CSS

**Design Spec:** `docs/superpowers/specs/2026-03-29-v4-tauri-react-design.md`
**Visual Reference:** `docs/design_specs/atask-screens-validation.html`
**Measurements:** `docs/design_specs_v2/MEASUREMENTS.md`

**Working Directory:** `/Users/arthur.soares/Github/openthings` (new `atask-v4/` directory at root)

---

## Phase 1: Scaffold + Rust Backend

### Task 1: Tauri Project Scaffold

**Files:**
- Create: `atask-v4/` — full Tauri 2 + React + Vite project
- Create: `atask-v4/src-tauri/Cargo.toml`
- Create: `atask-v4/package.json`
- Create: `atask-v4/vite.config.ts`
- Create: `atask-v4/tsconfig.json`
- Create: `atask-v4/index.html`
- Create: `atask-v4/src/main.tsx` — minimal React entry
- Create: `atask-v4/src/App.tsx` — "atask v4 — scaffold works"

- [ ] **Step 1: Create Tauri project**

```bash
cd /Users/arthur.soares/Github/openthings
npm create tauri-app@latest atask-v4 -- --template react-ts
cd atask-v4
```

- [ ] **Step 2: Add rusqlite dependency**

In `atask-v4/src-tauri/Cargo.toml`, add:
```toml
[dependencies]
rusqlite = { version = "0.32", features = ["bundled"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
uuid = { version = "1", features = ["v4"] }
chrono = { version = "0.4", features = ["serde"] }
```

- [ ] **Step 3: Add Zustand to frontend**

```bash
npm install zustand
```

- [ ] **Step 4: Verify builds**

```bash
npm run tauri dev
```
Expected: Window opens with placeholder text.

- [ ] **Step 5: Commit**

```bash
git add atask-v4/
git commit -m "feat(v4): scaffold Tauri 2 + React + Vite project"
```

---

### Task 2: Database Layer (Rust)

**Files:**
- Create: `atask-v4/src-tauri/src/db.rs` — database init, migrations, helpers
- Modify: `atask-v4/src-tauri/src/main.rs` — wire DB into Tauri state

- [ ] **Step 1: Create db.rs with migrations**

```rust
// db.rs
use rusqlite::{Connection, Result};
use std::path::PathBuf;
use std::sync::Mutex;

pub struct Database {
    pub conn: Mutex<Connection>,
}

impl Database {
    pub fn new(path: PathBuf) -> Result<Self> {
        let conn = Connection::open(&path)?;
        conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")?;
        let db = Self { conn: Mutex::new(conn) };
        db.migrate()?;
        Ok(db)
    }

    pub fn new_in_memory() -> Result<Self> {
        let conn = Connection::open_in_memory()?;
        conn.execute_batch("PRAGMA foreign_keys=ON;")?;
        let db = Self { conn: Mutex::new(conn) };
        db.migrate()?;
        Ok(db)
    }

    fn migrate(&self) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute_batch(include_str!("migrations/001_schema.sql"))?;
        Ok(())
    }
}
```

- [ ] **Step 2: Create migration SQL**

Create `atask-v4/src-tauri/src/migrations/001_schema.sql`:
```sql
CREATE TABLE IF NOT EXISTS areas (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    "index" INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    color TEXT NOT NULL DEFAULT '',
    areaId TEXT REFERENCES areas(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    completedAt TEXT,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sections (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    projectId TEXT NOT NULL REFERENCES projects(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    collapsed INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    schedule INTEGER NOT NULL DEFAULT 0,
    startDate TEXT,
    deadline TEXT,
    completedAt TEXT,
    "index" INTEGER NOT NULL DEFAULT 0,
    todayIndex INTEGER,
    timeSlot TEXT,
    projectId TEXT REFERENCES projects(id),
    sectionId TEXT REFERENCES sections(id),
    areaId TEXT REFERENCES areas(id),
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL,
    syncStatus INTEGER NOT NULL DEFAULT 0,
    repeatRule TEXT
);

CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    "index" INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS taskTags (
    taskId TEXT NOT NULL REFERENCES tasks(id),
    tagId TEXT NOT NULL REFERENCES tags(id),
    PRIMARY KEY (taskId, tagId)
);

CREATE TABLE IF NOT EXISTS checklistItems (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    taskId TEXT NOT NULL REFERENCES tasks(id),
    "index" INTEGER NOT NULL DEFAULT 0,
    createdAt TEXT NOT NULL,
    updatedAt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pendingOps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    body TEXT,
    createdAt TEXT NOT NULL,
    synced INTEGER NOT NULL DEFAULT 0
);
```

- [ ] **Step 3: Wire DB into Tauri state**

In `main.rs`:
```rust
mod db;

use db::Database;
use std::path::PathBuf;
use tauri::Manager;

fn main() {
    tauri::Builder::default()
        .setup(|app| {
            let app_dir = app.path().app_data_dir().expect("app data dir");
            std::fs::create_dir_all(&app_dir)?;
            let db_path = app_dir.join("atask.sqlite");
            let database = Database::new(db_path).expect("init database");
            app.manage(database);
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error running tauri");
}
```

- [ ] **Step 4: Build and verify**

```bash
npm run tauri dev
```
Expected: App starts, creates `atask.sqlite` in app data dir.

- [ ] **Step 5: Commit**

```bash
git add atask-v4/src-tauri/
git commit -m "feat(v4): add rusqlite database with schema migrations"
```

---

### Task 3: Rust Models + Serde

**Files:**
- Create: `atask-v4/src-tauri/src/models.rs` — all structs with Serialize/Deserialize

- [ ] **Step 1: Create models.rs**

```rust
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Task {
    pub id: String,
    pub title: String,
    pub notes: String,
    pub status: i32,
    pub schedule: i32,
    pub start_date: Option<String>,
    pub deadline: Option<String>,
    pub completed_at: Option<String>,
    pub index: i32,
    pub today_index: Option<i32>,
    pub time_slot: Option<String>,
    pub project_id: Option<String>,
    pub section_id: Option<String>,
    pub area_id: Option<String>,
    pub created_at: String,
    pub updated_at: String,
    pub sync_status: i32,
    pub repeat_rule: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Project {
    pub id: String,
    pub title: String,
    pub notes: String,
    pub status: i32,
    pub color: String,
    pub area_id: Option<String>,
    pub index: i32,
    pub completed_at: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Area {
    pub id: String,
    pub title: String,
    pub index: i32,
    pub archived: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Section {
    pub id: String,
    pub title: String,
    pub project_id: String,
    pub index: i32,
    pub archived: bool,
    pub collapsed: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Tag {
    pub id: String,
    pub title: String,
    pub index: i32,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct TaskTag {
    pub task_id: String,
    pub tag_id: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct ChecklistItem {
    pub id: String,
    pub title: String,
    pub status: i32,
    pub task_id: String,
    pub index: i32,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AppState {
    pub tasks: Vec<Task>,
    pub projects: Vec<Project>,
    pub areas: Vec<Area>,
    pub sections: Vec<Section>,
    pub tags: Vec<Tag>,
    pub task_tags: Vec<TaskTag>,
    pub checklist_items: Vec<ChecklistItem>,
}
```

- [ ] **Step 2: Add `mod models;` to main.rs**

- [ ] **Step 3: Build**

```bash
cd atask-v4 && npm run tauri build -- --debug 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git commit -am "feat(v4): add Rust models with serde serialization"
```

---

### Task 4a: Rust Commands — load_all + create_task

**Files:**
- Create: `atask-v4/src-tauri/src/commands.rs` — Tauri invoke handlers
- Modify: `atask-v4/src-tauri/src/main.rs` — register commands

- [ ] **Step 1: Create commands.rs with load_all**

Read all 7 tables + taskTags join, map rows to model structs, return `AppState`. Use `rusqlite::params![]` and row mapping.

- [ ] **Step 2: Add create_task command**

Accept `CreateTaskParams` struct (title, notes?, schedule?, startDate?, deadline?, timeSlot?, projectId?, sectionId?, areaId?, tagIds?, repeatRule?). Generate UUID. Insert into tasks table. If `tagIds` provided, insert into taskTags. All in one transaction. Return created Task.

- [ ] **Step 3: Register commands + build + verify with devtools console**

```bash
npm run tauri dev
```
In console: `window.__TAURI__.core.invoke('load_all')` → `AppState` with empty arrays.

- [ ] **Step 4: Commit**

```bash
git commit -am "feat(v4): add load_all and create_task Rust commands"
```

---

### Task 4b: Rust Commands — complete_task with recurrence

**Files:**
- Modify: `atask-v4/src-tauri/src/commands.rs`

- [ ] **Step 1: Add complete_task**

Set status=1, completedAt=now. If `repeatRule` is set: parse JSON, compute next startDate based on type (fixed: from startDate, afterCompletion: from now), create new task row with same properties + new UUID + new startDate + carry tags. All in one transaction.

- [ ] **Step 2: Add cancel_task** — set status=2, completedAt=now
- [ ] **Step 3: Add reopen_task** — set status=0, clear completedAt
- [ ] **Step 4: Register, build, verify**
- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add complete/cancel/reopen with recurrence logic"
```

---

### Task 4c: Rust Commands — update, duplicate, delete, reorder, move

**Files:**
- Modify: `atask-v4/src-tauri/src/commands.rs`

- [ ] **Step 1: Add update_task** — partial update of any fields including tagIds (diff + replace)
- [ ] **Step 2: Add duplicate_task** — copy all fields + tags, new UUID
- [ ] **Step 3: Add delete_task** — cascade: delete checklistItems, taskTags, then task
- [ ] **Step 4: Add reorder_tasks** — accept `Vec<{id, index}>`, batch update indices
- [ ] **Step 5: Add set_today_index** — update todayIndex for Today view ordering
- [ ] **Step 6: Add move_task_to_section** — update sectionId (null to remove from section)
- [ ] **Step 7: Register all, build**
- [ ] **Step 8: Commit**

```bash
git commit -am "feat(v4): add update/duplicate/delete/reorder/move task commands"
```

---

### Task 5: Rust Commands — Project, Area, Section, Tag, Checklist CRUD

**Files:**
- Modify: `atask-v4/src-tauri/src/commands.rs`
- Modify: `atask-v4/src-tauri/src/main.rs` — register new commands

- [ ] **Step 1: Project commands** — create, update, complete, reopen, delete (cascade), move_to_area, reorder
- [ ] **Step 2: Area commands** — create, update, delete, toggle_archived, reorder
- [ ] **Step 3: Section commands** — create, update, delete, toggle_collapsed, toggle_archived, reorder
- [ ] **Step 4: Tag commands** — create (fail on dupe), update, delete (cascade taskTags), add_to_task, remove_from_task
- [ ] **Step 5: Checklist commands** — create, update title, toggle, delete, reorder
- [ ] **Step 6: Register all commands, build**
- [ ] **Step 7: Commit**

```bash
git commit -am "feat(v4): add project/area/section/tag/checklist Rust commands"
```

---

### Task 6: Rust Tests

**Files:**
- Create: `atask-v4/src-tauri/src/tests.rs`

- [ ] **Step 1: Test task CRUD** — create, load_all, complete, reopen, cancel, delete
- [ ] **Step 2: Test recurrence** — complete repeating task creates next occurrence
- [ ] **Step 3: Test project cascade** — delete project nullifies task.projectId
- [ ] **Step 4: Test tag uniqueness** — duplicate title returns error
- [ ] **Step 5: Test checklist** — create, toggle, delete
- [ ] **Step 6: Run tests**

```bash
cd atask-v4/src-tauri && cargo test
```

- [ ] **Step 7: Commit**

```bash
git commit -am "test(v4): add Rust unit tests for all commands"
```

---

## Phase 2: React Frontend — Foundation

### Task 7: CSS + Theme + Font

**Files:**
- Create: `atask-v4/src/theme.css` — verbatim from HTML validation `:root` block + all component classes
- Create: `atask-v4/src/app.css` — minimal app overrides
- Create: `atask-v4/public/fonts/` — 4 Atkinson Hyperlegible .ttf files
- Modify: `atask-v4/src/main.tsx` — import CSS

- [ ] **Step 1: Copy CSS** from `docs/design_specs/atask-screens-validation.html` lines 9-200 into `theme.css`. Add `@font-face` declarations for Atkinson Hyperlegible. Include scrollbar styling (`::-webkit-scrollbar`) and selection styling (`::selection { background: var(--accent-ring) }`).
- [ ] **Step 2: Copy font files** from v3 `atask-app/Sources/Resources/Fonts/` to `public/fonts/`
- [ ] **Step 3: Import CSS in main.tsx**, verify fonts render
- [ ] **Step 4: Commit**

```bash
git commit -am "feat(v4): add theme CSS and Atkinson Hyperlegible fonts"
```

---

### Task 8: TypeScript Types + Zustand Store

**Files:**
- Create: `atask-v4/src/types.ts` — all domain types matching Rust models
- Create: `atask-v4/src/store.ts` — Zustand store with computed selectors
- Create: `atask-v4/src/hooks/useTauri.ts` — typed invoke() wrappers

- [ ] **Step 1: Create types.ts** — Task, Project, Area, Section, Tag, TaskTag, ChecklistItem, AppState, RepeatRule

- [ ] **Step 2: Create useTauri.ts** — typed wrappers for every invoke() command

```typescript
import { invoke } from '@tauri-apps/api/core';
import type { AppState, Task } from '../types';

export async function loadAll(): Promise<AppState> {
  return invoke('load_all');
}
export async function createTask(params: CreateTaskParams): Promise<Task> {
  return invoke('create_task', { params });
}
// ... all commands
```

- [ ] **Step 3: Create store.ts** — Zustand store with:
  - Data state: tasks, projects, areas, sections, tags, taskTags, checklistItems
  - `tagsByTaskId: Map<string, Set<string>>` (rebuilt on load and tag mutations)
  - UI state: activeView, selectedTaskId, selectedTaskIds, expandedTaskId, showPalette, showSidebar, etc.
  - Computed selectors: inbox, today, todayMorning, todayEvening, upcoming, someday, logbook, tasksForProject
  - **Completed-today logic**: tasks completed today stay visible in their original view with strikethrough. Use `completedAt` date check.
  - **Tag filtering**: `activeTagFilters: Set<string>`. All computed views apply AND filter via `tagsByTaskId` lookup.
  - `loadAll()` action that calls useTauri and rebuilds state + tagsByTaskId
  - Mutation actions that call invoke() then optimistically update local state
  - **Context-aware createTask**: reads `activeView` to set defaults (Today→schedule=1, Inbox→schedule=0, Project→projectId)

- [ ] **Step 4: Wire store into App.tsx** — call `loadAll()` on mount

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add TypeScript types, Zustand store, Tauri invoke wrappers"
```

---

### Task 9: App Shell — Three-Pane Layout

**Files:**
- Modify: `atask-v4/src/App.tsx` — `.app-frame` flexbox shell
- Create: `atask-v4/src/components/Sidebar.tsx` — nav items, areas, projects
- Create: `atask-v4/src/components/Toolbar.tsx` — view title, buttons

- [ ] **Step 1: Build App.tsx** — `.app-frame` with sidebar (240px, toggleable via `showSidebar` state, hidden with `display:none`) + `.app-main` (flex:1) + detail panel (340px, conditional when task selected)
- [ ] **Step 2: Build Sidebar.tsx** — nav items with icons (SVG from HTML), badges, separators, area group labels, project dots. Active state. Click to switch view.
- [ ] **Step 3: Build Toolbar.tsx** — view title + icon, subtitle (Today shows date), toolbar buttons (search, new task, ⌘K)
- [ ] **Step 4: Verify** — app shows three-pane layout matching HTML validation
- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add three-pane app shell with sidebar and toolbar"
```

---

### Task 10: Core Components — Checkbox, Tag, SectionHeader, EmptyState

**Files:**
- Create: `atask-v4/src/components/CheckboxCircle.tsx`
- Create: `atask-v4/src/components/CheckboxSquare.tsx`
- Create: `atask-v4/src/components/Tag.tsx`
- Create: `atask-v4/src/components/SectionHeader.tsx`
- Create: `atask-v4/src/components/DateGroupHeader.tsx`
- Create: `atask-v4/src/components/EmptyState.tsx`
- Create: `atask-v4/src/components/ProgressBar.tsx`
- Create: `atask-v4/src/components/AgentIndicator.tsx`
- Create: `atask-v4/src/components/Button.tsx`
- Create: `atask-v4/src/components/ContextMenu.tsx`

- [ ] **Step 1: CheckboxCircle** — 4 states: default, today, checked, cancelled. SVG checkmark. ✕ for cancelled.
- [ ] **Step 2: CheckboxSquare** — 2 states: unchecked, done.
- [ ] **Step 3: Tag** — 8 variants via `variant` prop. All CSS from `.tag-*` classes.
- [ ] **Step 4: SectionHeader** — title + count + line. Muted variant. Collapsible chevron.
- [ ] **Step 5: DateGroupHeader** — bold date + optional relative span.
- [ ] **Step 6: EmptyState, ProgressBar, AgentIndicator, Button** — small components from spec.
- [ ] **Step 7: ContextMenu** — reusable positioned dropdown with keyboard nav. Used by TaskRow, Sidebar, SectionHeader in later tasks.
- [ ] **Step 7: Commit**

```bash
git commit -am "feat(v4): add core UI components — checkbox, tag, section header, empty state"
```

---

## Phase 3: React Frontend — Task List Views

### Task 11: TaskRow + NewTaskRow

**Files:**
- Create: `atask-v4/src/components/TaskRow.tsx`
- Create: `atask-v4/src/components/NewTaskRow.tsx`

- [ ] **Step 1: TaskRow** — 32px row with CheckboxCircle + title (truncate) + meta (project pill, deadline, agent indicator, checklist count, tag pills). Hover bg. Selected bg. Click to select. Double-click to expand.
- [ ] **Step 2: Context menu** — right-click menu: Complete/Cancel, Schedule submenu, Move to Project, Duplicate, Delete. Use native browser `contextmenu` event with custom dropdown.
- [ ] **Step 3: NewTaskRow** — dashed plus circle + "New Task". Click → inline TextField. Enter → create task + collapse. Escape → cancel.
- [ ] **Step 4: Verify** with hardcoded data — rows render matching HTML validation
- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add TaskRow and NewTaskRow components"
```

---

### Task 12: TaskInlineEditor

**Files:**
- Create: `atask-v4/src/components/TaskInlineEditor.tsx`

- [ ] **Step 1: Expanded card** — accent border, sidebar-selected bg, padding per spec.
- [ ] **Step 2: Title input** (14px, plain) + notes textarea (13px, auto-grow). Title Enter → collapse. Empty title → delete.
- [ ] **Step 3: Attribute bar** — schedule pill (removable), project pill, action buttons (When, +Tag, Repeat, Project) with dashed borders. Each opens popover (stub for now).
- [ ] **Step 4: Wire expand/collapse** — only one open at a time. Click outside → collapse. Escape → collapse.
- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add TaskInlineEditor with attribute bar"
```

---

### Task 13: Views — Inbox, Today, Upcoming, Someday, Logbook

**Files:**
- Create: `atask-v4/src/views/InboxView.tsx`
- Create: `atask-v4/src/views/TodayView.tsx`
- Create: `atask-v4/src/views/UpcomingView.tsx`
- Create: `atask-v4/src/views/SomedayView.tsx`
- Create: `atask-v4/src/views/LogbookView.tsx`

- [ ] **Step 1: InboxView** — flat list of inbox tasks + hover triage actions (★ 📅 💤 📁). NewTaskRow.
- [ ] **Step 2: TodayView** — morning flat list + "This Evening" muted header + evening list. Amber checkboxes. NewTaskRow.
- [ ] **Step 3: UpcomingView** — date-grouped with DateGroupHeader. Relative dates.
- [ ] **Step 4: SomedayView** — flat list + NewTaskRow.
- [ ] **Step 5: LogbookView** — date-grouped. Completed: checked + strikethrough + "Completed 3h ago". Cancelled: ✕ + tag. Hover Reopen button.
- [ ] **Step 6: Wire view switching** — sidebar click + ⌘1-5.
- [ ] **Step 7: Verify** each view against HTML validation
- [ ] **Step 8: Commit**

```bash
git commit -am "feat(v4): add all list views — inbox, today, upcoming, someday, logbook"
```

---

### Task 14: ProjectView

**Files:**
- Create: `atask-v4/src/views/ProjectView.tsx`

- [ ] **Step 1: Toolbar extras** — progress bar + "Add Section" ghost button.
- [ ] **Step 2: Sectionless tasks** + NewTaskRow.
- [ ] **Step 3: Sections** — collapsible SectionHeader + tasks + NewTaskRow per section. Archived sections hidden.
- [ ] **Step 4: Section context menu** — Rename, Archive/Unarchive, Delete.
- [ ] **Step 5: Verify** against HTML validation project view.
- [ ] **Step 6: Commit**

```bash
git commit -am "feat(v4): add ProjectView with sections and progress bar"
```

---

## Phase 4: React Frontend — Detail Panel + Pickers

### Task 15: DetailPanel

**Files:**
- Create: `atask-v4/src/components/DetailPanel.tsx`
- Create: `atask-v4/src/components/ActivityFeed.tsx`
- Create: `atask-v4/src/components/ActivityEntry.tsx`
- Create: `atask-v4/src/components/ChecklistSection.tsx`

- [ ] **Step 1: DetailPanel shell** — 340px, border-left, header (title + meta), body (scrollable fields).
- [ ] **Step 2: Field rows** — Project, Schedule, Start Date, Deadline, Tags with editable values.
- [ ] **Step 3: Notes textarea + title input** — both support line breaks (textarea), 300ms debounce on save for both title and notes.
- [ ] **Step 4: ChecklistSection** — CheckboxSquare items, add new, toggle, drag to reorder (HTML5 drag within the list).
- [ ] **Step 5: ActivityFeed + ActivityEntry** — human/agent avatars, author, time, text, agent-card.
- [ ] **Step 6: Verify** against HTML validation detail panel.
- [ ] **Step 7: Commit**

```bash
git commit -am "feat(v4): add DetailPanel with checklist and activity feed"
```

---

### Task 16: Pickers — When, Tag, Project, Repeat, QuickMove

**Files:**
- Create: `atask-v4/src/components/pickers/WhenPicker.tsx`
- Create: `atask-v4/src/components/pickers/TagPicker.tsx`
- Create: `atask-v4/src/components/pickers/ProjectPicker.tsx`
- Create: `atask-v4/src/components/pickers/RepeatPicker.tsx`
- Create: `atask-v4/src/components/pickers/QuickMovePicker.tsx`
- Create: `atask-v4/src/lib/naturalDateParser.ts`
- Create: `atask-v4/src/lib/dateFormatting.ts`

- [ ] **Step 1: dateFormatting.ts** — relative dates, deadline formatting, section dates. Port from v3.
- [ ] **Step 2: naturalDateParser.ts** — "tomorrow", "next monday", "in 3 days". Port from v3.
- [ ] **Step 3: WhenPicker** — natural language input + quick options + date picker + clear.
- [ ] **Step 4: TagPicker** — searchable list, toggle, create new.
- [ ] **Step 5: ProjectPicker** — colored dots, checkmark, "No Project".
- [ ] **Step 6: RepeatPicker** — presets + custom + after-completion toggle.
- [ ] **Step 7: QuickMovePicker** — searchable, grouped by area. ⇧⌘M trigger.
- [ ] **Step 8: Wire pickers** into TaskInlineEditor attribute bar buttons.
- [ ] **Step 9: Commit**

```bash
git commit -am "feat(v4): add all picker popovers + date formatting utilities"
```

---

## Phase 5: React Frontend — Keyboard, Command Palette, Drag-and-Drop

### Task 17: Keyboard Shortcuts

**Files:**
- Create: `atask-v4/src/hooks/useKeyboard.ts`

- [ ] **Step 1: Global shortcuts** — ⌘K (palette), ⌘F (also opens palette — Quick Find is the same UI as Command Palette), ⌘N (new task), ⌘1-5 (navigate), ⌘/ (toggle sidebar), ⌘, (open settings as modal).
- [ ] **Step 2: List shortcuts** — ↑↓ navigate, Return open editor, Space new task, ⇧⌘C complete, ⌥⌘K cancel, ⌫ delete, ⌘D duplicate, ⌘T/E/R/O schedule, ⌘S when picker, ⇧⌘D deadline, ⇧⌘M move, ⇧⌘T tags, ⇧⌘R repeat, ⌘↑↓ reorder, ⌥⌘↑↓ move to top/bottom, Ctrl+]/[ date adjust, ⇧↑↓ extend selection, ⌘A select all.
- [ ] **Step 3: Editor shortcuts** — ⌘Return close, Escape close, ⇧⌘C new checklist.
- [ ] **Step 4: Context gating** — skip list shortcuts when `e.target` is INPUT/TEXTAREA. `e.preventDefault()` for ⌘S, ⌘D, ⌘E, ⌘R, ⌘O.
- [ ] **Step 5: Type Travel** — printable chars (no modifiers) → open palette with char as initial query.
- [ ] **Step 6: Verify** every shortcut works and doesn't interfere with text editing.
- [ ] **Step 7: Commit**

```bash
git commit -am "feat(v4): add keyboard shortcuts with context-aware gating"
```

---

### Task 18: Command Palette

**Files:**
- Create: `atask-v4/src/components/CommandPalette.tsx`

- [ ] **Step 1: Overlay** — backdrop + centered palette (560px, top 18%, radius-xl, shadow-popover).
- [ ] **Step 2: Input** — icon + text input (17px) + "⌘K" hint. Autofocus on open.
- [ ] **Step 3: Results** — grouped (Navigation, Task Actions, Create). Items: icon + label + shortcut. Active item highlight.
- [ ] **Step 4: Search** — commands + task titles + project names + notes content. Fuzzy match.
- [ ] **Step 5: Keyboard nav** — ↑↓ move active, Enter execute, Escape close.
- [ ] **Step 6: Verify** against HTML validation palette.
- [ ] **Step 7: Commit**

```bash
git commit -am "feat(v4): add command palette with search and keyboard navigation"
```

---

### Task 19: Drag and Drop

**Files:**
- Create: `atask-v4/src/components/dnd/DragPreview.tsx`
- Create: `atask-v4/src/components/dnd/DropIndicator.tsx`
- Modify: `atask-v4/src/components/TaskRow.tsx` — add draggable + drop target
- Modify: `atask-v4/src/components/Sidebar.tsx` — add drop targets on nav items

- [ ] **Step 1: DragPreview** — floating copy of task row during drag.
- [ ] **Step 2: DropIndicator** — 32px gap between rows at drop position.
- [ ] **Step 3: TaskRow drag** — HTML5 drag API. `onDragStart` sets task ID. `onDragOver` shows DropIndicator. `onDrop` calls `reorder_tasks`.
- [ ] **Step 4: Sidebar drops** — drag task to Inbox/Today/Someday → schedule change. Drag to project → move to project.
- [ ] **Step 5: Section header drag** — drag section to reorder. Drop task on header → move to section.
- [ ] **Step 6: Commit**

```bash
git commit -am "feat(v4): add drag-and-drop for task reorder and sidebar drops"
```

---

## Phase 6: Sync Engine + Settings

### Task 20: Rust Sync Engine

**Files:**
- Create: `atask-v4/src-tauri/src/sync.rs` — pending ops flush, SSE inbound, Tauri events
- Modify: `atask-v4/src-tauri/src/commands.rs` — add sync commands
- Modify: `atask-v4/src-tauri/src/main.rs` — start sync on setup

- [ ] **Step 1: Pending ops flush** — read unsynced ops, execute via HTTP, mark synced. Exponential backoff. Stop after 3 consecutive failures.
- [ ] **Step 2: SSE inbound** — connect to Go API event stream. Parse events. Upsert local SQLite. Emit `"store-changed"` Tauri event.
- [ ] **Step 3: SSE reconnect** — exponential backoff (5s→60s max).
- [ ] **Step 4: Sync commands** — `configure_sync`, `trigger_sync`, `get_sync_status`. **Local-only guard**: when no server URL configured, skip all sync, don't queue pendingOps.
- [ ] **Step 5: Initial sync** — fetch all entities from server, upsert by ID.
- [ ] **Step 6: Commit**

```bash
git commit -am "feat(v4): add Rust sync engine with SSE and pending ops flush"
```

---

### Task 21: Settings + Login + Sync UI

**Files:**
- Create: `atask-v4/src/components/SettingsPanel.tsx`
- Create: `atask-v4/src/components/LoginView.tsx`
- Create: `atask-v4/src/hooks/useSync.ts`

- [ ] **Step 1: useSync.ts** — listen for `"store-changed"` Tauri events, call `loadAll()` on change. Listen for `"sync-status-changed"` for UI indicator.
- [ ] **Step 2: SettingsPanel** — server URL, connection status, sign in/out, test connection, auto-archive picker.
- [ ] **Step 3: LoginView** — email/password, register/sign in toggle, error display.
- [ ] **Step 4: Sync status in toolbar** — spinner during sync, error indicator, pending ops count (only when server configured).
- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add settings, login, and sync status UI"
```

---

## Phase 7: Polish + Multi-Select + Final Testing

### Task 22: Multi-Select + Bulk Operations

**Files:**
- Modify: `atask-v4/src/store.ts` — `selectedTaskIds: Set<string>`, bulk actions
- Modify: `atask-v4/src/components/TaskRow.tsx` — Shift+click, multi-select visual
- Modify: `atask-v4/src/hooks/useKeyboard.ts` — ⇧↑↓, ⌘A

- [ ] **Step 1: Store** — `selectedTaskIds`, `effectiveSelection`, `bulkComplete`, `bulkDelete`, `bulkSetSchedule`, `clearSelection`.
- [ ] **Step 2: TaskRow** — Shift+click extends selection. Multi-select highlight.
- [ ] **Step 3: Keyboard** — ⇧↑↓ extends, ⌘A selects all in current view.
- [ ] **Step 4: Bulk operations** — when multiple selected, ⇧⌘C completes all, ⌫ deletes all, etc.
- [ ] **Step 5: Commit**

```bash
git commit -am "feat(v4): add multi-select with Shift+click and bulk operations"
```

---

### Task 23: Visual Verification Pass

- [ ] **Step 1: Compare every view** against `docs/design_specs/atask-screens-validation.html` pixel-by-pixel.
- [ ] **Step 2: Fix deviations** — spacing, colors, font sizes, hover states, border radii.
- [ ] **Step 3: Test all keyboard shortcuts** — verify none conflict with text editing.
- [ ] **Step 4: Test drag-and-drop** — reorder, sidebar drops, section drops.
- [ ] **Step 5: Test sync** — connect to Go API, create tasks, verify SSE updates.
- [ ] **Step 6: Final commit**

```bash
git commit -am "fix(v4): visual polish pass — pixel-perfect alignment with design spec"
```

---

## Summary

| Phase | Tasks | What |
|-------|-------|------|
| 1 | 1-6 | Scaffold + Rust backend (DB, models, commands split into 4a/4b/4c, tests) |
| 2 | 7-9 | CSS/theme, Zustand store, app shell (with sidebar toggle) |
| 3 | 10-14 | Core components (incl. ContextMenu) + all views |
| 4 | 15-16 | Detail panel (with checklist drag + debounce) + pickers |
| 5 | 17-19 | Keyboard (⌥⌘K, ⌘F=palette), command palette, drag-and-drop |
| 6 | 20-21 | Sync engine (local-only guard) + settings + login |
| 7 | 22-23 | Multi-select, visual polish |

**Total:** 25 tasks (Task 4 split into 4a/4b/4c). Phases 1 and 2 can run in parallel (Rust and React are independent until wired). Phase 3+ is sequential.
