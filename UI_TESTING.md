# UI Testing Scenarios

## Setup

- Base URL: `http://localhost:8322`
- All tests use `data-qa` attributes for element selection
- The server must be running with a clean database (or a seeded test database)
- Tests assume a Playwright project configured with `baseURL: process.env.BASE_URL ?? 'http://localhost:8322'`
- Authentication: tests must log in as `admin@agach.local` / `admin` to obtain a bearer token

### Playwright config essentials

```ts
// playwright.config.ts
import { defineConfig } from '@playwright/test';
export default defineConfig({
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:8322',
    trace: 'on-first-retry',
  },
});
```

---

## Test Suites

### 1. Login & Authentication

#### 1.1 — Login page renders

- **Pre-conditions**: Server is running.
- **Steps**:
  1. Navigate to `/login`
- **Expected**: Email input `[data-qa="login-email-input"]`, password input `[data-qa="login-password-input"]`, and submit button `[data-qa="login-submit-btn"]` are visible.

#### 1.2 — Successful login redirects to home

- **Pre-conditions**: A valid user exists (admin@agach.local / admin).
- **Steps**:
  1. Navigate to `/login`
  2. Fill `[data-qa="login-email-input"]` with `"admin@agach.local"`
  3. Fill `[data-qa="login-password-input"]` with `"admin"`
  4. Click `[data-qa="login-submit-btn"]`
- **Expected**: URL changes to `/`; home page is rendered.

#### 1.3 — Invalid credentials show error

- **Pre-conditions**: On the login page.
- **Steps**:
  1. Fill email and password with invalid credentials
  2. Click `[data-qa="login-submit-btn"]`
- **Expected**: An error message is displayed; user stays on the login page.

#### 1.4 — Toggle password visibility

- **Pre-conditions**: On the login page.
- **Steps**:
  1. Click `[data-qa="toggle-password-visibility-btn"]`
- **Expected**: Password input type toggles between "password" and "text".

---

### 2. Home Page — Projects List

#### 2.1 — Render projects list on load

- **Pre-conditions**: Server is running; at least one project exists in the database.
- **Steps**:
  1. Navigate to `/`
  2. Wait for `[data-qa="project-card"]` elements to be visible
- **Expected**: One or more project cards are displayed with project names.

#### 2.2 — Display empty state when no projects exist

- **Pre-conditions**: Server is running; database has no projects.
- **Steps**:
  1. Navigate to `/`
  2. Wait for page to load
- **Expected**: An empty state with `[data-qa="create-project-empty-btn"]` is visible.

#### 2.3 — Create a new project via the create button

- **Pre-conditions**: Home page is loaded.
- **Steps**:
  1. Navigate to `/`
  2. Click `[data-qa="create-project-btn"]`
  3. Fill `[data-qa="create-project-name-input"]` with `"Test Project"`
  4. Optionally fill `[data-qa="create-project-description-input"]` with `"A test project description"`
  5. Optionally fill `[data-qa="create-project-giturl-input"]` with a Git URL
  6. Optionally select a dockerfile from `[data-qa="create-project-dockerfile-select"]`
  7. Optionally toggle agent checkboxes `[data-qa="create-project-role-{slug}"]`
  8. Click `[data-qa="create-project-submit-btn"]`
- **Expected**: The dialog closes and a new project card with the name "Test Project" appears in the list.

#### 2.4 — Create project: cancel closes dialog

- **Pre-conditions**: Create project dialog is open.
- **Steps**:
  1. Click `[data-qa="create-project-cancel-btn"]` or `[data-qa="create-project-close-btn"]`
- **Expected**: Dialog closes; no project is created.

#### 2.5 — Click on a project card navigates to the kanban board

- **Pre-conditions**: At least one project exists.
- **Steps**:
  1. Navigate to `/`
  2. Click `[data-qa="project-open-btn"]` on the desired project card
- **Expected**: URL changes to `/projects/:projectId` and the kanban board is rendered.

#### 2.6 — Project card shows task summary counts

- **Pre-conditions**: A project with tasks in various columns exists.
- **Steps**:
  1. Navigate to `/`
  2. Inspect `[data-qa="project-card"]` for task counts
- **Expected**: The card displays todo, in-progress, done, and/or blocked counts.

---

### 3. Navigation

#### 3.1 — Sidebar navigation links are visible on project pages

- **Pre-conditions**: A project exists; user is on the kanban board.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Check visibility of `[data-qa="nav-kanban-btn"]`, `[data-qa="nav-features-btn"]`, `[data-qa="nav-statistics-btn"]`, `[data-qa="nav-settings-btn"]`
- **Expected**: All sidebar navigation links are visible.

#### 3.2 — Navigate to Features via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-features-btn"]`
- **Expected**: URL changes to `/projects/:projectId/features` and the Features page is displayed.

#### 3.3 — Navigate to Statistics via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-statistics-btn"]`
- **Expected**: URL changes to `/projects/:projectId/statistics` and the Statistics page heading is visible.

#### 3.4 — Navigate to Settings via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-settings-btn"]`
- **Expected**: URL changes to `/projects/:projectId/settings` and the Settings page is visible.

#### 3.5 — Navigate to global Agents page

- **Pre-conditions**: App is loaded (no project selected).
- **Steps**:
  1. Click `[data-qa="nav-roles-btn"]` in the global sidebar
- **Expected**: URL changes to `/roles` and the Agents page is displayed.

#### 3.6 — Navigate to global Skills page

- **Pre-conditions**: App is loaded.
- **Steps**:
  1. Click `[data-qa="nav-skills-btn"]` in the global sidebar
- **Expected**: URL changes to `/skills` and the Skills page heading is visible.

#### 3.7 — Navigate to global Dockerfiles page

- **Pre-conditions**: App is loaded.
- **Steps**:
  1. Click `[data-qa="nav-dockerfiles-btn"]` in the global sidebar
- **Expected**: URL changes to `/dockerfiles` and the Dockerfiles page is displayed.

#### 3.8 — Home (logo) navigates to project list

- **Pre-conditions**: User is on a project subpage.
- **Steps**:
  1. Click `[data-qa="logo-home-link"]`
- **Expected**: URL changes to `/` and the home page is rendered.

#### 3.9 — User menu: navigate to Account

- **Pre-conditions**: App is loaded.
- **Steps**:
  1. Click `[data-qa="user-menu-btn"]` (if dropdown) or observe user menu area
  2. Click `[data-qa="user-menu-account-btn"]`
