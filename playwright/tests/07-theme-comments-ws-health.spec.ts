import { test, expect, request as pwRequest } from '@playwright/test';
import { createProject, deleteProject, createTask, moveTask, BASE_URL } from './helpers';

// ─────────────────────────────────────────────
// Suite 15: Account
// ─────────────────────────────────────────────

test.describe('15. Account', () => {
  test('15.1 — Account page loads profile', async ({ page }) => {
    await page.goto(`${BASE_URL}/account`);
    await page.waitForLoadState('networkidle');

    const input = page.locator('[data-qa="account-display-name-input"]');
    await expect(input).toBeVisible();
    await expect(input).not.toHaveValue('');
  });

  test('15.2 — Update display name', async ({ page }) => {
    await page.goto(`${BASE_URL}/account`);
    await page.waitForLoadState('networkidle');

    const input = page.locator('[data-qa="account-display-name-input"]');
    await expect(input).toBeVisible();

    // Save original name for restoration
    const originalName = await input.inputValue();
    const newName = 'Test User ' + Date.now();

    await input.clear();
    await input.fill(newName);
    await page.click('[data-qa="account-save-profile-btn"]');

    // Expect success message
    await expect(page.getByText('Profile updated')).toBeVisible({ timeout: 5000 });

    // Restore original name
    await input.clear();
    await input.fill(originalName);
    await page.click('[data-qa="account-save-profile-btn"]');
    await expect(page.getByText('Profile updated')).toBeVisible({ timeout: 5000 });
  });

  test('15.3 — Change password', async ({ page }) => {
    await page.goto(`${BASE_URL}/account`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="account-current-password"]').fill('admin');
    await page.locator('[data-qa="account-new-password"]').fill('admin');
    await page.locator('[data-qa="account-confirm-password"]').fill('admin');
    await page.click('[data-qa="account-change-password-btn"]');

    // Expect success message
    await expect(page.getByText('Password changed')).toBeVisible({ timeout: 5000 });
  });
});

// ─────────────────────────────────────────────
// Suite 16: API Keys
// ─────────────────────────────────────────────

test.describe('16. API Keys', () => {
  test('16.1 — API Keys page loads', async ({ page }) => {
    await page.goto(`${BASE_URL}/account/api-keys`);
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-qa="create-api-key-btn"]')).toBeVisible();
  });

  test('16.2 — Create a new API key', async ({ page }) => {
    await page.goto(`${BASE_URL}/account/api-keys`);
    await page.waitForLoadState('networkidle');

    await page.click('[data-qa="create-api-key-btn"]');

    const nameInput = page.locator('[data-qa="api-key-name-input"]');
    await expect(nameInput).toBeVisible();
    await nameInput.fill('Test Key');

    // Ensure scopes are checked (they default to both checked)
    await expect(page.locator('[data-qa="scope-kanban:read"]')).toBeVisible();
    await expect(page.locator('[data-qa="scope-kanban:write"]')).toBeVisible();

    await page.click('[data-qa="create-api-key-submit-btn"]');

    // Key value shown with copy button (shown only once)
    await expect(page.locator('[data-qa="copy-api-key-btn"]')).toBeVisible({ timeout: 5000 });
  });

  test('16.3 — Revoke an API key', async ({ page }) => {
    await page.goto(`${BASE_URL}/account/api-keys`);
    await page.waitForLoadState('networkidle');

    // Create a key first if none exists
    await page.click('[data-qa="create-api-key-btn"]');
    await page.locator('[data-qa="api-key-name-input"]').fill('Key to Revoke');
    await page.click('[data-qa="create-api-key-submit-btn"]');
    await expect(page.locator('[data-qa="copy-api-key-btn"]')).toBeVisible({ timeout: 5000 });

    // Find a revoke button (data-qa="revoke-key-{id}")
    const revokeBtn = page.locator('[data-qa^="revoke-key-"]').first();
    await expect(revokeBtn).toBeVisible();

    // Count keys before revocation
    const keyCountBefore = await page.locator('[data-qa^="revoke-key-"]').count();

    await revokeBtn.click();

    // Key should be removed from list
    await expect(page.locator('[data-qa^="revoke-key-"]')).toHaveCount(keyCountBefore - 1, {
      timeout: 5000,
    });
  });
});

// ─────────────────────────────────────────────
// Suite 19: Theme Toggle
// ─────────────────────────────────────────────

