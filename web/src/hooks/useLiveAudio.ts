/**
 * useLiveAudio Hook
 * Manages LiveKit audio room connection, participants, and token refresh
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Room,
  RoomEvent,
  ConnectionQuality as LKConnectionQuality,
  Participant as LKParticipant,
  Track,
  DisconnectReason,
} from 'livekit-client';
import { apiClient } from '../lib/api-client';
import { useParticipantStore, normalizeIdentity } from '../stores/participantStore';
import { useLatencyStore } from '../stores/latencyStore';
import type { AudioRoomState, Participant, ConnectionQuality } from '../types/streaming';

/**
 * Hook options
 */
export interface UseLiveAudioOptions {
  sceneId?: string;
  eventId?: string;
  onError?: (error: Error) => void;
}

/**
 * Hook result
 */
export interface UseLiveAudioResult extends AudioRoomState {
  connect: () => Promise<void>;
  disconnect: () => void;
  toggleMute: () => Promise<void>;
  setVolume: (volume: number) => void;
}

/**
 * Token expiry threshold for refresh (30 seconds before expiry)
 */
const TOKEN_REFRESH_THRESHOLD_MS = 30 * 1000;

/**
 * Map LiveKit connection quality to our quality type
 */
function mapConnectionQuality(lkQuality: LKConnectionQuality): ConnectionQuality {
  switch (lkQuality) {
    case LKConnectionQuality.Excellent:
      return 'excellent';
    case LKConnectionQuality.Good:
      return 'good';
    case LKConnectionQuality.Poor:
      return 'poor';
    default:
      return 'unknown';
  }
}

/**
 * Convert LiveKit participant to our Participant type
 */
function convertParticipant(participant: LKParticipant, isLocal: boolean): Participant {
  const audioTrack = participant.getTrackPublication(Track.Source.Microphone);

  return {
    identity: participant.identity,
    name: participant.name || participant.identity,
    isLocal,
    isMuted: audioTrack?.isMuted ?? true,
    isSpeaking: participant.isSpeaking,
  };
}

/**
 * useLiveAudio hook
 * Manages LiveKit room connection lifecycle and state
 */
