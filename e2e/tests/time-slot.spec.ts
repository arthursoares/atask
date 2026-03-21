import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Time Slot (Morning/Evening)', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('set time slot to evening', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Evening task' } })).json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });

    const resp = await request.put(`/tasks/${task.ID}/time-slot`, { headers, data: { time_slot: 'evening' } });
    expect(resp.ok()).toBeTruthy();

    const fetched = await (await request.get(`/tasks/${task.ID}`, { headers })).json();
    expect(fetched.TimeSlot).toBe('evening');

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('clear time slot', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Clear slot' } })).json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${task.ID}/time-slot`, { headers, data: { time_slot: 'evening' } });

    await request.put(`/tasks/${task.ID}/time-slot`, { headers, data: { time_slot: null } });

    const fetched = await (await request.get(`/tasks/${task.ID}`, { headers })).json();
    expect(fetched.TimeSlot).toBeNull();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('evening tasks appear after morning in today view', async ({ request }) => {
    const { data: morning } = await (await request.post('/tasks', { headers, data: { title: 'Morning task' } })).json();
    const { data: evening } = await (await request.post('/tasks', { headers, data: { title: 'Evening task' } })).json();

    await request.put(`/tasks/${morning.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${evening.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${evening.ID}/time-slot`, { headers, data: { time_slot: 'evening' } });

    const today = await (await request.get('/views/today', { headers })).json();
    const morningIdx = today.findIndex((t: any) => t.ID === morning.ID);
    const eveningIdx = today.findIndex((t: any) => t.ID === evening.ID);

    expect(morningIdx).toBeGreaterThan(-1);
    expect(eveningIdx).toBeGreaterThan(-1);
    expect(morningIdx).toBeLessThan(eveningIdx);

    await request.delete(`/tasks/${morning.ID}`, { headers });
    await request.delete(`/tasks/${evening.ID}`, { headers });
  });

  test('SSE delivers time_slot_set event', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'SSE slot test' } })).json();

    const controller = new AbortController();
    const events: any[] = [];

    const sseResp = await fetch('http://localhost:8080/events/stream?topics=task.time_slot_set', {
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

    await new Promise(r => setTimeout(r, 500));
    await request.put(`/tasks/${task.ID}/time-slot`, { headers, data: { time_slot: 'evening' } });
    await Promise.race([readPromise, new Promise(r => setTimeout(r, 3000))]);
    controller.abort();

    expect(events.length).toBeGreaterThan(0);
    expect(events[0].type).toBe('task.time_slot_set');

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});
