/**
 * Image URL utilities for R2 CDN integration
 * Handles image transformations, format conversion, and responsive sizing
 */

/**
 * Image transformation options for R2/CDN
 */
export interface ImageTransformOptions {
  /**
   * Target width in pixels
   */
  width?: number;
  /**
   * Target height in pixels
   */
  height?: number;
  /**
   * Image format (webp, jpeg, png)
   */
  format?: 'webp' | 'jpeg' | 'png';
  /**
   * Quality (1-100, default 80 for balance)
   */
  quality?: number;
  /**
   * Fit mode for resizing
   */
  fit?: 'cover' | 'contain' | 'fill' | 'inside' | 'outside';
}

/**
 * Get R2 CDN base URL from environment
 * Falls back to API proxy if CDN URL is not configured
 */
function getR2BaseUrl(): string {
  // In production, this would be the CloudFlare R2 CDN URL
  // Example: https://cdn.subcults.com or https://pub-xxxxx.r2.dev
  const cdnUrl = import.meta.env.VITE_R2_CDN_URL;
  
  if (cdnUrl) {
    return cdnUrl;
  }
  
  // Fallback to API proxy for development
  return '/api/media';
}

/**
 * Build image URL with transformations for R2/CDN
 * 
 * @param key - R2 object key (e.g., "posts/temp/uuid.jpg")
 * @param options - Transformation options
 * @returns Transformed image URL
 * 
 * @example
 * ```ts
 * // Get WebP version at 640px width
 * buildImageUrl('posts/123/avatar.jpg', { width: 640, format: 'webp' })
 * // => '/api/media/posts/123/avatar.jpg?w=640&f=webp&q=80'
 * ```
 */
export function buildImageUrl(
  key: string,
  options: ImageTransformOptions = {}
): string {
  if (!key) {
    return '';
  }

  const baseUrl = getR2BaseUrl();
  const params = new URLSearchParams();

  // Add transformation parameters
  if (options.width) {
    params.set('w', options.width.toString());
  }
  if (options.height) {
    params.set('h', options.height.toString());
  }
  if (options.format) {
    params.set('f', options.format);
  }
  if (options.quality) {
    params.set('q', options.quality.toString());
  } else if (options.format === 'webp' || options.format === 'jpeg') {
    // Default quality for compressed formats
    params.set('q', '80');
  }
  if (options.fit) {
    params.set('fit', options.fit);
  }

  const queryString = params.toString();
  const url = `${baseUrl}/${key}`;

  return queryString ? `${url}?${queryString}` : url;
}

/**
 * Generate responsive srcset for multiple widths
 * 
 * @param key - R2 object key
 * @param widths - Array of target widths
 * @param format - Image format (webp or jpeg)
 * @returns srcset string
 * 
 * @example
 * ```ts
 * generateSrcSet('posts/123/cover.jpg', [640, 1024, 1920], 'webp')
 * // => '/api/media/posts/123/cover.jpg?w=640&f=webp&q=80 640w, ...'
 * ```
 */
export function generateSrcSet(
  key: string,
  widths: number[] = [320, 640, 768, 1024, 1280, 1536, 2048],
  format: 'webp' | 'jpeg' = 'jpeg',
  quality: number = 80
): string {
  return widths
    .map((width) => {
      const url = buildImageUrl(key, { width, format, quality });
      return `${url} ${width}w`;
    })
    .join(', ');
}

/**
 * Avatar image sizes
 */
export const AVATAR_SIZES = {
  xs: 32,
  sm: 48,
  md: 64,
  lg: 96,
  xl: 128,
} as const;

export type AvatarSize = keyof typeof AVATAR_SIZES;

/**
 * Get avatar URL with size transformation
 * 
 * @param key - R2 object key for avatar
 * @param size - Avatar size preset
 * @param format - Image format
 * @returns Transformed avatar URL
 */
export function getAvatarUrl(
  key: string | undefined,
  size: AvatarSize = 'md',
  format: 'webp' | 'jpeg' = 'webp'
): string {
  if (!key) {
    return '';
  }

  const dimension = AVATAR_SIZES[size];
  
  return buildImageUrl(key, {
    width: dimension,
    height: dimension,
    format,
    quality: 85, // Higher quality for avatars
    fit: 'cover',
  });
}

/**
 * Scene cover image sizes for responsive layouts
 */
export const COVER_SIZES = {
  thumbnail: 320,
  small: 640,
  medium: 1024,
  large: 1920,
  xlarge: 2560,
} as const;

export type CoverSize = keyof typeof COVER_SIZES;

/**
 * Get scene cover URL with size transformation
 * 
 * @param key - R2 object key for cover image
 * @param size - Cover size preset
 * @param format - Image format
 * @returns Transformed cover URL
 */
export function getCoverUrl(
  key: string | undefined,
  size: CoverSize = 'medium',
  format: 'webp' | 'jpeg' = 'webp'
): string {
  if (!key) {
    return '';
  }

  const width = COVER_SIZES[size];
  
  return buildImageUrl(key, {
    width,
    format,
    quality: 80,
    fit: 'cover',
  });
}

/**
 * Check if browser supports WebP format
 * Uses feature detection with canvas
 */
export function supportsWebP(): boolean {
  // Check if we're in a browser environment
  if (typeof document === 'undefined') {
    return false;
  }

  // Try to create a WebP image and check if it's supported
  const canvas = document.createElement('canvas');
  if (canvas.getContext && canvas.getContext('2d')) {
    // Check if canvas.toDataURL supports WebP
    return canvas.toDataURL('image/webp').indexOf('data:image/webp') === 0;
  }
  
  return false;
}

/**
 * Get optimal image format based on browser support
 */
export function getOptimalFormat(): 'webp' | 'jpeg' {
  return supportsWebP() ? 'webp' : 'jpeg';
}
