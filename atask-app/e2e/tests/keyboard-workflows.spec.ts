import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Keyboard Shortcut Workflows', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('⌘N workflow: create task lands in inbox', async ({ request }) => {
    // ⌘N creates a task — verify it appears in inbox
    const resp = await request.post('/tasks', { headers, data: { title: 'New task' } });
    expect(resp.ok()).toBeTruthy();
    const { data: task } = await resp.json();
    expect(task.Schedule).toBe(0); // inbox

    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('⌘⇧C workflow: complete task', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Complete shortcut' } })).json();

    await request.post(`/tasks/${task.ID}/complete`, { headers });

    const logbook = await request.get('/views/logbook', { headers });
    const tasks = await logbook.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('⌘T workflow: schedule for today', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Today shortcut' } })).json();

    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('Backspace workflow: delete task', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Delete shortcut' } })).json();

    await request.delete(`/tasks/${task.ID}`, { headers });

    // Should not appear in inbox
    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    expect((tasks || []).some((t: any) => t.ID === task.ID)).toBeFalsy();
  });

  test('Space workflow: toggle completion', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Space complete' } })).json();

    // Complete
    await request.post(`/tasks/${task.ID}/complete`, { headers });

    const logbook = await request.get('/views/logbook', { headers });
    const tasks = await logbook.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    // Reopen
    await request.post(`/tasks/${task.ID}/reopen`, { headers });

    const inbox = await request.get('/views/inbox', { headers });
    const inboxTasks = await inbox.json();
    expect(inboxTasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('view navigation: all 5 views return data', async ({ request }) => {
    // Simulates ⌘1-5 by verifying each view endpoint works
    const views = ['/views/inbox', '/views/today', '/views/upcoming', '/views/someday', '/views/logbook'];
    for (const view of views) {
      const resp = await request.get(view, { headers });
      expect(resp.ok()).toBeTruthy();
      const data = await resp.json();
      expect(Array.isArray(data) || data === null).toBeTruthy();
    }
  });

  test('defer to someday workflow', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Someday shortcut' } })).json();

    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'someday' } });

    const someday = await request.get('/views/someday', { headers });
    const tasks = await someday.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('move to inbox workflow', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Inbox shortcut' } })).json();

    // Schedule as anytime first
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    // Then move back to inbox
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'inbox' } });

    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});
