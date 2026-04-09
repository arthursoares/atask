import { Fragment, useState } from 'react';
import TaskRow, { shouldHandleTaskRowPointerDown } from '../../components/TaskRow';
import TaskInlineEditor from '../../components/TaskInlineEditor';
import NewTaskRow from '../../components/NewTaskRow';
import DropSlot from '../../components/task-row/DropSlot';
import DragOverlay from '../../components/DragOverlay';
import usePointerReorder from '../../hooks/usePointerReorder';
import type { ReorderMove, Task } from '../../types';
import { startTaskPointerDrag, endTaskPointerDrag, updateTask, $projects, $selectedTaskIds } from '../../store/index';
import { useStore } from '@nanostores/react';
import { $taskPointerDrag } from '../../store/ui';
import { todayLocal, tomorrowLocal } from '../../lib/dates';

interface ProjectTaskListProps {
  tasks: Task[];
  projectId: string;
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

  const { reorderState, getPointerHandlers, registerItem, getItemRect } = usePointerReorder({
    getSelectedIds: () => $selectedTaskIds.get(),
    items: tasks,
    onReorder: onReorderTasks,
    shouldHandlePointerDown: (event) => shouldHandleTaskRowPointerDown(event.target),
    onDragStart: startTaskPointerDrag,
    onDragEnd: endTaskPointerDrag,
    onCrossListDrop: handleCrossListDrop,
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
    if (!isDragging) return null;

    const isVisible = reorderState.dropIndex === index
      && index !== draggedTaskIndex
      && index !== draggedTaskIndex + 1;
    const edgeClass = index === 0
      ? ' task-drop-zone-edge-top'
      : index === tasks.length
        ? ' task-drop-zone-edge-bottom'
        : '';

    if (!isVisible) return null;

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
              reorderRef={registerItem(task.id)}
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
