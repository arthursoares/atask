import { useStore } from '@nanostores/react';
import { $selectedTaskIds } from '../../store/index';
import type { Task } from '../../types';

interface TaskDragCloneProps {
  task: Task;
}

/**
 * Floating card rendered inside DragOverlay while a task drag is active.
 * Mirrors the row's identity (checkbox + title) instead of an anonymous
 * pill, and when the dragged task is part of a multi-selection it renders
 * as a small stack with a count badge — the whole selection moves.
 */
export default function TaskDragClone({ task }: TaskDragCloneProps) {
  const selectedIds = useStore($selectedTaskIds);
  const groupCount = selectedIds.size > 1 && selectedIds.has(task.id) ? selectedIds.size : 0;

  return (
    <div className={`drag-clone${groupCount ? ' drag-clone-stacked' : ''}`}>
      <span className="drag-clone-checkbox" aria-hidden="true" />
      <span className="drag-clone-title">{task.title || 'Untitled task'}</span>
      {groupCount > 0 && <span className="drag-clone-count">{groupCount}</span>}
    </div>
  );
}
