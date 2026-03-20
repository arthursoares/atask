import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Today View Workflows', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('new task lands in inbox by default', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Inbox default' } });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    const task = body.data;
    expect(task.Schedule).toBe(0); // inbox

    const inbox = await request.get('/views/inbox', { headers });
    const tasks = await inbox.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('schedule task for today (anytime)', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Today task' } });
    const { data: task } = await resp.json();

    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    expect(tasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('complete task removes from today, adds to logbook', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Complete me' } });
    const { data: task } = await resp.json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    await request.post(`/tasks/${task.ID}/complete`, { headers });

    const today = await request.get('/views/today', { headers });
    const todayTasks = await today.json();
    expect(todayTasks.some((t: any) => t.ID === task.ID)).toBeFalsy();

    const logbook = await request.get('/views/logbook', { headers });
    const logbookTasks = await logbook.json();
    expect(logbookTasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('reschedule completed task back to inbox', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Reschedule me' } });
    const { data: task } = await resp.json();

    await request.post(`/tasks/${task.ID}/complete`, { headers });

    // Move back to inbox by changing schedule (no dedicated reopen endpoint)
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'inbox' } });

    // After rescheduling to inbox, task should appear in inbox view
    // (the schedule change sets schedule=inbox; it remains completed unless server resets status)
    const get = await request.get(`/tasks/${task.ID}`, { headers });
    const fetched = await get.json();
    expect(fetched.Schedule).toBe(0); // inbox

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('reorder tasks in today view', async ({ request }) => {
    const r1 = await request.post('/tasks', { headers, data: { title: 'First' } });
    const { data: t1 } = await r1.json();
    const r2 = await request.post('/tasks', { headers, data: { title: 'Second' } });
    const { data: t2 } = await r2.json();

    await request.put(`/tasks/${t1.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${t2.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${t1.ID}/reorder`, { headers, data: { index: 1 } });
    await request.put(`/tasks/${t2.ID}/reorder`, { headers, data: { index: 0 } });

    const today = await request.get('/views/today', { headers });
    const tasks = await today.json();
    const idx1 = tasks.findIndex((t: any) => t.ID === t1.ID);
    const idx2 = tasks.findIndex((t: any) => t.ID === t2.ID);
    // t2 has index 0, t1 has index 1 — t2 should come first
    expect(idx2).toBeLessThan(idx1);

    await request.delete(`/tasks/${t1.ID}`, { headers });
    await request.delete(`/tasks/${t2.ID}`, { headers });
  });

  test('update task title and notes', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Original' } });
    const { data: task } = await resp.json();

    await request.put(`/tasks/${task.ID}/title`, { headers, data: { title: 'Updated' } });
    await request.put(`/tasks/${task.ID}/notes`, { headers, data: { notes: 'Some notes' } });

    const fetched = await request.get(`/tasks/${task.ID}`, { headers });
    const updated = await fetched.json();
    expect(updated.Title).toBe('Updated');
    expect(updated.Notes).toBe('Some notes');

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('move task to project', async ({ request }) => {
    const projResp = await request.post('/projects', { headers, data: { title: 'Test Project' } });
    const { data: project } = await projResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'Project task' } });
    const { data: task } = await taskResp.json();

    await request.put(`/tasks/${task.ID}/project`, { headers, data: { id: project.ID } });

    const tasks = await request.get(`/tasks?project_id=${project.ID}`, { headers });
    const projectTasks = await tasks.json();
    expect(projectTasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('checklist CRUD', async ({ request }) => {
    const taskResp = await request.post('/tasks', { headers, data: { title: 'Checklist task' } });
    const { data: task } = await taskResp.json();

    const itemResp = await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 1' } });
    const itemBody = await itemResp.json();
    const item = itemBody.data;
    expect(item.Title).toBe('Step 1');
    expect(item.Status).toBe(0);

    await request.post(`/tasks/${task.ID}/checklist/${item.ID}/complete`, { headers });

    const items = await request.get(`/tasks/${task.ID}/checklist`, { headers });
    const checklist = await items.json();
    const completed = checklist.find((i: any) => i.ID === item.ID);
    expect(completed.Status).toBe(1);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('set start date and deadline', async ({ request }) => {
    const resp = await request.post('/tasks', { headers, data: { title: 'Dated task' } });
    const { data: task } = await resp.json();

    await request.put(`/tasks/${task.ID}/start-date`, { headers, data: { date: '2026-04-01' } });
    await request.put(`/tasks/${task.ID}/deadline`, { headers, data: { date: '2026-04-15' } });

    const fetched = await request.get(`/tasks/${task.ID}`, { headers });
    const updated = await fetched.json();
    expect(updated.StartDate).toBeTruthy();
    expect(updated.Deadline).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('project CRUD', async ({ request }) => {
    const resp = await request.post('/projects', { headers, data: { title: 'My Project' } });
    expect(resp.status()).toBe(201);
    const { data: project } = await resp.json();
    expect(project.Title).toBe('My Project');

    await request.put(`/projects/${project.ID}/title`, { headers, data: { title: 'Renamed' } });
    await request.put(`/projects/${project.ID}/notes`, { headers, data: { notes: 'Project notes' } });

    const fetched = await request.get(`/projects/${project.ID}`, { headers });
    const p = await fetched.json();
    expect(p.Title).toBe('Renamed');
    expect(p.Notes).toBe('Project notes');
  });

  test('SSE delivers task.created event', async ({ request }) => {
    const controller = new AbortController();
    const events: any[] = [];

    const sseResp = await fetch('http://localhost:8080/events/stream?topics=task.created', {
      headers: { Authorization: `Bearer ${token}` },
      signal: controller.signal,
    });

    const reader = sseResp.body!.getReader();
    const decoder = new TextDecoder();
    const readPromise = (async () => {
      while (events.length < 1) {
        const { value, done } = await reader.read();
        if (done) break;
        const text = decoder.decode(value);
        for (const line of text.split('\n')) {
          if (line.startsWith('event:')) events.push({ type: line.replace('event: ', '').trim() });
          if (line.startsWith('data:') && events.length > 0) {
            events[events.length - 1].data = JSON.parse(line.replace('data: ', '').trim());
          }
        }
      }
    })();

    await new Promise((r) => setTimeout(r, 500));
    await request.post('/tasks', { headers, data: { title: 'SSE test' } });
    await Promise.race([readPromise, new Promise((r) => setTimeout(r, 3000))]);
    controller.abort();

    expect(events.length).toBeGreaterThan(0);
    expect(events[0].type).toBe('task.created');
    expect(events[0].data.entity_type).toBe('task');
  });
});
