import { useState } from 'react';
import { todayLocal } from '../lib/dates';
import { useStore } from '@nanostores/react';
import {
  $projects,
  $tags,
  $tagsByTaskId,
  $selectedTaskIds,
  $checklistCountsByTaskId,
  $showQuickMove,
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
  const checklistCounts = useStore($checklistCountsByTaskId);

  const project = task.projectId ? (projects.find((p) => p.id === task.projectId) ?? null) : null;
  const taskTagIds = tagsByTaskId.get(task.id);
  const taskTags = taskTagIds && taskTagIds.size > 0
    ? tags.filter((t) => taskTagIds.has(t.id))
    : [];
  const checklistCount = checklistCounts.get(task.id);

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

  const contextMenuItems = buildTaskContextMenuItems(task, isCompleted, isCancelled);

  return (
    <>
      <div
        ref={reorderRef}
        className={`task-item${isSelected ? ' selected' : ''}${isMultiSelected ? ' selected' : ''}${isReordering ? ' task-item-dragging' : ''}`}
        style={{ position: 'relative' }}
        data-task-id={task.id}
        onClick={handleClick}
        onDoubleClick={onDoubleClick}
        onContextMenu={handleContextMenu}
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
          <span className={`task-title${isCompleted ? ' completed' : ''}`} title={task.title}>
            {task.title}
          </span>

          <TaskMeta
            task={task}
            project={project}
            taskTags={taskTags}
            checklistCount={checklistCount}
            hideProjectPill={hideProjectPill}
          />
        </div>

        {showTriageActions && (
          <div className="task-actions" data-reorder-ignore>
            <button
              className="task-action-btn today-btn"
              title="Schedule for Today (⌘T)"
              aria-label="Schedule for Today"
              onClick={(e) => {
                e.stopPropagation();
                const today = todayLocal();
                updateTask({ id: task.id, schedule: 1, startDate: today });
              }}
            >
              <svg viewBox="0 0 16 16" width="14" height="14" fill="currentColor" stroke="none" aria-hidden="true">
                <polygon points="8 2 9.8 5.6 14 6.2 11 9 11.8 13 8 11.2 4.2 13 5 9 2 6.2 6.2 5.6" />
              </svg>
            </button>
            <button
              className="task-action-btn someday-btn"
              title="Schedule for Someday (⌘S)"
              aria-label="Schedule for Someday"
              onClick={(e) => {
                e.stopPropagation();
                updateTask({ id: task.id, schedule: 2 });
              }}
            >
              <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                <circle cx="8" cy="8" r="5.5" />
                <line x1="8" y1="5" x2="8" y2="8" />
                <line x1="8" y1="8" x2="10.5" y2="10" />
              </svg>
            </button>
            <button
              className="task-action-btn"
              title="Move to Project (⇧⌘M)"
              aria-label="Move to project"
              onClick={(e) => {
                e.stopPropagation();
                selectTask(task.id);
                $showQuickMove.set(true);
              }}
            >
              <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                <path d="M2 4.5c0-.8.7-1.5 1.5-1.5h3l1.5 2h4.5c.8 0 1.5.7 1.5 1.5v5c0 .8-.7 1.5-1.5 1.5h-9C2.7 13 2 12.3 2 11.5z" />
              </svg>
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
