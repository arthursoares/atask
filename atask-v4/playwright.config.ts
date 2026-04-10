import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for atask-v4 frontend UI tests.
 *
 * Scope: tests the React + nanostores frontend against the Vite dev
 * server at http://localhost:1420. Tauri IPC (window.__TAURI_INTERNALS__)
 * is stubbed in the test setup (see tests/playwright/fixtures.ts) so the
 * frontend can load without a running Rust backend.
 *
 * Full end-to-end tests that drive the native Tauri window live in
 * tests/e2e/ and run via WebdriverIO with tauri-driver / tauri-wd. Use
 * Playwright for fast UI iteration and store/selector assertions;
 * use WDIO for integration coverage against the real Rust sync worker.
 */
export default defineConfig({
  testDir: './tests/playwright',
  testMatch: '**/*.spec.ts',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? 'github' : [['list'], ['html', { open: 'never' }]],

  use: {
    baseURL: 'http://localhost:1420',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Auto-start Vite dev for local runs. In CI you may want to prebuild
  // and serve a static bundle instead; wire that up when needed.
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:1420',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});
