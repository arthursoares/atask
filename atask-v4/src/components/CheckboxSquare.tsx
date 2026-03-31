interface CheckboxSquareProps {
  done: boolean;
  onChange: () => void;
}

export default function CheckboxSquare({ done, onChange }: CheckboxSquareProps) {
  return (
    <div className={`cl-check${done ? ' done' : ''}`} onClick={onChange}>
      {done && (
        <svg viewBox="0 0 12 12">
          <polyline points="2.5 6 5 8.5 9.5 3.5" />
        </svg>
      )}
    </div>
  );
}
