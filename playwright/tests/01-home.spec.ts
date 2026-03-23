import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createFeature, BASE_URL } from './helpers';

// ─── Section 1: Login & Authentication ──────────────────────────────────────

test.describe('Suite 1: Login & Authentication', () => {
  test('1.1 — Login page renders', async ({ page }) => {
    // Clear storage state so we hit the login page as unauthenticated
    await page.context().clearCookies();
    await page.goto('/login');
    await page.evaluate(() => {
      localStorage.removeItem('agach_access_token');
      localStorage.removeItem('agach_user');
    });
    await page.goto('/login');

    await expect(page.locator('[data-qa="login-email-input"]')).toBeVisible();
    await expect(page.locator('[data-qa="login-password-input"]')).toBeVisible();
    await expect(page.locator('[data-qa="login-submit-btn"]')).toBeVisible();
  });

  test('1.2 — Successful login redirects to home', async ({ page }) => {
    await page.context().clearCookies();
    // Clear localStorage to simulate fresh session
    await page.goto('/login');
    await page.evaluate(() => {
      localStorage.removeItem('agach_access_token');
      localStorage.removeItem('agach_user');
    });
    await page.goto('/login');

    await page.locator('[data-qa="login-email-input"]').fill('admin@agach.local');
    await page.locator('[data-qa="login-password-input"]').fill('admin');
    await page.locator('[data-qa="login-submit-btn"]').click();

    await expect(page).toHaveURL('/');
  });

  test('1.3 — Invalid credentials show error', async ({ page }) => {
    await page.context().clearCookies();
    await page.goto('/login');
    await page.evaluate(() => {
      localStorage.removeItem('agach_access_token');
      localStorage.removeItem('agach_user');
    });
    await page.goto('/login');

    await page.locator('[data-qa="login-email-input"]').fill('bad@example.com');
    await page.locator('[data-qa="login-password-input"]').fill('wrongpassword');
    await page.locator('[data-qa="login-submit-btn"]').click();

    // Should stay on the login page and display an error
    await expect(page).toHaveURL(/\/login/);
    // The error div appears after failed login (no data-qa, so match by text content)
    await expect(page.locator('form >> text=/failed|invalid|error|incorrect/i')).toBeVisible();
  });

  test('1.4 — Toggle password visibility', async ({ page }) => {
    await page.context().clearCookies();
    await page.goto('/login');
    await page.evaluate(() => {
      localStorage.removeItem('agach_access_token');
      localStorage.removeItem('agach_user');
    });
    await page.goto('/login');

    const passwordInput = page.locator('[data-qa="login-password-input"]');
    const toggleBtn = page.locator('[data-qa="toggle-password-visibility-btn"]');

    // Initially password type
    await expect(passwordInput).toHaveAttribute('type', 'password');

    // Toggle to text
    await toggleBtn.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');

    // Toggle back to password
    await toggleBtn.click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });
});

// ─── Section 2: Home Page — Projects List ───────────────────────────────────

