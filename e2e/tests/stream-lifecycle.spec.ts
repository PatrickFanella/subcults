/**
 * Stream Lifecycle E2E Tests
 * 
 * Tests the complete stream lifecycle:
 * - Create stream
 * - Join stream
 * - Leave stream
 * - End stream
 */

import { test, expect } from '@playwright/test';
import { MockAPIServer } from '../mocks/mock-api-server';

let mockServer: MockAPIServer;

test.beforeAll(async () => {
  // Start mock servers
  mockServer = new MockAPIServer(8080, 7880);
  await mockServer.start();
});

test.afterAll(async () => {
  // Stop mock servers
  if (mockServer) {
    await mockServer.stop();
  }
});

test.describe('Stream Lifecycle', () => {
  test('should create, join, leave, and end stream', async ({ page }) => {
    // Navigate to stream page
    await page.goto('/stream');
    
    // Wait for page to load
    await expect(page.locator('h1')).toContainText('Stream', { timeout: 10000 });
    
    // Create/Join stream
    const joinButton = page.locator('button', { hasText: /join.*stream/i });
    await expect(joinButton).toBeVisible();
    await joinButton.click();
    
    // Wait for connection
    await page.waitForTimeout(2000);
    
    // Verify we're connected (look for audio controls)
    const audioControls = page.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible({ timeout: 5000 });
    
    // Verify participant list shows local user
    const participantList = page.locator('[data-testid="participant-list"]');
    await expect(participantList).toBeVisible();
    
    // Leave stream
    const leaveButton = page.locator('button', { hasText: /leave/i });
    await expect(leaveButton).toBeVisible();
    await leaveButton.click();
    
    // Verify we've left (join button should be visible again)
    await expect(joinButton).toBeVisible({ timeout: 3000 });
    
    // Verify audio controls are hidden
    await expect(audioControls).not.toBeVisible();
  });

  test('should show connection indicator', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    const joinButton = page.locator('button', { hasText: /join.*stream/i });
    await joinButton.click();
    
    // Wait for connection
    await page.waitForTimeout(2000);
    
    // Check for connection quality indicator
    const connectionIndicator = page.locator('[data-testid="connection-indicator"]');
    await expect(connectionIndicator).toBeVisible({ timeout: 5000 });
    
    // Should show quality level (excellent by default in mock)
    await expect(connectionIndicator).toContainText(/excellent|good|poor/i);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('should handle multiple participants', async ({ page, context }) => {
    // First participant (organizer)
    await page.goto('/stream');
    const joinButton = page.locator('button', { hasText: /join.*stream/i });
    await joinButton.click();
    await page.waitForTimeout(2000);
    
    // Open second tab for second participant
    const page2 = await context.newPage();
    await page2.goto('/stream');
    const joinButton2 = page2.locator('button', { hasText: /join.*stream/i });
    await joinButton2.click();
    await page2.waitForTimeout(2000);
    
    // First participant should see second participant
    const participantList = page.locator('[data-testid="participant-list"]');
    await expect(participantList).toContainText(/user-\d+|participant/i);
    
    // Second participant should see first participant
    const participantList2 = page2.locator('[data-testid="participant-list"]');
    await expect(participantList2).toContainText(/user-\d+|participant/i);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
    await page2.locator('button', { hasText: /leave/i }).click();
    await page2.close();
  });

  test('should persist volume settings', async ({ page }) => {
    await page.goto('/stream');
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Find volume slider
    const volumeSlider = page.locator('input[type="range"][aria-label*="volume" i]');
    await expect(volumeSlider).toBeVisible({ timeout: 5000 });
    
    // Change volume
    await volumeSlider.fill('75');
    
    // Leave and rejoin
    await page.locator('button', { hasText: /leave/i }).click();
    await page.waitForTimeout(1000);
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Volume should be persisted
    const volumeSlider2 = page.locator('input[type="range"][aria-label*="volume" i]');
    const volume = await volumeSlider2.inputValue();
    expect(volume).toBe('75');
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('should show latency overlay in dev mode', async ({ page }) => {
    // Enable dev mode by setting localStorage
    await page.goto('/stream');
    await page.evaluate(() => {
      localStorage.setItem('showLatencyOverlay', 'true');
    });
    await page.reload();
    
    // Join stream
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(3000);
    
    // Check for latency overlay
    const latencyOverlay = page.locator('[data-testid="latency-overlay"]');
    await expect(latencyOverlay).toBeVisible();
    
    // Should show latency values
    await expect(latencyOverlay).toContainText(/token|connect|audio/i);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });
});

test.describe('Stream Error Handling', () => {
  test('should handle connection errors gracefully', async ({ page }) => {
    await page.goto('/stream');
    
    // Mock a connection error by stopping the mock server temporarily
    await mockServer.stop();
    
    // Try to join
    const joinButton = page.locator('button', { hasText: /join.*stream/i });
    await joinButton.click();
    
    // Should show error message
    const errorMessage = page.locator('[role="alert"]');
    await expect(errorMessage).toBeVisible({ timeout: 10000 });
    
    // Restart server for other tests
    await mockServer.start();
  });
});
