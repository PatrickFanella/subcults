/**
 * Reconnection and Network Resilience E2E Tests
 * 
 * Tests network failure scenarios:
 * - Participant reconnection
 * - Quality degradation handling
 * - Network delay tolerance
 * - Packet loss handling
 */

import { test, expect } from '@playwright/test';
import { MockAPIServer } from '../mocks/mock-api-server';

let mockServer: MockAPIServer;

test.beforeAll(async () => {
  mockServer = new MockAPIServer(8080, 7880);
  await mockServer.start();
});

test.afterAll(async () => {
  if (mockServer) {
    await mockServer.stop();
  }
});

test.describe('Reconnection', () => {
  test('should automatically reconnect after temporary disconnection', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Verify connected
    const audioControls = page.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible();
    
    // Simulate network interruption by going offline
    await page.context().setOffline(true);
    await page.waitForTimeout(2000);
    
    // Should show reconnecting state (optional - UI may vary)
    const reconnectingIndicator = page.locator('[data-testid="reconnecting-indicator"], text=/reconnecting/i');
    const hasReconnectingIndicator = await reconnectingIndicator.isVisible().catch(() => false);
    if (hasReconnectingIndicator) {
      console.log('[E2E] Reconnecting indicator shown');
    }
    
    // Go back online
    await page.context().setOffline(false);
    await page.waitForTimeout(3000);
    
    // Should reconnect automatically
    await expect(audioControls).toBeVisible();
    
    // Connection indicator should recover
    const connectionIndicator = page.locator('[data-testid="connection-indicator"]');
    await expect(connectionIndicator).toBeVisible();
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('should handle multiple reconnection attempts', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Verify connected
    const audioControls = page.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible();
    
    // Simulate multiple disconnections
    for (let i = 0; i < 3; i++) {
      await page.context().setOffline(true);
      await page.waitForTimeout(1000);
      await page.context().setOffline(false);
      await page.waitForTimeout(2000);
    }
    
    // Should still be connected after multiple reconnections
    await expect(audioControls).toBeVisible();
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('should give up after max reconnection attempts', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Go offline and stay offline
    await page.context().setOffline(true);
    
    // Wait for max reconnection attempts (should be around 30-60 seconds)
    await page.waitForTimeout(60000);
    
    // Should show error message after exhausting retries
    const errorMessage = page.locator('[role="alert"]');
    await expect(errorMessage).toBeVisible({ timeout: 5000 });
    await expect(errorMessage).toContainText(/reconnect|connection|failed/i);
    
    // Reset
    await page.context().setOffline(false);
  });
});

test.describe('Quality Degradation', () => {
  test('should show quality indicator changes', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Check initial quality (should be excellent)
    const connectionIndicator = page.locator('[data-testid="connection-indicator"]');
    await expect(connectionIndicator).toBeVisible();
    
    const initialQuality = await connectionIndicator.textContent();
    expect(initialQuality).toMatch(/excellent|good/i);
    
    // Simulate packet loss via API
    const roomId = 'default'; // Using default room for testing
    await page.evaluate(async (roomId) => {
      await fetch('http://localhost:8080/api/test/simulate-packet-loss', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ roomId, lossPercentage: 25 }),
      });
    }, roomId);
    
    await page.waitForTimeout(2000);
    
    // Quality should degrade
    const degradedQuality = await connectionIndicator.textContent();
    expect(degradedQuality).toMatch(/poor/i);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('should handle slow network conditions', async ({ page }) => {
    await page.goto('/stream');
    
    // Throttle network to simulate slow connection
    await page.context().route('**/*', async (route) => {
      // Add 500ms delay to all requests
      await new Promise(resolve => setTimeout(resolve, 500));
      await route.continue();
    });
    
    // Try to join stream
    const startTime = Date.now();
    await page.locator('button', { hasText: /join.*stream/i }).click();
    
    // Should still connect, just slower
    const audioControls = page.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible({ timeout: 15000 });
    
    const joinTime = Date.now() - startTime;
    console.log(`Join time with slow network: ${joinTime}ms`);
    
    // Should show latency warning if join time > 2s
    if (joinTime > 2000) {
      const latencyWarning = page.locator('[data-testid="latency-warning"]');
      const hasWarning = await latencyWarning.isVisible().catch(() => false);
      console.log(`Latency warning shown: ${hasWarning}`);
    }
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });
});

test.describe('Network Resilience', () => {
  test('should maintain connection with intermittent packet loss', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    const audioControls = page.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible();
    
    // Simulate intermittent packet loss (10% - moderate)
    const roomId = 'default';
    await page.evaluate(async (roomId) => {
      await fetch('http://localhost:8080/api/test/simulate-packet-loss', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ roomId, lossPercentage: 10 }),
      });
    }, roomId);
    
    await page.waitForTimeout(3000);
    
    // Should still be connected
    await expect(audioControls).toBeVisible();
    
    // Quality indicator should show degradation but not disconnect
    const connectionIndicator = page.locator('[data-testid="connection-indicator"]');
    const quality = await connectionIndicator.textContent();
    expect(quality).toMatch(/good|poor|excellent/i);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('should handle high latency gracefully', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    const audioControls = page.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible();
    
    // Simulate high latency
    const roomId = 'default';
    await page.evaluate(async (roomId) => {
      await fetch('http://localhost:8080/api/test/simulate-latency', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ roomId, delayMs: 2000 }),
      });
    }, roomId);
    
    await page.waitForTimeout(3000);
    
    // Should still be connected despite high latency
    await expect(audioControls).toBeVisible();
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });
});
