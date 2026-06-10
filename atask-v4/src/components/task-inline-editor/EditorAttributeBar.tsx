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

// Interactive pills render as real buttons so they're focusable and
// activate on Enter/Space; static pills (attribute display) stay spans.
function EditorPill({
  className,
  children,
  onClick,
  label,
}: {
  className: string;
  children: ReactNode;
  onClick?: () => void;
  label?: string;
}) {
  if (onClick) {
    return (
      <button type="button" className={className} onClick={onClick} aria-label={label}>
        {children}
      </button>
    );
  }
  return <span className={className}>{children}</span>;
}

function RemoveButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button type="button" className="remove" aria-label={label} onClick={onClick}>
      ×
    </button>
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
          <RemoveButton label="Remove schedule" onClick={onRemoveSchedule} />
        </EditorPill>
      )}

      {project && (
        <EditorPill className="attr-pill attr-project">
          <span
            className="attr-project-dot"
            style={{ background: project.color || 'var(--accent)' }}
          />
          {project.title}
          <RemoveButton label="Remove from project" onClick={onRemoveProject} />
        </EditorPill>
      )}

      {taskTags.map((tag) => (
        <EditorPill key={tag.id} className="attr-pill attr-tag">
          {tag.title}
          <RemoveButton label={`Remove tag ${tag.title}`} onClick={() => onRemoveTag(tag.id)} />
        </EditorPill>
      ))}

      {!scheduleLabel && (
        <EditorPill className="attr-pill attr-add" onClick={onShowWhenPicker} label="Set schedule">
          When
        </EditorPill>
      )}
      <EditorPill className="attr-pill attr-add" onClick={onShowTagPicker} label="Add tag">
        +Tag
      </EditorPill>
      {!task.repeatRule && (
        <EditorPill className="attr-pill attr-add" onClick={onShowRepeatPicker} label="Set repeat rule">
          Repeat
        </EditorPill>
      )}
      {!task.projectId && (
        <EditorPill className="attr-pill attr-add" onClick={onShowProjectPicker} label="Move to project">
          Project
        </EditorPill>
      )}
    </div>
  );
}
