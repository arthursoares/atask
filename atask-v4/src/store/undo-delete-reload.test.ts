import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import type { Task } from '../types';

// Mock the Tauri IPC layer. vi.hoisted lets the vi.mock factory reference these.
const { loadAllMock, deleteTaskMock } = vi.hoisted(() => ({
  loadAllMock: vi.fn(),
  deleteTaskMock: vi.fn(),
}));

vi.mock('../hooks/useTauri', () => ({
  loadAll: loadAllMock,
  deleteTask: deleteTaskMock,
}));

import { loadAll, deleteTasksWithUndo } from './mutations';
import { $tasks } from './tasks';

function payload(tasks: Task[]) {
  return {
    tasks,
    projects: [],
    areas: [],
    sections: [],
    tags: [],
    taskTags: [],
    taskLinks: [],
    projectTags: [],
    checklistItems: [],
    activities: [],
    locations: [],
  };
}

const doomed = { id: 't1', title: 'Doomed' } as unknown as Task;

describe('undo-delete survives a reload mid-grace-window', () => {
  beforeEach(() => {
    // The unit project runs in Node (no DOM); mutations.ts uses window.setTimeout
    // for the undo grace timer. Alias window to globalThis so its timers (which
    // vi.useFakeTimers patches) back window.setTimeout — no DOM lib needed.
    vi.stubGlobal('window', globalThis);
    vi.useFakeTimers();
    loadAllMock.mockReset();
    deleteTaskMock.mockReset().mockResolvedValue(undefined);
    $tasks.set([doomed]);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('loadAll during the window does not resurrect a pending-delete task', async () => {
    await deleteTasksWithUndo(['t1']);
    expect($tasks.get()).toHaveLength(0); // hidden immediately

    // A sync `store-changed` / error reload returns the row that is still in
    // SQLite (the backend delete is deferred to the grace timer).
    loadAllMock.mockResolvedValue(payload([doomed]));
    await loadAll();
    expect($tasks.get()).toHaveLength(0); // THE FIX: still hidden, not resurrected

    // Once the grace timer fires, the backend delete actually runs...
    await vi.advanceTimersByTimeAsync(6000);
    expect(deleteTaskMock).toHaveBeenCalledWith('t1');

    // ...and a later reload (row now gone from SQLite) keeps it gone.
    loadAllMock.mockResolvedValue(payload([]));
    await loadAll();
    expect($tasks.get()).toHaveLength(0);
  });
});
