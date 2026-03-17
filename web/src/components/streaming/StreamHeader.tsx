/**
 * StreamHeader Component
 * Displays stream information including title, organizer, and listener count
 */

import React from 'react';
import { useTranslation } from 'react-i18next';

export interface StreamHeaderProps {
  title: string;
  organizer?: string;
  listenerCount: number;
  isLive?: boolean;
}

export const StreamHeader: React.FC<StreamHeaderProps> = ({
  title,
  organizer,
  listenerCount,
  isLive = false,
}) => {
  const { t } = useTranslation('streaming');

  return (
    <div
      className="stream-header"
      role="banner"
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '0.75rem',
        padding: '1.5rem',
        backgroundColor: '#1f2937',
        borderRadius: '0.75rem',
        border: isLive ? '2px solid #ef4444' : '2px solid #374151',
      }}
    >
      {/* Title and Live Indicator */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '0.75rem',
          flexWrap: 'wrap',
        }}
      >
        <h1
          style={{
            fontSize: '1.875rem',
            fontWeight: 700,
            color: 'white',
            margin: 0,
            flex: 1,
            minWidth: '200px',
          }}
        >
          {title}
        </h1>

        {isLive && (
          <div
            className="live-indicator"
            role="status"
            aria-label={t('streamHeader.live')}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
              padding: '0.375rem 0.75rem',
              backgroundColor: '#7f1d1d',
              borderRadius: '9999px',
              border: '1px solid #ef4444',
            }}
          >
            <div
              className="live-pulse"
              aria-hidden="true"
              style={{
                width: '0.625rem',
                height: '0.625rem',
                backgroundColor: '#ef4444',
                borderRadius: '50%',
                animation: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
              }}
            />
            <span
              style={{
                fontSize: '0.875rem',
                fontWeight: 600,
                color: '#fca5a5',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}
            >
              {t('streamHeader.live')}
            </span>
          </div>
        )}
      </div>

      {/* Organizer and Listener Count */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '1.5rem',
          flexWrap: 'wrap',
          fontSize: '0.875rem',
          color: '#9ca3af',
        }}
      >
        {organizer && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
            }}
          >
            <span aria-hidden="true">👤</span>
            <span>
              {t('streamHeader.organizer')}: <strong style={{ color: '#d1d5db' }}>{organizer}</strong>
            </span>
          </div>
        )}

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem',
          }}
          aria-label={t('streamHeader.listeners', { count: listenerCount })}
        >
          <span aria-hidden="true">👥</span>
          <span>
            <strong style={{ color: '#d1d5db' }}>{listenerCount}</strong> {t('streamHeader.listeners', { count: listenerCount })}
          </span>
        </div>
      </div>

      {/* Pulse animation */}
      <style>{`
        @keyframes pulse {
          0%, 100% {
            opacity: 1;
          }
          50% {
            opacity: 0.5;
          }
        }
      `}</style>
    </div>
  );
};
