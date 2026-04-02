import type { ReactNode } from 'react';
import TaskEditField from './TaskEditField';

interface TaskEditScheduleFieldProps {
  value: ReactNode;
  onTogglePicker: () => void;
  showPicker: boolean;
  picker?: ReactNode;
  label?: ReactNode;
}

export default function TaskEditScheduleField({
  value,
  onTogglePicker,
  showPicker,
  picker,
  label = 'Schedule',
}: TaskEditScheduleFieldProps) {
  return (
    <TaskEditField label={label} popover>
      <span
        className="task-edit-field-trigger"
        onClick={onTogglePicker}
      >
        {value}
      </span>
      {showPicker ? picker : null}
    </TaskEditField>
  );
}
