import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, uniqueSlug } from './helpers';

test.describe('Invite', () => {
  test('full invite workflow via API', async ({ request }) => {
    const token = await apiLogin(request);
    const email = `pw-invited-${Date.now()}@test.local`;

    // Admin invites user
    const resp = await request.post('/api/identity/users/invite', {
      headers: { Authorization: `Bearer ${token}` },
      data: { email },
    });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    const inviteToken = body.data.invite_token;
    expect(inviteToken).toBeTruthy();

    // Verify user exists in list
    const users = await apiGet<any[]>(request, '/api/identity/users', token);
    const invited = users.find((u: any) => u.email === email);
    expect(invited).toBeTruthy();
    expect(invited.role).toBe('member');

    // Complete invite
    const completeResp = await request.post('/api/auth/complete-invite', {
      data: {
        token: inviteToken,
        display_name: 'Invited User',
        password: 'SecurePass123!',
      },
    });
    expect(completeResp.ok()).toBeTruthy();
    const completeBody = await completeResp.json();
    expect(completeBody.data.access_token).toBeTruthy();
    expect(completeBody.data.user.email).toBe(email);
    expect(completeBody.data.user.display_name).toBe('Invited User');

    // Login with new credentials
    const loginResp = await request.post('/api/auth/login', {
      data: { email, password: 'SecurePass123!' },
    });
    expect(loginResp.ok()).toBeTruthy();
  });

  test('complete invite twice fails', async ({ request }) => {
    const token = await apiLogin(request);
    const email = `pw-twice-${Date.now()}@test.local`;

    const inv = await apiPost<{ invite_token: string }>(
      request, '/api/identity/users/invite', token, { email },
    );

    // First completion
    const resp1 = await request.post('/api/auth/complete-invite', {
      data: { token: inv.invite_token, display_name: 'First', password: 'SecurePass123!' },
    });
    expect(resp1.ok()).toBeTruthy();

    // Second attempt should fail
    const resp2 = await request.post('/api/auth/complete-invite', {
      data: { token: inv.invite_token, display_name: 'Second', password: 'AnotherPass123!' },
    });
    expect(resp2.status()).toBe(400);
  });

  test('invalid invite token rejected', async ({ request }) => {
    const resp = await request.post('/api/auth/complete-invite', {
      data: { token: 'invalid.jwt.token', display_name: 'Hacker', password: 'Password123!' },
    });
    expect(resp.status()).toBe(401);
  });

  test('password too short rejected', async ({ request }) => {
    const token = await apiLogin(request);
    const email = `pw-short-${Date.now()}@test.local`;

    const inv = await apiPost<{ invite_token: string }>(
      request, '/api/identity/users/invite', token, { email },
    );

    const resp = await request.post('/api/auth/complete-invite', {
      data: { token: inv.invite_token, display_name: 'Short Pass', password: 'short' },
    });
    expect([400, 422]).toContain(resp.status());
  });

  test('duplicate email invite fails', async ({ request }) => {
    const token = await apiLogin(request);

    const resp = await request.post('/api/identity/users/invite', {
      headers: { Authorization: `Bearer ${token}` },
      data: { email: 'admin@agach.local' },
    });
    expect(resp.status()).toBe(409);
  });

  test('non-admin cannot invite', async ({ request }) => {
    const email = `pw-member-${Date.now()}@test.local`;
    // Register a member
    const regResp = await request.post('/api/auth/register', {
      data: { email, password: 'MemberPass123!', display_name: 'Member User' },
    });
    expect(regResp.ok()).toBeTruthy();
    const regBody = await regResp.json();
    const memberToken = regBody.data.access_token;

    const resp = await request.post('/api/identity/users/invite', {
      headers: { Authorization: `Bearer ${memberToken}` },
      data: { email: 'someone@test.local' },
    });
    expect(resp.status()).toBe(403);
  });

  test('invite page UI with valid token', async ({ page, request }) => {
    const token = await apiLogin(request);
    const email = `pw-uiinvite-${Date.now()}@test.local`;

    const inv = await apiPost<{ invite_token: string }>(
      request, '/api/identity/users/invite', token, { email },
    );

    // Navigate to invite page with token
    await page.goto(`/invite?token=${inv.invite_token}`);

    // Fill in the form
    await page.locator('[data-qa="invite-display-name"]').fill('PW Invited User');
    await page.locator('[data-qa="invite-password"]').fill('SecurePass123!');
    await page.locator('[data-qa="invite-confirm-password"]').fill('SecurePass123!');
    await page.locator('[data-qa="invite-submit-btn"]').click();

    // Should redirect to home after successful registration
    await expect(page).toHaveURL('/', { timeout: 15_000 });
  });

  test('invite page without token shows error', async ({ page }) => {
    await page.goto('/invite');
    // Should show invalid invite link message
    await expect(page.locator('text=Invalid invite link')).toBeVisible();
  });
});
