import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createTask, moveTask, BASE_URL } from './helpers';

test.describe('4. Task Management', () => {
  let projectId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Task Management Test Project');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('4.1 — Create task via New Task modal (happy path)', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-title-input"]').fill('My New Task');
    await page.locator('[data-qa="new-task-summary-input"]').fill('This is the task summary');
    await page.locator('[data-qa="new-task-submit-btn"]').click();

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'My New Task' })).toBeVisible();
  });

  test('4.2 — Create task validation: title required', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-summary-input"]').fill('Some summary');
    // Submit button should be disabled when title is empty
    await expect(page.locator('[data-qa="new-task-submit-btn"]')).toBeDisabled();

    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();
  });

  test('4.3 — Create task validation: summary required', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-title-input"]').fill('Some title');
    // Submit button should be disabled when summary is empty
    await expect(page.locator('[data-qa="new-task-submit-btn"]')).toBeDisabled();

    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();
  });

  test('4.4 — Cancel new task modal', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-cancel-btn"]').click();

    await expect(page.locator('[data-qa="new-task-title-input"]')).not.toBeVisible();
  });

  test('4.5 — Click task card opens task drawer', async ({ request, page }) => {
    await createTask(request, projectId, 'Drawer Test Task', 'Summary for drawer test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();

    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });

  test('4.6 — Edit task title inline in drawer', async ({ request, page }) => {
    await createTask(request, projectId, 'Original Title', 'Summary for title edit test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-title-edit-btn"]').click();
    await page.locator('[data-qa="task-title-input"]').clear();
    await page.locator('[data-qa="task-title-input"]').fill('Updated Title');
    await page.locator('[data-qa="task-title-input"]').press('Enter');

    await expect(page.locator('[data-qa="task-title-input"]')).not.toBeVisible();
    await expect(page.getByText('Updated Title')).toBeVisible();
  });

  test('4.7 — Edit task summary in drawer', async ({ request, page }) => {
    await createTask(request, projectId, 'Summary Edit Task', 'Original summary text');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-summary-edit-btn"]').click();
    await page.locator('[data-qa="task-summary-input"]').clear();
    await page.locator('[data-qa="task-summary-input"]').fill('Updated summary text');
    await page.locator('[data-qa="task-summary-save-btn"]').click();

    await expect(page.getByText('Updated summary text')).toBeVisible();
  });

  test('4.8 — Change task priority in drawer', async ({ request, page }) => {
    await createTask(request, projectId, 'Priority Change Task', 'Summary for priority test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-priority-btn"]').click();
    await page.locator('[data-qa="task-priority-option-critical"]').click();

    await expect(page.getByText(/critical/i)).toBeVisible();
  });

  test('4.9 — Delete task via context menu', async ({ request, page }) => {
    await createTask(request, projectId, 'Task To Delete', 'Summary for delete test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').first();
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-delete-btn"]').click();
    await page.locator('[data-qa="delete-task-confirm-btn"]').click();

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Delete' })).not.toBeVisible();
  });

  test('4.10 — Block task via context menu', async ({ request, page }) => {
    await createTask(request, projectId, 'Task To Block', 'Summary for block test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').first();
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-block-btn"]').click();
    await page.locator('[data-qa="block-reason-input"]').fill('Blocked for test');
    await page.locator('[data-qa="block-task-submit-btn"]').click();

    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Block' })).not.toBeVisible();
  });

  test('4.11 — Complete task via context menu', async ({ request, page }) => {
    const taskId = await createTask(request, projectId, 'Task To Complete', 'Summary for complete test');
    await moveTask(request, projectId, taskId, 'in_progress');
    await page.reload();

    const inProgressColumn = page.locator('[data-qa="column"]').filter({ hasText: 'In Progress' });
    const taskCard = inProgressColumn.locator('[data-qa="task-card"]').first();
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-complete-btn"]').click();
    await page.locator('[data-qa="complete-summary-input"]').fill('Completed for test');
    await page.locator('[data-qa="complete-task-submit-btn"]').click();

    await expect(inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Complete' })).not.toBeVisible();
  });

  test('4.12 — Cancel delete modal', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Not Deleted', 'Summary for cancel delete test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').first();
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-delete-btn"]').click();
    await page.locator('[data-qa="delete-task-cancel-btn"]').click();

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Deleted' })).toBeVisible();
  });

  test('4.13 — Move task to In Progress via context menu', async ({ request, page }) => {
    await createTask(request, projectId, 'Task To Move', 'Summary for move test');
    await page.reload();

    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    const taskCard = todoColumn.locator('[data-qa="task-card"]').first();
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-move_in_progress-btn"]').click();

    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Move' })).not.toBeVisible();
  });
});