- **Expected**: URL changes to `/account`.

#### 3.10 — User menu: navigate to API Keys

- **Pre-conditions**: App is loaded.
- **Steps**:
  1. Click `[data-qa="user-menu-api-keys-btn"]`
- **Expected**: URL changes to `/account/api-keys`.

#### 3.11 — User menu: sign out

- **Pre-conditions**: User is logged in.
- **Steps**:
  1. Click `[data-qa="user-menu-logout-btn"]`
- **Expected**: User is logged out and redirected to the login page.

#### 3.12 — Sidebar shows features list with add button

- **Pre-conditions**: A project with features exists; on a project page.
- **Steps**:
  1. Observe sidebar for `[data-qa="nav-feature-btn"]` items
  2. Observe `[data-qa="nav-add-feature-btn"]`
- **Expected**: Feature links and "Add Feature" button are visible in the sidebar.

---

### 4. Kanban Board

#### 4.1 — Board displays four columns

- **Pre-conditions**: On a project kanban page.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Wait for `[data-qa="column"]` elements to load
- **Expected**: Four columns are visible with `[data-qa="column-title"]`: **Todo**, **In Progress**, **Done**, **Blocked**.

#### 4.2 — Task cards appear in the correct column

- **Pre-conditions**: A project with at least one task in the "todo" column.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Locate the Todo column
  3. Check for `[data-qa="task-card"]` inside the Todo column
- **Expected**: Task cards are visible in the Todo column.

#### 4.3 — Column displays task count badge

- **Pre-conditions**: A column with at least 2 tasks.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Observe the count badge in each column header
- **Expected**: The count badge reflects the number of tasks in the column.

#### 4.4 — WIP limit warning appears when column is at capacity

- **Pre-conditions**: The "In Progress" column has a WIP limit of 3 and exactly 3 tasks.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Observe the In Progress column header
- **Expected**: The count displays as `3/3`, highlighted in red/warning color.

#### 4.5 — Drag-and-drop task to reorder within a column

- **Pre-conditions**: A column with at least 2 tasks.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Hover over a task card to reveal the drag handle `[data-qa="task-card-drag-handle"]`
  3. Drag the first task card below the second task card in the same column
- **Expected**: The order of the tasks updates visually and persists after a page refresh.

#### 4.6 — Click task card opens the Task Drawer

- **Pre-conditions**: At least one task exists on the board.
- **Steps**:
  1. Click on a `[data-qa="task-card"]`
- **Expected**: The Task Drawer slides in from the right, displaying the task title, summary, priority, and column status.

#### 4.7 — Right-click (context menu) on a task card

- **Pre-conditions**: At least one task exists in the Todo or In Progress column.
- **Steps**:
  1. Right-click on a `[data-qa="task-card"]`
- **Expected**: A context menu appears with action buttons `[data-qa="context-menu-{action}-btn"]` such as "complete", "block", "delete", "move_to_project", "duplicate", priority and role options.

#### 4.8 — Board search filters visible tasks

- **Pre-conditions**: Multiple tasks exist with distinct titles.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Click `[data-qa="search-input"]`
  3. Type a partial title of an existing task
- **Expected**: Only matching task cards remain visible; non-matching cards are hidden.

#### 4.9 — Clear search button resets filter

- **Pre-conditions**: Search input has text.
- **Steps**:
  1. Click `[data-qa="search-clear-btn"]`
- **Expected**: Search is cleared; all tasks are visible again.

#### 4.10 — "New Task" button opens the create task modal

- **Pre-conditions**: On the kanban board.
- **Steps**:
  1. Click `[data-qa="new-task-btn"]`
- **Expected**: The New Task modal `[data-qa="new-task-modal"]` appears.

#### 4.11 — Keyboard shortcut "/" focuses search

- **Pre-conditions**: On the kanban board, no modal/drawer open.
- **Steps**:
  1. Press `/`
- **Expected**: The `[data-qa="search-input"]` receives focus.

#### 4.12 — Done column filter dropdown

- **Pre-conditions**: On the kanban board.
- **Steps**:
  1. Locate `[data-qa="done-filter-select"]`
  2. Select a time range (e.g., "Last 24h")
- **Expected**: Only tasks completed within the selected time range appear in the Done column.

#### 4.13 — Toggle sub-projects visibility

- **Pre-conditions**: A project with sub-projects (features) exists.
- **Steps**:
  1. Click `[data-qa="kanban-toggle-subprojects-btn"]`
- **Expected**: Tasks from sub-projects are included/excluded from the board.

#### 4.14 — Role filter buttons

- **Pre-conditions**: Tasks assigned to different roles exist.
- **Steps**:
  1. Click a `[data-qa="kanban-role-filter-btn"]` for a specific role
- **Expected**: Only tasks assigned to that role are shown. Click `[data-qa="kanban-clear-filters-btn"]` to reset.

#### 4.15 — Won't-do-requested badge is shown on blocked tasks

- **Pre-conditions**: A task exists in the Blocked column with `wont_do_requested=1`.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Observe the Blocked column
- **Expected**: The task card shows a "Won't Do Requested" badge.

#### 4.16 — Parent project link

- **Pre-conditions**: Viewing a sub-project (feature) board.
- **Steps**:
  1. Observe `[data-qa="kanban-parent-project-link"]`
- **Expected**: A link to the parent project is visible; clicking it navigates to the parent board.

#### 4.17 — Bulk select tasks with Ctrl+click

- **Pre-conditions**: Multiple tasks exist on the board.
- **Steps**:
  1. Ctrl+click on two or more `[data-qa="task-card"]` elements
- **Expected**: Selected tasks are highlighted; a bulk actions bar appears at the bottom.

#### 4.18 — Bulk actions bar: move, block, delete

- **Pre-conditions**: Multiple tasks are selected.
- **Steps**:
  1. Observe bulk actions bar with buttons:
     - `[data-qa="bulk-move-in-progress-btn"]` (from todo)
     - `[data-qa="bulk-move-todo-btn"]` (from in_progress/done)
     - `[data-qa="bulk-block-btn"]` (from todo/in_progress)
     - `[data-qa="bulk-complete-btn"]` (from in_progress)
     - `[data-qa="bulk-unblock-btn"]` (from blocked)
     - `[data-qa="bulk-delete-btn"]` (always)
  2. Click a bulk action
- **Expected**: All selected tasks are affected; bulk bar disappears.

#### 4.19 — Bulk actions: cancel

