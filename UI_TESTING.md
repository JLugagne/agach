# UI Testing Scenarios

## Setup

- Base URL: `http://localhost:8322`
- All tests use `data-qa` attributes for element selection
- The server must be running with a clean SQLite database (or a seeded test database)
- Tests assume a Playwright project configured with `baseURL: process.env.BASE_URL ?? 'http://localhost:8322'`

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

### 1. Home Page - Projects List

#### 1.1 — Render projects list on load

- **Pre-conditions**: Server is running; at least one project exists in the database.
- **Steps**:
  1. Navigate to `/`
  2. Wait for `[data-qa="project-card"]` elements to be visible
- **Expected**: One or more project cards are displayed with project names.

#### 1.2 — Display empty state when no projects exist

- **Pre-conditions**: Server is running; database has no projects.
- **Steps**:
  1. Navigate to `/`
  2. Wait for page to load (spinner disappears)
- **Expected**: An empty state message is visible (e.g. "No projects yet" or similar).

#### 1.3 — Create a new project via the create button

- **Pre-conditions**: Home page is loaded.
- **Steps**:
  1. Navigate to `/`
  2. Click `[data-qa="create-project-btn"]`
  3. Fill `[data-qa="project-name-input"]` with `"Test Project"`
  4. Optionally fill `[data-qa="project-description-input"]` with `"A test project description"`
  5. Click `[data-qa="create-project-submit-btn"]`
- **Expected**: The modal closes and a new project card with the name "Test Project" appears in the list.

#### 1.4 — Click on a project card navigates to the kanban board

- **Pre-conditions**: At least one project exists.
- **Steps**:
  1. Navigate to `/`
  2. Click on `[data-qa="project-card"]` for the desired project
- **Expected**: URL changes to `/projects/:projectId` and the kanban board is rendered.

#### 1.5 — Project card shows task summary counts

- **Pre-conditions**: A project with tasks in various columns exists.
- **Steps**:
  1. Navigate to `/`
  2. Inspect `[data-qa="project-card"]` for task counts
- **Expected**: The card displays todo, in-progress, done, and/or blocked counts.

---

### 2. Navigation

#### 2.1 — Sidebar navigation links are visible on project pages

- **Pre-conditions**: A project exists; user is on the kanban board.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Check visibility of `[data-qa="nav-kanban"]`, `[data-qa="nav-backlog"]`, `[data-qa="nav-features"]`, `[data-qa="nav-settings"]`, `[data-qa="nav-statistics"]`
- **Expected**: All sidebar navigation links are visible.

#### 2.2 — Navigate to Backlog via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-backlog"]`
- **Expected**: URL changes to `/projects/:projectId/backlog` and the Backlog page header is visible.

#### 2.3 — Navigate to Features via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-features"]`
- **Expected**: URL changes to `/projects/:projectId/features` and the Features page is displayed.

#### 2.4 — Navigate to Settings via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-settings"]`
- **Expected**: URL changes to `/projects/:projectId/settings` and the Settings page title "Project Definition" is visible.

#### 2.5 — Navigate to Statistics via sidebar

- **Pre-conditions**: On a project page.
- **Steps**:
  1. Click `[data-qa="nav-statistics"]`
- **Expected**: URL changes to `/projects/:projectId/statistics` and the Statistics page heading is visible.

#### 2.6 — Navigate to global Roles page

- **Pre-conditions**: App is loaded.
- **Steps**:
  1. Click `[data-qa="nav-roles"]` in the global sidebar
- **Expected**: URL changes to `/roles` and the Roles page is displayed.

#### 2.7 — Navigate to global Skills page

- **Pre-conditions**: App is loaded.
- **Steps**:
  1. Click `[data-qa="nav-skills"]` in the global sidebar
- **Expected**: URL changes to `/skills` and the Skills page heading is visible.

#### 2.8 — Home (logo/breadcrumb) navigates to project list

- **Pre-conditions**: User is on a project subpage.
- **Steps**:
  1. Click `[data-qa="nav-home"]` or the app logo link
- **Expected**: URL changes to `/` and the home page is rendered.

---

### 3. Kanban Board

#### 3.1 — Board displays four columns

- **Pre-conditions**: On a project kanban page.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Wait for columns to load
- **Expected**: Four columns are visible: **Todo**, **In Progress**, **Done**, **Blocked**.

#### 3.2 — Task cards appear in the correct column

- **Pre-conditions**: A project with at least one task in the "todo" column.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Locate the Todo column
  3. Check for `[data-qa="task-card"]` inside the Todo column
- **Expected**: Task cards are visible in the Todo column.

#### 3.3 — Column displays task count badge

- **Pre-conditions**: A column with at least 2 tasks.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Observe the count badge in each column header
- **Expected**: The count badge reflects the number of tasks in the column.

#### 3.4 — WIP limit warning appears when column is at capacity

- **Pre-conditions**: The "In Progress" column has a WIP limit of 3 and exactly 3 tasks.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Observe the In Progress column header
- **Expected**: An alert/warning icon and the count displays as `3/3`, highlighted in red/blocked color.

#### 3.5 — Drag-and-drop task to reorder within a column

