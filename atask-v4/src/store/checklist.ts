import { atom } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { ChecklistItem } from '../types';

export const $checklistItems = atom<ChecklistItem[]>([]);

export function useChecklistForTask(taskId: string): ChecklistItem[] {
  const items = useStore($checklistItems);
  return items
    .filter((ci) => ci.taskId === taskId)
    .sort((a, b) => a.index - b.index);
}
