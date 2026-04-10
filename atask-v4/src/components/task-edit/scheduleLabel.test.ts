import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import scheduleLabel from './scheduleLabel';

/**
 * Unit tests for scheduleLabel — pure formatter that turns
 * {schedule, timeSlot, startDate} into a display label. Lives at the
 * heart of T1.4 and the search-overlay routing fix; if it regresses,
 * future tasks start showing as "Today" again and search lands users
 * in the wrong view.
 */
describe('scheduleLabel', () => {
  // Use a fixed "today" so date-based assertions are deterministic.
  // Vitest's vi.useFakeTimers + system time both freeze new Date().
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-10T12:00:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns null for inbox (schedule = 0)', () => {
    expect(scheduleLabel(0, null, null)).toBeNull();
  });

  it('returns "Someday" for schedule = 2 regardless of date', () => {
    expect(scheduleLabel(2, null, null)).toBe('Someday');
    expect(scheduleLabel(2, null, '2026-04-15')).toBe('Someday');
  });

  it('returns "Today" for schedule = 1 with no startDate', () => {
    expect(scheduleLabel(1, null, null)).toBe('Today');
  });

  it('returns "Today (Evening)" when timeSlot = evening and startDate is today', () => {
    expect(scheduleLabel(1, 'evening', '2026-04-10')).toBe('Today (Evening)');
  });

  it('returns "Today" when startDate is the system date (not in future)', () => {
    expect(scheduleLabel(1, null, '2026-04-10')).toBe('Today');
  });

  it('returns "Today" for past startDates (overdue tasks)', () => {
    expect(scheduleLabel(1, null, '2026-04-01')).toBe('Today');
  });

  it('returns a future-date label for startDate > today (Upcoming)', () => {
    // 2026-04-15 is 5 days after the frozen system time of 2026-04-10.
    const label = scheduleLabel(1, null, '2026-04-15');
    // Format: "<weekday short> <day> <month short>" e.g. "Wed 15 Apr"
    // (locale-dependent — assert it contains the day + month name).
    expect(label).toContain('15');
    expect(label).toContain('Apr');
    // Must NOT say "Today" — that's the bug we're guarding against.
    expect(label).not.toBe('Today');
    expect(label).not.toBe('Today (Anytime)');
  });

  it('includes the year for cross-year future dates', () => {
    const label = scheduleLabel(1, null, '2027-01-05');
    expect(label).toContain('2027');
    expect(label).toContain('Jan');
  });

  it('returns null for unknown schedule values', () => {
    expect(scheduleLabel(99, null, null)).toBeNull();
  });

  it('past startDate with evening time slot still says Today (Evening)', () => {
    expect(scheduleLabel(1, 'evening', '2026-04-08')).toBe('Today (Evening)');
  });

  it('regression: schedule=1 + future startDate is NEVER labeled "Today (Anytime)"', () => {
    // The original bug. This test guards forever against it.
    const label = scheduleLabel(1, null, '2026-05-01');
    expect(label).not.toMatch(/^today/i);
  });
});
