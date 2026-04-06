import { useState } from 'react';
import { initialSync } from '../hooks/useTauri';
import { loadAll } from '../store';
import { Button } from '../ui';

interface InitialSyncDialogProps {
  open: boolean;
  onClose: () => void;
}

type SyncMode = 'fresh' | 'merge' | 'push';

const syncModes: Array<{
  mode: SyncMode;
  title: string;
  description: string;
}> = [
  { mode: 'fresh', title: 'Fresh sync from server', description: 'Replace local data with server data' },
  { mode: 'merge', title: 'Merge with server', description: 'Combine local and server data, server wins conflicts' },
  { mode: 'push', title: 'Push local to server', description: 'Upload local data to server, overwriting server state' },
];

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

  return (
    <div className={`cmd-backdrop${open ? ' open' : ''}`} onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className={`cmd-palette initial-sync-dialog${open ? ' open' : ''}`}>
        <h3 className="initial-sync-title">Initial Sync</h3>
        <p className="initial-sync-copy">Choose how to sync this device with the server.</p>

        {syncing ? (
          <div className="initial-sync-loading">Syncing...</div>
        ) : (
          <div className="initial-sync-options">
            {syncModes.map((option) => (
              <button
                key={option.mode}
                type="button"
                className={`initial-sync-option${syncing ? ' is-disabled' : ''}`}
                disabled={syncing}
                onClick={() => handleSync(option.mode)}
              >
                <div className="initial-sync-option-title">{option.title}</div>
                <div className="initial-sync-option-copy">{option.description}</div>
              </button>
            ))}
          </div>
        )}

        {error && (
          <div className="initial-sync-error">{error}</div>
        )}

        {!syncing && (
          <Button variant="ghost" className="initial-sync-cancel" onClick={onClose}>
            Cancel
          </Button>
        )}
      </div>
    </div>
  );
}
