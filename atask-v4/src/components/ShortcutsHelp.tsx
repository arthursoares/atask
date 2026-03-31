import { useEffect } from 'react';
import { useStore } from '@nanostores/react';
import { $showShortcuts } from '../store';

interface ShortcutItem {
  keys: string;
  action: string;
}

interface ShortcutCategory {
  category: string;
  items: ShortcutItem[];
}

const shortcuts: ShortcutCategory[] = [
  { category: 'Navigation', items: [
    { keys: '⌘1', action: 'Inbox' },
    { keys: '⌘2', action: 'Today' },
    { keys: '⌘3', action: 'Upcoming' },
    { keys: '⌘4', action: 'Someday' },
    { keys: '⌘5', action: 'Logbook' },
    { keys: '⌘,', action: 'Settings' },
    { keys: '⌘/', action: 'Toggle sidebar' },
  ]},
  { category: 'Tasks', items: [
    { keys: '⌘N', action: 'New task' },
    { keys: 'Space', action: 'New task (quick)' },
    { keys: '⇧⌘C', action: 'Complete task' },
    { keys: '⌫', action: 'Delete task' },
    { keys: '⌘D', action: 'Duplicate task' },
    { keys: '↑↓ / JK', action: 'Navigate tasks' },
    { keys: '⌘↑↓', action: 'Move task up/down' },
    { keys: '⇧↑↓', action: 'Extend selection' },
    { keys: '⌘A', action: 'Select all' },
    { keys: 'Enter', action: 'Open detail' },
  ]},
  { category: 'Scheduling', items: [
    { keys: '⌘T', action: 'Schedule Today' },
    { keys: '⌘E', action: 'Schedule Evening' },
    { keys: '⌘O', action: 'Schedule Someday' },
  ]},
  { category: 'Tools', items: [
    { keys: '⌘K', action: 'Command palette' },
    { keys: '⇧⌘O', action: 'Command palette' },
    { keys: '⌘F', action: 'Search' },
    { keys: '⇧⌘M', action: 'Move to project' },
    { keys: '⌘?', action: 'This help' },
    { keys: 'Escape', action: 'Close / Deselect' },
  ]},
];

export default function ShortcutsHelp() {
  const show = useStore($showShortcuts);

  useEffect(() => {
    if (!show) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        $showShortcuts.set(false);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [show]);

  if (!show) return null;

  return (
    <>
      <div className="cmd-backdrop open" onClick={() => $showShortcuts.set(false)} />
      <div
        style={{
          position: 'fixed',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          zIndex: 1000,
          width: 560,
          maxHeight: '80vh',
          overflowY: 'auto',
          background: 'var(--canvas-elevated)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-xl)',
          padding: 'var(--sp-5)',
        }}
      >
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 'var(--sp-4)',
        }}>
          <h2 style={{
            fontSize: 'var(--text-base)',
            fontWeight: 700,
            color: 'var(--ink-primary)',
            margin: 0,
          }}>
            Keyboard Shortcuts
          </h2>
          <button
            onClick={() => $showShortcuts.set(false)}
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--ink-tertiary)',
              cursor: 'pointer',
              fontSize: 'var(--text-sm)',
              padding: 'var(--sp-1) var(--sp-2)',
              borderRadius: 'var(--radius-sm)',
            }}
          >
            ✕
          </button>
        </div>

        <div style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 'var(--sp-5)',
        }}>
          {shortcuts.map(({ category, items }) => (
            <div key={category}>
              <div style={{
                fontSize: 'var(--text-xs)',
                fontWeight: 700,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                color: 'var(--ink-tertiary)',
                marginBottom: 'var(--sp-2)',
              }}>
                {category}
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--sp-1)' }}>
                {items.map(({ keys, action }) => (
                  <div
                    key={keys}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: 'var(--sp-3)',
                    }}
                  >
                    <span style={{
                      fontSize: 'var(--text-sm)',
                      color: 'var(--ink-secondary)',
                    }}>
                      {action}
                    </span>
                    <span style={{
                      fontFamily: "'SF Mono', 'Menlo', monospace",
                      fontSize: 'var(--text-xs)',
                      color: 'var(--ink-secondary)',
                      background: 'var(--canvas-sunken)',
                      borderRadius: 'var(--radius-sm)',
                      padding: '2px 6px',
                      whiteSpace: 'nowrap',
                      flexShrink: 0,
                    }}>
                      {keys}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
