# atask v4 — Tauri/React Desktop Client Design Spec

> **Status:** Design approved, pending implementation plan.
> **Supersedes:** v3 SwiftUI spec (`docs/superpowers/specs/2026-03-21-v3-swiftui-design.md`)
> **Visual reference:** `docs/design_specs/atask-screens-validation.html` (open in browser — this IS the target)
> **Measurements:** `docs/design_specs_v2/MEASUREMENTS.md`
> **CSS tokens:** `docs/design_specs/theme.css`

## Why v4 / Tauri+React

v1 (Dioxus) and v2 (Dioxus manual) failed on framework-level signal reactivity bugs. v3 (SwiftUI) failed on keyboard handling (`.onKeyPress` steals from TextFields), layout fights (`NavigationSplitView` column sizing), text rendering (white-on-beige), and the fundamental mismatch between CSS-defined design tokens and SwiftUI's Color/Font/Spacing enums.

Tauri+React eliminates these problems:
- **CSS is CSS.** The validation HTML already works — `theme.css` ports verbatim.
- **Keyboard handling is solved.** DOM `e.target` tells you exactly if a text field has focus. No `@FocusState` hacks.
- **Layout is flexbox.** Three-pane layout is `display: flex` exactly as the validation HTML renders it.
- **Font rendering.** Atkinson Hyperlegible via `@font-face` — no bundling issues.

## Architecture

```
┌──────────────────────────────────────┐
│           React (TypeScript)         │
│  ┌─────────────────────────────────┐ │
│  │  Zustand Store (in-memory)      │ │
│  │  - tasks, projects, areas, etc  │ │
│  │  - tagsByTaskId lookup map      │ │
│  │  - computed views (selectors)   │ │
│  │  - UI state (selection, expand) │ │
│  └──────────┬──────────────────────┘ │
│             │ invoke()                │
├─────────────┼────────────────────────┤
│           Tauri (Rust)               │
│  ┌──────────▼──────────────────────┐ │
│  │  Commands Layer (thick)         │ │
│  │  - Batched CRUD with validation │ │
│  │  - Recurrence on complete       │ │
│  │  - Cascade on delete            │ │
│  │  - Transaction per mutation     │ │
│  └──────────┬──────────────────────┘ │
│  ┌──────────▼──────────────────────┐ │
│  │  SQLite (rusqlite)              │ │
│  │  ~/Library/.../atask.sqlite     │ │
│  └──────────┬──────────────────────┘ │
│  ┌──────────▼──────────────────────┐ │
│  │  Sync Engine (Rust)             │ │
│  │  - pendingOps queue + flush     │ │
│  │  - SSE inbound → upsert local  │ │
│  │  - Exponential backoff          │ │
│  │  - Emits Tauri events to React  │ │
│  └─────────────────────────────────┘ │
└──────────────────────────────────────┘
```

### Data Flow

**Mutation:** React calls `invoke("create_task", { title, schedule, tagIds, ... })` → Rust validates, persists to SQLite in one transaction (including tags, pendingOp), returns the created record → React updates Zustand store instantly.

**Load:** On startup and after sync events, React calls `invoke("load_all")` → Rust returns flat `AppState` (all tables including `task_tags` join rows) → React rebuilds store + `tagsByTaskId` lookup map.

**Sync inbound:** Rust SSE thread receives server event → Rust fetches affected entity from Go API → Rust upserts into local SQLite → Rust emits `"store-changed"` Tauri event → React calls `invoke("load_all")` → store refreshes.

**Sync outbound:** Rust background thread periodically flushes `pendingOps` to Go API with exponential backoff. React is not involved.

### Key Design Decisions

1. **Rust owns all SQLite writes.** React never writes to the DB directly. Single-writer constraint is respected.
2. **Batched invoke() commands.** `create_task` accepts full payload including `tagIds`. No chatty IPC.
3. **Flat data return.** `load_all` returns raw `task_tags` join rows. React builds the `tagsByTaskId: Map<string, Set<string>>` for O(1) tag lookups.
4. **Repeating task logic in Rust.** Completing a repeating task creates the next occurrence in the same SQLite transaction.
5. **Zustand selectors with shallow equality.** Data state (tasks, projects) separated from UI state (selectedTaskId, expandedTaskId) to avoid unnecessary recomputation.

