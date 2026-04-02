import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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
  onCrossListDrop?: (id: string, dropTarget: Element) => boolean;
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

const POINTER_DRAG_THRESHOLD = 4;

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
  const itemElementsRef = useRef(new Map<string, HTMLElement>());
  const onDragStartRef = useRef(onDragStart);
  const onDragEndRef = useRef(onDragEnd);
  const onCrossListDropRef = useRef(onCrossListDrop);

  onDragStartRef.current = onDragStart;
  onDragEndRef.current = onDragEnd;
  onCrossListDropRef.current = onCrossListDrop;

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
    const prevId = sessionRef.current?.id ?? null;
    sessionRef.current = null;
    setReorderStateSync({
      activeId: null,
      dropIndex: null,
      isPointerDragging: false,
      cursorX: null,
      cursorY: null,
    });
    if (prevId) {
      onDragEndRef.current?.(prevId);
    }
  }, [setReorderStateSync]);

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

    setReorderStateSync({
      activeId: args.id,
      dropIndex: null,
      isPointerDragging: false,
      cursorX: args.clientX,
      cursorY: args.clientY,
    });
    onDragStartRef.current?.(args.id);
  }, [setReorderStateSync]);

  const updateReorder = useCallback((event: MouseEvent | PointerEvent, inputType: 'pointer' | 'mouse') => {
    const session = sessionRef.current;
    if (!session || session.inputType !== inputType) return;
    if (inputType === 'pointer' && session.pointerId !== (event as PointerEvent).pointerId) return;

    const movedEnough = Math.abs(event.clientX - session.startX) >= POINTER_DRAG_THRESHOLD
      || Math.abs(event.clientY - session.startY) >= POINTER_DRAG_THRESHOLD;
    if (!movedEnough && !reorderStateRef.current.isPointerDragging) return;

    const dropIndex = getDropIndex(event.clientY);
    setReorderStateSync({
      activeId: session.id,
      dropIndex,
      isPointerDragging: true,
      cursorX: event.clientX,
      cursorY: event.clientY,
    });
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

    const cursorX = event.clientX;
    const cursorY = event.clientY;
    const dropTarget = document.elementFromPoint(cursorX, cursorY);
    const sidebarItem = dropTarget?.closest('[data-sidebar-item-id]');

    if (sidebarItem && onCrossListDropRef.current) {
      const handled = onCrossListDropRef.current(currentState.activeId, sidebarItem);
      if (handled) {
        cancelReorder();
        return;
      }
    }

    if (!currentState.isPointerDragging) {
      cancelReorder();
      return;
    }

    if (currentState.dropIndex == null) {
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
      cancelMouseSession();
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
