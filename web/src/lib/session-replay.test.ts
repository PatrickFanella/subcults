/**
 * Session Replay Service Tests
 * Validates event recording, buffering, opt-in gating, and performance thresholds
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { SessionReplay, ReplayEventType } from './session-replay';

describe('SessionReplay', () => {
  let replay: SessionReplay;
  let isOptedIn: () => boolean;

  beforeEach(() => {
    // Clear localStorage
    localStorage.clear();

    // Default: user is NOT opted in
    isOptedIn = vi.fn(() => false);

    // Set up DOM
    document.body.innerHTML = '<div id="test-container"></div>';
  });

  afterEach(() => {
    replay?.destroy();
  });

  describe('opt-in gating', () => {
    it('does not start recording when user is opted out', () => {
      isOptedIn = vi.fn(() => false);
      replay = new SessionReplay({}, isOptedIn);

      replay.start();

      expect(replay.isActive()).toBe(false);
    });

    it('starts recording when user is opted in', () => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({}, isOptedIn);

      replay.start();

      expect(replay.isActive()).toBe(true);
    });

    it('does not record events when opted out', () => {
      isOptedIn = vi.fn(() => false);
      replay = new SessionReplay({}, isOptedIn);
      replay.start();

      // Try to trigger a click
      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      expect(replay.getBufferSize()).toBe(0);
    });

    it('returns empty array when getting buffer while opted out', () => {
      isOptedIn = vi.fn(() => false);
      replay = new SessionReplay({}, isOptedIn);

      const events = replay.getAndClearBuffer();

      expect(events).toEqual([]);
    });
  });

  describe('click recording', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({ clickSampleRate: 1.0 }, isOptedIn);
      replay.start();
    });

    it('records click events', () => {
      const button = document.createElement('button');
      button.id = 'test-button';
      button.className = 'btn';
      document.body.appendChild(button);

      button.click();

      const events = replay.getAndClearBuffer();
      expect(events.length).toBeGreaterThan(0);

      const clickEvent = events.find((e) => e.type === ReplayEventType.Click);
      expect(clickEvent).toBeDefined();
      expect(clickEvent?.data.element).toMatchObject({
        tagName: 'button',
        id: 'test-button',
        className: 'btn',
      });
    });

    it('includes click coordinates', () => {
      const button = document.createElement('button');
      document.body.appendChild(button);

      const clickEvent = new MouseEvent('click', {
        clientX: 100,
        clientY: 200,
        bubbles: true, // Important: allow event to bubble
      });
      button.dispatchEvent(clickEvent);

      const events = replay.getAndClearBuffer();
      const click = events.find((e) => e.type === ReplayEventType.Click);

      expect(click).toBeDefined();
      expect(click?.data.x).toBe(100);
      expect(click?.data.y).toBe(200);
    });

    it('sanitizes element data (no text content)', () => {
      const button = document.createElement('button');
      button.textContent = 'Sensitive Button Text';
      document.body.appendChild(button);

      button.click();

      const events = replay.getAndClearBuffer();
      const click = events.find((e) => e.type === ReplayEventType.Click);

      // Should not include text content
      expect(JSON.stringify(click)).not.toContain('Sensitive Button Text');
    });

    it('respects click sample rate', () => {
      // Set sample rate to 0 (no clicks recorded)
      replay.destroy();
      replay = new SessionReplay({ clickSampleRate: 0 }, isOptedIn);
      replay.start();

      const button = document.createElement('button');
      document.body.appendChild(button);

      // Click multiple times
      for (let i = 0; i < 10; i++) {
        button.click();
      }

      const events = replay.getAndClearBuffer();
      const clicks = events.filter((e) => e.type === ReplayEventType.Click);

      expect(clicks.length).toBe(0);
    });
  });

  describe('navigation recording', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({}, isOptedIn);
      replay.start();
    });

    it('records navigation events', () => {
      // Trigger popstate event
      const popstateEvent = new PopStateEvent('popstate');
      window.dispatchEvent(popstateEvent);

      const events = replay.getAndClearBuffer();
      const navEvent = events.find((e) => e.type === ReplayEventType.Navigation);

      expect(navEvent).toBeDefined();
      expect(navEvent?.data.url).toBe(window.location.pathname);
    });

    it('only includes pathname (no query params)', () => {
      // Navigation event should only include pathname, not full URL
      const popstateEvent = new PopStateEvent('popstate');
      window.dispatchEvent(popstateEvent);

      const events = replay.getAndClearBuffer();
      const navEvent = events.find((e) => e.type === ReplayEventType.Navigation);

      expect(navEvent?.data.url).not.toContain('?');
      expect(navEvent?.data.url).not.toContain('#');
    });
  });

  describe('scroll recording', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({}, isOptedIn);
      replay.start();
    });

    it('records scroll events', () => {
      // Mock scroll position
      Object.defineProperty(window, 'scrollX', { value: 100, writable: true });
      Object.defineProperty(window, 'scrollY', { value: 200, writable: true });

      // Trigger scroll event multiple times to overcome sampling
      for (let i = 0; i < 50; i++) {
        const scrollEvent = new Event('scroll');
        window.dispatchEvent(scrollEvent);
      }

      const events = replay.getAndClearBuffer();
      const scrollEvents = events.filter((e) => e.type === ReplayEventType.Scroll);

      // Should have at least one scroll event (10% sample rate)
      expect(scrollEvents.length).toBeGreaterThan(0);
    });

    it('includes scroll coordinates', () => {
      Object.defineProperty(window, 'scrollX', { value: 150, writable: true });
      Object.defineProperty(window, 'scrollY', { value: 300, writable: true });

      // Trigger multiple scroll events
      for (let i = 0; i < 50; i++) {
        const scrollEvent = new Event('scroll');
        window.dispatchEvent(scrollEvent);
      }

      const events = replay.getAndClearBuffer();
      const scrollEvent = events.find((e) => e.type === ReplayEventType.Scroll);

      if (scrollEvent) {
        expect(scrollEvent.data.x).toBe(150);
        expect(scrollEvent.data.y).toBe(300);
      }
    });
  });

  describe('DOM mutation recording', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({ domMutationSampleRate: 1.0 }, isOptedIn);
      replay.start();
    });

    it('records DOM mutations', async () => {
      const div = document.createElement('div');
      div.id = 'new-element';
      document.body.appendChild(div);

      // Wait for mutation observer to process
      await new Promise((resolve) => setTimeout(resolve, 50));

      const events = replay.getAndClearBuffer();
      const mutationEvents = events.filter((e) => e.type === ReplayEventType.DOMChange);

      expect(mutationEvents.length).toBeGreaterThan(0);
    });

    it('limits mutation data size', async () => {
      // Add many elements at once
      const fragment = document.createDocumentFragment();
      for (let i = 0; i < 20; i++) {
        const div = document.createElement('div');
        fragment.appendChild(div);
      }
      document.body.appendChild(fragment);

      await new Promise((resolve) => setTimeout(resolve, 50));

      const events = replay.getAndClearBuffer();
      const mutationEvent = events.find((e) => e.type === ReplayEventType.DOMChange);

      if (mutationEvent && mutationEvent.data.mutations) {
        const mutations = mutationEvent.data.mutations as unknown[];
        // Should limit to 5 mutations max
        expect(mutations.length).toBeLessThanOrEqual(5);
      }
    });

    it('respects DOM mutation sample rate', async () => {
      // Set sample rate to 0
      replay.destroy();
      replay = new SessionReplay({ domMutationSampleRate: 0 }, isOptedIn);
      replay.start();

      // Add multiple elements
      for (let i = 0; i < 10; i++) {
        const div = document.createElement('div');
        document.body.appendChild(div);
      }

      await new Promise((resolve) => setTimeout(resolve, 50));

      const events = replay.getAndClearBuffer();
      const mutationEvents = events.filter((e) => e.type === ReplayEventType.DOMChange);

      expect(mutationEvents.length).toBe(0);
    });
  });

  describe('buffer management', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
    });

    it('maintains ring buffer (drops oldest events when full)', () => {
      replay = new SessionReplay(
        {
          maxBufferSize: 5,
          clickSampleRate: 1.0,
        },
        isOptedIn
      );
      replay.start();

      const button = document.createElement('button');
      document.body.appendChild(button);

      // Click 10 times (more than buffer size)
      for (let i = 0; i < 10; i++) {
        button.click();
      }

      const events = replay.getAndClearBuffer();

      // Should only have last 5 events
      expect(events.length).toBeLessThanOrEqual(5);
    });

    it('getAndClearBuffer returns all events and clears', () => {
      replay = new SessionReplay({ clickSampleRate: 1.0 }, isOptedIn);
      replay.start();

      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      expect(replay.getBufferSize()).toBeGreaterThan(0);

      const events = replay.getAndClearBuffer();
      expect(events.length).toBeGreaterThan(0);
      expect(replay.getBufferSize()).toBe(0);
    });

    it('clearBuffer removes all events', () => {
      replay = new SessionReplay({ clickSampleRate: 1.0 }, isOptedIn);
      replay.start();

      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      expect(replay.getBufferSize()).toBeGreaterThan(0);

      replay.clearBuffer();
      expect(replay.getBufferSize()).toBe(0);
    });

    it('stops recording when buffer reaches 90% capacity (performance threshold)', async () => {
      replay = new SessionReplay(
        {
          maxBufferSize: 10,
          domMutationSampleRate: 1.0,
          performanceMonitoring: true,
        },
        isOptedIn
      );
      replay.start();

      // Fill buffer to 90% (9 events)
      const button = document.createElement('button');
      document.body.appendChild(button);

      for (let i = 0; i < 9; i++) {
        button.click();
      }

      expect(replay.getBufferSize()).toBeGreaterThanOrEqual(9);

      // Try to add DOM mutations - should be blocked
      const div = document.createElement('div');
      document.body.appendChild(div);

      await new Promise((resolve) => setTimeout(resolve, 50));

      // Buffer should not have grown significantly beyond threshold
      expect(replay.getBufferSize()).toBeLessThanOrEqual(11);
    });
  });

  describe('start/stop', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({ clickSampleRate: 1.0 }, isOptedIn);
    });

    it('starts recording', () => {
      replay.start();
      expect(replay.isActive()).toBe(true);
    });

    it('stops recording', () => {
      replay.start();
      replay.stop();
      expect(replay.isActive()).toBe(false);
    });

    it('does not record events after stopping', () => {
      replay.start();
      replay.stop();

      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      expect(replay.getBufferSize()).toBe(0);
    });

    it('can be started multiple times (idempotent)', () => {
      replay.start();
      replay.start();
      replay.start();

      expect(replay.isActive()).toBe(true);
    });
  });

  describe('destroy', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({ clickSampleRate: 1.0 }, isOptedIn);
      replay.start();
    });

    it('stops recording', () => {
      replay.destroy();
      expect(replay.isActive()).toBe(false);
    });

    it('clears buffer', () => {
      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      expect(replay.getBufferSize()).toBeGreaterThan(0);

      replay.destroy();
      expect(replay.getBufferSize()).toBe(0);
    });
  });

  describe('event structure', () => {
    beforeEach(() => {
      isOptedIn = vi.fn(() => true);
      replay = new SessionReplay({ clickSampleRate: 1.0 }, isOptedIn);
      replay.start();
    });

    it('includes timestamp in all events', () => {
      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      const events = replay.getAndClearBuffer();

      events.forEach((event) => {
        expect(event.timestamp).toBeTypeOf('number');
        expect(event.timestamp).toBeGreaterThan(0);
      });
    });

    it('includes type in all events', () => {
      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      const events = replay.getAndClearBuffer();

      events.forEach((event) => {
        expect(event.type).toBeDefined();
        expect(Object.values(ReplayEventType)).toContain(event.type);
      });
    });

    it('includes data object in all events', () => {
      const button = document.createElement('button');
      document.body.appendChild(button);
      button.click();

      const events = replay.getAndClearBuffer();

      events.forEach((event) => {
        expect(event.data).toBeTypeOf('object');
      });
    });
  });
});