## CSS Tokens (complete set from HTML validation)

All tokens defined in `theme.css` / validation HTML. The spec references them by name; this is the full list for implementation.

```css
:root {
  /* Colors */
  --canvas:#f6f5f2; --canvas-elevated:#fefefe; --canvas-sunken:#eceae7;
  --sidebar-bg:rgba(238,236,231,0.72); --sidebar-hover:rgba(0,0,0,0.04);
  --sidebar-active:rgba(0,0,0,0.06); --sidebar-selected:rgba(70,112,160,0.10);
  --ink-primary:#222120; --ink-secondary:#686664; --ink-tertiary:#a09e9a;
  --ink-quaternary:#c8c6c2; --ink-on-accent:#fff;
  --accent:#4670a0; --accent-hover:#3a5f8a;
  --accent-subtle:rgba(70,112,160,0.10); --accent-ring:rgba(70,112,160,0.30);
  --today-star:#c88c30; --today-bg:rgba(200,140,48,0.08);
  --someday-tint:#8878a0;
  --deadline-red:#c04848; --deadline-bg:rgba(192,72,72,0.08);
  --success:#4a8860; --success-bg:rgba(74,136,96,0.08);
  --agent-tint:#7868a8; --agent-bg:rgba(120,104,168,0.07); --agent-border:rgba(120,104,168,0.20);
  --border:rgba(0,0,0,0.06); --border-strong:rgba(0,0,0,0.12); --separator:rgba(0,0,0,0.05);

  /* Spacing (4px base) */
  --sp-1:4px; --sp-2:8px; --sp-3:12px; --sp-4:16px; --sp-5:20px; --sp-6:24px;
  --sp-7:28px; --sp-8:32px; --sp-10:40px; --sp-12:48px; --sp-16:64px; --sp-20:80px;

  /* Radii */
  --radius-xs:4px; --radius-sm:6px; --radius-md:8px; --radius-lg:12px; --radius-xl:16px; --radius-full:9999px;

  /* Typography */
  --text-xs:11px; --text-sm:12px; --text-base:14px; --text-md:15px;
  --text-lg:17px; --text-xl:20px; --text-2xl:24px; --text-3xl:32px;
  --leading-tight:1.2; --leading-normal:1.5; --leading-relaxed:1.65;

  /* Layout */
  --sidebar-width:240px; --toolbar-height:52px;

  /* Shadows */
  --shadow-sm:0 1px 2px rgba(0,0,0,0.04), 0 1px 1px rgba(0,0,0,0.03);
  --shadow-md:0 2px 8px rgba(0,0,0,0.06), 0 1px 3px rgba(0,0,0,0.04);
  --shadow-lg:0 8px 30px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04);
  --shadow-popover:0 12px 40px rgba(0,0,0,0.12), 0 4px 12px rgba(0,0,0,0.06);

  /* Transitions (hover smoothing only — NO entrance/exit animations) */
  --ease-out:cubic-bezier(0.16,1,0.3,1); --ease-spring:cubic-bezier(0.34,1.56,0.64,1);
  --dur-fast:120ms; --dur-normal:200ms; --dur-slow:350ms;
  /* Only use: transition: all var(--dur-fast) var(--ease-out) on interactive hover states */
}
```

**Font files to bundle (4 variants):** AtkinsonHyperlegible-Regular.ttf, AtkinsonHyperlegible-Bold.ttf, AtkinsonHyperlegible-Italic.ttf, AtkinsonHyperlegible-BoldItalic.ttf

**Scrollbar styling:** `::-webkit-scrollbar` width 8px, transparent track, rounded thumb `rgba(0,0,0,0.12)` with 2px border. Port verbatim from validation HTML.

**Selection styling:** `::selection { background: var(--accent-ring) }`

## Tech Stack

