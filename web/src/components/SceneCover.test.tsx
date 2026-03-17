/**
 * SceneCover Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SceneCover } from './SceneCover';

// Mock the imageUrl utility
vi.mock('../utils/imageUrl', () => ({
  getCoverUrl: (key: string, size: string, format: string) =>
    `/api/media/${key}?size=${size}&format=${format}`,
  generateSrcSet: (key: string, widths: number[], format: string) =>
    widths.map((w) => `/api/media/${key}?w=${w}&f=${format} ${w}w`).join(', '),
}));

describe('SceneCover', () => {
  let intersectionObserverCallback: IntersectionObserverCallback;
  let observeMock: ReturnType<typeof vi.fn>;
  let disconnectMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Mock IntersectionObserver
    observeMock = vi.fn();
    disconnectMock = vi.fn();

    global.IntersectionObserver = class IntersectionObserver {
      constructor(callback: IntersectionObserverCallback) {
        intersectionObserverCallback = callback;
      }
      observe = observeMock;
      disconnect = disconnectMock;
      unobserve = vi.fn();
      takeRecords = vi.fn(() => []);
      root = null;
      rootMargin = '';
      thresholds = [];
    } as any;
  });

  it('renders with scene name', () => {
    const { container } = render(
      <SceneCover sceneName="Underground Techno" priority />
    );
    
    expect(container.firstChild).toBeInTheDocument();
  });

  it('renders image when src is provided', () => {
    render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    const img = screen.getByAltText('Underground Techno cover');
    expect(img).toBeInTheDocument();
  });

  it('renders WebP and JPEG sources', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    const picture = container.querySelector('picture');
    expect(picture).toBeInTheDocument();
    
    const webpSource = container.querySelector('source[type="image/webp"]');
    expect(webpSource).toBeInTheDocument();
    
    const jpegSource = container.querySelector('source[type="image/jpeg"]');
    expect(jpegSource).toBeInTheDocument();
  });

  it('shows placeholder when no src is provided', () => {
    const { container } = render(
      <SceneCover sceneName="Underground Techno" priority />
    );
    
    const placeholder = container.querySelector('svg');
    expect(placeholder).toBeInTheDocument();
  });

  it('applies custom className', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        className="custom-class"
        priority
      />
    );
    
    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('applies custom aspect ratio', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        aspectRatio="1 / 1"
        priority
      />
    );
    
    expect(container.firstChild).toHaveStyle({ aspectRatio: '1 / 1' });
  });

  it('uses default 16/9 aspect ratio', () => {
    const { container } = render(
      <SceneCover sceneName="Underground Techno" priority />
    );
    
    expect(container.firstChild).toHaveStyle({ aspectRatio: '16 / 9' });
  });

  it('uses eager loading for priority images', () => {
    render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    const img = screen.getByAltText('Underground Techno cover');
    expect(img).toHaveAttribute('loading', 'eager');
  });

  it('uses lazy loading by default', async () => {
    const { container } = render(
      <SceneCover sceneName="Underground Techno" src="posts/123/cover.jpg" />
    );
    
    // Initially shows placeholder (not loaded yet)
    const placeholder = container.querySelector('svg');
    expect(placeholder).toBeInTheDocument();
    
    // Trigger intersection to load the image
    const coverDiv = container.firstChild as HTMLElement;
    const entries: IntersectionObserverEntry[] = [
      {
        isIntersecting: true,
        target: coverDiv,
        boundingClientRect: {} as DOMRectReadOnly,
        intersectionRatio: 1,
        intersectionRect: {} as DOMRectReadOnly,
        rootBounds: null,
        time: Date.now(),
      },
    ];
    
    intersectionObserverCallback(entries, {} as IntersectionObserver);
    
    await waitFor(() => {
      const img = screen.getByAltText('Underground Techno cover');
      expect(img).toHaveAttribute('loading', 'lazy');
    });
  });

  it('sets up IntersectionObserver for lazy loading', () => {
    render(
      <SceneCover sceneName="Underground Techno" src="posts/123/cover.jpg" />
    );
    
    expect(observeMock).toHaveBeenCalled();
  });

  it('does not set up IntersectionObserver for priority images', () => {
    render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    expect(observeMock).not.toHaveBeenCalled();
  });

  it('calls onClick when clicked', () => {
    const onClick = vi.fn();
    render(
      <SceneCover
        sceneName="Underground Techno"
        onClick={onClick}
        priority
      />
    );
    
    const cover = screen.getByRole('button');
    fireEvent.click(cover);
    
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('supports keyboard interaction when clickable', () => {
    const onClick = vi.fn();
    render(
      <SceneCover
        sceneName="Underground Techno"
        onClick={onClick}
        priority
      />
    );
    
    const cover = screen.getByRole('button');
    
    fireEvent.keyDown(cover, { key: 'Enter' });
    expect(onClick).toHaveBeenCalledTimes(1);
    
    fireEvent.keyDown(cover, { key: ' ' });
    expect(onClick).toHaveBeenCalledTimes(2);
  });

  it('shows loading skeleton before image loads', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    const skeleton = container.querySelector('.animate-pulse');
    expect(skeleton).toBeInTheDocument();
  });

  it('shows overlay when overlay prop is true and image is loaded', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        overlay
        priority
      />
    );
    
    const img = screen.getByAltText('Underground Techno cover');
    fireEvent.load(img);
    
    const overlay = container.querySelector('.bg-gradient-to-t');
    expect(overlay).toBeInTheDocument();
  });

  it('does not show overlay by default', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    const img = screen.getByAltText('Underground Techno cover');
    fireEvent.load(img);
    
    const overlay = container.querySelector('.bg-gradient-to-t');
    expect(overlay).not.toBeInTheDocument();
  });

  it('falls back to placeholder on image error', () => {
    const { container } = render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/broken.jpg"
        priority
      />
    );
    
    const img = screen.getByAltText('Underground Techno cover');
    fireEvent.error(img);
    
    // Should show placeholder icon
    const placeholder = container.querySelector('svg');
    expect(placeholder).toBeInTheDocument();
  });

  it('applies custom sizes attribute', () => {
    render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        sizes="(max-width: 768px) 100vw, 50vw"
        priority
      />
    );
    
    const webpSource = document.querySelector('source[type="image/webp"]');
    expect(webpSource).toHaveAttribute(
      'sizes',
      '(max-width: 768px) 100vw, 50vw'
    );
  });

  it('uses default 100vw sizes', () => {
    render(
      <SceneCover
        sceneName="Underground Techno"
        src="posts/123/cover.jpg"
        priority
      />
    );
    
    const webpSource = document.querySelector('source[type="image/webp"]');
    expect(webpSource).toHaveAttribute('sizes', '100vw');
  });

  it('loads image when it enters viewport', async () => {
    const { container } = render(
      <SceneCover sceneName="Underground Techno" src="posts/123/cover.jpg" />
    );
    
    // Initially shows placeholder (lazy loading)
    expect(screen.queryByAltText('Underground Techno cover')).not.toBeInTheDocument();
    
    // Simulate intersection
    const coverDiv = container.firstChild as HTMLElement;
    const entries: IntersectionObserverEntry[] = [
      {
        isIntersecting: true,
        target: coverDiv,
        boundingClientRect: {} as DOMRectReadOnly,
        intersectionRatio: 1,
        intersectionRect: {} as DOMRectReadOnly,
        rootBounds: null,
        time: Date.now(),
      },
    ];
    
    intersectionObserverCallback(entries, {} as IntersectionObserver);
    
    await waitFor(() => {
      const img = screen.getByAltText('Underground Techno cover');
      expect(img).toHaveAttribute('src');
    });
  });
});
