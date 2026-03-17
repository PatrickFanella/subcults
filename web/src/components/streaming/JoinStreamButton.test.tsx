/**
 * JoinStreamButton Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { JoinStreamButton } from './JoinStreamButton';

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe('JoinStreamButton', () => {
  it('renders join button when not connected', () => {
    render(<JoinStreamButton isConnected={false} isConnecting={false} onJoin={vi.fn()} />);

    const button = screen.getByRole('button', { name: /joinButton\.join/i });
    expect(button).toBeInTheDocument();
    expect(button).not.toBeDisabled();
  });

  it('shows connecting state', () => {
    render(<JoinStreamButton isConnected={false} isConnecting={true} onJoin={vi.fn()} />);

    const button = screen.getByRole('button', { name: /joinButton\.connecting/i });
    expect(button).toBeInTheDocument();
    expect(button).toBeDisabled();
  });

  it('shows connected state', () => {
    render(<JoinStreamButton isConnected={true} isConnecting={false} onJoin={vi.fn()} />);

    const button = screen.getByRole('button', { name: /joinButton\.connected/i });
    expect(button).toBeInTheDocument();
    expect(button).toBeDisabled();
  });

  it('calls onJoin when clicked', async () => {
    const onJoin = vi.fn();
    const user = userEvent.setup();

    render(<JoinStreamButton isConnected={false} isConnecting={false} onJoin={onJoin} />);

    const button = screen.getByRole('button', { name: /joinButton\.join/i });
    await user.click(button);

    expect(onJoin).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    render(
      <JoinStreamButton isConnected={false} isConnecting={false} onJoin={vi.fn()} disabled={true} />
    );

    const button = screen.getByRole('button', { name: /joinButton\.join/i });
    expect(button).toBeDisabled();
  });

  it('does not call onJoin when disabled', async () => {
    const onJoin = vi.fn();
    const user = userEvent.setup();

    render(
      <JoinStreamButton isConnected={false} isConnecting={false} onJoin={onJoin} disabled={true} />
    );

    const button = screen.getByRole('button', { name: /joinButton\.join/i });
    await user.click(button);

    expect(onJoin).not.toHaveBeenCalled();
  });
});
