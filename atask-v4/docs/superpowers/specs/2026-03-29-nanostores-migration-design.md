# Nanostores Migration Design

**Date:** 2026-03-29
**Scope:** Replace Zustand with Nanostores for state management in atask v4

## Problem

The current Zustand store is a 790-line monolith requiring `useShallow` in 19 files, a `useRef` memoization hack for `useUpcoming`, and manual `tagsByTaskId` rebuilds in every tag mutation. These patterns are error-prone (caused React Error #185 infinite re-render) and add cognitive overhead to every component.

## Solution

Replace with nanostores: atomic stores per entity, `computed()` for derived state, `batched()` for cascading mutations, plain async functions for actions.

## File Structure

```
store/
  tasks.ts        — $tasks, $taskTags, $tagsByTaskId (computed)
  projects.ts     — $projects, $activeProjects (computed)
  areas.ts        — $areas, $activeAreas (computed)
  sections.ts     — $sections, useSectionsForProject
  tags.ts         — $tags, useTagsForTask
  checklist.ts    — $checklistItems, useChecklistForTask
  ui.ts           — $activeView, $selectedTaskId, $selectedTaskIds,
                     $expandedTaskId, $showPalette, $showQuickMove,
                     $showSidebar, $activeTagFilters
  selectors.ts    — useInbox, useTodayMorning, useTodayEvening,
                     useUpcoming, useSomeday, useLogbook,
                     useTasksForProject
  mutations.ts    — all 47 async Tauri-calling actions
  index.ts        — re-exports everything
```

## Key Patterns

### Data atoms
One `atom()` per entity collection. Single source of truth.

```typescript
export const $tasks = atom<Task[]>([]);
```

### Computed atoms
Derived state auto-recomputes when dependencies change. Replaces manual rebuilds and the `useRef` hack.

```typescript
export const $tagsByTaskId = computed($taskTags, (taskTags) => { ... });
export const $inbox = computed([$tasks, $activeTagFilters, $tagsByTaskId], ...);
```

### UI atoms
Independent atoms per flag. No `useShallow` needed.

```typescript
export const $activeView = atom<ActiveView>('inbox');
export const $selectedTaskId = atom<string | null>(null);
```

### Mutations
Plain async functions using `batched()` for atomic multi-atom updates.

```typescript
export async function deleteProject(id: string) {
  await tauri.invokeDeleteProject(id);
  batched(() => {
    $tasks.set($tasks.get().filter(...));
    $sections.set($sections.get().filter(...));
    // ...all cascades in one batch
  });
}
```

### Component pattern
Mechanical transformation: `useShallow` destructure becomes individual atom subscriptions + direct function imports.

```typescript
// Before
const { selectedTaskId, createTask } = useStore(useShallow(s => ({ ... })));

// After
const selectedTaskId = useStore($selectedTaskId);
import { createTask } from '../store/mutations';
```

### Imperative access
`useStore.getState()` becomes `$atom.get()`. Used in event handlers (useKeyboard, CommandPalette, TaskRow).

## Dependencies

- Add: `nanostores`, `@nanostores/react`
- Remove: `zustand`

## Migration Scope

- Delete: `src/store.ts` (790 lines)
- Create: 10 files in `store/` (~600 lines total)
- Modify: 22 component/hook files (mechanical pattern swap)
- Test: 163 E2E tests validate no regressions

## Sync Integration Point

Each mutation function in `mutations.ts` becomes the natural hook for the Go API sync layer. A future `pendingOps` queue wraps the Tauri call, and inbound SSE events write directly to atoms via `batched()`.
