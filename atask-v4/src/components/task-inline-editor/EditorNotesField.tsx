import type { ChangeEvent, KeyboardEvent, RefObject } from 'react';

interface EditorNotesFieldProps {
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  value: string;
  onChange: (event: ChangeEvent<HTMLTextAreaElement>) => void;
  onKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
}

export default function EditorNotesField({
  textareaRef,
  value,
  onChange,
  onKeyDown,
}: EditorNotesFieldProps) {
  return (
    <textarea
      ref={textareaRef}
      className="task-notes-input"
      value={value}
      onChange={onChange}
      onKeyDown={onKeyDown}
      placeholder="Notes"
      rows={1}
    />
  );
}
