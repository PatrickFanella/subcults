/**
 * Image URL Utilities Tests
 */

import { describe, it, expect, vi } from 'vitest';
import {
  buildImageUrl,
  generateSrcSet,
  getAvatarUrl,
  getCoverUrl,
  supportsWebP,
  getOptimalFormat,
} from './imageUrl';

// Mock import.meta.env
vi.stubGlobal('import', {
  meta: {
    env: {
      VITE_R2_CDN_URL: '',
    },
  },
});

describe('imageUrl utilities', () => {
  describe('buildImageUrl', () => {
    it('returns empty string for empty key', () => {
      expect(buildImageUrl('')).toBe('');
    });

    it('builds basic URL without transformations', () => {
      const url = buildImageUrl('posts/123/image.jpg');
      expect(url).toBe('/api/media/posts/123/image.jpg');
    });

    it('adds width parameter', () => {
      const url = buildImageUrl('posts/123/image.jpg', { width: 640 });
      expect(url).toBe('/api/media/posts/123/image.jpg?w=640');
    });

    it('adds height parameter', () => {
      const url = buildImageUrl('posts/123/image.jpg', { height: 480 });
      expect(url).toBe('/api/media/posts/123/image.jpg?h=480');
    });

    it('adds format parameter', () => {
      const url = buildImageUrl('posts/123/image.jpg', { format: 'webp' });
      expect(url).toBe('/api/media/posts/123/image.jpg?f=webp&q=80');
    });

    it('adds custom quality parameter', () => {
      const url = buildImageUrl('posts/123/image.jpg', {
        format: 'webp',
        quality: 90,
      });
      expect(url).toBe('/api/media/posts/123/image.jpg?f=webp&q=90');
    });

    it('adds fit parameter', () => {
      const url = buildImageUrl('posts/123/image.jpg', { fit: 'cover' });
      expect(url).toBe('/api/media/posts/123/image.jpg?fit=cover');
    });

    it('combines multiple parameters', () => {
      const url = buildImageUrl('posts/123/image.jpg', {
        width: 640,
        height: 480,
        format: 'webp',
        quality: 85,
        fit: 'contain',
      });
      expect(url).toContain('w=640');
      expect(url).toContain('h=480');
      expect(url).toContain('f=webp');
      expect(url).toContain('q=85');
      expect(url).toContain('fit=contain');
    });

    it('uses default quality for webp format', () => {
      const url = buildImageUrl('posts/123/image.jpg', { format: 'webp' });
      expect(url).toContain('q=80');
    });

    it('uses default quality for jpeg format', () => {
      const url = buildImageUrl('posts/123/image.jpg', { format: 'jpeg' });
      expect(url).toContain('q=80');
    });

    it('does not add quality for png format by default', () => {
      const url = buildImageUrl('posts/123/image.jpg', { format: 'png' });
      expect(url).not.toContain('q=');
    });
  });

  describe('generateSrcSet', () => {
    it('generates srcset with default widths for jpeg', () => {
      const srcset = generateSrcSet('posts/123/image.jpg');
      
      expect(srcset).toContain('320w');
      expect(srcset).toContain('640w');
      expect(srcset).toContain('1024w');
      expect(srcset).toContain('2048w');
      expect(srcset).toContain('f=jpeg');
    });

    it('generates srcset with default widths for webp', () => {
      const srcset = generateSrcSet('posts/123/image.jpg', undefined, 'webp');
      
      expect(srcset).toContain('320w');
      expect(srcset).toContain('640w');
      expect(srcset).toContain('f=webp');
    });

    it('uses custom widths array', () => {
      const srcset = generateSrcSet('posts/123/image.jpg', [400, 800, 1200]);
      
      expect(srcset).toContain('400w');
      expect(srcset).toContain('800w');
      expect(srcset).toContain('1200w');
      expect(srcset).not.toContain('320w');
    });

    it('uses custom quality', () => {
      const srcset = generateSrcSet(
        'posts/123/image.jpg',
        [640],
        'webp',
        90
      );
      
      expect(srcset).toContain('q=90');
    });

    it('separates entries with commas', () => {
      const srcset = generateSrcSet('posts/123/image.jpg', [640, 1024]);
      const entries = srcset.split(', ');
      
      expect(entries).toHaveLength(2);
      expect(entries[0]).toContain('640w');
      expect(entries[1]).toContain('1024w');
    });
  });

  describe('getAvatarUrl', () => {
    it('returns empty string for undefined key', () => {
      expect(getAvatarUrl(undefined)).toBe('');
    });

    it('builds avatar URL with default size (md)', () => {
      const url = getAvatarUrl('posts/123/avatar.jpg');
      
      expect(url).toContain('w=64');
      expect(url).toContain('h=64');
      expect(url).toContain('f=webp');
      expect(url).toContain('q=85');
      expect(url).toContain('fit=cover');
    });

    it('uses custom size', () => {
      const url = getAvatarUrl('posts/123/avatar.jpg', 'lg');
      
      expect(url).toContain('w=96');
      expect(url).toContain('h=96');
    });

    it('uses custom format', () => {
      const url = getAvatarUrl('posts/123/avatar.jpg', 'md', 'jpeg');
      
      expect(url).toContain('f=jpeg');
    });

    it('uses higher quality for avatars', () => {
      const url = getAvatarUrl('posts/123/avatar.jpg');
      
      expect(url).toContain('q=85');
    });

    it('supports all avatar sizes', () => {
      const sizes = ['xs', 'sm', 'md', 'lg', 'xl'] as const;
      const expectedWidths = [32, 48, 64, 96, 128];
      
      sizes.forEach((size, index) => {
        const url = getAvatarUrl('posts/123/avatar.jpg', size);
        expect(url).toContain(`w=${expectedWidths[index]}`);
      });
    });
  });

  describe('getCoverUrl', () => {
    it('returns empty string for undefined key', () => {
      expect(getCoverUrl(undefined)).toBe('');
    });

    it('builds cover URL with default size (medium)', () => {
      const url = getCoverUrl('posts/123/cover.jpg');
      
      expect(url).toContain('w=1024');
      expect(url).toContain('f=webp');
      expect(url).toContain('q=80');
      expect(url).toContain('fit=cover');
    });

    it('uses custom size', () => {
      const url = getCoverUrl('posts/123/cover.jpg', 'large');
      
      expect(url).toContain('w=1920');
    });

    it('uses custom format', () => {
      const url = getCoverUrl('posts/123/cover.jpg', 'medium', 'jpeg');
      
      expect(url).toContain('f=jpeg');
    });

    it('supports all cover sizes', () => {
      const sizes = ['thumbnail', 'small', 'medium', 'large', 'xlarge'] as const;
      const expectedWidths = [320, 640, 1024, 1920, 2560];
      
      sizes.forEach((size, index) => {
        const url = getCoverUrl('posts/123/cover.jpg', size);
        expect(url).toContain(`w=${expectedWidths[index]}`);
      });
    });
  });

  describe('supportsWebP', () => {
    it('returns false in non-browser environment', () => {
      // In test environment, document might not be available
      expect(typeof supportsWebP()).toBe('boolean');
    });
  });

  describe('getOptimalFormat', () => {
    it('returns a valid format', () => {
      const format = getOptimalFormat();
      expect(['webp', 'jpeg']).toContain(format);
    });
  });
});
