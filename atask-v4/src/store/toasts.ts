import { atom } from 'nanostores';

export interface Toast {
  id: number;
  message: string;
  kind: 'info' | 'success' | 'error';
  /** Optional action button (e.g. "Undo"). */
  actionLabel?: string;
  onAction?: () => void;
  /** Auto-dismiss delay in ms. */
  duration: number;
}

export const $toasts = atom<Toast[]>([]);

let nextToastId = 1;
const dismissTimers = new Map<number, number>();

export interface ShowToastOptions {
  kind?: Toast['kind'];
  actionLabel?: string;
  onAction?: () => void;
  duration?: number;
}

export function showToast(message: string, options: ShowToastOptions = {}): number {
  const id = nextToastId++;
  const toast: Toast = {
    id,
    message,
    kind: options.kind ?? 'info',
    actionLabel: options.actionLabel,
    onAction: options.onAction,
    duration: options.duration ?? 5000,
  };
  $toasts.set([...$toasts.get(), toast]);

  const timer = window.setTimeout(() => dismissToast(id), toast.duration);
  dismissTimers.set(id, timer);
  return id;
}

export function dismissToast(id: number): void {
  const timer = dismissTimers.get(id);
  if (timer !== undefined) {
    window.clearTimeout(timer);
    dismissTimers.delete(id);
  }
  $toasts.set($toasts.get().filter((t) => t.id !== id));
}

/** Convenience wrapper for error feedback. Errors linger a bit longer. */
export function showErrorToast(message: string): number {
  return showToast(message, { kind: 'error', duration: 8000 });
}