export function useLiveAudio(
  roomName: string | null,
  options: UseLiveAudioOptions = {}
): UseLiveAudioResult {
  // Destructure options to avoid dependency on the entire object
  const { sceneId, eventId, onError } = options;

  const [state, setState] = useState<AudioRoomState>({
    roomName: roomName || '',
    isConnected: false,
    isConnecting: false,
    participants: [],
    localParticipant: null,
    connectionQuality: 'unknown',
    error: null,
  });

  const roomRef = useRef<Room | null>(null);
  const tokenExpiryRef = useRef<number | null>(null);
  const refreshTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /**
   * Update participants list from room
   */
  const updateParticipants = useCallback(() => {
    const room = roomRef.current;
    if (!room) return;

    // Get stable store reference
    const store = useParticipantStore.getState();

    // Update remote participants in store
    room.remoteParticipants.forEach((participant) => {
      const converted = convertParticipant(participant, false);
      store.addParticipant(converted);
    });

    // Update local participant in store
    if (room.localParticipant) {
      const localPart = convertParticipant(room.localParticipant, true);
      store.addParticipant(localPart);
      store.setLocalIdentity(localPart.identity);
    }

    // Sync state with store for component access
    const remoteParticipants = store
      .getParticipantsArray()
      .filter((p) => p.identity !== store.localIdentity);
    const localPart = store.getLocalParticipant();

    setState((prev) => ({
      ...prev,
      participants: remoteParticipants,
      localParticipant: localPart,
    }));
  }, []);

  /**
   * Schedule token refresh before expiry
   */
  const scheduleTokenRefresh = useCallback(async (expiresAt: string) => {
    // Clear any existing refresh timeout
    if (refreshTimeoutRef.current) {
      clearTimeout(refreshTimeoutRef.current);
      refreshTimeoutRef.current = null;
    }

    const expiryTime = new Date(expiresAt).getTime();
    tokenExpiryRef.current = expiryTime;

    const now = Date.now();
    const timeUntilRefresh = expiryTime - now - TOKEN_REFRESH_THRESHOLD_MS;

    // Only schedule if we have time before expiry
    if (timeUntilRefresh > 0) {
      refreshTimeoutRef.current = setTimeout(() => {
        // Note: Token refresh in LiveKit 2.x requires reconnection
        // For now, we'll let the connection expire and require manual rejoin
        // TODO: Implement seamless reconnection with new token
        console.info('LiveKit token will expire soon; manual rejoin required.');

        // Set a warning in state to notify user
        setState((prev) => ({
          ...prev,
          error: 'Session will expire soon. Please rejoin if disconnected.',
        }));
      }, timeUntilRefresh);
    }
  }, []);

  /**
   * Connect to room
   */
  const connect = useCallback(async () => {
    if (!roomName || state.isConnected || state.isConnecting) return;

    // Reset latency tracking for new join attempt (including lastLatency to avoid stale metrics)
    const latencyStore = useLatencyStore.getState();
    latencyStore.resetLatency();
    useLatencyStore.setState((prev) => ({
      ...prev,
      lastLatency: null,
    }));

    setState((prev) => ({ ...prev, isConnecting: true, error: null }));

    try {
      // Fetch token (t1: token received)
      const { token, expires_at } = await apiClient.getLiveKitToken(roomName, sceneId, eventId);

      // Record token received timestamp
      latencyStore.recordTokenReceived();

      // Create and connect room
      const room = new Room();
      roomRef.current = room;

      // Set up event listeners before connecting
      room.on(RoomEvent.ParticipantConnected, (participant: LKParticipant) => {
        const store = useParticipantStore.getState();
        const converted = convertParticipant(participant, false);
        store.addParticipant(converted);
        updateParticipants();
      });

      room.on(RoomEvent.ParticipantDisconnected, (participant: LKParticipant) => {
        const store = useParticipantStore.getState();
        store.removeParticipant(participant.identity);
        updateParticipants();
      });

      room.on(RoomEvent.LocalTrackPublished, updateParticipants);
      room.on(RoomEvent.LocalTrackUnpublished, updateParticipants);

      room.on(RoomEvent.TrackMuted, (publication, participant: LKParticipant) => {
        if (publication.source === Track.Source.Microphone) {
          const store = useParticipantStore.getState();
          store.updateParticipantMute(participant.identity, true);
          updateParticipants();
        }
      });

      room.on(RoomEvent.TrackUnmuted, (publication, participant: LKParticipant) => {
        if (publication.source === Track.Source.Microphone) {
          const store = useParticipantStore.getState();
          store.updateParticipantMute(participant.identity, false);
          updateParticipants();
        }
      });

      room.on(RoomEvent.ActiveSpeakersChanged, (speakers: LKParticipant[]) => {
        const store = useParticipantStore.getState();
        // Get current speaking state
        const allParticipants = store.getParticipantsArray();

        // Normalize LiveKit identities to match store participants
        // LiveKit may provide 'user:alice' while store has 'alice'
        const speakerIdentities = new Set(speakers.map((s) => normalizeIdentity(s.identity)));

        // Only update participants whose speaking status changed
        allParticipants.forEach((p) => {
          const shouldBeSpeaking = speakerIdentities.has(p.identity);
          if (p.isSpeaking !== shouldBeSpeaking) {
            store.updateParticipantSpeaking(p.identity, shouldBeSpeaking);
          }
        });

        updateParticipants();
      });

      // Track first audio subscription for latency measurement (t3)
      let firstAudioTracked = false;
      room.on(RoomEvent.TrackSubscribed, (track) => {
        if (!firstAudioTracked && track.kind === 'audio') {
          const latencyStore = useLatencyStore.getState();
          latencyStore.recordFirstAudioSubscribed();
          latencyStore.finalizeLatency();
          firstAudioTracked = true;
        }
      });

      // Connection quality monitoring
      room.on(RoomEvent.ConnectionQualityChanged, (quality: LKConnectionQuality) => {
        setState((prev) => ({
          ...prev,
          connectionQuality: mapConnectionQuality(quality),
        }));
      });

      // Error handling
      room.on(RoomEvent.Disconnected, (reason?: DisconnectReason) => {
        // Only set error for unexpected disconnects
        const isClientInitiated = reason === DisconnectReason.CLIENT_INITIATED;
        const reasonStr = reason ? String(reason) : undefined;
        setState((prev) => ({
          ...prev,
          isConnected: false,
          isConnecting: false,
          error: isClientInitiated ? null : reasonStr || null,
        }));
      });

      // Connect to room
      const wsUrl = import.meta.env.VITE_LIVEKIT_WS_URL;
      if (!wsUrl || typeof wsUrl !== 'string' || wsUrl.trim() === '') {
        setState((prev) => ({
          ...prev,
          isConnected: false,
          isConnecting: false,
          error:
            'LiveKit WebSocket URL is not configured. Please set VITE_LIVEKIT_WS_URL in your environment.',
        }));
        return;
      }

      await room.connect(wsUrl, token);

      // Record room connected timestamp (t2)
      latencyStore.recordRoomConnected();

      // Enable local microphone
      await room.localParticipant.setMicrophoneEnabled(true);

      // Update state
      setState((prev) => ({
        ...prev,
        isConnected: true,
        isConnecting: false,
        roomName,
      }));

      // Update participants
      updateParticipants();

      // Schedule token refresh
      scheduleTokenRefresh(expires_at);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to connect to room';

      setState((prev) => ({
        ...prev,
        isConnecting: false,
        error: errorMessage,
      }));

      if (onError) {
        onError(error instanceof Error ? error : new Error(errorMessage));
      }

      // Clean up on error
      if (roomRef.current) {
        roomRef.current.disconnect();
        roomRef.current = null;
      }
    }
  }, [
    roomName,
    state.isConnected,
    state.isConnecting,
    sceneId,
    eventId,
    onError,
    updateParticipants,
    scheduleTokenRefresh,
  ]);

  /**
   * Disconnect from room
   */
  const disconnect = useCallback(() => {
    if (refreshTimeoutRef.current) {
      clearTimeout(refreshTimeoutRef.current);
      refreshTimeoutRef.current = null;
    }

    if (roomRef.current) {
      // Remove all event listeners to prevent memory leaks and duplicate handlers
      roomRef.current.removeAllListeners();
      roomRef.current.disconnect();
      roomRef.current = null;
    }

    // Clear participant store
    const store = useParticipantStore.getState();
    store.clearParticipants();

    setState({
      roomName: roomName || '',
      isConnected: false,
      isConnecting: false,
      participants: [],
      localParticipant: null,
      connectionQuality: 'unknown',
      error: null,
    });
  }, [roomName]);

  /**
   * Toggle local microphone mute
   */
  const toggleMute = useCallback(async () => {
    const room = roomRef.current;
    if (!room) return;

    const isEnabled = room.localParticipant.isMicrophoneEnabled;
    await room.localParticipant.setMicrophoneEnabled(!isEnabled);
    updateParticipants();
  }, [updateParticipants]);

  /**
   * Set playback volume (0-100)
   * Note: This adjusts the volume of remote audio tracks
   */
  const setVolume = useCallback((volume: number) => {
    const room = roomRef.current;
    if (!room) return;

    // Clamp volume between 0 and 100
    const clampedVolume = Math.max(0, Math.min(100, volume));
    const normalizedVolume = clampedVolume / 100;

    // Set volume for all remote audio tracks
    room.remoteParticipants.forEach((participant) => {
      participant.audioTrackPublications.forEach((publication) => {
        if (publication.audioTrack) {
          // Volume control may not be available in all LiveKit versions
          // This is a best-effort approach using type guard
          try {
            // Type guard: check if setVolume exists and is a function
            if (
              typeof (publication.audioTrack as { setVolume?: unknown }).setVolume === 'function'
            ) {
              (publication.audioTrack as { setVolume: (volume: number) => void }).setVolume(
                normalizedVolume
              );
            }
          } catch (error) {
            console.warn('Volume control not supported:', error);
          }
        }
      });
    });
  }, []);

  /**
   * Cleanup on unmount
   */
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return {
    ...state,
    connect,
    disconnect,
    toggleMute,
    setVolume,
  };
}
