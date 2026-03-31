import { useState } from 'react';
import { initialSync } from '../hooks/useTauri';
import { loadAll } from '../store';

interface InitialSyncDialogProps {
  open: boolean;
  onClose: () => void;
}

type SyncMode = 'fresh' | 'merge' | 'push';

export default function InitialSyncDialog({ open, onClose }: InitialSyncDialogProps) {
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSync = async (mode: SyncMode) => {
    setSyncing(true);
    setError(null);
    try {
      await initialSync(mode);
      await loadAll();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSyncing(false);
    }
  };

  const buttonStyle = (disabled: boolean): React.CSSProperties => ({
    display: 'block',
    width: '100%',
    padding: 'var(--sp-3) var(--sp-4)',
    marginBottom: 'var(--sp-2)',
    background: disabled ? 'var(--canvas-sunken)' : 'var(--canvas-elevated)',
    border: '1px solid var(--border-strong)',
    borderRadius: 'var(--radius-md)',
    color: disabled ? 'var(--ink-tertiary)' : 'var(--ink-primary)',
    fontFamily: 'inherit',
    fontSize: 'var(--text-sm)',
    fontWeight: 500,
    textAlign: 'left',
    cursor: disabled ? 'not-allowed' : 'pointer',
  });

  return (
    <div className={`cmd-backdrop${open ? ' open' : ''}`} onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className={`cmd-palette${open ? ' open' : ''}`} style={{ maxWidth: 400, padding: 'var(--sp-5)' }}>
        <h3 style={{ fontSize: 'var(--text-md)', fontWeight: 700, color: 'var(--ink-primary)', marginBottom: 'var(--sp-2)' }}>
          Initial Sync
        </h3>
        <p style={{ fontSize: 'var(--text-sm)', color: 'var(--ink-secondary)', marginBottom: 'var(--sp-4)', lineHeight: 'var(--leading-relaxed)' }}>
          Choose how to sync this device with the server.
        </p>

        {syncing ? (
          <div style={{ textAlign: 'center', padding: 'var(--sp-5)', color: 'var(--ink-secondary)', fontSize: 'var(--text-sm)' }}>
            Syncing...
          </div>
        ) : (
          <>
            <button style={buttonStyle(syncing)} disabled={syncing} onClick={() => handleSync('fresh')}>
              <div style={{ fontWeight: 600 }}>Fresh sync from server</div>
              <div style={{ fontSize: 'var(--text-xs)', color: 'var(--ink-tertiary)', marginTop: 2 }}>
                Replace local data with server data
              </div>
            </button>
            <button style={buttonStyle(syncing)} disabled={syncing} onClick={() => handleSync('merge')}>
              <div style={{ fontWeight: 600 }}>Merge with server</div>
              <div style={{ fontSize: 'var(--text-xs)', color: 'var(--ink-tertiary)', marginTop: 2 }}>
                Combine local and server data, server wins conflicts
              </div>
            </button>
            <button style={buttonStyle(syncing)} disabled={syncing} onClick={() => handleSync('push')}>
              <div style={{ fontWeight: 600 }}>Push local to server</div>
              <div style={{ fontSize: 'var(--text-xs)', color: 'var(--ink-tertiary)', marginTop: 2 }}>
                Upload local data to server, overwriting server state
              </div>
            </button>
          </>
        )}

        {error && (
          <div style={{ marginTop: 'var(--sp-3)', padding: 'var(--sp-2) var(--sp-3)', background: 'var(--canvas-sunken)', border: '1px solid var(--deadline-red)', borderRadius: 'var(--radius-md)', fontSize: 'var(--text-xs)', color: 'var(--deadline-red)' }}>
            {error}
          </div>
        )}

        {!syncing && (
          <button
            onClick={onClose}
            style={{ marginTop: 'var(--sp-3)', background: 'none', border: 'none', color: 'var(--ink-tertiary)', fontSize: 'var(--text-sm)', cursor: 'pointer', padding: 'var(--sp-1) 0' }}
          >
            Cancel
          </button>
        )}
      </div>
    </div>
  );
}