- **Pre-conditions**: Bulk actions bar is visible.
- **Steps**:
  1. Click `[data-qa="bulk-cancel-btn"]`
- **Expected**: Selection is cleared; bulk bar disappears.

---

### 5. Task Creation (New Task Modal)

#### 5.1 — Create a task with only required fields

- **Pre-conditions**: On the kanban board.
- **Steps**:
  1. Click `[data-qa="new-task-btn"]`
  2. Fill `[data-qa="new-task-title-input"]` with `"My New Task"`
  3. Fill `[data-qa="new-task-summary-input"]` with `"Brief summary of the task"`
  4. Click `[data-qa="new-task-submit-btn"]`
- **Expected**: Modal closes; the new task card appears in the Todo column.

#### 5.2 — Create task validation: title is required

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Leave `[data-qa="new-task-title-input"]` empty
  2. Fill `[data-qa="new-task-summary-input"]` with text
  3. Click `[data-qa="new-task-submit-btn"]`
- **Expected**: Submit button is disabled or an error is shown; modal stays open.

#### 5.3 — Create task validation: summary is required

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Fill `[data-qa="new-task-title-input"]` with `"My Task"`
  2. Leave `[data-qa="new-task-summary-input"]` empty
  3. Click `[data-qa="new-task-submit-btn"]`
- **Expected**: Submit button is disabled or an error is shown; modal stays open.

#### 5.4 — Create task with all optional fields

- **Pre-conditions**: New Task modal is open; at least one agent and one feature exist.
- **Steps**:
  1. Fill title and summary
  2. Fill `[data-qa="new-task-description-input"]` with `"Detailed description"`
  3. Select priority `"critical"` from `[data-qa="new-task-priority-select"]`
  4. Select an agent from `[data-qa="new-task-role-select"]`
  5. Select a feature from `[data-qa="new-task-feature-select"]`
  6. Check `[data-qa="new-task-backlog-checkbox"]` to add to backlog
  7. Type `"frontend"` in `[data-qa="new-task-tag-input"]` and click `[data-qa="new-task-add-tag-btn"]`
  8. Type `"src/main.go"` in `[data-qa="new-task-file-input"]` and click `[data-qa="new-task-add-file-btn"]`
  9. Click `[data-qa="new-task-submit-btn"]`
- **Expected**: Task is created with correct priority, role, feature, tags, and context files.

#### 5.5 — Add to backlog checkbox places task in backlog instead of todo

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Fill title and summary
  2. Check `[data-qa="new-task-backlog-checkbox"]`
  3. Click `[data-qa="new-task-submit-btn"]`
- **Expected**: The task does NOT appear in the Todo column; it appears in the Backlog page.

#### 5.6 — Close New Task modal with cancel or close button

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Click `[data-qa="new-task-cancel-btn"]` or `[data-qa="new-task-close-btn"]`
- **Expected**: The modal closes without creating a task.

#### 5.7 — Remove tag from new task

- **Pre-conditions**: New Task modal has at least one tag added.
- **Steps**:
  1. Click `[data-qa="new-task-remove-tag-btn"]` next to a tag
- **Expected**: The tag is removed from the list.

#### 5.8 — Remove context file from new task

- **Pre-conditions**: New Task modal has at least one context file added.
- **Steps**:
  1. Click `[data-qa="new-task-remove-file-btn"]` next to a file
- **Expected**: The file is removed from the list.

#### 5.9 — Attach image to description

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Click `[data-qa="new-task-attach-image-btn"]`
- **Expected**: File picker opens accepting image files.

---

### 6. Task Drawer

#### 6.1 — View task details

- **Pre-conditions**: A task with title, summary, description, priority, and tags exists.
- **Steps**:
  1. Click on the task card
- **Expected**: Drawer shows: title, summary, description (Markdown), priority badge, column status, tags, context files, dependencies, comments.

#### 6.2 — Close drawer with X button

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="drawer-close-btn"]`
- **Expected**: The drawer closes.

#### 6.3 — Inline edit title

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="task-title-edit-btn"]`
  2. Clear `[data-qa="task-title-input"]` and type `"Updated Title"`
  3. Press Enter or blur the input
- **Expected**: The title updates in the drawer and on the task card.

#### 6.4 — Inline edit summary

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="task-summary-edit-btn"]`
  2. Update `[data-qa="task-summary-input"]`
  3. Click `[data-qa="task-summary-save-btn"]`
- **Expected**: Summary updates in the drawer. Cancel via `[data-qa="task-summary-cancel-btn"]`.

#### 6.5 — Edit description

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="task-description-edit-btn"]`
  2. Update `[data-qa="task-description-input"]`
  3. Click `[data-qa="task-description-save-btn"]`
- **Expected**: Description updates. Cancel via `[data-qa="task-description-cancel-btn"]`. Attach image via `[data-qa="task-description-attach-image-btn"]`.

#### 6.6 — Edit priority via dropdown

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="task-priority-btn"]`
  2. Choose `[data-qa="task-priority-option-high"]`
- **Expected**: Priority badge changes to "high".

#### 6.7 — Edit assigned role (agent)

- **Pre-conditions**: Task Drawer is open; at least one agent exists.
- **Steps**:
  1. Click `[data-qa="task-role-btn"]`
  2. Select `[data-qa="task-role-option-{slug}"]`
- **Expected**: The role assignment updates. Unassign via `[data-qa="task-role-unassign-btn"]`.

#### 6.8 — Edit effort estimate

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="task-effort-btn"]`
  2. Select an option `[data-qa="task-effort-option-{size}"]`
- **Expected**: Effort estimate is updated. Clear via `[data-qa="task-effort-clear-btn"]`.

