import type { APIRequestContext } from '@playwright/test';

export async function registerAndLogin(request: APIRequestContext): Promise<string> {
  const email = `test-${Date.now()}-${Math.random().toString(36).slice(2)}@test.com`;
  await request.post('/auth/register', {
    data: { email, password: 'testpass', name: 'E2E Test' },
  });
  const resp = await request.post('/auth/login', {
    data: { email, password: 'testpass' },
  });
  const { token } = await resp.json();
  return token;
}

export function authHeaders(token: string) {
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
}
