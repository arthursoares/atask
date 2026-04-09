import { useEffect, useRef, useState } from 'react';
import { useStore } from '@nanostores/react';
import { $tags, $tagsByTaskId, addTagToTask, removeTagFromTask, createTag } from '../store/index';
import { PopoverPanel } from '../ui';

interface TagPickerProps {
  taskId: string;
  onClose: () => void;
}

export default function TagPicker({ taskId, onClose }: TagPickerProps) {
  const tags = useStore($tags);
  const tagsByTaskId = useStore($tagsByTaskId);

  const taskTagIds = tagsByTaskId.get(taskId) ?? new Set<string>();

  const [newTagInput, setNewTagInput] = useState('');
  const popoverRef = useRef<HTMLDivElement>(null);

  // Click-outside to close
  useEffect(() => {
    const handleMouseDown = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, [onClose]);

  // Layered Escape: capture-phase listener that handles Esc before any
  // surrounding DetailPanel-level handler sees it. Stops propagation so
  // dismissing the picker does not also close the whole panel.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [onClose]);

  const handleToggle = (tagId: string) => {
    if (taskTagIds.has(tagId)) {
      removeTagFromTask(taskId, tagId);
    } else {
      addTagToTask(taskId, tagId);
    }
  };

  const handleNewTagKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const title = newTagInput.trim();
      if (!title) return;
      const newTag = await createTag({ title });
      if (newTag) {
        await addTagToTask(taskId, newTag.id);
      }
      setNewTagInput('');
    } else if (e.key === 'Escape') {
      onClose();
    }
  };

  return (
    <PopoverPanel title="Tags" popoverRef={popoverRef}>
      {tags.length === 0 && (
        <div className="ui-picker-empty-state">
          No tags yet
        </div>
      )}
      {tags.map((tag) => {
        const checked = taskTagIds.has(tag.id);
        return (
          <button
            key={tag.id}
            type="button"
            className="ui-picker-row"
            onClick={() => handleToggle(tag.id)}
          >
            <input
              type="checkbox"
              checked={checked}
              readOnly
              className="ui-picker-checkbox"
            />
            <span className="ui-picker-label">{tag.title}</span>
          </button>
        );
      })}

      <div className="ui-popover-separator" />

      <div className="ui-picker-input-wrap">
        <input
          type="text"
          className="ui-picker-input"
          placeholder="New tag…"
          value={newTagInput}
          onChange={(e) => setNewTagInput(e.target.value)}
          onKeyDown={handleNewTagKeyDown}
        />
      </div>
    </PopoverPanel>
  );
}
