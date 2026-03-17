/**
 * ToastContainer Component
 * Displays toast notifications with accessibility support
 */

import { useToastStore, type Toast, type ToastType } from '../stores/toastStore';

/**
 * Get icon for toast type
 */
function getToastIcon(type: ToastType): string {
  switch (type) {
    case 'success':
      return '✓';
    case 'error':
      return '✕';
    case 'info':
      return 'ℹ';
  }
}

/**
 * Get Tailwind classes for toast type
 */
function getToastClasses(type: ToastType): string {
  switch (type) {
    case 'success':
      return 'bg-status-success text-black border border-status-success';
    case 'error':
      return 'bg-status-error text-white border border-status-error';
    case 'info':
      return 'bg-neon-cyan text-black border border-neon-cyan';
  }
}

/**
 * Individual toast notification
 */
function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: (id: string) => void }) {
  const toastClasses = getToastClasses(toast.type);

  return (
    <div
      role="status"
      aria-live="polite"
      aria-atomic="true"
      className={`
        flex items-center gap-3 p-4 rounded-none shadow-none
        min-w-[300px] max-w-[500px]
        animate-slide-in
        ${toastClasses}
      `.trim()}
    >
      <div className="text-xl font-bold flex-shrink-0" aria-hidden="true">
        {getToastIcon(toast.type)}
      </div>
      <div className="flex-1 text-sm leading-5">{toast.message}</div>
      {toast.dismissible && (
        <button
          onClick={() => onDismiss(toast.id)}
          aria-label="Dismiss notification"
          className="
            bg-transparent border-0 text-current cursor-pointer
            p-1 text-xl leading-none opacity-80 hover:opacity-100
            flex-shrink-0 transition-none
            focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
          "
        >
          ×
        </button>
      )}
    </div>
  );
}

/**
 * Toast container component
 * Renders all active toasts in a fixed position
 */
export function ToastContainer() {
  const toasts = useToastStore((state) => state.toasts);
  const removeToast = useToastStore((state) => state.removeToast);

  if (toasts.length === 0) {
    return null;
  }

  return (
    <div
      aria-label="Notifications"
      role="region"
      aria-live="polite"
      className="fixed top-4 right-4 flex flex-col gap-3 z-[9999] pointer-events-none"
    >
      {toasts.map((toast) => (
        <div key={toast.id} className="pointer-events-auto">
          <ToastItem toast={toast} onDismiss={removeToast} />
        </div>
      ))}
    </div>
  );
}
