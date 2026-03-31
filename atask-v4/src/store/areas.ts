import { atom, computed } from 'nanostores';
import { useStore } from '@nanostores/react';
import type { Area } from '../types';

export const $areas = atom<Area[]>([]);

export const $activeAreas = computed($areas, (areas) =>
  areas.filter((a) => !a.archived).sort((a, b) => a.index - b.index),
);

export function useActiveAreas(): Area[] {
  return useStore($activeAreas);
}
