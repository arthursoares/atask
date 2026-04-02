import { useState, useRef } from 'react';

interface NewTaskRowProps {
  onCreate: (title: string) => void;
}

export default function NewTaskRow({ onCreate }: NewTaskRowProps) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  const startEditing = () => {
    setEditing(true);
    // autoFocus via ref after state update
    setTimeout(() => inputRef.current?.focus(), 0);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      if (value.trim()) {
        onCreate(value.trim());
        setValue('');
        // stay in editing for rapid creation
      } else {
        setEditing(false);
        setValue('');
      }
    } else if (e.key === 'Escape') {
      setEditing(false);
      setValue('');
    }
  };

  const handleBlur = () => {
    // If blur happens with no content, exit editing
    if (!value.trim()) {
      setEditing(false);
      setValue('');
    }
  };

  return (
    <div className="new-task-inline" onClick={!editing ? startEditing : undefined}>
      <div className="new-task-plus">+</div>
      {editing ? (
        <input
          ref={inputRef}
          autoFocus
          className="new-task-input"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={handleBlur}
          placeholder="New Task"
        />
      ) : (
        <span>New Task</span>
      )}
    </div>
  );
}
