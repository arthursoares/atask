import type { ReactNode } from 'react';
import type { Project, Tag, Task } from '../../types';

interface EditorAttributeBarProps {
  task: Task;
  project: Project | null;
  taskTags: Tag[];
  scheduleLabel: string | null;
  onRemoveSchedule: () => void;
  onRemoveProject: () => void;
  onRemoveTag: (tagId: string) => void;
  onShowWhenPicker: () => void;
  onShowTagPicker: () => void;
  onShowRepeatPicker: () => void;
  onShowProjectPicker: () => void;
}

function EditorPill({
  className,
  children,
  onClick,
}: {
  className: string;
  children: ReactNode;
  onClick?: () => void;
}) {
  return (
    <span className={className} onClick={onClick}>
      {children}
    </span>
  );
}

export default function EditorAttributeBar({
  task,
  project,
  taskTags,
  scheduleLabel,
  onRemoveSchedule,
  onRemoveProject,
  onRemoveTag,
  onShowWhenPicker,
  onShowTagPicker,
  onShowRepeatPicker,
  onShowProjectPicker,
}: EditorAttributeBarProps) {
  return (
    <div className="attr-bar">
      {scheduleLabel && (
        <EditorPill className="attr-pill attr-today">
          {scheduleLabel}
          <span className="remove" onClick={onRemoveSchedule}>×</span>
        </EditorPill>
      )}

      {project && (
        <EditorPill className="attr-pill attr-project">
          <span
            className="attr-project-dot"
            style={{ background: project.color || 'var(--accent)' }}
          />
          {project.title}
          <span className="remove" onClick={onRemoveProject}>×</span>
        </EditorPill>
      )}

      {taskTags.map((tag) => (
        <EditorPill key={tag.id} className="attr-pill attr-tag">
          {tag.title}
          <span className="remove" onClick={() => onRemoveTag(tag.id)}>×</span>
        </EditorPill>
      ))}

      {!scheduleLabel && (
        <EditorPill className="attr-pill attr-add" onClick={onShowWhenPicker}>
          When
        </EditorPill>
      )}
      <EditorPill className="attr-pill attr-add" onClick={onShowTagPicker}>
        +Tag
      </EditorPill>
      {!task.repeatRule && (
        <EditorPill className="attr-pill attr-add" onClick={onShowRepeatPicker}>
          Repeat
        </EditorPill>
      )}
      {!task.projectId && (
        <EditorPill className="attr-pill attr-add" onClick={onShowProjectPicker}>
          Project
        </EditorPill>
      )}
    </div>
  );
}
