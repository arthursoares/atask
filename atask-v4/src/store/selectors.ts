import { computed } from 'nanostores';
import { useStore } from '@nanostores/react';
import { $tasks, $tagsByTaskId } from './tasks';
import { $activeTagFilters } from './ui';
import type { Task } from '../types';

// --- Helpers ---

function todayDateStr(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function isCompletedToday(task: Task): boolean {
  if (!task.completedAt) return false;
  return task.completedAt.slice(0, 10) === todayDateStr();
}

function passesTagFilter(
  taskId: string,
  activeTagFilters: Set<string>,
  tagsByTaskId: Map<string, Set<string>>,
): boolean {
  if (activeTagFilters.size === 0) return true;
  const taskTagIds = tagsByTaskId.get(taskId);
  if (!taskTagIds) return false;
  for (const tagId of activeTagFilters) {
    if (!taskTagIds.has(tagId)) return false;
  }
  return true;
}

// --- Inbox ---

export const $inbox = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) =>
    tasks
      .filter(
        (t) =>
          t.schedule === 0 &&
          t.projectId === null &&
          (t.status === 0 || (t.status === 1 && isCompletedToday(t))) &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => {
        if (a.status !== b.status) return a.status - b.status;
        return a.index - b.index;
      }),
);

export function useInbox(): Task[] {
  return useStore($inbox);
}

// --- Today ---

export const $today = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) => {
    const today = todayDateStr();
    return tasks
      .filter(
        (t) =>
          t.schedule === 1 &&
          (t.status === 0 || (t.status === 1 && isCompletedToday(t))) &&
          (t.todayIndex != null ||
            (t.startDate && t.startDate.slice(0, 10) <= today) ||
            isCompletedToday(t)) &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => {
        if (a.status !== b.status) return a.status - b.status;
        return (a.todayIndex ?? a.index) - (b.todayIndex ?? b.index);
      });
  },
);

export const $todayMorning = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) => {
    const today = todayDateStr();
    return tasks
      .filter(
        (t) =>
          t.schedule === 1 &&
          t.timeSlot !== 'evening' &&
          (t.status === 0 || (t.status === 1 && isCompletedToday(t))) &&
          (t.todayIndex != null ||
            (t.startDate && t.startDate.slice(0, 10) <= today) ||
            isCompletedToday(t)) &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => {
        if (a.status !== b.status) return a.status - b.status;
        return (a.todayIndex ?? a.index) - (b.todayIndex ?? b.index);
      });
  },
);

export function useTodayMorning(): Task[] {
  return useStore($todayMorning);
}

export const $todayEvening = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) => {
    const today = todayDateStr();
    return tasks
      .filter(
        (t) =>
          t.schedule === 1 &&
          t.timeSlot === 'evening' &&
          (t.status === 0 || (t.status === 1 && isCompletedToday(t))) &&
          (t.todayIndex != null ||
            (t.startDate && t.startDate.slice(0, 10) <= today) ||
            isCompletedToday(t)) &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => {
        if (a.status !== b.status) return a.status - b.status;
        return (a.todayIndex ?? a.index) - (b.todayIndex ?? b.index);
      });
  },
);

export function useTodayEvening(): Task[] {
  return useStore($todayEvening);
}

// --- Upcoming ---

export interface UpcomingGroup {
  date: string;
  tasks: Task[];
}

export const $upcoming = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) => {
    const today = todayDateStr();
    const upcoming = tasks
      .filter(
        (t) =>
          t.schedule === 1 &&
          t.status === 0 &&
          t.startDate != null &&
          t.startDate.slice(0, 10) > today &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => {
        const dateCompare = (a.startDate ?? '').localeCompare(b.startDate ?? '');
        if (dateCompare !== 0) return dateCompare;
        return a.index - b.index;
      });

    const groups: UpcomingGroup[] = [];
    for (const task of upcoming) {
      const date = task.startDate!.slice(0, 10);
      const last = groups[groups.length - 1];
      if (last && last.date === date) {
        last.tasks.push(task);
      } else {
        groups.push({ date, tasks: [task] });
      }
    }
    return groups;
  },
);

export function useUpcoming(): UpcomingGroup[] {
  return useStore($upcoming);
}

// --- Someday ---

export const $someday = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) =>
    tasks
      .filter(
        (t) =>
          t.schedule === 2 &&
          t.status === 0 &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => a.index - b.index),
);

export function useSomeday(): Task[] {
  return useStore($someday);
}

// --- Logbook ---

export const $logbook = computed(
  [$tasks, $activeTagFilters, $tagsByTaskId],
  (tasks, filters, tagMap) =>
    tasks
      .filter(
        (t) =>
          (t.status === 1 || t.status === 2) &&
          passesTagFilter(t.id, filters, tagMap),
      )
      .sort((a, b) => (b.completedAt ?? '').localeCompare(a.completedAt ?? '')),
);

export function useLogbook(): Task[] {
  return useStore($logbook);
}

// --- Tasks for project ---

export function useTasksForProject(projectId: string): Task[] {
  const tasks = useStore($tasks);
  const filters = useStore($activeTagFilters);
  const tagMap = useStore($tagsByTaskId);
  return tasks
    .filter(
      (t) =>
        t.projectId === projectId &&
        (t.status === 0 || (t.status === 1 && isCompletedToday(t))) &&
        passesTagFilter(t.id, filters, tagMap),
    )
    .sort((a, b) => {
      if (a.status !== b.status) return a.status - b.status;
      return a.index - b.index;
    });
}
