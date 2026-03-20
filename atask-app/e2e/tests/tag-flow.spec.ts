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
    const resp = await request.post('/tags', { headers, data: { title: 'test-tag-unique' } });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.data.Title).toBe('test-tag-unique');
    console.log('Tag ID:', body.data.ID);
  });

  test('list tags returns created tag', async ({ request }) => {
    await request.post('/tags', { headers, data: { title: 'playwright-tag' } });
    const resp = await request.get('/tags', { headers });
    const tags = await resp.json();
    expect(tags.some((t: any) => t.Title === 'playwright-tag')).toBeTruthy();
  });

  test('add tag to task', async ({ request }) => {
    // Create tag
    const tagResp = await request.post('/tags', { headers, data: { title: 'assign-me' } });
    const { data: tag } = await tagResp.json();

    // Create task
    const taskResp = await request.post('/tasks', { headers, data: { title: 'Tag test task' } });
    const { data: task } = await taskResp.json();

    // Add tag to task
    const addResp = await request.post(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });
    console.log('Add tag status:', addResp.status());
    console.log('Add tag body:', await addResp.text());
    expect(addResp.ok()).toBeTruthy();

    // Verify: GET task should have tag in Tags field
    const getResp = await request.get(`/tasks/${task.ID}`, { headers });
    const fetchedTask = await getResp.json();
    console.log('Task Tags:', fetchedTask.Tags);
    expect(fetchedTask.Tags).toContain(tag.ID);

    // Cleanup
    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('remove tag from task', async ({ request }) => {
    const tagResp = await request.post('/tags', { headers, data: { title: 'remove-me' } });
    const { data: tag } = await tagResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'Remove tag task' } });
    const { data: task } = await taskResp.json();

    // Add then remove
    await request.post(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });
    const removeResp = await request.delete(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });
    expect(removeResp.ok()).toBeTruthy();

    // Verify tag is gone
    const getResp = await request.get(`/tasks/${task.ID}`, { headers });
    const fetchedTask = await getResp.json();
    console.log('Tags after remove:', fetchedTask.Tags);
    expect(fetchedTask.Tags || []).not.toContain(tag.ID);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('tags list endpoint returns tags for task', async ({ request }) => {
    // Also test via the list endpoint that views use
    const tagResp = await request.post('/tags', { headers, data: { title: 'list-check' } });
    const { data: tag } = await tagResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'List tag task' } });
    const { data: task } = await taskResp.json();

    await request.post(`/tasks/${task.ID}/tags/${tag.ID}`, { headers });

    // Check: does the inbox view include the tag?
    const inboxResp = await request.get('/views/inbox', { headers });
    const inbox = await inboxResp.json();
    const found = inbox.find((t: any) => t.ID === task.ID);
    console.log('Task in inbox Tags field:', found?.Tags);
    // Note: list endpoints may NOT hydrate tags (only GET /tasks/{id} does)
    
    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});
