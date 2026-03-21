import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Cancel & Schedule Flows', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('cancel task', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Cancel me' } })).json();
    const resp = await request.post(`/tasks/${task.ID}/cancel`, { headers });
    expect(resp.ok()).toBeTruthy();

    const logbook = await request.get('/views/logbook', { headers });
    const tasks = await logbook.json();
    const cancelled = tasks.find((t: any) => t.ID === task.ID);
    expect(cancelled).toBeTruthy();
    expect(cancelled.Status).toBe(2); // cancelled
  });

  test('schedule flow: inbox → today → someday → inbox', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Schedule flow' } })).json();
    expect(task.Schedule).toBe(0); // inbox

    // → today
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    let today = await (await request.get('/views/today', { headers })).json();
    expect(today.some((t: any) => t.ID === task.ID)).toBeTruthy();

    // → someday
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'someday' } });
    let someday = await (await request.get('/views/someday', { headers })).json();
    expect(someday.some((t: any) => t.ID === task.ID)).toBeTruthy();

    // → back to inbox
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'inbox' } });
    let inbox = await (await request.get('/views/inbox', { headers })).json();
    expect(inbox.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('set today index', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Today idx' } })).json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    const resp = await request.put(`/tasks/${task.ID}/today-index`, { headers, data: { index: 5 } });
    expect(resp.ok()).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});

test.describe('Checklist Full CRUD', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('add, complete, uncomplete, list checklist items', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Checklist full' } })).json();

    // Add items
    const { data: item1 } = await (await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 1' } })).json();
    const { data: item2 } = await (await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 2' } })).json();

    // List
    let items = await (await request.get(`/tasks/${task.ID}/checklist`, { headers })).json();
    expect(items.length).toBe(2);

    // Complete
    await request.post(`/tasks/${task.ID}/checklist/${item1.ID}/complete`, { headers });
    items = await (await request.get(`/tasks/${task.ID}/checklist`, { headers })).json();
    expect(items.find((i: any) => i.ID === item1.ID).Status).toBe(1);

    // Uncomplete
    await request.post(`/tasks/${task.ID}/checklist/${item1.ID}/uncomplete`, { headers });
    items = await (await request.get(`/tasks/${task.ID}/checklist`, { headers })).json();
    expect(items.find((i: any) => i.ID === item1.ID).Status).toBe(0);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});

test.describe('Views Return Data', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('list_someday returns someday tasks', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Someday task' } })).json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'someday' } });

    const resp = await request.get('/views/someday', { headers });
    const tasks = await resp.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('list_upcoming returns tasks with future start date', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Upcoming task' } })).json();
    await request.put(`/tasks/${task.ID}/start-date`, { headers, data: { date: '2026-12-01' } });

    const resp = await request.get('/views/upcoming', { headers });
    const tasks = await resp.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('list_projects returns projects', async ({ request }) => {
    await request.post('/projects', { headers, data: { title: 'List test project' } });
    const resp = await request.get('/projects', { headers });
    const projects = await resp.json();
    expect(projects.length).toBeGreaterThan(0);
  });
});

test.describe('Activity', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('list activity for task (empty)', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Activity task' } })).json();
    const resp = await request.get(`/tasks/${task.ID}/activity`, { headers });
    expect(resp.ok()).toBeTruthy();
    const activity = await resp.json();
    expect(Array.isArray(activity) || activity === null).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});

test.describe('Area Archive', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('archive and unarchive area', async ({ request }) => {
    const { data: area } = await (await request.post('/areas', { headers, data: { title: 'Archive me' } })).json();

    // Archive
    const archResp = await request.post(`/areas/${area.ID}/archive`, { headers });
    expect(archResp.ok()).toBeTruthy();

    // Should not appear in non-archived list
    let areas = await (await request.get('/areas', { headers })).json();
    expect(areas.some((a: any) => a.ID === area.ID)).toBeFalsy();

    // Unarchive
    const unarchResp = await request.post(`/areas/${area.ID}/unarchive`, { headers });
    expect(unarchResp.ok()).toBeTruthy();

    // Should appear again
    areas = await (await request.get('/areas', { headers })).json();
    expect(areas.some((a: any) => a.ID === area.ID)).toBeTruthy();
  });
});
