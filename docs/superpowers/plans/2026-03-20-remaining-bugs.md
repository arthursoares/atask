# Remaining Bugs & Issues — v2 Client

> Updated: 2026-03-20 end of session

## Fixed This Session
- [x] Date picker spamming API (13+ events per date change)
- [x] Empty project can't add tasks (NewTaskInline missing)
- [x] Dates showing as ISO strings instead of relative ("Due Tomorrow")
- [x] Duplicate traffic light buttons
- [x] Detail panel saves not updating UI (removed refetch, fire-and-forget)
- [x] Schedule picker not visually updating on click (local draft signal)
- [x] Section headers were collapsible (should be static dividers)
- [x] Grip/reorder icon creating bad alignment

## Fixed
- [x] Tags can't be added — optimistic add/remove with hydrated fetch
- [x] Tags not unique by name — migration 003: unique index
- [x] Window title bar — document::Title
- [x] View query rules — inbox excludes tasks with dates, someday excludes dated tasks, upcoming excludes someday

## Still Open (moved to Plan 3)
- [ ] **Checklist count in task row** — "3/5" in metadata. Needs API checklist_count fields or per-task fetch.
- [ ] **Inline task editing** — Things-style expanded card in list.

## Architecture Issues (Plan 3)
- [ ] Local-first sync — spec written at `docs/superpowers/specs/2026-03-20-local-first-sync.md`
- [ ] All mutations should be local-first, not API-first
- [ ] SSE handles inbound sync from server

## Not Yet Built (Plan 3/4)
- [ ] Command palette (⌘K)
- [ ] Global keyboard shortcuts
- [ ] Drag-and-drop with gap indicator (tasks + projects between areas)
- [ ] Right-click context menus (tasks, projects, sections, areas)
- [ ] Assign project to area (via context menu or drag onto area in sidebar)
- [ ] Move project to area API: `PUT /projects/{id}/area` body `{"id":"area-uuid"}`
- [ ] Settings page (server URL, auto-archive)
- [ ] Inline task editing (Things-style expanded card)
