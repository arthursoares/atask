import type { ReactNode } from 'react';

interface EmptyStateProps {
  icon: ReactNode;
  text: string;
}

export default function EmptyState({ icon, text }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon">{icon}</div>
      <p className="empty-state-text">{text}</p>
    </div>
  );
}

export type { EmptyStateProps };
