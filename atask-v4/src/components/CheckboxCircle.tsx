interface CheckboxCircleProps {
  checked: boolean;
  cancelled?: boolean;
  today?: boolean;
  onChange: () => void;
  ariaLabel?: string;
}

/**
 * Task-completion checkbox. Rendered as <div role="checkbox"> rather than a
 * real <input type="checkbox"> so the existing checked/today/cancelled CSS
 * states (which style the surrounding chip) keep working, but with full
 * keyboard + screen reader semantics: Tab-reachable, Enter/Space toggles,
 * aria-checked announced, and focus-visible ring handled by the
 * .checkbox:focus-visible rule in theme.css.
 */
export default function CheckboxCircle({
  checked,
  cancelled,
  today,
  onChange,
  ariaLabel = 'Toggle task completion',
}: CheckboxCircleProps) {
  const classes = [
    'checkbox',
    today ? 'today' : '',
    checked ? 'checked' : '',
    cancelled ? 'cancelled' : '',
  ].filter(Boolean).join(' ');

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      e.stopPropagation();
      onChange();
    }
  };

  const handleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    e.stopPropagation();
    onChange();
  };

  return (
    <div
      className={classes}
      role="checkbox"
      aria-checked={cancelled ? 'mixed' : checked}
      aria-label={ariaLabel}
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
    >
      <svg viewBox="0 0 12 12" aria-hidden="true">
        <polyline points="2.5 6 5 8.5 9.5 3.5" />
      </svg>
    </div>
  );
}
