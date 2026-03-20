import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('SSE Events', () => {
  test('receives task.created event via SSE', async ({ request }) => {
    const token = await registerAndLogin(request);
    const headers = authHeaders(token);

    // Connect to SSE
    const events: any[] = [];
    const controller = new AbortController();

    const ssePromise = fetch('http://localhost:8080/events/stream?topics=task.*', {
      headers: { Authorization: `Bearer ${token}` },
      signal: controller.signal,
    })
      .then(async (resp) => {
        const reader = resp.body!.getReader();
        const decoder = new TextDecoder();
        while (events.length < 1) {
          const { value, done } = await reader.read();
          if (done) break;
          const text = decoder.decode(value);
          for (const line of text.split('\n')) {
            if (line.startsWith('event:')) {
              events.push({ type: line.replace('event: ', '').trim() });
            }
            if (line.startsWith('data:') && events.length > 0) {
              events[events.length - 1].data = JSON.parse(
                line.replace('data: ', '').trim()
              );
            }
          }
        }
      })
      .catch(() => {});

    // Wait for SSE to connect
    await new Promise((r) => setTimeout(r, 500));

    // Create a task
    await request.post('/tasks', {
      headers,
      data: { title: 'SSE test task' },
    });

    // Wait for event
    await Promise.race([ssePromise, new Promise((r) => setTimeout(r, 3000))]);
    controller.abort();

    expect(events.length).toBeGreaterThan(0);
    expect(events[0].type).toBe('task.created');
    expect(events[0].data.entity_type).toBe('task');
  });
});
