import { useEffect, useRef, useState } from 'react';
import { useStore } from '@nanostores/react';
import {
  $projects,
  $tags,
  $tagsByTaskId,
  updateTask,
  deleteTask,
  completeTask,
  reopenTask,
  removeTagFromTask,
} from '../store/index';
import CheckboxCircle from './CheckboxCircle';
import WhenPicker from './WhenPicker';
import TagPicker from './TagPicker';
import ProjectPicker from './ProjectPicker';
import RepeatPicker from './RepeatPicker';
import type { Task, UpdateTaskParams } from '../types';

interface TaskInlineEditorProps {
  task: Task;
  isToday?: boolean;
  onClose: () => void;
}

export default function TaskInlineEditor({ task, isToday, onClose }: TaskInlineEditorProps) {
  const [title, setTitle] = useState(task.title);
  const [notes, setNotes] = useState(task.notes ?? '');

  const [showWhenPicker, setShowWhenPicker] = useState(false);
  const [showTagPicker, setShowTagPicker] = useState(false);
  const [showRepeatPicker, setShowRepeatPicker] = useState(false);
  const [showProjectPicker, setShowProjectPicker] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const projects = useStore($projects);
  const tags = useStore($tags);
  const tagsByTaskId = useStore($tagsByTaskId);

  const project = task.projectId ? projects.find((p) => p.id === task.projectId) ?? null : null;
  const taskTagIds = tagsByTaskId.get(task.id);
  const taskTags = taskTagIds && taskTagIds.size > 0
    ? tags.filter((t) => taskTagIds.has(t.id))
    : [];

  const isCompleted = task.status === 1;
  const isCancelled = task.status === 2;

  // Debounced update helper
  const debouncedUpdate = (params: UpdateTaskParams) => {
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => updateTask(params), 300);
  };

  // Auto-grow textarea
  const adjustHeight = () => {
    const el = textareaRef.current;
    if (el) {
      el.style.height = 'auto';
      el.style.height = el.scrollHeight + 'px';
    }
  };

  // Focus title on mount and adjust textarea height
  useEffect(() => {
    titleRef.current?.focus();
    adjustHeight();
    return () => clearTimeout(timerRef.current);
  }, []);

  const handleClose = () => {
    clearTimeout(timerRef.current);
    const trimmed = title.trim();
    if (!trimmed) {
      deleteTask(task.id);
    } else {
      // Flush any pending debounced updates immediately
      updateTask({ id: task.id, title: trimmed, notes });
    }
    onClose();
  };

  // Click-outside to close — ref avoids re-registering on every render
  const handleCloseRef = useRef(handleClose);
  handleCloseRef.current = handleClose;

  useEffect(() => {
    const handleMouseDown = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        handleCloseRef.current();
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, []);

  const handleCheckboxChange = () => {
    if (isCompleted || isCancelled) {
      reopenTask(task.id);
    } else {
      completeTask(task.id);
    }
  };

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setTitle(val);
    debouncedUpdate({ id: task.id, title: val });
  };

  const handleTitleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleClose();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      handleClose();
    }
  };

  const handleNotesChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const val = e.target.value;
    setNotes(val);
    adjustHeight();
    debouncedUpdate({ id: task.id, notes: val });
  };

  const handleNotesKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      handleClose();
    }
  };

  // Schedule label
  let scheduleLabel: string | null = null;
  if (task.schedule === 1) {
    scheduleLabel = task.timeSlot === 'evening' ? 'This Evening' : 'Today';
  } else if (task.schedule === 2) {
    scheduleLabel = 'Someday';
  }

  const handleRemoveSchedule = () => {
    updateTask({ id: task.id, schedule: 0 });
  };

  const handleRemoveProject = () => {
    updateTask({ id: task.id, projectId: null });
  };

  const handleRemoveTag = (tagId: string) => {
    removeTagFromTask(task.id, tagId);
  };

  return (
    <div className="task-item editing" ref={containerRef}>
      {/* Top row: checkbox + title */}
      <div className="task-editing-top">
        <CheckboxCircle
          checked={isCompleted}
          cancelled={isCancelled}
          today={isToday}
          onChange={handleCheckboxChange}
        />
        <input
          ref={titleRef}
          className="task-title-input"
          type="text"
          value={title}
          onChange={handleTitleChange}
          onKeyDown={handleTitleKeyDown}
          placeholder="Task title"
        />
      </div>

      {/* Notes textarea */}
      <textarea
        ref={textareaRef}
        value={notes}
        onChange={handleNotesChange}
        onKeyDown={handleNotesKeyDown}
        placeholder="Notes"
        rows={1}
        style={{
          marginLeft: 27,
          border: 'none',
          background: 'none',
          outline: 'none',
          resize: 'none',
          fontFamily: 'inherit',
          fontSize: 'var(--text-sm)',
          color: 'var(--ink-secondary)',
          width: 'calc(100% - 27px)',
          padding: 0,
          overflow: 'hidden',
        }}
      />

      {/* Attribute bar */}
      <div className="attr-bar" style={{ paddingLeft: 27 }}>
        {/* Schedule pill */}
        {scheduleLabel && (
          <span className="attr-pill attr-today">
            {scheduleLabel}
            <span className="remove" onClick={handleRemoveSchedule}>×</span>
          </span>
        )}

        {/* Project pill */}
        {project && (
          <span className="attr-pill attr-project">
            <span
              style={{
                display: 'inline-block',
                width: 6,
                height: 6,
                borderRadius: '50%',
                background: project.color || 'var(--accent)',
                marginRight: 4,
                flexShrink: 0,
                verticalAlign: 'middle',
              }}
            />
            {project.title}
            <span className="remove" onClick={handleRemoveProject}>×</span>
          </span>
        )}

        {/* Tag pills */}
        {taskTags.map((tag) => (
          <span key={tag.id} className="attr-pill attr-tag">
            {tag.title}
            <span className="remove" onClick={() => handleRemoveTag(tag.id)}>×</span>
          </span>
        ))}

        {/* Add buttons */}
        {!scheduleLabel && (
          <span className="attr-pill attr-add" onClick={() => setShowWhenPicker(true)}>
            When
          </span>
        )}
        <span className="attr-pill attr-add" onClick={() => setShowTagPicker(true)}>
          +Tag
        </span>
        {!task.repeatRule && (
          <span className="attr-pill attr-add" onClick={() => setShowRepeatPicker(true)}>
            Repeat
          </span>
        )}
        {!task.projectId && (
          <span className="attr-pill attr-add" onClick={() => setShowProjectPicker(true)}>
            Project
          </span>
        )}
      </div>

      {/* Pickers */}
      {showWhenPicker && (
        <WhenPicker
          taskId={task.id}
          currentSchedule={task.schedule}
          currentTimeSlot={task.timeSlot}
          currentStartDate={task.startDate}
          anchorRef={containerRef}
          onClose={() => setShowWhenPicker(false)}
        />
      )}
      {showTagPicker && (
        <TagPicker
          taskId={task.id}
          onClose={() => setShowTagPicker(false)}
        />
      )}
      {showRepeatPicker && (
        <RepeatPicker
          taskId={task.id}
          currentRepeatRule={task.repeatRule}
          onClose={() => setShowRepeatPicker(false)}
        />
      )}
      {showProjectPicker && (
        <ProjectPicker
          taskId={task.id}
          onClose={() => setShowProjectPicker(false)}
        />
      )}
    </div>
  );
}