- **Pre-conditions**: A column with at least 2 tasks.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Hover over a task card to reveal the drag handle `[data-qa="task-drag-handle"]`
  3. Drag the first task card below the second task card in the same column
- **Expected**: The order of the tasks updates visually and persists after a page refresh.

#### 3.6 — Click task card opens the Task Drawer

- **Pre-conditions**: At least one task exists on the board.
- **Steps**:
  1. Click on a `[data-qa="task-card"]`
- **Expected**: The Task Drawer slides in from the right, displaying the task title, summary, priority, and column status.

#### 3.7 — Right-click (context menu) on a task card

- **Pre-conditions**: At least one task exists in the Todo or In Progress column.
- **Steps**:
  1. Right-click on a `[data-qa="task-card"]`
- **Expected**: A context menu appears with actions such as "Complete", "Block", "Delete", "Move to project".

#### 3.8 — Board search filters visible tasks

- **Pre-conditions**: Multiple tasks exist with distinct titles.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Click `[data-qa="board-search-input"]` or similar search field
  3. Type a partial title of an existing task
- **Expected**: Only matching task cards remain visible; non-matching cards are hidden.

#### 3.9 — "New Task" button opens the create task modal

- **Pre-conditions**: On the kanban board.
- **Steps**:
  1. Click `[data-qa="create-task-btn"]`
- **Expected**: The New Task modal appears.

#### 3.10 — Won't-do-requested badge is shown on blocked tasks

- **Pre-conditions**: A task exists in the Blocked column with `wont_do_requested=1`.
- **Steps**:
  1. Navigate to `/projects/:projectId`
  2. Observe the Blocked column
- **Expected**: The task card shows a "Won't Do Requested" badge.

---

### 4. Task Management

#### 4.1 — Create a task with only required fields

- **Pre-conditions**: On the kanban board.
- **Steps**:
  1. Click `[data-qa="create-task-btn"]`
  2. Fill `[data-qa="task-title-input"]` with `"My New Task"`
  3. Fill `[data-qa="task-summary-input"]` with `"Brief summary of the task"`
  4. Click `[data-qa="create-task-submit-btn"]`
- **Expected**: Modal closes; the new task card appears in the Todo column.

#### 4.2 — Create task validation: title is required

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Leave `[data-qa="task-title-input"]` empty
  2. Fill `[data-qa="task-summary-input"]` with text
  3. Click `[data-qa="create-task-submit-btn"]`
- **Expected**: An error message "Title is required" is shown; modal stays open.

#### 4.3 — Create task validation: summary is required

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Fill `[data-qa="task-title-input"]` with `"My Task"`
  2. Leave `[data-qa="task-summary-input"]` empty
  3. Click `[data-qa="create-task-submit-btn"]`
- **Expected**: An error message "Summary is required" is shown; modal stays open.

#### 4.4 — Create task with all optional fields

- **Pre-conditions**: New Task modal is open; at least one role exists.
- **Steps**:
  1. Fill title and summary
  2. Fill `[data-qa="task-description-input"]` with `"Detailed description"`
  3. Select priority `"critical"` from `[data-qa="task-priority-select"]`
  4. Select a role from `[data-qa="task-role-select"]`
  5. Check `[data-qa="task-add-to-backlog-checkbox"]` to add to backlog
  6. Type `"frontend"` in `[data-qa="task-tag-input"]` and press Enter
  7. Type `"src/main.go"` in `[data-qa="task-context-file-input"]` and press Enter
  8. Click `[data-qa="create-task-submit-btn"]`
- **Expected**: Task is created with correct priority, role, tags, and context files visible in the drawer.

#### 4.5 — Add to backlog checkbox places task in backlog instead of todo

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Fill title and summary
  2. Check `[data-qa="task-add-to-backlog-checkbox"]`
  3. Click `[data-qa="create-task-submit-btn"]`
- **Expected**: The task does NOT appear in the Todo column; it appears in the Backlog page.

#### 4.6 — Close New Task modal with Escape key

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Press `Escape`
- **Expected**: The modal closes without creating a task.

#### 4.7 — Close New Task modal by clicking backdrop

- **Pre-conditions**: New Task modal is open.
- **Steps**:
  1. Click outside the modal area (on the dark overlay)
- **Expected**: The modal closes.

#### 4.8 — Task Drawer: view task details

- **Pre-conditions**: A task with title, summary, description, priority, and tags exists.
- **Steps**:
  1. Click on the task card
- **Expected**: Drawer shows: title, summary, description (rendered as Markdown), priority badge, column status badge, tags, context files, comment count.

#### 4.9 — Task Drawer: inline edit title

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click on the task title (or the pencil/edit icon next to it)
  2. Clear the title input and type `"Updated Title"`
  3. Press Enter or click the confirm (check) button
- **Expected**: The title updates immediately in the drawer; the task card on the board also reflects the new title.

#### 4.10 — Task Drawer: inline edit summary

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click the edit icon next to the summary
  2. Update the summary text
  3. Confirm the edit
- **Expected**: Summary updates in the drawer.

#### 4.11 — Task Drawer: edit priority via dropdown

- **Pre-conditions**: Task Drawer is open; task is in Todo.
- **Steps**:
  1. Click the priority badge or select in the drawer
  2. Choose `"high"` from the priority options
