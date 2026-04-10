import { describe, it, expect, beforeEach } from 'vitest';
import { $checklistItems, $checklistCountsByTaskId } from './checklist';

/**
 * Unit tests for the $checklistCountsByTaskId computed atom that
 * powers the inline checklist count badge in TaskMeta (T3 design
 * spec parity).
 */
describe('$checklistCountsByTaskId', () => {
  beforeEach(() => {
    $checklistItems.set([]);
  });

  it('returns an empty map when there are no checklist items', () => {
    expect($checklistCountsByTaskId.get().size).toBe(0);
  });

  it('counts done and total per task', () => {
    $checklistItems.set([
      { id: 'a', taskId: 't1', title: 'one', status: 0, index: 0, createdAt: '', updatedAt: '' },
      { id: 'b', taskId: 't1', title: 'two', status: 1, index: 1, createdAt: '', updatedAt: '' },
      { id: 'c', taskId: 't1', title: 'three', status: 1, index: 2, createdAt: '', updatedAt: '' },
    ]);
    const map = $checklistCountsByTaskId.get();
    expect(map.get('t1')).toEqual({ done: 2, total: 3 });
  });

  it('keeps separate counts per task', () => {
    $checklistItems.set([
      { id: 'a', taskId: 't1', title: 'a1', status: 0, index: 0, createdAt: '', updatedAt: '' },
      { id: 'b', taskId: 't2', title: 'b1', status: 1, index: 0, createdAt: '', updatedAt: '' },
      { id: 'c', taskId: 't2', title: 'b2', status: 1, index: 1, createdAt: '', updatedAt: '' },
    ]);
    const map = $checklistCountsByTaskId.get();
    expect(map.get('t1')).toEqual({ done: 0, total: 1 });
    expect(map.get('t2')).toEqual({ done: 2, total: 2 });
  });

  it('returns undefined for tasks without any checklist items', () => {
    $checklistItems.set([
      { id: 'a', taskId: 't1', title: 'a1', status: 0, index: 0, createdAt: '', updatedAt: '' },
    ]);
    const map = $checklistCountsByTaskId.get();
    expect(map.get('untracked')).toBeUndefined();
  });

  it('treats every non-1 status as not-done (pending only counts complete)', () => {
    $checklistItems.set([
      { id: 'a', taskId: 't1', title: 'a', status: 0, index: 0, createdAt: '', updatedAt: '' },
      { id: 'b', taskId: 't1', title: 'b', status: 2, index: 1, createdAt: '', updatedAt: '' },
    ]);
    expect($checklistCountsByTaskId.get().get('t1')).toEqual({ done: 0, total: 2 });
  });
});
