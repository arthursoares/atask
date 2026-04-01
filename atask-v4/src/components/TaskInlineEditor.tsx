import { useEffect, useRef } from 'react';
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
import EditorAttributeBar from './task-inline-editor/EditorAttributeBar';
import EditorNotesField from './task-inline-editor/EditorNotesField';
import useTaskDraft from './task-edit/useTaskDraft';
import useTaskPickers from './task-edit/useTaskPickers';
import type { Task } from '../types';

interface TaskInlineEditorProps {
  task: Task;
  isToday?: boolean;
  onClose: () => void;
}

export default function TaskInlineEditor({ task, isToday, onClose }: TaskInlineEditorProps) {
  const {
    titleValue,
    notesValue,
    setTitleValue,
    setNotesValue,
    flushDraft,
    clearPendingDraft,
  } = useTaskDraft({
    taskId: task.id,
    title: task.title,
    notes: task.notes ?? '',
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
  } = useTaskPickers();

  const containerRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

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
  }, []);

  const handleClose = () => {
    clearPendingDraft();
    const trimmed = titleValue.trim();
    if (!trimmed) {
      deleteTask(task.id);
    } else {
      flushDraft();
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
    setTitleValue(val);
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
    setNotesValue(val);
    adjustHeight();
  };

  const handleNotesKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      handleClose();
    }
  };

  const schedulePillLabel = task.schedule === 1
    ? task.timeSlot === 'evening' ? 'This Evening' : 'Today'
    : task.schedule === 2
      ? 'Someday'
      : null;

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
          value={titleValue}
          onChange={handleTitleChange}
          onKeyDown={handleTitleKeyDown}
          placeholder="Task title"
        />
      </div>

      <EditorNotesField
        textareaRef={textareaRef}
        value={notesValue}
        onChange={handleNotesChange}
        onKeyDown={handleNotesKeyDown}
      />

      <EditorAttributeBar
        task={task}
        project={project}
        taskTags={taskTags}
        scheduleLabel={schedulePillLabel}
        onRemoveSchedule={handleRemoveSchedule}
        onRemoveProject={handleRemoveProject}
        onRemoveTag={handleRemoveTag}
        onShowWhenPicker={() => setShowWhenPicker(true)}
        onShowTagPicker={() => setShowTagPicker(true)}
        onShowRepeatPicker={() => setShowRepeatPicker(true)}
        onShowProjectPicker={() => setShowProjectPicker(true)}
      />

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
