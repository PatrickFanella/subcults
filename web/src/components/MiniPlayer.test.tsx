/**
 * MiniPlayer Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MiniPlayer } from './MiniPlayer';
import { useStreamingStore } from '../stores/streamingStore';

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe('MiniPlayer', () => {
  beforeEach(() => {
    // Reset store
    useStreamingStore.setState({
      room: null,
      roomName: null,
      isConnected: false,
      isConnecting: false,
      error: null,
      connectionQuality: 'unknown',
      volume: 100,
      isLocalMuted: false,
      reconnectAttempts: 0,
      isReconnecting: false,
    });
  });

  describe('Visibility', () => {
    it('should not render when not connected', () => {
      const { container } = render(<MiniPlayer />);
      expect(container.firstChild).toBeNull();
    });

    it('should render when connected to a stream', () => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
      });

      render(<MiniPlayer />);

      expect(screen.getByRole('region')).toBeInTheDocument();
    });
  });

  describe('Stream Info Display', () => {
    beforeEach(() => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
      });
    });

    it('should display room name', () => {
      render(<MiniPlayer />);

      expect(screen.getByText('test-room')).toBeInTheDocument();
    });

    it('should display "Now Playing" label', () => {
      render(<MiniPlayer />);

      expect(screen.getByText('miniPlayer.nowPlaying')).toBeInTheDocument();
    });
  });

  describe('Mute Control', () => {
    beforeEach(() => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
      });
    });

    it('should render mute button', () => {
      render(<MiniPlayer />);

      const muteButton = screen.getByRole('button', {
        name: /miniPlayer\.mute/i,
      });
      expect(muteButton).toBeInTheDocument();
    });

    it('should toggle mute when clicked', async () => {
      const toggleMute = vi.fn();
      useStreamingStore.setState({
        toggleMute,
      });

      const user = userEvent.setup();
      render(<MiniPlayer />);

      const muteButton = screen.getByRole('button', {
        name: /miniPlayer\.mute/i,
      });

      await user.click(muteButton);

      expect(toggleMute).toHaveBeenCalledTimes(1);
    });

    it('should show muted state', () => {
      useStreamingStore.setState({
        isLocalMuted: true,
      });

      render(<MiniPlayer />);

      const muteButton = screen.getByRole('button', {
        name: /miniPlayer\.unmute/i,
      });
      expect(muteButton).toBeInTheDocument();
    });

    it('should toggle mute on spacebar press', async () => {
      const toggleMute = vi.fn();
      useStreamingStore.setState({
        toggleMute,
      });

      render(<MiniPlayer />);

      const miniPlayer = screen.getByRole('region');
      fireEvent.keyDown(miniPlayer, { key: ' ' });

      await waitFor(() => {
        expect(toggleMute).toHaveBeenCalled();
      });
    });
  });

  describe('Volume Control', () => {
    beforeEach(() => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
        volume: 75,
      });
    });

    it('should render volume button', () => {
      render(<MiniPlayer />);

      const volumeButton = screen.getByRole('button', {
        name: /miniPlayer\.volumeControl/i,
      });
      expect(volumeButton).toBeInTheDocument();
    });

    it('should show volume slider when volume button clicked', async () => {
      const user = userEvent.setup();
      render(<MiniPlayer />);

      const volumeButton = screen.getByRole('button', {
        name: /miniPlayer\.volumeControl/i,
      });

      await user.click(volumeButton);

      const slider = screen.getByRole('slider');
      expect(slider).toBeInTheDocument();
    });

    it('should update volume when slider changed', async () => {
      const setVolume = vi.fn();
      useStreamingStore.setState({
        setVolume,
      });

      const user = userEvent.setup();
      render(<MiniPlayer />);

      // Open volume slider
      const volumeButton = screen.getByRole('button', {
        name: /miniPlayer\.volumeControl/i,
      });
      await user.click(volumeButton);

      // Change volume
      const slider = screen.getByRole('slider');
      fireEvent.change(slider, { target: { value: '50' } });

      expect(setVolume).toHaveBeenCalledWith(50);
    });

    it('should close volume slider on escape key', async () => {
      const user = userEvent.setup();
      render(<MiniPlayer />);

      // Open volume slider
      const volumeButton = screen.getByRole('button', {
        name: /miniPlayer\.volumeControl/i,
      });
      await user.click(volumeButton);

      expect(screen.getByRole('slider')).toBeInTheDocument();

      // Press escape
      const miniPlayer = screen.getByRole('region');
      fireEvent.keyDown(miniPlayer, { key: 'Escape' });

      await waitFor(() => {
        expect(screen.queryByRole('slider')).not.toBeInTheDocument();
      });
    });

    it('should display correct volume icon based on level', () => {
      render(<MiniPlayer />);

      const volumeButton = screen.getByRole('button', {
        name: /miniPlayer\.volumeControl/i,
      });

      // Volume at 75 should show high volume icon
      expect(volumeButton.textContent).toContain('🔊');
    });
  });

  describe('Leave Button', () => {
    beforeEach(() => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
      });
    });

    it('should render leave button', () => {
      render(<MiniPlayer />);

      const leaveButton = screen.getByRole('button', {
        name: /miniPlayer\.leave/i,
      });
      expect(leaveButton).toBeInTheDocument();
    });

    it('should disconnect when leave button clicked', async () => {
      const disconnect = vi.fn();
      useStreamingStore.setState({
        disconnect,
      });

      const user = userEvent.setup();
      render(<MiniPlayer />);

      const leaveButton = screen.getByRole('button', {
        name: /miniPlayer\.leave/i,
      });

      await user.click(leaveButton);

      expect(disconnect).toHaveBeenCalledTimes(1);
    });
  });

  describe('Connection Quality Indicator', () => {
    beforeEach(() => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
      });
    });

    it('should display connection quality indicator', () => {
      render(<MiniPlayer />);

      const indicator = screen.getByLabelText(/miniPlayer\.quality/i);
      expect(indicator).toBeInTheDocument();
    });

    it('should change color based on connection quality', () => {
      const { rerender } = render(<MiniPlayer />);

      // Good quality - wrap state update in act
      act(() => {
        useStreamingStore.setState({ connectionQuality: 'good' });
      });
      rerender(<MiniPlayer />);

      const indicator = screen.getByLabelText(/miniPlayer\.quality\.good/i);
      expect(indicator).toHaveStyle({ backgroundColor: '#f59e0b' });
    });
  });

  describe('Accessibility', () => {
    beforeEach(() => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
        volume: 75,
      });
    });

    it('should have proper ARIA labels', () => {
      render(<MiniPlayer />);

      expect(screen.getByRole('region')).toHaveAttribute('aria-label', 'miniPlayer.label');
    });

    it('should have accessible volume slider', async () => {
      const user = userEvent.setup();
      render(<MiniPlayer />);

      // Open volume slider
      const volumeButton = screen.getByRole('button', {
        name: /miniPlayer\.volumeControl/i,
      });
      await user.click(volumeButton);

      const slider = screen.getByRole('slider');
      expect(slider).toHaveAttribute('aria-valuemin', '0');
      expect(slider).toHaveAttribute('aria-valuemax', '100');
      expect(slider).toHaveAttribute('aria-valuenow', '75');
      expect(slider).toHaveAttribute('aria-valuetext', '75%');
    });

    it('should have keyboard navigation support', () => {
      render(<MiniPlayer />);

      const miniPlayer = screen.getByRole('region');
      expect(miniPlayer).toBeInTheDocument();

      // Test keyboard event handling
      fireEvent.keyDown(miniPlayer, { key: ' ' });
      // Should not throw
    });
  });

  describe('Persistence Across Routes', () => {
    it('should remain visible when route changes', () => {
      useStreamingStore.setState({
        isConnected: true,
        roomName: 'test-room',
      });

      const { rerender } = render(<MiniPlayer />);

      // Simulate route change by rerendering
      rerender(<MiniPlayer />);

      expect(screen.getByRole('region')).toBeInTheDocument();
      expect(screen.getByText('test-room')).toBeInTheDocument();
    });
  });
});
