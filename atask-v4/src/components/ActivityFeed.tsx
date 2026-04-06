import { useState } from 'react';
import { useActivitiesForTask, createActivity } from '../store/index';

interface ActivityEntryProps {
  actorType: 'human' | 'agent';
  type: string;
  author: string;
  time: string;
  text: string;
}

function ActivityEntry({ actorType, type, author, time, text }: ActivityEntryProps) {
  if (type === 'status_change') {
    return (
      <div className="activity-entry activity-mutation">
        <div className="activity-body">
          <span className="activity-text activity-mutation-text">{text}</span>
          <span className="activity-time">{time}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="activity-entry">
      <div className={`activity-avatar ${actorType}`}>
        {actorType === 'agent' ? '✦' : author.charAt(0).toUpperCase()}
      </div>
      <div className="activity-body">
        <div className="activity-header">
          <span className={`activity-author${actorType === 'agent' ? ' agent-name' : ''}`}>
            {author}
          </span>
          <span className="activity-time">{time}</span>
        </div>
        <div className="activity-text">{text}</div>
      </div>
    </div>
  );
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  return d.toLocaleDateString();
}

interface ActivityFeedProps {
  taskId: string;
}

export default function ActivityFeed({ taskId }: ActivityFeedProps) {
  const activities = useActivitiesForTask(taskId);
  const [comment, setComment] = useState('');

  const handleSubmit = async () => {
    const text = comment.trim();
    if (!text) return;
    const draft = comment;
    setComment('');
    try {
      await createActivity(taskId, text);
    } catch {
      setComment(draft);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="activity-stream">
      {activities.length === 0 ? (
        <div className="activity-empty">No activity yet</div>
      ) : (
        activities.map((a) => (
          <ActivityEntry
            key={a.id}
            actorType={a.actorType}
            type={a.type}
            author={a.actorId ?? 'You'}
            time={formatTime(a.createdAt)}
            text={a.content}
          />
        ))
      )}
      <div className="activity-comment-input">
        <input
          type="text"
          className="activity-comment-field"
          placeholder="Add a comment…"
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          onKeyDown={handleKeyDown}
        />
      </div>
    </div>
  );
}
