import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createFeature, createRole, deleteRole, BASE_URL } from './helpers';

test.describe('5. Roles Management', () => {
  test('5.1 — Roles page shows list of roles', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await expect(page.getByRole('heading', { name: /roles/i })).toBeVisible();
    const count = await page.locator('[data-qa="role-card"]').count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('5.2 — Create a new role (global)', async ({ page, request }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-role-btn"]').click();

    await expect(page.locator('[data-qa="role-name-input"]')).toBeVisible();
    await page.locator('[data-qa="role-name-input"]').fill('Test Role QA');

    // Slug should auto-fill
    const slugValue = await page.locator('[data-qa="role-slug-input"]').inputValue();
    expect(slugValue.length).toBeGreaterThan(0);

    await page.locator('[data-qa="role-save-btn"]').click();

    await expect(
      page.locator('[data-qa="role-card"]').filter({ hasText: 'Test Role QA' }),
    ).toBeVisible();

    const finalSlug = slugValue || 'test-role-qa';
    try {
      await deleteRole(request, finalSlug);
    } catch {
      // Best-effort cleanup
    }
  });

  test('5.3 — Role name is required', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-role-btn"]').click();

    await expect(page.locator('[data-qa="role-name-input"]')).toBeVisible();
    // Leave name empty — save button should be disabled
    await expect(page.locator('[data-qa="role-save-btn"]')).toBeDisabled();

    // Modal should stay open
    await expect(page.locator('[data-qa="role-name-input"]')).toBeVisible();
  });

  test('5.4 — Cancel role creation', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);

    const cardsBefore = await page.locator('[data-qa="role-card"]').count();

    await page.locator('[data-qa="new-role-btn"]').click();
    await expect(page.locator('[data-qa="role-name-input"]')).toBeVisible();

    await page.locator('[data-qa="role-name-input"]').fill('Temp');
    await page.locator('[data-qa="role-cancel-btn"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="role-name-input"]')).not.toBeVisible();

    // No new card with "Temp" should exist
    await expect(
      page.locator('[data-qa="role-card"]').filter({ hasText: 'Temp' }),
    ).toHaveCount(0);

    // Total card count unchanged
    const cardsAfter = await page.locator('[data-qa="role-card"]').count();
    expect(cardsAfter).toBe(cardsBefore);
  });

  test('5.5 — Edit an existing role', async ({ page, request }) => {
    await createRole(request, 'qa-edit-role', 'QA Edit Role');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="role-card"]').filter({ hasText: 'QA Edit Role' });
      await expect(card).toBeVisible();
      await card.click();

      // Edit mode: delete button should be visible
      await expect(page.locator('[data-qa="role-delete-btn"]')).toBeVisible();

      await page.locator('[data-qa="role-name-input"]').fill('QA Edit Role Updated');
      await page.locator('[data-qa="role-save-btn"]').click();

      await expect(
        page.locator('[data-qa="role-card"]').filter({ hasText: 'QA Edit Role Updated' }),
      ).toBeVisible();
    } finally {
      await deleteRole(request, 'qa-edit-role');
    }
  });

  test('5.6 — Delete a role', async ({ page, request }) => {
    await createRole(request, 'qa-delete-role', 'QA Delete Role');

    await page.goto(`${BASE_URL}/roles`);

    const card = page.locator('[data-qa="role-card"]').filter({ hasText: 'QA Delete Role' });
    await expect(card).toBeVisible();
    await card.click();

    await expect(page.locator('[data-qa="role-delete-btn"]')).toBeVisible();
    await page.locator('[data-qa="role-delete-btn"]').click();

    // Modal should close
    await expect(page.locator('[data-qa="role-name-input"]')).not.toBeVisible();

    // Card should be gone
    await expect(
      page.locator('[data-qa="role-card"]').filter({ hasText: 'QA Delete Role' }),
    ).toHaveCount(0);
  });

  test('5.7 — Project-scoped roles page', async ({ page, request }) => {
    const projectId = await createProject(request, 'Roles Scoped Project 5.7');
    try {
      await page.goto(`${BASE_URL}/projects/${projectId}/roles`);
      await expect(page.getByRole('heading', { name: /roles/i })).toBeVisible();
    } finally {
      await deleteProject(request, projectId);
    }
  });
});