- **Expected**: Priority badge changes to "high".

#### 4.12 — Task Drawer: edit assigned role

- **Pre-conditions**: Task Drawer is open; at least one role exists.
- **Steps**:
  1. Click the role field or dropdown in the drawer
  2. Select a role
- **Expected**: The role assignment updates on the card and in the drawer.

#### 4.13 — Task Drawer: change column (move task)

- **Pre-conditions**: Task Drawer is open; task is in Todo.
- **Steps**:
  1. Click the column status badge or the move button in the drawer
  2. Select "In Progress" as the target column
- **Expected**: Task card moves from Todo to In Progress on the board.

#### 4.14 — Task Drawer: add a dependency

- **Pre-conditions**: Task Drawer is open; at least two tasks exist.
- **Steps**:
  1. Scroll to the Dependencies section
  2. Click `[data-qa="add-dependency-btn"]` or the "+" button near dependencies
  3. Type part of another task title in the search input
  4. Select the task from the autocomplete results
- **Expected**: The selected task appears in the "Depends On" list.

#### 4.15 — Task Drawer: remove a dependency

- **Pre-conditions**: Task Drawer is open; the task has at least one dependency.
- **Steps**:
  1. Scroll to the Dependencies section
  2. Click the remove button next to the dependency
- **Expected**: The dependency is removed from the list.

#### 4.16 — Task Drawer: displays dependents (tasks that depend on this task)

- **Pre-conditions**: Another task depends on the current task.
- **Steps**:
  1. Open the Task Drawer
  2. Scroll to the "Dependents" section
- **Expected**: The dependent task appears in the list with its status badge.

#### 4.17 — Task Drawer: close with X button

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Click the `[data-qa="close-drawer-btn"]` or the X icon in the drawer header
- **Expected**: The drawer closes.

#### 4.18 — Task Drawer: close with Escape key

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Press `Escape`
- **Expected**: The drawer closes.

#### 4.19 — Delete task via Delete Task modal

- **Pre-conditions**: A task exists on the board.
- **Steps**:
  1. Right-click the task card to open context menu
  2. Click "Delete"
  3. The `[data-qa="delete-task-modal"]` appears
  4. Confirm the task title is shown in the modal
  5. Click `[data-qa="delete-task-confirm-btn"]`
- **Expected**: Modal closes; task card is removed from the board.

#### 4.20 — Delete task modal: cancel closes without deleting

- **Pre-conditions**: Delete Task modal is open.
- **Steps**:
  1. Click `[data-qa="delete-task-cancel-btn"]` or press `Escape`
- **Expected**: Modal closes; task card is still visible on the board.

#### 4.21 — Complete task via Complete Task modal

- **Pre-conditions**: A task is in the "In Progress" column.
- **Steps**:
  1. Right-click the task or open the drawer and choose "Complete"
  2. The `[data-qa="complete-task-modal"]` appears
  3. Fill `[data-qa="completion-summary-input"]` with 100+ characters
  4. Fill `[data-qa="completed-by-agent-input"]` with `"human"`
  5. Optionally add a file path in `[data-qa="files-modified-input"]` and click Add
  6. Click `[data-qa="complete-task-submit-btn"]`
- **Expected**: Modal closes; task card moves to the Done column.

#### 4.22 — Complete task validation: summary must be 100+ chars

- **Pre-conditions**: Complete Task modal is open.
- **Steps**:
  1. Fill the summary with fewer than 100 characters
  2. Fill the agent name
  3. Click the submit button
- **Expected**: Error message "Completion summary must be at least 100 characters" is shown; submit button is disabled.

#### 4.23 — Complete task validation: agent name is required

- **Pre-conditions**: Complete Task modal is open.
- **Steps**:
  1. Fill 100+ characters in the summary
  2. Leave agent name empty
  3. Click the submit button
- **Expected**: Error message "Completed by agent is required" is shown.

#### 4.24 — Block task via Block Task modal

- **Pre-conditions**: A task exists in the Todo or In Progress column.
- **Steps**:
  1. Right-click the task card or open the drawer and choose "Block"
  2. The `[data-qa="block-task-modal"]` appears
  3. Fill `[data-qa="block-reason-input"]` with 50+ characters
  4. Fill `[data-qa="block-agent-name-input"]` with `"human"`
  5. Click `[data-qa="block-task-submit-btn"]`
- **Expected**: Modal closes; task card moves to the Blocked column with `is_blocked=1`.

#### 4.25 — Block task validation: reason must be 50+ chars

- **Pre-conditions**: Block Task modal is open.
- **Steps**:
  1. Fill fewer than 50 characters in the reason field
  2. Fill the agent name
  3. Observe the submit button state
- **Expected**: Submit button is disabled; character counter shows the current count and the 50-character minimum.

#### 4.26 — Block task validation: agent name is required

- **Pre-conditions**: Block Task modal is open.
- **Steps**:
  1. Fill 50+ characters in the reason
  2. Leave agent name empty
  3. Observe the submit button state
- **Expected**: Submit button remains disabled.

#### 4.27 — Mark task as Won't Do