test.describe('19. Theme Toggle', () => {
  test('19.1 — Theme toggle button is visible', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-qa="theme-toggle-btn"]')).toBeVisible();
  });

  test('19.2 — Toggle from dark to light theme', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    const html = page.locator('html');
    const initialTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    await page.click('[data-qa="theme-toggle-btn"]');

    const newTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    expect(newTheme).not.toEqual(initialTheme);

    // Restore original theme
    await page.click('[data-qa="theme-toggle-btn"]');
  });

  test('19.3 — Toggle from light to dark theme', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    const html = page.locator('html');

    // Switch to light first
    await page.click('[data-qa="theme-toggle-btn"]');
    const lightTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    // Now toggle back to dark
    await page.click('[data-qa="theme-toggle-btn"]');
    const darkTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    expect(darkTheme).not.toEqual(lightTheme);
  });

  test('19.4 — Theme preference persists across navigation', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    const html = page.locator('html');
    const initialTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    // Toggle theme
    await page.click('[data-qa="theme-toggle-btn"]');
    const switchedTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));
    expect(switchedTheme).not.toEqual(initialTheme);

    // Navigate to a different route
    await page.goto(`${BASE_URL}/agents`);
    await page.waitForLoadState('networkidle');

    const themeAfterNav =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));
    expect(themeAfterNav).toEqual(switchedTheme);

    // Restore original theme
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');
    await page.click('[data-qa="theme-toggle-btn"]');
  });
});

// ─────────────────────────────────────────────
// Suite 20: Comments
// ─────────────────────────────────────────────

test.describe('20. Comments', () => {
  let projectId: string;
  let taskId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Comments Test Project');
    taskId = await createTask(request, projectId, 'Comment Test Task', 'Task for comment tests');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });

  test.afterEach(async ({ request }) => {
    try {
      await deleteProject(request, projectId);
    } catch {
      // best-effort cleanup
    }
  });

  test('20.1 — Comments load when Task Drawer is opened', async ({ page }) => {
    // Comment section should be visible with header text
    await expect(page.locator('[data-qa="comment-content-input"]')).toBeVisible();
    // Comments header should exist
    await expect(page.getByText('Comments')).toBeVisible();
  });

  test('20.2 — Post a new comment', async ({ page }) => {
    // Select author (defaults to Human already)
    await expect(page.locator('[data-qa="comment-author-select"]')).toBeVisible();

    // Type and submit
    await page.locator('[data-qa="comment-content-input"]').fill('This is a test comment');
    await page.click('[data-qa="comment-submit-btn"]');

    // Comment appears in list
    await expect(page.getByText('This is a test comment')).toBeVisible({ timeout: 5000 });

    // Input is cleared after submission
    await expect(page.locator('[data-qa="comment-content-input"]')).toHaveValue('');
  });

  test('20.3 — Comment "Post as" selector defaults to "Human"', async ({ page }) => {
    const select = page.locator('[data-qa="comment-author-select"]');
    await expect(select).toBeVisible();
    await expect(select).toHaveValue('human');
  });

  test('20.4 — Send comment with Ctrl+Enter', async ({ page }) => {
    await page.locator('[data-qa="comment-content-input"]').fill('Ctrl+Enter comment');
    await page.locator('[data-qa="comment-content-input"]').press('Control+Enter');
    await expect(page.getByText('Ctrl+Enter comment')).toBeVisible({ timeout: 5000 });
  });

  test('20.5 — Upload image button is visible', async ({ page }) => {
    await expect(page.locator('[data-qa="comment-upload-image-btn"]')).toBeVisible();
  });

  test('20.6 — Empty comment cannot be submitted', async ({ page }) => {
    const input = page.locator('[data-qa="comment-content-input"]');
    await expect(input).toHaveValue('');
    await expect(page.locator('[data-qa="comment-submit-btn"]')).toBeDisabled();
  });
});

// ─────────────────────────────────────────────
// Suite 21: Real-time WebSocket Updates
// ─────────────────────────────────────────────

