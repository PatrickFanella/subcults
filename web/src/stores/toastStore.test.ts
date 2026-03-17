/**
 * Toast Store Tests
 * Validates toast notification management
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useToastStore, useToasts } from './toastStore';

describe('toastStore', () => {
  beforeEach(() => {
    // Clear toasts before each test
    useToastStore.setState({ toasts: [] });
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('addToast', () => {
    it('adds a toast to the store', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({
          type: 'success',
          message: 'Test message',
        });
      });

      expect(result.current.toasts).toHaveLength(1);
      expect(result.current.toasts[0]).toMatchObject({
        type: 'success',
        message: 'Test message',
      });
      expect(result.current.toasts[0].id).toBeDefined();
    });

    it('generates unique IDs for each toast', () => {
      const { result } = renderHook(() => useToastStore());

      let id1: string, id2: string;

      act(() => {
        id1 = result.current.addToast({ type: 'info', message: 'First' });
        id2 = result.current.addToast({ type: 'info', message: 'Second' });
      });

      expect(id1).not.toBe(id2);
      expect(result.current.toasts).toHaveLength(2);
    });

    it('applies default duration and dismissible values', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({
          type: 'success',
          message: 'Default settings',
        });
      });

      expect(result.current.toasts[0].duration).toBe(5000);
      expect(result.current.toasts[0].dismissible).toBe(true);
    });

    it('respects custom duration and dismissible values', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({
          type: 'error',
          message: 'Custom settings',
          duration: 10000,
          dismissible: false,
        });
      });

      expect(result.current.toasts[0].duration).toBe(10000);
      expect(result.current.toasts[0].dismissible).toBe(false);
    });

    it('auto-dismisses toast after duration', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({
          type: 'info',
          message: 'Auto dismiss',
          duration: 1000,
        });
      });

      expect(result.current.toasts).toHaveLength(1);

      // Fast-forward time
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      expect(result.current.toasts).toHaveLength(0);
    });

    it('does not auto-dismiss when duration is 0', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({
          type: 'info',
          message: 'No auto dismiss',
          duration: 0,
        });
      });

      expect(result.current.toasts).toHaveLength(1);

      // Fast-forward time
      act(() => {
        vi.advanceTimersByTime(10000);
      });

      expect(result.current.toasts).toHaveLength(1);
    });
  });

  describe('removeToast', () => {
    it('removes specific toast by ID', () => {
      const { result } = renderHook(() => useToastStore());

      let id1: string, id2: string;

      act(() => {
        id1 = result.current.addToast({ type: 'info', message: 'First' });
        id2 = result.current.addToast({ type: 'info', message: 'Second' });
      });

      expect(result.current.toasts).toHaveLength(2);

      act(() => {
        result.current.removeToast(id1);
      });

      expect(result.current.toasts).toHaveLength(1);
      expect(result.current.toasts[0].id).toBe(id2);
    });

    it('does nothing if toast ID does not exist', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({ type: 'info', message: 'Test' });
      });

      expect(result.current.toasts).toHaveLength(1);

      act(() => {
        result.current.removeToast('non-existent-id');
      });

      expect(result.current.toasts).toHaveLength(1);
    });
  });

  describe('clearAll', () => {
    it('removes all toasts', () => {
      const { result } = renderHook(() => useToastStore());

      act(() => {
        result.current.addToast({ type: 'info', message: 'First' });
        result.current.addToast({ type: 'info', message: 'Second' });
        result.current.addToast({ type: 'info', message: 'Third' });
      });

      expect(result.current.toasts).toHaveLength(3);

      act(() => {
        result.current.clearAll();
      });

      expect(result.current.toasts).toHaveLength(0);
    });
  });

  describe('useToasts hook', () => {
    it('provides success method', () => {
      const { result } = renderHook(() => useToasts());

      act(() => {
        result.current.success('Success message');
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('success');
      expect(toasts[0].message).toBe('Success message');
    });

    it('provides error method', () => {
      const { result } = renderHook(() => useToasts());

      act(() => {
        result.current.error('Error message');
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('error');
      expect(toasts[0].message).toBe('Error message');
    });

    it('provides info method', () => {
      const { result } = renderHook(() => useToasts());

      act(() => {
        result.current.info('Info message');
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('info');
      expect(toasts[0].message).toBe('Info message');
    });

    it('allows custom duration in convenience methods', () => {
      const { result } = renderHook(() => useToasts());

      act(() => {
        result.current.success('Custom duration', 3000);
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts[0].duration).toBe(3000);
    });

    it('provides dismiss method', () => {
      const { result } = renderHook(() => useToasts());

      let id: string;

      act(() => {
        id = result.current.success('To be dismissed');
      });

      expect(useToastStore.getState().toasts).toHaveLength(1);

      act(() => {
        result.current.dismiss(id);
      });

      expect(useToastStore.getState().toasts).toHaveLength(0);
    });

    it('provides clearAll method', () => {
      const { result } = renderHook(() => useToasts());

      act(() => {
        result.current.success('First');
        result.current.error('Second');
        result.current.info('Third');
      });

      expect(useToastStore.getState().toasts).toHaveLength(3);

      act(() => {
        result.current.clearAll();
      });

      expect(useToastStore.getState().toasts).toHaveLength(0);
    });

    it('provides custom method for advanced usage', () => {
      const { result } = renderHook(() => useToasts());

      act(() => {
        result.current.custom({
          type: 'error',
          message: 'Custom toast',
          duration: 0,
          dismissible: false,
        });
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts[0].type).toBe('error');
      expect(toasts[0].message).toBe('Custom toast');
      expect(toasts[0].duration).toBe(0);
      expect(toasts[0].dismissible).toBe(false);
    });
  });
});
