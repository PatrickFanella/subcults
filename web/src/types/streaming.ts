/**
 * Streaming Types
 * Types for LiveKit audio streaming functionality
 */

/**
 * LiveKit token response from backend
 */
export interface LiveKitTokenResponse {
  token: string;
  expires_at: string; // RFC3339 timestamp
}

/**
 * Request payload for LiveKit token
 */
export interface LiveKitTokenRequest {
  room_id: string;
  scene_id?: string;
  event_id?: string;
}

/**
 * Participant in an audio room
 */
export interface Participant {
  identity: string;
  name?: string;
  isLocal: boolean;
  isMuted: boolean;
  isSpeaking: boolean;
}

/**
 * Connection quality levels
 */
export type ConnectionQuality = 'excellent' | 'good' | 'poor' | 'unknown';

/**
 * Audio room state
 */
export interface AudioRoomState {
  roomName: string;
  isConnected: boolean;
  isConnecting: boolean;
  participants: Participant[];
  localParticipant: Participant | null;
  connectionQuality: ConnectionQuality;
  error: string | null;
}

/**
 * Stats for determining connection quality
 */
export interface ConnectionStats {
  packetLoss: number; // 0-1 percentage
  jitter: number; // milliseconds
  latency: number; // milliseconds
}

/**
 * Stream join latency timestamps
 * Tracks the latency from join button click to first audio packet
 */
export interface StreamJoinLatency {
  /** t0: When user clicks join button (ms timestamp) */
  joinClicked: number | null;
  /** t1: When token is received from backend (ms timestamp) */
  tokenReceived: number | null;
  /** t2: When room connection is established (ms timestamp) */
  roomConnected: number | null;
  /** t3: When first audio track is subscribed (ms timestamp) */
  firstAudioSubscribed: number | null;
}

/**
 * Computed latency segments from join timestamps
 */
export interface LatencySegments {
  /** Time from join click to token received (ms) */
  tokenFetch: number | null;
  /** Time from token received to room connected (ms) */
  roomConnection: number | null;
  /** Time from room connected to first audio (ms) */
  audioSubscription: number | null;
  /** Total time from join click to first audio (ms) */
  total: number | null;
}
