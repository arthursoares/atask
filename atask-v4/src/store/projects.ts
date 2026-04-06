import { atom, computed } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { Project, ProjectTag } from '../types';

export const $projects = atom<Project[]>([]);
export const $projectTags = atom<ProjectTag[]>([]);

export const $tagsByProjectId = computed($projectTags, (projectTags) => {
  const map = new Map<string, Set<string>>();
  for (const pt of projectTags) {
    let set = map.get(pt.projectId);
    if (!set) {
      set = new Set();
      map.set(pt.projectId, set);
    }
    set.add(pt.tagId);
  }
  return map;
});

export const $activeProjects = computed($projects, (projects) =>
  projects.filter((p) => p.status === 0).sort((a, b) => a.index - b.index),
);

export function useActiveProjects(): Project[] {
  return useStore($activeProjects);
}
