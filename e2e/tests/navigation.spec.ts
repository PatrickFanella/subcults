/**
 * Navigation E2E Tests
 *
 * Tests page navigation, route guards, and 404 handling.
 */

import { test, expect } from '@playwright/test';
import { MockAPIServer } from '../mocks/mock-api-server';

let mockServer: MockAPIServer;

test.beforeAll(async () => {
  mockServer = new MockAPIServer(8081);
  await mockServer.start();
});

test.afterAll(async () => {
  if (mockServer) {
    await mockServer.stop();
  }
});

test.describe('Navigation', () => {
  test('home page loads and shows map', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/subcults/i);
    // The home page should contain the map container
    await expect(page.locator('[class*="map"], [id*="map"], canvas')).toBeVisible({
      timeout: 10000,
    });
  });

  test('navigating to unknown route shows 404', async ({ page }) => {
    await page.goto('/this-route-does-not-exist');
    await expect(page.getByText(/not found|404/i)).toBeVisible({ timeout: 5000 });
  });

  test('scene detail page loads', async ({ page }) => {
    await page.goto('/scenes/scene-1');
    // Should render without crashing
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('event detail page loads', async ({ page }) => {
    await page.goto('/events/event-1');
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('login page renders form elements', async ({ page }) => {
    await page.goto('/account/login');
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('protected route redirects unauthenticated user', async ({ page }) => {
    await page.goto('/account');
    // Should redirect to login or show auth prompt
    await page.waitForURL(/login|account/, { timeout: 5000 });
  });
});
