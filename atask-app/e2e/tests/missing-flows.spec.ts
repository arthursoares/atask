import { test, expect } from '@playwright/test';
import { registerAndLogin, authHeaders } from './helpers';

test.describe('Project & Section Flows', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('create project and assign task', async ({ request }) => {
    const projResp = await request.post('/projects', { headers, data: { title: 'Flow Project' } });
    const { data: project } = await projResp.json();
    expect(project.Title).toBe('Flow Project');

    const taskResp = await request.post('/tasks', { headers, data: { title: 'Assign me' } });
    const { data: task } = await taskResp.json();

    // Assign to project
    await request.put(`/tasks/${task.ID}/project`, { headers, data: { id: project.ID } });

    // Verify
    const tasks = await request.get(`/tasks?project_id=${project.ID}`, { headers });
    const projectTasks = await tasks.json();
    expect(projectTasks.some((t: any) => t.ID === task.ID)).toBeTruthy();

    // Remove from project
    await request.put(`/tasks/${task.ID}/project`, { headers, data: { id: null } });
    const tasks2 = await request.get(`/tasks?project_id=${project.ID}`, { headers });
    const projectTasks2 = await tasks2.json();
    expect((projectTasks2 || []).some((t: any) => t.ID === task.ID)).toBeFalsy();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('create section in project', async ({ request }) => {
    const projResp = await request.post('/projects', { headers, data: { title: 'Section Project' } });
    const { data: project } = await projResp.json();

    const sectResp = await request.post(`/projects/${project.ID}/sections`, { headers, data: { title: 'Phase 1' } });
    expect(sectResp.ok()).toBeTruthy();
    const { data: section } = await sectResp.json();
    expect(section.Title).toBe('Phase 1');

    // List sections
    const sections = await request.get(`/projects/${project.ID}/sections`, { headers });
    const sectionList = await sections.json();
    expect(sectionList.some((s: any) => s.ID === section.ID)).toBeTruthy();
  });

  test('assign task to section', async ({ request }) => {
    const projResp = await request.post('/projects', { headers, data: { title: 'Task Section Project' } });
    const { data: project } = await projResp.json();

    const sectResp = await request.post(`/projects/${project.ID}/sections`, { headers, data: { title: 'Design' } });
    const { data: section } = await sectResp.json();

    const taskResp = await request.post('/tasks', { headers, data: { title: 'Design task' } });
    const { data: task } = await taskResp.json();

    await request.put(`/tasks/${task.ID}/project`, { headers, data: { id: project.ID } });
    await request.put(`/tasks/${task.ID}/section`, { headers, data: { id: section.ID } });

    // Verify task has section
    const getResp = await request.get(`/tasks/${task.ID}`, { headers });
    const fetched = await getResp.json();
    expect(fetched.SectionID).toBe(section.ID);

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('reorder section', async ({ request }) => {
    const projResp = await request.post('/projects', { headers, data: { title: 'Reorder Project' } });
    const { data: project } = await projResp.json();

    await request.post(`/projects/${project.ID}/sections`, { headers, data: { title: 'A' } });
    await request.post(`/projects/${project.ID}/sections`, { headers, data: { title: 'B' } });

    const sections = await request.get(`/projects/${project.ID}/sections`, { headers });
    const list = await sections.json();
    const sectionB = list.find((s: any) => s.Title === 'B');

    const reorderResp = await request.put(
      `/projects/${project.ID}/sections/${sectionB.ID}/reorder`,
      { headers, data: { index: 0 } }
    );
    expect(reorderResp.ok()).toBeTruthy();
  });
});

test.describe('Date Flows', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('set and clear start date', async ({ request }) => {
    const taskResp = await request.post('/tasks', { headers, data: { title: 'Date task' } });
    const { data: task } = await taskResp.json();

    // Set start date
    await request.put(`/tasks/${task.ID}/start-date`, { headers, data: { date: '2026-04-01' } });

    let fetched = await request.get(`/tasks/${task.ID}`, { headers });
    let t = await fetched.json();
    expect(t.StartDate).toContain('2026-04-01');

    // Clear start date
    await request.put(`/tasks/${task.ID}/start-date`, { headers, data: { date: null } });

    fetched = await request.get(`/tasks/${task.ID}`, { headers });
    t = await fetched.json();
    expect(t.StartDate).toBeNull();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });

  test('set and clear deadline', async ({ request }) => {
    const taskResp = await request.post('/tasks', { headers, data: { title: 'Deadline task' } });
    const { data: task } = await taskResp.json();

    await request.put(`/tasks/${task.ID}/deadline`, { headers, data: { date: '2026-04-15' } });

    let fetched = await request.get(`/tasks/${task.ID}`, { headers });
    let t = await fetched.json();
    expect(t.Deadline).toContain('2026-04-15');

    // Clear
    await request.put(`/tasks/${task.ID}/deadline`, { headers, data: { date: null } });

    fetched = await request.get(`/tasks/${task.ID}`, { headers });
    t = await fetched.json();
    expect(t.Deadline).toBeNull();

    await request.delete(`/tasks/${task.ID}`, { headers });
  });
});

test.describe('Area CRUD', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('create and list areas', async ({ request }) => {
    const resp = await request.post('/areas', { headers, data: { title: 'Work' } });
    expect(resp.ok()).toBeTruthy();
    const { data: area } = await resp.json();
    expect(area.Title).toBe('Work');

    const list = await request.get('/areas', { headers });
    const areas = await list.json();
    expect(areas.some((a: any) => a.ID === area.ID)).toBeTruthy();
  });

  test('rename area', async ({ request }) => {
    const resp = await request.post('/areas', { headers, data: { title: 'Old Name' } });
    const { data: area } = await resp.json();

    await request.put(`/areas/${area.ID}`, { headers, data: { title: 'New Name' } });

    const list = await request.get('/areas', { headers });
    const areas = await list.json();
    const renamed = areas.find((a: any) => a.ID === area.ID);
    expect(renamed.Title).toBe('New Name');
  });
});

test.describe('Project Color', () => {
  let token: string;
  let headers: ReturnType<typeof authHeaders>;

  test.beforeAll(async ({ request }) => {
    token = await registerAndLogin(request);
    headers = authHeaders(token);
  });

  test('set and get project color', async ({ request }) => {
    const resp = await request.post('/projects', { headers, data: { title: 'Colored' } });
    const { data: project } = await resp.json();

    await request.put(`/projects/${project.ID}/color`, { headers, data: { color: '#e74c3c' } });

    const fetched = await request.get(`/projects/${project.ID}`, { headers });
    const p = await fetched.json();
    expect(p.Color).toBe('#e74c3c');
  });
});
