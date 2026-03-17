/**
 * ConnectionIndicator Component
 * Shows connection quality based on LiveKit stats
 */

import React from 'react';
import { useTranslation } from 'react-i18next';
import type { ConnectionQuality } from '../../types/streaming';

export interface ConnectionIndicatorProps {
  quality: ConnectionQuality;
  showLabel?: boolean;
}

/**
 * Get color for connection quality
 */
function getQualityColor(quality: ConnectionQuality): string {
  switch (quality) {
    case 'excellent':
      return '#10b981'; // green
    case 'good':
      return '#f59e0b'; // amber
    case 'poor':
      return '#ef4444'; // red
    case 'unknown':
    default:
      return '#6b7280'; // gray
  }
}

/**
 * Get translation key for connection quality
 */
function getQualityKey(quality: ConnectionQuality): string {
  switch (quality) {
    case 'excellent':
      return 'connectionIndicator.excellent';
    case 'good':
      return 'connectionIndicator.good';
    case 'poor':
      return 'connectionIndicator.poor';
    case 'unknown':
    default:
      return 'connectionIndicator.unknown';
  }
}

/**
 * Get number of bars to show (1-3)
 */
function getQualityBars(quality: ConnectionQuality): number {
  switch (quality) {
    case 'excellent':
      return 3;
    case 'good':
      return 2;
    case 'poor':
      return 1;
    case 'unknown':
    default:
      return 0;
  }
}

export const ConnectionIndicator: React.FC<ConnectionIndicatorProps> = ({
  quality,
  showLabel = true,
}) => {
  const { t } = useTranslation('streaming');
  const color = getQualityColor(quality);
  const label = t(getQualityKey(quality));
  const bars = getQualityBars(quality);

  return (
    <div
      className="connection-indicator"
      role="status"
      aria-label={`${t('connectionIndicator.quality')}: ${label}`}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '0.5rem',
        padding: '0.5rem 0.75rem',
        backgroundColor: '#1f2937',
        borderRadius: '0.375rem',
      }}
    >
      {/* Signal bars */}
      <div
        className="signal-bars"
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: '2px',
          height: '1rem',
        }}
        aria-hidden="true"
      >
        {[1, 2, 3].map((barIndex) => (
          <div
            key={barIndex}
            className={`signal-bar ${barIndex <= bars ? 'active' : 'inactive'}`}
            style={{
              width: '4px',
              height: `${barIndex * 33.33}%`,
              backgroundColor: barIndex <= bars ? color : '#374151',
              borderRadius: '1px',
              transition: 'all 0.3s ease',
            }}
          />
        ))}
      </div>

      {/* Label */}
      {showLabel && (
        <span
          style={{
            fontSize: '0.875rem',
            fontWeight: 500,
            color: color,
          }}
        >
          {label}
        </span>
      )}
    </div>
  );
};
