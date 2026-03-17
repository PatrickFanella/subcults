/**
 * StreamPage Component
 * Live audio streaming room with enhanced UI components
 * This is lazy-loaded due to heavy dependencies
 */

import React, { useMemo, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  useStreamingStore,
  useStreamingIsConnected,
  useStreamingIsConnecting,
  useStreamingError,
  useStreamingConnectionQuality,
  useStreamingConnect,
  useStreamingDisconnect,
} from '../stores/streamingStore';
import { useParticipantStore } from '../stores/participantStore';
import {
  JoinStreamButton,
  ParticipantList,
  AudioControls,
  ConnectionIndicator,
  StreamHeader,
  ShareInviteButtons,
  ChatSidebar,
  AudioLevelVisualization,
} from '../components/streaming';
import { useToasts } from '../stores/toastStore';
import type { ChatMessage } from '../components/streaming';

export const StreamPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation('streaming');
  const { error: showError } = useToasts();
  
  // Global streaming state (individual selectors to avoid unnecessary re-renders)
  const isConnected = useStreamingIsConnected();
  const isConnecting = useStreamingIsConnecting();
  const error = useStreamingError();
  const connectionQuality = useStreamingConnectionQuality();
  const connect = useStreamingConnect();
  const disconnect = useStreamingDisconnect();
  
  // Participant state
  const participants = useParticipantStore((state) => 
    state.getParticipantsArray().filter(p => p.identity !== state.localIdentity)
  );
  const localParticipant = useParticipantStore((state) => state.getLocalParticipant());
  
  // Audio controls from global store
  const isLocalMuted = useStreamingStore((state) => state.isLocalMuted);
  const setVolume = useStreamingStore((state) => state.setVolume);
  const toggleMute = useStreamingStore((state) => state.toggleMute);

  // Chat state (mock for now - will be integrated with backend later)
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);

  // Show error toasts
  useEffect(() => {
    if (error) {
      showError(error);
    }
  }, [error, showError]);

  // Calculate total participant count
  const participantCount = useMemo(
    () => participants.length + (localParticipant ? 1 : 0),
    [participants.length, localParticipant]
  );
  
  // Handle connection
  const handleConnect = async () => {
    if (id) {
      await connect(id);
    }
  };

  // Handle sending chat messages
  const handleSendMessage = (message: string) => {
    const newMessage: ChatMessage = {
      id:
        typeof crypto !== 'undefined' && 'randomUUID' in crypto
          ? crypto.randomUUID()
          : `${Date.now()}-${Math.random().toString(36).slice(2)}`,
      sender: localParticipant?.name || 'You',
      message,
      timestamp: Date.now(),
      isLocal: true,
    };
    setChatMessages((prev) => [...prev, newMessage]);
  };

  // Get current stream URL for sharing
  const streamUrl = window.location.href;

  if (!id) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        <h1>{t('streamPage.invalidRoom')}</h1>
        <p>{t('streamPage.noRoomId')}</p>
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '1rem',
        maxWidth: '1400px',
        margin: '0 auto',
        color: 'white',
      }}
    >
      {/* Stream Header */}
      <div style={{ marginBottom: '1.5rem' }}>
        <StreamHeader
          title={`Stream ${id}`}
          organizer={localParticipant?.name || 'Unknown'}
          listenerCount={participantCount}
          isLive={isConnected}
        />
      </div>

      {/* Error Display */}
      {error && (
        <div
          style={{
            padding: '1rem',
            marginBottom: '1.5rem',
            backgroundColor: '#7f1d1d',
            borderRadius: '0.5rem',
            border: '1px solid #ef4444',
          }}
          role="alert"
        >
          <strong>{t('streamPage.error')}:</strong> {error}
        </div>
      )}

      {/* Join Button */}
      {!isConnected && (
        <div style={{ marginBottom: '2rem' }}>
          <JoinStreamButton
            isConnected={isConnected}
            isConnecting={isConnecting}
            onJoin={handleConnect}
          />
        </div>
      )}

      {/* Connected View */}
      {isConnected && (
        <>
          {/* Main Content Area */}
          <div
            className="grid grid-cols-1 lg:grid-cols-[minmax(0,1fr)_350px] gap-6 mb-6"
          >
            {/* Left Column: Controls and Participants */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
              {/* Connection Quality and Share Buttons */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  flexWrap: 'wrap',
                  gap: '1rem',
                }}
              >
                <ConnectionIndicator quality={connectionQuality} />
                <ShareInviteButtons
                  streamUrl={streamUrl}
                  streamTitle={`Stream ${id}`}
                />
              </div>

              {/* Audio Controls */}
              <AudioControls
                isMuted={isLocalMuted}
                onToggleMute={toggleMute}
                onLeave={disconnect}
                onVolumeChange={setVolume}
              />

              {/* Audio Level Visualization Demo */}
              {localParticipant && (
                <div
                  style={{
                    padding: '1rem',
                    backgroundColor: '#1f2937',
                    borderRadius: '0.5rem',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '1rem',
                  }}
                >
                  <span style={{ fontSize: '0.875rem', color: '#9ca3af' }}>
                    Your Audio Level:
                  </span>
                  <AudioLevelVisualization
                    level={localParticipant.isSpeaking ? 0.7 : 0.1}
                    isMuted={localParticipant.isMuted}
                    size="medium"
                    showSpeaking={true}
                  />
                </div>
              )}

              {/* Participants */}
              <div>
                <h2
                  style={{
                    fontSize: '1.25rem',
                    fontWeight: 600,
                    marginBottom: '1rem',
                  }}
                >
                  {t('streamPage.participants')} ({participantCount})
                </h2>
                <ParticipantList
                  participants={participants}
                  localParticipant={localParticipant}
                />
              </div>
            </div>

            {/* Right Column: Chat Sidebar */}
            <div>
              <ChatSidebar
                messages={chatMessages}
                onSendMessage={handleSendMessage}
                maxHeight="calc(100vh - 250px)"
              />
            </div>
          </div>
        </>
      )}
    </div>
  );
};

