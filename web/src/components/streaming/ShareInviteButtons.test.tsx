/**
 * ShareInviteButtons Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ShareInviteButtons } from './ShareInviteButtons';

// Mock react-i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: { title?: string }) => {
      const translations: Record<string, string> = {
        'shareButtons.share': 'Share',
        'shareButtons.copyLink': 'Copy Link',
        'shareButtons.copied': 'Copied!',
        'shareButtons.invite': 'Invite',
        'shareButtons.shareOptions': 'Share options',
        'shareButtons.shareText': `Check out ${options?.title}!`,
      };
      return translations[key] || key;
    },
  }),
}));

// Mock clipboard API
const mockWriteText = vi.fn();
Object.assign(navigator, {
  clipboard: {
    writeText: mockWriteText,
  },
});

describe('ShareInviteButtons', () => {
  beforeEach(() => {
    mockWriteText.mockClear();
    mockWriteText.mockResolvedValue(undefined);
  });

  it('renders share and copy link buttons', () => {
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
      />
    );

    expect(screen.getByRole('button', { name: 'Share' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy Link' })).toBeInTheDocument();
  });

  it('copies link to clipboard when copy button is clicked', async () => {
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
      />
    );

    const copyButton = screen.getByRole('button', { name: 'Copy Link' });
    fireEvent.click(copyButton);

    await waitFor(() => {
      expect(mockWriteText).toHaveBeenCalledWith('https://example.com/stream/123');
    });
  });

  it('shows "Copied!" feedback after copying', async () => {
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
      />
    );

    const copyButton = screen.getByRole('button', { name: 'Copy Link' });
    fireEvent.click(copyButton);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Copied!' })).toBeInTheDocument();
    });
  });

  it('calls onShare when share button is clicked', () => {
    const mockOnShare = vi.fn();
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
        onShare={mockOnShare}
      />
    );

    const shareButton = screen.getByRole('button', { name: 'Share' });
    fireEvent.click(shareButton);

    expect(mockOnShare).toHaveBeenCalled();
  });

  it('disables all buttons when disabled prop is true', () => {
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
        disabled={true}
      />
    );

    expect(screen.getByRole('button', { name: 'Share' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Copy Link' })).toBeDisabled();
  });

  it('has proper accessibility group label', () => {
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
      />
    );

    const group = screen.getByRole('group', { name: 'Share options' });
    expect(group).toBeInTheDocument();
  });

  it('handles clipboard write failure gracefully', async () => {
    mockWriteText.mockRejectedValue(new Error('Clipboard access denied'));
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
      />
    );

    const copyButton = screen.getByRole('button', { name: 'Copy Link' });
    fireEvent.click(copyButton);

    await waitFor(() => {
      expect(consoleError).toHaveBeenCalled();
    });

    consoleError.mockRestore();
  });

  it('uses native share API when available', async () => {
    const mockShare = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      share: mockShare,
    });

    const mockOnShare = vi.fn();
    render(
      <ShareInviteButtons
        streamUrl="https://example.com/stream/123"
        streamTitle="Test Stream"
        onShare={mockOnShare}
      />
    );

    const shareButton = screen.getByRole('button', { name: 'Share' });
    fireEvent.click(shareButton);

    await waitFor(() => {
      expect(mockOnShare).toHaveBeenCalled();
      expect(mockShare).toHaveBeenCalledWith({
        title: 'Test Stream',
        text: 'Check out Test Stream!',
        url: 'https://example.com/stream/123',
      });
    });
  });
});
