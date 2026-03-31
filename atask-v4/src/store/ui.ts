import { atom } from 'nanostores';
import type { ActiveView } from '../types';

export interface SyncStatusState {
  isSyncing: boolean;
  lastSyncAt: string | null;
  lastError: string | null;
  pendingOpsCount: number;
}

export const $syncStatus = atom<SyncStatusState>({
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

export function toggleTagFilter(tagId: string) {
  const next = new Set($activeTagFilters.get());
  if (next.has(tagId)) next.delete(tagId);
  else next.add(tagId);
  $activeTagFilters.set(next);
}

export function clearTagFilters() {
  $activeTagFilters.set(new Set());
}
