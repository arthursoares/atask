# atask Dioxus Client — Remaining Features

> **Status:** In progress. Updated 2026-03-20.

## Gap Analysis

The shell is built (13 tasks complete) but interactive editing is largely missing. The detail panel is display-only, there are no picker components, and some task actions don't work end-to-end.

## Priority Order

### P0: Core Task Actions (blocking basic usage)

- [ ] **Fix task creation** — NewTaskInline must call API and refresh view. Test in Today, Inbox, Someday, Project views.
- [ ] **Detail panel: title editing** — ghost input, on blur/Enter calls `update_task_title`
- [ ] **Detail panel: notes editing** — textarea, on blur calls `update_task_notes`
- [ ] **Detail panel: schedule picker** — Inbox/Today/Someday selector, calls `update_task_schedule`

### P1: Picker Components (needed for detail panel + inbox triage)

- [ ] **Schedule picker component** (`src/components/schedule_picker.rs`) — dropdown/popover with 3 options: Inbox, Today (Anytime), Someday. Used in detail panel and inbox hover actions.
- [ ] **Project picker component** (`src/components/project_picker.rs`) — searchable list of projects from ProjectState. On select calls `move_task_to_project`. Used in detail panel and inbox hover.
- [ ] **Date picker component** (`src/components/date_picker.rs`) — simple date input or calendar. Used for start date and deadline in detail panel.
- [ ] **Tag picker component** (`src/components/tag_picker.rs`) — list of tags from state, toggle on/off. Used in detail panel.

### P2: Inbox Triage (key workflow)

- [ ] **Inbox hover quick-actions** — on task hover, show action buttons:
  - ★ Schedule Today → `PUT /tasks/{id}/schedule {"schedule":"anytime"}`
  - 💤 Someday → `PUT /tasks/{id}/schedule {"schedule":"someday"}`
  - 📁 Move to Project → open project picker
  - 📅 Set Date → open date picker for start_date
- [ ] After any triage action, remove task from inbox and refetch

### P3: Detail Panel Full Editing

- [ ] **Project field** — click to open project picker, display current project with colored dot
- [ ] **Start date field** — click to open date picker, display formatted date or "None"
- [ ] **Deadline field** — click to open date picker
- [ ] **Tags field** — show tag pills + "Add" button opening tag picker, click × to remove
- [ ] **Checklist add item** — input at bottom of checklist, Enter to add via API
- [ ] **Recurrence display** — read-only for now (rule summary)

### P4: Project View Enhancements

- [ ] **Add section** — toolbar button opens inline input, calls `POST /projects/{id}/sections`
- [ ] **Drag tasks between sections** — drop on section header moves task to that section

### P5: UX Polish

- [ ] **Confirmation dialog** — for task/project deletion (⌫)
- [ ] **Error toasts** — flash message on API failures
- [ ] **Empty states** — proper empty state messages per view (match mockup)
- [ ] **Loading indicators** — subtle spinner or skeleton while fetching

## API Endpoints Used

| Action | Method | Endpoint | Body |
|--------|--------|----------|------|
| Create task | POST | `/tasks` | `{"title":"..."}` |
| Update title | PUT | `/tasks/{id}/title` | `{"title":"..."}` |
| Update notes | PUT | `/tasks/{id}/notes` | `{"notes":"..."}` |
| Update schedule | PUT | `/tasks/{id}/schedule` | `{"schedule":"inbox\|anytime\|someday"}` |
| Set start date | PUT | `/tasks/{id}/start-date` | `{"date":"2026-03-20"}` or `{"date":null}` |
| Set deadline | PUT | `/tasks/{id}/deadline` | `{"date":"2026-03-20"}` or `{"date":null}` |
| Move to project | PUT | `/tasks/{id}/project` | `{"id":"uuid"}` or `{"id":null}` |
| Move to section | PUT | `/tasks/{id}/section` | `{"id":"uuid"}` or `{"id":null}` |
| Add tag | POST | `/tasks/{id}/tags/{tagId}` | — |
| Remove tag | DELETE | `/tasks/{id}/tags/{tagId}` | — |
| Add checklist | POST | `/tasks/{id}/checklist` | `{"title":"..."}` |
| Complete checklist | POST | `/tasks/{id}/checklist/{itemId}/complete` | — |
| Create section | POST | `/projects/{id}/sections` | `{"title":"..."}` |

## Dioxus Patterns (from debugging)

- Signal reads MUST be inside `rsx!` for reactivity
- Writes from `spawn` async blocks may not trigger parent re-renders
- Use newtype wrappers for context signals shared between parent/child
- Optimistic updates: mutate local signal first, API call in spawn, refetch on completion
- No animations — instant state changes, only 80ms hover smoothing

## File Structure for New Components

```
src/components/
  schedule_picker.rs    ← new
  project_picker.rs     ← new
  date_picker.rs        ← new
  tag_picker.rs         ← new
  confirm_dialog.rs     ← new
  toast.rs              ← new
```