- **Pre-conditions**: A task exists on the board.
- **Steps**:
  1. Open the Task Actions for the task (right-click or drawer action)
  2. Choose "Won't Do" / `wontdo` action
  3. The MarkWontDo modal appears
  4. Fill in the reason
  5. Submit
- **Expected**: Task moves to the Blocked column; `wont_do_requested=1`; "Won't Do Requested" badge appears on the card.

#### 4.28 — Approve Won't Do

- **Pre-conditions**: A task is in the Blocked column with `wont_do_requested=1`.
- **Steps**:
  1. Open the Task Actions for the blocked task
  2. Choose "Approve Won't Do"
  3. Confirm in the modal
- **Expected**: Task moves to the Done column; still shows won't-do state.

#### 4.29 — Reject Won't Do with comment

- **Pre-conditions**: A task is in the Blocked column with `wont_do_requested=1`.
- **Steps**:
  1. Open the Task Actions for the blocked task
  2. Choose "Reject Won't Do" (comment_wontdo action)
  3. Add a rejection comment
  4. Submit
- **Expected**: Task moves back to the Todo column; `wont_do_requested` is cleared; a comment is added.

#### 4.30 — Unblock a blocked task

- **Pre-conditions**: A task is in the Blocked column with `is_blocked=1`.
- **Steps**:
  1. Open Task Actions for the blocked task
  2. Choose "Unblock"
  3. Confirm in the modal
- **Expected**: Task moves back to the Todo column; `is_blocked` is cleared.

#### 4.31 — Move task to another project

- **Pre-conditions**: At least two projects exist; a task is on the board.
- **Steps**:
  1. Right-click the task card
  2. Choose "Move to Project"
  3. The MoveToProjectModal appears
  4. Select the target project from the list
  5. Confirm
- **Expected**: Task disappears from the current board; it appears on the target project's board.

#### 4.32 — Task card shows priority badge

- **Pre-conditions**: Tasks with different priorities exist.
- **Steps**:
  1. Navigate to the board
  2. Inspect task cards
- **Expected**: Each card has a colored priority pill (critical / high / medium / low).

#### 4.33 — Task card shows role badge when assigned

- **Pre-conditions**: A task is assigned to a role.
- **Steps**:
  1. Navigate to the board
  2. Find a task card with an assigned role
- **Expected**: The role name is displayed as a badge on the card with the role's color.

#### 4.34 — Task card shows comment count

- **Pre-conditions**: A task has at least one comment.
- **Steps**:
  1. Navigate to the board
  2. Find the task card
- **Expected**: A speech-bubble icon with the comment count is visible on the card.

#### 4.35 — Task card shows duration for completed tasks

- **Pre-conditions**: A completed task has `duration_seconds > 0`.
- **Steps**:
  1. Navigate to the board
  2. Find the task card in the Done column
- **Expected**: A formatted duration string (e.g. "2h 30m") is shown on the card.

#### 4.36 — Task card shows unresolved dependency icon

- **Pre-conditions**: A task has unresolved dependencies.
- **Steps**:
  1. Navigate to the board
  2. Find the task card with `has_unresolved_deps=true`
- **Expected**: A GitBranch icon is visible on the card.

#### 4.37 — Task card shows feature dot when in a feature

- **Pre-conditions**: A task belongs to a feature (sub-project).
- **Steps**:
  1. Navigate to the board
  2. Find the task card
- **Expected**: A small colored dot is visible on the card indicating feature membership.

---

### 5. Roles Management

#### 5.1 — Roles page lists all global roles

- **Pre-conditions**: At least one role exists.
- **Steps**:
  1. Navigate to `/roles`
  2. Wait for the roles to load
- **Expected**: Role cards are displayed with role name, slug, icon, and color.

#### 5.2 — Create a new role

- **Pre-conditions**: On the Roles page.
- **Steps**:
  1. Click `[data-qa="create-role-btn"]`
  2. Fill `[data-qa="role-name-input"]` with `"Backend Engineer"`
  3. Fill `[data-qa="role-slug-input"]` with `"backend-engineer"` (or verify auto-generation)
  4. Click `[data-qa="create-role-submit-btn"]`
- **Expected**: New role appears in the list.

#### 5.3 — Edit an existing role

- **Pre-conditions**: At least one role exists.
- **Steps**:
  1. Click the edit button on a role card
  2. Update the name field
  3. Save
- **Expected**: The role name updates in the list.

#### 5.4 — Delete a role

- **Pre-conditions**: A role exists that has no tasks assigned.
- **Steps**:
  1. Click the delete button on the role card
  2. Confirm in the delete confirmation dialog
- **Expected**: The role is removed from the list.

#### 5.5 — Clone a role

- **Pre-conditions**: A role exists.
- **Steps**:
  1. Click the clone/copy button on a role card
  2. Provide a new slug in the clone modal
  3. Confirm
- **Expected**: A new role with the cloned data appears in the list.

#### 5.6 — Project-specific Roles page shows roles for the project

- **Pre-conditions**: A project exists; on the project's Roles page.
- **Steps**:
  1. Navigate to `/projects/:projectId/roles`
- **Expected**: Roles assigned to or available for this project are displayed.

---

### 6. Features / Sub-projects

#### 6.1 — Features page lists existing features

