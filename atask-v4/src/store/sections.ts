import { atom } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { Section } from '../types';

export const $sections = atom<Section[]>([]);

// Parameterized selector — returns a new computed per projectId
// For React, use the hook wrapper below
export function useSectionsForProject(projectId: string): Section[] {
  // Direct filtering — nanostores useStore on the atom recomputes on change
  const sections = useStore($sections);
  return sections
    .filter((sec) => sec.projectId === projectId && !sec.archived)
    .sort((a, b) => a.index - b.index);
}
