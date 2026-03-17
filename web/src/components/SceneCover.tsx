/**
 * SceneCover Component
 * Optimized scene cover images with responsive sizing and loading states
 */

import React from 'react';
import { getCoverUrl, generateSrcSet, type CoverSize } from '../utils/imageUrl';

export interface SceneCoverProps {
  /**
   * R2 object key for the cover image
   */
  src?: string;
  /**
   * Scene name (used for alt text)
   */
  sceneName: string;
  /**
   * Cover size preset
   */
  size?: CoverSize;
  /**
   * Additional CSS classes
   */
  className?: string;
  /**
   * Aspect ratio (default: 16/9)
   */
  aspectRatio?: string;
  /**
   * Priority loading (disable lazy loading for above-fold images)
   */
  priority?: boolean;
  /**
   * Sizes attribute for responsive images
   */
  sizes?: string;
  /**
   * Click handler
   */
  onClick?: () => void;
  /**
   * Overlay gradient for better text readability
   */
  overlay?: boolean;
}

/**
 * SceneCover displays optimized cover images for scenes
 * with responsive sizing and WebP support
 */
export const SceneCover: React.FC<SceneCoverProps> = ({
  src,
  sceneName,
  size = 'medium',
  className = '',
  aspectRatio = '16 / 9',
  priority = false,
  sizes = '100vw',
  onClick,
  overlay = false,
}) => {
  const [imageError, setImageError] = React.useState(false);
  const [imageLoaded, setImageLoaded] = React.useState(false);
  const [isInView, setIsInView] = React.useState(priority);
  const containerRef = React.useRef<HTMLDivElement>(null);

  // Lazy loading with Intersection Observer
  React.useEffect(() => {
    if (priority || isInView) return;

    const currentElement = containerRef.current;
    if (!currentElement) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsInView(true);
            observer.disconnect();
          }
        });
      },
      {
        rootMargin: '50px',
        threshold: 0.01,
      }
    );

    observer.observe(currentElement);

    return () => {
      observer.disconnect();
    };
  }, [priority, isInView]);

  const handleImageError = () => {
    setImageError(true);
  };

  const handleImageLoad = () => {
    setImageLoaded(true);
  };

  // Generate responsive srcsets
  const webpSrcSet =
    src && !imageError && isInView
      ? generateSrcSet(src, [320, 640, 768, 1024, 1280, 1536, 2048], 'webp', 80)
      : '';
  const jpegSrcSet =
    src && !imageError && isInView
      ? generateSrcSet(src, [320, 640, 768, 1024, 1280, 1536, 2048], 'jpeg', 80)
      : '';
  const jpegUrl = src && !imageError ? getCoverUrl(src, size, 'jpeg') : '';

  return (
    <div
      ref={containerRef}
      className={`
        relative overflow-hidden bg-background-secondary
        ${onClick ? 'cursor-pointer' : ''}
        ${className}
      `}
      style={{ aspectRatio }}
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
      {/* Cover Image */}
      {src && !imageError && isInView ? (
        <picture>
          <source type="image/webp" srcSet={webpSrcSet} sizes={sizes} />
          <source type="image/jpeg" srcSet={jpegSrcSet} sizes={sizes} />
          <img
            src={jpegUrl}
            alt={`${sceneName} cover`}
            loading={priority ? 'eager' : 'lazy'}
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
        /* Fallback placeholder */
        <div className="w-full h-full flex items-center justify-center">
          <svg
            className="w-16 h-16 text-foreground-muted"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
        </div>
      )}

      {/* Loading skeleton */}
      {src && !imageError && !imageLoaded && isInView && (
        <div className="absolute inset-0 bg-background-hover animate-pulse" aria-hidden="true" />
      )}

      {/* Overlay gradient */}
      {overlay && imageLoaded && (
        <div
          className="
            absolute inset-0
            bg-gradient-to-t from-black/60 via-black/20 to-transparent
          "
          aria-hidden="true"
        />
      )}
    </div>
  );
};
