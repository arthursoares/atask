import { useStore } from '@nanostores/react';
import { $syncStatus } from '../store';

export default function SyncStatusIndicator() {
  const sync = useStore($syncStatus);

  if (sync.lastError) {
    return (
      <span
        title={`Sync error: ${sync.lastError}`}
        className="sync-status sync-status-error"
      >
        <svg viewBox="0 0 16 16" className="sync-status-icon sync-status-icon-error" fill="var(--deadline-red)" stroke="none">
          <polygon points="8 2 15 14 1 14" />
          <text x="8" y="12.5" textAnchor="middle" fill="white" fontSize="7" fontWeight="700" fontFamily="system-ui">!</text>
        </svg>
      </span>
    );
  }

  if (sync.pendingOpsCount > 0) {
    return (
      <span
        title={`${sync.pendingOpsCount} pending operation${sync.pendingOpsCount !== 1 ? 's' : ''}`}
        className="sync-status sync-status-pending"
      >
        <svg viewBox="0 0 16 16" className="sync-status-icon sync-status-icon-pending" fill="none" stroke="var(--ink-tertiary)" strokeWidth={1.5}>
          <line x1="8" y1="13" x2="8" y2="4" />
          <polyline points="4 7 8 3 12 7" />
        </svg>
        <span className="sync-status-count">{sync.pendingOpsCount}</span>
      </span>
    );
  }

  return (
    <span
      title={sync.lastSyncAt ? `Last synced ${new Date(sync.lastSyncAt).toLocaleTimeString()}` : 'Sync up to date'}
      className="sync-status sync-status-idle"
    >
      <span className="sync-status-dot" />
    </span>
  );
}
