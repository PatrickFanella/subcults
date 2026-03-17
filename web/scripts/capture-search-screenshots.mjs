/**
 * Screenshot script for search bar demo
 */

import { chromium } from '@playwright/test';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

async function captureScreenshots() {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
  });
  const page = await context.newPage();

  // Navigate to the demo page
  await page.goto('http://localhost:5173/demo.html');
  
  // Wait for page to load
  await page.waitForLoadState('networkidle');
  
  // Screenshot 1: Initial search bar
  await page.screenshot({ 
    path: join(__dirname, 'screenshots', 'search-bar-initial.png'),
    fullPage: true 
  });
  
  console.log('Screenshot 1: Initial state captured');

  // Screenshot 2: Search bar with focus (should show history if any)
  const searchInput = page.locator('input[role="combobox"]').first();
  await searchInput.click();
  await page.waitForTimeout(500);
  
  await page.screenshot({ 
    path: join(__dirname, 'screenshots', 'search-bar-focused.png'),
    fullPage: true 
  });
  
  console.log('Screenshot 2: Focused state captured');

  // Screenshot 3: Typing a search query
  await searchInput.fill('test scene');
  await page.waitForTimeout(1000); // Wait for debounce and results
  
  await page.screenshot({ 
    path: join(__dirname, 'screenshots', 'search-bar-with-query.png'),
    fullPage: true 
  });
  
  console.log('Screenshot 3: Search query state captured');

  // Add to history by clicking clear (simulating a search)
  const clearButton = page.locator('button[aria-label="Clear search"]');
  if (await clearButton.isVisible()) {
    await clearButton.click();
  }

  // Screenshot 4: Show keyboard shortcut info
  await page.goto('http://localhost:5173/demo.html#features');
  await page.waitForTimeout(500);
  
  await page.screenshot({ 
    path: join(__dirname, 'screenshots', 'search-bar-features.png'),
    fullPage: true 
  });
  
  console.log('Screenshot 4: Features section captured');

  await browser.close();
  console.log('All screenshots captured successfully!');
}

captureScreenshots().catch(console.error);
