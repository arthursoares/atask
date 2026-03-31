import { useEffect, useRef, useState, type ReactNode } from 'react';

export type MenuItem =
  | {
      label: string;
      icon?: ReactNode;
      shortcut?: string;
      danger?: boolean;
      disabled?: boolean;
      separator?: false;
      onClick?: () => void;
    }
  | { separator: true };

interface ContextMenuProps {
  items: MenuItem[];
  position: { x: number; y: number };
  onClose: () => void;
}

export default function ContextMenu({ items, position, onClose }: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [activeIndex, setActiveIndex] = useState<number>(-1);

  // Compute actionable indices (non-separator, non-disabled)
  const actionableIndices = items
    .map((item, i) => ({ item, i }))
    .filter(({ item }) => !('separator' in item) && !('disabled' in item && item.disabled))
    .map(({ i }) => i);

  // Adjust position to stay within viewport
  const [adjustedPos, setAdjustedPos] = useState(position);

  useEffect(() => {
    if (!menuRef.current) return;
    const rect = menuRef.current.getBoundingClientRect();
    let { x, y } = position;
    if (x + rect.width > window.innerWidth) x = window.innerWidth - rect.width - 8;
    if (y + rect.height > window.innerHeight) y = window.innerHeight - rect.height - 8;
    if (x < 8) x = 8;
    if (y < 8) y = 8;
    setAdjustedPos({ x, y });
  }, [position]);

  // Click outside to close
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [onClose]);

  // Keyboard navigation
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIndex(prev => {
          const currentPos = actionableIndices.indexOf(prev);
          const next = currentPos < actionableIndices.length - 1 ? currentPos + 1 : 0;
          return actionableIndices[next];
        });
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIndex(prev => {
          const currentPos = actionableIndices.indexOf(prev);
          const next = currentPos > 0 ? currentPos - 1 : actionableIndices.length - 1;
          return actionableIndices[next];
        });
        return;
      }
      if (e.key === 'Enter' && activeIndex >= 0) {
        const item = items[activeIndex];
        if (item && !('separator' in item) && !item.disabled) {
          item.onClick?.();
          onClose();
        }
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [activeIndex, actionableIndices, items, onClose]);

  const handleItemClick = (item: MenuItem) => {
    if ('separator' in item) return;
    if (item.disabled) return;
    item.onClick?.();
    onClose();
  };

  return (
    <div
      ref={menuRef}
      style={{
        position: 'fixed',
        left: adjustedPos.x,
        top: adjustedPos.y,
        zIndex: 200,
        background: 'var(--canvas-elevated)',
        border: '1px solid var(--border-strong)',
        borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-popover)',
        overflow: 'hidden',
        minWidth: 200,
        padding: 'var(--sp-1) 0',
      }}
    >
      {items.map((item, i) => {
        if ('separator' in item) {
          return (
            <div
              key={i}
              style={{
                height: 1,
                background: 'var(--separator)',
                margin: 'var(--sp-1) 0',
              }}
            />
          );
        }

        const isActive = activeIndex === i;

        return (
          <div
            key={i}
            onMouseEnter={() => setActiveIndex(i)}
            onMouseLeave={() => setActiveIndex(-1)}
            onClick={() => handleItemClick(item)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--sp-2)',
              padding: 'var(--sp-1) var(--sp-3)',
              cursor: item.disabled ? 'default' : 'pointer',
              color: item.danger ? 'var(--deadline-red)' : 'var(--ink-primary)',
              opacity: item.disabled ? 0.4 : 1,
              pointerEvents: item.disabled ? 'none' : 'auto',
              background: isActive ? 'var(--sidebar-hover)' : 'transparent',
              fontSize: 'var(--text-sm)',
            }}
          >
            {item.icon && (
              <span style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
                {item.icon}
              </span>
            )}
            <span style={{ flex: 1 }}>{item.label}</span>
            {item.shortcut && (
              <span style={{ color: 'var(--ink-tertiary)', fontSize: 'var(--text-xs)' }}>
                {item.shortcut}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}
