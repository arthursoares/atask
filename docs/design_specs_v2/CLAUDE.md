# CLAUDE.md — atask SwiftUI App (v3)

## What This Project Is

atask is an AI-first personal task manager. This is the **SwiftUI macOS client** — local-first, works offline, optionally syncs with the atask Go API.

## Design Spec

**Read `DESIGN.md` before writing any UI code.** It covers tokens, components, views, shortcuts, data layer, and sync.

**`atask-screens-validation.html`** is the visual reference. Open in a browser. Click "Inline Editor" in the sidebar for the expanded card examples.

## Architecture: Local-First

Local SQLite (GRDB.swift) is the source of truth. All mutations write to local DB first — instant, no network. Server sync is optional (built last).

- Single `@Observable TaskStore` holds everything
- Views are computed properties on TaskStore (inbox, today, upcoming, etc.)
- Never fetch from API for UI updates. API is sync, not source of truth.
- `pendingOps` table queues mutations for background server sync

## Key Patterns

**One store, computed views.** No separate state per view. `store.inbox`, `store.today`, `store.todayEvening` — all computed from the same `tasks` array.

**One expanded card at a time.** `store.expandedTaskId`. Setting it collapses any other.

**Inline editor is primary.** Click a task → expand to card (title + notes + action bar). This is where scheduling, tagging, and project assignment happen. Detail panel (Enter/double-click) is the deep-dive for checklist + activity.

**Context-aware creation.** `⌘N` in Today creates with schedule=1. In Inbox → schedule=0. In a project → projectId set.

## Aesthetic Rules

- **Theme:** "Bone" — `Theme.canvas` (#f6f5f2), `Theme.accent` (#4670a0). Use `enum Theme` for all colors.
- **Font:** Atkinson Hyperlegible via `Font.atkinson(size, weight:)`. Never system font for content.
- **Task rows:** 32pt, single-line. Checkbox + title + right-aligned metadata.
- **Checkboxes:** Circular 20pt for tasks (amber in Today), square 16pt for checklist items.
- **No animations.** Never `withAnimation`. Never `.animation()`. Suppress implicit with `.animation(.none, value:)`.
- **No priority flags.** Position = priority.

## Keyboard Shortcuts (Things-compatible)

**Critical differences from standard macOS:**
- `⌘K` = **Complete task** (NOT command palette)
- `⇧⌘O` = Command palette
- `⌘S` = Show When picker (NOT save — saves are automatic/local)
- `⌘E` = This Evening
- `Space` = New task below selection (when list focused, not in text fields)
- `Return` = Open inline editor
- `⌘Return` = Save and close inline editor

## Models

Int-based status/schedule. GRDB records (`Codable + FetchableRecord + PersistableRecord`).

```
status:   0=pending, 1=completed, 2=cancelled
schedule: 0=inbox, 1=anytime, 2=someday
timeSlot: nil, "morning", "evening"
```

## Sidebar Hierarchy

Areas are non-selectable Section headers. Projects nest under their area. Standalone projects (areaId == nil) appear after all areas.

## Don't

- Don't use `ObservableObject` / `@Published`. Use `@Observable` (Swift 5.9+).
- Don't create separate state objects per view. One TaskStore, computed views.
- Don't call API for UI updates. Local DB is truth. API is sync.
- Don't use `withAnimation`. No `.animation()`. No `.transition()`.
- Don't use sheets/modals for task editing. Inline editor (expanded card) or detail panel.
- Don't add priority fields. Position = priority.
- Don't use system font for content. Atkinson Hyperlegible everywhere.
- Don't make areas selectable in the sidebar. They're headers only.
- Don't make task items two-line. 32pt single row.
- Don't use ⌘K for command palette. That's Complete Task. Use ⇧⌘O.

## Interaction Model: Inline First

Primary editing: click task → expand to card (title + notes + bottom action bar). Popovers for When, Project, Tags.
Detail panel: Enter/double-click for full view (checklist, activity).
Fast creation: ⌘N → type → Escape → repeat.
