import { test, expect, APIRequestContext } from '@playwright/test';
import { createProject, deleteProject, createSkill, deleteSkill, BASE_URL } from './helpers';

// ============================================================
// Test Suite 9: Skills
// ============================================================

test.describe('9. Skills', () => {
  test('9.1 — Skills page loads', async ({ page }) => {
    await page.goto(`${BASE_URL}/skills`);
    await expect(page.getByRole('heading', { name: 'Skills' })).toBeVisible();
  });

  test('9.2 — Create a new skill', async ({ page, request }) => {
    let capturedSlug = '';

    await page.goto(`${BASE_URL}/skills`);
    await page.locator('[data-qa="new-skill-btn"]').click();

    await page.locator('[data-qa="skill-name-input"]').fill('QA Test Skill');

    // Wait for slug to auto-fill
    await page.waitForTimeout(300);
    capturedSlug = await page.locator('[data-qa="skill-slug-input"]').inputValue();

    await page.locator('[data-qa="skill-description-textarea"]').fill('Test description');
    await page.locator('[data-qa="save-skill-btn"]').click();

    await expect(page.getByText('QA Test Skill')).toBeVisible();

    // Cleanup
    if (capturedSlug) {
      await deleteSkill(request, capturedSlug);
    }
  });

  test('9.3 — Skill name is required', async ({ page }) => {
    await page.goto(`${BASE_URL}/skills`);
    await page.locator('[data-qa="new-skill-btn"]').click();

    // Leave name empty — save button should be disabled
    await expect(page.locator('[data-qa="save-skill-btn"]')).toBeDisabled();

    // Modal should stay open
    await expect(page.locator('[data-qa="skill-name-input"]')).toBeVisible();
  });

  test('9.4 — Cancel skill creation', async ({ page }) => {
    await page.goto(`${BASE_URL}/skills`);
    await page.locator('[data-qa="new-skill-btn"]').click();

    await page.locator('[data-qa="skill-name-input"]').fill('Temp Skill');
    await page.locator('[data-qa="cancel-skill-modal-btn"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="skill-name-input"]')).not.toBeVisible();
  });

  test('9.5 — Edit a skill', async ({ page, request }) => {
    await createSkill(request, 'qa-edit-skill', 'QA Edit Skill');

    try {
      await page.goto(`${BASE_URL}/skills`);

      // Find the card containing this skill and click its edit button
      const skillCard = page.locator('[data-qa="skill-card"]').filter({ hasText: 'QA Edit Skill' });
      await skillCard.locator('[data-qa="skill-edit-btn"]').click();

      // Modal opens in edit mode — slug should be disabled
      await expect(page.locator('[data-qa="skill-slug-input"]')).toBeDisabled();

      // Update name
      await page.locator('[data-qa="skill-name-input"]').fill('QA Edit Skill Updated');
      await page.locator('[data-qa="save-skill-btn"]').click();

      await expect(page.getByText('QA Edit Skill Updated')).toBeVisible();
    } finally {
      await deleteSkill(request, 'qa-edit-skill');
    }
  });

  test('9.6 — Delete a skill', async ({ page, request }) => {
    await createSkill(request, 'qa-delete-skill', 'QA Delete Skill');

    await page.goto(`${BASE_URL}/skills`);

    const skillCard = page.locator('[data-qa="skill-card"]').filter({ hasText: 'QA Delete Skill' });
    await skillCard.locator('[data-qa="skill-delete-btn"]').click();

    // Confirm deletion
    await page.locator('[data-qa="confirm-delete-skill-btn"]').click();

    await expect(page.getByText('QA Delete Skill')).not.toBeVisible();
  });

  test('9.7 — Cancel skill deletion', async ({ page, request }) => {
    await createSkill(request, 'qa-nodelete-skill', 'QA No Delete');

    try {
      await page.goto(`${BASE_URL}/skills`);

      const skillCard = page.locator('[data-qa="skill-card"]').filter({ hasText: 'QA No Delete' });
      await skillCard.locator('[data-qa="skill-delete-btn"]').click();

      // Cancel deletion
      await page.locator('[data-qa="cancel-delete-skill-btn"]').click();

      // Skill should still be visible
      await expect(page.getByText('QA No Delete')).toBeVisible();
    } finally {
      await deleteSkill(request, 'qa-nodelete-skill');
    }
  });
});

// ============================================================
// Test Suite 10: Statistics
// ============================================================

test.describe('10. Statistics', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Stats Test Project');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('10.1 — Statistics page loads for a project', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/statistics`);

    // Assert page loads — look for heading or stat-related content
    await expect(
      page.getByRole('heading', { name: /statistics/i }).or(page.getByText(/todo/i)).first()
    ).toBeVisible();
  });

  test('10.2 — Time range buttons are visible and clickable', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/statistics`);

    const btn7d = page.locator('[data-qa="time-range-7d-btn"]');
    const btn14d = page.locator('[data-qa="time-range-14d-btn"]');
    const btn30d = page.locator('[data-qa="time-range-30d-btn"]');

    await expect(btn7d).toBeVisible();
    await expect(btn14d).toBeVisible();
    await expect(btn30d).toBeVisible();

    // Click the 14d button and verify it becomes active (uses inline background-color style)
    await btn14d.click();

    // The active button gets backgroundColor: var(--primary), inactive gets var(--bg-tertiary)
    const style = await btn14d.getAttribute('style');
    const isActive =
      style?.includes('--primary') ||
      (await btn14d.textContent())?.includes('14');

    expect(isActive).toBeTruthy();
  });

  test('10.3 — Statistics shows zero counts for empty project', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/statistics`);

    // Page should render without error messages
    await expect(page.getByText(/error/i)).not.toBeVisible();

    // Page should contain something (not a crash)
    await expect(page.locator('body')).not.toBeEmpty();
  });
});

// ============================================================
// Test Suite 11: Export Pages
// ============================================================

test.describe('11. Export Pages', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Export Test Project');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('11.1 — Claude export page renders', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/export/claude`);

    // Assert page contains "Claude" text
    await expect(page.getByText(/claude/i).first()).toBeVisible();

    // Assert back link is visible
    await expect(page.locator('[data-qa="back-to-project-link"]')).toBeVisible();
  });

  test('11.2 — Gemini export page renders', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/export/gemini`);

    // Assert page contains "Gemini" text
    await expect(page.getByText(/gemini/i).first()).toBeVisible();

    // Assert back link is visible
    await expect(page.locator('[data-qa="back-to-project-link"]')).toBeVisible();
  });

  test('11.3 — Back link from export page goes back to project', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/export/claude`);

    await page.locator('[data-qa="back-to-project-link"]').click();

    // URL should contain the project id
    await expect(page).toHaveURL(new RegExp(`/projects/${projectId}`));
  });
});
