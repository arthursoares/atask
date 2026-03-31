interface ActivityEntryProps {
  type: 'human' | 'agent';
  author: string;
  time: string;
  text: string;
}

export function ActivityEntry({ type, author, time, text }: ActivityEntryProps) {
  return (
    <div className="activity-entry">
      <div className={`activity-avatar ${type}`}>
        {type === 'agent' ? '✦' : author.charAt(0).toUpperCase()}
      </div>
      <div className="activity-body">
        <div className="activity-header">
          <span className={`activity-author${type === 'agent' ? ' agent-name' : ''}`}>
            {author}
          </span>
          <span className="activity-time">{time}</span>
        </div>
        <div className="activity-text">{text}</div>
      </div>
    </div>
  );
}

interface ActivityFeedProps {
  taskId: string;
}

export default function ActivityFeed({ taskId: _taskId }: ActivityFeedProps) {
  return (
    <div className="activity-stream">
      <div style={{ fontSize: 'var(--text-xs)', color: 'var(--ink-tertiary)', padding: 'var(--sp-2) 0' }}>
        No activity yet
      </div>
    </div>
  );
}
