import { type Page, type APIRequestContext, expect } from '@playwright/test';

// ─── API helpers ────────────────────────────────────────────────────────────

export interface LoginResponse {
  user: { id: string; email: string; display_name: string; role: string };
  access_token: string;
}

/**
 * Login via API and return the access token.
 */
export async function apiLogin(
  request: APIRequestContext,
  email = 'admin@agach.local',
  password = 'admin',
): Promise<string> {
  const resp = await request.post('/api/auth/login', {
    data: { email, password },
  });
  expect(resp.ok()).toBeTruthy();
  const body = await resp.json();
  return body.data.access_token as string;
}

/**
 * Create a resource via POST and return decoded data.
 */
export async function apiPost<T = any>(
  request: APIRequestContext,
  path: string,
  token: string,
  body: Record<string, unknown>,
): Promise<T> {
  const resp = await request.post(path, {
    headers: { Authorization: `Bearer ${token}` },
    data: body,
  });
  expect(resp.ok(), `POST ${path} failed: ${resp.status()}`).toBeTruthy();
  const json = await resp.json();
  return json.data as T;
}

export async function apiGet<T = any>(
  request: APIRequestContext,
  path: string,
  token: string,
): Promise<T> {
  const resp = await request.get(path, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok(), `GET ${path} failed: ${resp.status()}`).toBeTruthy();
  const json = await resp.json();
  return json.data as T;
}

export async function apiPatch<T = any>(
  request: APIRequestContext,
  path: string,
  token: string,
  body: Record<string, unknown>,
): Promise<T> {
  const resp = await request.patch(path, {
    headers: { Authorization: `Bearer ${token}` },
    data: body,
  });
  expect(resp.ok(), `PATCH ${path} failed: ${resp.status()}`).toBeTruthy();
  const json = await resp.json();
  return json.data as T;
}

export async function apiDelete(
  request: APIRequestContext,
  path: string,
  token: string,
): Promise<void> {
  const resp = await request.delete(path, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok(), `DELETE ${path} failed: ${resp.status()}`).toBeTruthy();
}

export async function apiPut(
  request: APIRequestContext,
  path: string,
  token: string,
  body?: Record<string, unknown>,
): Promise<void> {
  const resp = await request.put(path, {
    headers: { Authorization: `Bearer ${token}` },
    data: body,
  });
  expect(resp.ok(), `PUT ${path} failed: ${resp.status()}`).toBeTruthy();
}

// ─── UI helpers ─────────────────────────────────────────────────────────────

/**
 * Login via the UI login page and wait for navigation to home.
 */
export async function uiLogin(
  page: Page,
  email = 'admin@agach.local',
  password = 'admin',
): Promise<void> {
  await page.goto('/login');
  await page.locator('[data-qa="login-email-input"]').fill(email);
  await page.locator('[data-qa="login-password-input"]').fill(password);
  await page.locator('[data-qa="login-submit-btn"]').click();
  // Wait for redirect to home page
  await page.waitForURL('/', { timeout: 15_000 });
}

/**
 * Ensure the page is authenticated. Uses localStorage injection for speed.
 */
export async function ensureLoggedIn(
  page: Page,
  request: APIRequestContext,
): Promise<string> {
  const token = await apiLogin(request);
  // Set auth in localStorage before navigating
  await page.goto('/login');
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t);
  }, token);
  return token;
}

let slugCounter = 0;
export function uniqueSlug(prefix: string): string {
  slugCounter++;
  return `${prefix}-pw-${slugCounter}-${Date.now()}`;
}
