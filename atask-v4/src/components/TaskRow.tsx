import { useState } from 'react';
import { todayLocal } from '../lib/dates';
import { useStore } from '@nanostores/react';
import {
  $projects,
  $tags,
  $tagsByTaskId,
  $selectedTaskIds,
  $selectedTaskId,
  completeTask,
  reopenTask,
  cancelTask,
  deleteTask,
  duplicateTask,
  updateTask,
} from '../store/index';
import CheckboxCircle from './CheckboxCircle';
import TagPill from './TagPill';
import ContextMenu, { type MenuItem } from './ContextMenu';
import type { Task } from '../types';

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
  dragHandlers?: {
    draggable: true;
    onDragStart: (e: React.DragEvent) => void;
    onDragEnd: () => void;
  };
  dropHandlers?: {
    onDragOver: (e: React.DragEvent) => void;
    onDragLeave: () => void;
    onDrop: (e: React.DragEvent) => void;
  };
  isDragOver?: boolean;
}

function formatDeadline(deadline: string): string {
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const d = new Date(deadline);
  d.setHours(0, 0, 0, 0);

  const diffDays = Math.round((d.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

  if (diffDays < 0) return 'Overdue';
  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Tomorrow';

  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
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
  dragHandlers,
  dropHandlers,
  isDragOver,
}: TaskRowProps) {
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);

  const projects = useStore($projects);
  const tags = useStore($tags);
  const tagsByTaskId = useStore($tagsByTaskId);

  const project = task.projectId ? projects.find((p) => p.id === task.projectId) : null;
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
    const currentSelectedId = $selectedTaskId.get();

    if (e.metaKey || e.ctrlKey) {
      e.preventDefault();
      const next = new Set(currentSelectedIds);
      if (next.has(task.id)) {
        next.delete(task.id);
      } else {
        next.add(task.id);
      }
      $selectedTaskIds.set(next);
      return;
    }

    if (e.shiftKey && taskList) {
      e.preventDefault();
      const lastId = currentSelectedId || (currentSelectedIds.size > 0 ? [...currentSelectedIds].pop() : null);
      if (lastId) {
        const lastIdx = taskList.findIndex(t => t.id === lastId);
        const currentIdx = taskList.findIndex(t => t.id === task.id);
        if (lastIdx >= 0 && currentIdx >= 0) {
          const start = Math.min(lastIdx, currentIdx);
          const end = Math.max(lastIdx, currentIdx);
          const range = taskList.slice(start, end + 1).map(t => t.id);
          $selectedTaskIds.set(new Set([...currentSelectedIds, ...range]));
          return;
        }
      }
    }

    // Normal click: clear multi-select, single select
    if (currentSelectedIds.size > 0) {
      $selectedTaskIds.set(new Set());
    }
    onClick();
  };

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY });
  };

  const metaItems: React.ReactNode[] = [];

  if (project && !hideProjectPill) {
    metaItems.push(
      <span key="project" className="task-project-pill">
        <span
          className="dot"
          style={{
            background: project.color || 'var(--accent)',
            width: 6,
            height: 6,
            borderRadius: '50%',
            flexShrink: 0,
          }}
        />
        {project.title}
      </span>,
    );
  }

  if (task.deadline) {
    if (metaItems.length > 0) {
      metaItems.push(<span key="sep-deadline" className="task-meta-sep">·</span>);
    }
    metaItems.push(
      <span key="deadline" className="task-deadline">
        {formatDeadline(task.deadline)}
      </span>,
    );
  }

  if (taskTags.length > 0) {
    for (const tag of taskTags) {
      if (metaItems.length > 0) {
        metaItems.push(<span key={`sep-${tag.id}`} className="task-meta-sep">·</span>);
      }
      metaItems.push(<TagPill key={tag.id} label={tag.title} variant="default" />);
    }
  }

  const contextMenuItems: MenuItem[] = [
    {
      label: isCompleted || isCancelled ? 'Reopen' : 'Complete',
      onClick: () => {
        if (isCompleted || isCancelled) {
          reopenTask(task.id);
        } else {
          completeTask(task.id);
        }
      },
    },
    {
      label: 'Cancel',
      onClick: () => cancelTask(task.id),
    },
    { separator: true },
    {
      label: 'Today',
      shortcut: '⌘T',
      onClick: () => {
        const today = todayLocal();
        updateTask({ id: task.id, schedule: 1, startDate: today });
      },
    },
    {
      label: 'Evening',
      shortcut: '⌘E',
      onClick: () => {
        const today = todayLocal();
        updateTask({ id: task.id, schedule: 1, timeSlot: 'evening', startDate: today });
      },
    },
    {
      label: 'Someday',
      shortcut: '⌘O',
      onClick: () => updateTask({ id: task.id, schedule: 2 }),
    },
    {
      label: 'Inbox',
      onClick: () => updateTask({ id: task.id, schedule: 0 }),
    },
    { separator: true },
    {
      label: 'Duplicate',
      shortcut: '⌘D',
      onClick: () => duplicateTask(task.id),
    },
    {
      label: 'Delete',
      shortcut: '⌫',
      danger: true,
      onClick: () => deleteTask(task.id),
    },
  ];

  return (
    <>
      <div
        className={`task-item${isSelected ? ' selected' : ''}${isMultiSelected ? ' selected' : ''}`}
        style={{
          position: 'relative',
          background: isDragOver ? 'var(--accent-subtle)' : undefined,
        }}
        onClick={handleClick}
        onDoubleClick={onDoubleClick}
        onContextMenu={handleContextMenu}
        {...dragHandlers}
        {...dropHandlers}
      >
        {isDragOver && (
          <div style={{
            position: 'absolute',
            top: -1,
            left: 0,
            right: 0,
            height: 3,
            background: 'var(--accent)',
            borderRadius: 2,
            zIndex: 10,
          }}>
            <div style={{
              position: 'absolute',
              left: -3,
              top: -3,
              width: 9,
              height: 9,
              borderRadius: '50%',
              background: 'var(--accent)',
            }} />
          </div>
        )}
        <CheckboxCircle
          checked={isCompleted}
          cancelled={isCancelled}
          today={isToday}
          onChange={handleCheckboxChange}
        />

        <div className="task-content">
          <span className={`task-title${isCompleted ? ' completed' : ''}`}>
            {task.title}
          </span>

          {metaItems.length > 0 && (
            <div className="task-meta">
              {metaItems}
            </div>
          )}
        </div>

        {showTriageActions && (
          <div className="task-actions">
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
