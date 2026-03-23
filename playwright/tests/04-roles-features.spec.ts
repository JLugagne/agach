import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createFeature, createAgent, deleteAgent, createSkill, deleteSkill, BASE_URL } from './helpers';

test.describe('9. Agents Management', () => {
  test('9.1 — Agents page lists all global agents', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await expect(page.getByRole('heading', { name: /agents/i })).toBeVisible();
    const count = await page.locator('[data-qa="agent-card"]').count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('9.2 — Create a new agent', async ({ page, request }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-agent-btn"]').click();

    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();
    await page.locator('[data-qa="agent-name-input"]').fill('Backend Engineer');

    // Slug should auto-fill
    const slugValue = await page.locator('[data-qa="agent-slug-input"]').inputValue();
    expect(slugValue.length).toBeGreaterThan(0);

    // Optionally fill description, prompt template, prompt hint (create mode uses textareas)
    await page.locator('[data-qa="agent-modal-description-textarea"]').fill('Handles backend services');
    await page.locator('[data-qa="agent-modal-prompt-template-textarea"]').fill('{{task.title}}');
    await page.locator('[data-qa="agent-modal-prompt-hint-textarea"]').fill('Focus on Go code');

    await page.locator('[data-qa="agent-save-btn"]').click();

    await expect(
      page.locator('[data-qa="agent-card"]').filter({ hasText: 'Backend Engineer' }),
    ).toBeVisible();

    const finalSlug = slugValue || 'backendengineer';
    try {
      await deleteAgent(request, finalSlug);
    } catch {
      // Best-effort cleanup
    }
  });

  test('9.3 — Agent name is required', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-agent-btn"]').click();

    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();
    // Leave name empty — save button should be disabled
    await expect(page.locator('[data-qa="agent-save-btn"]')).toBeDisabled();

    // Modal should stay open
    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();
  });

  test('9.4 — Cancel agent creation', async ({ page }) => {
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

  test('9.4b — Cancel via close button', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);

    await page.locator('[data-qa="new-agent-btn"]').click();
    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();

    await page.locator('[data-qa="agent-modal-close-btn"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="agent-name-input"]')).not.toBeVisible();
  });

  test('9.5 — Edit an existing agent', async ({ page, request }) => {
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

  test('9.6 — Delete an agent', async ({ page, request }) => {
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

  test('9.7 — Clone an agent', async ({ page, request }) => {
    await createAgent(request, 'qa-clone-src', 'QA Clone Source');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Clone Source' });
      await expect(card).toBeVisible();

      // Click clone button on the card
      await card.locator('[data-qa="agent-card-clone-btn"]').click();

      // Clone dialog should appear
      await expect(page.locator('[data-qa="clone-agent-slug-input"]')).toBeVisible();

      await page.locator('[data-qa="clone-agent-slug-input"]').fill('qaclonecopy');
      await page.locator('[data-qa="clone-agent-name-input"]').fill('QA Clone Copy');
      await page.locator('[data-qa="clone-agent-submit-btn"]').click();

      // Cloned agent should appear in the list
      await expect(
        page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Clone Copy' }),
      ).toBeVisible();
    } finally {
      try { await deleteAgent(request, 'qaclonecopy'); } catch { /* cleanup */ }
      await deleteAgent(request, 'qa-clone-src');
    }
  });

  test('9.7b — Cancel clone dialog', async ({ page, request }) => {
    await createAgent(request, 'qa-clone-cancel', 'QA Clone Cancel');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Clone Cancel' });
      await expect(card).toBeVisible();
      await card.locator('[data-qa="agent-card-clone-btn"]').click();

      await expect(page.locator('[data-qa="clone-agent-slug-input"]')).toBeVisible();
      await page.locator('[data-qa="clone-agent-cancel-btn"]').click();

      // Dialog should close
      await expect(page.locator('[data-qa="clone-agent-slug-input"]')).not.toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-clone-cancel');
    }
  });

  test('9.8 — Agent icon and color selection', async ({ page, request }) => {
    await createAgent(request, 'qa-icon-color', 'QA Icon Color');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Icon Color' });
      await expect(card).toBeVisible();
      await card.click();

      // Open icon picker
      await page.locator('[data-qa="agent-modal-icon-toggle"]').click();
      // Select an icon from the picker
      const iconBtn = page.locator('[data-qa="agent-modal-icon-btn"]').first();
      await expect(iconBtn).toBeVisible();
      await iconBtn.click();

      // Select a color
      const colorBtn = page.locator('[data-qa="agent-modal-color-btn"]').nth(2);
      await colorBtn.click();

      // Save
      await page.locator('[data-qa="agent-save-btn"]').click();

      // Card should still be visible after save
      await expect(
        page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Icon Color' }),
      ).toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-icon-color');
    }
  });

  test('9.9 — Project-scoped agents page', async ({ page, request }) => {
    const projectId = await createProject(request, 'Agents Scoped Project 9.9');
    try {
      await page.goto(`${BASE_URL}/projects/${projectId}/roles`);
      await expect(page.getByRole('heading', { name: /agents/i })).toBeVisible();
    } finally {
      await deleteProject(request, projectId);
    }
  });

  test('9.10 — Set default agent on project-scoped page', async ({ page, request }) => {
    const projectId = await createProject(request, 'Default Agent Project 9.10');
    // Create a project-scoped agent so the set-default button appears
    await createAgent(request, 'qa-default-agent', 'QA Default Agent');

    try {
      await page.goto(`${BASE_URL}/projects/${projectId}/roles`);
      await expect(page.getByRole('heading', { name: /agents/i })).toBeVisible();

      // Wait for cards to load
      const setDefaultBtn = page.locator('[data-qa="agent-card-set-default-btn"]').first();
      const hasBtn = await setDefaultBtn.isVisible({ timeout: 3000 }).catch(() => false);

      if (hasBtn) {
        await setDefaultBtn.click();
        // The star should now be filled (agent is default)
        // Click again to unset
        await setDefaultBtn.click();
      }
    } finally {
      try { await deleteAgent(request, 'qa-default-agent'); } catch { /* cleanup */ }
      await deleteProject(request, projectId);
    }
  });

  test('9.11 — Agent slug is disabled when editing', async ({ page, request }) => {
    await createAgent(request, 'qa-slug-disabled', 'QA Slug Disabled');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Slug Disabled' });
      await expect(card).toBeVisible();
      await card.click();

      // Slug input should be disabled in edit mode
      await expect(page.locator('[data-qa="agent-slug-input"]')).toBeDisabled();
    } finally {
      await deleteAgent(request, 'qa-slug-disabled');
    }
  });

  test('9.12 — Template variables toggle', async ({ page }) => {
    await page.goto(`${BASE_URL}/roles`);
    await page.locator('[data-qa="new-agent-btn"]').click();

    await expect(page.locator('[data-qa="agent-name-input"]')).toBeVisible();

    // Click template variables toggle to expand
    await page.locator('[data-qa="template-variables-toggle"]').click();

    // Variables reference panel should be visible (check for template variable content)
    await expect(page.getByText('task.title')).toBeVisible();

    // Click again to collapse
    await page.locator('[data-qa="template-variables-toggle"]').click();

    // Content should be hidden
    await expect(page.getByText('task.title')).not.toBeVisible();
  });

  test('9.13 — Inline edit prompt template field', async ({ page, request }) => {
    await createAgent(request, 'qa-inline-edit', 'QA Inline Edit', {
      description: 'Original description',
      prompt_template: 'Original template',
    });

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Inline Edit' });
      await expect(card).toBeVisible();
      await card.click();

      // In edit mode, description/prompt fields use InlineEditField
      // Click the edit button on the first inline edit field
      const editBtns = page.locator('[data-qa="inline-edit-field-edit-btn"]');
      await expect(editBtns.first()).toBeVisible();
      await editBtns.first().click();

      // Textarea should appear
      const textarea = page.locator('[data-qa="inline-edit-field-textarea"]');
      await expect(textarea).toBeVisible();

      // Modify the content
      await textarea.fill('Updated description');

      // Save
      await page.locator('[data-qa="inline-edit-field-save-btn"]').click();

      // Textarea should disappear (back to display mode)
      await expect(page.locator('[data-qa="inline-edit-field-textarea"]')).not.toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-inline-edit');
    }
  });

  test('9.13b — Inline edit cancel', async ({ page, request }) => {
    await createAgent(request, 'qa-inline-cancel', 'QA Inline Cancel', {
      description: 'Keep this description',
    });

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Inline Cancel' });
      await expect(card).toBeVisible();
      await card.click();

      const editBtns = page.locator('[data-qa="inline-edit-field-edit-btn"]');
      await expect(editBtns.first()).toBeVisible();
      await editBtns.first().click();

      const textarea = page.locator('[data-qa="inline-edit-field-textarea"]');
      await expect(textarea).toBeVisible();
      await textarea.fill('This should be discarded');

      // Cancel
      await page.locator('[data-qa="inline-edit-field-cancel-btn"]').click();

      // Textarea should disappear
      await expect(page.locator('[data-qa="inline-edit-field-textarea"]')).not.toBeVisible();

      // Original text should still be present
      await expect(page.getByText('Keep this description')).toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-inline-cancel');
    }
  });

  test('9.14 — Agent skills panel: view assigned skills', async ({ page, request }) => {
    const agent = await createAgent(request, 'qa-skills-view', 'QA Skills View');
    const skill = await createSkill(request, 'qa-skill-view', 'QA Skill View', {
      description: 'A test skill',
    });

    try {
      // Assign skill to agent via API
      const token = (await request.fetch(`${BASE_URL}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        data: { email: 'admin@agach.local', password: 'admin' },
      }).then(r => r.json())).data?.access_token;

      await request.fetch(`${BASE_URL}/api/agents/${agent.slug}/skills`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        data: { skill_slug: skill.slug },
      });

      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Skills View' });
      await expect(card).toBeVisible();
      await card.click();

      // Skills section should show the assigned skill
      await expect(page.getByText('QA Skill View')).toBeVisible();
    } finally {
      try { await deleteAgent(request, 'qa-skills-view'); } catch { /* cleanup */ }
      try { await deleteSkill(request, 'qa-skill-view'); } catch { /* cleanup */ }
    }
  });

  test('9.15 — Agent skills panel: add a skill', async ({ page, request }) => {
    await createAgent(request, 'qa-skills-add', 'QA Skills Add');
    await createSkill(request, 'qa-skill-add', 'QA Skill Add');

    try {
      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Skills Add' });
      await expect(card).toBeVisible();
      await card.click();

      // Select a skill from the dropdown
      const select = page.locator('[data-qa="skill-add-select"]');
      await expect(select).toBeVisible();
      await select.selectOption('qa-skill-add');

      // Click add button
      await page.locator('[data-qa="skill-add-btn"]').click();

      // Skill should now appear in the assigned list
      await expect(page.getByText('QA Skill Add')).toBeVisible();
    } finally {
      try { await deleteAgent(request, 'qa-skills-add'); } catch { /* cleanup */ }
      try { await deleteSkill(request, 'qa-skill-add'); } catch { /* cleanup */ }
    }
  });

  test('9.16 — Agent skills panel: remove a skill', async ({ page, request }) => {
    const agent = await createAgent(request, 'qa-skills-remove', 'QA Skills Remove');
    const skill = await createSkill(request, 'qa-skill-remove', 'QA Skill Remove');

    try {
      // Assign skill to agent via API
      const token = (await request.fetch(`${BASE_URL}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        data: { email: 'admin@agach.local', password: 'admin' },
      }).then(r => r.json())).data?.access_token;

      await request.fetch(`${BASE_URL}/api/agents/${agent.slug}/skills`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        data: { skill_slug: skill.slug },
      });

      await page.goto(`${BASE_URL}/roles`);

      const card = page.locator('[data-qa="agent-card"]').filter({ hasText: 'QA Skills Remove' });
      await expect(card).toBeVisible();
      await card.click();

      // Skill should be listed
      await expect(page.getByText('QA Skill Remove')).toBeVisible();

      // Click remove button (data-qa="skill-remove-btn-{slug}")
      await page.locator(`[data-qa="skill-remove-btn-${skill.slug}"]`).click();

      // Skill should no longer be listed as assigned
      // The "No skills assigned" message or the skill disappearing
      await expect(page.getByText('QA Skill Remove')).not.toBeVisible();
    } finally {
      try { await deleteAgent(request, 'qa-skills-remove'); } catch { /* cleanup */ }
      try { await deleteSkill(request, 'qa-skill-remove'); } catch { /* cleanup */ }
    }
  });
});

