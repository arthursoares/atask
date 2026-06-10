import DropSlot from './DropSlot';

interface DropGapProps {
  /** A drag that could drop into this list is in progress. */
  active: boolean;
  /** This gap is the current insertion point. */
  open: boolean;
  edge?: 'top' | 'bottom' | null;
}

/**
 * Persistent insertion gap between rows. Always mounted (collapsed to
 * 0px) so the gap can animate both open AND closed as the drop index
 * moves — an unmounting indicator can only ever pop shut.
 *
 * DOM contract (the WDIO e2e suite counts these): every gap carries
 * `task-drop-zone` while a drag is active (n+1 of them), and exactly
 * the open one contains a `task-drop-slot` indicator line.
 */
export default function DropGap({ active, open, edge }: DropGapProps) {
  const edgeClass = edge === 'top'
    ? ' task-drop-zone-edge-top'
    : edge === 'bottom'
      ? ' task-drop-zone-edge-bottom'
      : '';

  const className = [
    'task-drop-gap',
    active ? ` task-drop-zone${edgeClass}` : '',
    open ? ' task-drop-gap-open' : '',
  ].join('');

  return (
    <div className={className} aria-hidden="true">
      {open && <DropSlot />}
    </div>
  );
}
