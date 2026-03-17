/**
 * Toast Store
 * Manages ephemeral toast notifications
 */

import { useMemo } from 'react';
import { create } from 'zustand';

export type ToastType = 'success' | 'error' | 'info';

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  duration?: number; // milliseconds, undefined means no auto-dismiss
  dismissible?: boolean; // whether user can manually dismiss
}

interface ToastState {
  toasts: Toast[];
}

interface ToastActions {
  addToast: (toast: Omit<Toast, 'id'>) => string;
  removeToast: (id: string) => void;
  clearAll: () => void;
}

export type ToastStore = ToastState & ToastActions;

/**
 * Default toast configuration
 */
const DEFAULT_DURATION = 5000; // 5 seconds
const DEFAULT_DISMISSIBLE = true;

/**
 * Generate unique toast ID
 */
function generateToastId(): string {
  return `toast-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Track auto-dismiss timers to prevent memory leaks
 */
const dismissTimers = new Map<string, ReturnType<typeof setTimeout>>();

/**
 * Toast notification store
 */
export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],

  addToast: (toast) => {
    const id = generateToastId();
    const newToast: Toast = {
      id,
      type: toast.type,
      message: toast.message,
      duration: toast.duration ?? DEFAULT_DURATION,
      dismissible: toast.dismissible ?? DEFAULT_DISMISSIBLE,
    };

    set((state) => ({
      toasts: [...state.toasts, newToast],
    }));

    // Auto-dismiss if duration is specified
    if (newToast.duration !== undefined && newToast.duration > 0) {
      const timerId = setTimeout(() => {
        dismissTimers.delete(id);
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }));
      }, newToast.duration);
      dismissTimers.set(id, timerId);
    }

    return id;
  },

  removeToast: (id) => {
    // Cancel auto-dismiss timer if it exists
    const timerId = dismissTimers.get(id);
    if (timerId) {
      clearTimeout(timerId);
      dismissTimers.delete(id);
    }
    
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  clearAll: () => {
    // Clear all pending timers
    dismissTimers.forEach((timerId) => clearTimeout(timerId));
    dismissTimers.clear();
    
    set({ toasts: [] });
  },
}));

/**
 * Hook for toast notifications
 * Provides convenient methods for common toast types
 */
export function useToasts() {
  const addToast = useToastStore((state) => state.addToast);
  const removeToast = useToastStore((state) => state.removeToast);
  const clearAll = useToastStore((state) => state.clearAll);

  return useMemo(() => ({
    success: (message: string, duration?: number) =>
      addToast({ type: 'success', message, duration }),

    error: (message: string, duration?: number) =>
      addToast({ type: 'error', message, duration }),

    info: (message: string, duration?: number) =>
      addToast({ type: 'info', message, duration }),

    dismiss: removeToast,

    clearAll,

    // Advanced: add custom toast with full control
    custom: addToast,
  }), [addToast, removeToast, clearAll]);
}
