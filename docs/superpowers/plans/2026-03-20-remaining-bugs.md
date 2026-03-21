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

## Still Broken
- [ ] **Tags can't be added** — clicking "+ Add" shows picker but clicking a tag may not fire API. Debug: add println to on_add handler, verify API call fires.
- [ ] **Tags not unique by name** — server allows duplicate tag names. Fix: add unique constraint or check-before-create in Go API.
- [ ] **Checklist count not in task row** — DESIGN.md says show "3/5" in metadata. Requires either API returning checklist counts per task, or client fetching per visible task (expensive).
- [ ] **Window title bar** — "atask" not showing in macOS title bar. Dioxus WebView may need explicit title setting.
- [ ] **Date format in detail panel** — date picker shows YYYY-MM-DD (HTML input limitation). Consider showing relative date label ABOVE the date input.
- [ ] **Logbook not refreshing** — completing tasks in other views may not update logbook. SSE should handle this but needs verification.
- [ ] **Inline task editing** — Things allows clicking a task in the list to edit it inline (expanded card). Not implemented. Design spec needs this added.

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
