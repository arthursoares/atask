# Pointer Reorder Design

Date: 2026-04-02

## Summary

Replace the current native HTML5 drag/drop reorder behavior in `atask-v4` with a pointer-driven reorder system for mouse and trackpad. The existing half-row slot placeholder remains the visual insertion affordance, but gesture tracking and insertion-index calculation move to a custom pointer-based model that works reliably in the Tauri desktop app.

This pass covers:

- task reordering in list views
- area/project reordering in the sidebar
- real pointer-action test coverage for reorder

This pass explicitly does not cover:

- task-to-sidebar move dragging
- touch/mobile/tablet behavior

## Problem

The current reorder implementation depends on native `dragstart` / `dragover` / `drop` events on DOM nodes. Synthetic tests can trigger those handlers, but real mouse/trackpad dragging in the Tauri WKWebView app is not behaving reliably enough for actual use. That means the current slot placeholder design is visually implemented but functionally blocked by the underlying drag model.

The issue is not the slot rendering itself. The issue is the transport mechanism used to drive reorder state.

## Goals

- make task and sidebar reordering work reliably with mouse and trackpad in the Tauri app
- preserve the slot placeholder as the visual destination indicator
- avoid depending on browser-native HTML5 drag/drop for reorder
- make automated reorder tests reflect real pointer interaction rather than synthetic `DragEvent` dispatch

## Non-Goals

- redesigning task-to-sidebar move behavior
- adding mobile/touch support
- changing reorder persistence semantics in the store/backend

## Approach

Use a custom pointer-based reorder controller instead of HTML5 drag/drop.

At a high level:

1. A row begins a reorder gesture on pointer press.
2. A list-level controller tracks pointer movement.
3. The controller computes a destination insertion index from row geometry.
4. The list renders the slot placeholder at that computed index.
5. Pointer release commits the reorder through the existing store mutation path.
6. Pointer cancel or `Escape` aborts the gesture and clears the slot.

This keeps the user-facing interaction simple while removing dependence on native drag-transfer behavior.

## UI Behavior

### Task Lists

- Pressing on a task row starts a reorder gesture.
- Moving vertically through the list updates the insertion slot.
- The slot placeholder appears between rows, including before the first row and after the last row.
- The dragged task gets a lightweight visual “lifted” state.
- Releasing commits the reorder.
- Releasing without a valid movement can be treated as a no-op reorder.
- `Escape` cancels the reorder and clears all temporary state.

### Sidebar

- Project rows and area labels can begin reorder gestures.
- Sidebar insertion uses the existing compact slot visual language.
- Reordering remains constrained by type:
  - areas reorder within the area list
  - projects reorder within their current area grouping or root grouping
- This pass does not preserve task-to-sidebar native drop behavior as part of the new pointer reorder flow. That path remains out of scope for this spec.

## Architecture

### New Reorder Hook

Introduce a pointer-based reorder hook that replaces `useDragReorder` for reorderable lists.

Responsibilities:

- track active dragged item id
- track current insertion index
- track whether a gesture is active
- expose pointer-start handlers for rows
- expose container or row-registration mechanisms for geometry measurement
- commit reorder on pointer release
- cancel reorder on pointer cancel or `Escape`

The hook should be generic over item `id` plus current ordered items so it can be reused by:

- Inbox
- Today morning/evening
- Someday
- Project task list
- Sidebar project groups
- Sidebar area list

### Geometry Model

Insertion index should be computed from actual row positions rather than inferred from native drag target events.

Recommended model:

- collect the vertical centers of visible reorderable rows
- compare the live pointer Y against those centers
- derive the insertion index as the first row whose center is below the pointer
- if the pointer is below all row centers, use the terminal insertion index

This produces stable between-row insertion behavior and maps directly to the existing slot placeholder UI.

### Rendering

List views continue to render slot components between rows, but visibility now depends on pointer reorder state instead of `DragEvent` state.

Rows should no longer use:

- `draggable`
- `onDragStart`
- `onDragEnd`
- native `dataTransfer`

Instead they receive:

- pointer start handler
- optional “dragging” visual state

### Commit Path

Do not change persistence behavior.

Continue committing reorders through the existing action layer:

- tasks: `reorderTasks`
- Today sections: `setTodayIndex`
- projects: `reorderProjects`
- areas: `reorderAreas`

The new hook changes only interaction/control flow, not store semantics.

## Testing

Replace synthetic `DragEvent` reorder coverage with real pointer-action coverage through WDIO.

Minimum coverage for this pass:

- task reorder in Inbox using real pointer movement
- sidebar project reorder using real pointer movement

Tests should verify:

- the slot appears during pointer movement
- releasing commits the expected new order
- the slot clears after commit/cancel

Synthetic DOM-dispatched drag tests are insufficient for this feature because they do not validate the real Tauri input path.

## Risks and Mitigations

### Risk: Geometry drift while lists rerender

Mitigation:

- compute geometry from currently rendered rows only
- refresh measurements when gesture starts and on meaningful movement if needed

### Risk: Conflict with click/double-click selection behavior

Mitigation:

- begin reorder only after movement exceeds a small threshold
- keep plain click behavior unchanged when there is no drag movement

### Risk: Sidebar/task behavior diverges

Mitigation:

- share the same pointer reorder core and specialize only the rendering and commit adapters

## Rollout

1. Implement pointer reorder core.
2. Migrate task list views.
3. Migrate sidebar reorder.
4. Replace drag e2e tests with pointer-based tests.
5. Remove obsolete native drag/drop reorder code.

## Success Criteria

- users can reorder tasks in the desktop app with mouse/trackpad
- users can reorder sidebar areas/projects in the desktop app with mouse/trackpad
- slot placeholders appear in the right insertion position during pointer movement
- reorder tests use real pointer actions and pass against the Tauri app
