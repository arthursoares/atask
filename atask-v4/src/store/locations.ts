import { atom } from 'nanostores';
import type { Location } from '../types';

export const $locations = atom<Location[]>([]);
