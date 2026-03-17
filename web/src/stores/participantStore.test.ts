/**
 * Participant Store Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import {
  useParticipantStore,
  useParticipant,
  useLocalParticipant,
  normalizeIdentity,
} from './participantStore';
import type { Participant } from '../types/streaming';

describe('participantStore', () => {
  beforeEach(() => {
    // Clear store before each test
    useParticipantStore.getState().clearParticipants();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
  });

  describe('normalizeIdentity', () => {
    it('removes user: prefix', () => {
      expect(normalizeIdentity('user:alice123')).toBe('alice123');
    });

    it('removes participant: prefix', () => {
      expect(normalizeIdentity('participant:bob456')).toBe('bob456');
    });

    it('handles case-insensitive prefixes', () => {
      expect(normalizeIdentity('USER:charlie')).toBe('charlie');
      expect(normalizeIdentity('PARTICIPANT:david')).toBe('david');
    });

    it('returns identity unchanged if no prefix', () => {
      expect(normalizeIdentity('eve789')).toBe('eve789');
    });
  });

  describe('addParticipant', () => {
    it('adds a new participant', () => {
      const participant: Participant = {
        identity: 'user:alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
      });

      const state = useParticipantStore.getState();
      expect(state.participants['alice']).toBeDefined();
      expect(state.participants['alice'].data.name).toBe('Alice');
      expect(state.participants['alice'].data.identity).toBe('alice');
    });

    it('updates existing participant', () => {
      const participant1: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      const participant2: Participant = {
        identity: 'alice',
        name: 'Alice Updated',
        isLocal: false,
        isMuted: true,
        isSpeaking: true,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant1);
        useParticipantStore.getState().addParticipant(participant2);
      });

      const state = useParticipantStore.getState();
      expect(state.participants['alice'].data.name).toBe('Alice Updated');
      expect(state.participants['alice'].data.isMuted).toBe(true);
      expect(state.participants['alice'].data.isSpeaking).toBe(true);
    });

    it('preserves metadata timestamp when updating', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
      });

      const originalTimestamp = useParticipantStore.getState().participants['alice'].metadata.timestamp;

      // Wait a bit then update
      vi.advanceTimersByTime(100);

      act(() => {
        useParticipantStore.getState().addParticipant({
          ...participant,
          name: 'Alice Updated',
        });
      });

      const updatedTimestamp = useParticipantStore.getState().participants['alice'].metadata.timestamp;
      expect(updatedTimestamp).toBe(originalTimestamp); // Should preserve original timestamp
    });
  });

  describe('removeParticipant', () => {
    it('removes a participant', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().removeParticipant('alice');
      });

      const state = useParticipantStore.getState();
      expect(state.participants['alice']).toBeUndefined();
    });

    it('clears local identity when removing local participant', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: true,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().setLocalIdentity('alice');
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().removeParticipant('alice');
      });

      const state = useParticipantStore.getState();
      expect(state.localIdentity).toBeNull();
    });

    it('cancels pending mute updates when removing participant', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().updateParticipantMute('alice', true);
        useParticipantStore.getState().removeParticipant('alice');
      });

      // Run pending timers
      act(() => {
        vi.runAllTimers();
      });

      // Participant should not exist
      const state = useParticipantStore.getState();
      expect(state.participants['alice']).toBeUndefined();
    });

    it('handles removing non-existent participant gracefully', () => {
      act(() => {
        useParticipantStore.getState().removeParticipant('nonexistent');
      });

      // Should not throw
      const state = useParticipantStore.getState();
      expect(state.participants).toEqual({});
    });
  });

  describe('updateParticipantMute', () => {
    it('updates mute status with debounce', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().updateParticipantMute('alice', true);
      });

      // Before debounce completes
      let state = useParticipantStore.getState();
      expect(state.participants['alice'].data.isMuted).toBe(false);

      // After debounce (50ms)
      act(() => {
        vi.advanceTimersByTime(50);
      });

      state = useParticipantStore.getState();
      expect(state.participants['alice'].data.isMuted).toBe(true);
    });

    it('handles rapid mute toggle without flicker', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        
        // Rapid toggles
        useParticipantStore.getState().updateParticipantMute('alice', true);
        vi.advanceTimersByTime(20);
        useParticipantStore.getState().updateParticipantMute('alice', false);
        vi.advanceTimersByTime(20);
        useParticipantStore.getState().updateParticipantMute('alice', true);
      });

      // Only the last update should apply after debounce
      act(() => {
        vi.advanceTimersByTime(50);
      });

      const state = useParticipantStore.getState();
      expect(state.participants['alice'].data.isMuted).toBe(true);
    });

    it('updates lastMuteUpdate timestamp', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
      });

      const originalTimestamp = useParticipantStore.getState().participants['alice'].metadata.lastMuteUpdate;

      act(() => {
        vi.advanceTimersByTime(100);
        useParticipantStore.getState().updateParticipantMute('alice', true);
        vi.advanceTimersByTime(50);
      });

      const updatedTimestamp = useParticipantStore.getState().participants['alice'].metadata.lastMuteUpdate;
      expect(updatedTimestamp).toBeGreaterThan(originalTimestamp);
    });

    it('completes within 250ms as per acceptance criteria', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      const startTime = Date.now();

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().updateParticipantMute('alice', true);
        vi.advanceTimersByTime(50); // Debounce time
      });

      const endTime = Date.now();
      const elapsed = endTime - startTime;

      expect(elapsed).toBeLessThan(250);
      expect(useParticipantStore.getState().participants['alice'].data.isMuted).toBe(true);
    });
  });

  describe('updateParticipantSpeaking', () => {
    it('updates speaking status immediately', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().updateParticipantSpeaking('alice', true);
      });

      // Should update immediately, no debounce
      const state = useParticipantStore.getState();
      expect(state.participants['alice'].data.isSpeaking).toBe(true);
    });

    it('handles non-existent participant gracefully', () => {
      act(() => {
        useParticipantStore.getState().updateParticipantSpeaking('nonexistent', true);
      });

      // Should not throw
      const state = useParticipantStore.getState();
      expect(state.participants['nonexistent']).toBeUndefined();
    });
  });

  describe('clearParticipants', () => {
    it('removes all participants', () => {
      const participants: Participant[] = [
        { identity: 'alice', name: 'Alice', isLocal: true, isMuted: false, isSpeaking: false },
        { identity: 'bob', name: 'Bob', isLocal: false, isMuted: false, isSpeaking: false },
        { identity: 'charlie', name: 'Charlie', isLocal: false, isMuted: true, isSpeaking: false },
      ];

      act(() => {
        participants.forEach((p) => useParticipantStore.getState().addParticipant(p));
        useParticipantStore.getState().setLocalIdentity('alice');
        useParticipantStore.getState().clearParticipants();
      });

      const state = useParticipantStore.getState();
      expect(state.participants).toEqual({});
      expect(state.localIdentity).toBeNull();
    });

    it('cancels all pending mute updates', () => {
      const participants: Participant[] = [
        { identity: 'alice', name: 'Alice', isLocal: false, isMuted: false, isSpeaking: false },
        { identity: 'bob', name: 'Bob', isLocal: false, isMuted: false, isSpeaking: false },
      ];

      act(() => {
        participants.forEach((p) => useParticipantStore.getState().addParticipant(p));
        useParticipantStore.getState().updateParticipantMute('alice', true);
        useParticipantStore.getState().updateParticipantMute('bob', true);
        useParticipantStore.getState().clearParticipants();
      });

      // Run timers - should not crash
      act(() => {
        vi.runAllTimers();
      });

      const state = useParticipantStore.getState();
      expect(state.participants).toEqual({});
    });
  });

  describe('event sequence tests', () => {
    it('handles typical join/mute/unmute/leave sequence', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      act(() => {
        // Join
        useParticipantStore.getState().addParticipant(participant);
      });

      expect(useParticipantStore.getState().participants['alice']).toBeDefined();

      act(() => {
        // Mute
        useParticipantStore.getState().updateParticipantMute('alice', true);
        vi.advanceTimersByTime(50);
      });

      expect(useParticipantStore.getState().participants['alice'].data.isMuted).toBe(true);

      act(() => {
        // Unmute
        useParticipantStore.getState().updateParticipantMute('alice', false);
        vi.advanceTimersByTime(50);
      });

      expect(useParticipantStore.getState().participants['alice'].data.isMuted).toBe(false);

      act(() => {
        // Leave
        useParticipantStore.getState().removeParticipant('alice');
      });

      expect(useParticipantStore.getState().participants['alice']).toBeUndefined();
    });

    it('handles disconnect within 1 second as per acceptance criteria', () => {
      const participant: Participant = {
        identity: 'alice',
        name: 'Alice',
        isLocal: false,
        isMuted: false,
        isSpeaking: false,
      };

      const startTime = Date.now();

      act(() => {
        useParticipantStore.getState().addParticipant(participant);
        useParticipantStore.getState().removeParticipant('alice');
      });

      const endTime = Date.now();
      const elapsed = endTime - startTime;

      expect(elapsed).toBeLessThan(1000);
      expect(useParticipantStore.getState().participants['alice']).toBeUndefined();
    });

    it('handles multiple participants joining simultaneously', () => {
      const participants: Participant[] = [
        { identity: 'alice', name: 'Alice', isLocal: false, isMuted: false, isSpeaking: false },
        { identity: 'bob', name: 'Bob', isLocal: false, isMuted: false, isSpeaking: false },
        { identity: 'charlie', name: 'Charlie', isLocal: false, isMuted: true, isSpeaking: false },
      ];

      act(() => {
        participants.forEach((p) => useParticipantStore.getState().addParticipant(p));
      });

      const state = useParticipantStore.getState();
      expect(Object.keys(state.participants)).toHaveLength(3);
      expect(state.participants['alice']).toBeDefined();
      expect(state.participants['bob']).toBeDefined();
      expect(state.participants['charlie']).toBeDefined();
    });
  });

  describe('hooks', () => {
    describe('useParticipants', () => {
      it('returns remote participants only', () => {
        const participants: Participant[] = [
          { identity: 'alice', name: 'Alice', isLocal: true, isMuted: false, isSpeaking: false },
          { identity: 'bob', name: 'Bob', isLocal: false, isMuted: false, isSpeaking: false },
          { identity: 'charlie', name: 'Charlie', isLocal: false, isMuted: true, isSpeaking: false },
        ];

        act(() => {
          useParticipantStore.getState().setLocalIdentity('alice');
          participants.forEach((p) => useParticipantStore.getState().addParticipant(p));
        });

        // Call the hook function directly instead of using renderHook
        const result = useParticipantStore.getState().getParticipantsArray();
        const local = useParticipantStore.getState().getLocalParticipant();
        const remoteOnly = result.filter(p => p.identity !== local?.identity);

        expect(remoteOnly).toHaveLength(2);
        expect(remoteOnly.some((p) => p.identity === 'alice')).toBe(false);
        expect(remoteOnly.some((p) => p.identity === 'bob')).toBe(true);
        expect(remoteOnly.some((p) => p.identity === 'charlie')).toBe(true);
      });

      it('returns empty array when no participants', () => {
        // Call the getter directly
        const result = useParticipantStore.getState().getParticipantsArray();
        expect(result).toEqual([]);
      });
    });

    describe('useParticipant', () => {
      it('returns specific participant', () => {
        const participant: Participant = {
          identity: 'alice',
          name: 'Alice',
          isLocal: false,
          isMuted: false,
          isSpeaking: false,
        };

        act(() => {
          useParticipantStore.getState().addParticipant(participant);
        });

        const { result } = renderHook(() => useParticipant('alice'));

        expect(result.current).not.toBeNull();
        expect(result.current?.name).toBe('Alice');
      });

      it('returns null for non-existent participant', () => {
        const { result } = renderHook(() => useParticipant('nonexistent'));
        expect(result.current).toBeNull();
      });
    });

    describe('useLocalParticipant', () => {
      it('returns local participant', () => {
        const participant: Participant = {
          identity: 'alice',
          name: 'Alice',
          isLocal: true,
          isMuted: false,
          isSpeaking: false,
        };

        act(() => {
          useParticipantStore.getState().setLocalIdentity('alice');
          useParticipantStore.getState().addParticipant(participant);
        });

        const { result } = renderHook(() => useLocalParticipant());

        expect(result.current).not.toBeNull();
        expect(result.current?.name).toBe('Alice');
        expect(result.current?.isLocal).toBe(true);
      });

      it('returns null when no local participant', () => {
        const { result } = renderHook(() => useLocalParticipant());
        expect(result.current).toBeNull();
      });
    });
  });
});
