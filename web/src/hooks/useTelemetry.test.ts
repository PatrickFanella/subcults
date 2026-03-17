/**
 * useTelemetry Hook Tests
 * Validates telemetry hook behavior with auth and opt-out
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTelemetry } from './useTelemetry';
import { telemetryService } from '../lib/telemetry-service';
import { authStore } from '../stores/authStore';
import { useSettingsStore } from '../stores/settingsStore';

// Mock telemetry service
vi.mock('../lib/telemetry-service', () => ({
  telemetryService: {
    emit: vi.fn(),
    flush: vi.fn(),
  },
}));

describe('useTelemetry', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    
    // Reset stores
    authStore.setUser({ did: 'did:plc:test123', role: 'user' }, 'mock-token');
    useSettingsStore.setState({ telemetryOptOut: false });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('basic emission', () => {
    it('returns emit function', () => {
      const { result } = renderHook(() => useTelemetry());

      expect(typeof result.current).toBe('function');
    });

    it('emits events with name and payload', () => {
      const { result } = renderHook(() => useTelemetry());

      act(() => {
        result.current('test.event', { key: 'value' });
      });

      expect(telemetryService.emit).toHaveBeenCalledWith(
        'test.event',
        { key: 'value' },
        'did:plc:test123'
      );
    });

    it('emits events without payload', () => {
      const { result } = renderHook(() => useTelemetry());

      act(() => {
        result.current('test.event');
      });

      expect(telemetryService.emit).toHaveBeenCalledWith(
        'test.event',
        undefined,
        'did:plc:test123'
      );
    });
  });

  describe('authentication integration', () => {
    it('includes userId when user is authenticated', () => {
      authStore.setUser({ did: 'did:plc:user456', role: 'user' }, 'token');
      
      const { result } = renderHook(() => useTelemetry());

      act(() => {
        result.current('test.event');
      });

      expect(telemetryService.emit).toHaveBeenCalledWith(
        'test.event',
        undefined,
        'did:plc:user456'
      );
    });

    it('emits without userId when user is not authenticated', () => {
      // Reset to unauthenticated state
      authStore.resetForTesting();
      
      const { result } = renderHook(() => useTelemetry());

      act(() => {
        result.current('test.event');
      });

      expect(telemetryService.emit).toHaveBeenCalledWith(
        'test.event',
        undefined,
        undefined
      );
    });

    it('updates userId when user changes', () => {
      const { result } = renderHook(() => useTelemetry());

      // First emission with initial user
      act(() => {
        result.current('event.1');
      });

      expect(telemetryService.emit).toHaveBeenCalledWith(
        'event.1',
        undefined,
        'did:plc:test123'
      );

      // Change user
      act(() => {
        authStore.setUser({ did: 'did:plc:newuser', role: 'user' }, 'new-token');
      });

      // Second emission with new user
      act(() => {
        result.current('event.2');
      });

      expect(telemetryService.emit).toHaveBeenCalledWith(
        'event.2',
        undefined,
        'did:plc:newuser'
      );
    });
  });

  describe('opt-out behavior', () => {
    it('does not emit when user has opted out', () => {
      useSettingsStore.setState({ telemetryOptOut: true });
      
      const { result } = renderHook(() => useTelemetry());

      act(() => {
        result.current('test.event');
      });

      expect(telemetryService.emit).not.toHaveBeenCalled();
    });

    it('emits when user opts back in', () => {
      useSettingsStore.setState({ telemetryOptOut: true });
      
      const { result } = renderHook(() => useTelemetry());

      // No emission when opted out
      act(() => {
        result.current('event.1');
      });
      expect(telemetryService.emit).not.toHaveBeenCalled();

      // Opt back in
      act(() => {
        useSettingsStore.setState({ telemetryOptOut: false });
      });

      // Now emissions work
      act(() => {
        result.current('event.2');
      });
      expect(telemetryService.emit).toHaveBeenCalledWith(
        'event.2',
        undefined,
        'did:plc:test123'
      );
    });
  });

  describe('stability', () => {
    it('returns stable emit function when dependencies do not change', () => {
      const { result, rerender } = renderHook(() => useTelemetry());
      const firstEmit = result.current;

      rerender();
      const secondEmit = result.current;

      expect(firstEmit).toBe(secondEmit);
    });

    it('returns new emit function when user changes', () => {
      const { result, rerender } = renderHook(() => useTelemetry());
      const firstEmit = result.current;

      act(() => {
        authStore.setUser({ did: 'did:plc:different', role: 'user' }, 'token');
      });

      rerender();
      const secondEmit = result.current;

      expect(firstEmit).not.toBe(secondEmit);
    });

    it('returns new emit function when opt-out changes', () => {
      const { result, rerender } = renderHook(() => useTelemetry());
      const firstEmit = result.current;

      act(() => {
        useSettingsStore.setState({ telemetryOptOut: true });
      });

      rerender();
      const secondEmit = result.current;

      expect(firstEmit).not.toBe(secondEmit);
    });
  });
});
