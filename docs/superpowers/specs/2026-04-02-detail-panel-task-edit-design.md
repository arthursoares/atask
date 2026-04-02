# Detail Panel And Task Edit Consolidation

## Context

The frontend branch already introduced a small design-system layer, Storybook, shared picker shells, and shared task edit hooks. The two largest remaining task-edit surfaces are still split between:

- `atask-v4/src/components/DetailPanel.tsx`
- `atask-v4/src/components/TaskInlineEditor.tsx`

These components now share draft persistence and picker state logic, but they still diverge in markup, presentation, and metadata control composition. The result is duplicated UI patterns for project, schedule, tags, notes, and task metadata while the richer panel-specific sections remain mixed into the main file.

## Goal

Refactor the task editing UI so `DetailPanel` and `TaskInlineEditor` share common task-edit building blocks without collapsing into one large mode-switching component.

## Non-Goals

- No behavior change to task persistence semantics beyond preserving current behavior.
- No redesign of checklist or activity functionality.
- No attempt to merge the panel and inline editor into a single component with conditional rendering.
- No broad store architecture rewrite outside the task-edit surface.

## Chosen Approach

Use shared task-edit subcomponents under `atask-v4/src/components/task-edit/`, while keeping `DetailPanel` and `TaskInlineEditor` as distinct surfaces with different layout and lifecycle behavior.

This is preferred over a single canonical editor because the current surfaces have meaningfully different responsibilities:

- `TaskInlineEditor` is a quick-edit surface optimized for in-list editing, auto-focus, outside-click close, and delete-on-empty behavior.
- `DetailPanel` is a richer inspection and editing surface optimized for labeled fields, checklist, activity, and persistent side-panel editing.

The refactor should consolidate overlapping controls and presentation, not erase the distinction between the two experiences.

## Surface Boundaries

### Shared Across Both Surfaces

The following should converge on shared components or shared rendering helpers:

- title editing field shape where practical
- project display and selection trigger
- schedule display and selection trigger
- tag display and tag picker trigger
- notes field rendering where practical
- shared field labels, metadata rows, and control affordances

### Unique To `TaskInlineEditor`

The inline editor remains responsible for:

- outside-click close
- `Escape` close behavior inside the list context
- empty-title delete on close
- checkbox/status behavior
- compact layout optimized for list editing
- repeat-rule control if it remains specific to the quick-edit bar

### Unique To `DetailPanel`

The detail panel remains responsible for:

- panel-level close behavior
- richer labeled layout
- direct start-date and deadline controls
- checklist section
- activity feed section
- any wider panel-specific structure and spacing

## Proposed Component Shape

Add or expand app-specific shared task-edit components under:

- `atask-v4/src/components/task-edit/`

Likely components:

- `TaskEditTitleField`
- `TaskEditMetaSection`
- `TaskEditProjectField`
- `TaskEditScheduleField`
- `TaskEditTagSection`
- `TaskEditNotesField`
- `TaskDateFields`

The exact names can vary during implementation, but the design constraint is:

- each shared component should have one clear purpose
- each shared component should accept resolved values and callbacks where possible
- parent surfaces should retain lifecycle ownership

## Data Flow

Behavioral state remains hook-based:

- `useTaskDraft` for debounced title and notes persistence
- `useTaskPickers` for picker visibility state

The shared components should be mostly presentational. They should receive:

- current task-derived values
- resolved `project` data where needed
- resolved `tags` where needed
- callback props for opening pickers and updating fields

Store reads should stay near the surface boundary unless a shared component is inherently task-aware. This keeps the reusable pieces easier to test and reason about.

## Error Handling And State Safety

This refactor is structural, so correctness requirements are mainly behavioral parity:

- no loss of current draft persistence behavior
- no loss of close semantics in the inline editor
- no regression in panel close behavior
- no regression in picker opening/closing behavior
- no regression in date clearing or task metadata updates

If implementation reveals hidden divergence between surfaces, preserve existing behavior first and only unify behavior where it is clearly intentional.

## Storybook And Design-System Relationship

Not every extracted component belongs in `src/ui/`. The task-edit components are application-specific composition primitives, so they should generally remain under `components/task-edit/`.

Promote a piece into `src/ui/` only if it becomes broadly reusable outside task editing and has a stable enough API to deserve design-system ownership.

If a shared component is stable and visually reusable, add or expand Storybook coverage. Otherwise, keep Storybook focused on lower-level design-system primitives.

## Verification

Required verification for the implementation pass:

- `npm run build`
- `npm run build-storybook`
- manual smoke check for task editing flows:
  - edit title
  - edit notes
  - change project
  - change schedule
  - add/remove tags
  - update/clear start date
  - update/clear deadline
  - confirm checklist and activity still render in the detail panel
  - confirm inline editor still closes and deletes empty drafts correctly

## Rollout Plan

1. Extract shared task-edit presentation components.
2. Migrate `DetailPanel` to the shared pieces first because it has the richer field surface.
3. Migrate `TaskInlineEditor` onto the shared pieces where the UI overlaps.
4. Keep layout and lifecycle behavior distinct in each parent component.
5. Verify builds and smoke-test task editing behavior.

## Open Decisions Resolved

- Approach: use shared subcomponents, not a single `mode`-driven editor.
- Product boundary: preserve the unique roles of panel and inline editor.
- Priority: improve both code reuse and visual consistency in the same pass.

