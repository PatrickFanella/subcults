/**
 * Latency Store
 * Tracks stream join latency timestamps for performance monitoring
 */

import { create } from 'zustand';
import type { StreamJoinLatency, LatencySegments } from '../types/streaming';

/**
 * Latency state
 */
interface LatencyState {
  // Current join attempt timestamps
  currentLatency: StreamJoinLatency;
  
  // Last completed join latency (for display after connection)
  lastLatency: StreamJoinLatency | null;
}

/**
 * Latency actions
 */
interface LatencyActions {
  // Record timestamp for join button click (t0)
  recordJoinClicked: () => void;
  
  // Record timestamp for token received (t1)
  recordTokenReceived: () => void;
  
  // Record timestamp for room connected (t2)
  recordRoomConnected: () => void;
  
  // Record timestamp for first audio subscribed (t3)
  recordFirstAudioSubscribed: () => void;
  
  // Reset current latency tracking (for new join attempt)
  resetLatency: () => void;
  
  // Finalize current latency and store as last (when join completes)
  finalizeLatency: () => void;
  
  // Compute latency segments from timestamps
  computeSegments: (latency?: StreamJoinLatency) => LatencySegments;
}

/**
 * Full latency store type
 */
export type LatencyStore = LatencyState & LatencyActions;

/**
 * Initial empty latency state
 */
const emptyLatency: StreamJoinLatency = {
  joinClicked: null,
  tokenReceived: null,
  roomConnected: null,
  firstAudioSubscribed: null,
};

/**
 * Latency Store
 * Manages join latency tracking with high-resolution timestamps
 */
export const useLatencyStore = create<LatencyStore>((set, get) => ({
  // Initial state
  currentLatency: { ...emptyLatency },
  lastLatency: null,

  // Record join button click (t0)
  recordJoinClicked: () => {
    const timestamp = performance.now();
    set({
      currentLatency: {
        joinClicked: timestamp,
        tokenReceived: null,
        roomConnected: null,
        firstAudioSubscribed: null,
      },
    });
    
    if (import.meta.env.DEV) {
      console.log(`[Latency] Join clicked at t0=${timestamp.toFixed(2)}ms`);
    }
  },

  // Record token received (t1)
  recordTokenReceived: () => {
    const timestamp = performance.now();
    set((state) => ({
      currentLatency: {
        ...state.currentLatency,
        tokenReceived: timestamp,
      },
    }));
    
    if (import.meta.env.DEV) {
      const state = get();
      const elapsed = state.currentLatency.joinClicked
        ? timestamp - state.currentLatency.joinClicked
        : null;
      console.log(
        `[Latency] Token received at t1=${timestamp.toFixed(2)}ms (${elapsed?.toFixed(2)}ms from t0)`
      );
    }
  },

  // Record room connected (t2)
  recordRoomConnected: () => {
    const timestamp = performance.now();
    set((state) => ({
      currentLatency: {
        ...state.currentLatency,
        roomConnected: timestamp,
      },
    }));
    
    if (import.meta.env.DEV) {
      const state = get();
      const elapsed = state.currentLatency.tokenReceived
        ? timestamp - state.currentLatency.tokenReceived
        : null;
      console.log(
        `[Latency] Room connected at t2=${timestamp.toFixed(2)}ms (${elapsed?.toFixed(2)}ms from t1)`
      );
    }
  },

  // Record first audio subscribed (t3)
  recordFirstAudioSubscribed: () => {
    const timestamp = performance.now();
    set((state) => ({
      currentLatency: {
        ...state.currentLatency,
        firstAudioSubscribed: timestamp,
      },
    }));
    
    if (import.meta.env.DEV) {
      const state = get();
      const segments = state.computeSegments(state.currentLatency);
      console.log(
        `[Latency] First audio at t3=${timestamp.toFixed(2)}ms (Total: ${segments.total?.toFixed(2)}ms)`
      );
      console.log('[Latency] Segments:', {
        tokenFetch: segments.tokenFetch?.toFixed(2),
        roomConnection: segments.roomConnection?.toFixed(2),
        audioSubscription: segments.audioSubscription?.toFixed(2),
      });
    }
  },

  // Reset latency state
  // Note: This only resets currentLatency. To clear lastLatency as well,
  // call useLatencyStore.setState({ lastLatency: null }) separately
  resetLatency: () => {
    set({
      currentLatency: { ...emptyLatency },
    });
  },

  // Finalize current latency as last
  finalizeLatency: () => {
    set((state) => ({
      lastLatency: { ...state.currentLatency },
    }));
  },

  // Compute latency segments
  computeSegments: (latency?: StreamJoinLatency): LatencySegments => {
    const l = latency || get().currentLatency;
    
    // Compute segments if we have the necessary timestamps
    const tokenFetch =
      l.joinClicked !== null && l.tokenReceived !== null
        ? l.tokenReceived - l.joinClicked
        : null;
    
    const roomConnection =
      l.tokenReceived !== null && l.roomConnected !== null
        ? l.roomConnected - l.tokenReceived
        : null;
    
    const audioSubscription =
      l.roomConnected !== null && l.firstAudioSubscribed !== null
        ? l.firstAudioSubscribed - l.roomConnected
        : null;
    
    const total =
      l.joinClicked !== null && l.firstAudioSubscribed !== null
        ? l.firstAudioSubscribed - l.joinClicked
        : null;
    
    return {
      tokenFetch,
      roomConnection,
      audioSubscription,
      total,
    };
  },
}));
