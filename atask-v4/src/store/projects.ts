import { atom, computed } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { Project } from '../types';

export const $projects = atom<Project[]>([]);

export const $activeProjects = computed($projects, (projects) =>
  projects.filter((p) => p.status === 0).sort((a, b) => a.index - b.index),
);

export function useActiveProjects(): Project[] {
  return useStore($activeProjects);
}