#### 6.9 — Edit resolution

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="task-resolution-edit-btn"]`
  2. Update `[data-qa="task-resolution-input"]`
  3. Click `[data-qa="task-resolution-save-btn"]`
- **Expected**: Resolution updates. Cancel via `[data-qa="task-resolution-cancel-btn"]`.

#### 6.10 — Add and remove tags

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Type a tag in `[data-qa="task-tag-input"]`
  2. Click `[data-qa="task-add-tag-btn"]`
  3. Click `[data-qa="task-remove-tag-btn"]` to remove
- **Expected**: Tags are added/removed from the task.

#### 6.11 — Add and remove context files

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Type a path in `[data-qa="task-context-file-input"]`
  2. Click `[data-qa="task-add-context-file-btn"]`
  3. Click `[data-qa="task-remove-context-file-btn"]` to remove
- **Expected**: Context files are added/removed from the task.

#### 6.12 — Add a dependency

- **Pre-conditions**: Task Drawer is open; at least two tasks exist.
- **Steps**:
  1. Click `[data-qa="task-add-dependency-btn"]`
  2. Type in `[data-qa="task-dependency-search-input"]`
  3. Select a task from `[data-qa="task-dependency-result-btn"]`
- **Expected**: The selected task appears in the dependencies list `[data-qa="task-dependency-link"]`.

#### 6.13 — Remove a dependency

- **Pre-conditions**: Task has at least one dependency.
- **Steps**:
  1. Click `[data-qa="task-remove-dependency-btn"]` next to the dependency
- **Expected**: The dependency is removed.

#### 6.14 — Close dependency search

- **Pre-conditions**: Dependency search is open.
- **Steps**:
  1. Click `[data-qa="task-dependency-search-close-btn"]`
- **Expected**: Search closes.

#### 6.15 — Dependents section shows tasks that depend on this task

- **Pre-conditions**: Another task depends on the current task.
- **Steps**:
  1. Open the Task Drawer
  2. Scroll to the "Dependents" section
- **Expected**: Dependent tasks appear with `[data-qa="task-dependent-link"]`.

---

### 7. Task Action Modals

#### 7.1 — Delete task

- **Pre-conditions**: A task exists on the board.
- **Steps**:
  1. Right-click the task card and click `[data-qa="context-menu-delete-btn"]`
  2. Confirm via `[data-qa="delete-task-confirm-btn"]`
- **Expected**: Modal closes; task card is removed. Cancel via `[data-qa="delete-task-cancel-btn"]` or `[data-qa="delete-task-close-btn"]`.

#### 7.2 — Block task

- **Pre-conditions**: A task exists in Todo or In Progress.
- **Steps**:
  1. Right-click task and click `[data-qa="context-menu-block-btn"]`
  2. Fill `[data-qa="block-reason-input"]` with 50+ characters
  3. Fill `[data-qa="block-agent-name-input"]` with `"human"`
  4. Click `[data-qa="block-task-submit-btn"]`
- **Expected**: Modal closes; task moves to the Blocked column. Cancel via `[data-qa="block-task-cancel-btn"]` or `[data-qa="block-task-close-btn"]`.

#### 7.3 — Unblock a blocked task

- **Pre-conditions**: A task is in the Blocked column with `is_blocked=1`.
- **Steps**:
  1. Open Task Actions for the blocked task
  2. Click unblock action
  3. Confirm via `[data-qa="unblock-task-submit-btn"]`
- **Expected**: Task moves to Todo. Cancel via `[data-qa="unblock-task-cancel-btn"]` or `[data-qa="unblock-task-close-btn"]`.

#### 7.4 — Complete task

- **Pre-conditions**: A task is in the "In Progress" column.
- **Steps**:
  1. Right-click task and click `[data-qa="context-menu-complete-btn"]`
  2. Fill `[data-qa="complete-summary-input"]` with 100+ characters
  3. Fill `[data-qa="complete-agent-name-input"]` with `"human"`
  4. Optionally add files via `[data-qa="complete-file-path-input"]` + `[data-qa="complete-add-file-btn"]`
  5. Click `[data-qa="complete-task-submit-btn"]`
- **Expected**: Modal closes; task moves to Done. Cancel via `[data-qa="complete-task-cancel-btn"]` or `[data-qa="complete-task-close-btn"]`. Remove file via `[data-qa="complete-remove-file-btn"]`.

#### 7.5 — Complete task validation: summary must be 100+ chars

- **Pre-conditions**: Complete Task modal is open.
- **Steps**:
  1. Fill the summary with fewer than 100 characters
  2. Observe the submit button
- **Expected**: Submit button is disabled.

#### 7.6 — Mark task as Won't Do

- **Pre-conditions**: A task exists on the board.
- **Steps**:
  1. Open Task Actions and choose "Won't Do"
  2. Fill `[data-qa="wont-do-reason-input"]` with a reason
  3. Click `[data-qa="mark-wont-do-submit-btn"]`
- **Expected**: Task moves to Blocked column; "Won't Do Requested" badge appears. Cancel via `[data-qa="mark-wont-do-cancel-btn"]` or `[data-qa="mark-wont-do-close-btn"]`.

#### 7.7 — Approve Won't Do

- **Pre-conditions**: A task is in Blocked with `wont_do_requested=1`.
- **Steps**:
  1. Open Task Actions and choose "Approve Won't Do"
  2. Click `[data-qa="wont-do-approve-btn"]`
- **Expected**: Task moves to Done column; still shows won't-do state. Close via `[data-qa="approve-wont-do-close-btn"]`.

#### 7.8 — Reject Won't Do with optional comment

- **Pre-conditions**: A task is in Blocked with `wont_do_requested=1`.
- **Steps**:
  1. Open Task Actions and choose "Approve Won't Do" modal
  2. Fill `[data-qa="wont-do-rejection-reason-input"]`
  3. Click `[data-qa="wont-do-reject-btn"]`
- **Expected**: Task moves back to Todo; `wont_do_requested` is cleared; a comment is added.

#### 7.9 — Comment on Won't Do (CommentWontDoModal)

- **Pre-conditions**: A won't-do task exists.
- **Steps**:
  1. Open "Comment Won't Do" action
  2. Fill comment in `[data-qa="comment-content-input"]`
  3. Optionally check `[data-qa="comment-mark-wont-do-checkbox"]`
  4. Click `[data-qa="comment-wont-do-submit-btn"]`
- **Expected**: Comment is posted. Cancel via `[data-qa="comment-wont-do-cancel-btn"]` or `[data-qa="comment-wont-do-close-btn"]`.

#### 7.10 — Move task to another project

- **Pre-conditions**: At least two projects exist; a task is on the board.
- **Steps**:
  1. Right-click task and click `[data-qa="context-menu-move_to_project-btn"]`
  2. Select target project from `[data-qa="move-to-project-select"]`
  3. Click `[data-qa="move-to-project-submit-btn"]`
- **Expected**: Task disappears from current board; appears on target project's board. Cancel via `[data-qa="move-to-project-cancel-btn"]` or `[data-qa="move-to-project-close-btn"]`.

---

### 8. Task Card Display

#### 8.1 — Task card shows priority badge

- **Pre-conditions**: Tasks with different priorities exist.
- **Steps**:
  1. Navigate to the board
- **Expected**: Each card has a colored priority pill (critical / high / medium / low).

#### 8.2 — Task card shows role badge when assigned

- **Pre-conditions**: A task is assigned to an agent/role.
- **Steps**:
  1. Find a task card with an assigned role
- **Expected**: The role name is displayed as a badge with the role's color.

#### 8.3 — Task card shows comment count

- **Pre-conditions**: A task has at least one comment.
- **Steps**:
  1. Find the task card
- **Expected**: A speech-bubble icon with the comment count is visible.

#### 8.4 — Task card shows duration for completed tasks

- **Pre-conditions**: A completed task has `duration_seconds > 0`.
- **Steps**:
  1. Find the task card in the Done column
- **Expected**: A formatted duration string (e.g. "2h 30m") is shown.

#### 8.5 — Task card shows unresolved dependency icon

- **Pre-conditions**: A task has unresolved dependencies.
- **Steps**:
  1. Find the task card with unresolved deps
- **Expected**: A GitBranch icon is visible on the card.

#### 8.6 — Task card shows feature dot when in a feature

- **Pre-conditions**: A task belongs to a feature (sub-project).
- **Steps**:
  1. Find the task card
- **Expected**: A small colored dot indicates feature membership.

---

### 9. Agents Management (Roles Page)

#### 9.1 — Agents page lists all global agents

- **Pre-conditions**: At least one agent exists.
- **Steps**:
  1. Navigate to `/roles`
  2. Wait for agents to load
- **Expected**: Agent cards `[data-qa="agent-card"]` are displayed with name, slug, icon, and color.

#### 9.2 — Create a new agent

- **Pre-conditions**: On the Agents page.
- **Steps**:
  1. Click `[data-qa="new-agent-btn"]` (or `[data-qa="agents-create-first-agent-btn"]` if empty)
  2. Fill `[data-qa="agent-name-input"]` with `"Backend Engineer"`
  3. Fill `[data-qa="agent-slug-input"]` with `"backend-engineer"`
  4. Optionally fill `[data-qa="agent-modal-description-textarea"]`
  5. Optionally fill `[data-qa="agent-modal-prompt-template-textarea"]`
  6. Optionally fill `[data-qa="agent-modal-prompt-hint-textarea"]`
  7. Click `[data-qa="agent-save-btn"]`
- **Expected**: New agent appears in the list.

#### 9.3 — Agent name is required

- **Pre-conditions**: Agent modal is open.
- **Steps**:
  1. Leave name empty
  2. Click `[data-qa="agent-save-btn"]`
- **Expected**: Save button is disabled or error is shown.

#### 9.4 — Cancel agent creation

- **Pre-conditions**: Agent modal is open.
- **Steps**:
  1. Click `[data-qa="agent-cancel-btn"]` or `[data-qa="agent-modal-close-btn"]`
- **Expected**: Modal closes without creating an agent.

#### 9.5 — Edit an existing agent

- **Pre-conditions**: At least one agent exists.
- **Steps**:
  1. Click on an agent card `[data-qa="agent-card"]`
  2. Update fields in the modal
  3. Click `[data-qa="agent-save-btn"]`
- **Expected**: Agent details update in the list.

#### 9.6 — Delete an agent

- **Pre-conditions**: An agent exists with no tasks assigned.
- **Steps**:
  1. Open agent modal
  2. Click `[data-qa="agent-delete-btn"]`
  3. Confirm deletion
- **Expected**: The agent is removed from the list.

#### 9.7 — Clone an agent

- **Pre-conditions**: An agent exists.
- **Steps**:
  1. Click `[data-qa="agent-card-clone-btn"]`
  2. Fill `[data-qa="clone-agent-slug-input"]` with a new slug
  3. Optionally fill `[data-qa="clone-agent-name-input"]`
  4. Click `[data-qa="clone-agent-submit-btn"]`
- **Expected**: A cloned agent appears in the list. Cancel via `[data-qa="clone-agent-cancel-btn"]` or `[data-qa="clone-agent-close-btn"]`.

#### 9.8 — Agent icon and color selection

- **Pre-conditions**: Agent modal is open.
- **Steps**:
  1. Click `[data-qa="agent-modal-icon-toggle"]` and select `[data-qa="agent-modal-icon-btn"]`
  2. Click `[data-qa="agent-modal-color-btn"]` to select a color
- **Expected**: Icon and color are updated on the agent card.

#### 9.9 — Project-scoped agents page

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/roles`
- **Expected**: Agents assigned to the project are displayed. Default agent is indicated.

