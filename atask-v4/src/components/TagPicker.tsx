import { useEffect, useRef, useState } from 'react';
import { useStore } from '@nanostores/react';
import { $tags, $tagsByTaskId, addTagToTask, removeTagFromTask, createTag } from '../store/index';

interface TagPickerProps {
  taskId: string;
  onClose: () => void;
}

const popoverStyle: React.CSSProperties = {
  position: 'absolute',
  top: '100%',
  left: 0,
  marginTop: 6,
  background: 'var(--canvas-elevated)',
  border: '1px solid var(--border-strong)',
  borderRadius: 'var(--radius-lg)',
  boxShadow: 'var(--shadow-popover)',
  minWidth: 200,
  padding: 0,
  zIndex: 50,
  overflow: 'hidden',
  userSelect: 'none',
};

export default function TagPicker({ taskId, onClose }: TagPickerProps) {
  const tags = useStore($tags);
  const tagsByTaskId = useStore($tagsByTaskId);

  const taskTagIds = tagsByTaskId.get(taskId) ?? new Set<string>();

  const [newTagInput, setNewTagInput] = useState('');
  const popoverRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

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
    <div style={popoverStyle} ref={popoverRef}>
      {/* Header */}
      <div
        style={{
          fontSize: 'var(--text-xs)',
          fontWeight: 700,
          color: 'var(--ink-tertiary)',
          padding: 'var(--sp-3) var(--sp-4) var(--sp-2)',
          textAlign: 'center',
        }}
      >
        Tags
      </div>
      <div style={{ height: 1, background: 'var(--separator)' }} />

      {/* Tag list */}
      {tags.length === 0 && (
        <div
          style={{
            fontSize: 'var(--text-sm)',
            color: 'var(--ink-quaternary)',
            padding: 'var(--sp-3) var(--sp-4)',
            textAlign: 'center',
          }}
        >
          No tags yet
        </div>
      )}
      {tags.map((tag) => {
        const checked = taskTagIds.has(tag.id);
        return (
          <div
            key={tag.id}
            onClick={() => handleToggle(tag.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--sp-3)',
              padding: '5px var(--sp-4)',
              fontSize: 'var(--text-base)',
              color: 'var(--ink-primary)',
              cursor: 'pointer',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.background = 'var(--sidebar-hover)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.background = '';
            }}
          >
            <input
              type="checkbox"
              checked={checked}
              readOnly
              style={{ pointerEvents: 'none' }}
            />
            <span>{tag.title}</span>
          </div>
        );
      })}

      <div style={{ height: 1, background: 'var(--separator)' }} />

      {/* New tag input */}
      <div style={{ padding: 'var(--sp-2) var(--sp-4)' }}>
        <input
          ref={inputRef}
          type="text"
          placeholder="New tag…"
          value={newTagInput}
          onChange={(e) => setNewTagInput(e.target.value)}
          onKeyDown={handleNewTagKeyDown}
          style={{
            width: '100%',
            background: 'transparent',
            border: 'none',
            outline: 'none',
            fontSize: 'var(--text-sm)',
            color: 'var(--ink-primary)',
            fontFamily: 'inherit',
          }}
        />
      </div>
    </div>
  );
}
