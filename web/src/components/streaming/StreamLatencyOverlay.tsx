/**
 * StreamLatencyOverlay Component
 * Debug overlay showing stream join latency measurements
 * Only visible in development builds
 */

import React from 'react';
import { useLatencyStore } from '../../stores/latencyStore';

export interface StreamLatencyOverlayProps {
  /** Whether to show the overlay */
  show?: boolean;
  /** Position on screen */
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';
}

/**
 * StreamLatencyOverlay Component
 * Displays measured join latency with segment breakdown
 */
export const StreamLatencyOverlay: React.FC<StreamLatencyOverlayProps> = ({
  show = true,
  position = 'top-right',
}) => {
  const { lastLatency, computeSegments } = useLatencyStore();
  
  // Don't render in production
  if (import.meta.env.PROD) {
    return null;
  }
  
  // Don't render if explicitly hidden
  if (!show) {
    return null;
  }
  
  // Don't render if no latency data available
  if (!lastLatency || lastLatency.joinClicked === null) {
    return null;
  }
  
  const segments = computeSegments(lastLatency);
  
  // Position styles
  const positionStyles: Record<string, React.CSSProperties> = {
    'top-left': { top: '1rem', left: '1rem' },
    'top-right': { top: '1rem', right: '1rem' },
    'bottom-left': { bottom: '1rem', left: '1rem' },
    'bottom-right': { bottom: '1rem', right: '1rem' },
  };
  
  return (
    <div
      className="stream-latency-overlay"
      style={{
        position: 'fixed',
        ...positionStyles[position],
        backgroundColor: 'rgba(0, 0, 0, 0.85)',
        color: '#fff',
        padding: '0.75rem 1rem',
        borderRadius: '0.5rem',
        fontFamily: 'monospace',
        fontSize: '0.875rem',
        zIndex: 9999,
        minWidth: '220px',
        boxShadow: '0 4px 6px rgba(0, 0, 0, 0.3)',
      }}
      role="status"
      aria-label="Stream join latency measurements"
    >
      <div style={{ fontWeight: 'bold', marginBottom: '0.5rem', fontSize: '0.9rem' }}>
        🎵 Join Latency
      </div>
      
      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
        {/* Total latency */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            padding: '0.25rem 0',
            borderBottom: '1px solid rgba(255, 255, 255, 0.2)',
            fontWeight: 'bold',
          }}
        >
          <span>Total:</span>
          <span
            style={{
              color: segments.total && segments.total < 2000 ? '#10b981' : '#ef4444',
            }}
          >
            {segments.total !== null ? `${segments.total.toFixed(0)}ms` : 'N/A'}
          </span>
        </div>
        
        {/* Segment breakdown */}
        <div style={{ fontSize: '0.8rem', opacity: 0.9 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', padding: '0.15rem 0' }}>
            <span>Token fetch:</span>
            <span>{segments.tokenFetch !== null ? `${segments.tokenFetch.toFixed(0)}ms` : 'N/A'}</span>
          </div>
          
          <div style={{ display: 'flex', justifyContent: 'space-between', padding: '0.15rem 0' }}>
            <span>Room connect:</span>
            <span>{segments.roomConnection !== null ? `${segments.roomConnection.toFixed(0)}ms` : 'N/A'}</span>
          </div>
          
          <div style={{ display: 'flex', justifyContent: 'space-between', padding: '0.15rem 0' }}>
            <span>Audio sub:</span>
            <span>{segments.audioSubscription !== null ? `${segments.audioSubscription.toFixed(0)}ms` : 'N/A'}</span>
          </div>
        </div>
        
        {/* Target indicator */}
        {segments.total !== null && (
          <div
            style={{
              marginTop: '0.5rem',
              padding: '0.25rem',
              fontSize: '0.75rem',
              textAlign: 'center',
              borderRadius: '0.25rem',
              backgroundColor: segments.total < 2000 ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)',
            }}
          >
            {segments.total < 2000 ? '✓ Within target (<2s)' : '⚠ Above target (≥2s)'}
          </div>
        )}
      </div>
    </div>
  );
};