#### 9.10 — Set default agent on project-scoped page

- **Pre-conditions**: On project agents page; at least one agent assigned.
- **Steps**:
  1. Click `[data-qa="agent-card-set-default-btn"]` on an agent
- **Expected**: That agent is marked as default for the project.

#### 9.11 — Agent slug is disabled when editing

- **Pre-conditions**: Editing an existing agent.
- **Steps**:
  1. Observe `[data-qa="agent-slug-input"]`
- **Expected**: The slug field is disabled and cannot be changed.

---

### 10. Features / Sub-projects

#### 10.1 — Features page lists existing features

- **Pre-conditions**: A project with at least one feature exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/features`
  2. Wait for features to load
- **Expected**: Feature cards `[data-qa="feature-card"]` are visible with name, description, and task counts.

#### 10.2 — Create a new feature

- **Pre-conditions**: On the Features page.
- **Steps**:
  1. Click `[data-qa="add-feature-btn"]` (or `[data-qa="create-first-feature-btn"]` if empty)
  2. Fill `[data-qa="new-feature-name-input"]` with `"Authentication Feature"`
  3. Optionally fill `[data-qa="new-feature-description-textarea"]`
  4. Optionally add tags in `[data-qa="new-feature-tag-input"]`
  5. Click `[data-qa="confirm-create-feature-btn"]`
- **Expected**: The modal closes; the new feature appears in the list.

#### 10.3 — Create feature: name is required

- **Pre-conditions**: Create Feature modal is open.
- **Steps**:
  1. Leave the name field empty
  2. Click `[data-qa="confirm-create-feature-btn"]`
- **Expected**: Button is disabled; feature is not created.

#### 10.4 — Cancel feature creation

- **Pre-conditions**: Create Feature modal is open.
- **Steps**:
  1. Click `[data-qa="cancel-create-feature-btn"]` or the backdrop `[data-qa="create-feature-modal-backdrop"]`
- **Expected**: Modal closes; no feature is created.

#### 10.5 — Edit a feature name and description

- **Pre-conditions**: Feature exists.
- **Steps**:
  1. Click `[data-qa="edit-feature-btn"]` on a feature card
  2. Update `[data-qa="edit-feature-name-input"]`
  3. Update `[data-qa="edit-feature-description-textarea"]`
  4. Click `[data-qa="confirm-edit-feature-btn"]`
- **Expected**: Modal closes; feature name updates. Cancel via `[data-qa="cancel-edit-feature-btn"]` or `[data-qa="edit-feature-modal-backdrop"]`.

#### 10.6 — Open feature board link

- **Pre-conditions**: Feature card is visible.
- **Steps**:
  1. Click `[data-qa="open-feature-board-link"]`
- **Expected**: URL changes to `/projects/:featureId` (the feature's own board).

#### 10.7 — Toggle show done features

- **Pre-conditions**: On the Features page.
- **Steps**:
  1. Click `[data-qa="toggle-done-features-btn"]`
- **Expected**: Done/completed features are shown or hidden.

---

### 11. Backlog

#### 11.1 — Backlog page shows tasks in the backlog column

- **Pre-conditions**: A project with at least one task in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
  2. Wait for tasks to load
- **Expected**: Task rows are displayed with title, summary, and priority indicator.

#### 11.2 — Empty backlog shows empty state

- **Pre-conditions**: No tasks in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
- **Expected**: An empty state message is visible.

#### 11.3 — Move individual task from backlog to Todo

- **Pre-conditions**: At least one task is in the backlog.
- **Steps**:
  1. Click `[data-qa="move-task-to-todo-btn"]` on a task row
- **Expected**: The task disappears from the backlog; appears in the Todo column on the board.

#### 11.4 — Move all tasks from backlog to Todo

- **Pre-conditions**: Multiple tasks are in the backlog.
- **Steps**:
  1. Click `[data-qa="move-all-to-todo-btn"]`
- **Expected**: All tasks move from backlog to Todo.

#### 11.5 — Filter backlog by feature

- **Pre-conditions**: Tasks from multiple features are in the backlog.
- **Steps**:
  1. Select a feature from `[data-qa="feature-filter-select"]`
- **Expected**: Only tasks belonging to the selected feature are shown.

#### 11.6 — Open task drawer from backlog

- **Pre-conditions**: At least one task is in the backlog.
- **Steps**:
  1. Click `[data-qa="task-open-btn"]` on a task row
- **Expected**: The Task Drawer opens for that task.

---

### 12. Project Settings

#### 12.1 — Settings page displays project fields

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/settings`
- **Expected**: Name `[data-qa="project-name-input"]`, description `[data-qa="project-description-textarea"]`, and Git URL `[data-qa="project-git-url-input"]` are populated.

