# atask Dioxus Client — Remaining Features

> **Status:** In progress. Updated 2026-03-20.

## Gap Analysis

The shell is built (13 tasks complete) but interactive editing is largely missing. The detail panel is display-only, there are no picker components, and some task actions don't work end-to-end.

## Priority Order

### P0: Core Task Actions — DONE
- [x] Fix task creation (NewTaskInline wired in all views, Today auto-schedules anytime)
- [x] Detail panel: title editing (ghost input, blur/Enter saves)
- [x] Detail panel: notes editing (textarea, blur saves)
- [x] Detail panel: schedule picker (Inbox/Today/Someday pills)
- [x] Detail panel: checklist item creation (input at bottom)

### P1: Picker Components — DONE
- [x] Schedule picker (inline pills in detail panel)
- [x] Project picker (`project_picker.rs` — dropdown with search)
- [x] Date picker (`date_picker.rs` — native date input + clear)
- [x] Tag picker (`tag_picker.rs` — toggle list with add/remove)

### P2: Inbox Triage — DONE
- [x] Hover quick-actions (★ Today, 💤 Someday, 📁 Project)
- [x] Optimistic removal + API call + refetch

### P3: Detail Panel Full Editing — DONE
- [x] Project field (click to open picker, API call)
- [x] Start date field (date picker, API call)
- [x] Deadline field (date picker, API call)
- [x] Tags field (pills + add/remove via tag picker)
- [ ] Recurrence display — deferred (read-only, low priority)

### P4: Project View Enhancements — PARTIALLY DONE
- [x] Add section (toolbar button → inline input → API)
- [ ] Drag tasks between sections — deferred

### P5: UX Polish — DONE
- [x] Confirmation dialog (for task deletion)
- [x] Error toasts (flash message on API failures)
- [x] Empty states (per view with proper messages)
- [ ] Loading indicators — deferred (low priority, data loads fast)

### Testing — DONE
- [x] Rust API integration tests (7 tests)
- [x] Playwright e2e test suite (8 tests)

### Remaining (deferred)
- [ ] Recurrence display in detail panel
- [ ] Drag tasks between sections in project view
- [ ] Loading skeleton/spinner
- [ ] Dark mode

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