| Layer | Choice | Why |
|-------|--------|-----|
| Shell | Tauri 2 | Native window, filesystem, system tray |
| Frontend | React 19 + TypeScript | Mature, fast iteration, CSS-native |
| State | Zustand | Minimal, computed selectors, no boilerplate |
| Styling | Plain CSS (theme.css) | Direct port from validation HTML. No Tailwind, no CSS-in-JS |
| Font | Atkinson Hyperlegible | Bundled via @font-face |
| DB | rusqlite | Simple, sync, mature. Wrapped in spawn_blocking |
| Sync | Rust (reqwest + eventsource) | Background thread, no React involvement |

## Database Schema

Identical to v3. 8 tables:

```sql
tasks          (id, title, notes, status, schedule, startDate, deadline, completedAt,
                index, todayIndex, timeSlot, projectId, sectionId, areaId,
                createdAt, updatedAt, syncStatus, repeatRule)
projects       (id, title, notes, status, color, areaId, index, completedAt, createdAt, updatedAt)
areas          (id, title, index, archived, createdAt, updatedAt)
sections       (id, title, projectId, index, archived, collapsed, createdAt, updatedAt)
tags           (id, title UNIQUE, index, createdAt, updatedAt)
taskTags       (taskId, tagId) — composite PK
checklistItems (id, title, status, taskId, index, createdAt, updatedAt)
pendingOps     (id AUTO, method, path, body, createdAt, synced)
```

**Domain constants:**
- `status`: 0=pending, 1=completed, 2=cancelled
- `schedule`: 0=inbox, 1=anytime, 2=someday
- `timeSlot`: null, "morning", "evening"
- `repeatRule`: JSON string in SQLite, parsed to typed object in React:
  ```typescript
  type RepeatRule = { type: "fixed" | "afterCompletion"; interval: number; unit: "day" | "week" | "month" | "year" }
  // Stored as JSON string in DB, returned as string by Rust, parsed in store.ts on load
  ```

## Rust Commands (Tauri invoke API)

### Data Loading
- `load_all() → AppState` — returns all tables flat, including taskTags join rows

### Task CRUD
- `create_task({ title, notes?, schedule?, startDate?, deadline?, timeSlot?, projectId?, sectionId?, areaId?, tagIds?, repeatRule? }) → Task`
- `update_task({ id, ...partial fields including tagIds? }) → Task`
- `complete_task(id) → Task` — sets status=1, completedAt=now. If repeating, creates next occurrence in same transaction.
- `cancel_task(id) → Task` — sets status=2, completedAt=now
- `reopen_task(id) → Task` — sets status=0, clears completedAt
- `duplicate_task(id) → Task` — copies all fields + tags, new id
- `delete_task(id)`
- `reorder_tasks(moves: [{id, index}])` — batch reorder
- `set_today_index(id, index)` — Today view ordering
- `move_task_to_section(taskId, sectionId?)` — move between sections

### Project CRUD
- `create_project({ title, color?, areaId? }) → Project`
- `update_project({ id, ...partial }) → Project`
- `complete_project(id)` / `reopen_project(id)`
- `delete_project(id)` — cascades: tasks get projectId=null, sections deleted
- `move_project_to_area(projectId, areaId?)`
- `reorder_projects(moves: [{id, index}])`

### Area CRUD
- `create_area({ title }) → Area`
- `update_area({ id, title })` / `delete_area(id)`
- `toggle_area_archived(id)`
- `reorder_areas(moves: [{id, index}])`

### Section CRUD
- `create_section({ title, projectId }) → Section`
- `update_section({ id, title? })` / `delete_section(id)` — tasks get sectionId=null
- `toggle_section_collapsed(id)` / `toggle_section_archived(id)`
- `reorder_sections(projectId, moves: [{id, index}])`

### Tag CRUD
- `create_tag({ title }) → Tag?` — returns null if duplicate
- `update_tag({ id, title })` / `delete_tag(id)` — cascades taskTags
- `add_tag_to_task(taskId, tagId)` / `remove_tag_from_task(taskId, tagId)`

### Checklist CRUD
- `create_checklist_item({ title, taskId }) → ChecklistItem`
- `update_checklist_item({ id, title })` — rename a checklist item
- `toggle_checklist_item(id)` / `delete_checklist_item(id)`
- `reorder_checklist_items(taskId, moves: [{id, index}])`

