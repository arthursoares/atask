import { atom } from 'nanostores';
import type { ActiveView, Task } from '../types';

/**
 * Sync phase state machine. Replaces the previous implicit "if no error and
 * zero pending ops, render green idle dot" logic, which displayed a
 * successful-looking state before the app had ever fetched its first status
 * or confirmed the server was reachable.
 *
 *   unknown      - initial value, before any fetch completes. Don't claim
 *                  anything about sync status.
 *   checking     - actively fetching status from the backend (or sync is
 *                  in progress).
 *   unconfigured - sync is disabled in settings (no server / API key).
 *                  Render a subtle hint instead of a success state.
 *   synced       - last fetch succeeded, sync is enabled, zero errors.
 *   error        - last sync returned an error; show the error indicator.
 */
export type SyncPhase = 'unknown' | 'checking' | 'unconfigured' | 'synced' | 'error';

export interface SyncStatusState {
  phase: SyncPhase;
  isSyncing: boolean;
  lastSyncAt: string | null;
  lastError: string | null;
  pendingOpsCount: number;
}

export const $syncStatus = atom<SyncStatusState>({
  phase: 'unknown',
  isSyncing: false,
  lastSyncAt: null,
  lastError: null,
  pendingOpsCount: 0,
});

export const $activeView = atom<ActiveView>('inbox');
export const $selectedTaskId = atom<string | null>(null);
export const $selectedTaskIds = atom<Set<string>>(new Set());
export const $expandedTaskId = atom<string | null>(null);
export const $showPalette = atom<boolean>(false);
export const $showQuickMove = atom<boolean>(false);
export const $showSearch = atom<boolean>(false);
export const $showSidebar = atom<boolean>(true);
export const $showShortcuts = atom<boolean>(false);
export const $activeTagFilters = atom<Set<string>>(new Set());

export interface TaskPointerDragState {
  activeTaskId: string | null;
  hoverTargetId: string | null;
}

export const $taskPointerDrag = atom<TaskPointerDragState>({
  activeTaskId: null,
  hoverTargetId: null,
});

export function startTaskPointerDrag(taskId: string) {
  $taskPointerDrag.set({ activeTaskId: taskId, hoverTargetId: null });
}

export function setTaskPointerHoverTarget(targetId: string | null) {
  $taskPointerDrag.set({ ...$taskPointerDrag.get(), hoverTargetId: targetId });
}

export function endTaskPointerDrag() {
  $taskPointerDrag.set({ activeTaskId: null, hoverTargetId: null });
}

export function setActiveView(view: ActiveView) {
  $activeView.set(view);
}

export function selectTask(taskId: string | null, options?: { preserveMultiSelection?: boolean }) {
  if (taskId !== null && !options?.preserveMultiSelection) {
    $selectedTaskIds.set(new Set());
  }
  $selectedTaskId.set(taskId);
}

export function clearSelectedTask() {
  $selectedTaskId.set(null);
}

export function clearSelectedTasks() {
  $selectedTaskIds.set(new Set());
}

export function openTaskEditor(taskId: string) {
  $expandedTaskId.set(taskId);
}

export function closeTaskEditor() {
  $expandedTaskId.set(null);
}

export function toggleTaskSelection(taskId: string) {
  const next = new Set($selectedTaskIds.get());
  if (next.has(taskId)) {
    next.delete(taskId);
  } else {
    next.add(taskId);
  }
  $selectedTaskIds.set(next);
}

export function selectTaskRange(taskId: string, taskList: Task[]) {
  const currentSelectedIds = $selectedTaskIds.get();
  const currentSelectedId = $selectedTaskId.get();
  const lastId =
    currentSelectedId || (currentSelectedIds.size > 0 ? [...currentSelectedIds].pop() ?? null : null);

  if (!lastId) {
    selectTask(taskId);
    return;
  }

  const lastIdx = taskList.findIndex((task) => task.id === lastId);
  const currentIdx = taskList.findIndex((task) => task.id === taskId);

  if (lastIdx < 0 || currentIdx < 0) {
    selectTask(taskId);
    return;
  }

  const start = Math.min(lastIdx, currentIdx);
  const end = Math.max(lastIdx, currentIdx);
  const range = taskList.slice(start, end + 1).map((task) => task.id);
  $selectedTaskIds.set(new Set([...currentSelectedIds, ...range]));
}

export function toggleTagFilter(tagId: string) {
  const next = new Set($activeTagFilters.get());
  if (next.has(tagId)) next.delete(tagId);
  else next.add(tagId);
  $activeTagFilters.set(next);
}

export function clearTagFilters() {
  $activeTagFilters.set(new Set());
}
