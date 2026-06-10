import { useStore } from '@nanostores/react';
import { $toasts, dismissToast } from '../store/toasts';

/**
 * Renders the toast stack bottom-right, above everything else. Toasts
 * carry mutation feedback (errors, delete-undo) so the stack is a
 * polite live region for screen readers.
 */
export default function ToastHost() {
  const toasts = useStore($toasts);

  if (toasts.length === 0) return null;

  return (
    <div className="toast-host" role="status" aria-live="polite">
      {toasts.map((toast) => (
        <div key={toast.id} className={`toast toast-${toast.kind}`}>
          <span className="toast-message">{toast.message}</span>
          {toast.actionLabel && (
            <button
              className="toast-action"
              onClick={() => {
                toast.onAction?.();
                dismissToast(toast.id);
              }}
            >
              {toast.actionLabel}
            </button>
          )}
          <button
            className="toast-dismiss"
            aria-label="Dismiss notification"
            onClick={() => dismissToast(toast.id)}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
