/**
 * MiniPlayer Component
 * Persistent bottom-docked mini-player for ongoing streams
 * Allows navigation while keeping audio playing
 */

import React, { useState, useRef, useEffect, memo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  useStreamingConnection,
  useStreamingAudio,
  useStreamingActions,
} from '../stores/streamingStore';

/**
 * MiniPlayer Component
 * Shows when user is connected to a stream
 * Persists across route changes
 */
export const MiniPlayer: React.FC = memo(() => {
  const { t } = useTranslation('streaming');
  const { isConnected, roomName, connectionQuality } = useStreamingConnection();
  const { volume, isLocalMuted, setVolume, toggleMute } = useStreamingAudio();
  const { disconnect } = useStreamingActions();
  
  const [showVolumeSlider, setShowVolumeSlider] = useState(false);
  const volumeRef = useRef<HTMLDivElement>(null);
  const miniPlayerRef = useRef<HTMLDivElement>(null);

  // Close volume slider on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (volumeRef.current && !volumeRef.current.contains(event.target as Node)) {
        setShowVolumeSlider(false);
      }
    };

    if (showVolumeSlider) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => {
        document.removeEventListener('mousedown', handleClickOutside);
      };
    }
  }, [showVolumeSlider]);

  // Don't render if not connected
  if (!isConnected || !roomName) {
    return null;
  }

  // Handle keyboard shortcuts
  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Always prevent default spacebar behavior (page scroll) when used as a shortcut
    if (e.key === ' ' || e.key === 'Spacebar') {
      e.preventDefault();
    }

    if (e.key === 'Escape' && showVolumeSlider) {
      setShowVolumeSlider(false);
    } else if (e.key === ' ' || e.key === 'Spacebar') {
      toggleMute();
    }
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newVolume = parseInt(e.target.value, 10);
    setVolume(newVolume);
  };

  // Connection quality indicator color
  const qualityColor = {
    excellent: '#10b981',
    good: '#f59e0b',
    poor: '#ef4444',
    unknown: '#6b7280',
  }[connectionQuality];

  return (
    <div
      ref={miniPlayerRef}
      className="mini-player"
      role="region"
      aria-label={t('miniPlayer.label')}
      onKeyDown={handleKeyDown}
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        right: 0,
        backgroundColor: '#1f2937',
        borderTop: '1px solid #374151',
        padding: '0.5rem 0.75rem',
        display: 'flex',
        alignItems: 'center',
        gap: '0.5rem',
        zIndex: 1000,
        boxShadow: '0 -2px 10px rgba(0, 0, 0, 0.3)',
      }}
    >
      {/* Connection Quality Indicator */}
      <div
        style={{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: qualityColor,
          flexShrink: 0,
        }}
        title={t(`miniPlayer.quality.${connectionQuality}`)}
        aria-label={t(`miniPlayer.quality.${connectionQuality}`)}
      />

      {/* Stream Info */}
      <div
        style={{
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            fontSize: '0.875rem',
            fontWeight: 600,
            color: 'white',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {t('miniPlayer.nowPlaying')}
        </div>
        <div
          style={{
            fontSize: '0.75rem',
            color: '#9ca3af',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {roomName}
        </div>
      </div>

      {/* Mute Toggle */}
      <button
        onClick={toggleMute}
        className="mini-player-mute-btn"
        aria-label={isLocalMuted ? t('miniPlayer.unmute') : t('miniPlayer.mute')}
        style={{
          padding: '0.5rem',
          fontSize: '1.125rem',
          borderRadius: '50%',
          border: 'none',
          cursor: 'pointer',
          backgroundColor: isLocalMuted ? '#ef4444' : '#10b981',
          color: 'white',
          minWidth: '44px',
          minHeight: '44px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          transition: 'all 0.2s ease',
          flexShrink: 0,
          touchAction: 'manipulation',
        }}
      >
        {isLocalMuted ? '🔇' : '🎤'}
      </button>

      {/* Volume Control */}
      <div
        ref={volumeRef}
        className="mini-player-volume"
        style={{
          position: 'relative',
          display: 'flex',
          alignItems: 'center',
          flexShrink: 0,
        }}
      >
        <button
          onClick={() => setShowVolumeSlider(!showVolumeSlider)}
          className="mini-player-volume-btn"
          aria-label={t('miniPlayer.volumeControl')}
          aria-expanded={showVolumeSlider}
          aria-controls="mini-player-volume-slider"
          style={{
            padding: '0.5rem',
            fontSize: '1rem',
            borderRadius: '0.375rem',
            border: 'none',
            cursor: 'pointer',
            backgroundColor: '#374151',
            color: 'white',
            minWidth: '44px',
            minHeight: '44px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            touchAction: 'manipulation',
            transition: 'all 0.2s ease',
          }}
        >
          {volume === 0 ? '🔈' : volume < 50 ? '🔉' : '🔊'}
        </button>

        {showVolumeSlider && (
          <div
            id="mini-player-volume-slider"
            className="mini-player-volume-slider"
            style={{
              position: 'absolute',
              left: '50%',
              bottom: '100%',
              transform: 'translateX(-50%)',
              marginBottom: '0.5rem',
              backgroundColor: '#1f2937',
              padding: '1rem',
              borderRadius: '0.5rem',
              boxShadow: '0 4px 6px rgba(0, 0, 0, 0.3)',
              minWidth: '180px',
            }}
          >
            <label
              htmlFor="mini-player-volume-input"
              style={{
                display: 'block',
                fontSize: '0.75rem',
                color: '#9ca3af',
                marginBottom: '0.5rem',
                textAlign: 'center',
              }}
            >
              {t('miniPlayer.volume')}: {volume}%
            </label>
            <input
              id="mini-player-volume-input"
              type="range"
              min="0"
              max="100"
              step="1"
              value={volume}
              onChange={handleVolumeChange}
              aria-label={t('miniPlayer.volumeSlider')}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={volume}
              aria-valuetext={`${volume}%`}
              className="volume-slider"
              style={{
                width: '100%',
                height: '0.25rem',
                borderRadius: '0.125rem',
                cursor: 'pointer',
              }}
            />
          </div>
        )}
      </div>

      {/* Leave Button */}
      <button
        onClick={disconnect}
        className="mini-player-leave-btn"
        aria-label={t('miniPlayer.leave')}
        style={{
          padding: '0.5rem 0.75rem',
          fontSize: '0.875rem',
          fontWeight: 600,
          borderRadius: '0.375rem',
          border: 'none',
          cursor: 'pointer',
          backgroundColor: '#dc2626',
          color: 'white',
          transition: 'all 0.2s ease',
          flexShrink: 0,
          whiteSpace: 'nowrap',
          minHeight: '44px',
          touchAction: 'manipulation',
        }}
      >
        {t('miniPlayer.leave')}
      </button>

      <style>{`
        .mini-player-mute-btn:hover,
        .mini-player-volume-btn:hover {
          transform: scale(1.05);
        }
        
        .mini-player-leave-btn:hover {
          background-color: #b91c1c;
        }
        
        .mini-player-mute-btn:focus,
        .mini-player-volume-btn:focus,
        .mini-player-leave-btn:focus {
          outline: 2px solid #60a5fa;
          outline-offset: 2px;
        }
        
        /* Custom range slider styling */
        #mini-player-volume-input::-webkit-slider-thumb {
          -webkit-appearance: none;
          appearance: none;
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: #60a5fa;
          cursor: pointer;
        }
        
        #mini-player-volume-input::-moz-range-thumb {
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: #60a5fa;
          cursor: pointer;
          border: none;
        }
        
        #mini-player-volume-input::-webkit-slider-runnable-track {
          width: 100%;
          height: 4px;
          cursor: pointer;
          background: #4b5563;
          border-radius: 2px;
        }
        
        #mini-player-volume-input::-moz-range-track {
          width: 100%;
          height: 4px;
          cursor: pointer;
          background: #4b5563;
          border-radius: 2px;
        }
      `}</style>
    </div>
  );
});

MiniPlayer.displayName = 'MiniPlayer';