- **Pre-conditions**: A project with at least one feature exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/features`
  2. Wait for features to load
- **Expected**: Feature list items are visible with name, description, and task count.

#### 6.2 — Create a new feature

- **Pre-conditions**: On the Features page.
- **Steps**:
  1. Click `[data-qa="add-feature-btn"]` or the "Add Feature" button
  2. Fill `[data-qa="feature-name-input"]` with `"Authentication Feature"`
  3. Optionally fill `[data-qa="feature-description-input"]`
  4. Press Enter or click `[data-qa="create-feature-submit-btn"]`
- **Expected**: The modal closes; the new feature appears in the list.

#### 6.3 — Create feature: name is required

- **Pre-conditions**: Create Feature modal is open.
- **Steps**:
  1. Leave the name field empty
  2. Click the Create button
- **Expected**: Button is disabled; feature is not created.

#### 6.4 — Create feature via Enter key

- **Pre-conditions**: Create Feature modal is open with a name typed.
- **Steps**:
  1. Type a name in the name field
  2. Press `Enter`
- **Expected**: Feature is created (same as clicking the Create button).

#### 6.5 — Click a feature opens the right-side drawer with details

- **Pre-conditions**: At least one feature exists.
- **Steps**:
  1. Click on a feature row/card
- **Expected**: A right-side drawer opens showing the feature's name, description, and task summary counts (todo, in progress, done, blocked).

#### 6.6 — Feature drawer: Open Board link navigates to feature's kanban board

- **Pre-conditions**: Feature drawer is open.
- **Steps**:
  1. Click the "Open Board" link in the feature drawer
- **Expected**: URL changes to `/projects/:featureId` (the feature's own board).

#### 6.7 — Edit a feature name and description

- **Pre-conditions**: Feature drawer is open.
- **Steps**:
  1. Click the "Edit" button in the feature drawer
  2. The Edit Feature modal opens
  3. Update the name field to `"Renamed Feature"`
  4. Click the Save button
- **Expected**: The modal closes; the feature name updates in the list and drawer.

#### 6.8 — Delete a feature

- **Pre-conditions**: A feature exists.
- **Steps**:
  1. Open the feature drawer
  2. Click the "Delete" button
  3. The Delete Confirm modal shows the feature name
  4. Click the confirm button
- **Expected**: The feature is removed from the list; the drawer closes.

#### 6.9 — Feature shows "active" badge when it has in-progress or todo tasks

- **Pre-conditions**: A feature has at least one task in "todo" or "in_progress".
- **Steps**:
  1. Navigate to the Features page
- **Expected**: The feature row displays a small green "active" badge.

#### 6.10 — Close feature drawer with X button

- **Pre-conditions**: Feature drawer is open.
- **Steps**:
  1. Click the X button in the drawer header
- **Expected**: Drawer closes; no feature is selected.

---

### 7. Backlog

#### 7.1 — Backlog page shows tasks with column=backlog

- **Pre-conditions**: A project with at least one task in the backlog column.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
  2. Wait for tasks to load
- **Expected**: Task rows are displayed with title, summary, and priority indicator.

#### 7.2 — Empty backlog shows "No tasks in backlog" message

- **Pre-conditions**: No tasks in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
- **Expected**: The text "No tasks in backlog" is visible.

#### 7.3 — Move individual task from backlog to Todo

- **Pre-conditions**: At least one task is in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
  2. Hover over a task row to reveal the "Todo" arrow button
  3. Click the `[data-qa="move-to-todo-btn"]` button for that task
- **Expected**: The task row disappears from the backlog; the task appears in the Todo column on the board.

#### 7.4 — Move all tasks from backlog to Todo

- **Pre-conditions**: Multiple tasks are in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
  2. Click the `[data-qa="move-all-to-todo-btn"]` button in the header
- **Expected**: All task rows disappear from the backlog; they appear in the Todo column on the board.

#### 7.5 — Filter backlog by feature

- **Pre-conditions**: Tasks from multiple features are in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
  2. The feature filter dropdown is visible
  3. Select a specific feature from `[data-qa="feature-filter-select"]`
- **Expected**: Only tasks belonging to the selected feature are shown; the count badge updates.

#### 7.6 — Backlog task count badge in the header

- **Pre-conditions**: Tasks are in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
- **Expected**: A count badge next to the "Backlog" heading shows the number of visible tasks.

#### 7.7 — Click task row in backlog opens the Task Drawer

- **Pre-conditions**: At least one task is in the backlog.
- **Steps**:
  1. Click on the task title/row in the backlog list
- **Expected**: The Task Drawer opens for that task.

#### 7.8 — Priority color indicator on backlog task row

- **Pre-conditions**: Tasks with different priorities are in the backlog.
- **Steps**:
  1. Navigate to `/projects/:projectId/backlog`
  2. Observe the colored dot on each task row
- **Expected**: Critical tasks show red, high shows orange, medium shows blue, low shows muted color.

---

### 8. Settings

#### 8.1 — Settings page displays project name and description fields

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/settings`
- **Expected**: Name input and description textarea are populated with the current project values.

#### 8.2 — Update project name

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Clear `[data-qa="settings-name-input"]` and type `"Renamed Project"`
  2. Click `[data-qa="settings-save-btn"]`
