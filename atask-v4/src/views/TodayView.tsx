import { Fragment, useCallback } from 'react';
import { useStore } from '@nanostores/react';
import {
  useTodayMorning,
  useTodayEvening,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  $projects,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  createTask,
  setTodayIndex,
  startTaskPointerDrag,
  endTaskPointerDrag,
  updateTask,
} from '../store/index';
import TaskRow, { shouldHandleTaskRowPointerDown } from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import NewTaskRow from '../components/NewTaskRow';
import SectionHeader from '../components/SectionHeader';
import EmptyState from '../components/EmptyState';
import DropSlot from '../components/task-row/DropSlot';
import DragOverlay from '../components/DragOverlay';
import { todayLocal, tomorrowLocal } from '../lib/dates';
import usePointerReorder, { type PointerReorderReturn, type PointerReorderState } from '../hooks/usePointerReorder';

const StarIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <polygon
      points="24 6 29.4 16.8 42 18.6 33 27 35 39 24 33.6 13 39 15 27 6 18.6 18.6 16.8"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
  </svg>
);

export default function TodayView() {
  const morning = useTodayMorning();
  const evening = useTodayEvening();
  const selectedTaskId = useStore($selectedTaskId);
  const expandedTaskId = useStore($expandedTaskId);
  const selectedTaskIds = useStore($selectedTaskIds);

  const handleReorderMorning = useCallback(
    (moves: Array<{ id: string; index: number }>) => {
      for (const move of moves) {
        setTodayIndex(move.id, move.index);
      }
    },
    [],
  );

  const handleReorderEvening = useCallback(
    (moves: Array<{ id: string; index: number }>) => {
      for (const move of moves) {
        setTodayIndex(move.id, move.index);
      }
    },
    [],
  );

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

  const morningReorder = usePointerReorder({
    items: morning,
    onReorder: handleReorderMorning,
    shouldHandlePointerDown: (event) => shouldHandleTaskRowPointerDown(event.target),
    onDragStart: startTaskPointerDrag,
    onDragEnd: endTaskPointerDrag,
    onCrossListDrop: handleCrossListDrop,
    getSelectedIds: () => $selectedTaskIds.get(),
  });

  const eveningReorder = usePointerReorder({
    items: evening,
    onReorder: handleReorderEvening,
    shouldHandlePointerDown: (event) => shouldHandleTaskRowPointerDown(event.target),
    onDragStart: startTaskPointerDrag,
    onDragEnd: endTaskPointerDrag,
    onCrossListDrop: handleCrossListDrop,
    getSelectedIds: () => $selectedTaskIds.get(),
  });

  const renderTaskList = (
    tasks: typeof morning,
    reorderState: PointerReorderState,
    getPointerHandlers: PointerReorderReturn['getPointerHandlers'],
    registerItem: PointerReorderReturn['registerItem'],
    getItemRect: PointerReorderReturn['getItemRect'],
  ) => {
    const draggedTaskIndex = reorderState.activeId
      ? tasks.findIndex((task) => task.id === reorderState.activeId)
      : -1;
    const isDragging = reorderState.isPointerDragging;
    const itemWidth = reorderState.activeId ? getItemRect(reorderState.activeId)?.width ?? null : null;

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
      <>
        {tasks.map((task, index) => (
          <Fragment key={task.id}>
            {renderDropZone(index)}
            {expandedTaskId === task.id ? (
              <TaskInlineEditor
                task={task}
                isToday
                onClose={closeTaskEditor}
              />
            ) : (
              <TaskRow
                task={task}
                isSelected={selectedTaskId === task.id}
                isMultiSelected={selectedTaskIds.has(task.id)}
                taskList={tasks}
                isToday
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
        <DragOverlay
          activeId={reorderState.activeId}
          cursorX={reorderState.cursorX}
          cursorY={reorderState.cursorY}
          itemWidth={itemWidth}
          renderClone={renderDragClone}
        />
      </>
    );
  };

  return (
    <div>
      {renderTaskList(
        morning,
        morningReorder.reorderState,
        morningReorder.getPointerHandlers,
        morningReorder.registerItem,
        morningReorder.getItemRect,
      )}

      {evening.length > 0 && (
        <>
          <SectionHeader title="This Evening" muted />
          {renderTaskList(
            evening,
            eveningReorder.reorderState,
            eveningReorder.getPointerHandlers,
            eveningReorder.registerItem,
            eveningReorder.getItemRect,
          )}
        </>
      )}

      {morning.length === 0 && evening.length === 0 && (
        <EmptyState icon={StarIcon} text="What will you do today?" />
      )}

      <NewTaskRow onCreate={createTask} />
    </div>
  );
}