test.describe('21. Real-time WebSocket Updates', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'WebSocket Test Project');
  });

  test.afterEach(async ({ request }) => {
    try {
      await deleteProject(request, projectId);
    } catch {
      // best-effort cleanup
    }
  });

  test('21.1 — New task appears on board without refresh', async ({ page, request }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const title = 'WS Live Task ' + Date.now();
    await createTask(request, projectId, title, 'Real-time task summary');

    // Task should appear via WebSocket without page reload
    await expect(
      page.locator('[data-qa="task-card"]').filter({ hasText: title }),
    ).toBeVisible({ timeout: 5000 });
  });

  test('21.2 — Task move is reflected in all connected clients', async ({
    browser,
    request,
  }) => {
    const title = 'WS Move Task ' + Date.now();
    const taskId = await createTask(request, projectId, title, 'Task to be moved');

    // Open two browser contexts (simulating two tabs)
    const context1 = await browser.newContext({ storageState: '/tmp/auth-state.json' });
    const context2 = await browser.newContext({ storageState: '/tmp/auth-state.json' });
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    await page1.goto(`${BASE_URL}/projects/${projectId}`);
    await page1.waitForLoadState('networkidle');
    await page2.goto(`${BASE_URL}/projects/${projectId}`);
    await page2.waitForLoadState('networkidle');

    // Ensure task is visible on both pages
    await expect(
      page1.locator('[data-qa="task-card"]').filter({ hasText: title }),
    ).toBeVisible({ timeout: 5000 });

    // Move task via API (simulating Tab 2 action)
    await moveTask(request, projectId, taskId, 'in_progress');

    // Tab 1 should reflect the move automatically
    await expect(
      page1
        .locator('[data-qa="column"]')
        .filter({ hasText: 'In Progress' })
        .locator('[data-qa="task-card"]')
        .filter({ hasText: title }),
    ).toBeVisible({ timeout: 5000 });

    await context1.close();
    await context2.close();
  });

  test('21.3 — Task deletion is reflected in all connected clients', async ({
    browser,
    request,
  }) => {
    const title = 'WS Delete Task ' + Date.now();
    const taskId = await createTask(request, projectId, title, 'Task to be deleted');

    // Open two browser contexts
    const context1 = await browser.newContext({ storageState: '/tmp/auth-state.json' });
    const page1 = await context1.newPage();

    await page1.goto(`${BASE_URL}/projects/${projectId}`);
    await page1.waitForLoadState('networkidle');

    // Ensure task appears on page 1
    await expect(
      page1.locator('[data-qa="task-card"]').filter({ hasText: title }),
    ).toBeVisible({ timeout: 5000 });

    // Delete task via API
    const token = await (async () => {
      const fs = await import('fs');
      const raw = fs.readFileSync('/tmp/auth-state.json', 'utf8');
      const state = JSON.parse(raw);
      for (const origin of state.origins ?? []) {
        for (const item of origin.localStorage ?? []) {
          if (item.name === 'agach_access_token') return item.value;
        }
      }
      return '';
    })();

    await request.fetch(`${BASE_URL}/api/projects/${projectId}/tasks/${taskId}`, {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });

    // Task should disappear from page 1 via WebSocket
    await expect(
      page1.locator('[data-qa="task-card"]').filter({ hasText: title }),
    ).not.toBeVisible({ timeout: 5000 });

    await context1.close();
  });
});

// ─────────────────────────────────────────────
// Suite 22: API Health Check
// ─────────────────────────────────────────────

test.describe('22. API Health Check', () => {
  test('22.1 — Projects endpoint returns 200 (authenticated)', async ({ request }) => {
    // Read token from auth state
    const fs = await import('fs');
    const raw = fs.readFileSync('/tmp/auth-state.json', 'utf8');
    const state = JSON.parse(raw);
    let token = '';
    for (const origin of state.origins ?? []) {
      for (const item of origin.localStorage ?? []) {
        if (item.name === 'agach_access_token') token = item.value;
      }
    }

    const response = await request.fetch(`${BASE_URL}/api/projects`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.status).toBe('success');
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test('22.2 — Server returns correct Content-Type', async ({ request }) => {
    const fs = await import('fs');
    const raw = fs.readFileSync('/tmp/auth-state.json', 'utf8');
    const state = JSON.parse(raw);
    let token = '';
    for (const origin of state.origins ?? []) {
      for (const item of origin.localStorage ?? []) {
        if (item.name === 'agach_access_token') token = item.value;
      }
    }

    const response = await request.fetch(`${BASE_URL}/api/projects`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    const contentType = response.headers()['content-type'] ?? '';
    expect(contentType).toContain('application/json');
  });

  test('22.3 — Unauthenticated request returns 401', async () => {
    // Create a fresh request context without any storage state (no auth)
    const unauthContext = await pwRequest.newContext();
    const response = await unauthContext.fetch(`${BASE_URL}/api/projects`, {
      method: 'GET',
    });
    expect(response.status()).toBe(401);
    await unauthContext.dispose();
  });
});
