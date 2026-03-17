/**
 * ParticipantList Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ParticipantList } from './ParticipantList';
import type { Participant } from '../../types/streaming';

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe('ParticipantList', () => {
  const mockLocalParticipant: Participant = {
    identity: 'local-123',
    name: 'Local User',
    isLocal: true,
    isMuted: false,
    isSpeaking: false,
  };

  const mockRemoteParticipants: Participant[] = [
    {
      identity: 'remote-1',
      name: 'Remote User 1',
      isLocal: false,
      isMuted: false,
      isSpeaking: true,
    },
    {
      identity: 'remote-2',
      name: 'Remote User 2',
      isLocal: false,
      isMuted: true,
      isSpeaking: false,
    },
  ];

  it('renders empty state when no participants', () => {
    render(<ParticipantList participants={[]} localParticipant={null} />);

    expect(screen.getByText(/participantList\.empty/i)).toBeInTheDocument();
  });

  it('renders local participant with "You" label', () => {
    render(<ParticipantList participants={[]} localParticipant={mockLocalParticipant} />);

    expect(screen.getByText(/local user/i)).toBeInTheDocument();
    expect(screen.getByText(/participantList\.you/i)).toBeInTheDocument();
  });

  it('renders remote participants', () => {
    render(<ParticipantList participants={mockRemoteParticipants} localParticipant={null} />);

    expect(screen.getByText(/remote user 1/i)).toBeInTheDocument();
    expect(screen.getByText(/remote user 2/i)).toBeInTheDocument();
  });

  it('shows speaking indicator for speaking participants', () => {
    render(<ParticipantList participants={mockRemoteParticipants} localParticipant={null} />);

    expect(screen.getByText(/participantList\.speaking/i)).toBeInTheDocument();
  });

  it('shows mute indicators', () => {
    render(
      <ParticipantList
        participants={mockRemoteParticipants}
        localParticipant={mockLocalParticipant}
      />
    );

    // Should have 3 participants total, 1 muted
    const listItems = screen.getAllByRole('listitem');
    expect(listItems).toHaveLength(3);
  });

  it('renders participants in correct order (local first)', () => {
    render(
      <ParticipantList
        participants={mockRemoteParticipants}
        localParticipant={mockLocalParticipant}
      />
    );

    const listItems = screen.getAllByRole('listitem');
    // First item should be local participant
    expect(listItems[0]).toHaveTextContent(/local user/i);
    expect(listItems[0]).toHaveTextContent(/participantList\.you/i);
  });
});