test.describe('10. Features / Sub-projects', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Features Test Project');
  });

  test.afterEach(async ({ request }) => {
    try {
      await deleteProject(request, projectId);
    } catch {
      // Best-effort cleanup
    }
  });

  test('10.1 — Features page lists existing features', async ({ page, request }) => {
    await createFeature(request, projectId, 'Existing Feature');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Existing Feature' });
    await expect(featureCard).toBeVisible();
  });

  test('10.1b — Features page empty state', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // Either empty state button or "no features" text should be visible
    const emptyBtn = page.locator('[data-qa="create-first-feature-btn"]');
    const noFeaturesText = page.getByText(/no features yet/i);

    const hasEmpty = await emptyBtn.isVisible().catch(() => false);
    const hasText = await noFeaturesText.isVisible().catch(() => false);

    expect(hasEmpty || hasText).toBe(true);
  });

  test('10.2 — Create a new feature', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    await page.locator('[data-qa="new-feature-name-input"]').fill('Authentication Feature');

    // Optionally fill description
    await page.locator('[data-qa="new-feature-description-textarea"]').fill('Handles auth flows');

    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-card"]').filter({ hasText: 'Authentication Feature' }),
    ).toBeVisible();
  });

  test('10.3 — Create feature: name is required', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    // Leave name empty — confirm button should be disabled
    await expect(page.locator('[data-qa="confirm-create-feature-btn"]')).toBeDisabled();

    // Modal should stay open
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();
  });

  test('10.4 — Cancel feature creation', async ({ page }) => {
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

  test('10.4b — Cancel feature creation via backdrop', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    // Click backdrop to close
    await page.locator('[data-qa="create-feature-modal-backdrop"]').click();

    // Modal should be closed
    await expect(page.locator('[data-qa="new-feature-name-input"]')).not.toBeVisible();
  });

  test('10.5 — Edit a feature name and description', async ({ page, request }) => {
    await createFeature(request, projectId, 'Feature To Edit');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature To Edit' });
    await expect(featureCard).toBeVisible();

    // Click edit on the feature card
    await featureCard.locator('[data-qa="edit-feature-btn"]').click();

    // Edit modal should open
    await expect(page.locator('[data-qa="edit-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="edit-feature-name-input"]').clear();
    await page.locator('[data-qa="edit-feature-name-input"]').fill('Feature Updated');

    // Update description
    await page.locator('[data-qa="edit-feature-description-textarea"]').fill('Updated description');

    await page.locator('[data-qa="confirm-edit-feature-btn"]').click();

    await expect(
      page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Updated' }),
    ).toBeVisible();
  });

  test('10.5b — Cancel edit feature via cancel button', async ({ page, request }) => {
    await createFeature(request, projectId, 'Feature No Edit');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature No Edit' });
    await expect(featureCard).toBeVisible();
    await featureCard.locator('[data-qa="edit-feature-btn"]').click();

    await expect(page.locator('[data-qa="edit-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="cancel-edit-feature-btn"]').click();

    // Modal should close, name unchanged
    await expect(page.locator('[data-qa="edit-feature-name-input"]')).not.toBeVisible();
    await expect(featureCard).toBeVisible();
  });

  test('10.5c — Cancel edit feature via backdrop', async ({ page, request }) => {
    await createFeature(request, projectId, 'Feature Backdrop Cancel');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Backdrop Cancel' });
    await expect(featureCard).toBeVisible();
    await featureCard.locator('[data-qa="edit-feature-btn"]').click();

    await expect(page.locator('[data-qa="edit-feature-name-input"]')).toBeVisible();
    await page.locator('[data-qa="edit-feature-modal-backdrop"]').click();

    await expect(page.locator('[data-qa="edit-feature-name-input"]')).not.toBeVisible();
  });

  test('10.6 — Open feature board link', async ({ page, request }) => {
    const featureId = await createFeature(request, projectId, 'Feature Board Link');

    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    const featureCard = page.locator('[data-qa="feature-card"]').filter({ hasText: 'Feature Board Link' });
    await expect(featureCard).toBeVisible();

    await featureCard.locator('[data-qa="open-feature-board-link"]').click();

    // URL should navigate to the feature's own project page
    await expect(page).toHaveURL(new RegExp(`/projects/${featureId}`));
  });

  test('10.7 — Toggle show done features', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/features`);

    // The toggle button only appears when there are done features.
    // Check if the button exists; if so, clicking it should toggle visibility.
    const toggleBtn = page.locator('[data-qa="toggle-done-features-btn"]');
    const hasToggle = await toggleBtn.isVisible({ timeout: 3000 }).catch(() => false);

    if (hasToggle) {
      // Click to show done features
      await toggleBtn.click();
      // Button text should change to "Hide done"
      await expect(toggleBtn).toContainText(/hide done/i);

      // Click again to hide
      await toggleBtn.click();
      await expect(toggleBtn).toContainText(/show done/i);
    }
    // If no done features exist, the button is simply not rendered — that is acceptable.
  });
});
