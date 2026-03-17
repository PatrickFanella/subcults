/**
 * Modal Component
 * Unified modal/dialog component with consistent styling
 *
 * Features:
 * - Accessible focus trap
 * - ESC key to close
 * - Backdrop click to close (configurable)
 * - Slide and fade animations
 * - ARIA compliant (role="dialog", aria-modal)
 * - Focus management
 * - Prevent body scroll when open
 */

import { useEffect, useRef, useCallback } from 'react';
import { Button } from './Button';

export interface ModalProps {
  /**
   * Whether the modal is open
   */
  isOpen: boolean;

  /**
   * Handler called when modal should close
   */
  onClose: () => void;

  /**
   * Modal title
   */
  title: string;

  /**
   * Modal content
   */
  children: React.ReactNode;

  /**
   * Footer content (typically action buttons)
   */
  footer?: React.ReactNode;

  /**
   * Close on backdrop click
   * @default true
   */
  closeOnBackdrop?: boolean;

  /**
   * Close on ESC key
   * @default true
   */
  closeOnEsc?: boolean;

  /**
   * Size variant
   * @default 'md'
   */
  size?: 'sm' | 'md' | 'lg' | 'xl';

  /**
   * Additional CSS classes
   */
  className?: string;
}

/**
 * Get size-specific max-width
 */
function getSizeClass(size: ModalProps['size']): string {
  const sizes = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
  };
  return sizes[size || 'md'];
}

/**
 * Modal component with design system styling
 */
export function Modal({
  isOpen,
  onClose,
  title,
  children,
  footer,
  closeOnBackdrop = true,
  closeOnEsc = true,
  size = 'md',
  className = '',
}: ModalProps) {
  const modalRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Track focus before modal opens
  useEffect(() => {
    if (isOpen) {
      previousFocusRef.current = document.activeElement as HTMLElement;

      // Focus close button after animation
      setTimeout(() => {
        closeButtonRef.current?.focus();
      }, 100);
    } else if (previousFocusRef.current) {
      // Return focus when closing
      previousFocusRef.current.focus();
    }
  }, [isOpen]);

  // Handle ESC key
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && closeOnEsc) {
        e.preventDefault();
        onClose();
      }
    },
    [isOpen, closeOnEsc, onClose]
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  // Focus trap
  useEffect(() => {
    if (!isOpen || !modalRef.current) return;

    const modal = modalRef.current;
    const focusableElements = modal.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );

    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    const handleTabKey = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;

      if (e.shiftKey) {
        // Shift+Tab
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement.focus();
        }
      } else {
        // Tab
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement.focus();
        }
      }
    };

    modal.addEventListener('keydown', handleTabKey as EventListener);
    return () => modal.removeEventListener('keydown', handleTabKey as EventListener);
  }, [isOpen]);

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  if (!isOpen) return null;

  const handleBackdropClick = () => {
    if (closeOnBackdrop) {
      onClose();
    }
  };

  const sizeClass = getSizeClass(size);

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-[999] animate-fade-in"
        onClick={handleBackdropClick}
        aria-hidden="true"
      />

      {/* Modal */}
      <div className="fixed inset-0 z-[1000] overflow-y-auto">
        <div className="flex min-h-full items-center justify-center p-4">
          <div
            ref={modalRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
            className={`
              relative w-full ${sizeClass}
              bg-background-secondary border border-border
              rounded-none shadow-none
              animate-slide-up
              ${className}
            `.trim()}
          >
            {/* Header */}
            <div className="flex items-start justify-between p-6 pb-4 border-b border-border">
              <h2 id="modal-title" className="text-xl font-semibold text-foreground">
                {title}
              </h2>
              <button
                ref={closeButtonRef}
                onClick={onClose}
                aria-label="Close modal"
                className="
                  ml-4 p-1 rounded-none
                  text-foreground-secondary hover:text-foreground
                  hover:bg-brand-underground-lighter
                  focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                  transition-none
                  min-h-touch min-w-touch
                "
              >
                <span className="text-2xl" aria-hidden="true">
                  ×
                </span>
              </button>
            </div>

            {/* Body */}
            <div className="p-6 text-foreground">{children}</div>

            {/* Footer */}
            {footer && (
              <div className="flex items-center justify-end gap-3 p-6 pt-4 border-t border-border">
                {footer}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

/**
 * ConfirmModal - Pre-configured modal for confirmations
 */
export interface ConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'danger' | 'primary';
  isLoading?: boolean;
}

export function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'primary',
  isLoading = false,
}: ConfirmModalProps) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={isLoading}>
            {cancelText}
          </Button>
          <Button variant={variant} onClick={onConfirm} isLoading={isLoading}>
            {confirmText}
          </Button>
        </>
      }
    >
      <p className="text-foreground-secondary">{message}</p>
    </Modal>
  );
}
