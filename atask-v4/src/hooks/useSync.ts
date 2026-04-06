import { useEffect } from 'react';
import { listen } from '@tauri-apps/api/event';
import { loadAll, $syncStatus, $activeView } from '../store';
import { setOnMutation } from '../store/mutations';
import { getSyncStatus, triggerSync } from './useTauri';

let syncDebounceTimer: ReturnType<typeof setTimeout> | null = null;

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
    getSyncStatus().then((s) => $syncStatus.set(s)).catch(() => {});
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
      getSyncStatus().then((s) => $syncStatus.set(s)).catch(() => {});
    });

    // Sync on window focus
    const handleFocus = () => requestSync();
    window.addEventListener('focus', handleFocus);

    // 5-minute fallback poll for sync status
    const interval = setInterval(() => {
      getSyncStatus().then((s) => $syncStatus.set(s)).catch(() => {});
    }, 300000);

    // Wire mutation hook — triggers sync after every store mutation
    setOnMutation(() => requestSync());

    // Initial status fetch
    getSyncStatus().then((s) => $syncStatus.set(s)).catch(() => {});

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
