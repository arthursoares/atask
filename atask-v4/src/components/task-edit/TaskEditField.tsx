import type { ReactNode } from 'react';

interface TaskEditFieldProps {
  label: ReactNode;
  children: ReactNode;
  popover?: boolean;
  className?: string;
}

export default function TaskEditField({
  label,
  children,
  popover,
  className,
}: TaskEditFieldProps) {
  const rootClassName = [
    'task-edit-field',
    popover ? 'task-edit-field-popover' : '',
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div className={rootClassName}>
      <div className="task-edit-field-label">{label}</div>
      <div className="task-edit-field-value">{children}</div>
    </div>
  );
}
