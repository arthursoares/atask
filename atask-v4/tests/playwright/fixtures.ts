import { test as base, type Page } from '@playwright/test';

/**
 * Playwright fixtures for atask-v4 frontend tests.
 *
 * The atask v4 React frontend calls Rust IPC via `@tauri-apps/api/core`'s
 * `invoke`, which throws in a browser context because
 * `window.__TAURI_INTERNALS__` isn't present. To run UI tests without a
 * real Tauri binary we stub invoke with a minimal in-memory state store
 * so the React shell loads, selectors run, and mutations update state
 * just like they would in dev mode.
 *
 * The mock intentionally keeps the state narrow — enough for the sidebar,
 * views, and task rows to render. Rich scenarios (activities feed, sync
 * status, etc.) can extend the handler set in individual tests via
 * `page.evaluate` to reach into the mock.
 */

export interface MockState {
  tasks: Array<Record<string, unknown>>;
  projects: Array<Record<string, unknown>>;
  areas: Array<Record<string, unknown>>;
  sections: Array<Record<string, unknown>>;
  tags: Array<Record<string, unknown>>;
  taskTags: Array<Record<string, unknown>>;
  taskLinks: Array<Record<string, unknown>>;
  projectTags: Array<Record<string, unknown>>;
  checklistItems: Array<Record<string, unknown>>;
  activities: Array<Record<string, unknown>>;
  locations: Array<Record<string, unknown>>;
}

export const emptyState: MockState = {
  tasks: [],
  projects: [],
  areas: [],
  sections: [],
  tags: [],
  taskTags: [],
  taskLinks: [],
  projectTags: [],
  checklistItems: [],
  activities: [],
  locations: [],
};

/**
 * Install a Tauri IPC mock that short-circuits the `invoke` channel. Any
 * command name not explicitly handled returns `null`, which matches the
 * behavior of most mutation commands (Rust side returns `()`).
 */
export async function installTauriMock(page: Page, state: MockState = emptyState): Promise<void> {
  await page.addInitScript((initial: MockState) => {
    // Set a flag the real app can read to detect the mock environment
    // if it ever needs to branch on "running in Playwright".
    (window as unknown as { __ATASK_PLAYWRIGHT__: boolean }).__ATASK_PLAYWRIGHT__ = true;

    const state = JSON.parse(JSON.stringify(initial));

    const handlers: Record<string, (args?: Record<string, unknown>) => unknown> = {
      load_all: () => state,
      get_settings: () => ({ serverUrl: '', apiKey: '', syncEnabled: false }),
      get_sync_status: () => ({
        isSyncing: false,
        lastSyncAt: null,
        lastError: null,
        pendingOpsCount: 0,
      }),
      // Default no-ops for common mutations — individual tests can
      // override by re-adding specific handlers via page.evaluate.
      create_task: (args) => {
        const params = (args?.params ?? {}) as Record<string, unknown>;
        const id = `mock-task-${state.tasks.length + 1}`;
        const now = new Date().toISOString();
        const task = {
          id,
          title: params.title ?? '',
          notes: '',
          status: 0,
          schedule: 0,
          startDate: null,
          deadline: null,
          completedAt: null,
          index: state.tasks.length,
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
        };
        state.tasks.push(task);
        return task;
      },
      update_task: (args) => {
        const params = (args?.params ?? {}) as Record<string, unknown> & { id?: string };
        const task = state.tasks.find((t: Record<string, unknown>) => t.id === params.id);
        if (task) Object.assign(task, params);
        return task ?? null;
      },
      delete_task: (args) => {
        const id = (args as { id?: string } | undefined)?.id;
        const idx = state.tasks.findIndex((t: Record<string, unknown>) => t.id === id);
        if (idx >= 0) state.tasks.splice(idx, 1);
        return null;
      },
    };

    // Expose the mock state for test inspection + mutation from outside.
    (window as unknown as { __ATASK_MOCK_STATE__: MockState }).__ATASK_MOCK_STATE__ = state;
    (window as unknown as { __ATASK_MOCK_HANDLERS__: typeof handlers }).__ATASK_MOCK_HANDLERS__ = handlers;

    // Tauri 2 IPC hook: invoke() from @tauri-apps/api/core looks for
    // window.__TAURI_INTERNALS__.invoke and calls it with (cmd, args).
    (window as unknown as { __TAURI_INTERNALS__: unknown }).__TAURI_INTERNALS__ = {
      invoke: (cmd: string, args?: Record<string, unknown>) => {
        const handler = handlers[cmd];
        if (handler) {
          try {
            return Promise.resolve(handler(args));
          } catch (err) {
            return Promise.reject(err);
          }
        }
        // Unknown command → resolve null so mutations that return () don't
        // blow up. Tests that rely on specific commands should register
        // handlers explicitly.
        return Promise.resolve(null);
      },
      transformCallback: <T>(cb: T) => cb,
    };
  }, state);
}

interface Fixtures {
  mockPage: Page;
}

/**
 * Drop-in replacement for Playwright's `test` that auto-installs the
 * Tauri IPC mock on the page before any navigation. Use `mockPage`
 * instead of `page` in tests that don't need a custom mock state.
 */
export const test = base.extend<Fixtures>({
  mockPage: async ({ page }, use) => {
    await installTauriMock(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';
