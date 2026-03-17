/**
 * ChatSidebar Component
 * Provides a chat/comments interface for stream participants
 */

import React, { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

export interface ChatMessage {
  id: string;
  sender: string;
  message: string;
  timestamp: number;
  isLocal?: boolean;
}

export interface ChatSidebarProps {
  messages: ChatMessage[];
  onSendMessage: (message: string) => void;
  disabled?: boolean;
  maxHeight?: string;
}

export const ChatSidebar: React.FC<ChatSidebarProps> = ({
  messages,
  onSendMessage,
  disabled = false,
  maxHeight = '600px',
}) => {
  const { t } = useTranslation('streaming');
  const [inputValue, setInputValue] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll to bottom when new messages arrive (only if user is at bottom)
  useEffect(() => {
    const container = messagesEndRef.current?.parentElement;
    if (!container || !messagesEndRef.current) return;

    // Check if user is already at or near the bottom (within 100px)
    const isNearBottom = 
      container.scrollHeight - container.scrollTop - container.clientHeight < 100;

    if (isNearBottom && typeof messagesEndRef.current.scrollIntoView === 'function') {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();

    if (!inputValue.trim() || disabled) {
      return;
    }

    onSendMessage(inputValue.trim());
    setInputValue('');

    // Focus input after sending
    inputRef.current?.focus();
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp);
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  };

  return (
    <div
      className="chat-sidebar"
      role="complementary"
      aria-label={t('chat.sidebar')}
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        maxHeight,
        backgroundColor: '#1f2937',
        borderRadius: '0.75rem',
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          padding: '1rem 1.25rem',
          borderBottom: '1px solid #374151',
          backgroundColor: '#111827',
        }}
      >
        <h3
          style={{
            margin: 0,
            fontSize: '1.125rem',
            fontWeight: 600,
            color: 'white',
          }}
        >
          💬 {t('chat.title')}
        </h3>
      </div>

      {/* Messages List */}
      <div
        className="messages-container"
        role="log"
        aria-live="polite"
        aria-atomic="false"
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '1rem',
          display: 'flex',
          flexDirection: 'column',
          gap: '0.75rem',
          // Custom scrollbar styling
          scrollbarWidth: 'thin',
          scrollbarColor: '#4b5563 #1f2937',
        }}
      >
        {messages.length === 0 ? (
          <div
            style={{
              padding: '2rem',
              textAlign: 'center',
              color: '#6b7280',
              fontStyle: 'italic',
            }}
          >
            {t('chat.noMessages')}
          </div>
        ) : (
          messages.map((msg) => (
            <div
              key={msg.id}
              className={`chat-message ${msg.isLocal ? 'local' : 'remote'}`}
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '0.25rem',
                alignSelf: msg.isLocal ? 'flex-end' : 'flex-start',
                maxWidth: '80%',
              }}
            >
              {/* Sender and timestamp */}
              <div
                style={{
                  display: 'flex',
                  gap: '0.5rem',
                  fontSize: '0.75rem',
                  color: '#9ca3af',
                }}
              >
                <span style={{ fontWeight: 600 }}>
                  {msg.isLocal ? t('chat.you') : msg.sender}
                </span>
                <span>{formatTimestamp(msg.timestamp)}</span>
              </div>

              {/* Message bubble */}
              <div
                style={{
                  padding: '0.625rem 0.875rem',
                  backgroundColor: msg.isLocal ? '#3b82f6' : '#374151',
                  borderRadius: '0.75rem',
                  color: 'white',
                  fontSize: '0.875rem',
                  wordWrap: 'break-word',
                }}
              >
                {msg.message}
              </div>
            </div>
          ))
        )}
        <div ref={messagesEndRef} />

        {/* Custom scrollbar styling */}
        <style>{`
          .messages-container::-webkit-scrollbar {
            width: 8px;
          }
          .messages-container::-webkit-scrollbar-track {
            background: #1f2937;
            border-radius: 4px;
          }
          .messages-container::-webkit-scrollbar-thumb {
            background: #4b5563;
            border-radius: 4px;
          }
          .messages-container::-webkit-scrollbar-thumb:hover {
            background: #6b7280;
          }
        `}</style>
      </div>

      {/* Input Form */}
      <form
        onSubmit={handleSendMessage}
        style={{
          padding: '1rem',
          borderTop: '1px solid #374151',
          backgroundColor: '#111827',
        }}
      >
        <div
          style={{
            display: 'flex',
            gap: '0.5rem',
          }}
        >
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder={t('chat.placeholder')}
            disabled={disabled}
            aria-label={t('chat.messageInput')}
            maxLength={500}
            style={{
              flex: 1,
              padding: '0.625rem 0.875rem',
              fontSize: '0.875rem',
              backgroundColor: '#1f2937',
              border: '1px solid #4b5563',
              borderRadius: '0.5rem',
              color: 'white',
              outline: 'none',
            }}
            onFocus={(e) => {
              e.currentTarget.style.borderColor = '#3b82f6';
            }}
            onBlur={(e) => {
              e.currentTarget.style.borderColor = '#4b5563';
            }}
          />

          <button
            type="submit"
            disabled={disabled || !inputValue.trim()}
            aria-label={t('chat.send')}
            style={{
              padding: '0.625rem 1.25rem',
              fontSize: '0.875rem',
              fontWeight: 600,
              backgroundColor: '#3b82f6',
              color: 'white',
              border: 'none',
              borderRadius: '0.5rem',
              cursor: disabled || !inputValue.trim() ? 'not-allowed' : 'pointer',
              transition: 'all 0.2s ease',
              opacity: disabled || !inputValue.trim() ? 0.5 : 1,
            }}
            onMouseEnter={(e) => {
              if (!disabled && inputValue.trim()) {
                e.currentTarget.style.backgroundColor = '#2563eb';
              }
            }}
            onMouseLeave={(e) => {
              if (!disabled && inputValue.trim()) {
                e.currentTarget.style.backgroundColor = '#3b82f6';
              }
            }}
          >
            {t('chat.send')}
          </button>
        </div>
      </form>
    </div>
  );
};
