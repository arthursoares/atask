import { useState } from 'react';
import { useChecklistForTask, toggleChecklistItem, deleteChecklistItem, createChecklistItem } from '../store';

interface ChecklistSectionProps {
  taskId: string;
}

export default function ChecklistSection({ taskId }: ChecklistSectionProps) {
  const [newItemTitle, setNewItemTitle] = useState('');

  const items = useChecklistForTask(taskId);

  const handleNewItemKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && newItemTitle.trim()) {
      await createChecklistItem({ title: newItemTitle.trim(), taskId });
      setNewItemTitle('');
    }
  };

  return (
    <div>
      {items.map((item) => {
        const isDone = item.status === 1;
        return (
          <div key={item.id} className="cl-item" style={{ position: 'relative' }} role="group">
            <div
              className={`cl-check${isDone ? ' done' : ''}`}
              style={{ cursor: 'pointer', flexShrink: 0 }}
              onClick={() => toggleChecklistItem(item.id)}
            >
              {isDone && (
                <svg viewBox="0 0 12 12">
                  <polyline points="2.5 6 5 8.5 9.5 3.5" />
                </svg>
              )}
            </div>
            <span className={isDone ? 'cl-done-text' : undefined} style={{ flex: 1 }}>
              {item.title}
            </span>
            <span
              onClick={() => deleteChecklistItem(item.id)}
              style={{
                cursor: 'pointer',
                color: 'var(--ink-quaternary)',
                fontSize: 'var(--text-sm)',
                lineHeight: 1,
                padding: '0 2px',
                opacity: 0,
                transition: 'opacity 0.1s',
              }}
              className="cl-delete-btn"
              title="Delete item"
            >
              ×
            </span>
          </div>
        );
      })}
      <input
        type="text"
        value={newItemTitle}
        onChange={(e) => setNewItemTitle(e.target.value)}
        onKeyDown={handleNewItemKeyDown}
        placeholder="New item"
        style={{
          background: 'transparent',
          border: 'none',
          borderBottom: '1px dashed var(--border)',
          outline: 'none',
          fontSize: 'var(--text-sm)',
          color: 'var(--ink-secondary)',
          width: '100%',
          padding: '4px 0',
          marginTop: 'var(--sp-1)',
        }}
      />
    </div>
  );
}
