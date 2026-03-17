/**
 * ParticipantList Component
 * Displays list of participants in the audio room with their mute state
 */

import React from 'react';
import { useTranslation } from 'react-i18next';
import type { Participant } from '../../types/streaming';

export interface ParticipantListProps {
  participants: Participant[];
  localParticipant: Participant | null;
}

export const ParticipantList: React.FC<ParticipantListProps> = ({
  participants,
  localParticipant,
}) => {
  const { t } = useTranslation('streaming');
  
  const allParticipants = localParticipant
    ? [localParticipant, ...participants]
    : participants;

  if (allParticipants.length === 0) {
    return (
      <div
        className="participant-list-empty"
        style={{
          padding: '1rem',
          textAlign: 'center',
          color: '#9ca3af',
          fontStyle: 'italic',
        }}
      >
        {t('participantList.empty')}
      </div>
    );
  }

  return (
    <div
      className="participant-list"
      role="list"
      aria-label="Room participants"
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '0.5rem',
        maxHeight: '400px',
        overflowY: 'auto',
        padding: '0.5rem',
        // Custom scrollbar styling for dark theme
        scrollbarWidth: 'thin',
        scrollbarColor: '#4b5563 #1f2937',
      }}
    >
      {allParticipants.map((participant) => (
        <ParticipantItem key={participant.identity} participant={participant} />
      ))}
      <style>{`
        .participant-list::-webkit-scrollbar {
          width: 8px;
        }
        .participant-list::-webkit-scrollbar-track {
          background: #1f2937;
          border-radius: 4px;
        }
        .participant-list::-webkit-scrollbar-thumb {
          background: #4b5563;
          border-radius: 4px;
        }
        .participant-list::-webkit-scrollbar-thumb:hover {
          background: #6b7280;
        }
      `}</style>
    </div>
  );
};

interface ParticipantItemProps {
  participant: Participant;
}

const ParticipantItem: React.FC<ParticipantItemProps> = ({ participant }) => {
  const { t } = useTranslation('streaming');
  
  return (
    <div
      className={`participant-item ${participant.isSpeaking ? 'speaking' : ''}`}
      role="listitem"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '0.75rem',
        padding: '0.75rem',
        backgroundColor: participant.isSpeaking ? '#1e3a8a' : '#1f2937',
        borderRadius: '0.5rem',
        border: participant.isSpeaking ? '2px solid #3b82f6' : '2px solid transparent',
        transition: 'all 0.2s ease',
      }}
    >
      {/* Avatar/Icon */}
      <div
        className="participant-avatar"
        style={{
          width: '2.5rem',
          height: '2.5rem',
          borderRadius: '50%',
          backgroundColor: '#4b5563',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '1.25rem',
          fontWeight: 600,
          color: 'white',
        }}
        aria-hidden="true"
      >
        {participant.name?.charAt(0).toUpperCase() || '?'}
      </div>

      {/* Name and status */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: '0.875rem',
            fontWeight: 500,
            color: 'white',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {participant.name}
          {participant.isLocal && (
            <span
              style={{
                marginLeft: '0.5rem',
                fontSize: '0.75rem',
                color: '#9ca3af',
                fontWeight: 400,
              }}
            >
              ({t('participantList.you')})
            </span>
          )}
        </div>
        <div
          style={{
            fontSize: '0.75rem',
            color: participant.isSpeaking ? '#60a5fa' : '#6b7280',
            marginTop: '0.125rem',
          }}
        >
          {participant.isSpeaking ? t('participantList.speaking') : ''}
        </div>
      </div>

      {/* Mute indicator */}
      <div
        className="mute-indicator"
        aria-label={participant.isMuted ? 'Muted' : 'Unmuted'}
        style={{
          fontSize: '1.25rem',
          color: participant.isMuted ? '#ef4444' : '#10b981',
        }}
      >
        {participant.isMuted ? '🔇' : '🎤'}
      </div>
    </div>
  );
};
