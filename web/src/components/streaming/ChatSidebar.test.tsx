/**
 * ChatSidebar Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChatSidebar } from './ChatSidebar';
import type { ChatMessage } from './ChatSidebar';

// Mock react-i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'chat.sidebar': 'Chat sidebar',
        'chat.title': 'Chat',
        'chat.noMessages': 'No messages yet',
        'chat.you': 'You',
        'chat.placeholder': 'Type a message...',
        'chat.messageInput': 'Message input',
        'chat.send': 'Send',
      };
      return translations[key] || key;
    },
  }),
}));

describe('ChatSidebar', () => {
  const mockMessages: ChatMessage[] = [
    {
      id: '1',
      sender: 'Alice',
      message: 'Hello everyone!',
      timestamp: Date.now() - 60000,
      isLocal: false,
    },
    {
      id: '2',
      sender: 'You',
      message: 'Hi Alice!',
      timestamp: Date.now() - 30000,
      isLocal: true,
    },
  ];

  it('renders chat title', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
      />
    );

    expect(screen.getByText(/Chat/)).toBeInTheDocument();
  });

  it('displays "no messages" when messages array is empty', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
      />
    );

    expect(screen.getByText('No messages yet')).toBeInTheDocument();
  });

  it('renders all messages', () => {
    render(
      <ChatSidebar
        messages={mockMessages}
        onSendMessage={vi.fn()}
      />
    );

    expect(screen.getByText('Hello everyone!')).toBeInTheDocument();
    expect(screen.getByText('Hi Alice!')).toBeInTheDocument();
  });

  it('displays sender names correctly', () => {
    render(
      <ChatSidebar
        messages={mockMessages}
        onSendMessage={vi.fn()}
      />
    );

    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('You')).toBeInTheDocument();
  });

  it('calls onSendMessage when form is submitted', () => {
    const mockOnSend = vi.fn();
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={mockOnSend}
      />
    );

    const input = screen.getByPlaceholderText('Type a message...');
    const sendButton = screen.getByRole('button', { name: 'Send' });

    fireEvent.change(input, { target: { value: 'Test message' } });
    fireEvent.click(sendButton);

    expect(mockOnSend).toHaveBeenCalledWith('Test message');
  });

  it('clears input after sending message', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
      />
    );

    const input = screen.getByPlaceholderText('Type a message...') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'Test message' } });
    fireEvent.submit(input.closest('form')!);

    expect(input.value).toBe('');
  });

  it('does not send empty messages', () => {
    const mockOnSend = vi.fn();
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={mockOnSend}
      />
    );

    const input = screen.getByPlaceholderText('Type a message...');
    fireEvent.change(input, { target: { value: '   ' } });
    fireEvent.submit(input.closest('form')!);

    expect(mockOnSend).not.toHaveBeenCalled();
  });

  it('trims whitespace from messages', () => {
    const mockOnSend = vi.fn();
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={mockOnSend}
      />
    );

    const input = screen.getByPlaceholderText('Type a message...');
    fireEvent.change(input, { target: { value: '  Test message  ' } });
    fireEvent.submit(input.closest('form')!);

    expect(mockOnSend).toHaveBeenCalledWith('Test message');
  });

  it('disables input and send button when disabled prop is true', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
        disabled={true}
      />
    );

    const input = screen.getByPlaceholderText('Type a message...');
    const sendButton = screen.getByRole('button', { name: 'Send' });

    expect(input).toBeDisabled();
    expect(sendButton).toBeDisabled();
  });

  it('has proper accessibility role for messages container', () => {
    render(
      <ChatSidebar
        messages={mockMessages}
        onSendMessage={vi.fn()}
      />
    );

    expect(screen.getByRole('log')).toBeInTheDocument();
  });

  it('has proper complementary role for sidebar', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
      />
    );

    expect(screen.getByRole('complementary', { name: 'Chat sidebar' })).toBeInTheDocument();
  });

  it('disables send button when input is empty', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
      />
    );

    const sendButton = screen.getByRole('button', { name: 'Send' });
    expect(sendButton).toBeDisabled();
  });

  it('enables send button when input has text', () => {
    render(
      <ChatSidebar
        messages={[]}
        onSendMessage={vi.fn()}
      />
    );

    const input = screen.getByPlaceholderText('Type a message...');
    const sendButton = screen.getByRole('button', { name: 'Send' });

    fireEvent.change(input, { target: { value: 'Test' } });
    expect(sendButton).not.toBeDisabled();
  });

  it('applies local message styling', () => {
    const { container } = render(
      <ChatSidebar
        messages={mockMessages}
        onSendMessage={vi.fn()}
      />
    );

    const localMessage = container.querySelector('.chat-message.local');
    expect(localMessage).toBeInTheDocument();
  });

  it('applies remote message styling', () => {
    const { container } = render(
      <ChatSidebar
        messages={mockMessages}
        onSendMessage={vi.fn()}
      />
    );

    const remoteMessage = container.querySelector('.chat-message.remote');
    expect(remoteMessage).toBeInTheDocument();
  });
});
