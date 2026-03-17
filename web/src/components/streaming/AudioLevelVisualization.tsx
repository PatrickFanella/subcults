/**
 * AudioLevelVisualization Component
 * Displays audio level bars for participants with real-time audio levels
 */

import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

export interface AudioLevelVisualizationProps {
  /** Audio level from 0 (silent) to 1 (max) */
  level: number;
  /** Whether the participant is muted */
  isMuted?: boolean;
  /** Size variant */
  size?: 'small' | 'medium' | 'large';
  /** Whether to show as speaking indicator */
  showSpeaking?: boolean;
}

const BAR_COUNT = 5;
const SMOOTHING_FACTOR = 0.3; // Lower = smoother, higher = more responsive
const MIN_THRESHOLD = 0.01; // Minimum level to show any bars

export const AudioLevelVisualization: React.FC<AudioLevelVisualizationProps> = ({
  level,
  isMuted = false,
  size = 'medium',
  showSpeaking = false,
}) => {
  const { t } = useTranslation('streaming');
  const [smoothedLevel, setSmoothedLevel] = useState(0);
  const animationFrameRef = useRef<number | undefined>(undefined);

  // Smooth the audio level changes
  useEffect(() => {
    let isAnimating = true;

    const animate = () => {
      setSmoothedLevel((prev) => {
        const target = isMuted ? 0 : level;
        const delta = target - prev;
        const newLevel = prev + delta * SMOOTHING_FACTOR;
        
        // Stop animating if we're close enough to the target
        if (Math.abs(delta) < 0.001) {
          isAnimating = false;
          return target;
        }
        
        return newLevel;
      });

      if (isAnimating) {
        animationFrameRef.current = requestAnimationFrame(animate);
      }
    };

    animationFrameRef.current = requestAnimationFrame(animate);

    return () => {
      isAnimating = false;
      if (animationFrameRef.current !== undefined) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [level, isMuted]);

  // Determine how many bars should be active
  const activeBarCount = Math.ceil(
    smoothedLevel < MIN_THRESHOLD ? 0 : smoothedLevel * BAR_COUNT
  );

  // Size configurations
  const sizeConfig = {
    small: { barWidth: '3px', barHeight: '12px', gap: '2px' },
    medium: { barWidth: '4px', barHeight: '16px', gap: '3px' },
    large: { barWidth: '5px', barHeight: '20px', gap: '4px' },
  }[size];

  // Color for active bars
  const getBarColor = (barIndex: number) => {
    if (isMuted || activeBarCount === 0) {
      return '#374151'; // Gray for muted/inactive
    }

    if (barIndex > activeBarCount) {
      return '#374151'; // Inactive bar
    }

    // Gradient from green to yellow to red
    const ratio = barIndex / BAR_COUNT;
    if (ratio < 0.6) {
      return '#10b981'; // Green
    } else if (ratio < 0.8) {
      return '#f59e0b'; // Yellow
    } else {
      return '#ef4444'; // Red
    }
  };

  const ariaLabel = isMuted
    ? t('audioLevel.muted')
    : activeBarCount > 0 && showSpeaking
    ? t('audioLevel.speaking')
    : t('audioLevel.silent');

  return (
    <div
      className="audio-level-visualization"
      role="img"
      aria-label={ariaLabel}
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        gap: sizeConfig.gap,
        height: sizeConfig.barHeight,
      }}
    >
      {Array.from({ length: BAR_COUNT }).map((_, index) => {
        const barNumber = index + 1;
        const isActive = barNumber <= activeBarCount;
        const barHeight = `${(barNumber / BAR_COUNT) * 100}%`;

        return (
          <div
            key={index}
            className={`audio-bar ${isActive ? 'active' : 'inactive'}`}
            style={{
              width: sizeConfig.barWidth,
              height: barHeight,
              backgroundColor: getBarColor(barNumber),
              borderRadius: '1px',
              transition: 'background-color 0.15s ease-out',
            }}
            aria-hidden="true"
          />
        );
      })}
    </div>
  );
};
