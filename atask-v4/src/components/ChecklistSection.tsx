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
          <div key={item.id} className="cl-item cl-item-row" role="group">
            <div
              className={`cl-check${isDone ? ' done' : ''} cl-check-action`}
              onClick={() => toggleChecklistItem(item.id)}
            >
              {isDone && (
                <svg viewBox="0 0 12 12">
                  <polyline points="2.5 6 5 8.5 9.5 3.5" />
                </svg>
              )}
            </div>
            <span className={`cl-item-title${isDone ? ' cl-done-text' : ''}`}>
              {item.title}
            </span>
            <span
              onClick={() => deleteChecklistItem(item.id)}
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
        className="cl-new-input"
      />
    </div>
  );
}
