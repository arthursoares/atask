import { useStore } from "@nanostores/react";
import { useInbox, $selectedTaskId, $expandedTaskId, $selectedTaskIds, createTask, reorderTasks } from "../store/index";
import TaskRow from "../components/TaskRow";
import TaskInlineEditor from "../components/TaskInlineEditor";
import NewTaskRow from "../components/NewTaskRow";
import EmptyState from "../components/EmptyState";
import useDragReorder from "../hooks/useDragReorder";

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

  return (
    <div>
      {tasks.length === 0 ? (
        <EmptyState icon={InboxIcon} text="Inbox is empty" />
      ) : (
        tasks.map((task, index) =>
          expandedTaskId === task.id ? (
            <TaskInlineEditor
              key={task.id}
              task={task}
              onClose={() => $expandedTaskId.set(null)}
            />
          ) : (
            <TaskRow
              key={task.id}
              task={task}
              isSelected={selectedTaskId === task.id}
              isMultiSelected={selectedTaskIds.has(task.id)}
              taskList={tasks}
              onClick={() => $selectedTaskId.set(task.id)}
              onDoubleClick={() => $expandedTaskId.set(task.id)}
              showTriageActions={true}
              dragHandlers={getDragHandlers(task.id)}
              dropHandlers={getDropHandlers(index)}
              isDragOver={dragState.dropIndex === index && dragState.dragId !== task.id}
            />
          ),
        )
      )}
      <NewTaskRow onCreate={(title) => createTask(title)} />
    </div>
  );
}
