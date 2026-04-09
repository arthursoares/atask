import { atom, computed } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { ChecklistItem } from '../types';

export const $checklistItems = atom<ChecklistItem[]>([]);

export function useChecklistForTask(taskId: string): ChecklistItem[] {
  const items = useStore($checklistItems);
  return items
    .filter((ci) => ci.taskId === taskId)
    .sort((a, b) => a.index - b.index);
}

/**
 * Map of taskId → {done, total} for inline checklist-count badges in task
 * rows. Computed so TaskMeta can look up counts without subscribing to
 * the full checklist items list on every render.
 *
 * Matches the design spec in docs/superpowers/specs/2026-03-29-v4-tauri-
 * react-design.md line 310, which lists "checklist count" as part of
 * the task row meta.
 */
export interface ChecklistCount {
  done: number;
  total: number;
}

export const $checklistCountsByTaskId = computed($checklistItems, (items) => {
  const map = new Map<string, ChecklistCount>();
  for (const item of items) {
    const existing = map.get(item.taskId) ?? { done: 0, total: 0 };
    existing.total += 1;
    if (item.status === 1) existing.done += 1;
    map.set(item.taskId, existing);
  }
  return map;
});
