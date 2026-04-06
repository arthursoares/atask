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
  return (
    <div
      ref={popoverRef}
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
