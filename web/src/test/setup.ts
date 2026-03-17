import { expect, afterEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';
import * as matchers from '@testing-library/jest-dom/matchers';
import { axe } from 'vitest-axe';
import * as axeMatchers from 'vitest-axe/matchers';

// Extend Vitest's expect with jest-dom matchers
expect.extend(matchers);

// Extend Vitest's expect with axe accessibility matchers
expect.extend(axeMatchers);

// Export axe for use in tests
export { axe };

// Mock fetch globally for tests
// This prevents test failures when using relative URLs in authStore.logout()
// and supports i18next-http-backend which expects Response.text()
global.fetch = vi.fn(() => {
  // Mock response object with both json() and text() methods
  const mockResponse = {
    ok: true,
    status: 200,
    json: async () => ({}),
    text: async () => '{}', // Support i18next-http-backend
  } as Response;

  return Promise.resolve(mockResponse);
});

// Mock react-i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: {
      language: 'en',
      changeLanguage: vi.fn(),
    },
  }),
  initReactI18next: {
    type: '3rdParty',
    init: vi.fn(),
  },
}));

// Mock IntersectionObserver for OptimizedImage component
global.IntersectionObserver = class IntersectionObserver {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
  
  constructor(callback: IntersectionObserverCallback) {
    // Immediately trigger intersection for all observed elements
    // Use queueMicrotask for predictable timing in tests
    queueMicrotask(() => {
      callback(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        this as any
      );
    });
  }
} as any;

// Cleanup after each test
afterEach(() => {
  cleanup();
  // Reset fetch mock between tests
  vi.clearAllMocks();
});