### Sync
- `configure_sync({ serverUrl, token? })`
- `trigger_sync()` — manual sync
- `get_sync_status() → { isSyncing, lastSyncAt?, lastError?, pendingOpsCount }`

## React Project Structure

```
src/
├── App.tsx                    — three-pane flexbox shell
├── main.tsx                   — entry, Zustand provider
├── store.ts                   — Zustand store + computed selectors
├── types.ts                   — Task, Project, Area, Section, Tag, etc.
├── theme.css                  — verbatim from design spec
├── app.css                    — app-specific overrides (minimal)
│
├── hooks/
│   ├── useKeyboard.ts         — global shortcuts with context zones
│   ├── useSync.ts             — listen for Tauri "store-changed" events
│   └── useTauri.ts            — typed invoke() wrappers
│
├── components/
│   ├── Sidebar.tsx            — nav items, areas, projects, badges, dots, separators
│   ├── Toolbar.tsx            — view title, subtitle, buttons, progress bar, "Add Section"
│   ├── TaskRow.tsx            — 32px row: checkbox + title + meta + hover actions
│   ├── TaskInlineEditor.tsx   — expanded card: title + notes + attribute bar
│   ├── NewTaskRow.tsx         — "+ New Task" with dashed circle
│   ├── DetailPanel.tsx        — 340px right panel: all fields + checklist + activity
│   ├── CommandPalette.tsx     — ⌘K overlay: search input + grouped results
│   ├── CheckboxCircle.tsx     — 20px circular: default, today (amber), checked, cancelled (✕)
│   ├── CheckboxSquare.tsx     — 16px square: for checklist items
│   ├── Tag.tsx                — pill with 8 variants: default, accent, today, deadline, agent, success, someday, cancelled
│   ├── SectionHeader.tsx      — bold title + count + line, muted variant for "This Evening"
│   ├── DateGroupHeader.tsx    — "Tomorrow — Fri, Mar 29" with optional relative span
│   ├── EmptyState.tsx         — icon + message
│   ├── ProgressBar.tsx        — thin bar + "4/12" text
│   ├── AgentIndicator.tsx     — "✦ In progress" with agent-tint
│   ├── ActivityFeed.tsx       — list of ActivityEntry items
│   ├── ActivityEntry.tsx      — avatar (human/agent) + author + time + text + agent-card
│   ├── Button.tsx             — base + ghost variant (12px bold, padding 6px 14px, radius-sm)
│   │
│   ├── pickers/
│   │   ├── WhenPicker.tsx     — popover: Today/Evening/Someday + calendar + natural language input + Clear
│   │   ├── TagPicker.tsx      — popover: searchable tag list + create new
│   │   ├── ProjectPicker.tsx  — popover: project list with colored dots + "No Project"
│   │   ├── RepeatPicker.tsx   — popover: daily/weekly/monthly/yearly/custom + after-completion toggle
│   │   └── QuickMovePicker.tsx — ⇧⌘M: searchable project list
│   │
│   └── dnd/
│       ├── DragPreview.tsx    — floating task preview during drag
│       └── DropIndicator.tsx  — 32px gap indicator at drop position
│
├── views/
│   ├── InboxView.tsx          — flat task list + hover triage actions (★ 📅 💤 📁)
│   ├── TodayView.tsx          — morning section + "This Evening" muted header + evening section
│   ├── UpcomingView.tsx       — date-grouped tasks with DateGroupHeader
│   ├── SomedayView.tsx        — flat task list
│   ├── LogbookView.tsx        — date-grouped completed/cancelled tasks
│   └── ProjectView.tsx        — sectionless tasks + sections (collapsible) + progress bar + "Add Section"
│
└── lib/
    ├── dateFormatting.ts      — relative dates, deadline formatting, section dates
    └── naturalDateParser.ts   — "tomorrow", "next monday", "in 3 days"
```

## Component Specifications

All measurements from `MEASUREMENTS.md`. All CSS classes from `atask-screens-validation.html`.

### App Shell (`App.tsx`)
- `display: flex; width: 100vw; height: 100vh` — `.app-frame` class
- Three children: `.sidebar` (240px fixed) + `.app-main` (flex: 1) + `.detail-panel` (340px, conditional)
- Detail panel visible when a task is selected

