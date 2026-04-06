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
    <div className="bulk-action-bar">
      <span className="bulk-action-count">{count} selected</span>
      <span className="bulk-action-separator" />
      <button className="bulk-bar-btn" onClick={handleComplete}>✓ Complete</button>
      <button className="bulk-bar-btn" onClick={handleScheduleToday}>★ Today</button>
      <button className="bulk-bar-btn" onClick={handleScheduleSomeday}>Someday</button>
      <button className="bulk-bar-btn" onClick={handleMoveToInbox}>Inbox</button>
      <button className="bulk-bar-btn" onClick={handleCancel}>Cancel</button>
      <span className="bulk-action-separator" />
      <button className="bulk-bar-btn danger" onClick={handleDelete}>Delete</button>
      <button className="bulk-bar-btn" onClick={handleClearSelection} title="Clear selection">✕</button>
    </div>
  );
}
