import { useState, useCallback, useRef } from 'react';
import type { Task } from '../types';
export { default as usePointerReorder } from './usePointerReorder';

interface DragState {
  dragId: string | null;
  dropIndex: number | null;
}

interface UseDragReorderReturn {
  dragState: DragState;
  getDragHandlers: (taskId: string) => {
    draggable: true;
    onDragStart: (e: React.DragEvent) => void;
    onDragEnd: () => void;
  };
  getDropHandlers: (index: number) => {
    onDragOver: (e: React.DragEvent) => void;
    onDragLeave: () => void;
    onDrop: (e: React.DragEvent) => void;
  };
}

export default function useDragReorder(
  tasks: Task[],
  onReorder: (moves: Array<{ id: string; index: number }>) => void | Promise<void>,
): UseDragReorderReturn {
  const [dragState, setDragState] = useState<DragState>({ dragId: null, dropIndex: null });
  const dragIdRef = useRef<string | null>(null);
  const dragElementRef = useRef<HTMLElement | null>(null);
  const dragElementRestoredRef = useRef(true);

  const restoreDraggedElement = useCallback(() => {
    const dragElement = dragElementRef.current;
    if (!dragElement || dragElementRestoredRef.current) return;

    dragElement.style.opacity = '';
    dragElementRestoredRef.current = true;
  }, []);

  const resetDragState = useCallback(() => {
    setDragState({ dragId: null, dropIndex: null });
    dragIdRef.current = null;
  }, []);

  const getDragHandlers = useCallback((taskId: string) => ({
    draggable: true as const,
    onDragStart: (e: React.DragEvent) => {
      dragIdRef.current = taskId;
      dragElementRef.current = e.currentTarget as HTMLElement;
      dragElementRestoredRef.current = false;
      setDragState({ dragId: taskId, dropIndex: null });
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', taskId);
      // Make the drag image semi-transparent
      dragElementRef.current.style.opacity = '0.5';
    },
    onDragEnd: () => {
      restoreDraggedElement();
      resetDragState();
      dragElementRef.current = null;
    },
  }), [resetDragState, restoreDraggedElement]);

  const getDropHandlers = useCallback((index: number) => ({
    onDragOver: (e: React.DragEvent) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      setDragState(prev => ({ ...prev, dropIndex: index }));
    },
    onDragLeave: () => {
      // Only clear if leaving to outside
    },
    onDrop: async (e: React.DragEvent) => {
      e.preventDefault();
      const sourceId = dragIdRef.current;
      try {
        if (!sourceId) return;

        const sourceIndex = tasks.findIndex(t => t.id === sourceId);
        if (sourceIndex === -1) return;

        // Build new order
        const reordered = [...tasks];
        const [moved] = reordered.splice(sourceIndex, 1);
        const targetIndex = index > sourceIndex ? index - 1 : index;
        if (targetIndex === sourceIndex) {
          return;
        }
        reordered.splice(targetIndex, 0, moved);

        // Generate moves array with new indices
        const moves = reordered.map((t, i) => ({ id: t.id, index: i }));
        await onReorder(moves);
      } finally {
        restoreDraggedElement();
        resetDragState();
      }
    },
  }), [onReorder, resetDragState, restoreDraggedElement, tasks]);

  return { dragState, getDragHandlers, getDropHandlers };
}
