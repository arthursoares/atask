import type { KeyboardEvent, RefObject } from 'react';
import TaskEditNotesField from '../task-edit/TaskEditNotesField';

interface EditorNotesFieldProps {
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  value: string;
  onChange: (nextValue: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
}

export default function EditorNotesField({
  textareaRef,
  value,
  onChange,
  onKeyDown,
}: EditorNotesFieldProps) {
  return (
    <TaskEditNotesField
      textareaRef={textareaRef}
      className="task-notes-input task-inline-notes-input"
      value={value}
      onChange={onChange}
      onKeyDown={onKeyDown}
      placeholder="Notes"
      rows={1}
    />
  );
}
