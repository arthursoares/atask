import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Checklist Counts', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('GET /tasks/{id} includes checklist counts', async ({ request }) => {
    // Create task
    const createResp = await request.post('/tasks', { headers, data: { title: 'Count test' } });
    const { data: task } = await createResp.json();

    // Initially zero
    let fetched = await (await request.get(`/tasks/${task.ID}`, { headers })).json();
    expect(fetched.ChecklistTotal).toBe(0);
    expect(fetched.ChecklistDone).toBe(0);

    // Add 3 items
    await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 1' } });
    await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 2' } });
    const { data: item3 } = await (await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 3' } })).json();

    // Complete 1
    await request.post(`/tasks/${task.ID}/checklist/${item3.ID}/complete`, { headers });

    // Verify counts
    fetched = await (await request.get(`/tasks/${task.ID}`, { headers })).json();
    expect(fetched.ChecklistTotal).toBe(3);
    expect(fetched.ChecklistDone).toBe(1);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('list endpoints do NOT include checklist counts (default 0)', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'List count test' } })).json();
    await request.post(`/tasks/${task.ID}/checklist`, { headers, data: { title: 'Step 1' } });

    // Inbox list should have default 0 (not hydrated)
    const inbox = await (await request.get('/views/inbox', { headers })).json();
    const found = inbox.find((t: any) => t.ID === task.ID);
    expect(found).toBeTruthy();
    // Counts should be 0 (not hydrated in list)
    expect(found.ChecklistTotal || 0).toBe(0);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});
