/**
 * OptimizedImage Component
 * Provides responsive, lazy-loaded images with WebP format and JPEG fallback
 */

import React, { useState, useEffect, useRef } from 'react';
import { generateSrcSet as generateImageSrcSet } from '../utils/imageUrl';

export interface OptimizedImageProps {
  /**
   * Image source URL (base URL without format-specific extensions)
   */
  src: string;
  /**
   * Alt text for accessibility
   */
  alt: string;
  /**
   * Image width (for aspect ratio calculation)
   */
  width?: number;
  /**
   * Image height (for aspect ratio calculation)
   */
  height?: number;
  /**
   * CSS class name
   */
  className?: string;
  /**
   * Enable lazy loading (default: true)
   */
  lazy?: boolean;
  /**
   * Sizes attribute for responsive images
   * Example: "(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
   */
  sizes?: string;
  /**
   * Custom srcset for responsive images
   * If not provided, will generate based on src
   */
  srcSet?: string;
  /**
   * WebP srcset (optional, auto-generated if not provided)
   */
  webpSrcSet?: string;
  /**
   * Callback when image loads successfully
   */
  onLoad?: () => void;
  /**
   * Callback when image fails to load
   */
  onError?: () => void;
  /**
   * Object fit CSS property
   */
  objectFit?: 'contain' | 'cover' | 'fill' | 'none' | 'scale-down';
  /**
   * Object position CSS property
   */
  objectPosition?: string;
  /**
   * Priority loading (disables lazy loading for above-fold images)
   */
  priority?: boolean;
}

/**
 * OptimizedImage provides automatic WebP support with JPEG fallback,
 * responsive srcsets, and lazy loading
 */
export const OptimizedImage: React.FC<OptimizedImageProps> = ({
  src,
  alt,
  width,
  height,
  className = '',
  lazy = true,
  sizes,
  srcSet: customSrcSet,
  webpSrcSet: customWebpSrcSet,
  onLoad,
  onError,
  objectFit = 'cover',
  objectPosition = 'center',
  priority = false,
}) => {
  const [isLoaded, setIsLoaded] = useState(false);
  const [hasError, setHasError] = useState(false);
  const [isInView, setIsInView] = useState(!lazy || priority);
  const imgRef = useRef<HTMLImageElement>(null);

  // Lazy loading with Intersection Observer
  useEffect(() => {
    if (!lazy || priority || isInView) return;

    const currentElement = imgRef.current;
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
        rootMargin: '50px', // Start loading 50px before image enters viewport
        threshold: 0.01,
      }
    );

    observer.observe(currentElement);

    return () => {
      observer.disconnect();
    };
  }, [lazy, priority, isInView]);

  const handleLoad = () => {
    setIsLoaded(true);
    onLoad?.();
  };

  const handleError = () => {
    setHasError(true);
    onError?.();
  };

  // Calculate aspect ratio for layout stability (prevent CLS)
  const aspectRatio = width && height ? `${width} / ${height}` : undefined;

  // Generate srcsets if not provided
  const jpegSrcSet =
    customSrcSet || generateImageSrcSet(src, [320, 640, 768, 1024, 1280, 1536, 2048], 'jpeg', 80);
  const webpSrcSetValue =
    customWebpSrcSet ||
    generateImageSrcSet(src, [320, 640, 768, 1024, 1280, 1536, 2048], 'webp', 80);

  // Default sizes if not provided
  const sizesValue = sizes || '100vw';

  return (
    <picture>
      {/* WebP source for modern browsers */}
      {isInView && !hasError && (
        <source type="image/webp" srcSet={webpSrcSetValue} sizes={sizesValue} />
      )}

      {/* JPEG fallback for older browsers */}
      {!hasError && (
        <img
          ref={imgRef}
          src={isInView ? src : undefined}
          srcSet={isInView ? jpegSrcSet : undefined}
          sizes={isInView ? sizesValue : undefined}
          alt={alt}
          width={width}
          height={height}
          loading={priority ? 'eager' : lazy ? 'lazy' : 'eager'}
          decoding="async"
          onLoad={handleLoad}
          onError={handleError}
          className={className}
          style={{
            objectFit,
            objectPosition,
            aspectRatio,
            opacity: isLoaded ? 1 : 0,
            transition: 'opacity 0.3s ease-in-out',
          }}
        />
      )}

      {/* Error state */}
      {hasError && (
        <div
          className={`flex items-center justify-center bg-background-secondary ${className}`}
          style={{ aspectRatio }}
          role="img"
          aria-label={`Failed to load image: ${alt}`}
        >
          <svg
            className="w-12 h-12 text-foreground-muted"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
        </div>
      )}
    </picture>
  );
};
