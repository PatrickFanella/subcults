/**
 * Client Telemetry Types
 * Structured event types for analytics collection
 */

/**
 * Base telemetry event structure
 * All events must include name, timestamp, and sessionId
 */
export interface TelemetryEvent {
  /** Event name (use dot-notation: e.g., 'search.scene', 'stream.join') */
  name: string;
  /** Event timestamp in milliseconds since epoch */
  ts: number;
  /** Optional user DID (only if authenticated) */
  userId?: string;
  /** Session ID (UUID v4) unique to the tab */
  sessionId: string;
  /** Event-specific payload data (keep minimal for privacy) */
  payload?: Record<string, unknown>;
}

/**
 * Event naming conventions:
 * 
 * Use dot-notation to organize events hierarchically:
 * - search.scene - Scene search performed
 * - search.event - Event search performed
 * - search.post - Post search performed
 * - stream.join - Joined audio stream
 * - stream.join_completed - Completed joining audio stream with latency breakdown
 * - stream.leave - Left audio stream
 * - scene.view - Viewed scene details
 * - event.view - Viewed event details
 * - map.zoom - Map zoom level changed
 * - map.move - Map viewport moved
 * 
 * Payload guidelines:
 * - Keep payloads minimal (only essential metadata)
 * - Avoid sensitive content (PII, precise locations, full text)
 * - Example payloads:
 *   - search.scene: { query_length: 5, results_count: 10 }
 *   - stream.join: { room_id: "uuid", duration_ms: 1234 }
 *   - stream.join_completed: { total_ms: 2500, token_fetch_ms: 500, room_connect_ms: 1200, audio_sub_ms: 800 }
 *   - scene.view: { scene_id: "uuid" }
 */
