import type { ReactNode } from 'react';
import type { Project } from '../../types';
import TaskEditField from './TaskEditField';

interface TaskEditProjectFieldProps {
  project: Project | null;
  onTogglePicker: () => void;
  showPicker: boolean;
  picker?: ReactNode;
  label?: ReactNode;
  emptyLabel?: ReactNode;
}

export default function TaskEditProjectField({
  project,
  onTogglePicker,
  showPicker,
  picker,
  label = 'Project',
  emptyLabel = 'None',
}: TaskEditProjectFieldProps) {
  return (
    <TaskEditField label={label} popover>
      <span
        className="task-edit-field-trigger"
        onClick={onTogglePicker}
      >
        {project ? (
          <span className="task-edit-project-value">
            <span
              className="task-edit-project-dot"
              style={{ background: project.color || 'var(--accent)' }}
            />
            {project.title}
          </span>
        ) : (
          <span className="task-edit-empty-value">{emptyLabel}</span>
        )}
      </span>
      {showPicker ? picker : null}
    </TaskEditField>
  );
}
