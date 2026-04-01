import TaskRow from '../../components/TaskRow';
import TaskInlineEditor from '../../components/TaskInlineEditor';
import NewTaskRow from '../../components/NewTaskRow';
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

  return (
    <>
      {tasks.map((task, index) =>
        expandedTaskId === task.id ? (
          <TaskInlineEditor
            key={task.id}
            task={task}
            onClose={onCloseExpandedTask}
          />
        ) : (
          <TaskRow
            key={task.id}
            task={task}
            isSelected={selectedTaskId === task.id}
            isMultiSelected={selectedTaskIds.has(task.id)}
            taskList={tasks}
            hideProjectPill
            onClick={() => onSelectTask(task.id)}
            onDoubleClick={() => onExpandTask(task.id)}
            dragHandlers={getDragHandlers(task.id)}
            dropHandlers={getDropHandlers(index)}
            isDragOver={dragState.dropIndex === index && dragState.dragId !== task.id}
          />
        ),
      )}
      <NewTaskRow onCreate={onCreateTask} />
    </>
  );
}
