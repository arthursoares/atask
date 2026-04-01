export default function scheduleLabel(schedule: number, timeSlot: string | null): string {
  if (schedule === 0) return 'Inbox';
  if (schedule === 1 && timeSlot === 'evening') return 'Today (Evening)';
  if (schedule === 1) return 'Today';
  if (schedule === 2) return 'Someday';
  return 'Inbox';
}