### Sidebar (`Sidebar.tsx`)
- Width: 240px, bg: `sidebar-bg` with `backdrop-filter: blur(28px) saturate(160%); -webkit-backdrop-filter: blur(28px) saturate(160%)` (webkit prefix required for Tauri WebView), border-right: `1px solid var(--border)`
- Traffic lights area: 52px height (Tauri handles native window controls)
- Nav items: `.sidebar-item` — icon (20px SVG), label, badge (right-aligned, 11px)
- Active state: `.sidebar-item.active` — bg `sidebar-active`, bold, ink-primary
- Separator: 1px, `separator` color, margin 8px 16px
- Group labels: `.sidebar-group-label` — 11px bold uppercase, ink-tertiary, letter-spacing 0.8px
- Project dots: `.sidebar-dot` — 8px circle with project color
- Drop targets: nav items and projects accept task drops
- Context menus: projects (rename, move to area, delete), areas (rename, delete)

### Toolbar (`Toolbar.tsx`)
- Height: 52px, padding: 0 24px, border-bottom: 1px separator
- Left: view icon + title (20px bold) + subtitle (12px tertiary) + progress bar (project view)
- Right: toolbar buttons (30x30, radius-sm) — Search, New Task, ⌘K palette trigger
- Project view extra: "Add Section" ghost button

### TaskRow (`TaskRow.tsx`)
- Height: 32px, gap: 12px, padding: 6px 16px, radius: 8px
- Hover: bg `sidebar-hover`. Selected: bg `sidebar-selected`
- Children: CheckboxCircle + title (14px, truncate) + meta (right-aligned)
- Meta: project pill (dot + name, bg canvas-sunken, radius-full), deadline (red if overdue), agent indicator, checklist count, tag pills
- Completed: title ink-tertiary, strikethrough with ink-quaternary
- Inbox hover actions: absolute positioned, 4 buttons (★ Today, 📅 Date, 💤 Someday, 📁 Project) — 26px, radius-sm, border, shadow-sm
- Draggable: shows DragPreview on drag start
- Drop target: shows DropIndicator between rows
- Context menu: Complete/Cancel, Schedule submenu, Move to Project submenu, Duplicate, Delete

### TaskInlineEditor (`TaskInlineEditor.tsx`)
> **Note:** The HTML validation file does NOT contain an inline editor example. Measurements below come from `MEASUREMENTS.md` lines 193-216. This is the one component without a pixel-perfect visual reference.

- Replaces TaskRow when `expandedTaskId === task.id`
- Background: `sidebar-selected` (accent 10%), border: 1.5px solid accent, radius: 8px
- Padding: 6px 16px 8px
- Top row (32px): CheckboxCircle + title TextField (14px — matches task row size; the v3 SwiftUI spec used 16px but the HTML validation uses 14px as the task title input size, so we follow the HTML)
- Notes: TextArea below title, 13px, ink-secondary, auto-grows
- Attribute bar: left padding 27px (checkbox 20 + gap 7)
  - Schedule pill: "★ Today" with ✕ remove button (bg today-bg, color today-star)
  - Project pill: "● ProjectName" (bg canvas-sunken)
  - Action buttons: 📅 When, 🏷 +Tag, 🔁 Repeat, 📁 Project — dashed border pills
  - Each opens its respective picker popover
- Only ONE editor open at a time. Click outside or Escape collapses.
- Title TextField: Enter submits (collapses), empty title deletes task

### CheckboxCircle (`CheckboxCircle.tsx`)
- 20px, border-radius 50%, border 1.5px solid ink-quaternary, bg canvas-elevated
- Hover: border accent, bg accent-subtle
- `.today` variant: border today-star, hover bg today-bg
- `.checked`: border accent, bg accent, white checkmark SVG (11px, stroke-width 2.5)
- `.cancelled`: border ink-quaternary, bg transparent, displays "✕" character (11px, ink-tertiary)

### CheckboxSquare (`CheckboxSquare.tsx`)
- 16px, border-radius 4px (radius-xs), border 1.5px solid ink-quaternary
- `.done`: border accent, bg accent, white checkmark SVG (9px)

