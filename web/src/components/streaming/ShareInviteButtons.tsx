/**
 * ShareInviteButtons Component
 * Provides buttons to share and invite others to the stream
 */

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';

export interface ShareInviteButtonsProps {
  streamUrl: string;
  streamTitle: string;
  onShare?: () => void;
  disabled?: boolean;
}

export const ShareInviteButtons: React.FC<ShareInviteButtonsProps> = ({
  streamUrl,
  streamTitle,
  onShare,
  disabled = false,
}) => {
  const { t } = useTranslation('streaming');
  const [copied, setCopied] = useState(false);

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(streamUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy link:', error);
    }
  };

  const handleShare = async () => {
    if (onShare) {
      onShare();
    }

    // Use native share if available
    if (navigator.share) {
      try {
        await navigator.share({
          title: streamTitle,
          text: t('shareButtons.shareText', { title: streamTitle }),
          url: streamUrl,
        });
      } catch (error) {
        // User cancelled or share failed - not an error
        if ((error as Error).name !== 'AbortError') {
          console.error('Failed to share:', error);
        }
      }
    } else {
      // Fallback to copy
      handleCopyLink();
    }
  };

  return (
    <div
      className="share-invite-buttons"
      role="group"
      aria-label={t('shareButtons.shareOptions')}
      style={{
        display: 'flex',
        gap: '0.75rem',
        flexWrap: 'wrap',
      }}
    >
      {/* Share Button */}
      <button
        onClick={handleShare}
        disabled={disabled}
        className="share-button"
        aria-label={t('shareButtons.share')}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem',
          padding: '0.625rem 1rem',
          fontSize: '0.875rem',
          fontWeight: 600,
          borderRadius: '0.5rem',
          border: 'none',
          cursor: disabled ? 'not-allowed' : 'pointer',
          backgroundColor: '#3b82f6',
          color: 'white',
          transition: 'all 0.2s ease',
          opacity: disabled ? 0.5 : 1,
        }}
        onMouseEnter={(e) => {
          if (!disabled) {
            e.currentTarget.style.backgroundColor = '#2563eb';
          }
        }}
        onMouseLeave={(e) => {
          if (!disabled) {
            e.currentTarget.style.backgroundColor = '#3b82f6';
          }
        }}
      >
        <span aria-hidden="true">📤</span>
        {t('shareButtons.share')}
      </button>

      {/* Copy Link Button */}
      <button
        onClick={handleCopyLink}
        disabled={disabled}
        className="copy-link-button"
        aria-label={copied ? t('shareButtons.copied') : t('shareButtons.copyLink')}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem',
          padding: '0.625rem 1rem',
          fontSize: '0.875rem',
          fontWeight: 600,
          borderRadius: '0.5rem',
          border: '2px solid #4b5563',
          cursor: disabled ? 'not-allowed' : 'pointer',
          backgroundColor: copied ? '#10b981' : '#1f2937',
          color: 'white',
          transition: 'all 0.2s ease',
          opacity: disabled ? 0.5 : 1,
        }}
        onMouseEnter={(e) => {
          if (!disabled && !copied) {
            e.currentTarget.style.borderColor = '#6b7280';
          }
        }}
        onMouseLeave={(e) => {
          if (!copied) {
            e.currentTarget.style.borderColor = '#4b5563';
          }
        }}
      >
        <span aria-hidden="true">{copied ? '✓' : '🔗'}</span>
        {copied ? t('shareButtons.copied') : t('shareButtons.copyLink')}
      </button>
    </div>
  );
};
