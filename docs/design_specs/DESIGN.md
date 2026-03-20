# atask — Design Package for Dioxus Implementation

**Target:** Dioxus 0.7+ desktop app (macOS primary, cross-platform secondary)
**Renderer:** WebView (default), with future Blitz/WGPU migration path
**Styling:** Tailwind CSS utilities + custom CSS tokens via `assets/theme.css`
**Font:** Atkinson Hyperlegible (loaded from `assets/fonts/`)
**Aesthetic:** Things-inspired — warm neutral palette, restrained animation, typographic hierarchy

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [Design Tokens](#2-design-tokens)
3. [Typography](#3-typography)
4. [Layout Architecture](#4-layout-architecture)
5. [Component Specs](#5-component-specs)
6. [View Specs](#6-view-specs)
7. [Command Palette](#7-command-palette)
8. [Keyboard Shortcut Map](#8-keyboard-shortcut-map)
9. [Animation & Transition Specs](#9-animation--transition-specs)
10. [API-to-View Data Mapping](#10-api-to-view-data-mapping)

---

## 1. Project Structure

```
atask-app/
├── Cargo.toml
├── Dioxus.toml
├── assets/
│   ├── fonts/
│   │   ├── AtkinsonHyperlegible-Regular.woff2
│   │   ├── AtkinsonHyperlegible-Bold.woff2
│   │   ├── AtkinsonHyperlegible-Italic.woff2
│   │   └── AtkinsonHyperlegible-BoldItalic.woff2
│   ├── theme.css              ← design tokens + base styles
│   ├── components.css         ← component-level styles
│   ├── views.css              ← view-level layout styles
│   └── animations.css         ← keyframes + transition classes
├── src/
│   ├── main.rs                ← launch + router
│   ├── api/
│   │   ├── mod.rs
│   │   ├── client.rs          ← HTTP client (reqwest) to Go backend
│   │   ├── types.rs           ← API response types (serde)
│   │   └── events.rs          ← SSE subscription (eventsource)
│   ├── state/
│   │   ├── mod.rs
│   │   ├── app.rs             ← global app state (signals)
│   │   ├── tasks.rs           ← task list state + optimistic updates
│   │   ├── projects.rs        ← project state
│   │   ├── navigation.rs      ← sidebar selection, detail panel state
│   │   └── command.rs         ← command palette state
│   ├── components/
│   │   ├── mod.rs
│   │   ├── checkbox.rs
│   │   ├── task_item.rs
│   │   ├── task_detail.rs
│   │   ├── section_header.rs
│   │   ├── new_task_inline.rs
│   │   ├── activity_entry.rs
│   │   ├── sidebar.rs
│   │   ├── toolbar.rs
│   │   ├── tag_pill.rs
│   │   ├── button.rs
│   │   ├── text_input.rs
│   │   ├── command_palette.rs
│   │   └── checklist_item.rs
│   └── views/
│       ├── mod.rs
│       ├── today.rs
│       ├── inbox.rs
│       ├── upcoming.rs
│       ├── someday.rs
│       ├── logbook.rs
│       └── project.rs
```

### Dioxus.toml Configuration

```toml
[application]
name = "atask"
default_platform = "desktop"

[web.app]
title = "atask"

[desktop]
enable_wry = true

[[desktop.window]]
title = "atask"
width = 1080
height = 720
min_width = 640
min_height = 480
transparent = true           # enables sidebar vibrancy effect
decorations = true           # native macOS traffic lights
```

---

## 2. Design Tokens

All tokens live in `assets/theme.css` as CSS custom properties. See the companion file `theme.css` in this package for the complete token set. Summary:

### Canvas (Backgrounds)

| Token | Value | Usage |
|-------|-------|-------|
| `--canvas` | `#f6f5f2` | Main content background |
| `--canvas-elevated` | `#fefefe` | Cards, detail panel, popovers |
| `--canvas-sunken` | `#eceae7` | Inset areas, pill backgrounds |
| `--sidebar-bg` | `rgba(238,236,231,0.72)` | Sidebar with backdrop-filter blur |

### Ink (Text)

| Token | Value | Usage |
|-------|-------|-------|
| `--ink-primary` | `#222120` | Headings, task titles, primary text |
| `--ink-secondary` | `#686664` | Body text, descriptions |
| `--ink-tertiary` | `#a09e9a` | Metadata, timestamps, placeholders |
| `--ink-quaternary` | `#c8c6c2` | Disabled text, faint borders |
| `--ink-on-accent` | `#ffffff` | Text on accent backgrounds |

### Accent & Semantic

| Token | Value | Usage |
|-------|-------|-------|
| `--accent` | `#4670a0` | Primary actions, links, selected states |
| `--accent-hover` | `#3a5f8a` | Hover on accent elements |
| `--accent-subtle` | `rgba(70,112,160,0.10)` | Accent-tinted backgrounds |
| `--accent-ring` | `rgba(70,112,160,0.30)` | Focus rings |
| `--today-star` | `#c88c30` | Today view indicator, today badge |
| `--today-bg` | `rgba(200,140,48,0.08)` | Today badge background |
| `--someday-tint` | `#8878a0` | Someday view accent |
| `--deadline-red` | `#c04848` | Overdue/deadline indicators |
| `--deadline-bg` | `rgba(192,72,72,0.08)` | Deadline badge background |
| `--success` | `#4a8860` | Completed state, success |
| `--success-bg` | `rgba(74,136,96,0.08)` | Success badge background |
| `--agent-tint` | `#7868a8` | Agent-touched elements |
| `--agent-bg` | `rgba(120,104,168,0.07)` | Agent card backgrounds |
| `--agent-border` | `rgba(120,104,168,0.20)` | Agent card borders |

### Borders & Separators

| Token | Value |
|-------|-------|
| `--border` | `rgba(0,0,0,0.06)` |
| `--border-strong` | `rgba(0,0,0,0.12)` |
| `--separator` | `rgba(0,0,0,0.05)` |

### Shadows (macOS-style layered)

| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.04), 0 1px 1px rgba(0,0,0,0.03)` | Cards, inputs |
| `--shadow-md` | `0 2px 8px rgba(0,0,0,0.06), 0 1px 3px rgba(0,0,0,0.04)` | Elevated panels |
| `--shadow-lg` | `0 8px 30px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04)` | Modals |
| `--shadow-popover` | `0 12px 40px rgba(0,0,0,0.12), 0 4px 12px rgba(0,0,0,0.06)` | Command palette |

### Spacing (4px base)

```
--sp-1: 4px    --sp-2: 8px    --sp-3: 12px   --sp-4: 16px
--sp-5: 20px   --sp-6: 24px   --sp-7: 28px   --sp-8: 32px
--sp-10: 40px  --sp-12: 48px  --sp-16: 64px  --sp-20: 80px
```

### Border Radii

```
--radius-xs: 4px     --radius-sm: 6px     --radius-md: 8px
--radius-lg: 12px    --radius-xl: 16px    --radius-full: 9999px
```

### Transitions

```
--ease-out:    cubic-bezier(0.16, 1, 0.3, 1)
--ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1)
--dur-fast:    120ms
--dur-normal:  200ms
--dur-slow:    350ms
```

---

## 3. Typography

**Font family:** `'Atkinson Hyperlegible', system-ui, -apple-system, sans-serif`
**Rendering:** `-webkit-font-smoothing: antialiased`

| Role | Size | Weight | Line Height | Token |
|------|------|--------|-------------|-------|
| Page title (Today, Inbox) | 20px | 700 | 1.2 | `--text-xl` |
| Project title | 24px | 700 | 1.2 | `--text-2xl` |
| Section header | 17px | 700 | 1.2 | `--text-lg` |
| Task title | 14px | 400 | 1.2 | `--text-base` |
| Detail body | 15px | 400 | 1.65 | `--text-md` |
| Metadata / timestamps | 12px | 700 | 1.5 | `--text-sm` |
| Group labels (uppercase) | 11px | 700 | 1.5 | `--text-xs` + `letter-spacing: 1px` + `text-transform: uppercase` |

### Monospace (code, values)

`'SF Mono', 'Menlo', 'Cascadia Code', monospace` — used only for keyboard shortcut hints and technical metadata.

---

## 4. Layout Architecture

### Three-Panel Layout

```
┌──────────────────────────────────────────────────────────┐
│ ┌──────────┬────────────────────────┬─────────────────┐  │
│ │ Sidebar  │     Main Content       │  Detail Panel   │  │
│ │ 240px    │     flex: 1            │  340px          │  │
│ │          │                        │  (conditional)  │  │
│ │ traffic  │  ┌──────────────────┐  │                 │  │
│ │ lights   │  │ Toolbar  52px    │  │  Shows when a   │  │
│ │          │  ├──────────────────┤  │  task is        │  │
│ │ nav      │  │                  │  │  selected       │  │
│ │ items    │  │  Task List       │  │                 │  │
│ │          │  │  (scrollable)    │  │                 │  │
│ │ ──────── │  │                  │  │                 │  │
│ │ projects │  │                  │  │                 │  │
│ │          │  │                  │  │                 │  │
│ │ ──────── │  │                  │  │                 │  │
│ │ areas    │  └──────────────────┘  │                 │  │
│ └──────────┴────────────────────────┴─────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### RSX Shell

```rust
#[component]
fn App() -> Element {
    rsx! {
        div { class: "app-frame",
            Sidebar {}
            div { class: "app-main",
                Toolbar {}
                div { class: "app-content",
                    // Router renders active view here
                    Outlet::<Route> {}
                }
            }
            // Conditionally rendered
            if selected_task.read().is_some() {
                TaskDetail {}
            }
        }
        // Overlay layer
        if command_palette_open() {
            CommandPalette {}
        }
    }
}
```

### CSS Layout

```css
.app-frame {
    display: flex;
    width: 100vw;
    height: 100vh;
    background: var(--canvas);
    overflow: hidden;
}

.app-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0; /* prevent flex blowout */
}

.app-content {
    flex: 1;
    overflow-y: auto;
    padding: var(--sp-6);
}
```

---

## 5. Component Specs

### 5.1 Checkbox

Circular, Things-style. State change is instant — no animation.

**States:** unchecked, hover, checked, today (amber border), disabled
**Size:** 20×20px, border-radius: 50%, border: 1.5px

```rust
#[derive(Props, Clone, PartialEq)]
struct CheckboxProps {
    checked: bool,
    today: Option<bool>,       // amber border variant
    on_toggle: EventHandler<bool>,
}

#[component]
fn Checkbox(props: CheckboxProps) -> Element {
    let border_color = if props.today.unwrap_or(false) {
        "var(--today-star)"
    } else if props.checked {
        "var(--accent)"
    } else {
        "var(--ink-quaternary)"
    };

    rsx! {
        div {
            class: if props.checked { "checkbox checked" } else { "checkbox" },
            style: "border-color: {border_color}",
            onclick: move |_| props.on_toggle.call(!props.checked),
            if props.checked {
                svg { /* polyline points="2.5 6 5 8.5 9.5 3.5" */ }
            }
        }
    }
}
```

**CSS:**
```css
.checkbox.checked {
    border-color: var(--accent);
    background: var(--accent);
    /* No animation — instant fill */
}
```

---

### 5.2 TaskItem

Single-line row. Checkbox, title, and metadata all on one horizontal axis. 32px row height. Title truncates with ellipsis when it meets right-aligned metadata.

**States:** default, hover (subtle bg), selected (accent-tinted bg), completed (struck through), draggable (grip indicator)

**Completion behavior:** When a task is completed (checkbox click), it instantly shows as struck through with checked checkbox and faded text (`--ink-tertiary`). It remains visible in the current view until the next day, then rolls into the logbook. The user can also manually move it to the logbook via the command palette or by unchecking and re-checking. This matches the Things pattern — completed tasks stay in context so you can see your progress for the day.

**Drag affordance:** When a view supports reordering, each task row shows a subtle grip icon (6-dot drag handle, `--ink-quaternary`) on the far left, visible on hover. This signals that rows are draggable.

**Right-click context menu:** Right-clicking a task row shows a native-styled context menu with:

| Action | Shortcut | Separator |
|--------|----------|-----------|
| Complete | ⌘⇧C | |
| Schedule for Today | ⌘T | |
| Defer to Someday | | |
| Move to Inbox | | ── |
| Move to Project → | | submenu |
| Set Start Date... | ⌘D | |
| Set Deadline... | ⌘⇧D | ── |
| Add Tag → | | submenu |
| Duplicate | | |
| Copy Link | | ── |
| Delete | ⌫ | |

**Context menu for projects** (right-click in sidebar):
- Rename
- Set Color...
- Complete Project
- Delete Project

**Context menu for sections** (right-click section header):
- Rename
- Delete Section

The context menu uses the same visual style as the command palette results: `--canvas-elevated` background, `--shadow-popover`, `--radius-md`. Items are 30px height, `--text-sm`.

```rust
#[derive(Props, Clone, PartialEq)]
struct TaskItemProps {
    task: Task,
    selected: bool,
    on_select: EventHandler<Uuid>,
    on_complete: EventHandler<Uuid>,
}

#[component]
fn TaskItem(props: TaskItemProps) -> Element {
    let is_completed = props.task.status == TaskStatus::Completed;
    let is_today = props.task.today_index.is_some();

    rsx! {
        div {
            class: format_args!("task-item{}", if props.selected { " selected" } else { "" }),
            onclick: move |_| props.on_select.call(props.task.id),

            Checkbox {
                checked: is_completed,
                today: is_today,
                on_toggle: move |_| props.on_complete.call(props.task.id),
            }

            div { class: "task-content",
                div {
                    class: if is_completed { "task-title completed" } else { "task-title" },
                    "{props.task.title}"
                }
                TaskMeta { task: props.task.clone() }
            }
        }
    }
}
```

**CSS:**
```css
.task-item {
    display: flex;
    align-items: center;    /* single-line — all vertically centered */
    gap: var(--sp-3);
    padding: 6px var(--sp-4);
    border-radius: var(--radius-md);
    height: 32px;
    cursor: default;
}

.task-content {
    flex: 1;
    min-width: 0;
    display: flex;          /* horizontal — title + meta on same line */
    align-items: center;
    gap: var(--sp-3);
}

.task-title {
    flex: 1;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.task-meta {
    flex-shrink: 0;
    margin-left: auto;      /* pins metadata to right edge */
    white-space: nowrap;
}
```

**Metadata row** shows contextual pills depending on available data:
- Project pill (colored dot + name) — always show if task has a project
- Deadline pill (red text, "Due Mar 25") — show if deadline exists
- Today badge (★ Today in amber) — show if in today view from a non-today context
- Checklist progress ("3/5" in `--ink-tertiary`) — show if task has checklist items
- Agent indicator (✦ Claude assigned / ✦ In progress) — show if last activity is from an agent

**Priority of metadata display:** project → deadline → today badge → checklist → agent → tags. Truncate after 3 items on narrow widths.

**Checklist indicator:** When a task has checklist items, show a compact progress indicator in the metadata row: `"3/5"` (completed/total) in `--text-xs` `--ink-tertiary`. If all items are completed, show in `--success` color. This requires the API to include `checklist_count` and `open_checklist_count` in the task response, or the client fetches checklist data per visible task.

---

### 5.3 TaskDetail (Right Panel)

340px fixed-width panel. Displays the full task with editable fields.

**Sections (top to bottom):**
1. **Title** — editable, ghost input style (large, no border), `--text-lg` bold
2. **Meta row** — tag pills (Today, tags, project)
3. **Fields** — each with uppercase `--text-xs` label:
   - Project (dot + name, clickable to change)
   - Schedule (Inbox / Today / Someday selector)
   - Start Date (date picker or "None")
   - Deadline (date picker or "None")
   - Tags (pill list, + button to add)
   - Location (name or "None")
   - Recurrence (rule summary or "None")
4. **Notes** — markdown editor, `--text-md`, `--leading-relaxed`
5. **Checklist** — square checkboxes (4px radius, 16×16), lighter weight than task checkboxes
6. **Activity stream** — separator, then chronological entries (human + agent)

**Field label style:**
```css
.detail-field-label {
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--ink-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: var(--sp-1);
}
```

---

### 5.4 Sidebar

240px, translucent background with backdrop blur. Split into groups separated by 1px lines.

**Structure:**
```
[Traffic Lights area — 52px tall, drag region]

  Inbox          [badge: count]
  ★ Today        [badge: count]      ← active state = bold + bg
  Upcoming
  Someday
  Logbook

  ─────────────────────────────

  PROJECTS                           ← uppercase group label
  ● atask v0     [badge: count]      ← colored dot per project
  ● Homelab      [badge: count]
  ● Roon Ext     [badge: count]

  ─────────────────────────────

  AREAS                              ← uppercase group label
  Work
  Home
```

**Sidebar item states:**
- Default: `color: var(--ink-secondary)`
- Hover: `background: var(--sidebar-hover)`, `color: var(--ink-primary)`
- Active (selected): `background: var(--sidebar-active)`, `color: var(--ink-primary)`, `font-weight: 700`

**Item height:** 30px (5px vertical padding + content)
**Icon size:** 16×16, stroke-width 1.8, within a 20×20 container
**Badge:** right-aligned, `--text-xs`, `color: var(--ink-tertiary)`

**Sidebar icons per item:**
- Inbox: rounded rectangle (incoming tray)
- Today: filled star (amber)
- Upcoming: calendar
- Someday: clock (someday-tint color)
- Logbook: archive/book
- Projects: 8×8 colored circle (project-specific color)
- Areas: folder-like rectangle

---

### 5.5 Toolbar

52px height, spans main content area. Flex row, space-between.

**Left side:** View title (icon + name + date subtitle for Today)
**Right side:** Icon buttons (search, new task)

```rust
#[component]
fn Toolbar() -> Element {
    rsx! {
        div { class: "app-toolbar",
            div { class: "app-toolbar-left",
                // View icon + title
                span { class: "app-view-title", "Today" }
                span { class: "toolbar-subtitle", "Thursday, Mar 20" }
            }
            div { class: "app-toolbar-right",
                ToolbarButton { icon: "search", shortcut: "⌘F" }
                ToolbarButton { icon: "plus", shortcut: "⌘N" }
            }
        }
    }
}
```

**Toolbar button:** 30×30px, `--radius-sm`, ghost style. Icon 16×16.

---

### 5.6 SectionHeader

Lightweight divider within task lists (projects, or Today's "This Evening").

```
Section Name       3      ─────────────────────
[bold, --text-base]  [count, tertiary]  [1px line fills remaining space]
```

**Collapsible:** Click toggles visibility of tasks below. Chevron rotates 90° on collapse.

---

### 5.7 NewTaskInline

Bottom of every task list. Dashed circle + "New Task" text.

**Default:** `color: var(--ink-tertiary)`
**Hover:** `color: var(--accent)`, `background: var(--accent-subtle)`

On click, transforms into an inline text input with ghost styling. Press Enter to create task, Escape to cancel.

---

### 5.8 TagPill

Small inline badges for contextual information.

**Variants:**

| Variant | Background | Text Color | Example |
|---------|-----------|------------|---------|
| default | `--canvas-sunken` | `--ink-secondary` | "Work" |
| accent | `--accent-subtle` | `--accent` | "Design" |
| today | `--today-bg` | `--today-star` | "★ Today" |
| deadline | `--deadline-bg` | `--deadline-red` | "Mar 25" |
| agent | `--agent-bg` | `--agent-tint` | "✦ Agent" |
| success | `--success-bg` | `--success` | "Done" |

**Size:** `font-size: --text-xs`, `font-weight: 700`, `padding: 2px 8px`, `border-radius: --radius-full`

---

### 5.9 ActivityEntry

Single entry in the activity stream (detail panel or standalone).

**Layout:** avatar (28×28 circle) + body (author name + timestamp + content)

**Human avatar:** `background: var(--accent)`, white initial letter
**Agent avatar:** `background: var(--agent-tint)`, ✦ symbol, plus a small ✦ badge indicator at bottom-right (14×14, white bg, agent-tint text)

**Agent response card:** When an agent provides structured output, wrap it in:
```css
.activity-agent-card {
    margin-top: var(--sp-2);
    background: var(--agent-bg);
    border: 1px solid var(--agent-border);
    border-radius: var(--radius-md);
    padding: var(--sp-3);
    font-size: var(--text-sm);
    color: var(--ink-secondary);
}
```

---

### 5.10 Button

Four variants, all with `font-weight: 700`, `--radius-sm`, `font-size: --text-sm`.

| Variant | Background | Text | Border | Shadow |
|---------|-----------|------|--------|--------|
| primary | `--accent` | `--ink-on-accent` | none | inset highlight + drop |
| secondary | `--canvas-elevated` | `--ink-primary` | `--border-strong` | `--shadow-sm` |
| ghost | transparent | `--ink-secondary` | none | none |
| danger | `--deadline-red` | `--ink-on-accent` | none | red-tinted drop |

**Sizes:** sm (4px 10px, --text-xs), default (6px 14px), lg (8px 20px, --text-base)
**Active state:** `transform: scale(0.97)` on all variants.

---

### 5.11 TextInput

Two variants:

**Standard:** bordered, `--shadow-sm`, `--radius-sm`, focus ring `--accent-ring`
**Ghost:** no border, no background, large text — used for editable titles

```css
.input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-ring);
}
```

---

## 6. View Specs

### 6.1 Today View

**API:** `GET /views/today`
**Title:** ★ Today + current date ("Thursday, Mar 20")
**Icon:** Filled star, `--today-star`

**Content:**
- Task list ordered by `today_index`
- All checkboxes use amber border variant
- Optional "This Evening" section divider (user-created section within Today)
- NewTaskInline at bottom

**Empty state:** Centered message: "Your day is clear." in `--ink-tertiary`, with a subtle illustration or the ★ icon enlarged and faded.

**Triage interaction:** Tasks dragged from Inbox to the Today list get `PUT /tasks/{id}/schedule` with `{ "schedule": "anytime" }` and assigned a `today_index`.

---

### 6.2 Inbox View

**API:** `GET /views/inbox`
**Title:** Inbox (with tray icon)
**Icon:** Rounded rect tray, `--accent`

**Content:**
- Task list ordered by index
- No Today-amber checkboxes — standard style
- Each task shows a row of quick-action buttons on hover:
  - ★ Schedule Today → `PUT /tasks/{id}/schedule { "anytime" }`
  - 📅 Set Date → opens date picker → `PUT /tasks/{id}/start-date`
  - 💤 Someday → `PUT /tasks/{id}/schedule { "someday" }`
  - 📁 Move to Project → opens project picker → `PUT /tasks/{id}/project`

**Triage flow:** The core inbox interaction is processing top-to-bottom. After an action, the next task should feel like it "steps up" — use staggered fadeInUp animation as tasks leave.

**Empty state:** "Inbox Zero ✓" centered, success-green tinted.

**Badge behavior:** Sidebar badge shows inbox count. Animate badge decrement on triage.

---

### 6.3 Upcoming View

**API:** `GET /views/upcoming`
**Title:** Upcoming (with calendar icon)

**Content:** Tasks grouped by `start_date`, displayed as date-sectioned list.

**Date sections:**
```
Tomorrow — Fri, Mar 21
─────────────────────────
  [ ] Task with start date tomorrow
  [ ] Another task

Saturday, Mar 22
─────────────────────────
  [ ] Weekend task

Next Week — Mon, Mar 24
─────────────────────────
  [ ] Monday task
```

**Date header formatting rules:**
- Tomorrow → "Tomorrow — Day, Mon DD"
- This week → "DayName, Mon DD"
- Next week → "Next Week — Day, Mon DD" (for first day), then "Day, Mon DD"
- Further out → "Mon DD" or "Mon DD, YYYY" if different year

**Empty state:** "Nothing scheduled ahead." with a calendar icon.

---

### 6.4 Someday View

**API:** `GET /views/someday`
**Title:** Someday (with clock icon, `--someday-tint`)

**Content:**
- Flat task list ordered by `index`
- Standard checkboxes (no amber)
- Hover actions include: ★ Schedule Today, 📅 Set Date
- Tasks can be grouped by project (optional toggle in toolbar)

**Empty state:** "No someday tasks. Everything is decided."

---

### 6.5 Logbook View

**API:** `GET /views/logbook`
**Title:** Logbook (with archive icon)

**Content:** Completed and cancelled tasks grouped by completion date.

**Date sections:** Same formatting as Upcoming but using `completed_at`.

**Task display:**
- Completed tasks: checked checkbox (accent blue), struck-through title
- Cancelled tasks: ✕ icon (not checkbox), struck-through title in `--ink-tertiary`
- Show completion/cancellation time in metadata

**Interaction:** Tasks can be "uncompleted" — checkbox click removes completion. This is a `POST` to create a new task (since completion is final in the event model) or a domain-specific reopen endpoint if the API supports it.

**Empty state:** "Nothing completed yet. Get started!"

---

### 6.6 Project View

**API:** `GET /projects/{id}` (project with sections + task counts), `GET /tasks?project_id={id}`
**Title:** Project name (with colored dot matching sidebar)

**Content:**
```
[Sectionless tasks — ordered by index]
  [ ] Task A
  [ ] Task B
  + New Task

Section: Design          3  ─────────
  [ ] Task C
  [ ] Task D
  [ ] Task E
  + New Task

Section: Implementation  5  ─────────
  [ ] Task F
  ...
  + New Task
```

**Key behaviors:**
- Sectionless tasks appear at top, before any sections
- Each section is collapsible
- Each section has its own NewTaskInline
- Tasks show deadline and today badge in metadata, but not project pill (redundant)
- Sections can be reordered via drag
- Tasks can be dragged between sections

**Project progress:** Toolbar right side shows completion ratio: "7 / 15" with a thin progress bar beneath the toolbar.

**Toolbar extras:** + Add Section button (ghost style)

---

## 7. Command Palette

Overlay component, centered on screen. macOS Spotlight-style.

### Layout

```
┌─────────────────────────────────────────────┐
│  🔍  Type a command or search...     ⌘K     │
├─────────────────────────────────────────────┤
│  RECENT                                     │
│    ★ Schedule for Today         ⌘T          │
│    ➡ Move to Project            ⌘⇧M         │
│    📅 Set Start Date            ⌘D          │
│                                             │
│  NAVIGATION                                 │
│    → Go to Inbox                ⌘1          │
│    → Go to Today                ⌘2          │
│    → Go to Upcoming             ⌘3          │
│                                             │
│  ACTIONS                                    │
│    + New Task                   ⌘N          │
│    + New Project                ⌘⇧N         │
│    ✓ Complete Task              ⌘⇧C         │
└─────────────────────────────────────────────┘
```

### Specs

- **Width:** 560px, centered horizontally, offset ~20% from top
- **Background:** `--canvas-elevated` with `--shadow-popover`
- **Border:** `1px solid var(--border-strong)`, `--radius-xl`
- **Backdrop:** semi-transparent overlay `rgba(0,0,0,0.15)` with `backdrop-filter: blur(4px)`
- **Input:** ghost-style, full width, `--text-lg`, placeholder "Type a command or search..."
- **Results:** grouped by category with `--text-xs` uppercase group labels
- **Result item height:** 36px, with icon (20×20) + label + right-aligned shortcut hint in monospace `--text-xs` `--ink-tertiary`
- **Active result:** `background: var(--accent-subtle)`, `color: var(--accent)`

### Behavior

- **Open:** `⌘K` (global)
- **Close:** `Escape` or click backdrop
- **Navigation:** `↑↓` arrow keys, `Enter` to execute
- **Search:** Filters commands and tasks by fuzzy match on title. Tasks appear below commands.
- **Scoped commands:** When a task is selected, task-specific actions appear first (complete, schedule, move, etc.)

### Command Categories

**Navigation:**
| Command | Action | Shortcut |
|---------|--------|----------|
| Go to Inbox | Navigate to inbox view | `⌘1` |
| Go to Today | Navigate to today view | `⌘2` |
| Go to Upcoming | Navigate to upcoming view | `⌘3` |
| Go to Someday | Navigate to someday view | `⌘4` |
| Go to Logbook | Navigate to logbook view | `⌘5` |
| Go to Project: {name} | Navigate to project view | — |

**Task Actions (when task selected):**
| Command | API Call | Shortcut |
|---------|----------|----------|
| Complete Task | `POST /tasks/{id}/complete` | `⌘⇧C` |
| Cancel Task | `POST /tasks/{id}/cancel` | — |
| Schedule for Today | `PUT /tasks/{id}/schedule` → anytime | `⌘T` |
| Defer to Someday | `PUT /tasks/{id}/schedule` → someday | — |
| Move to Inbox | `PUT /tasks/{id}/schedule` → inbox | — |
| Set Start Date | `PUT /tasks/{id}/start-date` | `⌘D` |
| Set Deadline | `PUT /tasks/{id}/deadline` | `⌘⇧D` |
| Move to Project | `PUT /tasks/{id}/project` | `⌘⇧M` |
| Add Tag | `POST /tasks/{id}/tags/{tag_id}` | — |
| Delete Task | `DELETE /tasks/{id}` | `⌫` (with confirmation) |

**Creation:**
| Command | Action | Shortcut |
|---------|--------|----------|
| New Task | Create task in current view context | `⌘N` |
| New Task in Inbox | Create task with schedule=inbox | `⌘⇧N` |
| New Project | Create project | — |
| New Section | Create section in current project | — |

---

## 8. Keyboard Shortcut Map

### Global

| Shortcut | Action |
|----------|--------|
| `⌘K` | Open command palette |
| `⌘N` | New task (context-aware) |
| `⌘⇧N` | New task in inbox |
| `⌘F` | Focus search / filter in current view |
| `⌘,` | Open preferences |
| `⌘1` | Go to Inbox |
| `⌘2` | Go to Today |
| `⌘3` | Go to Upcoming |
| `⌘4` | Go to Someday |
| `⌘5` | Go to Logbook |
| `Escape` | Close detail panel / command palette / cancel inline edit |

### Task List Navigation

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Move selection up/down |
| `Enter` | Open detail panel for selected task |
| `Space` | Toggle completion on selected task |
| `⌘↑` / `⌘↓` | Reorder selected task (move up/down in list) |

### Task Actions (with task selected)

| Shortcut | Action |
|----------|--------|
| `⌘T` | Schedule selected task for Today |
| `⌘D` | Set start date |
| `⌘⇧D` | Set deadline |
| `⌘⇧M` | Move to project |
| `⌘⇧C` | Complete task |
| `⌫` | Delete task (with confirmation) |
| `Tab` | Move focus to detail panel |

### Detail Panel

| Shortcut | Action |
|----------|--------|
| `⌘Enter` | Save and close detail |
| `Tab` | Cycle through fields |
| `Escape` | Close detail panel, return focus to list |

### Inbox-Specific

| Shortcut | Action |
|----------|--------|
| `⌘T` | Triage to Today (on selected task) |
| `⌘⇧S` | Defer to Someday |
| `→` | Quick-open project picker for selected task |

---

## 9. Motion Policy

### Principle: Instant State Changes

atask does not use entrance animations, staggered reveals, slide-ins, or scale transitions. Every view switch, panel open, and state change is instant. Animation adds perceived latency — the app should feel like it responds at the speed of thought.

### What Stays

**Hover feedback only.** Hover state changes on interactive elements use a fast CSS transition to prevent flicker:

```css
transition: background-color 80ms ease-out;
```

80ms is below the threshold of conscious perception. This is smoothing, not animation.

### What's Removed

- No entrance/exit animations on views
- No staggered list reveals
- No slide-in for detail panel or command palette
- No scale/fade on overlays
- No checkPop bounce on checkbox completion
- No fadeInUp on task creation or removal

### Specific Behaviors

| Element | Behavior |
|---------|----------|
| Checkbox completion | Instant fill — border + background swap in one frame |
| Task title strikethrough | Instant — no color transition |
| View switch | Content swaps in one frame |
| Detail panel | Appears/disappears instantly |
| Command palette | Appears/disappears instantly on ⌘K / Escape |
| Section collapse | Content hides instantly (chevron may rotate) |
| Sidebar hover | 80ms bg-color transition (smoothing only) |
| Task item hover | 80ms bg-color transition (smoothing only) |
| Button hover | 80ms bg-color transition (smoothing only) |
| Focus ring | Instant |
| Task completion | Instant strikethrough, remains visible until next day |

### Drag-and-Drop UX (Things-inspired)

Drag-and-drop is the primary reordering mechanism for Today, Someday, and Project views.

**Grip handle:** On hover, a 6-dot grip icon appears at the left edge of the task row (before the checkbox). Color `--ink-quaternary`. This is the drag handle — clicking it initiates drag. Clicking the rest of the row still selects the task.

**Drag preview:** When dragging begins, the browser's native drag image shows the full task row (title + metadata). Apply `opacity: 0.7` to the drag source row to indicate it's being moved.

**Drop target gap:** As the user drags over other task rows, a visible gap opens between rows where the task will be inserted. This is achieved by adding a `margin-top` or a placeholder div at the drop position. The gap should be roughly the height of a task row (32px) so the user sees exactly where the task will land.

```css
.task-item.drag-source {
    opacity: 0.5;
}

.task-list-drop-indicator {
    height: 32px;
    background: var(--accent-subtle);
    border: 1px dashed var(--accent);
    border-radius: var(--radius-md);
    margin: 2px 0;
}
```

**Cross-section drag (Project view):** In project views, tasks can be dragged between sections. When hovering over a section header during drag, highlight the header to indicate the task will be moved to that section.

**Reorder on drop:** On drop, update the local signal immediately (optimistic) and call `PUT /tasks/{id}/reorder` with the new index.

---

## 9.5 Settings / Preferences

**Trigger:** `⌘,` or command palette "Open Preferences"

**Implementation:** A simple overlay panel (not a separate window) with:

1. **Server URL** — text input, default `http://localhost:8080`. Saved to `~/.config/atask/credentials.json`. Applied on next app restart or with a "Reconnect" button.

2. **Auto-archive** — toggle + duration selector:
   - "Archive completed tasks after: [Never / 1 day / 1 week / 1 month]"
   - When enabled, completed tasks older than the threshold are hidden from the logbook (or moved to a separate archive view)
   - Default: Never (show all completed tasks)

3. **Theme** — reserved for future dark mode toggle

Settings are stored locally in `~/.config/atask/settings.json`.

---

## 9.6 Date Formatting Rules

All dates displayed in the app follow these relative formatting rules:

| Condition | Format | Example |
|-----------|--------|---------|
| Today | "Today" | Today |
| Tomorrow | "Tomorrow" | Tomorrow |
| Yesterday | "Yesterday" | Yesterday |
| This week (within 7 days, future) | "DayName" | "Friday" |
| Last week (within 7 days, past) | "Last DayName" | "Last Monday" |
| This year | "Mon DD" | "Mar 25" |
| Different year | "Mon DD, YYYY" | "Mar 25, 2027" |
| Overdue (past deadline) | "Mon DD" in `--deadline-red` | "Mar 18" in red |

**Deadline display:** In task metadata, deadlines show as:
- Future: "Due Mon DD" or "Due Tomorrow" (neutral `--ink-tertiary`)
- Today: "Due Today" in `--today-star` (amber)
- Overdue: "Overdue · Mon DD" in `--deadline-red`

**Start date display:** In upcoming view section headers:
- "Tomorrow — Fri, Mar 21"
- "Saturday, Mar 22"
- "Next Week — Mon, Mar 24"

These rules apply everywhere dates appear: task metadata, detail panel fields, section headers, logbook groupings.

---

## 10. API-to-View Data Mapping

### Data Types (Rust)

```rust
use chrono::{NaiveDate, NaiveDateTime};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum TaskStatus { Pending, Completed, Cancelled }

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum Schedule { Inbox, Anytime, Someday }

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum ActorType { Human, Agent }

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Task {
    pub id: Uuid,
    pub title: String,
    pub notes: Option<String>,
    pub status: TaskStatus,
    pub schedule: Schedule,
    pub start_date: Option<NaiveDate>,
    pub deadline: Option<NaiveDate>,
    pub completed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub index: i32,
    pub today_index: Option<i32>,
    pub project_id: Option<Uuid>,
    pub section_id: Option<Uuid>,
    pub area_id: Option<Uuid>,
    pub location_id: Option<Uuid>,
    pub recurrence_rule: Option<RecurrenceRule>,
    pub tags: Vec<Uuid>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Project {
    pub id: Uuid,
    pub title: String,
    pub notes: Option<String>,
    pub status: TaskStatus,
    pub schedule: Schedule,
    pub start_date: Option<NaiveDate>,
    pub deadline: Option<NaiveDate>,
    pub completed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub index: i32,
    pub area_id: Option<Uuid>,
    pub tags: Vec<Uuid>,
    pub auto_complete: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Section {
    pub id: Uuid,
    pub title: String,
    pub project_id: Uuid,
    pub index: i32,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Area {
    pub id: Uuid,
    pub title: String,
    pub index: i32,
    pub archived: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Tag {
    pub id: Uuid,
    pub title: String,
    pub parent_id: Option<Uuid>,
    pub shortcut: Option<String>,
    pub index: i32,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ChecklistItem {
    pub id: Uuid,
    pub title: String,
    pub status: TaskStatus,  // Pending or Completed only
    pub task_id: Uuid,
    pub index: i32,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Activity {
    pub id: Uuid,
    pub task_id: Uuid,
    pub actor_id: String,
    pub actor_type: ActorType,
    pub activity_type: ActivityType,
    pub content: String,
    pub created_at: NaiveDateTime,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum ActivityType {
    Comment, ContextRequest, Reply, Artifact,
    StatusChange, Decomposition,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RecurrenceRule {
    pub mode: String,        // "fixed" | "after_completion"
    pub interval: u32,
    pub unit: String,        // "day" | "week" | "month"
    pub end: Option<RecurrenceEnd>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RecurrenceEnd {
    pub date: Option<NaiveDate>,
    pub count: Option<u32>,
}
```

### View → API Endpoint Mapping

| View | Primary Endpoint | Sort Key | Filter |
|------|-----------------|----------|--------|
| Inbox | `GET /views/inbox` | `index ASC` | `schedule = inbox` |
| Today | `GET /views/today` | `today_index ASC` | `schedule = anytime, start_date <= today` |
| Upcoming | `GET /views/upcoming` | `start_date ASC` | `start_date > today` |
| Someday | `GET /views/someday` | `index ASC` | `schedule = someday` |
| Logbook | `GET /views/logbook` | `completed_at DESC` | `status = completed \| cancelled` |
| Project | `GET /tasks?project_id={id}` | `index ASC` | `project_id = {id}` |

### Sidebar Data Requirements

```rust
// On app load, fetch in parallel:
GET /views/inbox           → count for badge
GET /views/today           → count for badge
GET /projects              → list for sidebar
GET /areas                 → list for sidebar
```

### SSE Subscription

```rust
// Subscribe to all events for real-time updates
GET /events/stream?topics=task.*,project.*,activity.added
```

**On event received:**
- `task.created` → add to appropriate view signal, increment sidebar badge
- `task.completed` → move from current view to logbook, decrement badge
- `task.reopened` → move from logbook back to appropriate view
- `task.scheduled_today` → add to Today view, remove from Inbox if present
- `task.deferred` → move to Someday view
- `activity.added` → append to detail panel if viewing that task

### Optimistic Updates

For snappy UX, update local state immediately on user action, then reconcile when the API responds:

```rust
// Example: completing a task
fn complete_task(task_id: Uuid) {
    // 1. Optimistic: update local signal immediately
    tasks.write().update_status(task_id, TaskStatus::Completed);
    
    // 2. Fire API call
    spawn(async move {
        let result = api::complete_task(task_id).await;
        if result.is_err() {
            // 3. Rollback on failure
            tasks.write().update_status(task_id, TaskStatus::Pending);
            // Show toast error
        }
    });
}
```

### API Client Configuration

```rust
pub struct ApiClient {
    base_url: String,          // e.g. "http://localhost:8080"
    token: Option<String>,     // JWT for user auth
    api_key: Option<String>,   // API key for agent auth
    client: reqwest::Client,
}
```

All requests include:
- `Authorization: Bearer {token}` or `X-API-Key: {api_key}`
- `Content-Type: application/json`
- `Accept: application/json`

**Response formats (two patterns):**
- **GET/list endpoints** return bare JSON arrays or objects: `[{Task}, ...]` or `{Project}`
- **Mutation endpoints** return an event envelope: `{"event": "task.created", "data": {...}}`

### Additional Endpoints (added for client support)

| Endpoint | Method | Body | Purpose |
|----------|--------|------|---------|
| `/tasks/{id}/today-index` | PUT | `{"index": 5}` or `{"index": null}` | Set/clear today ordering |
| `/tasks/{id}/reopen` | POST | — | Reopen completed/cancelled task to pending |
| `/projects/{id}/color` | PUT | `{"color": "#4670a0"}` | Set project color for sidebar dot |
| `/projects/{id}/sections/{sid}/reorder` | PUT | `{"index": 3}` | Reorder section within project |

### Tag Hydration

`GET /tasks/{id}` returns a `Tags` field with an array of tag IDs (strings). List endpoints (`GET /views/*`, `GET /tasks?...`) do NOT hydrate tags for performance — fetch them separately when opening the detail panel.

### Status & Schedule Integers

| Status | Value | | Schedule | Value |
|--------|-------|-|----------|-------|
| Pending | 0 | | Inbox | 0 |
| Completed | 1 | | Anytime | 1 |
| Cancelled | 2 | | Someday | 2 |

### SSE Event Types (complete list)

Task: `task.created`, `task.completed`, `task.cancelled`, `task.reopened`, `task.deleted`, `task.title_changed`, `task.notes_changed`, `task.scheduled_today`, `task.deferred`, `task.moved_to_inbox`, `task.start_date_set`, `task.deadline_set`, `task.deadline_removed`, `task.moved_to_project`, `task.removed_from_project`, `task.moved_to_section`, `task.moved_to_area`, `task.tag_added`, `task.tag_removed`, `task.location_set`, `task.recurrence_set`, `task.reordered`, `task.today_index_set`

Project: `project.created`, `project.completed`, `project.cancelled`, `project.deleted`, `project.title_changed`, `project.notes_changed`, `project.tag_added`, `project.moved_to_area`, `project.deadline_set`, `project.color_changed`

Section: `section.created`, `section.deleted`, `section.renamed`, `section.reordered`

SSE data format: `{"entity_type":"task","entity_id":"uuid","actor_id":"uuid",...}` — entity fields always included.

---

## Appendix: Design Decisions Log

| Decision | Rationale |
|----------|-----------|
| Bone ivory palette (#f6f5f2) not cool grey | Warm but not Anthropic-beige. Ivory reads as paper without the parchment tan. |
| Desaturated blue accent (#4670a0) not vivid blue | Quieter than the original #3a78ce. Feels more native-macOS, less SaaS. |
| Atkinson Hyperlegible over SF Pro | Legibility-first. Distinctive without being decorative. Works at all sizes. |
| Amber for Today (#c88c30), not red or blue | Urgent but not alarming. Today is a positive space, not a warning. Slightly deeper than original to pair with Bone canvas. |
| Purple for agents, not a separate UI | Agents are collaborators, not a separate product. Same stream, different tint. |
| No priority colors/flags | Position in list IS priority. Today IS the priority list. Honors constraint from domain model. |
| Circular checkboxes for tasks, square for checklist | Visual hierarchy: tasks are the primary unit, checklist items are secondary sub-steps. |
| Single-line 32px task rows, not two-line | Feels like a table — dense, scannable, fast. Title and metadata on the same horizontal axis. Title truncates; metadata pins right. |
| No animations whatsoever | Animations feel slow. Every state change is instant. Only hover feedback uses 80ms bg-color smoothing. |
| 340px detail panel, not modal | Always-visible context. Modal would break flow. Things uses same pattern. |
| Command palette over menus | Agent-era interaction: type intent, not navigate menus. Also better for keyboard-first users. |
| WebView (Dioxus) over native SwiftUI | Cross-platform from day one. CSS tokens map 1:1 from design library. Rust codebase aligned with existing tools. |
