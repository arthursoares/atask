import type { ReactNode } from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
type ButtonSize = 'sm' | 'default' | 'lg';

interface ButtonProps {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  onClick?: () => void;
  disabled?: boolean;
}

export default function Button({
  children,
  variant = 'secondary',
  size = 'default',
  onClick,
  disabled,
}: ButtonProps) {
  const classes = [
    'btn',
    `btn-${variant}`,
    size !== 'default' ? `btn-${size}` : '',
  ].filter(Boolean).join(' ');

  return (
    <button className={classes} onClick={onClick} disabled={disabled}>
      {children}
    </button>
  );
}
