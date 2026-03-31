type TagVariant = 'default' | 'accent' | 'today' | 'deadline' | 'agent' | 'success' | 'someday' | 'cancelled';

interface TagPillProps {
  label: string;
  variant?: TagVariant;
  onRemove?: () => void;
}

export default function TagPill({ label, variant = 'default', onRemove }: TagPillProps) {
  return (
    <span className={`tag tag-${variant}`}>
      {label}
      {onRemove && (
        <span onClick={onRemove} className="remove">×</span>
      )}
    </span>
  );
}
