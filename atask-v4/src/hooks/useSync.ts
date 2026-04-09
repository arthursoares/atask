import { useEffect } from 'react';
import { listen } from '@tauri-apps/api/event';
import { loadAll, $syncStatus, $activeView } from '../store';
import { setOnMutation } from '../store/mutations';
import { getSyncStatus, getSettings, triggerSync, type SyncStatus } from './useTauri';
import type { SyncPhase, SyncStatusState } from '../store/ui';

let syncDebounceTimer: ReturnType<typeof setTimeout> | null = null;

/**
 * Combine the raw Rust SyncStatus with the frontend phase state machine.
 * Callers are responsible for passing the correct enabled flag from
 * settings — this helper does not query it.
 */
function derivePhase(status: SyncStatus, enabled: boolean): SyncPhase {
  if (!enabled) return 'unconfigured';
  if (status.lastError) return 'error';
  // Only claim "synced" once we've actually observed a successful sync at
  // some point. A fresh install with no lastSyncAt and no pending ops
  // should stay in "unknown" until the first real sync completes.
  if (status.lastSyncAt) return 'synced';
  return 'unknown';
}

function mergeStatus(status: SyncStatus, enabled: boolean): SyncStatusState {
  return {
    phase: derivePhase(status, enabled),
    isSyncing: status.isSyncing,
    lastSyncAt: status.lastSyncAt,
    lastError: status.lastError,
    pendingOpsCount: status.pendingOpsCount,
  };
}

/**
 * Fetch the backend status AND settings, then merge them into a single
 * frontend-side status atom with a proper phase. Errors in either call
 * leave the phase at 'unknown' (not falsely 'synced') so the UI never
 * claims "all good" before knowing.
 */
async function refreshSyncStatus(): Promise<void> {
  const current = $syncStatus.get();
  // Signal that a check is in flight so the indicator can show a
  // transient "checking" state.
  $syncStatus.set({ ...current, phase: 'checking' });
  try {
    const [status, settings] = await Promise.all([getSyncStatus(), getSettings()]);
    $syncStatus.set(mergeStatus(status, settings.syncEnabled));
  } catch {
    // Both calls failed — fall back to unknown rather than pretending
    // anything about sync health.
    $syncStatus.set({
      ...current,
      phase: 'unknown',
      isSyncing: false,
    });
  }
}

/**
 * Trigger a sync cycle (flush outbound + pull deltas).
 * Debounced to 1s so rapid mutations don't spam the server.
 */
export function requestSync() {
  if (syncDebounceTimer) clearTimeout(syncDebounceTimer);
  syncDebounceTimer = setTimeout(async () => {
    try {
      await triggerSync();
    } catch {
      // Sync not configured or server unreachable — don't reload.
      // Local mutations already updated the store directly.
    }
    void refreshSyncStatus();
  }, 1000);
}

export default function useSync() {
  useEffect(() => {
    // Listen for store-changed events from Rust (after delta pull applied changes)
    const unlisten = listen('store-changed', () => {
      loadAll();
    });

    // Listen for sync-flushed events (update status indicator)
    const unlistenFlush = listen('sync-flushed', () => {
      void refreshSyncStatus();
    });

    // Sync on window focus
    const handleFocus = () => requestSync();
    window.addEventListener('focus', handleFocus);

    // 5-minute fallback poll for sync status
    const interval = setInterval(() => {
      void refreshSyncStatus();
    }, 300000);

    // Wire mutation hook — triggers sync after every store mutation
    setOnMutation(() => requestSync());

    // Initial status fetch
    void refreshSyncStatus();

    return () => {
      unlisten.then((f) => f());
      unlistenFlush.then((f) => f());
      window.removeEventListener('focus', handleFocus);
      clearInterval(interval);
      setOnMutation(() => {});
    };
  }, []);

  // Sync on view change
  useEffect(() => {
    const unsub = $activeView.subscribe(() => {
      requestSync();
    });
    return unsub;
  }, []);
}