### Tag (`Tag.tsx`)
- 11px bold, padding 2px 8px, radius-full
- 8 variants (prop `variant`):
  - `default`: bg canvas-sunken, color ink-secondary
  - `accent`: bg accent-subtle, color accent
  - `today`: bg today-bg, color today-star
  - `deadline`: bg deadline-bg, color deadline-red
  - `agent`: bg agent-bg, color agent-tint
  - `success`: bg success-bg, color success
  - `someday`: bg rgba(155,138,191,0.08), color someday-tint
  - `cancelled`: bg canvas-sunken, color ink-tertiary

### SectionHeader (`SectionHeader.tsx`)
- Flex row, gap 8px, padding 12px 0 4px
- Title: 14px bold, ink-primary
- `.muted` variant: 12px, ink-tertiary (used for "This Evening")
- Count: 11px, ink-tertiary
- Line: flex 1, height 1px, bg separator
- Collapsible: chevron button toggles collapsed state
- Draggable: drags section + all its tasks
- Context menu: Rename, Archive/Unarchive, Delete

### DateGroupHeader (`DateGroupHeader.tsx`)
- 12px bold, ink-primary, padding 8px 16px 4px
- Optional `.relative` span: ink-tertiary, normal weight (e.g., "— Next week")

### EmptyState (`EmptyState.tsx`)
- Centered, padding 80px 32px
- Icon: 48px, ink-quaternary, opacity 0.5
- Text: 15px, ink-tertiary

### DetailPanel (`DetailPanel.tsx`)
- Width: 340px, bg canvas-elevated, border-left 1px border
- Header: padding 20px 20px 12px, border-bottom separator
  - Title: 17px bold, ink-primary (editable ghost input)
  - Meta row: gap 8px, tags showing schedule/project
- Body: padding 16px 20px, overflow-y auto
  - Fields: label (11px bold uppercase, ink-tertiary, letter-spacing 0.5px) + value (12px, ink-secondary)
  - Field list: Project, Schedule, Start Date, Deadline, Tags, Notes (TextArea), Checklist, Activity
  - Notes TextArea supports line breaks (not TextField)
  - Title/Notes save with 300ms debounce
- Checklist: CheckboxSquare items, add new, drag to reorder
- Activity: ActivityFeed with human/agent entries

### CommandPalette (`CommandPalette.tsx`)
- Backdrop: fixed inset, rgba(0,0,0,0.15), backdrop-filter blur(4px)
- Palette: 560px, top 18%, centered, radius-xl, border border-strong, shadow-popover
- Input: icon (20px) + text input (17px) + "⌘K" shortcut hint
- Results: grouped (Navigation, Task Actions, Create) with group labels (11px bold uppercase)
- Items: icon (20px) + label (14px) + shortcut hint (11px mono, ink-tertiary)
- Active item: bg accent-subtle, color accent
- ↑↓ keyboard navigation, Enter executes, Escape closes
- Searches: commands + task titles + project names + notes content

### Pickers (popovers)

**WhenPicker:** 260px, radius-lg, shadow-popover. Natural language text input at top (parses as you type, shows resolved date below). Quick options (Today, This Evening, Someday). Calendar grid for date picking. Start date + deadline sections. Clear button.

**Natural language date parsing** (`naturalDateParser.ts`): Supports: "today", "tomorrow", "yesterday", weekday names ("monday", "tue", "fri"), "next [weekday]", "in N days/weeks/months", "Aug 1", "Mar 25", "2026-04-15". Returns `Date | null`. Ambiguous input returns null (no guess).

**TagPicker:** Searchable tag list with checkmarks. Create new tag inline. Toggle on/off.

**ProjectPicker:** Project list with colored dots. "No Project" option. Checkmark on current.

**RepeatPicker:** Presets (None/Daily/Weekly/Monthly/Yearly) + Custom (interval + unit). After-completion toggle.

**QuickMovePicker:** Searchable, grouped by area. Triggered by ⇧⌘M.

### Drag and Drop