test.describe('6. Features', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Features Test Project');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('6.1 — Features page empty state', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Either empty state button or "no features" text should be visible
    const emptyBtn = page.locator('[data-qa="create-first-feature-btn"]');
    const noFeaturesText = page.getByText(/no features/i);

    const hasEmpty = await emptyBtn.isVisible().catch(() => false);
    const hasText = await noFeaturesText.isVisible().catch(() => false);

    expect(hasEmpty || hasText).toBe(true);
  });

  test('6.2 — Create a feature', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    await page.locator('[data-qa="new-feature-name-input"]').fill('Feature Alpha');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Alpha' }),
    ).toBeVisible();
  });

  test('6.3 — Feature name is required', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    // Leave name empty — confirm button should be disabled
    await expect(page.locator('[data-qa="confirm-create-feature-btn"]')).toBeDisabled();

    // Modal should stay open
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
  });

  test('6.4 — Cancel feature creation', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const countBefore = await page.locator('[data-qa="feature-list-item-btn"]').count();

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    await page.locator('[data-qa="new-feature-name-input"]').fill('Temp Feature');
    await page.locator('[data-qa="cancel-create-feature-btn"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="new-feature-name-input"]')).not.toBeVisible();

    // No new feature created
    const countAfter = await page.locator('[data-qa="feature-list-item-btn"]').count();
    expect(countAfter).toBe(countBefore);
  });

  test('6.5 — Click feature to open drawer', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Create a feature via the UI
    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="new-feature-name-input"]').fill('Feature Beta');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    const featureItem = page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Beta' });
    await expect(featureItem).toBeVisible();
    await featureItem.click();

    // Drawer should open: close button or board link should be visible
    const closeBtn = page.locator('[data-qa="close-feature-drawer-btn"]');
    const boardLink = page.locator('[data-qa="open-feature-board-link"]');

    const hasClose = await closeBtn.isVisible().catch(() => false);
    const hasBoard = await boardLink.isVisible().catch(() => false);

    expect(hasClose || hasBoard).toBe(true);
  });

  test('6.6 — Open feature board navigates to feature project', async ({ page, request }) => {
    // Create feature via API
    const featureId = await createFeature(request, projectId, 'Feature Gamma');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Click feature to open drawer
    const featureItem = page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Gamma' });
    await expect(featureItem).toBeVisible();
    await featureItem.click();

    await expect(page.locator('[data-qa="open-feature-board-link"]')).toBeVisible();
    await page.locator('[data-qa="open-feature-board-link"]').click();

    // URL should navigate to the feature's own project page
    await expect(page).toHaveURL(new RegExp(`/projects/${featureId}`));
  });

  test('6.7 — Edit feature name', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Create feature
    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="new-feature-name-input"]').fill('Feature Delta');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    // Click feature to open drawer
    const featureItem = page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Delta' });
    await expect(featureItem).toBeVisible();
    await featureItem.click();

    // Open edit modal
    await expect(page.locator('[data-qa="edit-feature-btn"]')).toBeVisible();
    await page.locator('[data-qa="edit-feature-btn"]').click();

    await expect(page.locator('[data-qa="edit-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="edit-feature-name-input"]').clear();
    await page.locator('[data-qa="edit-feature-name-input"]').fill('Feature Updated');
    await page.locator('[data-qa="confirm-edit-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Updated' }),
    ).toBeVisible();
  });

  test('6.8 — Delete feature', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Create feature
    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="new-feature-name-input"]').fill('Feature Epsilon');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    const featureItem = page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Epsilon' });
    await expect(featureItem).toBeVisible();
    await featureItem.click();

    await expect(page.locator('[data-qa="delete-feature-btn"]')).toBeVisible();
    await page.locator('[data-qa="delete-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-list-item-btn"]').filter({ hasText: 'Feature Epsilon' }),
    ).toHaveCount(0);
  });

  test('6.9 — New task modal shows feature selector when features exist', async ({ page, request }) => {
    // Create a feature so the selector appears in the modal
    await createFeature(request, projectId, 'Feature Zeta');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    // Feature selector should be present since a feature exists
    await expect(page.locator('[data-qa="new-task-feature-select"]')).toBeVisible();
  });

  test('6.10 — Create task assigned to a feature', async ({ page, request }) => {
    const featureId = await createFeature(request, projectId, 'Feature Eta');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-title-input"]').fill('Feature Task Eta');
    await page.locator('[data-qa="new-task-summary-input"]').fill('Task belonging to feature Eta');

    // Select the feature
    await page.locator('[data-qa="new-task-feature-select"]').selectOption(featureId);
    await page.locator('[data-qa="new-task-submit-btn"]').click();

    // Task should appear on the board
    await expect(
      page.locator('[data-qa="task-card"]').filter({ hasText: 'Feature Task Eta' }),
    ).toBeVisible();
  });
});
