import { useEffect } from 'react';
import { useStore } from '@nanostores/react';
import { $showShortcuts } from '../store';
import { Button } from '../ui';

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
      <div className="shortcuts-help">
        <div className="shortcuts-help-header">
          <h2 className="shortcuts-help-title">Keyboard Shortcuts</h2>
          <Button
            variant="ghost"
            size="sm"
            className="shortcuts-help-close"
            onClick={() => $showShortcuts.set(false)}
          >
            ✕
          </Button>
        </div>

        <div className="shortcuts-help-grid">
          {shortcuts.map(({ category, items }) => (
            <div key={category}>
              <div className="shortcuts-help-category">{category}</div>
              <div className="shortcuts-help-list">
                {items.map(({ keys, action }) => (
                  <div key={keys} className="shortcuts-help-item">
                    <span className="shortcuts-help-action">
                      {action}
                    </span>
                    <span className="shortcuts-help-keys">
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
