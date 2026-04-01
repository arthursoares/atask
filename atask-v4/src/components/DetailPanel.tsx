import { useCallback, useEffect, useRef, useState } from 'react';
import { useStore } from '@nanostores/react';
import { $tasks, $projects, closeTaskEditor, clearSelectedTask, useTagsForTask, updateTask } from '../store/index';
import ChecklistSection from './ChecklistSection';
import ActivityFeed from './ActivityFeed';
import WhenPicker from './WhenPicker';
import TagPicker from './TagPicker';
import ProjectPicker from './ProjectPicker';
import { TagPill } from '../ui';

interface DetailPanelProps {
  taskId: string;
}

function scheduleLabel(schedule: number, timeSlot: string | null): string {
  if (schedule === 0) return 'Inbox';
  if (schedule === 1 && timeSlot === 'evening') return 'Today (Evening)';
  if (schedule === 1) return 'Today (Anytime)';
  if (schedule === 2) return 'Someday';
  return 'Inbox';
}

export default function DetailPanel({ taskId }: DetailPanelProps) {
  const tasks = useStore($tasks);
  const projects = useStore($projects);
  const task = tasks.find((t) => t.id === taskId);

  const tags = useTagsForTask(taskId);

  const project = task?.projectId ? projects.find((p) => p.id === task.projectId) : null;

  // Local title state for debounced editing
  const [titleValue, setTitleValue] = useState(task?.title ?? '');
  const titleDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Local notes state for debounced editing
  const [notesValue, setNotesValue] = useState(task?.notes ?? '');
  const notesDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Sync local state when task changes externally (different task selected)
  const prevTaskIdRef = useRef(taskId);
  useEffect(() => {
    if (prevTaskIdRef.current !== taskId) {
      prevTaskIdRef.current = taskId;
      setTitleValue(task?.title ?? '');
      setNotesValue(task?.notes ?? '');
    }
  }, [taskId, task?.title, task?.notes]);

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

  // Cleanup debounce timers on unmount
  useEffect(() => {
    return () => {
      if (titleDebounceRef.current) clearTimeout(titleDebounceRef.current);
      if (notesDebounceRef.current) clearTimeout(notesDebounceRef.current);
    };
  }, []);

  const [showWhenPicker, setShowWhenPicker] = useState(false);
  const [showTagPicker, setShowTagPicker] = useState(false);
  const [showProjectPicker, setShowProjectPicker] = useState(false);

  if (!task) return null;

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setTitleValue(value);
    if (titleDebounceRef.current) clearTimeout(titleDebounceRef.current);
    titleDebounceRef.current = setTimeout(() => {
      updateTask({ id: taskId, title: value });
    }, 300);
  };

  const handleNotesChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    setNotesValue(value);
    if (notesDebounceRef.current) clearTimeout(notesDebounceRef.current);
    notesDebounceRef.current = setTimeout(() => {
      updateTask({ id: taskId, notes: value });
    }, 300);
  };

  return (
    <div className="detail-panel">
      <div className="detail-header">
        <input
          className="detail-title"
          value={titleValue}
          onChange={handleTitleChange}
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
        <div className="detail-field detail-field-popover">
          <div className="detail-field-label">Project</div>
          <div className="detail-field-value">
            <span
              className="detail-field-trigger"
              onClick={() => setShowProjectPicker((v) => !v)}
            >
              {project ? (
                <span className="detail-project-value">
                  <span
                    className="detail-project-dot"
                    style={{ background: project.color || 'var(--accent)' }}
                  />
                  {project.title}
                </span>
              ) : (
                <span className="detail-empty-value">None</span>
              )}
            </span>
            {showProjectPicker && (
              <ProjectPicker
                taskId={taskId}
                onClose={() => setShowProjectPicker(false)}
              />
            )}
          </div>
        </div>

        <div className="detail-field detail-field-popover">
          <div className="detail-field-label">Schedule</div>
          <div className="detail-field-value">
            <span
              className="detail-field-trigger"
              onClick={() => setShowWhenPicker((v) => !v)}
            >
              {scheduleLabel(task.schedule, task.timeSlot)}
            </span>
            {showWhenPicker && (
              <WhenPicker
                taskId={taskId}
                currentSchedule={task.schedule}
                currentTimeSlot={task.timeSlot}
                currentStartDate={task.startDate}
                anchorRef={{ current: null }}
                onClose={() => setShowWhenPicker(false)}
              />
            )}
          </div>
        </div>

        <div className="detail-field">
          <div className="detail-field-label">Start Date</div>
          <div className="detail-field-value detail-inline-field">
            <input
              className="detail-date-input"
              type="date"
              value={task.startDate?.slice(0, 10) ?? ''}
              onChange={(e) => updateTask({ id: taskId, startDate: e.target.value || null })}
            />
            {task.startDate && (
              <span
                className="detail-clear-btn"
                onClick={() => updateTask({ id: taskId, startDate: null })}
              >×</span>
            )}
          </div>
        </div>

        <div className="detail-field">
          <div className="detail-field-label">Deadline</div>
          <div className="detail-field-value detail-inline-field">
            <input
              className="detail-date-input"
              type="date"
              value={task.deadline?.slice(0, 10) ?? ''}
              onChange={(e) => updateTask({ id: taskId, deadline: e.target.value || null })}
            />
            {task.deadline && (
              <span
                className="detail-clear-btn"
                onClick={() => updateTask({ id: taskId, deadline: null })}
              >×</span>
            )}
          </div>
        </div>

        <div className="detail-field detail-field-popover">
          <div className="detail-field-label">Tags</div>
          <div className="detail-field-value">
            <div className="detail-tag-row">
              {tags.map((tag) => (
                <TagPill key={tag.id} label={tag.title} variant="default" />
              ))}
              <span
                className="detail-add-link"
                onClick={() => setShowTagPicker((v) => !v)}
              >
                + Add
              </span>
            </div>
            {showTagPicker && (
              <TagPicker
                taskId={taskId}
                onClose={() => setShowTagPicker(false)}
              />
            )}
          </div>
        </div>

        <div className="detail-field">
          <div className="detail-field-label">Notes</div>
          <div className="detail-field-value">
            <textarea
              className="detail-notes-input"
              value={notesValue}
              onChange={handleNotesChange}
              placeholder="Add notes…"
              rows={3}
            />
          </div>
        </div>

        <div className="detail-field">
          <div className="detail-field-label">Checklist</div>
          <ChecklistSection taskId={taskId} />
        </div>

        <div className="detail-activity">
          <div className="detail-field-label detail-activity-label">Activity</div>
          <ActivityFeed taskId={taskId} />
        </div>
      </div>
    </div>
  );
}
