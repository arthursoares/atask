import TaskRow from '../../components/TaskRow';
import TaskInlineEditor from '../../components/TaskInlineEditor';
import type { Task } from '../../types';

interface AreaTaskListProps {
  tasks: Task[];
  selectedTaskId: string | null;
  selectedTaskIds: Set<string>;
  expandedTaskId: string | null;
  onSelectTask: (taskId: string) => void;
  onExpandTask: (taskId: string) => void;
  onCloseExpandedTask: () => void;
}

export default function AreaTaskList({
  tasks,
  selectedTaskId,
  selectedTaskIds,
  expandedTaskId,
  onSelectTask,
  onExpandTask,
  onCloseExpandedTask,
}: AreaTaskListProps) {
  if (tasks.length === 0) return null;

  return (
    <div className="area-section">
      <div className="area-section-label">Tasks</div>
      {tasks.map((task) =>
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
            onClick={() => onSelectTask(task.id)}
            onDoubleClick={() => onExpandTask(task.id)}
          />
        ),
      )}
    </div>
  );
}
