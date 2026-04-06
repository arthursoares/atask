interface SectionHeaderProps {
  title: string;
  count?: number;
  muted?: boolean;
  collapsible?: boolean;
  collapsed?: boolean;
  onToggle?: () => void;
}

export default function SectionHeader({
  title,
  count,
  muted,
  collapsible,
  collapsed,
  onToggle,
}: SectionHeaderProps) {
  return (
    <div className="section-header" onClick={onToggle}>
      {collapsible && (
        <svg
          viewBox="0 0 16 16"
          className={`section-header-chevron${collapsed ? ' collapsed' : ''}`}
        >
          <polyline
            points="5 3 11 8 5 13"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          />
        </svg>
      )}
      <span className={`section-header-title${muted ? ' muted' : ''}`}>{title}</span>
      {count !== undefined && (
        <span className="section-header-count">{count}</span>
      )}
      <div className="section-header-line" />
    </div>
  );
}

export type { SectionHeaderProps };
