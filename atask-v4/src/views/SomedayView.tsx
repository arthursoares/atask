import { Fragment } from 'react';
import { useStore } from '@nanostores/react';
import {
  useSomeday,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  $projects,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  createTask,
  reorderTasks,
  startTaskPointerDrag,
  endTaskPointerDrag,
  updateTask,
} from '../store/index';
import TaskRow, { shouldHandleTaskRowPointerDown } from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import NewTaskRow from '../components/NewTaskRow';
import EmptyState from '../components/EmptyState';
import DropGap from '../components/task-row/DropGap';
import DragOverlay from '../components/DragOverlay';
import TaskDragClone from '../components/task-row/TaskDragClone';
import usePointerReorder from '../hooks/usePointerReorder';
import { todayLocal, tomorrowLocal } from '../lib/dates';

const ClockIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <circle cx="24" cy="24" r="16.5" fill="none" stroke="currentColor" strokeWidth="2" />
    <line x1="24" y1="15" x2="24" y2="24" stroke="currentColor" strokeWidth="2" />
    <line x1="24" y1="24" x2="31.5" y2="30" stroke="currentColor" strokeWidth="2" />
  </svg>
);

export default function SomedayView() {
  const tasks = useSomeday();
  const selectedTaskId = useStore($selectedTaskId);
  const expandedTaskId = useStore($expandedTaskId);
  const selectedTaskIds = useStore($selectedTaskIds);

  const handleCrossListDrop = (taskId: string, target: Element) => {
    const sidebarItemId = target.getAttribute('data-sidebar-item-id');
    const sidebarItemKind = target.getAttribute('data-sidebar-item-kind');

    if (sidebarItemKind === 'project' && sidebarItemId) {
      const allProjects = $projects.get();
      const project = allProjects.find((p) => p.id === sidebarItemId);
      updateTask({ id: taskId, projectId: sidebarItemId, areaId: project?.areaId ?? null, sectionId: null, schedule: 0, startDate: null, timeSlot: null });
      return true;
    }

    if (sidebarItemKind === 'area' && sidebarItemId) {
      updateTask({ id: taskId, areaId: sidebarItemId, projectId: null, sectionId: null, schedule: 0, startDate: null, timeSlot: null });
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
    onReorder: reorderTasks,
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

  const renderDropZone = (index: number) => {
    const open = isDragging
      && reorderState.dropIndex === index
      && index !== draggedTaskIndex
      && index !== draggedTaskIndex + 1;
    const edge = index === 0 ? 'top' as const : index === tasks.length ? 'bottom' as const : null;
    return <DropGap key={`drop-zone-${index}`} active={isDragging} open={open} edge={edge} />;
  };

  const renderDragClone = (id: string) => {
    const task = tasks.find((t) => t.id === id);
    if (!task) return null;
    return <TaskDragClone task={task} />;
  };

  return (
    <div>
      {tasks.length === 0 ? (
        <EmptyState icon={ClockIcon} text="Nothing for someday" hint="Set a task to Someday to park it without a date." />
      ) : (
        <>
          {tasks.map((task, index) => (
            <Fragment key={task.id}>
              {renderDropZone(index)}
              {expandedTaskId === task.id ? (
                <TaskInlineEditor
                  task={task}
                  onClose={closeTaskEditor}
                />
              ) : (
                <TaskRow
                  task={task}
                  isSelected={selectedTaskId === task.id}
                  isMultiSelected={selectedTaskIds.has(task.id)}
                  taskList={tasks}
                  onClick={() => selectTask(task.id)}
                  onDoubleClick={() => openTaskEditor(task.id)}
                  reorderRef={registerItem(task.id)}
                  reorderHandlers={getPointerHandlers(task.id)}
                  isReordering={reorderState.activeId === task.id}
                />
              )}
            </Fragment>
          ))}
          {renderDropZone(tasks.length)}
        </>
      )}
      <NewTaskRow onCreate={createTask} />
      <DragOverlay
        activeId={reorderState.activeId}
        grabOffsetX={reorderState.grabOffsetX}
        grabOffsetY={reorderState.grabOffsetY}
        settleTo={reorderState.settleTo}
        cursorX={reorderState.cursorX}
        cursorY={reorderState.cursorY}
        itemWidth={itemWidth}
        renderClone={renderDragClone}
      />
    </div>
  );
}
