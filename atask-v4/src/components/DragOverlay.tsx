import { useEffect, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

interface DragOverlayProps {
  activeId: string | null;
  cursorX: number | null;
  cursorY: number | null;
  itemWidth: number | null;
  renderClone: (id: string) => ReactNode;
}

export default function DragOverlay({
  activeId,
  cursorX,
  cursorY,
  itemWidth,
  renderClone,
}: DragOverlayProps) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    ref.current = document.createElement('div');
    ref.current.style.position = 'fixed';
    ref.current.style.left = '0';
    ref.current.style.top = '0';
    ref.current.style.pointerEvents = 'none';
    ref.current.style.zIndex = '9999';
    document.body.appendChild(ref.current);

    return () => {
      if (ref.current) {
        document.body.removeChild(ref.current);
      }
    };
  }, []);

  useEffect(() => {
    if (ref.current) {
      ref.current.style.transform = `translate(${cursorX ?? 0}px, ${(cursorY ?? 0) - 20}px)`;
    }
  }, [cursorX, cursorY]);

  if (!activeId || !ref.current) return null;

  const clone = renderClone(activeId);

  return createPortal(
    <div style={{ width: itemWidth ?? 'auto' }}>
      {clone}
    </div>,
    ref.current
  );
}
