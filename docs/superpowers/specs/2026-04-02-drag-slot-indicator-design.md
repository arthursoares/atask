# Drag Slot Indicator Design

## Context

The current drag-and-drop affordance for task reordering uses an in-row line indicator rendered by `DragIndicator` inside `TaskRow`. This gives some feedback, but it still reads as “hovering a row” more than “dropping into a slot between rows.”

The user wants the destination to read as a real landing position, with a visible placeholder or slot showing where the dragged item will end up. They also want the same clarity applied to the sidebar, not just task lists.

## Goal

Replace the current line-style drag affordance with a clearer half-row slot placeholder that shows the insertion position between items in task lists and in the sidebar.

## Non-Goals

- No rewrite of the existing drag state model unless required by a small implementation detail.
- No redesign of drag-and-drop semantics beyond improving the insertion affordance.
- No unrelated list or sidebar layout changes.

## Chosen Approach

Use a half-row slot placeholder rendered between items, rather than an indicator drawn inside the currently hovered row.

This approach was chosen because:

- it communicates destination more explicitly than a line alone
- it matches the user’s request for a visible slot
- it keeps list motion lighter than a full-row placeholder
- it can likely reuse the existing `dropIndex` state already used by task reordering

## Behavior

### Task Lists

When dragging a task over a reorderable list:

- show a half-row placeholder at the destination index
- render the placeholder between rows, including before the first row
- do not render a misleading placeholder on the dragged row itself
- keep the slot visually lightweight but explicit, using a bordered or dashed insertion surface with subtle fill

The slot should read as “the task will land here,” not “this row is active.”

### Sidebar

Apply the same slot concept to sidebar drag targets for reorderable items such as projects or areas.

The sidebar version should:

- use the same visual language as task-list slots
- fit the denser sidebar layout
- avoid relying only on row highlight states

It does not need to share the exact same component implementation if the sidebar DOM structure differs, but it should communicate the same insertion model.

## Proposed Structure

### Task Lists

Update the reorderable task-list views that already use `useDragReorder`:

- `atask-v4/src/views/InboxView.tsx`
- `atask-v4/src/views/TodayView.tsx`
- `atask-v4/src/views/SomedayView.tsx`
- `atask-v4/src/views/project-view/ProjectTaskList.tsx`

Add a shared presentational slot component near the task-row layer, such as:

- `atask-v4/src/components/task-row/DropSlot.tsx`

This component should be responsible only for rendering the half-row insertion placeholder.

`TaskRow.tsx` should stop owning the insertion marker directly once the slot is rendered between rows. The old `DragIndicator` can be removed or reduced if it is no longer needed.

### Sidebar

Update the sidebar drag UI around:

- `atask-v4/src/components/sidebar/SidebarParts.tsx`
- `atask-v4/src/components/Sidebar.tsx` if needed for list assembly

Render sidebar insertion slots in the list structure rather than only styling the hovered row as a drag target.

## Data Flow

Keep the existing drag state model where possible.

For task lists, the key input remains:

- current drag item id
- current `dropIndex`

The rendering change is:

- list views render a `DropSlot` before or between rows based on `dropIndex`
- rows no longer need to visually fake the insertion position inside the row itself

For the sidebar, the same pattern should apply if a current insertion target/index already exists. If sidebar drag currently works only by row-over state, introduce the smallest additional state needed to render an insertion slot cleanly.

## Styling

Add centralized slot styling in:

- `atask-v4/src/theme.css`

Design constraints:

- half-row height, not full-row height
- clear but not heavy
- works against the current list and sidebar backgrounds
- consistent across task lists and sidebar

## Verification

Required verification for the implementation pass:

- `npm run build`
- `npm run build-storybook`
- manual drag smoke checks for:
  - Inbox
  - Today morning section
  - Today evening section
  - Someday
  - project task list
  - sidebar project or area dragging
- edge-position checks:
  - before the first item
  - between items
  - near the end of the list
- confirm the dragged item itself does not also show a confusing destination marker

## Rollout Plan

1. Implement task-list drop slots using the existing reorder state.
2. Remove or simplify the old in-row drag indicator.
3. Apply the same slot language to sidebar drag targets.
4. Verify build, Storybook build, and manual drag behavior.

## Open Decisions Resolved

- Use a half-row slot, not a full-row slot.
- Prefer real insertion-slot rendering between rows over re-skinning the current in-row line.
- Include the sidebar in the same pass.

