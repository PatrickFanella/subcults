/**
 * AudioControls Component
 * Controls for muting/unmuting, adjusting volume, and leaving the room
 */

import React, { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

export interface AudioControlsProps {
  isMuted: boolean;
  onToggleMute: () => void;
  onLeave: () => void;
  onVolumeChange: (volume: number) => void;
  disabled?: boolean;
}

export const AudioControls: React.FC<AudioControlsProps> = ({
  isMuted,
  onToggleMute,
  onLeave,
  onVolumeChange,
  disabled = false,
}) => {
  const { t } = useTranslation('streaming');
  const [volume, setVolume] = useState(100);
  const [showVolumeSlider, setShowVolumeSlider] = useState(false);
  const volumeRef = useRef<HTMLDivElement>(null);

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newVolume = parseInt(e.target.value, 10);
    setVolume(newVolume);
    onVolumeChange(newVolume);
  };

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

  // Handle keyboard navigation for volume slider
  const handleVolumeKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setShowVolumeSlider(false);
    }
  };

  return (
    <div
      className="audio-controls"
      role="group"
      aria-label="Audio controls"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '1rem',
        padding: '1rem',
        backgroundColor: '#1f2937',
        borderRadius: '0.5rem',
      }}
    >
      {/* Mute/Unmute Button */}
      <button
        onClick={onToggleMute}
        disabled={disabled}
        className={`mute-button ${isMuted ? 'muted' : 'unmuted'}`}
        aria-label={isMuted ? t('audioControls.unmute') : t('audioControls.mute')}
        style={{
          padding: '0.75rem',
          fontSize: '1.25rem',
          borderRadius: '50%',
          border: 'none',
          cursor: disabled ? 'not-allowed' : 'pointer',
          backgroundColor: isMuted ? '#ef4444' : '#10b981',
          color: 'white',
          width: '3rem',
          height: '3rem',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          transition: 'all 0.2s ease',
          opacity: disabled ? 0.5 : 1,
        }}
        onMouseEnter={(e) => {
          if (!disabled) {
            e.currentTarget.style.transform = 'scale(1.05)';
          }
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.transform = 'scale(1)';
        }}
      >
        {isMuted ? '🔇' : '🎤'}
      </button>

      {/* Volume Control */}
      <div
        ref={volumeRef}
        className="volume-control"
        style={{
          position: 'relative',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem',
        }}
      >
        <button
          onClick={() => setShowVolumeSlider(!showVolumeSlider)}
          disabled={disabled}
          className="volume-button"
          aria-label={t('audioControls.volumeControl')}
          aria-expanded={showVolumeSlider}
          style={{
            padding: '0.5rem',
            fontSize: '1.25rem',
            borderRadius: '0.375rem',
            border: 'none',
            cursor: disabled ? 'not-allowed' : 'pointer',
            backgroundColor: '#374151',
            color: 'white',
            width: '2.5rem',
            height: '2.5rem',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            transition: 'all 0.2s ease',
            opacity: disabled ? 0.5 : 1,
          }}
        >
          🔊
        </button>

        {showVolumeSlider && (
          <div
            className="volume-slider-container"
            onKeyDown={handleVolumeKeyDown}
            style={{
              position: 'absolute',
              left: '0',
              bottom: '100%',
              marginBottom: '0.5rem',
              backgroundColor: '#1f2937',
              padding: '1rem',
              borderRadius: '0.5rem',
              boxShadow: '0 4px 6px rgba(0, 0, 0, 0.3)',
              minWidth: '200px',
            }}
          >
            <label
              htmlFor="volume-slider"
              style={{
                display: 'block',
                fontSize: '0.875rem',
                color: '#9ca3af',
                marginBottom: '0.5rem',
              }}
            >
              {t('audioControls.volumeLabel')}: {volume}%
            </label>
            <input
              id="volume-slider"
              type="range"
              min="0"
              max="100"
              value={volume}
              onChange={handleVolumeChange}
              disabled={disabled}
              aria-label="Volume slider"
              className="volume-slider"
              style={{
                width: '100%',
                height: '0.25rem',
                borderRadius: '0.125rem',
                opacity: disabled ? 0.5 : 1,
                cursor: disabled ? 'not-allowed' : 'pointer',
              }}
            />
          </div>
        )}
      </div>

      {/* Spacer */}
      <div style={{ flex: 1 }} />

      {/* Leave Button */}
      <button
        onClick={onLeave}
        disabled={disabled}
        className="leave-button"
        aria-label={t('audioControls.leaveRoom')}
        style={{
          padding: '0.75rem 1.5rem',
          fontSize: '0.875rem',
          fontWeight: 600,
          borderRadius: '0.5rem',
          border: 'none',
          cursor: disabled ? 'not-allowed' : 'pointer',
          backgroundColor: '#dc2626',
          color: 'white',
          transition: 'all 0.2s ease',
          opacity: disabled ? 0.5 : 1,
        }}
        onMouseEnter={(e) => {
          if (!disabled) {
            e.currentTarget.style.backgroundColor = '#b91c1c';
          }
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = '#dc2626';
        }}
      >
        {t('audioControls.leave')}
      </button>
    </div>
  );
};
