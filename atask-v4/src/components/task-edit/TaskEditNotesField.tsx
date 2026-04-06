import type { ChangeEvent, KeyboardEventHandler, Ref } from 'react';

interface TaskEditNotesFieldProps {
  value: string;
  onChange: (nextValue: string) => void;
  placeholder?: string;
  rows?: number;
  className?: string;
  textareaRef?: Ref<HTMLTextAreaElement>;
  onKeyDown?: KeyboardEventHandler<HTMLTextAreaElement>;
}

export default function TaskEditNotesField({
  value,
  onChange,
  placeholder,
  rows,
  className,
  textareaRef,
  onKeyDown,
}: TaskEditNotesFieldProps) {
  return (
    <textarea
      ref={textareaRef}
      className={[
        'task-edit-notes-input',
        className ?? '',
      ]
        .filter(Boolean)
        .join(' ')}
      value={value}
      onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onChange(event.target.value)}
      onKeyDown={onKeyDown}
      placeholder={placeholder}
      rows={rows}
    />
  );
}
