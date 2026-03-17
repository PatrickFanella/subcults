/**
 * useKeyboardShortcut Hook
 * Hook for registering global keyboard shortcuts
 */

import { useEffect, useRef } from 'react';

export interface KeyboardShortcutOptions {
  /**
   * Key to listen for (e.g., 'k', 'Enter', 'Escape')
   */
  key: string;
  /**
   * Require Ctrl key (Windows/Linux)
   */
  ctrlKey?: boolean;
  /**
   * Require Meta key (Cmd on Mac)
   */
  metaKey?: boolean;
  /**
   * Require Shift key
   */
  shiftKey?: boolean;
  /**
   * Require Alt key
   */
  altKey?: boolean;
  /**
   * Prevent default browser behavior
   */
  preventDefault?: boolean;
}

/**
 * Hook for registering global keyboard shortcuts
 * Automatically cleans up event listeners on unmount
 *
 * @example
 * useKeyboardShortcut({
 *   key: 'k',
 *   ctrlKey: true,
 *   metaKey: true,
 *   preventDefault: true,
 * }, () => {
 *   console.log('Cmd/Ctrl+K pressed');
 * });
 */
export function useKeyboardShortcut(
  options: KeyboardShortcutOptions,
  callback: (event: KeyboardEvent) => void
): void {
  const {
    key,
    ctrlKey = false,
    metaKey = false,
    shiftKey = false,
    altKey = false,
    preventDefault = false,
  } = options;

  // Use a ref to store the latest callback without re-registering the event listener
  const callbackRef = useRef(callback);

  // Update the ref whenever callback changes
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Check if the key matches
      if (event.key.toLowerCase() !== key.toLowerCase()) {
        return;
      }

      // Check modifiers
      // For Cmd/Ctrl+K, we want either Ctrl OR Meta (not both required)
      const modifierMatch =
        (ctrlKey || metaKey ? event.ctrlKey || event.metaKey : true) &&
        (shiftKey ? event.shiftKey : !event.shiftKey) &&
        (altKey ? event.altKey : !event.altKey);

      if (!modifierMatch) {
        return;
      }

      // Don't trigger if user is typing in an input/textarea
      const target = event.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable
      ) {
        // Allow the shortcut only if we're not already focused on the search input
        // (we'll handle this by checking a specific ID or class if needed)
        return;
      }

      if (preventDefault) {
        event.preventDefault();
      }

      // Call the latest callback from the ref
      callbackRef.current(event);
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [key, ctrlKey, metaKey, shiftKey, altKey, preventDefault]);
}
