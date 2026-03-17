import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DetailPanel } from './DetailPanel';
import type { Scene, Event } from '../types/scene';

describe('DetailPanel', () => {
  const mockScene: Scene = {
    id: 'scene-1',
    name: 'Underground Techno Scene',
    description: 'A vibrant techno scene in the heart of the city',
    allow_precise: false,
    coarse_geohash: '9q8yy',
    tags: ['techno', 'underground', 'electronic'],
    visibility: 'public',
  };

  const mockEvent: Event = {
    id: 'event-1',
    scene_id: 'scene-1',
    name: 'Saturday Night Rave',
    description: 'Epic rave party',
    allow_precise: true,
    coarse_geohash: '9q8yy',
  };

  const mockOnClose = vi.fn();
  const mockOnAnalyticsEvent = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Reset body overflow
    document.body.style.overflow = '';
  });

  it('does not render when closed', () => {
    const { container } = render(
      <DetailPanel
        isOpen={false}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(container.firstChild).toBeNull();
  });

  it('renders when open with scene entity', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Underground Techno Scene')).toBeInTheDocument();
    expect(screen.getByText('A vibrant techno scene in the heart of the city')).toBeInTheDocument();
  });

  it('renders when open with event entity', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockEvent}
      />
    );
    
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Saturday Night Rave')).toBeInTheDocument();
    expect(screen.getByText('Epic rave party')).toBeInTheDocument();
  });

  it('displays scene type correctly', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(screen.getByText('scene')).toBeInTheDocument();
  });

  it('displays event type correctly', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockEvent}
      />
    );
    
    expect(screen.getByText('event')).toBeInTheDocument();
  });

  it('displays tags for scenes', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(screen.getByText('techno')).toBeInTheDocument();
    expect(screen.getByText('underground')).toBeInTheDocument();
    expect(screen.getByText('electronic')).toBeInTheDocument();
  });

  it('displays privacy notice for non-precise location', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(screen.getByText(/Approximate location \(privacy preserved\)/)).toBeInTheDocument();
  });

  it('displays privacy notice for precise location', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockEvent}
      />
    );
    
    expect(screen.getByText(/Precise location shared/)).toBeInTheDocument();
  });

  it('shows loading state', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={null}
        loading={true}
      />
    );
    
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    expect(screen.getByText('Loading details...')).toBeInTheDocument();
  });

  it('closes when close button is clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    const closeButton = screen.getByLabelText('Close detail panel');
    await user.click(closeButton);
    
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('closes when backdrop is clicked', async () => {
    const user = userEvent.setup();
    
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    const backdrop = document.querySelector('.detail-panel-backdrop');
    expect(backdrop).toBeInTheDocument();
    
    await user.click(backdrop!);
    
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('closes when ESC key is pressed', async () => {
    const user = userEvent.setup();
    
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    await user.keyboard('{Escape}');
    
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('has correct accessibility attributes', () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(dialog).toHaveAttribute('aria-labelledby', 'detail-panel-title');
  });

  it('prevents body scroll when open', () => {
    const { rerender } = render(
      <DetailPanel
        isOpen={false}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(document.body.style.overflow).toBe('');
    
    rerender(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(document.body.style.overflow).toBe('hidden');
  });

  it('restores body scroll when closed', () => {
    const { rerender } = render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(document.body.style.overflow).toBe('hidden');
    
    rerender(
      <DetailPanel
        isOpen={false}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    expect(document.body.style.overflow).toBe('');
  });

  it('emits analytics event on open', async () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
        onAnalyticsEvent={mockOnAnalyticsEvent}
      />
    );
    
    await waitFor(() => {
      expect(mockOnAnalyticsEvent).toHaveBeenCalledWith('detail_panel_open', {
        entity_type: 'scene',
        entity_id: 'scene-1',
      });
    });
  });

  it('emits analytics event on close', async () => {
    const { rerender } = render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
        onAnalyticsEvent={mockOnAnalyticsEvent}
      />
    );
    
    mockOnAnalyticsEvent.mockClear();
    
    rerender(
      <DetailPanel
        isOpen={false}
        onClose={mockOnClose}
        entity={mockScene}
        onAnalyticsEvent={mockOnAnalyticsEvent}
      />
    );
    
    await waitFor(() => {
      expect(mockOnAnalyticsEvent).toHaveBeenCalledWith('detail_panel_close', {
        entity_type: 'scene',
        entity_id: 'scene-1',
      });
    });
  });

  it('focuses close button when opened', async () => {
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    const closeButton = screen.getByLabelText('Close detail panel');
    
    await waitFor(() => {
      expect(document.activeElement).toBe(closeButton);
    }, { timeout: 500 });
  });

  it('traps focus within panel', async () => {
    const user = userEvent.setup();
    
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={mockScene}
      />
    );
    
    const closeButton = screen.getByLabelText('Close detail panel');
    
    await waitFor(() => {
      expect(document.activeElement).toBe(closeButton);
    }, { timeout: 500 });
    
    // Tab should cycle back to close button (only focusable element)
    await user.tab();
    expect(document.activeElement).toBe(closeButton);
    
    // Shift+Tab should also stay on close button
    await user.tab({ shift: true });
    expect(document.activeElement).toBe(closeButton);
  });

  it('does not display tags section if no tags', () => {
    const sceneWithoutTags: Scene = { ...mockScene, tags: [] };
    
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={sceneWithoutTags}
      />
    );
    
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });

  it('does not display description if not provided', () => {
    const sceneWithoutDescription: Scene = {
      ...mockScene,
      description: undefined,
    };
    
    render(
      <DetailPanel
        isOpen={true}
        onClose={mockOnClose}
        entity={sceneWithoutDescription}
      />
    );
    
    expect(screen.queryByText('A vibrant techno scene in the heart of the city')).not.toBeInTheDocument();
  });
});
