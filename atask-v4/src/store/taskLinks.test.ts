import { describe, it, expect, beforeEach } from 'vitest';
import { $taskLinks, $linksByTaskId } from './taskLinks';

/**
 * Unit tests for the $linksByTaskId computed atom that powers the
 * "linked tasks" surface in DetailPanel and the inline editor.
 *
 * Even though the Go server now stores both directions of every link
 * (Fix #2), the local store still uses a Set-based reverse lookup so
 * that older client state with single-direction rows continues to
 * resolve correctly.
 */
describe('$linksByTaskId', () => {
  beforeEach(() => {
    $taskLinks.set([]);
  });

  it('returns an empty map for an empty link list', () => {
    const map = $linksByTaskId.get();
    expect(map.size).toBe(0);
  });

  it('makes a single-direction link visible from BOTH peers', () => {
    $taskLinks.set([{ taskId: 'A', linkedTaskId: 'B' }]);
    const map = $linksByTaskId.get();
    expect(map.get('A')?.has('B')).toBe(true);
    expect(map.get('B')?.has('A')).toBe(true);
  });

  it('dedupes when both directions are present in the underlying array', () => {
    // After Fix #2, the local writer inserts both rows. The reverse
    // inference must not create duplicates — the lookup uses a Set.
    $taskLinks.set([
      { taskId: 'A', linkedTaskId: 'B' },
      { taskId: 'B', linkedTaskId: 'A' },
    ]);
    const map = $linksByTaskId.get();
    expect(map.get('A')?.size).toBe(1);
    expect(map.get('B')?.size).toBe(1);
  });

  it('handles multi-link tasks (one task linked to several peers)', () => {
    $taskLinks.set([
      { taskId: 'A', linkedTaskId: 'B' },
      { taskId: 'A', linkedTaskId: 'C' },
      { taskId: 'A', linkedTaskId: 'D' },
    ]);
    const map = $linksByTaskId.get();
    expect(map.get('A')?.size).toBe(3);
    // Each peer sees the link back to A.
    expect(map.get('B')?.has('A')).toBe(true);
    expect(map.get('C')?.has('A')).toBe(true);
    expect(map.get('D')?.has('A')).toBe(true);
  });

  it('returns undefined for tasks with no links', () => {
    $taskLinks.set([{ taskId: 'A', linkedTaskId: 'B' }]);
    const map = $linksByTaskId.get();
    expect(map.get('UNRELATED')).toBeUndefined();
  });
});
