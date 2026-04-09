import { useEffect, useRef, useState } from 'react';
import { updateTask } from '../store';
import { PopoverPanel } from '../ui';
import { todayLocal } from '../lib/dates';

interface WhenPickerProps {
  taskId: string;
  currentSchedule: number;
  currentTimeSlot: string | null;
  currentStartDate: string | null;
  anchorRef?: React.RefObject<HTMLElement | null>;
  onClose: () => void;
}

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate();
}

// Returns 0=Monday, 6=Sunday offset for first day of month
function getFirstDayOfMonth(year: number, month: number): number {
  const day = new Date(year, month, 1).getDay(); // 0=Sunday
  return day === 0 ? 6 : day - 1; // shift to Monday start
}

function toDateString(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

export default function WhenPicker({
  taskId,
  currentSchedule,
  currentTimeSlot,
  currentStartDate,
  onClose,
}: WhenPickerProps) {
  const popoverRef = useRef<HTMLDivElement>(null);

  // Navigable month state. Initial view opens on the current start date's
  // month if the task already has one (so editing an Upcoming task shows
  // where it lives), otherwise the current system month.
  const now = new Date();
  const [viewYear, setViewYear] = useState<number>(() => {
    if (currentStartDate) {
      const y = Number(currentStartDate.slice(0, 4));
      return Number.isFinite(y) ? y : now.getFullYear();
    }
    return now.getFullYear();
  });
  const [viewMonth, setViewMonth] = useState<number>(() => {
    if (currentStartDate) {
      const m = Number(currentStartDate.slice(5, 7)) - 1;
      return Number.isFinite(m) ? m : now.getMonth();
    }
    return now.getMonth();
  });

  // Click-outside to close
  useEffect(() => {
    const handleMouseDown = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, [onClose]);

  // Escape-to-close at the picker level so pressing Esc dismisses the
  // picker without closing a containing DetailPanel. Stops propagation to
  // shield the panel's global Esc listener.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [onClose]);

  const todayStr = todayLocal();
  const [systemTodayY, systemTodayM, systemTodayD] = todayStr.split('-').map(Number);
  const isViewingSystemMonth = viewYear === systemTodayY && viewMonth === systemTodayM - 1;

  const daysInMonth = getDaysInMonth(viewYear, viewMonth);
  const firstOffset = getFirstDayOfMonth(viewYear, viewMonth);

  const totalCells = Math.ceil((firstOffset + daysInMonth) / 7) * 7;
  const cells: (number | null)[] = [];
  for (let i = 0; i < totalCells; i++) {
    const dayNum = i - firstOffset + 1;
    cells.push(dayNum >= 1 && dayNum <= daysInMonth ? dayNum : null);
  }

  const rows: (number | null)[][] = [];
  for (let r = 0; r < cells.length / 7; r++) {
    rows.push(cells.slice(r * 7, r * 7 + 7));
  }

  const handleToday = async () => {
    await updateTask({ id: taskId, schedule: 1, timeSlot: null, startDate: todayStr });
    onClose();
  };

  const handleEvening = async () => {
    await updateTask({ id: taskId, schedule: 1, timeSlot: 'evening', startDate: todayStr });
    onClose();
  };

  const handleSomeday = async () => {
    await updateTask({ id: taskId, schedule: 2 });
    onClose();
  };

  const handleDay = async (day: number) => {
    const dateStr = toDateString(viewYear, viewMonth, day);
    // Guard: past dates are almost always accidental. No-op — the user can
    // still explicitly pick Today via the "Today" button above.
    if (dateStr < todayStr) return;
    // Always set schedule: 1 (scheduled). scheduleLabel derives the display
    // bucket ("Today" vs a formatted future date) from startDate, and the
    // Upcoming selector picks up anything with startDate > today.
    await updateTask({ id: taskId, schedule: 1, startDate: dateStr });
    onClose();
  };

  const handleClear = async () => {
    await updateTask({ id: taskId, schedule: 0, startDate: null, timeSlot: null });
    onClose();
  };

  const handlePrevMonth = () => {
    if (viewMonth === 0) {
      setViewYear((y) => y - 1);
      setViewMonth(11);
    } else {
      setViewMonth((m) => m - 1);
    }
  };

  const handleNextMonth = () => {
    if (viewMonth === 11) {
      setViewYear((y) => y + 1);
      setViewMonth(0);
    } else {
      setViewMonth((m) => m + 1);
    }
  };

  // Prev-month button is disabled when viewing the system month or earlier
  // because picking a past date is a no-op (guarded in handleDay).
  const canGoPrev =
    viewYear > systemTodayY || (viewYear === systemTodayY && viewMonth > systemTodayM - 1);

  const isToday = currentSchedule === 1 && currentTimeSlot !== 'evening';
  const isEvening = currentSchedule === 1 && currentTimeSlot === 'evening';
  const isSomeday = currentSchedule === 2;

  const dayLabels = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

  return (
    <PopoverPanel title="When" className="when-popover" popoverRef={popoverRef}>

      <div
        className={`when-option${isToday ? ' selected' : ''}`}
        onClick={handleToday}
      >
        <span className="when-icon">★</span>
        <span>Today</span>
        {isToday && <span className="when-check">✓</span>}
      </div>

      {/* This Evening */}
      <div
        className={`when-option${isEvening ? ' selected' : ''}`}
        onClick={handleEvening}
      >
        <span className="when-icon">🌙</span>
        <span>This Evening</span>
        {isEvening && <span className="when-check">✓</span>}
      </div>

      {/* Mini Calendar */}
      <div className="when-cal">
        <div className="when-cal-nav">
          <button
            type="button"
            className="when-cal-nav-btn"
            onClick={handlePrevMonth}
            disabled={!canGoPrev}
            aria-label="Previous month"
          >
            ‹
          </button>
          <span className="when-cal-title">
            {MONTH_NAMES[viewMonth]} {viewYear}
          </span>
          <button
            type="button"
            className="when-cal-nav-btn"
            onClick={handleNextMonth}
            aria-label="Next month"
          >
            ›
          </button>
        </div>
        <div className="when-cal-header">
          {dayLabels.map((d) => (
            <span key={d}>{d}</span>
          ))}
        </div>
        {rows.map((row, ri) => (
          <div className="when-cal-row" key={ri}>
            {row.map((day, ci) => {
              if (day === null) {
                return <div key={ci} className="when-cal-day empty" />;
              }
              const dateStr = toDateString(viewYear, viewMonth, day);
              const isSelected = currentStartDate === dateStr;
              const isTodayCell = isViewingSystemMonth && day === systemTodayD;
              const isPast = dateStr < todayStr;
              let cls = 'when-cal-day';
              if (isTodayCell) cls += ' today-cal';
              if (isSelected) cls += ' selected-day';
              if (isPast) cls += ' past';
              return (
                <div
                  key={ci}
                  className={cls}
                  onClick={() => handleDay(day)}
                  aria-disabled={isPast ? true : undefined}
                >
                  {day}
                </div>
              );
            })}
          </div>
        ))}
      </div>

      {/* Someday */}
      <div
        className={`when-option${isSomeday ? ' selected' : ''}`}
        onClick={handleSomeday}
      >
        <span className="when-icon">📦</span>
        <span>Someday</span>
        {isSomeday && <span className="when-check">✓</span>}
      </div>

      {/* Add Reminder (disabled) */}
      <div className="when-option when-disabled">
        <span className="when-icon">+</span>
        <span>Add Reminder</span>
      </div>

      <div className="when-sep" />
      <div className="when-clear" onClick={handleClear}>Clear</div>
    </PopoverPanel>
  );
}