#### 12.2 — Update project name

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Clear `[data-qa="project-name-input"]` and type `"Renamed Project"`
  2. Click `[data-qa="save-project-settings-btn"]`
- **Expected**: Project name updates; button shows "Saved" momentarily.

#### 12.3 — Update project description

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Update `[data-qa="project-description-textarea"]`
  2. Click `[data-qa="save-project-settings-btn"]`
- **Expected**: Description is persisted.

#### 12.4 — Save button is disabled when name is empty

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Clear the name field completely
- **Expected**: `[data-qa="save-project-settings-btn"]` is disabled.

#### 12.5 — WIP Limits section

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Scroll to WIP Limits section
  2. Change `[data-qa="wip-limit-in_progress-input"]` to `5`
  3. Click `[data-qa="save-wip-limits-btn"]`
- **Expected**: WIP limit updates; board reflects the new limit.

#### 12.6 — Project Agents section: list assigned agents

- **Pre-conditions**: At least one agent is assigned to the project.
- **Steps**:
  1. Scroll to the "Project Agents" section
- **Expected**: Each assigned agent is listed with name, slug, and color dot.

#### 12.7 — Project Agents section: add an agent

- **Pre-conditions**: At least one global agent exists not yet assigned to the project.
- **Steps**:
  1. Click `[data-qa="add-agent-btn"]`
  2. Select an agent from `[data-qa="add-agent-select"]`
  3. Click `[data-qa="add-agent-confirm-btn"]`
- **Expected**: The agent appears in the list. Cancel via `[data-qa="add-agent-cancel-btn"]`.

#### 12.8 — Project Agents section: set default agent

- **Pre-conditions**: At least one agent is assigned.
- **Steps**:
  1. Click `[data-qa="set-default-agent-btn"]` on an agent
- **Expected**: That agent is marked as default.

#### 12.9 — Project Agents section: remove an agent

- **Pre-conditions**: At least one agent is assigned.
- **Steps**:
  1. Click `[data-qa="remove-agent-btn"]`
  2. Choose action:
     - Reassign: select `[data-qa="remove-agent-reassign-radio"]` + choose target from `[data-qa="remove-agent-reassign-select"]`
     - Clear: select `[data-qa="remove-agent-clear-radio"]`
  3. Click `[data-qa="remove-agent-confirm-btn"]`
- **Expected**: Agent is removed; tasks are handled per chosen option. Cancel via `[data-qa="remove-agent-cancel-btn"]`.

#### 12.10 — Danger Zone: delete project

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Scroll to the Danger Zone
  2. Click `[data-qa="delete-project-btn"]`
  3. Confirm in the dialog
- **Expected**: Project is deleted; user is redirected to `/`.

---

### 13. Skills

#### 13.1 — Skills page lists all skills

- **Pre-conditions**: At least one skill exists.
- **Steps**:
  1. Navigate to `/skills`
  2. Wait for skills to load
- **Expected**: Skill cards `[data-qa="skill-card"]` are displayed with name, slug, and color.

#### 13.2 — Create a new skill

- **Pre-conditions**: On the Skills page.
- **Steps**:
  1. Click `[data-qa="new-skill-btn"]` (or `[data-qa="create-first-skill-btn"]` if empty)
  2. Fill `[data-qa="skill-name-input"]` with `"Go Testing"`
  3. Verify `[data-qa="skill-slug-input"]` is auto-populated
  4. Select a color from swatches `[data-qa="skill-color-{hex}-btn"]`
  5. Fill `[data-qa="skill-description-textarea"]`
  6. Fill `[data-qa="skill-content-textarea"]` with Markdown content
  7. Optionally set `[data-qa="skill-sort-order-input"]`
  8. Click `[data-qa="save-skill-btn"]`
