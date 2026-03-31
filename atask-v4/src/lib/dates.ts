/**
 * Returns today's date as a YYYY-MM-DD string in the local timezone.
 * Use this instead of `new Date().toISOString().slice(0, 10)`, which
 * returns the date in UTC and can be off by a day for users west of UTC.
 */
export function todayLocal(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
