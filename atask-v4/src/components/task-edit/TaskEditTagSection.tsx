import type { ReactNode } from 'react';
import type { Tag } from '../../types';
import { TagPill } from '../../ui';
import TaskEditField from './TaskEditField';

interface TaskEditTagSectionProps {
  tags: Tag[];
  onTogglePicker: () => void;
  showPicker: boolean;
  picker?: ReactNode;
  label?: ReactNode;
  addLabel?: ReactNode;
}

export default function TaskEditTagSection({
  tags,
  onTogglePicker,
  showPicker,
  picker,
  label = 'Tags',
  addLabel = '+ Add',
}: TaskEditTagSectionProps) {
  return (
    <TaskEditField label={label} popover>
      <div className="task-edit-tag-row">
        {tags.map((tag) => (
          <TagPill key={tag.id} label={tag.title} variant="default" />
        ))}
        <span
          className="task-edit-add-link"
          onClick={onTogglePicker}
        >
          {addLabel}
        </span>
      </div>
      {showPicker ? picker : null}
    </TaskEditField>
  );
}
