import { Fragment } from 'react';
import { useStore } from '@nanostores/react';
import {
  useSomeday,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  createTask,
  reorderTasks,
} from '../store/index';
import TaskRow, { shouldHandleTaskRowPointerDown } from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import NewTaskRow from '../components/NewTaskRow';
import EmptyState from '../components/EmptyState';
import DropSlot from '../components/task-row/DropSlot';
import usePointerReorder from '../hooks/usePointerReorder';

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

  const { reorderState, getPointerHandlers, registerItem } = usePointerReorder({
    items: tasks,
    onReorder: reorderTasks,
    shouldHandlePointerDown: (event) => shouldHandleTaskRowPointerDown(event.target),
  });
  const draggedTaskIndex = reorderState.activeId
    ? tasks.findIndex((task) => task.id === reorderState.activeId)
    : -1;
  const isDragging = reorderState.isPointerDragging;

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

    return (
      <div
        key={`drop-zone-${index}`}
        className={`task-drop-zone${edgeClass}`}
      >
        {isVisible ? <DropSlot /> : null}
      </div>
    );
  };

  return (
    <div>
      {tasks.length === 0 ? (
        <EmptyState icon={ClockIcon} text="Nothing for someday" />
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
                  isReordering={reorderState.activeId === task.id && reorderState.isPointerDragging}
                />
              )}
            </Fragment>
          ))}
          {renderDropZone(tasks.length)}
        </>
      )}
      <NewTaskRow onCreate={createTask} />
    </div>
  );
}
