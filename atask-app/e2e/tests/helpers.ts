export async function registerAndLogin(request: any): Promise<string> {
  const email = `test-${Date.now()}@test.com`;
  await request.post('/auth/register', {
    data: { email, password: 'testpass', name: 'E2E Test' },
  });
  const loginResp = await request.post('/auth/login', {
    data: { email, password: 'testpass' },
  });
  const { token } = await loginResp.json();
  return token;
}

export function authHeaders(token: string) {
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
}
