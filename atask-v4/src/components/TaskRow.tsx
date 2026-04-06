import { useState } from 'react';
import { todayLocal } from '../lib/dates';
import { useStore } from '@nanostores/react';
import {
  $projects,
  $tags,
  $tagsByTaskId,
  $selectedTaskIds,
  completeTask,
  reopenTask,
  selectTask,
  selectTaskRange,
  toggleTaskSelection,
  updateTask,
} from '../store/index';
import CheckboxCircle from './CheckboxCircle';
import ContextMenu from './ContextMenu';
import type { Task } from '../types';
import { buildTaskContextMenuItems, TaskMeta } from './task-row/taskRowHelpers';

export function shouldHandleTaskRowPointerDown(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) {
    return true;
  }

  return target.closest('[data-reorder-ignore]') === null;
}

interface TaskRowProps {
  task: Task;
  isSelected: boolean;
  isMultiSelected?: boolean;
  taskList?: Task[];
  isToday?: boolean;
  onClick: () => void;
  onDoubleClick: () => void;
  showTriageActions?: boolean;
  hideProjectPill?: boolean;
  reorderRef?: (node: HTMLDivElement | null) => void;
  reorderHandlers?: {
    onPointerDown: (e: React.PointerEvent<HTMLDivElement>) => void;
    onMouseDown: (e: React.MouseEvent<HTMLDivElement>) => void;
  };
  isReordering?: boolean;
}

export default function TaskRow({
  task,
  isSelected,
  isMultiSelected,
  taskList,
  isToday,
  onClick,
  onDoubleClick,
  showTriageActions,
  hideProjectPill,
  reorderRef,
  reorderHandlers,
  isReordering,
}: TaskRowProps) {
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);

  const projects = useStore($projects);
  const tags = useStore($tags);
  const tagsByTaskId = useStore($tagsByTaskId);

  const project = task.projectId ? (projects.find((p) => p.id === task.projectId) ?? null) : null;
  const taskTagIds = tagsByTaskId.get(task.id);
  const taskTags = taskTagIds && taskTagIds.size > 0
    ? tags.filter((t) => taskTagIds.has(t.id))
    : [];

  const isCompleted = task.status === 1;
  const isCancelled = task.status === 2;

  const handleCheckboxChange = () => {
    if (isCompleted || isCancelled) {
      reopenTask(task.id);
    } else {
      completeTask(task.id);
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    const currentSelectedIds = $selectedTaskIds.get();

    if (e.metaKey || e.ctrlKey) {
      e.preventDefault();
      toggleTaskSelection(task.id);
      return;
    }

    if (e.shiftKey && taskList) {
      e.preventDefault();
      selectTaskRange(task.id, taskList);
      return;
    }

    if (currentSelectedIds.size > 0) {
      selectTask(task.id);
      return;
    }

    onClick();
  };

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY });
  };

  const handleDragStart = (e: React.DragEvent<HTMLDivElement>) => {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', task.id);
  };

  const contextMenuItems = buildTaskContextMenuItems(task, isCompleted, isCancelled);

  return (
    <>
      <div
        ref={reorderRef}
        className={`task-item${isSelected ? ' selected' : ''}${isMultiSelected ? ' selected' : ''}${isReordering ? ' task-item-dragging' : ''}`}
        style={{ position: 'relative' }}
        draggable
        onClick={handleClick}
        onDoubleClick={onDoubleClick}
        onContextMenu={handleContextMenu}
        onDragStart={handleDragStart}
        {...reorderHandlers}
      >
        <div data-reorder-ignore>
          <CheckboxCircle
            checked={isCompleted}
            cancelled={isCancelled}
            today={isToday}
            onChange={handleCheckboxChange}
          />
        </div>

        <div className="task-content">
          <span className={`task-title${isCompleted ? ' completed' : ''}`}>
            {task.title}
          </span>

          <TaskMeta
            task={task}
            project={project}
            taskTags={taskTags}
            hideProjectPill={hideProjectPill}
          />
        </div>

        {showTriageActions && (
          <div className="task-actions" data-reorder-ignore>
            <button
              className="task-action-btn today-btn"
              title="Schedule for Today"
              onClick={(e) => {
                e.stopPropagation();
                const today = todayLocal();
                updateTask({ id: task.id, schedule: 1, startDate: today });
              }}
            >
              ★
            </button>
            <button
              className="task-action-btn"
              title="Schedule"
              onClick={(e) => {
                e.stopPropagation();
                const today = todayLocal();
                updateTask({ id: task.id, schedule: 1, startDate: today });
              }}
            >
              📅
            </button>
            <button
              className="task-action-btn someday-btn"
              title="Someday"
              onClick={(e) => {
                e.stopPropagation();
                updateTask({ id: task.id, schedule: 2 });
              }}
            >
              💤
            </button>
            <button
              className="task-action-btn"
              title="Move to Project"
              onClick={(e) => {
                e.stopPropagation();
                // noop for now
              }}
            >
              📁
            </button>
          </div>
        )}
      </div>

      {contextMenu && (
        <ContextMenu
          items={contextMenuItems}
          position={contextMenu}
          onClose={() => setContextMenu(null)}
        />
      )}
    </>
  );
}
