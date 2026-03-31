interface CheckboxCircleProps {
  checked: boolean;
  cancelled?: boolean;
  today?: boolean;
  onChange: () => void;
}

export default function CheckboxCircle({ checked, cancelled, today, onChange }: CheckboxCircleProps) {
  const classes = [
    'checkbox',
    today ? 'today' : '',
    checked ? 'checked' : '',
    cancelled ? 'cancelled' : '',
  ].filter(Boolean).join(' ');

  return (
    <div className={classes} onClick={onChange}>
      <svg viewBox="0 0 12 12">
        <polyline points="2.5 6 5 8.5 9.5 3.5" />
      </svg>
    </div>
  );
}
