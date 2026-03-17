/**
 * Avatar Component
 * Optimized avatar images with size variants and loading states
 */

import React from 'react';
import { getAvatarUrl, type AvatarSize, AVATAR_SIZES } from '../utils/imageUrl';

export interface AvatarProps {
  /**
   * R2 object key for the avatar image
   */
  src?: string;
  /**
   * User's display name (used for alt text and fallback)
   */
  name: string;
  /**
   * Avatar size preset
   */
  size?: AvatarSize;
  /**
   * Additional CSS classes
   */
  className?: string;
  /**
   * Whether to show online indicator
   */
  online?: boolean;
  /**
   * Click handler
   */
  onClick?: () => void;
}

/**
 * Generate initials from name for fallback display
 */
function getInitials(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return '?';

  const parts = trimmed.split(/\s+/);
  if (parts.length === 1) {
    return parts[0].substring(0, 2).toUpperCase();
  }

  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

/**
 * Generate a consistent background color based on name
 */
function getColorFromName(name: string): { bgClass: string; textClass: string } {
  const colors = [
    { bgClass: 'bg-brand-primary', textClass: 'text-foreground' },
    { bgClass: 'bg-neon-magenta', textClass: 'text-foreground' },
    { bgClass: 'bg-status-error', textClass: 'text-foreground' },
    { bgClass: 'bg-brand-accent', textClass: 'text-background' },
    { bgClass: 'bg-neon-cyan', textClass: 'text-background' },
    { bgClass: 'bg-status-info', textClass: 'text-background' },
    { bgClass: 'bg-neon-green', textClass: 'text-background' },
    { bgClass: 'bg-status-warning', textClass: 'text-background' },
  ];

  // Generate a hash from the name
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }

  const index = Math.abs(hash) % colors.length;
  return colors[index];
}

/**
 * Avatar displays user profile pictures with optimized loading
 * and automatic fallback to initials
 */
export const Avatar: React.FC<AvatarProps> = ({
  src,
  name,
  size = 'md',
  className = '',
  online = false,
  onClick,
}) => {
  const [imageError, setImageError] = React.useState(false);
  const [imageLoaded, setImageLoaded] = React.useState(false);

  const dimension = AVATAR_SIZES[size];
  const initials = getInitials(name);
  const fallbackColor = getColorFromName(name);

  // Get optimized URLs for WebP and JPEG
  const webpUrl = src && !imageError ? getAvatarUrl(src, size, 'webp') : '';
  const jpegUrl = src && !imageError ? getAvatarUrl(src, size, 'jpeg') : '';

  const handleImageError = () => {
    setImageError(true);
  };

  const handleImageLoad = () => {
    setImageLoaded(true);
  };

  // Size-specific text sizing
  const textSizeMap: Record<AvatarSize, string> = {
    xs: 'text-xs',
    sm: 'text-sm',
    md: 'text-base',
    lg: 'text-xl',
    xl: 'text-2xl',
  };

  const textSize = textSizeMap[size];

  return (
    <div
      className={`
        relative inline-flex items-center justify-center
        rounded-full overflow-hidden
        ${onClick ? 'cursor-pointer' : ''}
        ${className}
      `}
      style={{ width: dimension, height: dimension }}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                onClick();
              }
            }
          : undefined
      }
    >
      {/* Image */}
      {src && !imageError ? (
        <picture>
          <source type="image/webp" srcSet={webpUrl} />
          <img
            src={jpegUrl}
            alt={`${name}'s avatar`}
            width={dimension}
            height={dimension}
            loading="lazy"
            decoding="async"
            onError={handleImageError}
            onLoad={handleImageLoad}
            className={`
              w-full h-full object-cover
              transition-opacity duration-300
              ${imageLoaded ? 'opacity-100' : 'opacity-0'}
            `}
          />
        </picture>
      ) : (
        /* Fallback to initials */
        <div
          className={`
            w-full h-full flex items-center justify-center
            ${fallbackColor.bgClass} ${fallbackColor.textClass} font-semibold
            ${textSize}
          `}
          aria-label={`${name}'s avatar (initials)`}
        >
          {initials}
        </div>
      )}

      {/* Loading skeleton */}
      {src && !imageError && !imageLoaded && (
        <div className="absolute inset-0 bg-background-hover animate-pulse" aria-hidden="true" />
      )}

      {/* Online indicator */}
      {online && (
        <div
          className="
            absolute bottom-0 right-0
            w-3 h-3 bg-status-success border-2 border-background
            rounded-full
          "
          aria-label="Online"
        />
      )}
    </div>
  );
};
