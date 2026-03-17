import { useEffect, useRef, useCallback } from 'react';
import type { Scene, Event } from '../types/scene';

/**
 * Props for DetailPanel component
 */
export interface DetailPanelProps {
  /**
   * Whether the panel is open
   */
  isOpen: boolean;

  /**
   * Handler called when panel should close
   */
  onClose: () => void;

  /**
   * The scene or event to display
   */
  entity: Scene | Event | null;

  /**
   * Whether the panel is loading data
   */
  loading?: boolean;

  /**
   * Optional analytics callback for tracking panel events
   */
  onAnalyticsEvent?: (eventName: string, data?: Record<string, unknown>) => void;
}

/**
 * DetailPanel - Sliding side panel for scene/event details
 *
 * Features:
 * - Accessible keyboard focus trap
 * - ESC key to close
 * - Backdrop click to close
 * - Slide-in animation
 * - Privacy-first (no precise coords without consent)
 *
 * Accessibility:
 * - role="dialog"
 * - aria-modal="true"
 * - aria-labelledby for title
 * - Focus trapped within panel when open
 * - Returns focus to previously focused element on close
 */
export function DetailPanel({
  isOpen,
  onClose,
  entity,
  loading = false,
  onAnalyticsEvent,
}: DetailPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Track focus before panel opens
  useEffect(() => {
    if (isOpen) {
      previousFocusRef.current = document.activeElement as HTMLElement;

      // Emit analytics event
      if (onAnalyticsEvent && entity) {
        onAnalyticsEvent('detail_panel_open', {
          entity_type: 'scene_id' in entity ? 'event' : 'scene',
          entity_id: entity.id,
        });
      }

      // Focus close button after animation
      setTimeout(() => {
        closeButtonRef.current?.focus();
      }, 300);
    } else if (previousFocusRef.current) {
      // Return focus when closing
      previousFocusRef.current.focus();

      // Emit analytics event
      if (onAnalyticsEvent && entity) {
        onAnalyticsEvent('detail_panel_close', {
          entity_type: 'scene_id' in entity ? 'event' : 'scene',
          entity_id: entity.id,
        });
      }
    }
  }, [isOpen, entity, onAnalyticsEvent]);

  // Handle ESC key
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        e.preventDefault();
        onClose();
      }
    },
    [isOpen, onClose]
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  // Focus trap
  useEffect(() => {
    if (!isOpen || !panelRef.current) return;

    const panel = panelRef.current;
    const focusableElements = panel.querySelectorAll<HTMLElement>(
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

    panel.addEventListener('keydown', handleTabKey as EventListener);
    return () => panel.removeEventListener('keydown', handleTabKey as EventListener);
  }, [isOpen]);

  // Prevent body scroll when panel is open
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

  const isScene = entity && !('scene_id' in entity);
  const entityType = isScene ? 'scene' : 'event';

  return (
    <>
      {/* Backdrop */}
      <div
        className="detail-panel-backdrop fixed inset-0 bg-black/50 z-[999] animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="detail-panel-title"
        className="detail-panel fixed top-0 right-0 bottom-0 w-full max-w-[min(400px,100vw)] bg-background-secondary text-foreground z-[1000] overflow-y-auto border-l border-border shadow-none animate-slide-in"
      >
        {/* Header */}
        <div className="p-4 pb-6 border-b border-border flex items-start justify-between">
          <div className="flex-1">
            <div
              className={`text-xs uppercase tracking-wide mb-2 font-semibold ${isScene ? 'text-neon-cyan' : 'text-neon-magenta'}`}
            >
              {entityType}
            </div>
            {loading ? (
              <div className="text-xl font-semibold">Loading...</div>
            ) : (
              <h2 id="detail-panel-title" className="m-0 text-xl font-semibold leading-tight">
                {entity?.name || 'Unknown'}
              </h2>
            )}
          </div>

          <button
            ref={closeButtonRef}
            onClick={onClose}
            aria-label="Close detail panel"
            className="ml-4 p-2 min-w-touch min-h-touch bg-transparent border-0 text-foreground cursor-pointer text-2xl leading-none opacity-70 hover:opacity-100 transition-none focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary"
          >
            ×
          </button>
        </div>

        {/* Content */}
        {loading ? (
          <div className="p-4 text-center text-foreground-secondary">Loading details...</div>
        ) : entity ? (
          <div className="p-4">
            {/* Description */}
            {entity.description && (
              <div className="mb-6">
                <p className="m-0 leading-relaxed text-foreground-secondary">
                  {entity.description}
                </p>
              </div>
            )}

            {/* Tags */}
            {isScene && (entity as Scene).tags && (entity as Scene).tags!.length > 0 && (
              <div className="mb-6">
                <h3 className="m-0 mb-3 text-sm font-semibold uppercase tracking-wide text-foreground-muted">
                  Tags
                </h3>
                <div className="flex flex-wrap gap-2">
                  {(entity as Scene).tags!.map((tag) => (
                    <span
                      key={tag}
                      className="px-3 py-1 bg-background border border-border rounded-none text-sm text-foreground-secondary"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Location Privacy Notice */}
            <div className="mt-6 p-4 bg-background rounded-none border border-border text-sm text-foreground-secondary">
              <div className="flex items-center gap-2">
                <span aria-hidden="true">⌖</span>
                <span>
                  {entity.allow_precise
                    ? 'Precise location shared'
                    : 'Approximate location (privacy preserved)'}
                </span>
              </div>
            </div>

            {/* Placeholder for future features */}
            <div className="mt-6 p-4 bg-background rounded-none border border-border text-sm text-foreground-secondary text-center">
              <p className="m-0">
                More features coming soon:
                <br />
                • Trust score
                <br />
                • Upcoming events
                <br />
                • Join stream
                <br />• View posts
              </p>
            </div>
          </div>
        ) : null}
      </div>
    </>
  );
}
