import { atom } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { Tag } from '../types';
import { $tagsByTaskId } from './tasks';

export const $tags = atom<Tag[]>([]);

export function useTagsForTask(taskId: string): Tag[] {
  const tags = useStore($tags);
  const tagsByTaskId = useStore($tagsByTaskId);
  const tagIds = tagsByTaskId.get(taskId);
  if (!tagIds || tagIds.size === 0) return [];
  return tags.filter((tag) => tagIds.has(tag.id));
}
