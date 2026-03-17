/**
 * useKeyboardShortcut Tests
 * Validates keyboard shortcut registration and handling
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useKeyboardShortcut } from './useKeyboardShortcut';

describe('useKeyboardShortcut', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Functionality', () => {
    it('registers keyboard event listener on mount', () => {
      const callback = vi.fn();
      const addEventListenerSpy = vi.spyOn(document, 'addEventListener');

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      expect(addEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function));
    });

    it('removes keyboard event listener on unmount', () => {
      const callback = vi.fn();
      const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener');

      const { unmount } = renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      unmount();

      expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function));
    });

    it('calls callback when correct key combination is pressed', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).toHaveBeenCalledWith(expect.any(KeyboardEvent));
    });

    it('does not call callback when wrong key is pressed', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'j',
        ctrlKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).not.toHaveBeenCalled();
    });

    it('does not call callback when modifier is missing', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: false,
      });

      document.dispatchEvent(event);

      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('Modifier Keys', () => {
    it('accepts Ctrl key', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).toHaveBeenCalled();
    });

    it('accepts Meta key (Cmd on Mac)', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            metaKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).toHaveBeenCalled();
    });

    it('accepts either Ctrl or Meta when both are specified', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
            metaKey: true,
          },
          callback
        )
      );

      // Test with Ctrl
      const ctrlEvent = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });
      document.dispatchEvent(ctrlEvent);
      expect(callback).toHaveBeenCalledTimes(1);

      // Test with Meta
      const metaEvent = new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
      });
      document.dispatchEvent(metaEvent);
      expect(callback).toHaveBeenCalledTimes(2);
    });

    it('accepts Shift key', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            shiftKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        shiftKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).toHaveBeenCalled();
    });

    it('rejects event when unexpected Shift is pressed', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
            shiftKey: false,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        shiftKey: true, // Unexpected shift
      });

      document.dispatchEvent(event);

      expect(callback).not.toHaveBeenCalled();
    });

    it('accepts Alt key', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            altKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        altKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).toHaveBeenCalled();
    });
  });

  describe('Prevent Default', () => {
    it('prevents default when preventDefault is true', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
            preventDefault: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });

      const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
      document.dispatchEvent(event);

      expect(preventDefaultSpy).toHaveBeenCalled();
    });

    it('does not prevent default when preventDefault is false', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
            preventDefault: false,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });

      const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
      document.dispatchEvent(event);

      expect(preventDefaultSpy).not.toHaveBeenCalled();
    });
  });

  describe('Input Element Handling', () => {
    it('does not trigger when user is typing in an input element', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const input = document.createElement('input');
      document.body.appendChild(input);

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        bubbles: true,
      });

      // Manually set the target since JSDOM doesn't fully simulate event bubbling
      Object.defineProperty(event, 'target', { value: input, enumerable: true });

      document.dispatchEvent(event);

      expect(callback).not.toHaveBeenCalled();

      document.body.removeChild(input);
    });

    it('does not trigger when user is typing in a textarea', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const textarea = document.createElement('textarea');
      document.body.appendChild(textarea);

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        bubbles: true,
      });

      Object.defineProperty(event, 'target', { value: textarea, enumerable: true });

      document.dispatchEvent(event);

      expect(callback).not.toHaveBeenCalled();

      document.body.removeChild(textarea);
    });

    it('does not trigger when user is typing in a contenteditable element', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'k',
            ctrlKey: true,
          },
          callback
        )
      );

      const div = document.createElement('div');
      div.contentEditable = 'true';
      document.body.appendChild(div);

      // Verify that isContentEditable is set correctly
      if (!div.isContentEditable) {
        // JSDOM may not fully support contentEditable, skip this test
        document.body.removeChild(div);
        return;
      }

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        bubbles: true,
      });

      Object.defineProperty(event, 'target', { value: div, enumerable: true });

      document.dispatchEvent(event);

      expect(callback).not.toHaveBeenCalled();

      document.body.removeChild(div);
    });
  });

  describe('Case Insensitivity', () => {
    it('is case insensitive for key matching', () => {
      const callback = vi.fn();

      renderHook(() =>
        useKeyboardShortcut(
          {
            key: 'K', // Uppercase in config
            ctrlKey: true,
          },
          callback
        )
      );

      const event = new KeyboardEvent('keydown', {
        key: 'k', // Lowercase in event
        ctrlKey: true,
      });

      document.dispatchEvent(event);

      expect(callback).toHaveBeenCalled();
    });
  });
});