- TaskRow: draggable, shows DragPreview (floating row copy)
- Drop between rows: DropIndicator (32px gap placeholder)
- Drop on sidebar items: move task to that project/view
- Section headers: draggable (moves section + all tasks)
- Drop on section header: move task into that section
- Cross-section drag in project view

## Keyboard Shortcuts

Resolved conflict: **⌘K = Command Palette** (per HTML validation). Complete Task = ⇧⌘C.

### Global (always active)
| Shortcut | Action |
|----------|--------|
| ⌘K | Command Palette |
| ⌘1-5 | Navigate: Inbox, Today, Upcoming, Someday, Logbook |
| ⌘N | New task (context-aware) |
| ⌘, | Settings |
| ⌘/ | Toggle sidebar |
| ⌘F | Quick Find / Search |

### List focused (no text field active)
| Shortcut | Action |
|----------|--------|
| ↑↓ | Navigate task list |
| Return | Open inline editor |
| Space | New task below selection |
| ⇧⌘C | Complete selected task |
| ⌥⌘K | Cancel selected task |
| ⌫ | Delete selected task |
| ⌘D | Duplicate selected task |
| ⌘T | Schedule Today |
| ⌘E | This Evening |
| ⌘R | Start Anytime |
| ⌘O | Start Someday |
| ⌘S | Show When picker |
| ⇧⌘D | Set Deadline |
| ⇧⌘M | Move to Project (QuickMovePicker) |
| ⇧⌘T | Edit Tags |
| ⇧⌘R | Edit Repeat |
| ⌘↑/↓ | Move task up/down |
| ⌥⌘↑/↓ | Move task to top/bottom |
| Ctrl+]/[ | Start date +/- 1 day |
| ⇧↑/↓ | Extend selection |
| ⌘A | Select all in current view |

### Editor focused
| Shortcut | Action |
|----------|--------|
| ⌘Return | Save and close inline editor |
| Escape | Close inline editor |
| ⇧⌘C | New checklist item in open task (note: same chord as Complete Task in list context — context determines behavior) |
| (all typing goes to text field normally) |

### Implementation: `useKeyboard.ts`

```typescript
useEffect(() => {
  const handler = (e: KeyboardEvent) => {
    const target = e.target as HTMLElement;
    const inTextField = target.tagName === 'INPUT' ||
                        target.tagName === 'TEXTAREA' ||
                        target.isContentEditable;

    // Global shortcuts (always active)
    if (e.metaKey && e.key === 'k') { e.preventDefault(); togglePalette(); return; }
    if (e.metaKey && e.key === 'n') { e.preventDefault(); createTask(); return; }
    if (e.metaKey && /^[1-5]$/.test(e.key)) { e.preventDefault(); navigate(e.key); return; }

    // Skip list shortcuts when typing
    if (inTextField) {
      // Only handle Cmd+Return (close editor) and Escape
      if (e.metaKey && e.key === 'Enter') { e.preventDefault(); closeEditor(); return; }
      if (e.key === 'Escape') { e.preventDefault(); closeEditor(); return; }
      return;
    }

    // List shortcuts (no text field active)
    // ... full shortcut map
  };
  window.addEventListener('keydown', handler);
  return () => window.removeEventListener('keydown', handler);
}, [dependencies]);
```

## View Specifications

### InboxView
- Tasks: schedule=0, no startDate, pending (+ completed today with strikethrough)
- Sorted by index
- Hover actions: ★ Today (→ schedule=1), 📅 Date (→ WhenPicker), 💤 Someday (→ schedule=2), 📁 Project (→ QuickMovePicker)
- NewTaskRow at bottom

### TodayView
- Tasks: schedule=1, pending or completed, startDate null or ≤ today
- Sorted by timeSlot (null first = morning), then todayIndex
- Morning section (no header) + "This Evening" section (muted SectionHeader)
- Checkboxes: amber border (`.today` variant)
- Flat list within each section — tasks show project pill in their meta area (per HTML validation). No project sub-headers.
- NewTaskRow at bottom

### UpcomingView
- Tasks: pending, startDate > today, schedule ≠ 2
- Grouped by startDate with DateGroupHeader
- Relative dates: "Tomorrow — Fri, Mar 29", "Monday, Mar 31", "April 7 — Next week"

### SomedayView
- Tasks: schedule=2, no startDate, pending (+ completed today)
- Sorted by index. NewTaskRow at bottom.

### LogbookView
- Tasks: status=1 (completed) or status=2 (cancelled)
- Grouped by completion date (most recent first) with DateGroupHeader
- Completed: checked checkbox + strikethrough title + "Completed 3h ago" in green
- Cancelled: ✕ checkbox + strikethrough + "Cancelled" tag
- Hover shows "Reopen" button

### ProjectView
- Toolbar extras: progress bar ("4/12" + thin bar) + "Add Section" ghost button
- Sectionless tasks first, then sections (sorted by index)
- Each section: collapsible SectionHeader (chevron + title + count + line) + tasks + NewTaskRow
- Archived sections hidden, count shown at bottom
- Section context menu: Rename, Archive/Unarchive, Delete
- Section drag reorder (moves all contained tasks)

## Interaction Model

- **Single click** on task row → select (highlight)
- **Double-click** or **Return** on selected task → expand inline editor
- **Click outside** expanded editor or **Escape** → collapse
- **Only ONE** inline editor open at a time
- **⌘N** → create task in current view context (schedule/projectId set automatically) → expand inline editor
- **Space** → create task below selection → expand inline editor
- **Completed tasks** stay visible with strikethrough until next day, then roll to Logbook
- **Cancelled tasks** show ✕ in checkbox
- **Context-aware creation**: ⌘N in Today → schedule=1; in Inbox → schedule=0; in Project → projectId set
- **Agent indicators**: "✦ In progress", "✦ Claude assigned" with agent-tint purple
- **Type Travel**: start typing in list (no text field focused) → opens Command Palette with typed characters as initial query
- **Multi-select**: Shift+click, Shift+↑↓, ⌘A. Bulk operations: complete, schedule, move, tag, delete.

## Sync Engine (Rust)

- **Outbound**: mutations queue as `pendingOps`. Background thread flushes periodically (30s). Exponential backoff on failure (2s→4s→8s...60s max, reset on success). Stop after 3 consecutive failures per cycle.
- **Inbound**: SSE connection to `GET /events/stream?topics=task.*,project.*,area.*,section.*,tag.*`. On event: fetch affected entity, upsert local SQLite, emit Tauri event.
- **SSE reconnect**: exponential backoff (5s→10s→20s→40s→60s max, reset on successful connection).
- **Conflict resolution**: server wins. Delta events overwrite local fields.
- **Initial sync**: `load_all` from server, upsert by ID. Local-only records preserved.
- **Status**: observable via `get_sync_status()`. React shows spinner during sync, error triangle on failure, pending count badge.
- **Local-only mode**: when no server URL configured, no sync, no pendingOps queued.

## Aesthetic Rules

- **Theme:** "Bone" — canvas #f6f5f2, accent #4670a0. Use CSS custom properties for all tokens.
- **Font:** Atkinson Hyperlegible via @font-face. Never system font for content.
- **No animations.** Every state change instant. Only allowed: 80ms hover `background-color` transition on interactive elements.
- **No priority flags.** Position = priority.
- **Agent elements:** Use `--agent-tint` (#7868a8) purple. Same stream as human, different tint.
- **Today:** Amber `--today-star` (#c88c30). Checkbox borders amber in Today view.
- **Task rows:** Single-line, 32px. Never two-line.

## What NOT To Do

1. Don't use Tailwind or CSS-in-JS. Use semantic CSS classes from theme.css.
2. Don't hardcode colors, sizes, or spacing. Use CSS variables.
3. Don't treat the API as source of truth. Local SQLite is truth.
4. Don't refetch after mutations. Write locally, sync in background.
5. Don't build separate state per view. One Zustand store, computed selectors.
6. Don't add CSS animations, keyframes, transitions (except 80ms hover smoothing).
7. Don't make task items two-line. 32px single row.
8. Don't dispatch parallel agents for UI. One feature at a time, tested.
9. Don't skip manual testing. Every feature verified by running the app.
10. Don't use ⌘K for Complete Task. ⌘K = Command Palette. ⇧⌘C = Complete.
