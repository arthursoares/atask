import { useEffect, useRef, useState } from 'react';
import { updateTask } from '../store';
import type { RepeatRule } from '../types';

interface RepeatPickerProps {
  taskId: string;
  currentRepeatRule: string | null;
  onClose: () => void;
}

const popoverStyle: React.CSSProperties = {
  position: 'absolute',
  top: '100%',
  left: 0,
  marginTop: 6,
  background: 'var(--canvas-elevated)',
  border: '1px solid var(--border-strong)',
  borderRadius: 'var(--radius-lg)',
  boxShadow: 'var(--shadow-popover)',
  minWidth: 220,
  padding: 0,
  zIndex: 50,
  overflow: 'hidden',
  userSelect: 'none',
};

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

  const hoverOn = (e: React.MouseEvent<HTMLElement>) => {
    (e.currentTarget as HTMLElement).style.background = 'var(--sidebar-hover)';
  };
  const hoverOff = (e: React.MouseEvent<HTMLElement>) => {
    (e.currentTarget as HTMLElement).style.background = '';
  };

  return (
    <div style={popoverStyle} ref={popoverRef}>
      {/* Header */}
      <div
        style={{
          fontSize: 'var(--text-xs)',
          fontWeight: 700,
          color: 'var(--ink-tertiary)',
          padding: 'var(--sp-3) var(--sp-4) var(--sp-2)',
          textAlign: 'center',
        }}
      >
        Repeat
      </div>
      <div style={{ height: 1, background: 'var(--separator)' }} />

      {/* After Completion toggle */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '6px var(--sp-4)',
          fontSize: 'var(--text-sm)',
          color: 'var(--ink-secondary)',
          cursor: 'pointer',
        }}
        onClick={() => setAfterCompletion((v) => !v)}
        onMouseEnter={hoverOn}
        onMouseLeave={hoverOff}
      >
        <span>After Completion</span>
        <span
          style={{
            width: 32,
            height: 18,
            borderRadius: 9,
            background: afterCompletion ? 'var(--accent)' : 'var(--border-strong)',
            display: 'flex',
            alignItems: 'center',
            padding: '0 3px',
            transition: 'background 0.15s',
            flexShrink: 0,
          }}
        >
          <span
            style={{
              width: 12,
              height: 12,
              borderRadius: '50%',
              background: '#fff',
              marginLeft: afterCompletion ? 14 : 0,
              transition: 'margin-left 0.15s',
            }}
          />
        </span>
      </div>
      <div style={{ height: 1, background: 'var(--separator)' }} />

      {/* Quick options */}
      {quickOptions.map((opt) => {
        const selected = isSelected(opt);
        return (
          <div
            key={opt.label}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '5px var(--sp-4)',
              fontSize: 'var(--text-base)',
              color: selected ? 'var(--accent)' : 'var(--ink-primary)',
              cursor: 'pointer',
              fontWeight: selected ? 600 : undefined,
            }}
            onClick={() => handleSelect(opt.rule)}
            onMouseEnter={hoverOn}
            onMouseLeave={hoverOff}
          >
            <span>{opt.label}</span>
            {selected && <span style={{ fontSize: 14 }}>✓</span>}
          </div>
        );
      })}

      <div style={{ height: 1, background: 'var(--separator)' }} />

      {/* Clear */}
      <div
        style={{
          textAlign: 'center',
          padding: 'var(--sp-2)',
          fontSize: 'var(--text-sm)',
          fontWeight: 700,
          color: 'var(--ink-secondary)',
          cursor: 'pointer',
        }}
        onClick={handleClear}
        onMouseEnter={hoverOn}
        onMouseLeave={hoverOff}
      >
        Clear
      </div>
    </div>
  );
}
