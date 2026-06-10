import { test, expect } from '@playwright/test';
import { installTauriMock, type MockState } from './fixtures';

/**
 * Drag-state regression suite: lift clone identity, multi-drag count
 * badge, sidebar drop-target highlight, and edge autoscroll.
 */

const OUT = '/tmp/atask-audit/drag';
const now = new Date().toISOString();

function task(partial: Record<string, unknown>): Record<string, unknown> {
  return {
    id: '', title: '', notes: '', status: 0, schedule: 0,
    startDate: null, deadline: null, completedAt: null,
    index: 0, todayIndex: null, timeSlot: null,
    projectId: null, sectionId: null, areaId: null, locationId: null,
    createdAt: now, updatedAt: now, syncStatus: 0, repeatRule: null,
    ...partial,
  };
}

function buildState(taskCount: number): MockState {
  return {
    areas: [{ id: 'a-work', title: 'Work', index: 0, archived: false, createdAt: now, updatedAt: now }],
    projects: [{ id: 'p-web', title: 'Website Redesign', notes: '', status: 0, color: 'blue', areaId: 'a-work', index: 0, completedAt: null, createdAt: now, updatedAt: now }],
    sections: [], tags: [], taskTags: [], taskLinks: [], projectTags: [],
    checklistItems: [], activities: [], locations: [],
    tasks: Array.from({ length: taskCount }, (_, i) =>
      task({ id: `t-${i + 1}`, title: `Task number ${i + 1}`, index: i }),
    ),
  };
}

async function centerOf(page: import('@playwright/test').Page, text: string) {
  const box = await page.getByText(text, { exact: true }).boundingBox();
  if (!box) throw new Error(`no box for ${text}`);
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

test('single drag: lift clone, ghost row, drop gap', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await installTauriMock(page, buildState(6));
  await page.goto('/');
  await page.getByRole('navigation', { name: 'Main navigation' }).waitFor();
  await page.waitForTimeout(600);

  const from = await centerOf(page, 'Task number 2');
  const to = await centerOf(page, 'Task number 5');

  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(from.x + 4, from.y + 12, { steps: 3 });
  await page.mouse.move(to.x, to.y + 6, { steps: 12 });
  await page.waitForTimeout(300);
  await page.screenshot({ path: `${OUT}/d1-mid-drag.png` });

  // Drop gap exists and the floating clone shows the row identity
  await expect(page.locator('.task-drop-zone')).toHaveCount(1);
  await expect(page.locator('.drag-clone .drag-clone-title')).toHaveText('Task number 2');

  await page.mouse.up();
  await page.waitForTimeout(300);
  await page.screenshot({ path: `${OUT}/d2-after-drop.png` });

  // Order actually changed: task 2 now sits after task 4
  const titles = await page.locator('.task-title').allTextContents();
  expect(titles.indexOf('Task number 2')).toBeGreaterThan(titles.indexOf('Task number 4'));
});

test('multi-select drag shows stacked clone with count', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await installTauriMock(page, buildState(6));
  await page.goto('/');
  await page.getByRole('navigation', { name: 'Main navigation' }).waitFor();
  await page.waitForTimeout(600);

  await page.getByText('Task number 1', { exact: true }).click();
  await page.keyboard.press('Escape'); // close detail panel, keep going
  await page.getByText('Task number 1', { exact: true }).click();
  await page.getByText('Task number 3', { exact: true }).click({ modifiers: ['Shift'] });

  const from = await centerOf(page, 'Task number 2');
  const to = await centerOf(page, 'Task number 6');
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(from.x + 4, from.y + 12, { steps: 3 });
  await page.mouse.move(to.x, to.y + 6, { steps: 12 });
  await page.waitForTimeout(300);
  await page.screenshot({ path: `${OUT}/d3-multi-drag.png` });

  await expect(page.locator('.drag-clone-count')).toHaveText('3');
  await page.mouse.up();
});

test('drag over sidebar highlights the target', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await installTauriMock(page, buildState(4));
  await page.goto('/');
  await page.getByRole('navigation', { name: 'Main navigation' }).waitFor();
  await page.waitForTimeout(600);

  const from = await centerOf(page, 'Task number 1');
  const nav = page.getByRole('navigation', { name: 'Main navigation' });
  const todayBox = await nav.getByRole('button', { name: /today/i }).boundingBox();
  if (!todayBox) throw new Error('no today box');

  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(from.x, from.y + 14, { steps: 3 });
  await page.mouse.move(todayBox.x + todayBox.width / 2, todayBox.y + todayBox.height / 2, { steps: 14 });
  await page.waitForTimeout(300);
  await page.screenshot({ path: `${OUT}/d4-sidebar-target.png` });

  await expect(page.locator('.sidebar-item.drag-target')).toHaveCount(1);
  await page.mouse.up();
  await page.waitForTimeout(300);

  // Dropping on Today rescheduled it: Today badge appears
  await page.screenshot({ path: `${OUT}/d5-after-sidebar-drop.png` });
});

test('edge autoscroll while dragging in a long list', async ({ page }) => {
  await page.setViewportSize({ width: 1100, height: 520 });
  await installTauriMock(page, buildState(40));
  await page.goto('/');
  await page.getByRole('navigation', { name: 'Main navigation' }).waitFor();
  await page.waitForTimeout(600);

  const scrollTopBefore = await page.evaluate(() => {
    const el = document.querySelector('.app-content');
    return el ? el.scrollTop : -1;
  });

  const from = await centerOf(page, 'Task number 2');
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(from.x + 4, from.y + 12, { steps: 3 });
  // Park the cursor near the bottom edge of the window and let autoscroll run
  await page.mouse.move(from.x, 500, { steps: 10 });
  await page.waitForTimeout(1200);
  await page.screenshot({ path: `${OUT}/d6-autoscroll.png` });

  const scrollTopAfter = await page.evaluate(() => {
    const el = document.querySelector('.app-content');
    return el ? el.scrollTop : -1;
  });
  await page.mouse.up();

  expect(scrollTopBefore).toBe(0);
  expect(scrollTopAfter).toBeGreaterThan(50);
});
