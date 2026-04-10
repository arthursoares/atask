import { test, expect } from './fixtures';

/**
 * Smoke tests for the atask-v4 frontend. Verifies the React shell
 * loads against a mocked Tauri IPC layer and the main navigation is
 * reachable. These are fast UI-only checks — full Tauri integration
 * tests live in tests/e2e/ and run via WDIO.
 */

test.describe('frontend smoke', () => {
  test('app shell renders with sidebar nav', async ({ mockPage: page }) => {
    await page.goto('/');

    // The sidebar is a <nav aria-label="Main navigation"> landmark
    // (added in T2.3) — this is the anchor that tells us the shell
    // mounted and React rendered past the initial loading state.
    const nav = page.getByRole('navigation', { name: 'Main navigation' });
    await expect(nav).toBeVisible();

    // Core view entries should all be reachable by accessible name.
    await expect(nav.getByRole('button', { name: /inbox/i })).toBeVisible();
    await expect(nav.getByRole('button', { name: /today/i })).toBeVisible();
    await expect(nav.getByRole('button', { name: /upcoming/i })).toBeVisible();
    await expect(nav.getByRole('button', { name: /someday/i })).toBeVisible();
    await expect(nav.getByRole('button', { name: /logbook/i })).toBeVisible();
  });

  test('empty inbox shows onboarding hint', async ({ mockPage: page }) => {
    await page.goto('/');

    // The Inbox is the default view. With zero tasks it renders the
    // EmptyState with our onboarding hint added in T3.6.
    await expect(page.getByText('Inbox is empty')).toBeVisible();
    await expect(page.getByText(/capture a new task/i)).toBeVisible();
  });

  test('toolbar search button opens search overlay, not command palette', async ({ mockPage: page }) => {
    await page.goto('/');

    // Regression guard for T1.1 — clicking the magnifier must open the
    // SearchOverlay, not the CommandPalette. Both overlays share the
    // same .cmd-* class names (same popover shell), so we discriminate
    // on the input placeholder text: "Search tasks..." vs "Type a
    // command or search...".
    await page.getByRole('button', { name: /search tasks/i }).click();

    await expect(page.getByPlaceholder('Search tasks...')).toBeVisible();
    await expect(page.getByPlaceholder('Type a command or search...')).toHaveCount(0);
  });
});
