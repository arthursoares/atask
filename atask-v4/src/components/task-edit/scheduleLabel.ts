import { todayLocal } from '../../lib/dates';

/**
 * Return the human-readable label for a task's schedule state.
 *
 * schedule === 1 means "scheduled" but the display label depends on whether
 * startDate is in the past/today (-> Today) or future (-> Upcoming + date).
 * The previous implementation ignored startDate and mislabeled every
 * scheduled task as "Today (Anytime)", which also caused the SearchOverlay
 * to route future tasks to the wrong view.
 *
 * Returns null for schedule === 0 (Inbox) so callers can distinguish
 * "unscheduled" from "no label shown".
 */
export default function scheduleLabel(
  schedule: number,
  timeSlot: string | null,
  startDate: string | null,
): string | null {
  if (schedule === 2) return 'Someday';
  if (schedule === 0) return null;
  if (schedule !== 1) return null;

  // schedule === 1 (scheduled). Bucket by start date.
  const today = todayLocal();
  const date = startDate ? startDate.slice(0, 10) : null;

  if (date && date > today) {
    // Upcoming — show the future date.
    return formatUpcomingDate(date);
  }

  // No start date, or start date is today/past → Today.
  if (timeSlot === 'evening') return 'Today (Evening)';
  return 'Today';
}

/**
 * Format a YYYY-MM-DD string for display in the schedule pill.
 * Same-year dates: "Sun 12 Apr". Future-year dates include the year.
 */
function formatUpcomingDate(dateStr: string): string {
  const [yStr, mStr, dStr] = dateStr.split('-');
  const year = Number(yStr);
  const month = Number(mStr) - 1;
  const day = Number(dStr);
  const date = new Date(year, month, day);

  const today = new Date();
  const sameYear = today.getFullYear() === year;

  const weekday = date.toLocaleDateString(undefined, { weekday: 'short' });
  const monthName = date.toLocaleDateString(undefined, { month: 'short' });
  if (sameYear) {
    return `${weekday} ${day} ${monthName}`;
  }
  return `${day} ${monthName} ${year}`;
}
