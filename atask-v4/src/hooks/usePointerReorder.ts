import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  setTaskPointerHoverTarget,
  startPointerDragCursor,
  updatePointerDragCursor,
  endPointerDragCursor,
} from '../store/ui';

type ReorderableItem = { id: string };

export interface PointerReorderState {
  activeId: string | null;
  dropIndex: number | null;
  isPointerDragging: boolean;
  cursorX: number | null;
  cursorY: number | null;
}

type StartEvent = React.PointerEvent<HTMLElement> | React.MouseEvent<HTMLElement>;

interface PointerReorderOptions<T extends ReorderableItem> {
  items: T[];
  onReorder: (moves: Array<{ id: string; index: number }>) => void | Promise<void>;
  shouldHandlePointerDown?: (event: StartEvent, id: string) => boolean;
  onDragStart?: (id: string) => void;
  onDragEnd?: (id: string) => void;
  onCrossListDrop?: (
    id: string,
    dropTarget: Element,
    cursor: { x: number; y: number },
  ) => boolean;
  /**
   * Optional callback returning the current multi-selection set. If the
   * dragged item is in the set, commitReorder moves the whole selection
   * as a block; otherwise it falls back to single-item reorder.
   * Callers typically pass `() => $selectedTaskIds.get()`.
   */
  getSelectedIds?: () => Set<string>;
  /**
   * Optional list identifier + kind used to publish cross-list drag
   * cursor state via $pointerDragCursor. Set both to make other list
   * instances render an insertion indicator when this list's drag
   * hovers over them (see useForeignDropIndex).
   */
  listId?: string;
  kind?: 'task' | 'project';
}

interface PointerStartArgs {
  id: string;
  clientX: number;
  clientY: number;
  pointerId: number | null;
  inputType: 'pointer' | 'mouse';
}

interface PointerDragSession {
  id: string;
  pointerId: number | null;
  startX: number;
  startY: number;
  inputType: 'pointer' | 'mouse';
}

const POINTER_DRAG_THRESHOLD = 8;
const POINTER_DRAG_HOLD_MS = 150;

export interface PointerReorderReturn {
  reorderState: PointerReorderState;
  registerItem: (id: string) => (node: HTMLElement | null) => void;
  getPointerHandlers: (id: string) => {
    onPointerDown: (event: React.PointerEvent<HTMLElement>) => void;
    onMouseDown: (event: React.MouseEvent<HTMLElement>) => void;
  };
  cancelReorder: () => void;
  getItemRect: (id: string) => DOMRect | null;
}

