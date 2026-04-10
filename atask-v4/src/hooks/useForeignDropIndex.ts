import { useStore } from '@nanostores/react';
import { $pointerDragCursor } from '../store/ui';

interface UseForeignDropIndexOptions {
  /**
   * Stable identifier for THIS list. Must match the `listId` passed to
   * usePointerReorder for the same list. Used to detect "I am the source"
   * and skip the indicator (the source draws its own dropIndex internally).
   */
  listId: string;
  /**
   * What kind of items this list accepts. Filters the cross-list drag
   * state — a project drag won't light up a task list and vice-versa.
   */
  kind: 'task' | 'project';
  /**
   * Returns the current item DOM elements in render order. Used to
   * compute the local insertion index from the cursor's clientY. Must
   * be a stable callback that closes over the latest item refs (e.g.
   * read from a ref map populated by `registerItem`).
   */
  getItemElements: () => HTMLElement[];
  /**
   * Optional bounding container element. If provided, the foreign drop
   * indicator only fires when the cursor is within this container's
   * bounding rect — prevents distant lists from rendering a phantom
   * insertion line just because the cursor passed near their column.
   * If omitted, the hook falls back to a more permissive check based
   * on the items' own bounds.
   */
  containerRef?: { current: HTMLElement | null };
}

interface ForeignDropIndexResult {
  /** True when a foreign drag (different sourceListId, matching kind) is hovering THIS list. */
  isForeignHovering: boolean;
  /**
   * The position in this list's items where the foreign item would
   * land if released now. `null` when no foreign drag is hovering.
   * Range: [0, items.length] inclusive (0 = before first, items.length = after last).
   */
  dropIndex: number | null;
}

/**
 * Computes a "foreign drop index" — the visual insertion position for a
 * cross-list pointer drag that is currently hovering over this list.
 *
 * Reads `$pointerDragCursor` (published by the source usePointerReorder
 * instance) and walks the local item element rects to find where the
 * cursor would land. Returns `null` for both fields when no foreign
 * drag is active or the cursor is not within this list.
 *
 * Pair with `usePointerReorder({ listId, kind, ... })` on the source
 * side; the source publishes the cursor, every other list reading this
 * hook can render its own insertion line.
 */
export default function useForeignDropIndex({
  listId,
  kind,
  getItemElements,
  containerRef,
}: UseForeignDropIndexOptions): ForeignDropIndexResult {
  const drag = useStore($pointerDragCursor);

  // No drag, wrong kind, or this is the source list -> nothing to render.
  if (!drag.activeId || drag.kind !== kind || drag.sourceListId === listId) {
    return { isForeignHovering: false, dropIndex: null };
  }
  if (drag.cursorX == null || drag.cursorY == null) {
    return { isForeignHovering: false, dropIndex: null };
  }

  // Bounding-rect gate: only react when the cursor is actually within
  // this list's column. Prefer the explicit container if given,
  // otherwise derive from the items' own bounds with a small slop so
  // dropping just above/below the first/last item still works.
  const elements = getItemElements();
  if (containerRef?.current) {
    const rect = containerRef.current.getBoundingClientRect();
    if (
      drag.cursorX < rect.left ||
      drag.cursorX > rect.right ||
      drag.cursorY < rect.top ||
      drag.cursorY > rect.bottom
    ) {
      return { isForeignHovering: false, dropIndex: null };
    }
    // Empty list inside the container is still a valid drop target —
    // index 0.
    if (elements.length === 0) {
      return { isForeignHovering: true, dropIndex: 0 };
    }
  } else {
    if (elements.length === 0) {
      return { isForeignHovering: false, dropIndex: null };
    }
    const first = elements[0].getBoundingClientRect();
    const last = elements[elements.length - 1].getBoundingClientRect();
    const minY = first.top - 8;
    const maxY = last.bottom + 8;
    const minX = Math.min(first.left, last.left);
    const maxX = Math.max(first.right, last.right);
    if (
      drag.cursorX < minX ||
      drag.cursorX > maxX ||
      drag.cursorY < minY ||
      drag.cursorY > maxY
    ) {
      return { isForeignHovering: false, dropIndex: null };
    }
  }

  // Walk items top-to-bottom, picking the first whose center is below
  // the cursor. Mirrors usePointerReorder's getDropIndex.
  for (let i = 0; i < elements.length; i += 1) {
    const rect = elements[i].getBoundingClientRect();
    const centerY = rect.top + rect.height / 2;
    if (drag.cursorY < centerY) {
      return { isForeignHovering: true, dropIndex: i };
    }
  }
  return { isForeignHovering: true, dropIndex: elements.length };
}
