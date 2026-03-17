/**
 * AudioControls Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AudioControls } from './AudioControls';

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe('AudioControls', () => {
  const defaultProps = {
    isMuted: false,
    onToggleMute: vi.fn(),
    onLeave: vi.fn(),
    onVolumeChange: vi.fn(),
  };

  it('renders mute button with unmuted state', () => {
    render(<AudioControls {...defaultProps} />);

    const muteButton = screen.getByRole('button', {
      name: /audioControls\.mute/i,
    });
    expect(muteButton).toBeInTheDocument();
    expect(muteButton).not.toBeDisabled();
  });

  it('renders mute button with muted state', () => {
    render(<AudioControls {...defaultProps} isMuted={true} />);

    const muteButton = screen.getByRole('button', {
      name: /audioControls\.unmute/i,
    });
    expect(muteButton).toBeInTheDocument();
  });

  it('calls onToggleMute when mute button clicked', async () => {
    const onToggleMute = vi.fn();
    const user = userEvent.setup();

    render(<AudioControls {...defaultProps} onToggleMute={onToggleMute} />);

    const muteButton = screen.getByRole('button', {
      name: /audioControls\.mute/i,
    });
    await user.click(muteButton);

    expect(onToggleMute).toHaveBeenCalledTimes(1);
  });

  it('renders volume control button', () => {
    render(<AudioControls {...defaultProps} />);

    const volumeButton = screen.getByRole('button', { name: /audioControls\.volumeControl/i });
    expect(volumeButton).toBeInTheDocument();
  });

  it('shows volume slider when volume button clicked', async () => {
    const user = userEvent.setup();

    render(<AudioControls {...defaultProps} />);

    const volumeButton = screen.getByRole('button', { name: /audioControls\.volumeControl/i });
    await user.click(volumeButton);

    const slider = screen.getByRole('slider', { name: /volume slider/i });
    expect(slider).toBeInTheDocument();
  });

  it('calls onVolumeChange when slider changed', async () => {
    const onVolumeChange = vi.fn();
    const user = userEvent.setup();

    render(<AudioControls {...defaultProps} onVolumeChange={onVolumeChange} />);

    // Open volume slider
    const volumeButton = screen.getByRole('button', { name: /audioControls\.volumeControl/i });
    await user.click(volumeButton);

    // Change volume using fireEvent for React synthetic events
    const slider = screen.getByRole('slider', { name: /volume slider/i });
    fireEvent.change(slider, { target: { value: '75' } });

    expect(onVolumeChange).toHaveBeenCalledWith(75);
  });

  it('renders leave button', () => {
    render(<AudioControls {...defaultProps} />);

    const leaveButton = screen.getByRole('button', { name: /audioControls\.leaveRoom/i });
    expect(leaveButton).toBeInTheDocument();
  });

  it('calls onLeave when leave button clicked', async () => {
    const onLeave = vi.fn();
    const user = userEvent.setup();

    render(<AudioControls {...defaultProps} onLeave={onLeave} />);

    const leaveButton = screen.getByRole('button', { name: /audioControls\.leaveRoom/i });
    await user.click(leaveButton);

    expect(onLeave).toHaveBeenCalledTimes(1);
  });

  it('disables all controls when disabled prop is true', () => {
    render(<AudioControls {...defaultProps} disabled={true} />);

    const muteButton = screen.getByRole('button', {
      name: /audioControls\.mute/i,
    });
    const volumeButton = screen.getByRole('button', { name: /audioControls\.volumeControl/i });
    const leaveButton = screen.getByRole('button', { name: /audioControls\.leaveRoom/i });

    expect(muteButton).toBeDisabled();
    expect(volumeButton).toBeDisabled();
    expect(leaveButton).toBeDisabled();
  });
});
