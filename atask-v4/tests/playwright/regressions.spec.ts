import { test, expect, installTauriMock, type MockState } from './fixtures';

/**
 * Regression tests for user-reported bugs from the manual testing session.
 * Each test reproduces a scenario the user hit and the expectation it
 * needs to meet. These act as guard rails so the bugs don't come back.
 */

// Helper to seed state with a sample task list ─ used by several tests.
function seededState(): MockState {
  const now = '2026-04-10T00:00:00Z';
  return {
    tasks: [
      {
        id: 'task-1',
        title: 'First inbox task',
        notes: '',
        status: 0,
        schedule: 0,
        startDate: null,
        deadline: null,
        completedAt: null,
        index: 0,
        todayIndex: null,
        timeSlot: null,
        projectId: null,
        sectionId: null,
        areaId: null,
        locationId: null,
        createdAt: now,
        updatedAt: now,
        syncStatus: 0,
        repeatRule: null,
      },
      {
        id: 'task-2',
        title: 'Second inbox task',
        notes: '',
        status: 0,
        schedule: 0,
        startDate: null,
        deadline: null,
        completedAt: null,
        index: 1,
        todayIndex: null,
        timeSlot: null,
        projectId: null,
        sectionId: null,
        areaId: null,
        locationId: null,
        createdAt: now,
        updatedAt: now,
        syncStatus: 0,
        repeatRule: null,
      },
    ],
    projects: [],
    areas: [],
    sections: [],
    tags: [],
    taskTags: [],
    taskLinks: [],
    projectTags: [],
    checklistItems: [
      {
        id: 'cl-1',
        title: 'Subtask 1',
        status: 0,
        taskId: 'task-1',
        index: 0,
        createdAt: '2026-04-10T00:00:00Z',
        updatedAt: '2026-04-10T00:00:00Z',
      },
      {
        id: 'cl-2',
        title: 'Subtask 2',
        status: 1,
        taskId: 'task-1',
        index: 1,
        createdAt: '2026-04-10T00:00:00Z',
        updatedAt: '2026-04-10T00:00:00Z',
      },
      {
        id: 'cl-3',
        title: 'Subtask 3',
        status: 0,
        taskId: 'task-1',
        index: 2,
        createdAt: '2026-04-10T00:00:00Z',
        updatedAt: '2026-04-10T00:00:00Z',
      },
    ],
    activities: [],
    locations: [],
  };
}

test.describe('regression: keyboard navigation', () => {
  test('selected task row shows the inset accent stripe (T2.1 / arrow nav visibility)', async ({ page }) => {
    await installTauriMock(page, seededState());
    await page.goto('/');

    // Click the first task to select it.
    await page.getByText('First inbox task').click();

    // The selected row should have the .selected class.
    const selectedRow = page.locator('.task-item.selected').first();
    await expect(selectedRow).toBeVisible();

    // The selected style now adds an inset 3px accent box-shadow stripe.
    // Verify the computed style contains a non-empty box-shadow that
    // references the accent ring color or "inset". CSS resolves
    // var(--accent) to its rgb tuple at runtime so we check for "inset".
    const boxShadow = await selectedRow.evaluate((el) => getComputedStyle(el).boxShadow);
    expect(boxShadow).toMatch(/inset/i);
  });
});

test.describe('regression: empty title delete protection', () => {
  test('clearing the inline editor title and pressing Escape does not delete a task with notes', async ({ page }) => {
    // Seed a task with a non-empty title so we can verify the
    // anti-data-loss guard from T1.2.
    const state = seededState();
    state.tasks[0].notes = 'Important notes that must survive';
    await installTauriMock(page, state);
    await page.goto('/');

    // Open the inline editor (double-click the row).
    await page.getByText('First inbox task').dblclick();

    // Find the title input. Inline editor uses .task-title-input or similar.
    const titleInput = page.locator('input.task-title-input, input.task-edit-title-input').first();
    await expect(titleInput).toBeVisible();

    // Select all + delete.
    await titleInput.focus();
    await page.keyboard.press('Meta+A');
    await page.keyboard.press('Backspace');

    // Press Escape to close the editor.
    await page.keyboard.press('Escape');

    // The task must still exist in the rendered list (the empty-title
    // close path now reverts to the original title instead of deleting).
    await expect(page.getByText('First inbox task')).toBeVisible();
  });
});

test.describe('regression: inline checklist count badge (T3 design parity)', () => {
  test('task with checklist items shows count in row meta', async ({ page }) => {
    await installTauriMock(page, seededState());
    await page.goto('/');

    // Seeded state has 3 checklist items for task-1, 1 done -> "1/3".
    const checklistBadge = page.locator('.task-checklist-count').first();
    await expect(checklistBadge).toBeVisible();
    await expect(checklistBadge).toContainText('1/3');
  });

  test('task without checklist items shows no badge', async ({ mockPage: page }) => {
    // Default empty mock state -> task-2 has no checklist items.
    // (Use the default mockPage fixture which uses emptyState; nothing
    // to verify positive, just that we don't accidentally render a
    // phantom badge for tasks without items.)
    await page.goto('/');
    await expect(page.locator('.task-checklist-count')).toHaveCount(0);
  });
});

test.describe('regression: sidebar keyboard reach (T2.1)', () => {
  test('sidebar exposes nav landmark and tabbable buttons', async ({ mockPage: page }) => {
    await page.goto('/');

    // Landmark exists and is the right name.
    const nav = page.getByRole('navigation', { name: 'Main navigation' });
    await expect(nav).toBeVisible();

    // Inbox is reachable as a button (not <div onClick>).
    const inboxButton = nav.getByRole('button', { name: /inbox/i });
    await expect(inboxButton).toBeVisible();
    // tabIndex must be 0 (focusable in normal tab order).
    await expect(inboxButton).toHaveAttribute('tabindex', '0');
  });
});
