import { atom, computed } from 'nanostores';
import type { Task, TaskTag } from '../types';

export const $tasks = atom<Task[]>([]);
export const $taskTags = atom<TaskTag[]>([]);

export const $tagsByTaskId = computed($taskTags, (taskTags) => {
  const map = new Map<string, Set<string>>();
  for (const tt of taskTags) {
    let set = map.get(tt.taskId);
    if (!set) {
      set = new Set();
      map.set(tt.taskId, set);
    }
    set.add(tt.tagId);
  }
  return map;
});
