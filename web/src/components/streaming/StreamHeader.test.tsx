/**
 * StreamHeader Component Tests
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StreamHeader } from './StreamHeader';

// Mock react-i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) => {
      const translations: Record<string, string> = {
        'streamHeader.live': 'LIVE',
        'streamHeader.organizer': 'Organizer',
        'streamHeader.listeners': options?.count === 1 ? 'listener' : 'listeners',
      };
      return translations[key] || key;
    },
  }),
}));

describe('StreamHeader', () => {
  it('renders stream title', () => {
    render(
      <StreamHeader
        title="Underground Beats Session"
        listenerCount={0}
      />
    );

    expect(screen.getByText('Underground Beats Session')).toBeInTheDocument();
  });

  it('displays organizer when provided', () => {
    render(
      <StreamHeader
        title="Test Stream"
        organizer="DJ Alice"
        listenerCount={0}
      />
    );

    expect(screen.getByText(/DJ Alice/)).toBeInTheDocument();
    expect(screen.getByText(/Organizer:/)).toBeInTheDocument();
  });

  it('displays listener count', () => {
    render(
      <StreamHeader
        title="Test Stream"
        listenerCount={42}
      />
    );

    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText(/listeners/)).toBeInTheDocument();
  });

  it('shows live indicator when isLive is true', () => {
    render(
      <StreamHeader
        title="Live Stream"
        listenerCount={10}
        isLive={true}
      />
    );

    expect(screen.getByText('LIVE')).toBeInTheDocument();
    const liveIndicator = screen.getByRole('status', { name: 'LIVE' });
    expect(liveIndicator).toBeInTheDocument();
  });

  it('does not show live indicator when isLive is false', () => {
    render(
      <StreamHeader
        title="Recorded Stream"
        listenerCount={5}
        isLive={false}
      />
    );

    expect(screen.queryByText('LIVE')).not.toBeInTheDocument();
  });

  it('handles zero listeners', () => {
    render(
      <StreamHeader
        title="Empty Stream"
        listenerCount={0}
      />
    );

    expect(screen.getByText('0')).toBeInTheDocument();
  });

  it('has proper banner role for accessibility', () => {
    render(
      <StreamHeader
        title="Accessible Stream"
        listenerCount={1}
      />
    );

    const header = screen.getByRole('banner');
    expect(header).toBeInTheDocument();
  });

  it('applies live border styling when live', () => {
    const { container } = render(
      <StreamHeader
        title="Live Stream"
        listenerCount={5}
        isLive={true}
      />
    );

    const header = container.querySelector('.stream-header');
    expect(header).toBeInTheDocument();
    // Check that the style attribute contains the red border color (rgb format)
    const style = header?.getAttribute('style');
    expect(style).toContain('rgb(239, 68, 68)');
  });

  it('applies normal border styling when not live', () => {
    const { container } = render(
      <StreamHeader
        title="Not Live Stream"
        listenerCount={5}
        isLive={false}
      />
    );

    const header = container.querySelector('.stream-header');
    expect(header).toBeInTheDocument();
    // Check that the style attribute contains the gray border color (rgb format)
    const style = header?.getAttribute('style');
    expect(style).toContain('rgb(55, 65, 81)');
  });
});
