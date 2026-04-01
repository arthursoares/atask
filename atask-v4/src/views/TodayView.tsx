import { useCallback } from 'react';
import { useStore } from '@nanostores/react';
import {
  useTodayMorning,
  useTodayEvening,
  $selectedTaskId,
  $expandedTaskId,
  $selectedTaskIds,
  selectTask,
  openTaskEditor,
  closeTaskEditor,
  createTask,
  setTodayIndex,
} from '../store/index';
import TaskRow from '../components/TaskRow';
import TaskInlineEditor from '../components/TaskInlineEditor';
import NewTaskRow from '../components/NewTaskRow';
import SectionHeader from '../components/SectionHeader';
import EmptyState from '../components/EmptyState';
import useDragReorder from '../hooks/useDragReorder';

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

  const {
    dragState: morningDragState,
    getDragHandlers: getMorningDragHandlers,
    getDropHandlers: getMorningDropHandlers,
  } = useDragReorder(morning, handleReorderMorning);

  const {
    dragState: eveningDragState,
    getDragHandlers: getEveningDragHandlers,
    getDropHandlers: getEveningDropHandlers,
  } = useDragReorder(evening, handleReorderEvening);

  return (
    <div>
      {morning.map((task, index) =>
        expandedTaskId === task.id ? (
          <TaskInlineEditor
            key={task.id}
            task={task}
            isToday
            onClose={closeTaskEditor}
          />
        ) : (
          <TaskRow
            key={task.id}
            task={task}
            isSelected={selectedTaskId === task.id}
            isMultiSelected={selectedTaskIds.has(task.id)}
            taskList={morning}
            isToday
            onClick={() => selectTask(task.id)}
            onDoubleClick={() => openTaskEditor(task.id)}
            dragHandlers={getMorningDragHandlers(task.id)}
            dropHandlers={getMorningDropHandlers(index)}
            isDragOver={morningDragState.dropIndex === index && morningDragState.dragId !== task.id}
          />
        ),
      )}

      {evening.length > 0 && (
        <>
          <SectionHeader title="This Evening" muted />
          {evening.map((task, index) =>
            expandedTaskId === task.id ? (
              <TaskInlineEditor
                key={task.id}
                task={task}
                isToday
                onClose={closeTaskEditor}
              />
            ) : (
              <TaskRow
                key={task.id}
                task={task}
                isSelected={selectedTaskId === task.id}
                isMultiSelected={selectedTaskIds.has(task.id)}
                taskList={evening}
                isToday
                onClick={() => selectTask(task.id)}
                onDoubleClick={() => openTaskEditor(task.id)}
                dragHandlers={getEveningDragHandlers(task.id)}
                dropHandlers={getEveningDropHandlers(index)}
                isDragOver={eveningDragState.dropIndex === index && eveningDragState.dragId !== task.id}
              />
            ),
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
