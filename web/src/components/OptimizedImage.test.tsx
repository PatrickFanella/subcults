/**
 * OptimizedImage Component Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { OptimizedImage } from './OptimizedImage';

describe('OptimizedImage', () => {
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

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders image with alt text', () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" priority />);
    
    const img = screen.getByAltText('Test image');
    expect(img).toBeInTheDocument();
  });

  it('generates WebP and JPEG sources', () => {
    const { container } = render(
      <OptimizedImage src="/test.jpg" alt="Test image" priority />
    );

    const picture = container.querySelector('picture');
    expect(picture).toBeInTheDocument();

    const webpSource = container.querySelector('source[type="image/webp"]');
    expect(webpSource).toBeInTheDocument();

    const img = screen.getByAltText('Test image');
    expect(img).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(
      <OptimizedImage
        src="/test.jpg"
        alt="Test image"
        className="custom-class"
        priority
      />
    );

    const img = screen.getByAltText('Test image');
    expect(img).toHaveClass('custom-class');
  });

  it('sets aspect ratio when width and height provided', () => {
    render(
      <OptimizedImage
        src="/test.jpg"
        alt="Test image"
        width={800}
        height={600}
        priority
      />
    );

    const img = screen.getByAltText('Test image');
    expect(img).toHaveStyle({ aspectRatio: '800 / 600' });
  });

  it('sets object-fit and object-position styles', () => {
    render(
      <OptimizedImage
        src="/test.jpg"
        alt="Test image"
        objectFit="contain"
        objectPosition="top"
        priority
      />
    );

    const img = screen.getByAltText('Test image');
    expect(img).toHaveStyle({
      objectFit: 'contain',
      objectPosition: 'top',
    });
  });

  it('uses eager loading for priority images', () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" priority />);

    const img = screen.getByAltText('Test image');
    expect(img).toHaveAttribute('loading', 'eager');
  });

  it('uses lazy loading by default', () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" />);

    const img = screen.getByAltText('Test image');
    expect(img).toHaveAttribute('loading', 'lazy');
  });

  it('sets up IntersectionObserver for lazy loading', () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" lazy />);

    expect(observeMock).toHaveBeenCalled();
  });

  it('loads image when it enters viewport', async () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" lazy />);

    // Initially, img should not have src (lazy loading)
    const img = screen.getByAltText('Test image');
    expect(img).not.toHaveAttribute('src');

    // Simulate intersection
    const entries: IntersectionObserverEntry[] = [
      {
        isIntersecting: true,
        target: img,
        boundingClientRect: {} as DOMRectReadOnly,
        intersectionRatio: 1,
        intersectionRect: {} as DOMRectReadOnly,
        rootBounds: null,
        time: Date.now(),
      },
    ];

    intersectionObserverCallback(entries, {} as IntersectionObserver);

    await waitFor(() => {
      expect(img).toHaveAttribute('src', '/test.jpg');
    });
  });

  it('does not set up IntersectionObserver for priority images', () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" priority />);

    expect(observeMock).not.toHaveBeenCalled();
  });

  it('calls onLoad callback when image loads', async () => {
    const onLoad = vi.fn();
    render(
      <OptimizedImage src="/test.jpg" alt="Test image" onLoad={onLoad} priority />
    );

    const img = screen.getByAltText('Test image');
    
    // Simulate image load
    img.dispatchEvent(new Event('load'));

    await waitFor(() => {
      expect(onLoad).toHaveBeenCalled();
    });
  });

  it('calls onError callback and shows error state when image fails', async () => {
    const onError = vi.fn();
    render(
      <OptimizedImage src="/broken.jpg" alt="Test image" onError={onError} priority />
    );

    const img = screen.getByAltText('Test image');
    
    // Simulate image error
    img.dispatchEvent(new Event('error'));

    await waitFor(() => {
      expect(onError).toHaveBeenCalled();
    });

    // Should show error state
    const errorIcon = screen.getByRole('img', {
      name: /Failed to load image/i,
    });
    expect(errorIcon).toBeInTheDocument();
  });

  it('applies custom srcSet when provided', () => {
    const customSrcSet = '/test-320.jpg 320w, /test-640.jpg 640w';
    render(
      <OptimizedImage
        src="/test.jpg"
        alt="Test image"
        srcSet={customSrcSet}
        priority
      />
    );

    const img = screen.getByAltText('Test image');
    expect(img).toHaveAttribute('srcset', expect.stringContaining('320w'));
  });

  it('applies custom sizes attribute', () => {
    const sizes = '(max-width: 640px) 100vw, 50vw';
    render(
      <OptimizedImage
        src="/test.jpg"
        alt="Test image"
        sizes={sizes}
        priority
      />
    );

    const img = screen.getByAltText('Test image');
    expect(img).toHaveAttribute('sizes', sizes);
  });

  it('has async decoding attribute', () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" priority />);

    const img = screen.getByAltText('Test image');
    expect(img).toHaveAttribute('decoding', 'async');
  });

  it('transitions opacity on load', async () => {
    render(<OptimizedImage src="/test.jpg" alt="Test image" priority />);

    const img = screen.getByAltText('Test image');
    
    // Initially should have opacity 0
    expect(img).toHaveStyle({ opacity: '0' });

    // After load, should have opacity 1
    img.dispatchEvent(new Event('load'));

    await waitFor(() => {
      expect(img).toHaveStyle({ opacity: '1' });
    });
  });
});
