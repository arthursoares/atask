import { useCallback, useEffect } from 'react';
import { useStore } from '@nanostores/react';
import {
  $tasks,
  $projects,
  $locations,
  $linksByTaskId,
  closeTaskEditor,
  clearSelectedTask,
  useTagsForTask,
  updateTask,
  removeTagFromTask,
  removeTaskLink,
  setTaskLocation,
} from '../store/index';
import ChecklistSection from './ChecklistSection';
import ActivityFeed from './ActivityFeed';
import WhenPicker from './WhenPicker';
import TagPicker from './TagPicker';
import ProjectPicker from './ProjectPicker';
import RepeatPicker from './RepeatPicker';
import LocationPicker from './LocationPicker';
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

// Parse a repeat-rule JSON blob into a short display label.
function repeatRuleLabel(repeatRule: string | null): string | null {
  if (!repeatRule) return null;
  try {
    const rule = JSON.parse(repeatRule) as {
      type?: string;
      interval?: number;
      unit?: string;
    };
    const interval = rule.interval ?? 1;
    const unit = rule.unit ?? 'day';
    if (rule.type === 'afterCompletion') {
      return `Every ${interval === 1 ? unit : `${interval} ${unit}s`} after completion`;
    }
    if (interval === 1) {
      return `Every ${unit}`;
    }
    return `Every ${interval} ${unit}s`;
  } catch {
    return null;
  }
}

export default function DetailPanel({ taskId }: DetailPanelProps) {
  const tasks = useStore($tasks);
  const projects = useStore($projects);
  const locations = useStore($locations);
  const linksByTaskId = useStore($linksByTaskId);
  const task = tasks.find((t) => t.id === taskId);

  const tags = useTagsForTask(taskId);

  const project = task?.projectId
    ? projects.find((p) => p.id === task.projectId) ?? null
    : null;

  const currentLocation = task?.locationId
    ? locations.find((l) => l.id === task.locationId) ?? null
    : null;

  // Linked tasks resolved from the bidirectional map. Filter out the task
  // itself (safety net in case of self-links sneaking through old data)
  // and deleted/missing tasks.
  const linkedIds = linksByTaskId.get(taskId);
  const linkedTasks = linkedIds
    ? Array.from(linkedIds)
        .filter((id) => id !== taskId)
        .map((id) => tasks.find((t) => t.id === id))
        .filter((t): t is NonNullable<typeof t> => t != null)
    : [];

  // Escape key to close the panel, but ONLY when no picker is open and
  // focus is not in an input. Pickers install their own capture-phase Esc
  // listeners (see WhenPicker, LocationPicker, RepeatPicker) that call
  // stopPropagation, so this handler never fires while a picker is active.
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      const active = document.activeElement;
      const isEditing =
        active instanceof HTMLInputElement ||
        active instanceof HTMLTextAreaElement ||
        (active instanceof HTMLElement && active.isContentEditable);
      if (isEditing) return;
      closeTaskEditor();
      clearSelectedTask();
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
    showRepeatPicker,
    setShowRepeatPicker,
    showProjectPicker,
    setShowProjectPicker,
    showLocationPicker,
    setShowLocationPicker,
  } = useTaskPickers();

  if (!task) return null;

  const repeatLabel = repeatRuleLabel(task.repeatRule);

  return (
    <div className="detail-panel">
      <div className="detail-header">
        <input
          className="detail-title"
          value={titleValue}
          onChange={(e) => setTitleValue(e.target.value)}
        />
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

        <TaskEditField label="Repeat" popover>
          <div className="task-edit-inline-row">
            <span
              className="task-edit-add-link"
              onClick={() => setShowRepeatPicker((v) => !v)}
            >
              {repeatLabel ?? '+ Add'}
            </span>
            {repeatLabel && (
              <button
                type="button"
                className="task-edit-tag-remove"
                aria-label="Remove recurrence"
                onClick={() => updateTask({ id: taskId, repeatRule: null })}
              >
                ×
              </button>
            )}
          </div>
          {showRepeatPicker && (
            <RepeatPicker
              taskId={taskId}
              currentRepeatRule={task.repeatRule}
              onClose={() => setShowRepeatPicker(false)}
            />
          )}
        </TaskEditField>

        <TaskEditField label="Location" popover>
          <div className="task-edit-inline-row">
            <span
              className="task-edit-add-link"
              onClick={() => setShowLocationPicker((v) => !v)}
            >
              {currentLocation?.name ?? '+ Add'}
            </span>
            {currentLocation && (
              <button
                type="button"
                className="task-edit-tag-remove"
                aria-label="Remove location"
                onClick={() => setTaskLocation(taskId, null)}
              >
                ×
              </button>
            )}
          </div>
          {showLocationPicker && (
            <LocationPicker
              taskId={taskId}
              currentLocationId={task.locationId}
              onClose={() => setShowLocationPicker(false)}
            />
          )}
        </TaskEditField>

        <TaskEditTagSection
          tags={tags}
          onTogglePicker={() => setShowTagPicker((value) => !value)}
          showPicker={showTagPicker}
          onRemoveTag={(tagId) => removeTagFromTask(taskId, tagId)}
          picker={(
            <TagPicker
              taskId={taskId}
              onClose={() => setShowTagPicker(false)}
            />
          )}
        />

        {linkedTasks.length > 0 && (
          <TaskEditField label="Linked Tasks">
            <div className="task-edit-tag-row">
              {linkedTasks.map((linked) => (
                <span key={linked.id} className="task-edit-tag-chip">
                  <span className="task-edit-link-pill">{linked.title}</span>
                  <button
                    type="button"
                    className="task-edit-tag-remove"
                    aria-label={`Unlink ${linked.title}`}
                    onClick={() => removeTaskLink(taskId, linked.id)}
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          </TaskEditField>
        )}

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
