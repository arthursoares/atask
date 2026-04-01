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
import TaskRow from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import NewTaskRow from '../components/NewTaskRow';
import EmptyState from '../components/EmptyState';
import useDragReorder from '../hooks/useDragReorder';

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

  const { dragState, getDragHandlers, getDropHandlers } = useDragReorder(tasks, reorderTasks);

  return (
    <div>
      {tasks.length === 0 ? (
        <EmptyState icon={ClockIcon} text="Nothing for someday" />
      ) : (
        tasks.map((task, index) =>
          expandedTaskId === task.id ? (
            <TaskInlineEditor
              key={task.id}
              task={task}
              onClose={closeTaskEditor}
            />
          ) : (
            <TaskRow
              key={task.id}
              task={task}
              isSelected={selectedTaskId === task.id}
              isMultiSelected={selectedTaskIds.has(task.id)}
              taskList={tasks}
              onClick={() => selectTask(task.id)}
              onDoubleClick={() => openTaskEditor(task.id)}
              dragHandlers={getDragHandlers(task.id)}
              dropHandlers={getDropHandlers(index)}
              isDragOver={dragState.dropIndex === index && dragState.dragId !== task.id}
            />
          ),
        )
      )}
      <NewTaskRow onCreate={createTask} />
    </div>
  );
}
