import TaskEditField from './TaskEditField';

interface TaskDateFieldsProps {
  startDate: string | null;
  deadline: string | null;
  onStartDateChange: (nextValue: string | null) => void;
  onDeadlineChange: (nextValue: string | null) => void;
}

interface TaskDateFieldProps {
  label: string;
  value: string | null;
  onChange: (nextValue: string | null) => void;
}

function TaskDateField({ label, value, onChange }: TaskDateFieldProps) {
  return (
    <TaskEditField label={label}>
      <div className="task-edit-inline-field">
        <input
          className="task-edit-date-input"
          type="date"
          value={value?.slice(0, 10) ?? ''}
          onChange={(event) => onChange(event.target.value || null)}
        />
        {value ? (
          <span
            className="task-edit-clear-btn"
            onClick={() => onChange(null)}
          >
            ×
          </span>
        ) : null}
      </div>
    </TaskEditField>
  );
}

export default function TaskDateFields({
  startDate,
  deadline,
  onStartDateChange,
  onDeadlineChange,
}: TaskDateFieldsProps) {
  return (
    <>
      <TaskDateField
        label="Start Date"
        value={startDate}
        onChange={onStartDateChange}
      />
      <TaskDateField
        label="Deadline"
        value={deadline}
        onChange={onDeadlineChange}
      />
    </>
  );
}
