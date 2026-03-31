import { useStore } from '@nanostores/react';
import { todayLocal } from '../lib/dates';
import { $selectedTaskIds, completeTask, deleteTask, updateTask, cancelTask } from '../store/index';

export default function BulkActionBar() {
  const selectedTaskIds = useStore($selectedTaskIds);

  if (selectedTaskIds.size === 0) return null;

  const count = selectedTaskIds.size;
  const ids = [...selectedTaskIds];

  const handleComplete = async () => {
    for (const id of ids) await completeTask(id);
    $selectedTaskIds.set(new Set());
  };

  const handleDelete = async () => {
    for (const id of ids) await deleteTask(id);
    $selectedTaskIds.set(new Set());
  };

  const handleScheduleToday = async () => {
    const today = todayLocal();
    for (const id of ids) await updateTask({ id, schedule: 1, startDate: today });
    $selectedTaskIds.set(new Set());
  };

  const handleScheduleSomeday = async () => {
    for (const id of ids) await updateTask({ id, schedule: 2 });
    $selectedTaskIds.set(new Set());
  };

  const handleMoveToInbox = async () => {
    for (const id of ids) await updateTask({ id, schedule: 0 });
    $selectedTaskIds.set(new Set());
  };

  const handleCancel = async () => {
    for (const id of ids) await cancelTask(id);
    $selectedTaskIds.set(new Set());
  };

  const handleClearSelection = () => {
    $selectedTaskIds.set(new Set());
  };

  return (
    <div style={{
      position: 'fixed',
      bottom: 'var(--sp-5)',
      left: '50%',
      transform: 'translateX(-50%)',
      background: 'var(--canvas-elevated)',
      border: '1px solid var(--border-strong)',
      borderRadius: 'var(--radius-xl)',
      boxShadow: 'var(--shadow-popover)',
      padding: 'var(--sp-2) var(--sp-4)',
      display: 'flex',
      alignItems: 'center',
      gap: 'var(--sp-3)',
      zIndex: 90,
      fontSize: 'var(--text-sm)',
    }}>
      <span style={{ fontWeight: 700, color: 'var(--ink-primary)' }}>
        {count} selected
      </span>
      <span style={{ width: 1, height: 16, background: 'var(--separator)' }} />
      <button className="bulk-bar-btn" onClick={handleComplete}>✓ Complete</button>
      <button className="bulk-bar-btn" onClick={handleScheduleToday}>★ Today</button>
      <button className="bulk-bar-btn" onClick={handleScheduleSomeday}>Someday</button>
      <button className="bulk-bar-btn" onClick={handleMoveToInbox}>Inbox</button>
      <button className="bulk-bar-btn" onClick={handleCancel}>Cancel</button>
      <span style={{ width: 1, height: 16, background: 'var(--separator)' }} />
      <button className="bulk-bar-btn danger" onClick={handleDelete}>Delete</button>
      <button className="bulk-bar-btn" onClick={handleClearSelection} title="Clear selection">✕</button>
    </div>
  );
}
