import { useCallback, useEffect } from 'react';
import { useStore } from '@nanostores/react';
import { $tasks, $projects, closeTaskEditor, clearSelectedTask, useTagsForTask, updateTask } from '../store/index';
import ChecklistSection from './ChecklistSection';
import ActivityFeed from './ActivityFeed';
import WhenPicker from './WhenPicker';
import TagPicker from './TagPicker';
import ProjectPicker from './ProjectPicker';
import { TagPill } from '../ui';
import TaskDateFields from './task-edit/TaskDateFields';
import TaskEditField from './task-edit/TaskEditField';
import TaskEditNotesField from './task-edit/TaskEditNotesField';
import TaskEditProjectField from './task-edit/TaskEditProjectField';
import TaskEditScheduleField from './task-edit/TaskEditScheduleField';
import TaskEditTagSection from './task-edit/TaskEditTagSection';
import useTaskDraft from './task-edit/useTaskDraft';
import useTaskPickers from './task-edit/useTaskPickers';
import scheduleLabel from './task-edit/scheduleLabel';

interface DetailPanelProps {
  taskId: string;
}

export default function DetailPanel({ taskId }: DetailPanelProps) {
  const tasks = useStore($tasks);
  const projects = useStore($projects);
  const task = tasks.find((t) => t.id === taskId);

  const tags = useTagsForTask(taskId);

  const project = task?.projectId
    ? projects.find((p) => p.id === task.projectId) ?? null
    : null;

  // Escape key to close
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeTaskEditor();
        clearSelectedTask();
      }
    },
    [],
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  const {
    titleValue,
    notesValue,
    setTitleValue,
    setNotesValue,
  } = useTaskDraft({
    taskId,
    title: task?.title ?? '',
    notes: task?.notes ?? '',
  });
  const {
    showWhenPicker,
    setShowWhenPicker,
    showTagPicker,
    setShowTagPicker,
    showProjectPicker,
    setShowProjectPicker,
  } = useTaskPickers();

  if (!task) return null;

  return (
    <div className="detail-panel">
      <div className="detail-header">
        <input
          className="detail-title"
          value={titleValue}
          onChange={(e) => setTitleValue(e.target.value)}
        />
        {tags.length > 0 && (
          <div className="detail-meta-row">
            {tags.map((tag) => (
              <TagPill key={tag.id} label={tag.title} variant="default" />
            ))}
          </div>
        )}
      </div>

      <div className="detail-body">
        <TaskEditProjectField
          project={project}
          onTogglePicker={() => setShowProjectPicker((value) => !value)}
          showPicker={showProjectPicker}
          picker={(
            <ProjectPicker
              taskId={taskId}
              onClose={() => setShowProjectPicker(false)}
            />
          )}
        />

        <TaskEditScheduleField
          value={scheduleLabel(task.schedule, task.timeSlot ?? null, task.startDate ?? null) ?? 'Inbox'}
          onTogglePicker={() => setShowWhenPicker((value) => !value)}
          showPicker={showWhenPicker}
          picker={(
            <WhenPicker
              taskId={taskId}
              currentSchedule={task.schedule}
              currentTimeSlot={task.timeSlot}
              currentStartDate={task.startDate}
              anchorRef={{ current: null }}
              onClose={() => setShowWhenPicker(false)}
            />
          )}
        />

        <TaskDateFields
          startDate={task.startDate}
          deadline={task.deadline}
          onStartDateChange={(startDate) => updateTask({ id: taskId, startDate })}
          onDeadlineChange={(deadline) => updateTask({ id: taskId, deadline })}
        />

        <TaskEditTagSection
          tags={tags}
          onTogglePicker={() => setShowTagPicker((value) => !value)}
          showPicker={showTagPicker}
          picker={(
            <TagPicker
              taskId={taskId}
              onClose={() => setShowTagPicker(false)}
            />
          )}
        />

        <TaskEditField label="Notes">
          <TaskEditNotesField
            value={notesValue}
            onChange={setNotesValue}
            placeholder="Add notes…"
            rows={3}
          />
        </TaskEditField>

        <TaskEditField label="Checklist">
          <ChecklistSection taskId={taskId} />
        </TaskEditField>

        <div className="detail-activity">
          <div className="detail-field-label detail-activity-label">Activity</div>
          <ActivityFeed taskId={taskId} />
        </div>
      </div>
    </div>
  );
}
