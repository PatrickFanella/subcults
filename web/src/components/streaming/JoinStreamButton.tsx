/**
 * JoinStreamButton Component
 * Button to initiate joining a LiveKit audio room
 */

import React from 'react';
import { useTranslation } from 'react-i18next';

export interface JoinStreamButtonProps {
  isConnected: boolean;
  isConnecting: boolean;
  onJoin: () => void;
  disabled?: boolean;
}

export const JoinStreamButton: React.FC<JoinStreamButtonProps> = ({
  isConnected,
  isConnecting,
  onJoin,
  disabled = false,
}) => {
  const { t } = useTranslation('streaming');

  const buttonText = isConnecting
    ? t('joinButton.connecting')
    : isConnected
      ? t('joinButton.connected')
      : t('joinButton.join');

  const buttonClass = isConnected
    ? 'bg-status-success text-black border border-status-success'
    : isConnecting
      ? 'bg-background-secondary text-foreground-secondary border border-border'
      : 'bg-brand-primary hover:bg-brand-primary-dark text-white border border-brand-primary';

  return (
    <button
      onClick={onJoin}
      disabled={disabled || isConnected || isConnecting}
      className={`
        join-stream-button px-6 py-3 text-base font-bold uppercase tracking-[0.05em] rounded-none
        min-w-[150px]
        transition-none
        focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-brand-primary
        disabled:opacity-50 disabled:cursor-not-allowed
        ${buttonClass}
        ${isConnected ? 'connected' : ''}
        ${isConnecting ? 'connecting' : ''}
      `.trim()}
      aria-label={buttonText}
    >
      {buttonText}
    </button>
  );
};
