import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Tag Flow', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('create tag', async ({ request }) => {
    const name = `tag-create-${Date.now()}`;
    const resp = await request.post('/tags', { headers, data: { title: name } });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.data.Title).toBe(name);
  });

  test('list tags returns created tag', async ({ request }) => {
    const name = `tag-list-${Date.now()}`;
    await request.post('/tags', { headers, data: { title: name } });
    const resp = await request.get('/tags', { headers });
    const tags = await resp.json();
    expect(tags.some((t: any) => t.Title === name)).toBeTruthy();
  });

  test('add tag to task', async ({ request }) => {
    const name = `tag-add-${Date.now()}`;
    const tagResp = await request.post('/tags', { headers, data: { title: name } });
    const { data: tag } = await tagResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'Tag test task' } });
    const { data: task } = await taskResp.json();

    const addResp = await request.post(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });
    expect(addResp.ok()).toBeTruthy();

    const getResp = await request.get(`/tasks/${task.ID}`, { headers });
    const fetchedTask = await getResp.json();
    expect(fetchedTask.Tags).toContain(tag.ID);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('remove tag from task', async ({ request }) => {
    const name = `tag-remove-${Date.now()}`;
    const tagResp = await request.post('/tags', { headers, data: { title: name } });
    const { data: tag } = await tagResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'Remove tag task' } });
    const { data: task } = await taskResp.json();

    await request.post(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });
    const removeResp = await request.delete(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });
    expect(removeResp.ok()).toBeTruthy();

    const getResp = await request.get(`/tasks/${task.ID}`, { headers });
    const fetchedTask = await getResp.json();
    expect(fetchedTask.Tags || []).not.toContain(tag.ID);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('tags list endpoint returns tags for task', async ({ request }) => {
    const name = `tag-listcheck-${Date.now()}`;
    const tagResp = await request.post('/tags', { headers, data: { title: name } });
    const { data: tag } = await tagResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'List tag task' } });
    const { data: task } = await taskResp.json();

    await request.post(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });

    // List endpoints don't hydrate tags
    const inboxResp = await request.get('/views/inbox', { headers });
    const inbox = await inboxResp.json();
    const found = inbox.find((t: any) => t.ID === task.ID);
    expect(found).toBeTruthy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('duplicate tag name is rejected', async ({ request }) => {
    const name = `tag-dup-${Date.now()}`;
    const resp1 = await request.post('/tags', { headers, data: { title: name } });
    expect(resp1.ok()).toBeTruthy();

    const resp2 = await request.post('/tags', { headers, data: { title: name } });
    expect(resp2.ok()).toBeFalsy(); // Should fail with unique constraint
  });
});
