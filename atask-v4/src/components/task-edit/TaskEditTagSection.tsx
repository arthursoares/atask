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
  /**
   * When provided, each TagPill renders a × affordance that calls this with
   * the tag id. Keeps parity with the inline editor's per-tag remove button
   * so users don't have to reopen the tag picker just to drop one tag.
   */
  onRemoveTag?: (tagId: string) => void;
}

export default function TaskEditTagSection({
  tags,
  onTogglePicker,
  showPicker,
  picker,
  label = 'Tags',
  addLabel = '+ Add',
  onRemoveTag,
}: TaskEditTagSectionProps) {
  return (
    <TaskEditField label={label} popover>
      <div className="task-edit-tag-row">
        {tags.map((tag) => (
          <span key={tag.id} className="task-edit-tag-chip">
            <TagPill label={tag.title} variant="default" />
            {onRemoveTag && (
              <button
                type="button"
                className="task-edit-tag-remove"
                aria-label={`Remove tag ${tag.title}`}
                onClick={() => onRemoveTag(tag.id)}
              >
                ×
              </button>
            )}
          </span>
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