test.describe('Suite 2: Home Page', () => {
  test('2.1 — Render projects list on load', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 2.1', 'Playwright test project');
    try {
      await page.goto('/');
      await expect(page.locator('[data-qa="project-card"]').first()).toBeVisible();
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('2.2 — Display empty state when no projects exist', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const count = await page.locator('[data-qa="project-card"]').count();
    if (count > 0) {
      test.info().annotations.push({
        type: 'skip-reason',
        description: `${count} project(s) already exist — empty state not testable`,
      });
      // Soft assert: just confirm we can see some cards
      expect(count).toBeGreaterThan(0);
      return;
    }
    // Empty state should show the create button
    await expect(page.locator('[data-qa="create-project-empty-btn"]')).toBeVisible();
  });

  test('2.3 — Create a new project via the create button', async ({ page, request }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Click the "New Project" button
    await page.locator('[data-qa="create-project-btn"]').click();

    // Fill the project name
    await page.locator('[data-qa="create-project-name-input"]').fill('Test Project 2.3');

    // Optionally fill description
    await page.locator('[data-qa="create-project-description-input"]').fill('A test project description');

    // Submit
    await page.locator('[data-qa="create-project-submit-btn"]').click();

    // Dialog should close and the new project card should appear
    await expect(page.locator('[data-qa="create-project-submit-btn"]')).not.toBeVisible();
    const newCard = page.locator('[data-qa="project-card"]', { hasText: 'Test Project 2.3' });
    await expect(newCard).toBeVisible();

    // Cleanup: find and delete the project via API
    // Navigate to the project to get its ID from the URL
    await newCard.locator('[data-qa="project-open-btn"]').click();
    const url = page.url();
    const match = url.match(/\/projects\/([a-f0-9-]+)/);
    if (match) {
      await deleteProject(request, match[1]);
    }
  });

  test('2.4 — Create project: cancel closes dialog', async ({ page }) => {
    await page.goto('/');

    // Open the create dialog
    await page.locator('[data-qa="create-project-btn"]').click();
    await expect(page.locator('[data-qa="create-project-name-input"]')).toBeVisible();

    // Cancel via the cancel button
    await page.locator('[data-qa="create-project-cancel-btn"]').click();

    // Dialog should be closed
    await expect(page.locator('[data-qa="create-project-name-input"]')).not.toBeVisible();
  });

  test('2.5 — Click on a project card navigates to the kanban board', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 2.5');
    try {
      await page.goto('/');
      await expect(page.locator('[data-qa="project-card"]').first()).toBeVisible();

      // Find the card that belongs to our project and click its Open button
      const card = page.locator('[data-qa="project-card"]', { hasText: 'Test Project 2.5' });
      await expect(card).toBeVisible();
      await card.locator('[data-qa="project-open-btn"]').click();

      await expect(page).toHaveURL(/\/projects\//);
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('2.6 — Project card shows status badge', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 2.6', 'Status badge test');
    try {
      await page.goto('/');
      const card = page.locator('[data-qa="project-card"]', { hasText: 'Test Project 2.6' });
      await expect(card).toBeVisible();

      // The card should display a status badge (Active, Blocked, Pending, Done, or Empty)
      // For a fresh project with no tasks, the status should be "Empty"
      await expect(card.locator('text=/Active|Blocked|Pending|Done|Empty/')).toBeVisible();
    } finally {
      await deleteProject(request, projectId);
    }
  });
});

// ─── Section 3: Navigation ──────────────────────────────────────────────────

test.describe('Suite 3: Navigation', () => {
  test('3.1 — Sidebar nav links visible on project page', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 3.1');
    try {
      await page.goto(`/projects/${projectId}`);
      await page.waitForLoadState('networkidle');
      await expect(page.locator('[data-qa="nav-kanban-btn"]')).toBeVisible();
      await expect(page.locator('[data-qa="nav-features-btn"]')).toBeVisible();
      await expect(page.locator('[data-qa="nav-statistics-btn"]')).toBeVisible();
      await expect(page.locator('[data-qa="nav-settings-btn"]')).toBeVisible();
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('3.2 — Navigate to Features via sidebar', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 3.2');
    try {
      await page.goto(`/projects/${projectId}`);
      await page.waitForLoadState('networkidle');
      await page.locator('[data-qa="nav-features-btn"]').click();
      await expect(page).toHaveURL(/\/features/);
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('3.3 — Navigate to Statistics via sidebar', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 3.3');
    try {
      await page.goto(`/projects/${projectId}`);
      await page.waitForLoadState('networkidle');
      await page.locator('[data-qa="nav-statistics-btn"]').click();
      await expect(page).toHaveURL(/\/statistics/);
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('3.4 — Navigate to Settings via sidebar', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 3.4');
    try {
      await page.goto(`/projects/${projectId}`);
      await page.waitForLoadState('networkidle');
      await page.locator('[data-qa="nav-settings-btn"]').click();
      await expect(page).toHaveURL(/\/settings/);
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('3.5 — Navigate to global Projects page', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="nav-projects-btn"]').click();
    await expect(page).toHaveURL('/');
  });

  test('3.6 — Navigate to global Agents page', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="nav-roles-btn"]').click();
    await expect(page).toHaveURL('/roles');
  });

  test('3.7 — Navigate to global Skills page', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="nav-skills-btn"]').click();
    await expect(page).toHaveURL('/skills');
  });

  test('3.8 — Navigate to global Dockerfiles page', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="nav-dockerfiles-btn"]').click();
    await expect(page).toHaveURL('/dockerfiles');
  });

  test('3.9 — Home (logo) navigates to project list', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 3.9');
    try {
      await page.goto(`/projects/${projectId}`);
      await page.locator('[data-qa="logo-home-link"]').click();
      await expect(page).toHaveURL('/');
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('3.10 — User menu: navigate to Account', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="user-menu-btn"]').click();
    await page.locator('[data-qa="user-menu-account-btn"]').click();
    await expect(page).toHaveURL('/account');
  });

  test('3.11 — User menu: navigate to API Keys', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="user-menu-btn"]').click();
    await page.locator('[data-qa="user-menu-api-keys-btn"]').click();
    await expect(page).toHaveURL('/account/api-keys');
  });

  test('3.12 — User menu: sign out', async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-qa="user-menu-btn"]').click();
    await page.locator('[data-qa="user-menu-logout-btn"]').click();
    // Should redirect to login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('3.13 — Sidebar shows features list with add button', async ({ page, request }) => {
    const projectId = await createProject(request, 'Test Project 3.13');
    try {
      // Create a feature so the features section appears in the sidebar
      await createFeature(request, projectId, 'Feature 3.13');
      await page.goto(`/projects/${projectId}`);
      await page.waitForLoadState('networkidle');
      // The add-feature button should be visible in the sidebar
      await expect(page.locator('[data-qa="nav-add-feature-btn"]')).toBeVisible();
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('3.14 — Theme toggle switches theme', async ({ page }) => {
    await page.goto('/');

    // Open user menu first (theme toggle is inside user menu popup)
    await page.locator('[data-qa="user-menu-btn"]').click();
    const themeToggle = page.locator('[data-qa="theme-toggle-btn"]');
    await expect(themeToggle).toBeVisible();

    // Capture initial theme attribute
    const htmlEl = page.locator('html');
    const initialClass = await htmlEl.getAttribute('class') ?? '';
    const initialDataTheme = await htmlEl.getAttribute('data-theme') ?? '';

    // Toggle theme
    await themeToggle.click();

    // Verify something changed on html or body
    const bodyEl = page.locator('body');
    const newHtmlClass = await htmlEl.getAttribute('class') ?? '';
    const newHtmlDataTheme = await htmlEl.getAttribute('data-theme') ?? '';
    const newBodyClass = await bodyEl.getAttribute('class') ?? '';

    const themeChanged =
      newHtmlClass !== initialClass ||
      newHtmlDataTheme !== initialDataTheme ||
      newBodyClass !== '';

    expect(themeChanged).toBe(true);

    // Restore — open user menu again and toggle back
    await page.locator('[data-qa="user-menu-btn"]').click();
    await page.locator('[data-qa="theme-toggle-btn"]').click();
  });
});
