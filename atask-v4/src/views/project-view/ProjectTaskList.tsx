import { Fragment, useCallback, useRef, useState } from 'react';
import useForeignDropIndex from '../../hooks/useForeignDropIndex';
import TaskRow, { shouldHandleTaskRowPointerDown } from '../../components/TaskRow';
import TaskInlineEditor from '../../components/TaskInlineEditor';
import NewTaskRow from '../../components/NewTaskRow';
import DropSlot from '../../components/task-row/DropSlot';
import DragOverlay from '../../components/DragOverlay';
import usePointerReorder from '../../hooks/usePointerReorder';
import type { ReorderMove, Task } from '../../types';
import { startTaskPointerDrag, endTaskPointerDrag, updateTask, $tasks, $projects, $selectedTaskIds } from '../../store/index';
import { useStore } from '@nanostores/react';
import { $taskPointerDrag } from '../../store/ui';
import { todayLocal, tomorrowLocal } from '../../lib/dates';

interface ProjectTaskListProps {
  tasks: Task[];
  projectId: string;
  /**
   * Stable identifier for this task list instance. Each section's
   * ProjectTaskList must pass a different listId so cross-section drag
   * detection can tell them apart. Top-level (sectionless) lists pass
   * `task-sectionless:${projectId}`; per-section lists pass
   * `task-section:${section.id}`.
   */
  listId: string;
  expandedTaskId: string | null;
  selectedTaskId: string | null;
  selectedTaskIds: Set<string>;
  onSelectTask: (id: string) => void;
  onExpandTask: (id: string) => void;
  onCloseExpandedTask: () => void;
  onCreateTask: (title: string) => void;
  onReorderTasks: (moves: ReorderMove[]) => Promise<void>;
  onTaskDrop?: (taskId: string) => void;
}

