import { useEffect, useRef } from 'react';
import type { ReactNode, RefObject } from 'react';

interface PopoverPanelProps {
  title?: string;
  className?: string;
  popoverRef?: RefObject<HTMLDivElement | null>;
  children: ReactNode;
}

export default function PopoverPanel({
  title,
  className,
  popoverRef,
  children,
}: PopoverPanelProps) {
  const localRef = useRef<HTMLDivElement | null>(null);

  // Move focus into the popover on open so keyboard users land inside it,
  // and restore it to the trigger when the popover unmounts.
  useEffect(() => {
    const el = popoverRef?.current ?? localRef.current;
    if (!el) return;
    const previouslyFocused = document.activeElement as HTMLElement | null;
    const target = el.querySelector<HTMLElement>(
      'input, textarea, button:not(:disabled), [tabindex]:not([tabindex="-1"])',
    );
    target?.focus();
    return () => {
      if (previouslyFocused && document.contains(previouslyFocused)) {
        previouslyFocused.focus();
      }
    };
    // Run once on mount — pickers unmount on close.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div
      ref={(node) => {
        localRef.current = node;
        if (popoverRef) popoverRef.current = node;
      }}
      role="dialog"
      aria-label={title}
      className={['ui-popover', className].filter(Boolean).join(' ')}
    >
      {title && (
        <>
          <div className="ui-popover-header">{title}</div>
          <div className="ui-popover-separator" />
        </>
      )}
      {children}
    </div>
  );
}

export type { PopoverPanelProps };