- **Expected**: The modal closes; the new skill card appears in the grid.

#### 13.3 — Skill name is required

- **Pre-conditions**: Skill modal is open.
- **Steps**:
  1. Leave name empty
  2. Click `[data-qa="save-skill-btn"]`
- **Expected**: Save button is disabled.

#### 13.4 — Skill slug is disabled when editing

- **Pre-conditions**: Editing an existing skill.
- **Steps**:
  1. Observe `[data-qa="skill-slug-input"]`
- **Expected**: The slug field is disabled.

#### 13.5 — Edit an existing skill

- **Pre-conditions**: At least one skill exists.
- **Steps**:
  1. Click `[data-qa="skill-edit-btn"]` on a skill card
  2. Update description or content
  3. Click `[data-qa="save-skill-btn"]`
- **Expected**: Skill card updates.

#### 13.6 — Delete a skill (no agents assigned)

- **Pre-conditions**: A skill exists not assigned to any agent.
- **Steps**:
  1. Click `[data-qa="skill-delete-btn"]`
  2. Click `[data-qa="confirm-delete-skill-btn"]`
- **Expected**: Skill is removed from the grid.

#### 13.7 — Delete a skill that is in use shows error

- **Pre-conditions**: A skill is assigned to at least one agent.
- **Steps**:
  1. Click `[data-qa="skill-delete-btn"]`
  2. Click `[data-qa="confirm-delete-skill-btn"]`
- **Expected**: An error message is shown. Cancel via `[data-qa="cancel-delete-skill-btn"]`.

#### 13.8 — Cancel skill creation/edit

- **Pre-conditions**: Skill modal is open.
- **Steps**:
  1. Click `[data-qa="cancel-skill-modal-btn"]`
- **Expected**: Modal closes without saving.

---

### 14. Dockerfiles

#### 14.1 — Dockerfiles page lists all dockerfiles

- **Pre-conditions**: At least one dockerfile exists.
- **Steps**:
  1. Navigate to `/dockerfiles`
  2. Wait for dockerfiles to load
- **Expected**: Dockerfile cards are displayed grouped by slug with name, version, and description.

#### 14.2 — Create a new dockerfile

- **Pre-conditions**: On the Dockerfiles page.
- **Steps**:
  1. Click `[data-qa="new-dockerfile-btn"]` (or `[data-qa="create-first-dockerfile-btn"]` if empty)
  2. Fill `[data-qa="dockerfile-name-input"]` with `"Go Builder"`
  3. Fill `[data-qa="dockerfile-slug-input"]` with `"go-builder"`
  4. Fill `[data-qa="dockerfile-version-input"]` with `"1.0"`
  5. Fill `[data-qa="dockerfile-content-textarea"]` with Dockerfile content
  6. Optionally fill `[data-qa="dockerfile-description-textarea"]`
  7. Optionally check `[data-qa="dockerfile-is-latest-checkbox"]`
  8. Optionally set `[data-qa="dockerfile-sort-order-input"]`
  9. Click `[data-qa="save-dockerfile-btn"]`
- **Expected**: Modal closes; the new dockerfile appears in the list.

#### 14.3 — Edit an existing dockerfile

- **Pre-conditions**: At least one dockerfile exists.
- **Steps**:
  1. Click `[data-qa="dockerfile-edit-btn"]`
  2. Update fields
  3. Click `[data-qa="save-dockerfile-btn"]`
- **Expected**: Dockerfile updates.

#### 14.4 — Delete a dockerfile

- **Pre-conditions**: A dockerfile exists not assigned to any project.
- **Steps**:
  1. Click `[data-qa="dockerfile-delete-btn"]`
  2. Click `[data-qa="confirm-delete-dockerfile-btn"]`
- **Expected**: Dockerfile is removed. Cancel via `[data-qa="cancel-delete-dockerfile-btn"]`.

#### 14.5 — Cancel dockerfile modal

- **Pre-conditions**: Dockerfile modal is open.
- **Steps**:
  1. Click `[data-qa="cancel-dockerfile-modal-btn"]` or `[data-qa="cancel-dockerfile-modal-footer-btn"]`
- **Expected**: Modal closes without saving.

---

### 15. Account

#### 15.1 — Account page loads profile

- **Pre-conditions**: User is logged in.
- **Steps**:
  1. Navigate to `/account`
- **Expected**: Display name input `[data-qa="account-display-name-input"]` is visible with current name.

#### 15.2 — Update display name

- **Pre-conditions**: On the Account page.
- **Steps**:
  1. Update `[data-qa="account-display-name-input"]`
  2. Click `[data-qa="account-save-profile-btn"]`
- **Expected**: Profile is saved; success message shown.

#### 15.3 — Change password

- **Pre-conditions**: On the Account page.
- **Steps**:
  1. Fill current password, new password, and confirm password fields
  2. Click `[data-qa="account-change-password-btn"]`
- **Expected**: Password is changed; success message shown.

---

### 16. API Keys

#### 16.1 — API Keys page loads

- **Pre-conditions**: User is logged in.
- **Steps**:
  1. Navigate to `/account/api-keys`
- **Expected**: Create API Key button `[data-qa="create-api-key-btn"]` is visible.

#### 16.2 — Create a new API key

- **Pre-conditions**: On the API Keys page.
- **Steps**:
  1. Click `[data-qa="create-api-key-btn"]`
  2. Fill `[data-qa="api-key-name-input"]` with `"Test Key"`
  3. Check scopes `[data-qa="scope-kanban:read"]`, `[data-qa="scope-kanban:write"]`
  4. Click `[data-qa="create-api-key-submit-btn"]`
- **Expected**: Key is created; key value shown with `[data-qa="copy-api-key-btn"]` (shown only once).

#### 16.3 — Revoke an API key

- **Pre-conditions**: At least one API key exists.
- **Steps**:
  1. Click `[data-qa="revoke-key-{id}"]` on a key row
- **Expected**: Key is revoked and removed from the list.

---

### 17. Statistics

#### 17.1 — Statistics page shows summary cards

