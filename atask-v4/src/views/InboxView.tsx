import { Fragment } from 'react';
import { useStore } from '@nanostores/react';
import {
  useInbox,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  createTask,
  reorderTasks,
} from '../store/index';
import TaskRow from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import NewTaskRow from '../components/NewTaskRow';
import EmptyState from '../components/EmptyState';
import DropSlot from '../components/task-row/DropSlot';
import useDragReorder from '../hooks/useDragReorder';

const InboxIcon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <rect
      x="6"
      y="9"
      width="36"
      height="30"
      rx="6"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
    <polyline
      points="6 24 18 24 21 30 27 30 30 24 42 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
  </svg>
);

export default function InboxView() {
  const tasks = useInbox();
  const selectedTaskId = useStore($selectedTaskId);
  const expandedTaskId = useStore($expandedTaskId);
  const selectedTaskIds = useStore($selectedTaskIds);

  const { dragState, getDragHandlers, getDropHandlers } = useDragReorder(tasks, reorderTasks);
  const draggedTaskIndex = dragState.dragId
    ? tasks.findIndex((task) => task.id === dragState.dragId)
    : -1;
  const isDragging = dragState.dragId !== null;

  const renderDropZone = (index: number) => {
    if (!isDragging) return null;

    const isVisible = dragState.dropIndex === index
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
        {...getDropHandlers(index)}
      >
        {isVisible ? <DropSlot /> : null}
      </div>
    );
  };

  return (
    <div>
      {tasks.length === 0 ? (
        <EmptyState icon={InboxIcon} text="Inbox is empty" />
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
                  showTriageActions={true}
                  dragHandlers={getDragHandlers(task.id)}
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
