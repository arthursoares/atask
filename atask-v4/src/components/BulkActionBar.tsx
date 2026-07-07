import { useStore } from '@nanostores/react';
import { todayLocal } from '../lib/dates';
import { $selectedTaskIds, completeTask, deleteTasksWithUndo, updateTask, cancelTask } from '../store/index';

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
    // One grace-period toast for the whole batch — Undo restores all of them.
    await deleteTasksWithUndo(ids);
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
      <button className="bulk-bar-btn" onClick={handleComplete} aria-label="Complete selected tasks">✓ Complete</button>
      <button className="bulk-bar-btn" onClick={handleScheduleToday} aria-label="Schedule selected tasks for today">★ Today</button>
      <button className="bulk-bar-btn" onClick={handleScheduleSomeday} aria-label="Schedule selected tasks for someday">Someday</button>
      <button className="bulk-bar-btn" onClick={handleMoveToInbox} aria-label="Move selected tasks to inbox">Inbox</button>
      <button className="bulk-bar-btn" onClick={handleCancel} aria-label="Mark selected tasks as cancelled">Mark Cancelled</button>
      <span className="bulk-action-separator" />
      <button className="bulk-bar-btn danger" onClick={handleDelete} aria-label="Delete selected tasks">Delete</button>
      <button className="bulk-bar-btn" onClick={handleClearSelection} title="Clear selection" aria-label="Clear selection">✕</button>
    </div>
  );
}