- **Expected**: Button momentarily shows "Saved"; the project name updates in the sidebar.

#### 8.3 — Update project description

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Update the description textarea
  2. Click the Save Changes button
- **Expected**: Description is persisted; reloading the settings page shows the new description.

#### 8.4 — Save button is disabled when name is empty

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Clear the name field completely
  2. Observe the Save button
- **Expected**: The Save button is disabled.

#### 8.5 — Update Default Role for the project

- **Pre-conditions**: At least one role exists.
- **Steps**:
  1. On the Settings page, find the Default Role dropdown
  2. Select a role from `[data-qa="settings-default-role-select"]`
  3. Click Save
- **Expected**: The default role is saved; new tasks will pre-select this role.

#### 8.6 — WIP Limits section shows columns and allows editing

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Scroll to the "Column WIP Limits" section
  2. The inputs for each column (todo, in_progress, done, blocked) are visible
- **Expected**: Four number inputs are shown with the current WIP limit values.

#### 8.7 — Update WIP limit for In Progress column

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Change the In Progress WIP limit input to `5`
  2. Click `[data-qa="save-wip-limits-btn"]`
- **Expected**: The WIP limit is updated; the board reflects the new limit.

#### 8.8 — WIP limit of 0 means no limit

- **Pre-conditions**: On the Settings page; In Progress WIP limit is set to 5.
- **Steps**:
  1. Change the In Progress input to `0`
  2. Click Save WIP Limits
- **Expected**: No WIP limit badge appears in the In Progress column header.

#### 8.9 — Project Agents section: list assigned agents

- **Pre-conditions**: At least one agent is assigned to the project.
- **Steps**:
  1. Scroll to the "Project Agents" section on the settings page
- **Expected**: Each assigned agent is listed with name, slug, and color dot.

#### 8.10 — Project Agents section: add an agent

- **Pre-conditions**: At least one global role exists that is not yet assigned to the project.
- **Steps**:
  1. Click the "+ Add Agent" button
  2. The AddAgentToProjectDialog opens
  3. Select an agent from the list
  4. Confirm
- **Expected**: The agent appears in the Project Agents list.

#### 8.11 — Project Agents section: remove an agent

- **Pre-conditions**: At least one agent is assigned to the project.
- **Steps**:
  1. Click "Remove" next to an agent in the Project Agents list
  2. The RemoveAgentDialog opens
  3. Confirm removal
- **Expected**: The agent is removed from the Project Agents list.

#### 8.12 — Danger Zone: delete project

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Scroll to the Danger Zone section
  2. Click `[data-qa="delete-project-btn"]`
  3. The Delete Confirm modal appears with the project name
  4. Click the confirm button
- **Expected**: Project is deleted; user is redirected to `/` (home page).

#### 8.13 — Right drawer shows project definition JSON

- **Pre-conditions**: On the Settings page.
- **Steps**:
  1. Observe the right-side panel (if present)
- **Expected**: A read-only JSON view of the project definition is rendered.

---

### 9. Skills

#### 9.1 — Skills page lists all skills

- **Pre-conditions**: At least one skill exists.
- **Steps**:
  1. Navigate to `/skills`
  2. Wait for skills to load
- **Expected**: Skill cards are displayed in a grid with name, slug, icon, and color dot.

#### 9.2 — Skills page shows count in the subtitle

- **Pre-conditions**: Skills exist.
- **Steps**:
  1. Navigate to `/skills`
- **Expected**: The subtitle shows "N skills defined" where N is the correct count.

#### 9.3 — Create a new skill

- **Pre-conditions**: On the Skills page.
- **Steps**:
  1. Click `[data-qa="new-skill-btn"]` (the "New Skill" button)
  2. The Skill modal opens as a side drawer
  3. Fill `[data-qa="skill-name-input"]` with `"Go Testing"`
  4. Verify `[data-qa="skill-slug-input"]` is auto-populated with `"gotesting"`
  5. Set an icon emoji in `[data-qa="skill-icon-input"]`
  6. Select a color from the preset color swatches
  7. Fill `[data-qa="skill-description-input"]` with a description
  8. Fill `[data-qa="skill-content-input"]` with Markdown content
  9. Click `[data-qa="create-skill-submit-btn"]`
- **Expected**: The drawer closes; the new skill card appears in the grid.

#### 9.4 — Skill slug auto-generated from name

- **Pre-conditions**: New Skill modal is open.
- **Steps**:
  1. Type `"My Awesome Skill"` in the name field
  2. Observe the slug field
- **Expected**: Slug is auto-populated as `"myawesomeskill"` (lowercase, no spaces).

#### 9.5 — Skill slug is disabled when editing

- **Pre-conditions**: Skill modal is open in edit mode.
- **Steps**:
  1. Click the edit button on an existing skill
  2. Observe the slug field
- **Expected**: The slug field is disabled and cannot be changed.

#### 9.6 — Edit an existing skill

- **Pre-conditions**: At least one skill exists.
- **Steps**:
  1. Click the pencil (edit) icon on a skill card
  2. Update the description
  3. Click `[data-qa="save-skill-btn"]`
- **Expected**: The skill card updates with the new description.

#### 9.7 — Delete a skill (no agents assigned)

