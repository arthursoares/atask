import { test, expect, installTauriMock, type MockState } from './fixtures';

/**
 * Reproduction test for "starting a sidebar drag blanks the window".
 * Drives a synthetic pointer-down + threshold move on a project item
 * and asserts the React tree stays intact (sidebar nav landmark still
 * present after the drag begins).
 */
function stateWithProjectsAndAreas(): MockState {
  const now = '2026-04-10T00:00:00Z';
  return {
    tasks: [],
    projects: [
      {
        id: 'p-source',
        title: 'Source Project',
        notes: '',
        status: 0,
        color: '#888888',
        areaId: 'a-source',
        index: 0,
        completedAt: null,
        createdAt: now,
        updatedAt: now,
      },
      {
        id: 'p-target-1',
        title: 'Target Project 1',
        notes: '',
        status: 0,
        color: '#888888',
        areaId: 'a-target',
        index: 0,
        completedAt: null,
        createdAt: now,
        updatedAt: now,
      },
    ],
    areas: [
      { id: 'a-source', title: 'Source Area', index: 0, archived: false, createdAt: now, updatedAt: now },
      { id: 'a-target', title: 'Target Area', index: 1, archived: false, createdAt: now, updatedAt: now },
    ],
    sections: [],
    tags: [],
    taskTags: [],
    taskLinks: [],
    projectTags: [],
    checklistItems: [],
    activities: [],
    locations: [],
  };
}

test('sidebar project drag does not blank the window', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', (err) => errors.push(`pageerror: ${err.message}`));
  page.on('console', (msg) => {
    if (msg.type() === 'error') errors.push(`console.error: ${msg.text()}`);
  });

  await installTauriMock(page, stateWithProjectsAndAreas());
  await page.goto('/');

  const nav = page.getByRole('navigation', { name: 'Main navigation' });
  await expect(nav).toBeVisible();

  // Find the source project row.
  const sourceProject = page.locator('[data-sidebar-item-id="p-source"][data-sidebar-item-kind="project"]');
  await expect(sourceProject).toBeVisible();

  // Synthesize a pointer drag using the same event sequence the
  // pointer-reorder hook listens for.
  await page.evaluate(() => {
    const el = document.querySelector(
      '[data-sidebar-item-id="p-source"][data-sidebar-item-kind="project"]',
    ) as HTMLElement | null;
    if (!el) throw new Error('source project not found');
    const r = el.getBoundingClientRect();
    const x = r.left + r.width / 2;
    const y = r.top + r.height / 2;

    el.dispatchEvent(new MouseEvent('mousedown', {
      bubbles: true, cancelable: true, view: window,
      button: 0, buttons: 1, clientX: x, clientY: y,
    }));
    // Move 12px to cross the 8px threshold and start the drag.
    window.dispatchEvent(new MouseEvent('mousemove', {
      bubbles: true, cancelable: true, view: window,
      button: 0, buttons: 1, clientX: x + 12, clientY: y + 12,
    }));
    // A few more moves to simulate real drag motion.
    for (let i = 0; i < 20; i++) {
      window.dispatchEvent(new MouseEvent('mousemove', {
        bubbles: true, cancelable: true, view: window,
        button: 0, buttons: 1, clientX: x + 12 + i, clientY: y + 12 + i,
      }));
    }
  });

  // Give React a tick to flush.
  await page.waitForTimeout(100);

  // Dump diagnostics so failures are debuggable.
  const bodyHtml = await page.evaluate(() => document.body.innerHTML.length);
  const navCount = await page.locator('nav').count();
  // eslint-disable-next-line no-console
  console.log('[diag] body html length:', bodyHtml, 'nav count:', navCount);
  if (errors.length > 0) {
    // eslint-disable-next-line no-console
    console.log('[diag] captured errors:\n', errors.join('\n'));
  }

  // The sidebar landmark must still be ATTACHED to the DOM — if
  // dragging blanked the window, the React tree unmounted and this
  // fails. We use toBeAttached rather than toBeVisible because the
  // drag overlay portal may visually occlude parts of the sidebar
  // mid-drag, making toBeVisible flaky in synthetic-event tests.
  await expect(nav).toBeAttached();
  await expect(sourceProject).toBeAttached();

  // The TDZ failure that this test guards against doesn't always
  // bubble up as a pageerror — it can also surface as a console.error
  // from React's render error path. Either signal is a regression.
  const tdzError = errors.find((e) => e.includes('before initialization'));
  if (tdzError) {
    throw new Error(`TDZ regression — ${tdzError}`);
  }
});
