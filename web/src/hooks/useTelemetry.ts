/**
 * useTelemetry Hook
 * React hook for emitting telemetry events
 */

import { useCallback } from 'react';
import { telemetryService } from '../lib/telemetry-service';
import { useAuth } from '../stores/authStore';
import { useTelemetryOptOut } from '../stores/settingsStore';

/**
 * Emit function type
 */
export type EmitFunction = (name: string, payload?: Record<string, unknown>) => void;

/**
 * Hook for emitting telemetry events
 * Automatically includes userId if user is authenticated
 * Respects user opt-out preference
 * 
 * @returns emit function for sending telemetry events
 * 
 * @example
 * ```tsx
 * const emit = useTelemetry();
 * 
 * // Emit search event
 * emit('search.scene', { query_length: 5, results_count: 10 });
 * 
 * // Emit stream join event
 * emit('stream.join', { room_id: 'xyz', duration_ms: 1234 });
 * ```
 */
export function useTelemetry(): EmitFunction {
  const { user } = useAuth();
  const isOptedOut = useTelemetryOptOut();

  // Create stable emit function
  const emit = useCallback(
    (name: string, payload?: Record<string, unknown>) => {
      // Skip if user opted out
      if (isOptedOut) {
        return;
      }

      // Include userId if authenticated
      const userId = user?.did;
      telemetryService.emit(name, payload, userId);
    },
    [user?.did, isOptedOut]
  );

  return emit;
}
