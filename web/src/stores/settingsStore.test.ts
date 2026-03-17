/**
 * Settings Store Tests
 * Validates user preferences and privacy settings management
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { 
  useSettingsStore, 
  useTelemetryOptOut, 
  useSessionReplayOptIn,
  useSettingsActions 
} from './settingsStore';

describe('settingsStore', () => {
  beforeEach(() => {
    // Clear localStorage
    localStorage.clear();
    // Reset store to initial state
    useSettingsStore.setState({ 
      telemetryOptOut: false,
      sessionReplayOptIn: false,
    });
  });

  afterEach(() => {
    localStorage.clear();
  });

  describe('initialization', () => {
    it('starts with default settings when localStorage is empty', () => {
      const { result } = renderHook(() => useSettingsStore());

      expect(result.current.telemetryOptOut).toBe(false);
      expect(result.current.sessionReplayOptIn).toBe(false);
    });

    it('loads settings from localStorage on initialization', () => {
      localStorage.setItem('subcults-settings', JSON.stringify({ 
        telemetryOptOut: true,
        sessionReplayOptIn: true,
      }));
      
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.initializeSettings();
      });

      expect(result.current.telemetryOptOut).toBe(true);
      expect(result.current.sessionReplayOptIn).toBe(true);
    });

    it('falls back to defaults when localStorage data is corrupted', () => {
      localStorage.setItem('subcults-settings', 'invalid json');
      
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.initializeSettings();
      });

      expect(result.current.telemetryOptOut).toBe(false);
      expect(result.current.sessionReplayOptIn).toBe(false);
    });

    it('handles missing telemetryOptOut field in stored data', () => {
      localStorage.setItem('subcults-settings', JSON.stringify({}));
      
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.initializeSettings();
      });

      expect(result.current.telemetryOptOut).toBe(false);
    });
  });

  describe('setSessionReplayOptIn', () => {
    it('updates opt-in state', () => {
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.setSessionReplayOptIn(true);
      });

      expect(result.current.sessionReplayOptIn).toBe(true);
    });

    it('persists opt-in to localStorage', () => {
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.setSessionReplayOptIn(true);
      });

      const stored = localStorage.getItem('subcults-settings');
      expect(stored).toBeTruthy();
      const parsed = JSON.parse(stored!);
      expect(parsed.sessionReplayOptIn).toBe(true);
    });

    it('can toggle opt-in multiple times', () => {
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.setSessionReplayOptIn(true);
      });
      expect(result.current.sessionReplayOptIn).toBe(true);

      act(() => {
        result.current.setSessionReplayOptIn(false);
      });
      expect(result.current.sessionReplayOptIn).toBe(false);

      act(() => {
        result.current.setSessionReplayOptIn(true);
      });
      expect(result.current.sessionReplayOptIn).toBe(true);
    });

    it('defaults to false (opt-out) for privacy', () => {
      const { result } = renderHook(() => useSettingsStore());
      
      expect(result.current.sessionReplayOptIn).toBe(false);
    });
  });

  describe('setTelemetryOptOut', () => {
    it('updates opt-out state', () => {
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.setTelemetryOptOut(true);
      });

      expect(result.current.telemetryOptOut).toBe(true);
    });

    it('persists opt-out to localStorage', () => {
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.setTelemetryOptOut(true);
      });

      const stored = localStorage.getItem('subcults-settings');
      expect(stored).toBeTruthy();
      const parsed = JSON.parse(stored!);
      expect(parsed.telemetryOptOut).toBe(true);
    });

    it('can toggle opt-out multiple times', () => {
      const { result } = renderHook(() => useSettingsStore());

      act(() => {
        result.current.setTelemetryOptOut(true);
      });
      expect(result.current.telemetryOptOut).toBe(true);

      act(() => {
        result.current.setTelemetryOptOut(false);
      });
      expect(result.current.telemetryOptOut).toBe(false);

      act(() => {
        result.current.setTelemetryOptOut(true);
      });
      expect(result.current.telemetryOptOut).toBe(true);
    });
  });

  describe('useSessionReplayOptIn hook', () => {
    it('returns current opt-in state', () => {
      useSettingsStore.setState({ sessionReplayOptIn: true });
      const { result } = renderHook(() => useSessionReplayOptIn());

      expect(result.current).toBe(true);
    });

    it('updates when opt-in state changes', () => {
      const { result } = renderHook(() => useSessionReplayOptIn());

      expect(result.current).toBe(false);

      act(() => {
        useSettingsStore.getState().setSessionReplayOptIn(true);
      });

      expect(result.current).toBe(true);
    });
  });

  describe('useTelemetryOptOut hook', () => {
    it('returns current opt-out state', () => {
      useSettingsStore.setState({ telemetryOptOut: true });
      const { result } = renderHook(() => useTelemetryOptOut());

      expect(result.current).toBe(true);
    });

    it('updates when opt-out state changes', () => {
      const { result } = renderHook(() => useTelemetryOptOut());

      expect(result.current).toBe(false);

      act(() => {
        useSettingsStore.getState().setTelemetryOptOut(true);
      });

      expect(result.current).toBe(true);
    });
  });

  describe('useSettingsActions hook', () => {
    it('returns stable action references', () => {
      const { result, rerender } = renderHook(() => useSettingsActions());
      const firstRender = result.current;

      rerender();
      const secondRender = result.current;

      // Actions should maintain referential equality
      expect(firstRender.setTelemetryOptOut).toBe(secondRender.setTelemetryOptOut);
      expect(firstRender.setSessionReplayOptIn).toBe(secondRender.setSessionReplayOptIn);
      expect(firstRender.initializeSettings).toBe(secondRender.initializeSettings);
    });

    it('provides working action functions', () => {
      const { result: actionsResult } = renderHook(() => useSettingsActions());
      const { result: stateResult } = renderHook(() => useSettingsStore());

      act(() => {
        actionsResult.current.setTelemetryOptOut(true);
      });

      expect(stateResult.current.telemetryOptOut).toBe(true);
    });
  });
});