export default function ProjectTaskList({
  tasks,
  projectId,
  listId,
  expandedTaskId,
  selectedTaskId,
  selectedTaskIds,
  onSelectTask,
  onExpandTask,
  onCloseExpandedTask,
  onCreateTask,
  onReorderTasks,
  onTaskDrop,
}: ProjectTaskListProps) {
  const [isDropTarget, setIsDropTarget] = useState(false);
  const taskDrag = useStore($taskPointerDrag);
  const handleCrossListDrop = (taskId: string, target: Element) => {
    const sidebarItemId = target.getAttribute('data-sidebar-item-id');
    const sidebarItemKind = target.getAttribute('data-sidebar-item-kind');

    if (sidebarItemKind === 'project' && sidebarItemId && sidebarItemId !== projectId) {
      const allProjects = $projects.get();
      const project = allProjects.find((p) => p.id === sidebarItemId);
      updateTask({ id: taskId, projectId: sidebarItemId, areaId: project?.areaId ?? null, schedule: 0, startDate: null, timeSlot: null });
      return true;
    }

    if (sidebarItemKind === 'area' && sidebarItemId) {
      updateTask({ id: taskId, areaId: sidebarItemId, projectId: null, schedule: 0, startDate: null, timeSlot: null });
      return true;
    }

    if (sidebarItemKind === 'section' && sidebarItemId) {
      // Same-section drops fall through to within-list reorder (return
      // false) so the normal drop-index splice runs. Only cross-section
      // moves actually update the sectionId.
      const draggedTask = $tasks.get().find((t) => t.id === taskId);
      if (draggedTask && draggedTask.sectionId === sidebarItemId) {
        return false;
      }
      updateTask({ id: taskId, sectionId: sidebarItemId });
      return true;
    }

    if (sidebarItemKind === 'sectionless') {
      updateTask({ id: taskId, sectionId: null });
      return true;
    }

    const closestNavItem = target.closest('[data-sidebar-item-kind="nav"]');
    if (closestNavItem) {
      const view = closestNavItem.getAttribute('data-sidebar-item-id');
      if (view === 'inbox') {
        updateTask({ id: taskId, schedule: 0, startDate: null, timeSlot: null, projectId: null, areaId: null, sectionId: null });
        return true;
      }
      if (view === 'today') {
        const today = todayLocal();
        updateTask({ id: taskId, schedule: 1, startDate: today, projectId: null, areaId: null, sectionId: null });
        return true;
      }
      if (view === 'upcoming') {
        updateTask({ id: taskId, schedule: 3, startDate: tomorrowLocal(), projectId: null, areaId: null, sectionId: null });
        return true;
      }
      if (view === 'someday') {
        updateTask({ id: taskId, schedule: 2, projectId: null, areaId: null, sectionId: null });
        return true;
      }
    }

    return false;
  };

  const containerRef = useRef<HTMLDivElement | null>(null);
  const { reorderState, getPointerHandlers, registerItem, getItemRect } = usePointerReorder({
    getSelectedIds: () => $selectedTaskIds.get(),
    items: tasks,
    listId,
    kind: 'task',
    onReorder: onReorderTasks,
    shouldHandlePointerDown: (event) => shouldHandleTaskRowPointerDown(event.target),
    onDragStart: startTaskPointerDrag,
    onDragEnd: endTaskPointerDrag,
    onCrossListDrop: handleCrossListDrop,
  });

  // Foreign drop indicator: show an insertion line when a task being
  // dragged from another section/list is hovering this list.
  const taskItemRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const registerTaskItem = useCallback((id: string) => {
    const hookRegister = registerItem(id);
    return (node: HTMLDivElement | null) => {
      if (node) taskItemRefs.current.set(id, node);
      else taskItemRefs.current.delete(id);
      hookRegister(node);
    };
  }, [registerItem]);

  const foreignDrop = useForeignDropIndex({
    listId,
    kind: 'task',
    containerRef,
    getItemElements: () => {
      const elements: HTMLDivElement[] = [];
      for (const task of tasks) {
        const el = taskItemRefs.current.get(task.id);
        if (el) elements.push(el);
      }
      return elements;
    },
  });
  const draggedTaskIndex = reorderState.activeId
    ? tasks.findIndex((task) => task.id === reorderState.activeId)
    : -1;
  const isDragging = reorderState.isPointerDragging;
  const itemWidth = reorderState.activeId ? getItemRect(reorderState.activeId)?.width ?? null : null;

  const handlePointerEnter = () => {
    if (taskDrag.activeTaskId && onTaskDrop) {
      setIsDropTarget(true);
    }
  };

  const handlePointerLeave = () => {
    setIsDropTarget(false);
  };

  const handlePointerUp = () => {
    if (taskDrag.activeTaskId && onTaskDrop) {
      onTaskDrop(taskDrag.activeTaskId);
    }
    setIsDropTarget(false);
  };

  const renderDropZone = (index: number) => {
    const localVisible =
      isDragging &&
      reorderState.dropIndex === index &&
      index !== draggedTaskIndex &&
      index !== draggedTaskIndex + 1;
    // Foreign-list drag landing in this list -> mirror its dropIndex.
    const foreignVisible =
      !isDragging &&
      foreignDrop.isForeignHovering &&
      foreignDrop.dropIndex === index;

    if (!localVisible && !foreignVisible) return null;

    const edgeClass = index === 0
      ? ' task-drop-zone-edge-top'
      : index === tasks.length
        ? ' task-drop-zone-edge-bottom'
        : '';

    return (
      <div
        key={`drop-zone-${index}`}
        className={`task-drop-zone${edgeClass}`}
      >
        <DropSlot />
      </div>
    );
  };

  const renderDragClone = (id: string) => {
    const task = tasks.find((t) => t.id === id);
    if (!task) return null;
    return (
      <div
        style={{
          background: 'var(--sidebar-hover)',
          borderRadius: 'var(--radius-md)',
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
          padding: '8px 12px',
        }}
      >
        <span style={{ fontSize: 'var(--text-base)', color: 'var(--ink-primary)' }}>
          {task.title}
        </span>
      </div>
    );
  };

  return (
    <div
      ref={containerRef}
      onPointerEnter={handlePointerEnter}
      onPointerLeave={handlePointerLeave}
      onPointerUp={handlePointerUp}
      data-sidebar-item-kind="sectionless"
      style={isDropTarget ? { background: 'var(--accent-subtle)', borderRadius: 'var(--radius-md)' } : undefined}
    >
      {tasks.map((task, index) => (
        <Fragment key={task.id}>
          {renderDropZone(index)}
          {expandedTaskId === task.id ? (
            <TaskInlineEditor
              task={task}
              onClose={onCloseExpandedTask}
            />
          ) : (
            <TaskRow
              task={task}
              isSelected={selectedTaskId === task.id}
              isMultiSelected={selectedTaskIds.has(task.id)}
              taskList={tasks}
              hideProjectPill
              onClick={() => onSelectTask(task.id)}
              onDoubleClick={() => onExpandTask(task.id)}
              reorderRef={registerTaskItem(task.id)}
              reorderHandlers={getPointerHandlers(task.id)}
              isReordering={reorderState.activeId === task.id}
            />
          )}
        </Fragment>
      ))}
      {renderDropZone(tasks.length)}
      <NewTaskRow onCreate={onCreateTask} />
      <DragOverlay
        activeId={reorderState.activeId}
        cursorX={reorderState.cursorX}
        cursorY={reorderState.cursorY}
        itemWidth={itemWidth}
        renderClone={renderDragClone}
      />
    </div>
  );
}
