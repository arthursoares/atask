import { useEffect, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

interface DragOverlayProps {
  activeId: string | null;
  cursorX: number | null;
  cursorY: number | null;
  itemWidth: number | null;
  renderClone: (id: string) => ReactNode;
  /**
   * Pointer-to-item-origin offset captured at pickup (see
   * PointerReorderState.grabOffsetX/Y). Keeps the clone pinned under the
   * grab point instead of snapping its corner to the cursor.
   */
  grabOffsetX?: number;
  grabOffsetY?: number;
}

export default function DragOverlay({
  activeId,
  cursorX,
  cursorY,
  itemWidth,
  renderClone,
  grabOffsetX = 0,
  grabOffsetY = 0,
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

  // Things-style compact card: cap the clone width so carrying a task
  // across the window (e.g. onto a sidebar target) doesn't occlude the
  // drop target with a full-row-width slab. Clamp the grab anchor into
  // the capped card so the cursor always stays on the card.
  const cloneWidth = itemWidth != null ? Math.min(itemWidth, 340) : null;
  const effectiveOffsetX = cloneWidth != null
    ? Math.min(grabOffsetX, cloneWidth - 48)
    : grabOffsetX;

  useEffect(() => {
    if (ref.current) {
      const x = (cursorX ?? 0) - effectiveOffsetX;
      const y = (cursorY ?? 0) - grabOffsetY;
      ref.current.style.transform = `translate(${x}px, ${y}px)`;
    }
  }, [cursorX, cursorY, effectiveOffsetX, grabOffsetY]);

  if (!activeId || !ref.current) return null;

  const clone = renderClone(activeId);

  return createPortal(
    <div className="drag-overlay-lift" style={{ width: cloneWidth ?? 'auto' }}>
      {clone}
    </div>,
    ref.current
  );
}
