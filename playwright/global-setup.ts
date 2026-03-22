import { chromium, type FullConfig } from '@playwright/test';

const BASE_URL = process.env.BASE_URL ?? 'http://localhost:8322';

export default async function globalSetup(_config: FullConfig) {
  const browser = await chromium.launch();
  const page = await browser.newPage();

  // Login via API to get token, then inject into localStorage
  const res = await page.request.post(`${BASE_URL}/api/auth/login`, {
    data: { email: 'admin@agach.local', password: 'admin' },
    headers: { 'Content-Type': 'application/json' },
  });
  const json = await res.json();
  const token: string = json.data.access_token;
  const user = json.data.user;

  // Navigate to the app and inject the token into localStorage
  await page.goto(BASE_URL);
  await page.evaluate(
    ({ token, user }) => {
      localStorage.setItem('agach_access_token', token);
      localStorage.setItem('agach_user', JSON.stringify(user));
    },
    { token, user },
  );

  // Save storage state so all tests reuse it
  await page.context().storageState({ path: '/tmp/auth-state.json' });
  await browser.close();
}
