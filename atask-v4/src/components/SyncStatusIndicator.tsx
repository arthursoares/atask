import { useStore } from '@nanostores/react';
import { $syncStatus } from '../store';

/**
 * Sync status indicator. Reads the phase-based atom and renders one of
 * five visual states so users never see a false "all good" signal before
 * the app knows it's actually synced.
 *
 *   unknown       muted "?" — no confirmed sync yet
 *   checking      spinner dot — status fetch in flight
 *   unconfigured  muted dash — sync is disabled; points to Settings
 *   synced        green dot — successful last sync
 *   error         red warning
 *   (pending)     upload arrow + count (any phase can have pending ops)
 */
export default function SyncStatusIndicator() {
  const sync = useStore($syncStatus);

  // Error takes precedence over everything else so users immediately see
  // when something is actively wrong.
  if (sync.phase === 'error' || sync.lastError) {
    return (
      <span
        title={`Sync error: ${sync.lastError ?? 'unknown'}`}
        className="sync-status sync-status-error"
      >
        <svg viewBox="0 0 16 16" className="sync-status-icon sync-status-icon-error" fill="var(--deadline-red)" stroke="none">
          <polygon points="8 2 15 14 1 14" />
          <text x="8" y="12.5" textAnchor="middle" fill="white" fontSize="7" fontWeight="700" fontFamily="system-ui">!</text>
        </svg>
      </span>
    );
  }

  // Pending ops override the idle/unknown indicators because they
  // represent concrete work that hasn't been flushed yet.
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

  if (sync.phase === 'unconfigured') {
    return (
      <span title="Sync not configured — enable in Settings" className="sync-status sync-status-unconfigured">
        <span className="sync-status-dot sync-status-dot-unconfigured" aria-hidden="true">—</span>
      </span>
    );
  }

  if (sync.phase === 'unknown') {
    return (
      <span title="Checking sync status…" className="sync-status sync-status-unknown">
        <span className="sync-status-dot sync-status-dot-unknown" aria-hidden="true">?</span>
      </span>
    );
  }

  if (sync.phase === 'checking' || sync.isSyncing) {
    return (
      <span title="Syncing…" className="sync-status sync-status-checking">
        <span className="sync-status-dot sync-status-dot-checking" aria-hidden="true" />
      </span>
    );
  }

  // phase === 'synced'
  return (
    <span
      title={sync.lastSyncAt ? `Last synced ${new Date(sync.lastSyncAt).toLocaleTimeString()}` : 'Sync up to date'}
      className="sync-status sync-status-idle"
    >
      <span className="sync-status-dot" />
    </span>
  );
}
