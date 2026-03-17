import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Test Configuration for Stream Testing
 *
 * See https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests',

  // Maximum time one test can run
  timeout: 60 * 1000,

  // Test execution settings
  fullyParallel: false, // Stream tests should run sequentially to avoid port conflicts
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : 2,

  // Reporter configuration
  reporter: [
    ['html', { outputFolder: '../e2e-report' }],
    ['json', { outputFile: '../e2e-results.json' }],
    ['list'],
  ],

  // Shared settings for all tests
  use: {
    // Base URL for the application
    baseURL: 'http://localhost:5173',

    // Collect trace on failure
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video recording
    video: 'retain-on-failure',

    // Navigation timeout
    navigationTimeout: 30 * 1000,
  },

  // Configure projects for different browsers
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Permissions for microphone access in tests
        permissions: ['microphone'],
        launchOptions: {
          args: ['--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream'],
        },
      },
    },

    {
      name: 'firefox',
      use: {
        ...devices['Desktop Firefox'],
        permissions: ['microphone'],
        launchOptions: {
          firefoxUserPrefs: {
            'media.navigator.streams.fake': true,
            'media.navigator.permission.disabled': true,
          },
        },
      },
    },

    {
      name: 'webkit',
      use: {
        ...devices['Desktop Safari'],
        permissions: ['microphone'],
      },
    },

    // Mobile viewports for responsive testing
    {
      name: 'mobile-chrome',
      use: {
        ...devices['Pixel 5'],
        permissions: ['microphone'],
      },
    },
  ],

  // Web server configuration
  webServer: [
    {
      command: 'cd ../web && npm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
      env: {
        // Set LiveKit WebSocket URL for E2E tests (points to mock server)
        VITE_LIVEKIT_WS_URL: process.env.VITE_LIVEKIT_WS_URL ?? 'ws://localhost:7880',
      },
    },
  ],
});