- **Pre-conditions**: A skill exists that is not assigned to any agent.
- **Steps**:
  1. Click the trash icon on the skill card
  2. A confirmation panel appears below the card
  3. Click "Confirm"
- **Expected**: The skill is removed from the grid.

#### 9.8 — Delete a skill that is in use shows error

- **Pre-conditions**: A skill is assigned to at least one agent.
- **Steps**:
  1. Click the trash icon on the skill card
  2. Click "Confirm" in the inline confirmation
- **Expected**: An error message "Cannot delete: skill is still assigned to one or more agents" is shown.

#### 9.9 — Cancel skill deletion

- **Pre-conditions**: Skill deletion confirmation is showing.
- **Steps**:
  1. Click "Cancel" in the confirmation panel
- **Expected**: The confirmation panel closes; the skill remains in the grid.

#### 9.10 — Skill card shows "Has content" badge

- **Pre-conditions**: A skill with non-empty `content` field exists.
- **Steps**:
  1. Navigate to `/skills`
- **Expected**: The skill card displays a green "Has content" badge.

#### 9.11 — Close skill modal with X button or backdrop click

- **Pre-conditions**: Skill modal is open.
- **Steps**:
  1. Click the X button in the modal header, or click the dark backdrop
- **Expected**: The modal closes without saving.

---

### 10. Statistics

#### 10.1 — Statistics page shows summary cards

- **Pre-conditions**: A project with tasks in multiple columns exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/statistics`
  2. Wait for data to load
- **Expected**: Four summary stat cards are visible: "Total Tasks", "Done", "In Progress", "Blocked".

#### 10.2 — Token Usage section shows totals

- **Pre-conditions**: Tasks with recorded token usage exist.
- **Steps**:
  1. Navigate to the Statistics page
  2. Scroll to the "Token Usage" section
- **Expected**: Input, Output, Cache Read, and Cache Write token counts are displayed.

#### 10.3 — Token Usage empty state

- **Pre-conditions**: No token usage has been recorded.
- **Steps**:
  1. Navigate to the Statistics page
- **Expected**: "No token usage recorded yet." message is shown in the Token Usage section.

#### 10.4 — MCP Tool Calls section shows tool usage bars

- **Pre-conditions**: MCP tools have been called at least once.
- **Steps**:
  1. Navigate to the Statistics page
  2. Scroll to the "MCP Tool Calls" section
- **Expected**: A list of tool names with bar charts and call counts is shown; total call count is displayed.

#### 10.5 — Tasks by Priority section

- **Pre-conditions**: Tasks with different priorities exist.
- **Steps**:
  1. Navigate to the Statistics page
- **Expected**: Priority breakdown chips (critical, high, medium, low) are shown with counts.

#### 10.6 — Tasks by Role section

- **Pre-conditions**: Tasks assigned to different roles exist.
- **Steps**:
  1. Navigate to the Statistics page
- **Expected**: Role breakdown chips are shown with task counts per role.

#### 10.7 — Timing section shows when tasks have duration data

- **Pre-conditions**: At least one task has `duration_seconds > 0`.
- **Steps**:
  1. Navigate to the Statistics page
- **Expected**: Avg Solve Time, Total Duration cards are visible; fastest/slowest task cards may appear.

#### 10.8 — Activity time range selector changes the chart data

- **Pre-conditions**: Activity data exists over more than 7 days.
- **Steps**:
  1. Navigate to the Statistics page
  2. Click the "7d" time range button
  3. Click the "30d" time range button
- **Expected**: The velocity and burndown charts update to reflect the selected time range.

#### 10.9 — Velocity chart renders bars per day

- **Pre-conditions**: Tasks were completed in recent days.
- **Steps**:
  1. Navigate to the Statistics page
  2. Scroll to the "Velocity — tasks completed per day" chart
- **Expected**: A bar chart with date labels and completion counts per day is rendered.

#### 10.10 — Burndown chart renders an SVG line chart

- **Pre-conditions**: Activity data exists.
- **Steps**:
  1. Navigate to the Statistics page
  2. Scroll to the "Burndown — remaining tasks over time" section
- **Expected**: An SVG chart with a colored line, area fill, and axis labels is visible.

#### 10.11 — Cold Start Cost per Agent Role table

- **Pre-conditions**: Cold start stats are available.
- **Steps**:
  1. Navigate to the Statistics page
  2. Scroll to the "Cold Start Cost per Agent Role" section
- **Expected**: A table with columns (Role, Runs, Min Input, Avg Input, Max Input, Avg Cache Read) is shown.

#### 10.12 — Statistics auto-refresh on WebSocket events

- **Pre-conditions**: Statistics page is open; another client or agent creates/completes a task.
- **Steps**:
  1. While viewing the Statistics page, trigger a `task_completed` event (e.g. complete a task via the API)
- **Expected**: The statistics update automatically without a manual page refresh.

---

### 11. Export (Claude / Gemini)

#### 11.1 — Export to Claude page renders "coming soon" message

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/export/claude`
- **Expected**: The page title "Export to Claude Code" is shown; a "coming soon" description is visible; a "Back to Project" link is displayed.

#### 11.2 — Export to Claude: "Back to Project" link navigates correctly

