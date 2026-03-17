/**
 * Mock LiveKit Client
 * Mock implementation for testing
 */

import { vi } from 'vitest';

export const mockRoom = {
  connect: vi.fn(),
  disconnect: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
  removeAllListeners: vi.fn(),
  localParticipant: {
    setMicrophoneEnabled: vi.fn(),
    isMicrophoneEnabled: true,
    identity: 'local-user',
    name: 'Local User',
    isSpeaking: false,
    getTrackPublication: vi.fn(),
  },
  remoteParticipants: new Map(),
};

// Use a class to properly mock the Room constructor
export class Room {
  connect = mockRoom.connect;
  disconnect = mockRoom.disconnect;
  on = mockRoom.on;
  off = mockRoom.off;
  removeAllListeners = mockRoom.removeAllListeners;
  localParticipant = mockRoom.localParticipant;
  remoteParticipants = mockRoom.remoteParticipants;
}

export const RoomEvent = {
  Connected: 'connected',
  Disconnected: 'disconnected',
  ParticipantConnected: 'participantConnected',
  ParticipantDisconnected: 'participantDisconnected',
  LocalTrackPublished: 'localTrackPublished',
  LocalTrackUnpublished: 'localTrackUnpublished',
  TrackMuted: 'trackMuted',
  TrackUnmuted: 'trackUnmuted',
  ActiveSpeakersChanged: 'activeSpeakersChanged',
  ConnectionQualityChanged: 'connectionQualityChanged',
};

export const ConnectionQuality = {
  Excellent: 'excellent',
  Good: 'good',
  Poor: 'poor',
  Unknown: 'unknown',
};

export const Track = {
  Source: {
    Microphone: 'microphone',
    Camera: 'camera',
    ScreenShare: 'screen_share',
  },
};

export const DisconnectReason = {
  CLIENT_INITIATED: 'CLIENT_INITIATED',
  DUPLICATE_IDENTITY: 'DUPLICATE_IDENTITY',
  SERVER_SHUTDOWN: 'SERVER_SHUTDOWN',
};
