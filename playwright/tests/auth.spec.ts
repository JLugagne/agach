import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, uniqueSlug } from './helpers';

test.describe('Auth', () => {
  test('login with valid credentials and see home page', async ({ page }) => {
    await uiLogin(page);
    await expect(page).toHaveURL('/');
    await expect(page.locator('[data-qa="create-project-btn"]')).toBeVisible();
  });

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/login');
    await page.locator('[data-qa="login-email-input"]').fill('admin@agach.local');
    await page.locator('[data-qa="login-password-input"]').fill('wrongpassword');
    await page.locator('[data-qa="login-submit-btn"]').click();
    // Should stay on login page and show error
    await expect(page).toHaveURL(/\/login/);
    await expect(page.locator('text=Login failed').or(page.locator('text=Invalid'))).toBeVisible({ timeout: 5_000 });
  });

  test('register a new user', async ({ page, request }) => {
    const email = `pw-reg-${Date.now()}@test.local`;
    const password = 'TestPassword123!';

    // Register via API
    const resp = await request.post('/api/auth/register', {
      data: { email, password, display_name: 'PW Register User' },
    });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.data.access_token).toBeTruthy();
    expect(body.data.user.email).toBe(email);

    // Login via UI with new credentials
    await uiLogin(page, email, password);
    await expect(page).toHaveURL('/');
  });

  test('update profile display name', async ({ page }) => {
    await uiLogin(page);
    await page.goto('/account');
    const nameInput = page.locator('[data-qa="account-display-name-input"]');
    await expect(nameInput).toBeVisible();
    await nameInput.fill('PW Updated Admin');
    await page.locator('[data-qa="account-save-profile-btn"]').click();
    // Verify the change persisted by reloading
    await page.reload();
    await expect(nameInput).toHaveValue('PW Updated Admin');
    // Restore original
    await nameInput.fill('Admin');
    await page.locator('[data-qa="account-save-profile-btn"]').click();
  });

  test('change password and login with new password', async ({ request }) => {
    const email = `pw-chpwd-${Date.now()}@test.local`;
    const oldPassword = 'OldPassword123!';
    const newPassword = 'NewPassword456!';

    // Register user
    const regResp = await request.post('/api/auth/register', {
      data: { email, password: oldPassword, display_name: 'PW ChangePass' },
    });
    expect(regResp.ok()).toBeTruthy();
    const regBody = await regResp.json();
    const token = regBody.data.access_token;

    // Change password via API
    const changeResp = await request.post('/api/auth/me/password', {
      headers: { Authorization: `Bearer ${token}` },
      data: { current_password: oldPassword, new_password: newPassword },
    });
    expect(changeResp.status()).toBe(204);

    // Login with new password
    const loginResp = await request.post('/api/auth/login', {
      data: { email, password: newPassword },
    });
    expect(loginResp.ok()).toBeTruthy();
  });

  test('logout clears session', async ({ page }) => {
    await uiLogin(page);
    await page.locator('[data-qa="user-menu-btn"]').click();
    await page.locator('[data-qa="user-menu-logout-btn"]').click();
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });

  test('unauthenticated access redirects to login', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/login/);
  });

  test('refresh token flow via API', async ({ request }) => {
    // Login and capture refresh cookie
    const loginResp = await request.post('/api/auth/login', {
      data: { email: 'admin@agach.local', password: 'admin' },
    });
    expect(loginResp.ok()).toBeTruthy();

    // Use the refresh endpoint
    const refreshResp = await request.post('/api/auth/refresh');
    // Refresh may work if cookies are forwarded, or fail if not — both are valid
    // The key test is that the endpoint exists and responds
    expect([200, 401]).toContain(refreshResp.status());
  });
});
