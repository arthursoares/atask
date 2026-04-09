import { useEffect, useRef, useState } from 'react';
import { updateTask } from '../store';
import type { RepeatRule } from '../types';
import { PopoverPanel } from '../ui';

interface RepeatPickerProps {
  taskId: string;
  currentRepeatRule: string | null;
  onClose: () => void;
}

interface QuickOption {
  label: string;
  rule: RepeatRule;
}

const quickOptions: QuickOption[] = [
  { label: 'Daily', rule: { type: 'fixed', interval: 1, unit: 'day' } },
  { label: 'Weekly', rule: { type: 'fixed', interval: 1, unit: 'week' } },
  { label: 'Monthly', rule: { type: 'fixed', interval: 1, unit: 'month' } },
  { label: 'Yearly', rule: { type: 'fixed', interval: 1, unit: 'year' } },
];

function parseCurrentRule(repeatRule: string | null): RepeatRule | null {
  if (!repeatRule) return null;
  try {
    return JSON.parse(repeatRule) as RepeatRule;
  } catch {
    return null;
  }
}

export default function RepeatPicker({ taskId, currentRepeatRule, onClose }: RepeatPickerProps) {
  // updateTask imported directly from store

  const currentRule = parseCurrentRule(currentRepeatRule);
  const [afterCompletion, setAfterCompletion] = useState(currentRule?.type === 'afterCompletion');

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

  // Capture-phase Esc prevents the key from reaching DetailPanel's
  // close-on-Esc handler when the picker is open inside the panel.
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

  const handleSelect = (baseRule: RepeatRule) => {
    const rule: RepeatRule = {
      ...baseRule,
      type: afterCompletion ? 'afterCompletion' : 'fixed',
    };
    updateTask({ id: taskId, repeatRule: JSON.stringify(rule) });
    onClose();
  };

  const handleClear = () => {
    updateTask({ id: taskId, repeatRule: null });
    onClose();
  };

  const isSelected = (opt: QuickOption): boolean => {
    if (!currentRule) return false;
    return (
      currentRule.interval === opt.rule.interval &&
      currentRule.unit === opt.rule.unit &&
      (afterCompletion
        ? currentRule.type === 'afterCompletion'
        : currentRule.type === 'fixed')
    );
  };

  return (
    <PopoverPanel title="Repeat" popoverRef={popoverRef}>
      <button
        type="button"
        className="ui-picker-row ui-picker-row-between ui-picker-toggle-row"
        onClick={() => setAfterCompletion((v) => !v)}
      >
        <span>After Completion</span>
        <span className={`ui-toggle${afterCompletion ? ' is-on' : ''}`}>
          <span className="ui-toggle-thumb" />
        </span>
      </button>
      <div className="ui-popover-separator" />

      {quickOptions.map((opt) => {
        const selected = isSelected(opt);
        return (
          <button
            key={opt.label}
            type="button"
            className={`ui-picker-row ui-picker-row-between${selected ? ' is-selected' : ''}`}
            onClick={() => handleSelect(opt.rule)}
          >
            <span className="ui-picker-label">{opt.label}</span>
            {selected && <span className="ui-picker-check">✓</span>}
          </button>
        );
      })}

      <div className="ui-popover-separator" />

      <button
        type="button"
        className="ui-picker-clear"
        onClick={handleClear}
      >
        Clear
      </button>
    </PopoverPanel>
  );
}
