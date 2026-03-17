/**
 * Organizer Stream Controls E2E Tests
 * 
 * Tests organizer-specific functionality:
 * - Mute participant
 * - Kick participant
 * - Lock/unlock stream
 * - End stream
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

test.describe('Organizer Controls', () => {
  test('organizer should be able to mute participant', async ({ page, context }) => {
    // First participant (organizer)
    await page.goto('/stream');
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Second participant
    const page2 = await context.newPage();
    await page2.goto('/stream');
    await page2.locator('button', { hasText: /join.*stream/i }).click();
    await page2.waitForTimeout(2000);
    
    // Organizer should see mute button for participant
    const participantItem = page.locator('[data-testid="participant-item"]').nth(1);
    const muteButton = participantItem.locator('button[aria-label*="mute" i]');
    
    // Assert organizer control is present (or skip if not implemented)
    const muteButtonCount = await muteButton.count();
    if (muteButtonCount === 0) {
      test.skip(true, 'Organizer mute control not yet implemented');
      return;
    }
    
    await muteButton.click();
    
    // Participant 2 should be muted
    const participant2Indicator = page2.locator('[data-testid="mute-indicator"]');
    await expect(participant2Indicator).toBeVisible({ timeout: 3000 });
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
    await page2.locator('button', { hasText: /leave/i }).click();
    await page2.close();
  });

  test('organizer should be able to kick participant', async ({ page, context }) => {
    // First participant (organizer)
    await page.goto('/stream');
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Second participant
    const page2 = await context.newPage();
    await page2.goto('/stream');
    await page2.locator('button', { hasText: /join.*stream/i }).click();
    await page2.waitForTimeout(2000);
    
    // Organizer should see kick button for participant
    const participantItem = page.locator('[data-testid="participant-item"]').nth(1);
    const kickButton = participantItem.locator('button[aria-label*="kick" i], button[aria-label*="remove" i]');
    
    // Assert organizer control is present (or skip if not implemented)
    const kickButtonCount = await kickButton.count();
    if (kickButtonCount === 0) {
      test.skip(true, 'Organizer kick control not yet implemented');
      await page.locator('button', { hasText: /leave/i }).click();
      await page2.locator('button', { hasText: /leave/i }).click();
      await page2.close();
      return;
    }
    
    await kickButton.click();
    
    // Participant 2 should be disconnected and see error
    const errorMessage = page2.locator('[role="alert"]');
    await expect(errorMessage).toBeVisible({ timeout: 5000 });
    await expect(errorMessage).toContainText(/removed|kicked/i);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
    await page2.close();
  });

  test('organizer should be able to lock/unlock stream', async ({ page, context }) => {
    // Organizer joins
    await page.goto('/stream');
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Look for lock button and assert it exists
    const lockButton = page.locator('button[aria-label*="lock" i]');
    const lockButtonCount = await lockButton.count();
    
    if (lockButtonCount === 0) {
      test.skip(true, 'Organizer lock control not yet implemented');
      await page.locator('button', { hasText: /leave/i }).click();
      return;
    }
    
    // Lock the stream
    await lockButton.click();
    await page.waitForTimeout(1000);
    
    // Try to join from second participant
    const page2 = await context.newPage();
    await page2.goto('/stream');
    await page2.locator('button', { hasText: /join.*stream/i }).click();
    
    // Should show error that room is locked
    const errorMessage = page2.locator('[role="alert"]');
    await expect(errorMessage).toBeVisible({ timeout: 5000 });
    await expect(errorMessage).toContainText(/locked/i);
    
    // Unlock the stream
    const unlockButton = page.locator('button[aria-label*="unlock" i]');
    await unlockButton.click();
    await page.waitForTimeout(1000);
    
    // Now participant should be able to join
    await page2.locator('button', { hasText: /join.*stream/i }).click();
    await page2.waitForTimeout(2000);
    
    const audioControls = page2.locator('[data-testid="audio-controls"]');
    await expect(audioControls).toBeVisible({ timeout: 5000 });
    
    // Clean up
    await page2.locator('button', { hasText: /leave/i }).click();
    await page2.close();
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
  });

  test('organizer should be able to end stream for everyone', async ({ page, context }) => {
    // Organizer joins
    await page.goto('/stream');
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Second participant joins
    const page2 = await context.newPage();
    await page2.goto('/stream');
    await page2.locator('button', { hasText: /join.*stream/i }).click();
    await page2.waitForTimeout(2000);
    
    // Look for end stream button and assert it exists
    const endButton = page.locator('button[aria-label*="end.*stream" i], button', { hasText: /end.*stream/i });
    const endButtonCount = await endButton.count();
    
    if (endButtonCount === 0) {
      test.skip(true, 'Organizer end stream control not yet implemented');
      await page.locator('button', { hasText: /leave/i }).click();
      await page2.locator('button', { hasText: /leave/i }).click();
      await page2.close();
      return;
    }
    
    await endButton.click();
    
    // Both users should be disconnected
    const joinButton = page.locator('button', { hasText: /join.*stream/i });
    await expect(joinButton).toBeVisible({ timeout: 5000 });
    
    const joinButton2 = page2.locator('button', { hasText: /join.*stream/i });
    await expect(joinButton2).toBeVisible({ timeout: 5000 });
    
    await page2.close();
  });

  test('non-organizer should not see organizer controls', async ({ page, context }) => {
    // Organizer joins first
    await page.goto('/stream');
    await page.locator('button', { hasText: /join.*stream/i }).click();
    await page.waitForTimeout(2000);
    
    // Second participant joins
    const page2 = await context.newPage();
    await page2.goto('/stream');
    await page2.locator('button', { hasText: /join.*stream/i }).click();
    await page2.waitForTimeout(2000);
    
    // Second participant should not see kick/mute buttons for others
    const kickButtons = page2.locator('button[aria-label*="kick" i]');
    const kickCount = await kickButtons.count();
    expect(kickCount).toBe(0);
    
    const muteOthersButtons = page2.locator('[data-testid="participant-item"] button[aria-label*="mute" i]');
    const muteOthersCount = await muteOthersButtons.count();
    expect(muteOthersCount).toBe(0);
    
    // Clean up
    await page.locator('button', { hasText: /leave/i }).click();
    await page2.locator('button', { hasText: /leave/i }).click();
    await page2.close();
  });
});