- **Pre-conditions**: On the Export Claude page.
- **Steps**:
  1. Click the "Back to Project" link
- **Expected**: URL returns to `/projects/:projectId`.

#### 11.3 — Export to Gemini page renders "coming soon" message

- **Pre-conditions**: A project exists.
- **Steps**:
  1. Navigate to `/projects/:projectId/export/gemini`
- **Expected**: The page title "Export to Gemini" is shown; a "coming soon" description is visible; a "Back to Project" link is displayed.

#### 11.4 — Export to Gemini: "Back to Project" link navigates correctly

- **Pre-conditions**: On the Export Gemini page.
- **Steps**:
  1. Click the "Back to Project" link
- **Expected**: URL returns to `/projects/:projectId`.

---

### 12. Theme Toggle

#### 12.1 — Theme toggle button is visible

- **Pre-conditions**: App is loaded on any page.
- **Steps**:
  1. Observe the layout (sidebar or header) for the theme toggle
- **Expected**: A theme toggle button (sun/moon icon or similar) is visible.

#### 12.2 — Toggle from dark to light theme

- **Pre-conditions**: App is in dark theme (default).
- **Steps**:
  1. Click `[data-qa="theme-toggle"]`
- **Expected**: The page background changes to a light color scheme; CSS variables switch to light-mode values.

#### 12.3 — Toggle from light to dark theme

- **Pre-conditions**: App is in light theme.
- **Steps**:
  1. Click `[data-qa="theme-toggle"]`
- **Expected**: The page background changes to the dark color scheme.

#### 12.4 — Theme preference persists across page navigation

- **Pre-conditions**: App is switched to light theme.
- **Steps**:
  1. Switch to light theme
  2. Navigate to a different route (e.g. `/roles`)
- **Expected**: The light theme is still applied on the new page.

---

### 13. Comments

#### 13.1 — Comments load when Task Drawer is opened

- **Pre-conditions**: A task with existing comments.
- **Steps**:
  1. Open the Task Drawer for the task
  2. Scroll to the Comments section
- **Expected**: Comments are listed with author avatar, name, author_type label, and relative timestamp.

#### 13.2 — Post a new comment

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Scroll to the comment compose area
  2. Type `"This is a test comment"` in `[data-qa="comment-input"]`
  3. Click `[data-qa="send-comment-btn"]` or press Ctrl+Enter
- **Expected**: The comment appears in the list immediately; the compose area is cleared.

#### 13.3 — Comment "Post as" selector is set to "Human" by default

- **Pre-conditions**: Task Drawer is open, comment section is visible.
- **Steps**:
  1. Observe `[data-qa="comment-author-select"]`
- **Expected**: "Human" is selected as the default author.

#### 13.4 — Send comment with Ctrl+Enter keyboard shortcut

- **Pre-conditions**: Comment textarea has content.
- **Steps**:
  1. Type text in the comment input
  2. Press `Ctrl+Enter` (or `Cmd+Enter` on Mac)
- **Expected**: Comment is posted (same result as clicking Send).

#### 13.5 — Image upload button opens file picker in comments

- **Pre-conditions**: Task Drawer is open with comments section visible.
- **Steps**:
  1. Click the "Image" button in the comment toolbar
- **Expected**: The file picker dialog opens accepting image files.

#### 13.6 — Drag-and-drop image into comment textarea

- **Pre-conditions**: Task Drawer is open.
- **Steps**:
  1. Drag an image file onto the comment textarea
- **Expected**: The textarea border highlights (indicating drag-over); after dropping, a Markdown image reference `![](<url>)` is inserted.

#### 13.7 — Empty comment cannot be submitted

- **Pre-conditions**: Task Drawer is open; comment textarea is empty.
- **Steps**:
  1. Observe the Send button
- **Expected**: The Send button is disabled when the comment content is empty.

---

### 14. Real-time WebSocket Updates

#### 14.1 — New task appears on board without refresh

- **Pre-conditions**: Two browser tabs are open on the same board.
- **Steps**:
  1. In Tab 1, open the kanban board
  2. In Tab 2, create a new task via the UI
- **Expected**: The new task card appears in Tab 1's Todo column without a manual refresh.

#### 14.2 — Task move is reflected in all connected clients

- **Pre-conditions**: Two browser tabs are open on the same board.
- **Steps**:
  1. In Tab 1, view the board
  2. In Tab 2, move a task from Todo to In Progress
- **Expected**: The task card moves in Tab 1 automatically.

#### 14.3 — Task deletion is reflected in all connected clients

- **Pre-conditions**: Two browser tabs are open on the same board.
- **Steps**:
  1. In Tab 1, view the board
  2. In Tab 2, delete a task
- **Expected**: The task card disappears from Tab 1's board.

---

### 15. API Health Check

#### 15.1 — Health endpoint returns 200

- **Pre-conditions**: Server is running.
- **Steps**:
  1. Send a GET request to `/api/projects`
- **Expected**: HTTP 200 response with a JSON body.

#### 15.2 — Server returns correct Content-Type for API responses

- **Pre-conditions**: Server is running.
- **Steps**:
  1. Send a GET request to `/api/projects`
- **Expected**: Response header `Content-Type: application/json` is present.