export default function usePointerReorder<T extends ReorderableItem>({
  items,
  onReorder,
  shouldHandlePointerDown,
  onDragStart,
  onDragEnd,
  onCrossListDrop,
  getSelectedIds,
  listId,
  kind,
}: PointerReorderOptions<T>): PointerReorderReturn {
  const [reorderState, setReorderState] = useState<PointerReorderState>({
    activeId: null,
    dropIndex: null,
    isPointerDragging: false,
    cursorX: null,
    cursorY: null,
  });
  const reorderStateRef = useRef(reorderState);
  const sessionRef = useRef<PointerDragSession | null>(null);
  const holdTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const itemElementsRef = useRef(new Map<string, HTMLElement>());
  const onDragStartRef = useRef(onDragStart);
  const onDragEndRef = useRef(onDragEnd);
  const onCrossListDropRef = useRef(onCrossListDrop);

  const getSelectedIdsRef = useRef(getSelectedIds);
  onDragStartRef.current = onDragStart;
  onDragEndRef.current = onDragEnd;
  onCrossListDropRef.current = onCrossListDrop;
  getSelectedIdsRef.current = getSelectedIds;

  // Throttled cursor publish: pointermove fires at the display refresh rate
  // (typically 60-240 Hz). Publishing to $pointerDragCursor on every event
  // would re-render every subscribed consumer at the same rate, which then
  // churns ref callbacks and trashes pointer-event continuity. Coalesce to
  // one publish per animation frame.
  const cursorRafRef = useRef<number | null>(null);
  const pendingCursorRef = useRef<{ x: number; y: number } | null>(null);
  const flushCursor = () => {
    cursorRafRef.current = null;
    const pending = pendingCursorRef.current;
    pendingCursorRef.current = null;
    if (pending) updatePointerDragCursor(pending.x, pending.y);
  };
  const schedulePublishCursor = (x: number, y: number) => {
    pendingCursorRef.current = { x, y };
    if (cursorRafRef.current == null) {
      cursorRafRef.current = requestAnimationFrame(flushCursor);
    }
  };
  const cancelCursorPublish = () => {
    if (cursorRafRef.current != null) {
      cancelAnimationFrame(cursorRafRef.current);
      cursorRafRef.current = null;
    }
    pendingCursorRef.current = null;
  };

  const setReorderStateSync = useCallback((nextState: PointerReorderState) => {
    reorderStateRef.current = nextState;
    setReorderState(nextState);
  }, []);

  const registerItem = useCallback((id: string) => (node: HTMLElement | null) => {
    if (node) {
      itemElementsRef.current.set(id, node);
      return;
    }

    itemElementsRef.current.delete(id);
  }, []);

  const cancelReorder = useCallback(() => {
    if (holdTimerRef.current) { clearTimeout(holdTimerRef.current); holdTimerRef.current = null; }
    const prevId = sessionRef.current?.id ?? null;
    sessionRef.current = null;
    setReorderStateSync({
      activeId: null,
      dropIndex: null,
      isPointerDragging: false,
      cursorX: null,
      cursorY: null,
    });
    if (listId && kind) {
      cancelCursorPublish();
      endPointerDragCursor();
    }
    if (prevId) {
      onDragEndRef.current?.(prevId);
    }
  }, [setReorderStateSync, listId, kind]);

  const getOrderedItems = useCallback(() => {
    return items
      .map((item) => ({
        id: item.id,
        node: itemElementsRef.current.get(item.id),
      }))
      .filter((entry): entry is { id: string; node: HTMLElement } => Boolean(entry.node));
  }, [items]);

  const getDropIndex = useCallback((clientY: number) => {
    const orderedItems = getOrderedItems();

    for (let index = 0; index < orderedItems.length; index += 1) {
      const rect = orderedItems[index].node.getBoundingClientRect();
      const centerY = rect.top + rect.height / 2;
      if (clientY < centerY) {
        return items.findIndex((item) => item.id === orderedItems[index].id);
      }
    }

    return items.length;
  }, [getOrderedItems, items]);

  const beginReorder = useCallback((args: PointerStartArgs) => {
    sessionRef.current = {
      id: args.id,
      pointerId: args.pointerId,
      startX: args.clientX,
      startY: args.clientY,
      inputType: args.inputType,
    };

    if (holdTimerRef.current) clearTimeout(holdTimerRef.current);
    holdTimerRef.current = setTimeout(() => {
      if (!sessionRef.current || sessionRef.current.id !== args.id) return;
      setReorderStateSync({
        activeId: args.id,
        dropIndex: null,
        isPointerDragging: false,
        cursorX: args.clientX,
        cursorY: args.clientY,
      });
      if (listId && kind) {
        startPointerDragCursor(args.id, kind, listId, args.clientX, args.clientY);
      }
      onDragStartRef.current?.(args.id);
    }, POINTER_DRAG_HOLD_MS);
  }, [setReorderStateSync, listId, kind]);

  const updateReorder = useCallback((event: MouseEvent | PointerEvent, inputType: 'pointer' | 'mouse') => {
    const session = sessionRef.current;
    if (!session || session.inputType !== inputType) return;
    if (inputType === 'pointer' && session.pointerId !== (event as PointerEvent).pointerId) return;

    const movedEnough = Math.abs(event.clientX - session.startX) >= POINTER_DRAG_THRESHOLD
      || Math.abs(event.clientY - session.startY) >= POINTER_DRAG_THRESHOLD;
    if (!movedEnough && !reorderStateRef.current.isPointerDragging) return;

    // If movement threshold reached before hold timer, activate immediately
    const wasInactive = !reorderStateRef.current.activeId;
    if (wasInactive) {
      if (holdTimerRef.current) { clearTimeout(holdTimerRef.current); holdTimerRef.current = null; }
      onDragStartRef.current?.(session.id);
    }

    const dropIndex = getDropIndex(event.clientY);
    setReorderStateSync({
      activeId: session.id,
      dropIndex,
      isPointerDragging: true,
      cursorX: event.clientX,
      cursorY: event.clientY,
    });

    // Cross-list cursor publish: notify other list instances of the live
    // cursor position so they can compute their own foreign drop index.
    // Subsequent updates are rAF-batched so consumers don't re-render at
    // pointermove rate.
    if (listId && kind) {
      if (wasInactive) {
        startPointerDragCursor(session.id, kind, listId, event.clientX, event.clientY);
      } else {
        schedulePublishCursor(event.clientX, event.clientY);
      }
    }

    // Update sidebar hover target using elementFromPoint (pointer capture blocks pointerenter)
    if (onDragStartRef.current) {
      const el = document.elementFromPoint(event.clientX, event.clientY);
      const sidebarItem = el?.closest('[data-sidebar-item-id]');
      setTaskPointerHoverTarget(sidebarItem?.getAttribute('data-sidebar-item-id') ?? null);
    }
  }, [getDropIndex, setReorderStateSync]);

  const commitReorder = useCallback(async (event: MouseEvent | PointerEvent, inputType: 'pointer' | 'mouse') => {
    const session = sessionRef.current;
    if (!session || session.inputType !== inputType) return;
    if (inputType === 'pointer' && session.pointerId !== (event as PointerEvent).pointerId) return;

    const currentState = reorderStateRef.current;
    if (!currentState.activeId) {
      cancelReorder();
      return;
    }

    if (!currentState.isPointerDragging) {
      cancelReorder();
      return;
    }

    const cursorX = event.clientX;
    const cursorY = event.clientY;
    const dropTarget = document.elementFromPoint(cursorX, cursorY);
    const sidebarItem = dropTarget?.closest('[data-sidebar-item-id]');

    if (sidebarItem && onCrossListDropRef.current) {
      const handled = onCrossListDropRef.current(currentState.activeId, sidebarItem, {
        x: cursorX,
        y: cursorY,
      });
      if (handled) {
        cancelReorder();
        return;
      }
    }

    if (currentState.dropIndex == null) {
      cancelReorder();
      return;
    }

    // Multi-select group move: if the dragged item is inside the current
    // selection set, move the entire selection as a contiguous block rather
    // than only the dragged row.
    const selection = getSelectedIdsRef.current?.();
    const isGroupMove =
      selection != null && selection.size > 1 && selection.has(currentState.activeId);

    // The set of ids to extract (in list order) for the move.
    const movingIds: string[] = isGroupMove
      ? items.filter((item) => selection!.has(item.id)).map((item) => item.id)
      : [currentState.activeId];
    const movingSet = new Set(movingIds);

    // Remaining items after removing the moving set, preserving order.
    const remaining = items.filter((item) => !movingSet.has(item.id));

    // Translate dropIndex (an index in the ORIGINAL items list) to the
    // equivalent insertion point in the `remaining` list by subtracting
    // moving items that preceded the drop point.
    let targetIndex = currentState.dropIndex;
    for (const item of items.slice(0, currentState.dropIndex)) {
      if (movingSet.has(item.id)) targetIndex -= 1;
    }
    targetIndex = Math.max(0, Math.min(targetIndex, remaining.length));

    // Extract moved items in original order.
    const movedItems: T[] = [];
    for (const id of movingIds) {
      const item = items.find((it) => it.id === id);
      if (item) movedItems.push(item);
    }

    const reordered = [...remaining.slice(0, targetIndex), ...movedItems, ...remaining.slice(targetIndex)];

    // Bail if the order is unchanged (drop point lands on same spot).
    const unchanged =
      reordered.length === items.length &&
      reordered.every((item, i) => item.id === items[i].id);
    if (unchanged) {
      cancelReorder();
      return;
    }

    try {
      await onReorder(reordered.map((item, index) => ({ id: item.id, index })));
    } finally {
      cancelReorder();
    }
  }, [cancelReorder, items, onReorder]);

  useEffect(() => {
    const cancelMouseSession = () => {
      const session = sessionRef.current;
      if (!session || session.inputType !== 'mouse') return;
      cancelReorder();
    };

    const handlePointerMove = (event: PointerEvent) => {
      updateReorder(event, 'pointer');
    };
    const handlePointerUp = (event: PointerEvent) => {
      void commitReorder(event, 'pointer');
    };
    const handlePointerCancel = (event: PointerEvent) => {
      const session = sessionRef.current;
      if (!session || session.inputType !== 'pointer' || session.pointerId !== event.pointerId) return;
      cancelReorder();
    };
    const handleMouseMove = (event: MouseEvent) => {
      updateReorder(event, 'mouse');
    };
    const handleMouseUp = (event: MouseEvent) => {
      void commitReorder(event, 'mouse');
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        cancelReorder();
      }
    };
    const handleWindowBlur = () => {
      cancelReorder();
    };
    const handleVisibilityChange = () => {
      if (document.visibilityState !== 'visible') {
        cancelMouseSession();
      }
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', handlePointerUp);
    window.addEventListener('pointercancel', handlePointerCancel);
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('blur', handleWindowBlur);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      window.removeEventListener('pointercancel', handlePointerCancel);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('blur', handleWindowBlur);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [cancelReorder, commitReorder, updateReorder]);

  const getPointerHandlers = useCallback((id: string) => {
    const startReorder = (event: StartEvent, inputType: 'pointer' | 'mouse') => {
      if (shouldHandlePointerDown && !shouldHandlePointerDown(event, id)) return;
      if (event.button !== 0) return;
      if (inputType === 'pointer' && 'isPrimary' in event && !event.isPrimary) return;
      if (sessionRef.current) return;

      event.preventDefault();
      if (inputType === 'pointer' && 'pointerId' in event) {
        try {
          event.currentTarget.setPointerCapture(event.pointerId);
        } catch {
          // Pointer capture is a best-effort optimization.
        }
      }

      beginReorder({
        id,
        clientX: event.clientX,
        clientY: event.clientY,
        pointerId: 'pointerId' in event ? event.pointerId : null,
        inputType,
      });
    };

    return {
      onPointerDown: (event: React.PointerEvent<HTMLElement>) => {
        startReorder(event, 'pointer');
      },
      onMouseDown: (event: React.MouseEvent<HTMLElement>) => {
        startReorder(event, 'mouse');
      },
    };
  }, [beginReorder, shouldHandlePointerDown]);

  const getItemRect = useCallback((id: string) => {
    const node = itemElementsRef.current.get(id);
    if (!node) return null;
    return node.getBoundingClientRect();
  }, []);

  return useMemo(() => ({
    reorderState,
    registerItem,
    getPointerHandlers,
    cancelReorder,
    getItemRect,
  }), [cancelReorder, getItemRect, getPointerHandlers, registerItem, reorderState]);
}
