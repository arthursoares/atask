import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Task Lifecycle', () => {
  let token: string;
  let headers: Record<string, string>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('create task lands in inbox', async ({ request }) => {
    const resp = await request.post('/tasks', {
      headers,
      data: { title: 'E2E test task' },
    });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.data.Title).toBe('E2E test task');
    expect(body.data.Schedule).toBe(0); // inbox

    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    expect(tasks.some((t: any) => t.Title === 'E2E test task')).toBeTruthy();
  });

  test('complete task moves to logbook', async ({ request }) => {
    // Create
    const createResp = await request.post('/tasks', {
      headers,
      data: { title: 'Complete me' },
    });
    const { data: task } = await createResp.json();

    // Complete
    const completeResp = await request.post(`/tasks/${task.ID}/complete`, { headers });
    expect(completeResp.ok()).toBeTruthy();

    // Should be in logbook
    const logbook = await request.get('/views/logbook', { headers });
    const tasks = await logbook.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    // Should NOT be in inbox
    const inbox = await request.get('/views/inbox', { headers });
    const inboxTasks = await inbox.json();
    expect(inboxTasks.some((t: any) => t.ID === task.ID)).toBeFalsy();
  });

  test('schedule task for today', async ({ request }) => {
    const createResp = await request.post('/tasks', {
      headers,
      data: { title: 'Today task' },
    });
    const { data: task } = await createResp.json();

    // Schedule for today (anytime)
    await request.put(`/tasks/${task.ID}/schedule`, {
      headers,
      data: { schedule: 'anytime' },
    });

    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();
  });

  test('move task to project', async ({ request }) => {
    // Create project
    const projResp = await request.post('/projects', {
      headers,
      data: { title: 'E2E Project' },
    });
    const { data: project } = await projResp.json();

    // Create task
    const taskResp = await request.post('/tasks', {
      headers,
      data: { title: 'Project task' },
    });
    const { data: task } = await taskResp.json();

    // Move to project
    await request.put(`/tasks/${task.ID}/project`, {
      headers,
      data: { id: project.ID },
    });

    // Verify
    const tasks = await request.get(`/tasks?project_id=${project.ID}`, { headers });
    const projectTasks = await tasks.json();
    expect(projectTasks.some((t: any) => t.ID === task.ID)).toBeTruthy();
  });

  test('update task title and notes', async ({ request }) => {
    const createResp = await request.post('/tasks', {
      headers,
      data: { title: 'Original title' },
    });
    const { data: task } = await createResp.json();

    await request.put(`/tasks/${task.ID}/title`, {
      headers,
      data: { title: 'Updated title' },
    });

    await request.put(`/tasks/${task.ID}/notes`, {
      headers,
      data: { notes: 'Some important notes' },
    });

    // Verify via inbox (task should have updated fields)
    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    const updated = tasks.find((t: any) => t.ID === task.ID);
    expect(updated.Title).toBe('Updated title');
    expect(updated.Notes).toBe('Some important notes');
  });

  test('checklist items', async ({ request }) => {
    const createResp = await request.post('/tasks', {
      headers,
      data: { title: 'Checklist task' },
    });
    const { data: task } = await createResp.json();

    // Add item
    const itemResp = await request.post(`/tasks/${task.ID}/checklist`, {
      headers,
      data: { title: 'Step 1' },
    });
    const { data: item } = await itemResp.json();
    expect(item.Title).toBe('Step 1');

    // Complete item
    await request.post(`/tasks/${task.ID}/checklist/${item.ID}/complete`, { headers });

    // Verify
    const items = await request.get(`/tasks/${task.ID}/checklist`, { headers });
    const checklist = await items.json();
    const completed = checklist.find((i: any) => i.ID === item.ID);
    expect(completed.Status).toBe(1); // completed
  });

  test('set dates', async ({ request }) => {
    const createResp = await request.post('/tasks', {
      headers,
      data: { title: 'Dated task' },
    });
    const { data: task } = await createResp.json();

    await request.put(`/tasks/${task.ID}/start-date`, {
      headers,
      data: { date: '2026-04-01' },
    });

    await request.put(`/tasks/${task.ID}/deadline`, {
      headers,
      data: { date: '2026-04-15' },
    });

    // Should appear in upcoming
    const upcoming = await request.get('/views/upcoming', { headers });
    const tasks = await upcoming.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();
  });
});
