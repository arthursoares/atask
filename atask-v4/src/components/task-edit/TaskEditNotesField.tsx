import type { ChangeEvent } from 'react';

interface TaskEditNotesFieldProps {
  value: string;
  onChange: (nextValue: string) => void;
  placeholder?: string;
  rows?: number;
  className?: string;
}

export default function TaskEditNotesField({
  value,
  onChange,
  placeholder,
  rows,
  className,
}: TaskEditNotesFieldProps) {
  return (
    <textarea
      className={[
        'task-edit-notes-input',
        className ?? '',
      ]
        .filter(Boolean)
        .join(' ')}
      value={value}
      onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onChange(event.target.value)}
      placeholder={placeholder}
      rows={rows}
    />
  );
}
