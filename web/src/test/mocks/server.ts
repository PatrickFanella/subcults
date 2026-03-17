/**
 * MSW Test Server Setup
 * Configures Mock Service Worker for integration tests
 */

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

/**
 * Create MSW server instance with default handlers
 */
export const server = setupServer(...handlers);

/**
 * Setup function to be called in test setup
 * Starts the server and configures global listeners
 */
export function setupMockServer() {
  // Start server before all tests
  beforeAll(() => {
    server.listen({ onUnhandledRequest: 'warn' });
  });

  // Reset handlers after each test
  afterEach(() => {
    server.resetHandlers();
  });

  // Clean up after all tests
  afterAll(() => {
    server.close();
  });
}
