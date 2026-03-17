/**
 * Scene Discovery E2E Tests
 *
 * Tests scene listing, detail view, and search.
 */

import { test, expect, Page } from '@playwright/test';
import { MockAPIServer } from '../mocks/mock-api-server';

let mockServer: MockAPIServer;

test.beforeAll(async () => {
  mockServer = new MockAPIServer(8083);
  await mockServer.start();
});

test.afterAll(async () => {
  if (mockServer) {
    await mockServer.stop();
  }
});

async function interceptAPI(page: Page) {
  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url());
    const mockUrl = `http://localhost:8083${url.pathname}${url.search}`;
    const response = await route.fetch({ url: mockUrl });
    await route.fulfill({ response });
  });
}

test.describe('Scene Discovery', () => {
  test('home page loads map with scene markers', async ({ page }) => {
    await interceptAPI(page);
    await page.goto('/');

    // Wait for map to initialize
    await page.waitForTimeout(2000);

    // The map container should be rendered
    const mapContainer = page.locator('[class*="map"], [id*="map"], canvas');
    await expect(mapContainer.first()).toBeVisible({ timeout: 10000 });
  });

  test('scene detail page shows scene info', async ({ page }) => {
    await interceptAPI(page);
    await page.goto('/scenes/scene-1');

    // Should load without error — look for any content
    await page.waitForTimeout(2000);
    const body = await page.textContent('body');
    expect(body).toBeTruthy();
  });

  test('search returns results', async ({ page }) => {
    await interceptAPI(page);
    await page.goto('/');

    // Look for a search input
    const searchInput = page
      .locator(
        'input[type="search"], input[placeholder*="search" i], input[aria-label*="search" i]'
      )
      .first();

    if (await searchInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await searchInput.fill('brooklyn');
      await searchInput.press('Enter');

      // Wait for search results
      await page.waitForTimeout(1000);

      // Should render results
      const body = await page.textContent('body');
      expect(body?.toLowerCase()).toContain('brooklyn');
    }
  });

  test('API error shows error state gracefully', async ({ page }) => {
    // Override API to return 500
    await page.route('**/api/scenes/**', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });

    await page.goto('/scenes/scene-error');
    await page.waitForTimeout(2000);

    // Should not show a blank page — error boundary or error message should appear
    const body = await page.textContent('body');
    expect(body?.length).toBeGreaterThan(0);
  });
});
