import { useEffect, useRef, useState } from 'react';
import MenuList, { type MenuListItem } from '../ui/MenuList';

export type MenuItem =
  | (MenuListItem & { separator?: false })
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
    document.addEventListener('mousedown', handler, true);
    document.addEventListener('click', handler, true);
    return () => {
      document.removeEventListener('mousedown', handler, true);
      document.removeEventListener('click', handler, true);
    };
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
      className="context-menu-shell"
      style={{ left: adjustedPos.x, top: adjustedPos.y }}
    >
      <MenuList
        items={items}
        activeIndex={activeIndex}
        onItemHover={setActiveIndex}
        onItemLeave={() => setActiveIndex(-1)}
        onItemClick={(item) => handleItemClick(item as MenuItem)}
      />
    </div>
  );
}
