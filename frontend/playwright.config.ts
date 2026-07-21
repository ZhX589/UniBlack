import { defineConfig, devices } from '@playwright/test'

// Prefer system Google Chrome when available (channel: 'chrome').
// Set PLAYWRIGHT_USE_BUNDLED=1 to force Playwright-managed Chromium.
const useBundled = process.env.PLAYWRIGHT_USE_BUNDLED === '1'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 30_000,
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:3000',
    trace: 'on-first-retry',
    ...(useBundled ? {} : { channel: 'chrome' as const }),
  },
  projects: [
    {
      name: 'desktop-chrome',
      use: {
        ...devices['Desktop Chrome'],
        ...(useBundled ? {} : { channel: 'chrome' as const }),
        viewport: { width: 1280, height: 800 },
      },
    },
    {
      name: 'tablet',
      use: {
        ...devices['Desktop Chrome'],
        ...(useBundled ? {} : { channel: 'chrome' as const }),
        viewport: { width: 768, height: 1024 },
      },
    },
    {
      name: 'mobile',
      use: {
        ...devices['Desktop Chrome'],
        ...(useBundled ? {} : { channel: 'chrome' as const }),
        viewport: { width: 375, height: 812 },
      },
    },
  ],
  webServer: process.env.PLAYWRIGHT_SKIP_WEBSERVER
    ? undefined
    : {
        command: 'npm run dev',
        url: 'http://127.0.0.1:3000',
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
})
