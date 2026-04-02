import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

type ReorderableItem = { id: string };

export interface PointerReorderState {
  activeId: string | null;
  dropIndex: number | null;
  isPointerDragging: boolean;
}

interface PointerReorderOptions<T extends ReorderableItem> {
  items: T[];
  onReorder: (moves: Array<{ id: string; index: number }>) => void | Promise<void>;
}

interface PointerStartArgs {
  id: string;
  clientX: number;
  clientY: number;
  pointerId: number;
}

interface PointerDragSession {
  id: string;
  pointerId: number;
  startX: number;
  startY: number;
}

const POINTER_DRAG_THRESHOLD = 4;

export interface PointerReorderReturn {
  reorderState: PointerReorderState;
  registerItem: (id: string) => (node: HTMLElement | null) => void;
  getPointerHandlers: (id: string) => {
    onPointerDown: (event: React.PointerEvent<HTMLElement>) => void;
  };
  cancelReorder: () => void;
}

export default function usePointerReorder<T extends ReorderableItem>({
  items,
  onReorder,
}: PointerReorderOptions<T>): PointerReorderReturn {
  const [reorderState, setReorderState] = useState<PointerReorderState>({
    activeId: null,
    dropIndex: null,
    isPointerDragging: false,
  });
  const reorderStateRef = useRef(reorderState);
  const sessionRef = useRef<PointerDragSession | null>(null);
  const itemElementsRef = useRef(new Map<string, HTMLElement>());

  useEffect(() => {
    reorderStateRef.current = reorderState;
  }, [reorderState]);

  const registerItem = useCallback((id: string) => (node: HTMLElement | null) => {
    if (node) {
      itemElementsRef.current.set(id, node);
      return;
    }

    itemElementsRef.current.delete(id);
  }, []);

  const cancelReorder = useCallback(() => {
    sessionRef.current = null;
    setReorderState({
      activeId: null,
      dropIndex: null,
      isPointerDragging: false,
    });
  }, []);

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
        return index;
      }
    }

    return orderedItems.length;
  }, [getOrderedItems]);

  const beginReorder = useCallback((args: PointerStartArgs) => {
    sessionRef.current = {
      id: args.id,
      pointerId: args.pointerId,
      startX: args.clientX,
      startY: args.clientY,
    };

    setReorderState({
      activeId: args.id,
      dropIndex: null,
      isPointerDragging: false,
    });
  }, []);

  const updateReorder = useCallback((event: PointerEvent) => {
    const session = sessionRef.current;
    if (!session || session.pointerId !== event.pointerId) return;

    const movedEnough = Math.abs(event.clientX - session.startX) >= POINTER_DRAG_THRESHOLD
      || Math.abs(event.clientY - session.startY) >= POINTER_DRAG_THRESHOLD;
    if (!movedEnough && !reorderStateRef.current.isPointerDragging) return;

    const dropIndex = getDropIndex(event.clientY);
    setReorderState({
      activeId: session.id,
      dropIndex,
      isPointerDragging: true,
    });
  }, [getDropIndex]);

  const commitReorder = useCallback(async (event: PointerEvent) => {
    const session = sessionRef.current;
    if (!session || session.pointerId !== event.pointerId) return;

    const currentState = reorderStateRef.current;
    if (!currentState.activeId || currentState.dropIndex == null || !currentState.isPointerDragging) {
      cancelReorder();
      return;
    }

    const sourceIndex = items.findIndex((item) => item.id === currentState.activeId);
    if (sourceIndex === -1) {
      cancelReorder();
      return;
    }

    const reordered = [...items];
    const [moved] = reordered.splice(sourceIndex, 1);
    if (!moved) {
      cancelReorder();
      return;
    }

    const targetIndex = currentState.dropIndex > sourceIndex
      ? currentState.dropIndex - 1
      : currentState.dropIndex;

    if (targetIndex === sourceIndex) {
      cancelReorder();
      return;
    }

    reordered.splice(targetIndex, 0, moved);

    try {
      await onReorder(reordered.map((item, index) => ({ id: item.id, index })));
    } finally {
      cancelReorder();
    }
  }, [cancelReorder, items, onReorder]);

  useEffect(() => {
    const handlePointerMove = (event: PointerEvent) => {
      updateReorder(event);
    };
    const handlePointerUp = (event: PointerEvent) => {
      void commitReorder(event);
    };
    const handlePointerCancel = (event: PointerEvent) => {
      const session = sessionRef.current;
      if (!session || session.pointerId !== event.pointerId) return;
      cancelReorder();
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        cancelReorder();
      }
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', handlePointerUp);
    window.addEventListener('pointercancel', handlePointerCancel);
    window.addEventListener('keydown', handleKeyDown);

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      window.removeEventListener('pointercancel', handlePointerCancel);
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [cancelReorder, commitReorder, updateReorder]);

  const getPointerHandlers = useCallback((id: string) => ({
    onPointerDown: (event: React.PointerEvent<HTMLElement>) => {
      if (event.button !== 0 || !event.isPrimary) return;

      event.preventDefault();
      try {
        event.currentTarget.setPointerCapture(event.pointerId);
      } catch {
        // Pointer capture is a best-effort optimization.
      }

      beginReorder({
        id,
        clientX: event.clientX,
        clientY: event.clientY,
        pointerId: event.pointerId,
      });
    },
  }), [beginReorder]);

  return useMemo(() => ({
    reorderState,
    registerItem,
    getPointerHandlers,
    cancelReorder,
  }), [cancelReorder, getPointerHandlers, registerItem, reorderState]);
}
