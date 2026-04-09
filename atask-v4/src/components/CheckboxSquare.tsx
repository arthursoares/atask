interface CheckboxSquareProps {
  done: boolean;
  onChange: () => void;
  ariaLabel?: string;
}

/**
 * Checklist item checkbox. Same a11y treatment as CheckboxCircle: exposed as
 * role="checkbox" with aria-checked and keyboard activation (Enter / Space).
 */
export default function CheckboxSquare({
  done,
  onChange,
  ariaLabel = 'Toggle checklist item',
}: CheckboxSquareProps) {
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
      className={`cl-check${done ? ' done' : ''}`}
      role="checkbox"
      aria-checked={done}
      aria-label={ariaLabel}
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
    >
      {done && (
        <svg viewBox="0 0 12 12" aria-hidden="true">
          <polyline points="2.5 6 5 8.5 9.5 3.5" />
        </svg>
      )}
    </div>
  );
}
