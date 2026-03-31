import { useCallback, useEffect, useRef, useState } from 'react';
import { useStore } from '@nanostores/react';
import { $tasks, $projects, $selectedTaskId, useTagsForTask, updateTask } from '../store/index';
import TagPill from './TagPill';
import ChecklistSection from './ChecklistSection';
import ActivityFeed from './ActivityFeed';
import WhenPicker from './WhenPicker';
import TagPicker from './TagPicker';
import ProjectPicker from './ProjectPicker';

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
  const setSelectedTaskId = (id: string | null) => $selectedTaskId.set(id);

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
        setSelectedTaskId(null);
      }
    },
    [setSelectedTaskId],
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
      {/* Header */}
      <div className="detail-header">
        <input
          className="detail-title"
          value={titleValue}
          onChange={handleTitleChange}
          style={{
            background: 'transparent',
            border: 'none',
            outline: 'none',
            width: '100%',
            display: 'block',
            fontFamily: 'inherit',
          }}
        />
        {tags.length > 0 && (
          <div className="detail-meta-row">
            {tags.map((tag) => (
              <TagPill key={tag.id} label={tag.title} variant="default" />
            ))}
          </div>
        )}
      </div>

      {/* Body */}
      <div className="detail-body">
        {/* Project */}
        <div className="detail-field" style={{ position: 'relative' }}>
          <div className="detail-field-label">Project</div>
          <div className="detail-field-value">
            <span
              style={{ cursor: 'pointer' }}
              onClick={() => setShowProjectPicker((v) => !v)}
            >
              {project ? (
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      background: project.color || 'var(--accent)',
                      flexShrink: 0,
                      display: 'inline-block',
                    }}
                  />
                  {project.title}
                </span>
              ) : (
                <span style={{ color: 'var(--ink-quaternary)' }}>None</span>
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

        {/* Schedule */}
        <div className="detail-field" style={{ position: 'relative' }}>
          <div className="detail-field-label">Schedule</div>
          <div className="detail-field-value">
            <span
              style={{ cursor: 'pointer' }}
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

        {/* Start Date */}
        <div className="detail-field">
          <div className="detail-field-label">Start Date</div>
          <div className="detail-field-value" style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-2)' }}>
            <input
              type="date"
              value={task.startDate?.slice(0, 10) ?? ''}
              onChange={(e) => updateTask({ id: taskId, startDate: e.target.value || null })}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 'var(--text-sm)',
                color: 'var(--ink-secondary)',
                fontFamily: 'inherit',
                cursor: 'pointer',
              }}
            />
            {task.startDate && (
              <span
                style={{ cursor: 'pointer', color: 'var(--ink-quaternary)', fontSize: 'var(--text-xs)' }}
                onClick={() => updateTask({ id: taskId, startDate: null })}
              >×</span>
            )}
          </div>
        </div>

        {/* Deadline */}
        <div className="detail-field">
          <div className="detail-field-label">Deadline</div>
          <div className="detail-field-value" style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-2)' }}>
            <input
              type="date"
              value={task.deadline?.slice(0, 10) ?? ''}
              onChange={(e) => updateTask({ id: taskId, deadline: e.target.value || null })}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                fontSize: 'var(--text-sm)',
                color: 'var(--ink-secondary)',
                fontFamily: 'inherit',
                cursor: 'pointer',
              }}
            />
            {task.deadline && (
              <span
                style={{ cursor: 'pointer', color: 'var(--ink-quaternary)', fontSize: 'var(--text-xs)' }}
                onClick={() => updateTask({ id: taskId, deadline: null })}
              >×</span>
            )}
          </div>
        </div>

        {/* Tags */}
        <div className="detail-field" style={{ position: 'relative' }}>
          <div className="detail-field-label">Tags</div>
          <div className="detail-field-value">
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--sp-1)', alignItems: 'center' }}>
              {tags.map((tag) => (
                <TagPill key={tag.id} label={tag.title} variant="default" />
              ))}
              <span
                style={{
                  fontSize: 'var(--text-xs)',
                  color: 'var(--accent)',
                  cursor: 'pointer',
                }}
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

        {/* Notes */}
        <div className="detail-field">
          <div className="detail-field-label">Notes</div>
          <div className="detail-field-value">
            <textarea
              value={notesValue}
              onChange={handleNotesChange}
              placeholder="Add notes…"
              rows={3}
              style={{
                background: 'transparent',
                border: 'none',
                outline: 'none',
                resize: 'vertical',
                width: '100%',
                fontFamily: 'inherit',
                fontSize: 'var(--text-sm)',
                color: 'var(--ink-secondary)',
                lineHeight: 'var(--leading-relaxed)',
                padding: 0,
              }}
            />
          </div>
        </div>

        {/* Checklist */}
        <div className="detail-field">
          <div className="detail-field-label">Checklist</div>
          <ChecklistSection taskId={taskId} />
        </div>

        {/* Activity */}
        <div style={{ borderTop: '1px solid var(--separator)', marginTop: 'var(--sp-2)', paddingTop: 'var(--sp-4)' }}>
          <div className="detail-field-label" style={{ marginBottom: 'var(--sp-2)' }}>Activity</div>
          <ActivityFeed taskId={taskId} />
        </div>
      </div>
    </div>
  );
}
