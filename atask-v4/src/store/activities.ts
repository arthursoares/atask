import { atom } from 'nanostores';
import { useStore } from '@nanostores/react';
import { useMemo } from 'react';
import type { Activity } from '../types';

export const $activities = atom<Activity[]>([]);

export function useActivitiesForTask(taskId: string): Activity[] {
  const activities = useStore($activities);
  return useMemo(
    () => activities
      .filter((a) => a.taskId === taskId)
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    [activities, taskId],
  );
}
