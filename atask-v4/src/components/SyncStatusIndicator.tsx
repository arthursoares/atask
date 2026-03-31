import { useStore } from '@nanostores/react';
import { $syncStatus } from '../store';

export default function SyncStatusIndicator() {
  const sync = useStore($syncStatus);

  if (sync.lastError) {
    return (
      <span
        title={`Sync error: ${sync.lastError}`}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          fontSize: 'var(--text-xs)',
          color: 'var(--deadline-red)',
          cursor: 'default',
        }}
      >
        <svg viewBox="0 0 16 16" width={14} height={14} fill="var(--deadline-red)" stroke="none">
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
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          fontSize: 'var(--text-xs)',
          color: 'var(--ink-tertiary)',
          cursor: 'default',
        }}
      >
        <svg viewBox="0 0 16 16" width={12} height={12} fill="none" stroke="var(--ink-tertiary)" strokeWidth={1.5}>
          <line x1="8" y1="13" x2="8" y2="4" />
          <polyline points="4 7 8 3 12 7" />
        </svg>
        <span style={{ color: 'var(--ink-secondary)' }}>{sync.pendingOpsCount}</span>
      </span>
    );
  }

  // Synced / idle — green dot
  return (
    <span
      title={sync.lastSyncAt ? `Last synced ${new Date(sync.lastSyncAt).toLocaleTimeString()}` : 'Sync up to date'}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        cursor: 'default',
      }}
    >
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: '50%',
          background: 'var(--success)',
          display: 'inline-block',
        }}
      />
    </span>
  );
}