- **Pre-conditions**: A project with tasks exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/statistics`
  2. Wait for data to load
- **Expected**: Summary stat cards are visible (Total Tasks, Done, In Progress, Blocked).

#### 17.2 — Time range buttons

- **Pre-conditions**: On the Statistics page.
- **Steps**:
  1. Click `[data-qa="time-range-7d-btn"]`
  2. Click `[data-qa="time-range-14d-btn"]`
  3. Click `[data-qa="time-range-30d-btn"]`
- **Expected**: Charts and stats update to reflect the selected time range.

#### 17.3 — Token Usage section

- **Pre-conditions**: Tasks with recorded token usage exist.
- **Steps**:
  1. Scroll to the "Token Usage" section
- **Expected**: Input, Output, Cache Read, and Cache Write token counts are displayed.

#### 17.4 — MCP Tool Calls section

- **Pre-conditions**: MCP tools have been called at least once.
- **Steps**:
  1. Scroll to the "MCP Tool Calls" section
- **Expected**: Tool names with bar charts and call counts are shown.

#### 17.5 — Tasks by Priority section

- **Pre-conditions**: Tasks with different priorities exist.
- **Steps**:
  1. Scroll to priority breakdown
- **Expected**: Priority chips (critical, high, medium, low) with counts are shown.

#### 17.6 — Tasks by Role section

- **Pre-conditions**: Tasks assigned to different roles exist.
- **Steps**:
  1. Scroll to role breakdown
- **Expected**: Role chips with task counts are shown.

#### 17.7 — Velocity chart

- **Pre-conditions**: Tasks were completed in recent days.
- **Steps**:
  1. Scroll to "Velocity" section
- **Expected**: A bar chart with date labels and completion counts is rendered.

#### 17.8 — Burndown chart

- **Pre-conditions**: Activity data exists.
- **Steps**:
  1. Scroll to "Burndown" section
- **Expected**: An SVG chart with a line, area fill, and axis labels is visible.

#### 17.9 — Cold Start Cost per Agent Role table

- **Pre-conditions**: Cold start stats are available.
- **Steps**:
  1. Scroll to "Cold Start Cost" section
- **Expected**: A table with Role, Runs, Min/Avg/Max Input, Avg Cache Read columns.

#### 17.10 — Statistics zero state for empty project

- **Pre-conditions**: Project has no tasks.
- **Steps**:
  1. Navigate to statistics
- **Expected**: Zero counts displayed; no errors.

---

### 18. Export (Claude / Gemini)

#### 18.1 — Export to Claude page renders

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/export/claude`
- **Expected**: Page title "Export to Claude Code" is shown; `[data-qa="back-to-project-link"]` is visible.

#### 18.2 — Export to Gemini page renders

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/export/gemini`
- **Expected**: Page title "Export to Gemini" is shown; `[data-qa="back-to-project-link"]` is visible.

#### 18.3 — "Back to Project" link navigates correctly

- **Pre-conditions**: On an export page.
- **Steps**:
  1. Click `[data-qa="back-to-project-link"]`
- **Expected**: URL returns to `/projects/:projectId`.

---

### 19. Theme Toggle

#### 19.1 — Theme toggle button is visible

- **Pre-conditions**: App is loaded on any page.
- **Steps**:
  1. Observe sidebar for `[data-qa="theme-toggle-btn"]`
- **Expected**: A theme toggle button is visible.

#### 19.2 — Toggle from dark to light theme

- **Pre-conditions**: App is in dark theme.
- **Steps**:
  1. Click `[data-qa="theme-toggle-btn"]`
- **Expected**: Page switches to light color scheme.

#### 19.3 — Toggle from light to dark theme

- **Pre-conditions**: App is in light theme.
- **Steps**:
  1. Click `[data-qa="theme-toggle-btn"]`
- **Expected**: Page switches to dark color scheme.

#### 19.4 — Theme preference persists across navigation

- **Pre-conditions**: App is switched to light theme.
- **Steps**:
  1. Navigate to a different route
- **Expected**: Light theme is still applied.

---

### 20. Comments

#### 20.1 — Comments load when Task Drawer is opened

- **Pre-conditions**: A task with existing comments.
- **Steps**:
  1. Open the Task Drawer
  2. Scroll to the Comments section
- **Expected**: Comments are listed with author, content, and timestamp.

#### 20.2 — Post a new comment

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Select author from `[data-qa="comment-author-select"]`
  2. Type text in `[data-qa="comment-content-input"]`
  3. Click `[data-qa="comment-submit-btn"]`
- **Expected**: Comment appears in the list; input is cleared.

#### 20.3 — Comment "Post as" selector defaults to "Human"

- **Pre-conditions**: Task Drawer is open, comment section visible.
- **Steps**:
  1. Observe `[data-qa="comment-author-select"]`
- **Expected**: "Human" is selected by default.

#### 20.4 — Send comment with Ctrl+Enter

- **Pre-conditions**: Comment textarea has content.
- **Steps**:
  1. Press `Ctrl+Enter`
- **Expected**: Comment is posted.

#### 20.5 — Upload image in comments

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click `[data-qa="comment-upload-image-btn"]`
- **Expected**: File picker opens accepting image files.

#### 20.6 — Empty comment cannot be submitted

- **Pre-conditions**: Comment textarea is empty.
- **Steps**:
  1. Observe `[data-qa="comment-submit-btn"]`
- **Expected**: Submit button is disabled.

---

### 21. Real-time WebSocket Updates

#### 21.1 — New task appears on board without refresh

- **Pre-conditions**: Two browser tabs on the same board.
- **Steps**:
  1. In Tab 2, create a new task via the UI or API
- **Expected**: The new task card appears in Tab 1's Todo column automatically.

#### 21.2 — Task move is reflected in all connected clients

- **Pre-conditions**: Two browser tabs on the same board.
- **Steps**:
  1. In Tab 2, move a task from Todo to In Progress
- **Expected**: The task card moves in Tab 1 automatically.

#### 21.3 — Task deletion is reflected in all connected clients

- **Pre-conditions**: Two browser tabs on the same board.
- **Steps**:
  1. In Tab 2, delete a task
- **Expected**: The task card disappears from Tab 1.

---

### 22. API Health Check

#### 22.1 — Projects endpoint returns 200 (authenticated)

- **Pre-conditions**: Server is running; valid auth token obtained.
- **Steps**:
  1. Send `GET /api/projects` with `Authorization: Bearer <token>`
- **Expected**: HTTP 200 with JSON array.

#### 22.2 — Server returns correct Content-Type

- **Pre-conditions**: Server is running.
- **Steps**:
  1. Send `GET /api/projects` with valid auth
- **Expected**: Response header `Content-Type: application/json`.

#### 22.3 — Unauthenticated request returns 401

- **Pre-conditions**: Server is running.
- **Steps**:
  1. Send `GET /api/projects` without Authorization header
- **Expected**: HTTP 401 response.
