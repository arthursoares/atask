export function InboxIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <rect x="2" y="3" width="12" height="10" rx="2" />
      <polyline points="2 8 6 8 7 10 9 10 10 8 14 8" />
    </svg>
  );
}

export function TodayIcon() {
  return (
    <svg viewBox="0 0 16 16" fill="var(--today-star)" stroke="none">
      <polygon points="8 2 9.8 5.6 14 6.2 11 9 11.8 13 8 11.2 4.2 13 5 9 2 6.2 6.2 5.6" />
    </svg>
  );
}

export function UpcomingIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <rect x="2" y="3" width="12" height="11" rx="2" />
      <line x1="2" y1="7" x2="14" y2="7" />
      <line x1="5" y1="1" x2="5" y2="4" />
      <line x1="11" y1="1" x2="11" y2="4" />
    </svg>
  );
}

export function SomedayIcon() {
  return (
    <svg viewBox="0 0 16 16" stroke="var(--someday-tint)">
      <circle cx="8" cy="8" r="5.5" />
      <line x1="8" y1="5" x2="8" y2="8" />
      <line x1="8" y1="8" x2="10.5" y2="10" />
    </svg>
  );
}

export function LogbookIcon() {
  return (
    <svg viewBox="0 0 16 16">
      <path d="M4 2h8l1 4-5 3-5-3z" />
      <path d="M3 6v6c0 1 2 2 5 2s5-1 5-2V6" />
    </svg>
  );
}

export function SettingsIcon() {
  return (
    <svg viewBox="0 0 16 16" style={{ width: 16, height: 16 }}>
      <circle cx="8" cy="8" r="2.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <path
        d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.2 3.2l1.4 1.4M11.4 11.4l1.4 1.4M3.2 12.8l1.4-1.4M11.4 4.6l1.4-1.4"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
    </svg>
  );
}
