import { Fragment } from 'react';
import TaskRow from '../../components/TaskRow';
import TaskInlineEditor from '../../components/TaskInlineEditor';
import NewTaskRow from '../../components/NewTaskRow';
import DropSlot from '../../components/task-row/DropSlot';
import useDragReorder from '../../hooks/useDragReorder';
import type { ReorderMove, Task } from '../../types';

interface ProjectTaskListProps {
  tasks: Task[];
  expandedTaskId: string | null;
  selectedTaskId: string | null;
  selectedTaskIds: Set<string>;
  onSelectTask: (id: string) => void;
  onExpandTask: (id: string) => void;
  onCloseExpandedTask: () => void;
  onCreateTask: (title: string) => void;
  onReorderTasks: (moves: ReorderMove[]) => Promise<void>;
}

export default function ProjectTaskList({
  tasks,
  expandedTaskId,
  selectedTaskId,
  selectedTaskIds,
  onSelectTask,
  onExpandTask,
  onCloseExpandedTask,
  onCreateTask,
  onReorderTasks,
}: ProjectTaskListProps) {
  const { dragState, getDragHandlers, getDropHandlers } = useDragReorder(tasks, onReorderTasks);
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
    <>
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
              dragHandlers={getDragHandlers(task.id)}
            />
          )}
        </Fragment>
      ))}
      {renderDropZone(tasks.length)}
      <NewTaskRow onCreate={onCreateTask} />
    </>
  );
}
