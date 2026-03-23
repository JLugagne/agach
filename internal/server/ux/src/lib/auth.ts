// Auth token storage and management utilities.

const ACCESS_TOKEN_KEY = 'agach_access_token';

export function getToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
}

export interface LoginResponse {
  user: {
    id: string;
    email: string;
    display_name: string;
    role: string;
  };
  access_token: string;
}

export async function login(email: string, password: string, rememberMe = false): Promise<LoginResponse> {
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, remember_me: rememberMe }),
  });
  const json = await res.json();
  if (!res.ok || json.status !== 'success') {
    throw new Error(json.error?.message || 'Login failed');
  }
  return json.data as LoginResponse;
}

export async function logout(): Promise<void> {
  const token = getToken();
  if (token) {
    await fetch('/api/auth/logout', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => {});
  }
  clearToken();
}

export async function refreshAccessToken(): Promise<string | null> {
  const res = await fetch('/api/auth/refresh', { method: 'POST' });
  if (!res.ok) return null;
  const json = await res.json();
  if (json.status !== 'success') return null;
  return (json.data as { access_token: string }).access_token;
}
