/**
 * Participant Store
 * Manages LiveKit participant state with real-time synchronization
 */

import { create } from 'zustand';
import type { Participant } from '../types/streaming';

/**
 * Participant metadata for tracking state changes
 */
export interface ParticipantMetadata {
  timestamp: number; // When participant was added/updated
  lastMuteUpdate: number; // Last mute state change timestamp
}

/**
 * Cached participant with metadata
 */
export interface CachedParticipant {
  data: Participant;
  metadata: ParticipantMetadata;
}

/**
 * Participant state
 */
interface ParticipantState {
  // Participants keyed by identity
  participants: Record<string, CachedParticipant>;

  // Local participant identity (for quick lookup)
  localIdentity: string | null;

  // Pending mute updates (for debouncing)
  pendingMuteUpdates: Record<string, ReturnType<typeof setTimeout>>;
}

/**
 * Participant actions
 */
interface ParticipantActions {
  // Add or update a participant
  addParticipant: (participant: Participant) => void;

  // Remove a participant
  removeParticipant: (identity: string) => void;

  // Update participant mute status (debounced)
  updateParticipantMute: (identity: string, isMuted: boolean) => void;

  // Update participant speaking status (immediate)
  updateParticipantSpeaking: (identity: string, isSpeaking: boolean) => void;

  // Set local participant identity
  setLocalIdentity: (identity: string | null) => void;

  // Clear all participants
  clearParticipants: () => void;

  // Get all participants as array
  getParticipantsArray: () => Participant[];

  // Get specific participant
  getParticipant: (identity: string) => Participant | null;

  // Get local participant
  getLocalParticipant: () => Participant | null;
}

/**
 * Full participant store type
 */
export type ParticipantStore = ParticipantState & ParticipantActions;

/**
 * Debounce delay for mute updates (50ms as per requirements)
 */
const MUTE_DEBOUNCE_MS = 50;

/**
 * Normalize participant identity
 * Strips any prefix to ensure consistent identity for display
 */
export function normalizeIdentity(identity: string): string {
  // Remove common prefixes if present
  return identity.replace(/^(user:|participant:)/i, '');
}

/**
 * Create fresh participant metadata
 */
function createFreshMetadata(): ParticipantMetadata {
  const now = Date.now();
  return {
    timestamp: now,
    lastMuteUpdate: now,
  };
}

/**
 * Participant Store
 */
export const useParticipantStore = create<ParticipantStore>((set, get) => ({
  // Initial state
  participants: {},
  localIdentity: null,
  pendingMuteUpdates: {},

  // Add or update participant
  addParticipant: (participant: Participant) => {
    const normalizedIdentity = normalizeIdentity(participant.identity);

    set((state) => ({
      participants: {
        ...state.participants,
        [normalizedIdentity]: {
          data: {
            ...participant,
            identity: normalizedIdentity,
          },
          metadata: state.participants[normalizedIdentity]?.metadata || createFreshMetadata(),
        },
      },
    }));
  },

  // Remove participant
  removeParticipant: (identity: string) => {
    const normalizedIdentity = normalizeIdentity(identity);

    set((state) => {
      // Cancel any pending mute updates
      const pendingTimeout = state.pendingMuteUpdates[normalizedIdentity];
      if (pendingTimeout) {
        clearTimeout(pendingTimeout);
      }

      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { [normalizedIdentity]: _removed, ...remainingParticipants } = state.participants;
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { [normalizedIdentity]: _removedTimeout, ...remainingTimeouts } =
        state.pendingMuteUpdates;

      return {
        participants: remainingParticipants,
        pendingMuteUpdates: remainingTimeouts,
        // Clear local identity if this was the local participant
        localIdentity: state.localIdentity === normalizedIdentity ? null : state.localIdentity,
      };
    });
  },

  // Update participant mute status with debouncing
  updateParticipantMute: (identity: string, isMuted: boolean) => {
    const normalizedIdentity = normalizeIdentity(identity);
    const state = get();

    // Cancel any existing pending update for this participant
    const existingTimeout = state.pendingMuteUpdates[normalizedIdentity];
    if (existingTimeout) {
      clearTimeout(existingTimeout);
    }

    // Schedule the update with debounce
    const timeoutId = setTimeout(() => {
      set((state) => {
        const cached = state.participants[normalizedIdentity];
        if (!cached) return state;

        // Remove timeout from pending updates
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [normalizedIdentity]: _removed, ...remainingTimeouts } = state.pendingMuteUpdates;

        return {
          participants: {
            ...state.participants,
            [normalizedIdentity]: {
              data: {
                ...cached.data,
                isMuted,
              },
              metadata: {
                ...cached.metadata,
                lastMuteUpdate: Date.now(),
              },
            },
          },
          pendingMuteUpdates: remainingTimeouts,
        };
      });
    }, MUTE_DEBOUNCE_MS);

    // Store the timeout ID
    set((state) => ({
      pendingMuteUpdates: {
        ...state.pendingMuteUpdates,
        [normalizedIdentity]: timeoutId,
      },
    }));
  },

  // Update participant speaking status (immediate, no debounce)
  updateParticipantSpeaking: (identity: string, isSpeaking: boolean) => {
    const normalizedIdentity = normalizeIdentity(identity);

    set((state) => {
      const cached = state.participants[normalizedIdentity];
      if (!cached) return state;

      return {
        participants: {
          ...state.participants,
          [normalizedIdentity]: {
            data: {
              ...cached.data,
              isSpeaking,
            },
            metadata: {
              ...cached.metadata,
              timestamp: Date.now(),
            },
          },
        },
      };
    });
  },

  // Set local participant identity
  // Note: Caller is responsible for ensuring the participant exists in the store
  // Typically called after addParticipant() for the local participant
  setLocalIdentity: (identity: string | null) => {
    set({
      localIdentity: identity ? normalizeIdentity(identity) : null,
    });
  },

  // Clear all participants
  clearParticipants: () => {
    const state = get();

    // Cancel all pending mute updates
    Object.values(state.pendingMuteUpdates).forEach((timeout) => {
      if (timeout) clearTimeout(timeout);
    });

    set({
      participants: {},
      localIdentity: null,
      pendingMuteUpdates: {},
    });
  },

  // Get all participants as array
  getParticipantsArray: () => {
    const state = get();
    return Object.values(state.participants).map((cached) => cached.data);
  },

  // Get specific participant
  getParticipant: (identity: string) => {
    const normalizedIdentity = normalizeIdentity(identity);
    const state = get();
    return state.participants[normalizedIdentity]?.data || null;
  },

  // Get local participant
  getLocalParticipant: () => {
    const state = get();
    if (!state.localIdentity) return null;
    return state.participants[state.localIdentity]?.data || null;
  },
}));

/**
 * Hook to get all participants (remote only, excludes local)
 */
export function useParticipants(): Participant[] {
  return useParticipantStore((state) => {
    const localIdentity = state.localIdentity;
    return Object.values(state.participants)
      .filter((cached) => cached.data.identity !== localIdentity)
      .map((cached) => cached.data);
  });
}

/**
 * Hook to get specific participant by identity
 */
export function useParticipant(identity: string): Participant | null {
  const normalizedIdentity = normalizeIdentity(identity);
  return useParticipantStore((state) => state.participants[normalizedIdentity]?.data || null);
}

/**
 * Hook to get local participant
 */
export function useLocalParticipant(): Participant | null {
  return useParticipantStore((state) => {
    if (!state.localIdentity) return null;
    return state.participants[state.localIdentity]?.data || null;
  });
}
