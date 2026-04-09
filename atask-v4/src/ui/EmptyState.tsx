import type { ReactNode } from 'react';

interface EmptyStateProps {
  icon: ReactNode;
  text: string;
  /**
   * Optional secondary line of muted helper text. Use this for onboarding
   * hints like "Press ⌘N to create your first task" that teach the user
   * how to get out of the empty state.
   */
  hint?: ReactNode;
  /**
   * Optional call-to-action slot rendered below the hint. Typically a
   * Button element, but any ReactNode is accepted for custom layouts.
   */
  action?: ReactNode;
}

export default function EmptyState({ icon, text, hint, action }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon" aria-hidden="true">{icon}</div>
      <p className="empty-state-text">{text}</p>
      {hint && <p className="empty-state-hint">{hint}</p>}
      {action && <div className="empty-state-action">{action}</div>}
    </div>
  );
}

export type { EmptyStateProps };
