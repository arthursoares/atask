import { useEffect, useRef } from 'react';
import { updateTask } from '../store';

interface WhenPickerProps {
  taskId: string;
  currentSchedule: number;
  currentTimeSlot: string | null;
  currentStartDate: string | null;
  anchorRef: React.RefObject<HTMLElement | null>;
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

export default function WhenPicker({
  taskId,
  currentSchedule,
  currentTimeSlot,
  currentStartDate,
  onClose,
}: WhenPickerProps) {
  // updateTask imported directly from store

  const popoverRef = useRef<HTMLDivElement>(null);

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

  const now = new Date();
  const year = now.getFullYear();
  const month = now.getMonth();
  const todayDay = now.getDate();

  const daysInMonth = getDaysInMonth(year, month);
  const firstOffset = getFirstDayOfMonth(year, month);

  // Build calendar grid rows (7 cols, Monday start)
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

  const todayStr = toDateString(year, month, todayDay);

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
    const dateStr = toDateString(year, month, day);
    await updateTask({ id: taskId, schedule: 1, startDate: dateStr });
    onClose();
  };

  const handleClear = async () => {
    await updateTask({ id: taskId, schedule: 0, startDate: null, timeSlot: null });
    onClose();
  };

  const isToday = currentSchedule === 1 && currentTimeSlot !== 'evening';
  const isEvening = currentSchedule === 1 && currentTimeSlot === 'evening';
  const isSomeday = currentSchedule === 2;

  const dayLabels = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

  return (
    <div className="when-popover" ref={popoverRef}>
      <div className="when-header">When</div>
      <div className="when-sep" />

      {/* Today */}
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
              const dateStr = toDateString(year, month, day);
              const isSelected = currentStartDate === dateStr;
              const isTodayCell = day === todayDay;
              let cls = 'when-cal-day';
              if (isTodayCell) cls += ' today-cal';
              if (isSelected) cls += ' selected-day';
              return (
                <div key={ci} className={cls} onClick={() => handleDay(day)}>
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
    </div>
  );
}
