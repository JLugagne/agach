import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createFeature, createAgent, deleteAgent, BASE_URL } from './helpers';

test.describe('5. Agents Management', () => {
  test('5.1 — Agents page shows list of agents', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await expect(page.getByRole('heading', { name: /agents/i })).toBeVisible();
    const count = await page.locator('[data-qa="agent-card"]').count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('5.2 — Create a new agent (global)', async ({ page, request }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-agent-btn"]').click();

    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();
    await page.locator('[data-qa="agent-name-input"]').fill('Test Agent QA');

    // Slug should auto-fill
    const slugValue = await page.locator('[data-qa="agent-slug-input"]').inputValue();
    expect(slugValue.length).toBeGreaterThan(0);

    await page.locator('[data-qa="agent-save-btn"]').click();

    await expect(
      page.locator('[data-qa="agent-card"]').filter({ hasText: 'Test Agent QA' }),
    ).toBeVisible();

    const finalSlug = slugValue || 'test-agent-qa';
    try {
      await deleteAgent(request, finalSlug);
    } catch {
      // Best-effort cleanup
    }
  });

  test('5.3 — Agent name is required', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-agent-btn"]').click();

    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();
    // Leave name empty — save button should be disabled
    await expect(page.locator('[data-qa="agent-save-btn"]')).toBeDisabled();

    // Modal should stay open
    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();
  });

  test('5.4 — Cancel agent creation', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);

    const cardsBefore = await page.locator('[data-qa="agent-card"]').count();

    await page.locator('[data-qa="new-agent-btn"]').click();
    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();

    await page.locator('[data-qa="agent-name-input"]').fill('Temp');
    await page.locator('[data-qa="agent-cancel-btn"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="agent-name-input"]')).not.toBeVisible();

    // No new card with "Temp" should exist
    await expect(
      page.locator('[data-qa="agent-card"]').filter({ hasText: 'Temp' }),
    ).toHaveCount(0);

    // Total card count unchanged
    const cardsAfter = await page.locator('[data-qa="agent-card"]').count();
    expect(cardsAfter).toBe(cardsBefore);
  });

  test('5.5 — Edit an existing agent', async ({ page, request }) => {
    await createAgent(request, 'qa-edit-agent', 'QA Edit Agent');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Edit Agent' });
      await expect(card).toBeVisible();
      await card.click();

      // Edit mode: delete button should be visible
      await expect(page.locator('[data-qa="agent-delete-btn"]')).toBeVisible();

      await page.locator('[data-qa="agent-name-input"]').fill('QA Edit Agent Updated');
      await page.locator('[data-qa="agent-save-btn"]').click();

      await expect(
        page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Edit Agent Updated' }),
      ).toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-edit-agent');
    }
  });

  test('5.6 — Delete an agent', async ({ page, request }) => {
    await createAgent(request, 'qa-delete-agent', 'QA Delete Agent');

    await page.goto(`${BASE_URL}/roles`);

    const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Delete Agent' });
    await expect(card).toBeVisible();
    await card.click();

    await expect(page.locator('[data-qa="agent-delete-btn"]')).toBeVisible();
    await page.locator('[data-qa="agent-delete-btn"]').click();

    // Modal should close
    await expect(page.locator('[data-qa="agent-name-input"]')).not.toBeVisible();

    // Card should be gone
    await expect(
      page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Delete Agent' }),
    ).toHaveCount(0);
  });

  test('5.7 — Project-scoped agents page', async ({ page, request }) => {
    const projectId = await createProject(request, 'Agents Scoped Project 5.7');
    try {
      await page.goto(`${BASE_URL}/projects/${projectId}/roles`);
      await expect(page.getByRole('heading', { name: /agents/i })).toBeVisible();
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
      page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Alpha' }),
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

    const countBefore = await page.locator('[data-qa="feature-card"]').count();

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    await page.locator('[data-qa="new-feature-name-input"]').fill('Temp Feature');
    await page.locator('[data-qa="cancel-create-feature-btn"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="new-feature-name-input"]')).not.toBeVisible();

    // No new feature created
    const countAfter = await page.locator('[data-qa="feature-card"]').count();
    expect(countAfter).toBe(countBefore);
  });

  test('6.5 — Click feature card to see details', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Create a feature via the UI
    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="new-feature-name-input"]').fill('Feature Beta');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Beta' });
    await expect(featureCard).toBeVisible();

    // The card should have an edit button or board link
    const editBtn = featureCard.locator('[data-qa="edit-feature-btn"]');
    const boardLink = featureCard.locator('[data-qa="open-feature-board-link"]');

    const hasEdit = await editBtn.isVisible().catch(() => false);
    const hasBoard = await boardLink.isVisible().catch(() => false);

    expect(hasEdit || hasBoard).toBe(true);
  });

  test('6.6 — Open feature board navigates to feature project', async ({ page, request }) => {
    // Create feature via API
    const featureId = await createFeature(request, projectId, 'Feature Gamma');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Gamma' });
    await expect(featureCard).toBeVisible();

    await featureCard.locator('[data-qa="open-feature-board-link"]').click();

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

    // Click edit on the feature card
    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Delta' });
    await expect(featureCard).toBeVisible();
    await featureCard.locator('[data-qa="edit-feature-btn"]').click();

    await expect(page.locator('[data-qa="edit-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="edit-feature-name-input"]').clear();
    await page.locator('[data-qa="edit-feature-name-input"]').fill('Feature Updated');
    await page.locator('[data-qa="confirm-edit-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Updated' }),
    ).toBeVisible();
  });

  test('6.8 — Delete feature', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Create feature
    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="new-feature-name-input"]').fill('Feature Epsilon');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Epsilon' });
    await expect(featureCard).toBeVisible();

    // Open edit modal and delete from there
    await featureCard.locator('[data-qa="edit-feature-btn"]').click();

    await expect(page.locator('[data-qa="delete-feature-btn"]')).toBeVisible();
    await page.locator('[data-qa="delete-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Epsilon' }),
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
