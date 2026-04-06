import { atom, computed } from 'nanostores';
import type { TaskLink } from '../types';

export const $taskLinks = atom<TaskLink[]>([]);

export const $linksByTaskId = computed($taskLinks, (links) => {
  const map = new Map<string, Set<string>>();
  for (const link of links) {
    // Store bidirectional lookups
    if (!map.has(link.taskId)) map.set(link.taskId, new Set());
    map.get(link.taskId)!.add(link.linkedTaskId);
    if (!map.has(link.linkedTaskId)) map.set(link.linkedTaskId, new Set());
    map.get(link.linkedTaskId)!.add(link.taskId);
  }
  return map;
});
