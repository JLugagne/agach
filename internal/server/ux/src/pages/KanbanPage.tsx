import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { Plus, FolderTree, X, Search } from 'lucide-react';
import { getBoard, getProject, getTask, createTask, listProjectAgents, listFeatures, moveTask as apiMoveTask, markTaskSeen, updateTask } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type {
  BoardResponse,
  ColumnWithTasksResponse,
  ProjectResponse,
  FeatureWithSummaryResponse,
  AgentResponse,
  TaskWithDetailsResponse,
} from '../lib/types';
import Column from '../components/kanban/Column';
import TaskDrawer from '../components/kanban/TaskDrawer';
import TaskContextMenu from '../components/kanban/TaskContextMenu';
import TaskActions from '../components/kanban/TaskActions';
import NewTaskModal from '../components/kanban/NewTaskModal';
import BulkActionsBar from '../components/kanban/BulkActionsBar';
import FeatureDetailDrawer from '../components/kanban/FeatureDetailDrawer';

const DONE_FILTER_OPTIONS = [
  { label: 'Last 1h', value: '1h' },
  { label: 'Last 2h', value: '2h' },
  { label: 'Last 4h', value: '4h' },
  { label: 'Last 8h', value: '8h' },
  { label: 'Last 24h', value: '24h' },
  { label: 'Last 3 days', value: '72h' },
  { label: 'Last 7 days', value: '168h' },
  { label: 'Last 2 weeks', value: '336h' },
  { label: 'Last month', value: '720h' },
  { label: 'All', value: '' },
];

export default function KanbanPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [board, setBoard] = useState<BoardResponse | null>(null);
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [parentProject, setParentProject] = useState<ProjectResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [doneSince, setDoneSince] = useState('');

  const [features, setFeatures] = useState<FeatureWithSummaryResponse[]>([]);
  const [selectedFeature, setSelectedFeature] = useState<FeatureWithSummaryResponse | null>(null);

  // Filter state
  const [includeChildren, setIncludeChildren] = useState(true);
  const [roles, setRoles] = useState<AgentResponse[]>([]);
  const [selectedRoles, setSelectedRoles] = useState<Set<string>>(new Set());

  // Role color lookup map
  const roleColorMap = useMemo((): Record<string, string> => {
    const map: Record<string, string> = {};
    for (const r of roles) map[r.slug] = r.color;
    return map;
  }, [roles]);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [childProjectIds, _setChildProjectIds] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  // Track task column positions to detect moves and highlight
  const [highlightedTaskIds, setHighlightedTaskIds] = useState<Set<string>>(new Set());
  const prevTaskColumnsRef = useRef<Map<string, string>>(new Map());

  // Search input ref for keyboard shortcut focus
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Keyboard shortcuts help overlay
  const [showShortcutsHelp, setShowShortcutsHelp] = useState(false);

  // Multi-select state
  const [selectedTaskIds, setSelectedTaskIds] = useState<Set<string>>(new Set());

  // Build a map of taskId -> columnSlug for the current board
  const taskColumnMap = useMemo((): Map<string, string> => {
    const map = new Map<string, string>();
    if (!board) return map;
    for (const col of board.columns) {
      for (const task of col.tasks || []) {
        map.set(task.id, col.slug);
      }
    }
    return map;
  }, [board]);

  // Build a flat list of selected TaskWithDetailsResponse objects
  const selectedTaskObjects = useMemo((): TaskWithDetailsResponse[] => {
    if (!board || selectedTaskIds.size === 0) return [];
    const result: TaskWithDetailsResponse[] = [];
    for (const col of board.columns) {
      for (const task of col.tasks || []) {
        if (selectedTaskIds.has(task.id)) result.push(task);
      }
    }
    return result;
  }, [board, selectedTaskIds]);

  // Drawer state — driven by URL ?task= param
  const selectedTaskId = searchParams.get('task');
  const setSelectedTaskId = useCallback(
    (taskId: string | null) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (taskId) {
            next.set('task', taskId);
          } else {
            next.delete('task');
          }
          return next;
        },
      );
    },
    [setSearchParams],
  );

  // New task modal
  const [showNewTask, setShowNewTask] = useState(false);

  // Context menu
  const [contextMenu, setContextMenu] = useState<{
    task: TaskWithDetailsResponse;
    column: string;
    position: { x: number; y: number };
  } | null>(null);

  // Task action modal (block, delete, complete, etc.)
  const [actionState, setActionState] = useState<{
    task: TaskWithDetailsResponse;
    action: string;
  } | null>(null);

  // Detect tasks that moved columns and highlight them
  useEffect(() => {
    if (!board) return;
    const prev = prevTaskColumnsRef.current;
    const curr = new Map<string, string>();
    const moved = new Set<string>();

    for (const col of board.columns) {
      for (const task of col.tasks || []) {
        curr.set(task.id, col.id);
        const prevCol = prev.get(task.id);
        // Only highlight if the task was known before AND changed column
        if (prevCol !== undefined && prevCol !== col.id) {
          moved.add(task.id);
        }
      }
    }

    prevTaskColumnsRef.current = curr;

    if (moved.size > 0) {
      setHighlightedTaskIds(moved);
      const timer = setTimeout(() => setHighlightedTaskIds(new Set()), 2000);
      return () => clearTimeout(timer);
    }
  }, [board]);

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Check if an input-like element is focused — skip shortcuts in that case,
      // except for Escape which should always work.
      const tag = (document.activeElement as HTMLElement)?.tagName?.toLowerCase();
      const isInputFocused = tag === 'input' || tag === 'textarea' || tag === 'select';

      if (e.key === 'Escape') {
        // Priority: close help overlay → close drawer → clear selection → close modal
        if (showShortcutsHelp) {
          setShowShortcutsHelp(false);
          return;
        }
        if (selectedTaskId) {
          setSelectedTaskId(null);
          return;
        }
        if (selectedTaskIds.size > 0) {
          setSelectedTaskIds(new Set());
          return;
        }
        if (actionState) {
          setActionState(null);
          return;
        }
        if (showNewTask) {
          setShowNewTask(false);
          return;
        }
        return;
      }

      // All other shortcuts are skipped when an input is focused
      if (isInputFocused) return;

      if (e.key === '/') {
        e.preventDefault();
        searchInputRef.current?.focus();
        return;
      }

      if (e.key === '?') {
        e.preventDefault();
        setShowShortcutsHelp((prev) => !prev);
        return;
      }
    };

    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [showShortcutsHelp, selectedTaskId, selectedTaskIds, actionState, showNewTask, setSelectedTaskId]);

  const fetchBoard = useCallback(async () => {
    if (!projectId) return;
    try {
      const data = await getBoard(projectId, doneSince || undefined, true, debouncedSearch || undefined);
      setBoard(data);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  }, [projectId, doneSince, debouncedSearch]);

  const fetchProject = useCallback(async () => {
    if (!projectId) return;
    try {
      const proj = await getProject(projectId);
      setProject(proj);
      if (proj.parent_id) {
        try {
          const parent = await getProject(proj.parent_id);
          setParentProject(parent);
        } catch {
          /* ignore */
        }
      } else {
        setParentProject(null);
      }
    } catch {
      /* ignore */
    }
  }, [projectId]);

  // Fetch roles and child project IDs
  useEffect(() => {
    if (!projectId) return;
    listProjectAgents(projectId).then((r) => setRoles(r ?? [])).catch(() => {});
  }, [projectId]);

  const fetchFeatures = useCallback(async () => {
    if (!projectId) return;
    try {
      const data = await listFeatures(projectId);
      setFeatures(data ?? []);
    } catch {
      setFeatures([]);
    }
  }, [projectId]);

  useEffect(() => {
    if (!projectId) return;
    fetchFeatures();
  }, [projectId, fetchFeatures]);

  useEffect(() => {
    fetchBoard();
    fetchProject();
  }, [fetchBoard, fetchProject]);

  // WebSocket: refetch board on task events (including child projects)
  useWebSocket(
    useCallback(
      (event) => {
        if (!projectId) return;
        const type = event.type || '';
        const eventProjectId = event.project_id;
        if (!eventProjectId) return;

        // Accept events from this project OR any child project
        const isRelevant = eventProjectId === projectId || childProjectIds.has(eventProjectId);
        if (!isRelevant) return;

        // Handle task_seen for multi-tab sync: update seen_at in local state without refetch
        if (type === 'task_seen') {
          const data = event.data as { task_id?: string; seen_at?: string };
          if (data?.task_id && data?.seen_at) {
            setBoard((prev) => {
              if (!prev) return prev;
              return {
                ...prev,
                columns: prev.columns.map((c) =>
                  c.slug !== 'done'
                    ? c
                    : {
                        ...c,
                        tasks: c.tasks.map((t) =>
                          t.id === data.task_id ? { ...t, seen_at: data.seen_at! } : t,
                        ),
                      },
                ),
              };
            });
          }
          return;
        }

        // Refetch on any task-related or project event
        if (
          type.startsWith('task_') ||
          type.startsWith('comment_') ||
          type === 'project_updated'
        ) {
          fetchBoard();
          if (type.startsWith('task_')) {
            fetchFeatures();
          }
        }
      },
      [projectId, fetchBoard, fetchFeatures, childProjectIds],
    ),
  );

  // Apply client-side filters to the board
  const filteredBoard = useMemo((): BoardResponse | null => {
    if (!board) return null;

    const filterTasks = (tasks: TaskWithDetailsResponse[]): TaskWithDetailsResponse[] => {
      let filtered = tasks;

      // Filter out sub-project tasks if include_children is off
      if (!includeChildren) {
        filtered = filtered.filter((t) => !t.project_id);
      }

      // Filter by selected roles
      if (selectedRoles.size > 0) {
        filtered = filtered.filter((t) => t.assigned_role && selectedRoles.has(t.assigned_role));
      }

      return filtered;
    };

    return {
      ...board,
      columns: board.columns
        .filter((col) => col.slug !== 'backlog')
        .map((col): ColumnWithTasksResponse => ({
          ...col,
          tasks: filterTasks(col.tasks || []),
        })),
    };
  }, [board, includeChildren, selectedRoles]);

  const toggleRole = (slug: string) => {
    setSelectedRoles((prev) => {
      const next = new Set(prev);
      if (next.has(slug)) {
        next.delete(slug);
      } else {
        next.add(slug);
      }
      return next;
    });
  };

  // Handle task selection (ctrl+click multi-select)
  const handleTaskSelect = useCallback((taskId: string, ctrlKey: boolean) => {
    if (ctrlKey) {
      setSelectedTaskIds((prev) => {
        const next = new Set(prev);
        if (next.has(taskId)) {
          next.delete(taskId);
        } else {
          next.add(taskId);
        }
        return next;
      });
    } else {
      // Normal click: clear selection
      setSelectedTaskIds(new Set());
    }
  }, []);

  const handleTaskClick = (task: TaskWithDetailsResponse) => {
    setSelectedTaskId(task.id);
    // Mark done-column tasks as seen when the drawer is opened
    const col = board?.columns.find((c) => c.tasks?.some((t) => t.id === task.id));
    if (col?.slug === 'done' && task.seen_at === null) {
      // Fire-and-forget: persist seen_at on the server (use task's project_id for sub-project tasks)
      const taskProjId = task.project_id || projectId!;
      markTaskSeen(taskProjId, task.id).catch(() => { /* ignore */ });
      // Optimistically set seen_at in local state so badge disappears immediately
      setBoard((prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          columns: prev.columns.map((c) =>
            c.slug !== 'done'
              ? c
              : {
                  ...c,
                  tasks: c.tasks.map((t) =>
                    t.id === task.id ? { ...t, seen_at: new Date().toISOString() } : t,
                  ),
                },
          ),
        };
      });
    }
  };

  const handleTaskContextMenu = (
    task: TaskWithDetailsResponse,
    e: React.MouseEvent,
  ) => {
    // Find column slug for this task
    const col = board?.columns.find((c) =>
      c.tasks?.some((t) => t.id === task.id),
    );
    setContextMenu({
      task,
      column: col?.slug || 'todo',
      position: { x: e.clientX, y: e.clientY },
    });
  };

  const handleContextMenuAction = async (action: string) => {
    if (!contextMenu || !projectId) return;
    const { task } = contextMenu;
    const taskProjId = task.project_id || projectId;

    // Priority actions
    if (action.startsWith('priority_')) {
      const priority = action.replace('priority_', '');
      try {
        await updateTask(taskProjId, task.id, { priority });
        fetchBoard();
      } catch {
        /* ignore */
      }
      return;
    }

    // Role assign/unassign actions
    if (action.startsWith('role_')) {
      const role = action === 'role_unassign' ? '' : action.replace('role_', '');
      try {
        await updateTask(taskProjId, task.id, { assigned_role: role });
        fetchBoard();
      } catch {
        /* ignore */
      }
      return;
    }

    // Duplicate action
    if (action === 'duplicate') {
      try {
        const fullTask = await getTask(taskProjId, task.id);
        await createTask(taskProjId, {
          title: `Copy of ${fullTask.title}`,
          summary: fullTask.summary,
          description: fullTask.description,
          priority: fullTask.priority,
          assigned_role: fullTask.assigned_role,
          tags: fullTask.tags,
          context_files: fullTask.context_files,
          estimated_effort: fullTask.estimated_effort,
        });
        fetchBoard();
      } catch {
        /* ignore */
      }
      return;
    }

    // Actions that open modals
    if (['block', 'unblock', 'wontdo', 'delete', 'complete', 'move_to_project'].includes(action)) {
      setActionState({ task, action });
      return;
    }

    // Actions that open the drawer in edit mode (for now just open drawer)
    if (action === 'edit') {
      setSelectedTaskId(task.id);
      return;
    }

    if (action === 'move_in_progress') {
      try {
        await apiMoveTask(taskProjId, task.id, { target_column: 'in_progress' });
        fetchBoard();
      } catch {
        /* ignore */
      }
      return;
    }

    if (action === 'move_todo') {
      try {
        await apiMoveTask(taskProjId, task.id, { target_column: 'todo' });
        fetchBoard();
      } catch {
        /* ignore */
      }
      return;
    }
  };

  const handleDrawerAction = (action: string) => {
    if (!selectedTaskId || !board) return;
    // Find the task from the board
    let task: TaskWithDetailsResponse | null = null;
    for (const col of board.columns) {
      const found = col.tasks?.find((t) => t.id === selectedTaskId);
      if (found) {
        task = found;
        break;
      }
    }
    if (task) {
      setActionState({ task, action });
    }
  };

  const handleActionSuccess = () => {
    setActionState(null);
    fetchBoard();
  };

  if (!projectId) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-[var(--text-muted)] text-sm font-['Inter']">No project selected.</p>
      </div>
    );
  }

  const hasChildren = childProjectIds.size > 0;
  const hasActiveFilters = selectedRoles.size > 0 || !includeChildren || searchQuery !== '';

  return (
    <div className="flex flex-col h-full bg-[var(--bg-primary)]">
      {/* Top bar */}
      <div className="flex items-center justify-between px-4 sm:px-8 py-4 sm:py-5 flex-shrink-0">
        <div>
          {parentProject ? (
            <>
              <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Project Board
              </h1>
              <p className="text-sm text-[var(--text-muted)] mt-0.5" style={{ fontFamily: 'Inter, sans-serif' }}>
                <Link
                  to={`/projects/${parentProject.id}`}
                  data-qa="kanban-parent-project-link"
                  className="hover:text-[var(--text-secondary)] transition-colors"
                >
                  {parentProject.name}
                </Link>
                {' \u00B7 '}
                <span className="text-[var(--text-secondary)]">{project?.name}</span>
              </p>
            </>
          ) : (
            <>
              <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Project Board
              </h1>
              <p className="text-sm text-[var(--text-muted)] mt-0.5" style={{ fontFamily: 'Inter, sans-serif' }}>
                {project?.name || 'Loading...'}
                {project?.description ? ` \u00B7 ${project.description.slice(0, 60)}${project.description.length > 60 ? '...' : ''}` : ''}
              </p>
            </>
          )}
        </div>
        <div className="flex items-center gap-3">
          {/* Search */}
          <div className="relative">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" />
            <input
              ref={searchInputRef}
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search tasks..."
              data-qa="search-input"
              className="w-52 pl-9 pr-8 py-2.5 bg-[var(--bg-tertiary)] border border-[var(--border-primary)] text-[var(--text-primary)] text-[13px] rounded-lg focus:outline-none focus:border-[var(--primary)] placeholder:text-[var(--text-muted)] transition-colors"
              style={{ fontFamily: 'Inter, sans-serif' }}
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                data-qa="search-clear-btn"
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
              >
                <X size={14} />
              </button>
            )}
          </div>

          {/* Filter button group */}
          <div className="flex items-center gap-1.5">
            {/* Sub-projects toggle */}
            {hasChildren && (
              <button
                onClick={() => setIncludeChildren((v) => !v)}
                data-qa="kanban-toggle-subprojects-btn"
                className={`flex items-center gap-1.5 px-3 py-2.5 rounded-lg text-[13px] transition-colors cursor-pointer border ${
                  includeChildren
                    ? 'bg-[var(--primary)]/10 text-[var(--primary)] border-[var(--primary)]/30'
                    : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--border-primary)]'
                }`}
                style={{ fontFamily: 'Inter, sans-serif' }}
                title={includeChildren ? 'Showing sub-project tasks' : 'Sub-project tasks hidden'}
              >
                <FolderTree size={14} />
              </button>
            )}

            {/* Role filters */}
            {roles.map((role) => {
              const isActive = selectedRoles.has(role.slug);
              return (
                <button
                  key={role.slug}
                  onClick={() => toggleRole(role.slug)}
                  data-qa="kanban-role-filter-btn"
                  className={`flex items-center gap-1 px-2.5 py-2.5 rounded-lg text-[11px] font-medium uppercase tracking-wider transition-colors cursor-pointer border ${
                    isActive
                      ? 'border-current bg-[var(--bg-tertiary)]'
                      : 'border-transparent hover:bg-[var(--bg-tertiary)]'
                  }`}
                  style={{
                    color: isActive ? role.color : 'var(--text-muted)',
                    fontFamily: 'JetBrains Mono, monospace',
                  }}
                  title={role.name}
                >
                  {role.icon && <span>{role.icon}</span>}
                  {role.name}
                </button>
              );
            })}
            {hasActiveFilters && (
              <button
                onClick={() => {
                  setSelectedRoles(new Set());
                  setIncludeChildren(true);
                  setSearchQuery('');
                }}
                data-qa="kanban-clear-filters-btn"
                className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors ml-0.5 p-1"
                title="Clear all filters"
              >
                <X size={14} />
              </button>
            )}
          </div>

          {/* Done filter */}
          <select
            value={doneSince}
            onChange={(e) => setDoneSince(e.target.value)}
            data-qa="done-filter-select"
            className="bg-[var(--bg-tertiary)] border border-[var(--border-primary)] text-[var(--text-secondary)] text-[13px] rounded-lg px-3 py-2.5 focus:outline-none focus:border-[var(--primary)] cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            {DONE_FILTER_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>

          <button
            onClick={() => setShowNewTask(true)}
            data-qa="new-task-btn"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Task
          </button>
        </div>
      </div>

      {/* Board */}
      {loading ? (
        <div className="flex items-center justify-center flex-1">
          <div className="w-6 h-6 border-2 border-[var(--primary)] border-t-transparent rounded-full animate-spin" />
        </div>
      ) : !filteredBoard || filteredBoard.columns.length === 0 ? (
        <div className="flex items-center justify-center flex-1">
          <p className="text-[var(--text-dim)] text-sm font-['Inter']">No board data available.</p>
        </div>
      ) : (
        <div className="flex gap-5 px-4 sm:px-8 pb-6 flex-1 overflow-x-auto overflow-y-hidden min-h-0">
          {filteredBoard.columns.map((col) => {
            const displayCol =
              col.slug === 'done' && col.tasks && col.tasks.length > 0
                ? {
                    ...col,
                    tasks: [...col.tasks].sort((a, b) => {
                      if (b.priority_score !== a.priority_score) {
                        return b.priority_score - a.priority_score;
                      }
                      return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
                    }),
                  }
                : col;
            return (
              <Column
                key={col.id}
                column={displayCol}
                projectId={projectId}
                roleColorMap={roleColorMap}
                onTaskClick={handleTaskClick}
                onTaskContextMenu={handleTaskContextMenu}
                isTaskNew={col.slug === 'done' ? (taskId) => {
                  // Check from unfiltered board to get correct seen_at
                  const origCol = board?.columns.find((c) => c.slug === 'done');
                  const task = origCol?.tasks?.find((t) => t.id === taskId);
                  return task?.seen_at === null;
                } : undefined}
                isTaskHighlighted={(taskId) => highlightedTaskIds.has(taskId)}
                isTaskSelected={(taskId) => selectedTaskIds.has(taskId)}
                onTaskSelect={handleTaskSelect}
                onRefresh={fetchBoard}
                onAddTask={col.slug === 'todo' ? () => setShowNewTask(true) : undefined}
              />
            );
          })}
        </div>
      )}

      {/* Context menu */}
      {contextMenu && (
        <TaskContextMenu
          task={contextMenu.task}
          column={contextMenu.column}
          position={contextMenu.position}
          projectId={projectId}
          roles={roles.map((r) => ({ slug: r.slug, name: r.name, color: r.color }))}
          onClose={() => setContextMenu(null)}
          onAction={handleContextMenuAction}
        />
      )}

      {/* Task drawer */}
      {selectedTaskId && board && (() => {
        // Find the task's actual project ID (may differ for sub-project tasks)
        let taskProjectId = projectId;
        for (const col of board.columns) {
          const found = col.tasks?.find((t) => t.id === selectedTaskId);
          if (found?.project_id) {
            taskProjectId = found.project_id;
            break;
          }
        }
        return (
          <TaskDrawer
            projectId={taskProjectId}
            taskId={selectedTaskId}
            columns={board.columns}
            features={features}
            onClose={() => setSelectedTaskId(null)}
            onAction={handleDrawerAction}
            onTaskNavigate={(taskId) => setSelectedTaskId(taskId)}
          />
        );
      })()}

      {/* Feature detail drawer */}
      {selectedFeature && (
        <FeatureDetailDrawer
          feature={selectedFeature}
          projectId={projectId}
          onClose={() => setSelectedFeature(null)}
          onTaskClick={(taskId: string, _taskProjectId: string) => {
            setSelectedFeature(null);
            setSelectedTaskId(taskId);
          }}
        />
      )}

      {/* New task modal */}
      {showNewTask && (
        <NewTaskModal
          projectId={projectId}
          defaultRole={project?.default_role}
          features={features}
          onClose={() => setShowNewTask(false)}
          onSuccess={() => {
            setShowNewTask(false);
            fetchBoard();
          }}
        />
      )}

      {/* Task action modals (block, delete, complete, etc.) */}
      <TaskActions
        projectId={projectId}
        task={actionState?.task || null}
        action={actionState?.action || null}
        onClose={() => setActionState(null)}
        onSuccess={handleActionSuccess}
      />

      {/* Bulk actions bar */}
      {selectedTaskIds.size > 0 && (
        <BulkActionsBar
          selectedTasks={selectedTaskObjects}
          columnMap={taskColumnMap}
          projectId={projectId}
          onClear={() => setSelectedTaskIds(new Set())}
          onRefresh={fetchBoard}
        />
      )}

      {/* Keyboard shortcuts help overlay */}
      {showShortcutsHelp && (
        <>
          {/* Backdrop to dismiss on click outside */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setShowShortcutsHelp(false)}
          />
          <div className="fixed bottom-5 right-5 z-50 bg-[var(--bg-elevated)]/95 backdrop-blur-sm border border-[var(--border-primary)] rounded-lg shadow-xl p-4 min-w-[200px]">
            <div className="flex items-center justify-between mb-3">
              <span className="text-[var(--text-primary)] text-xs font-['Inter'] font-semibold uppercase tracking-wider">Shortcuts</span>
              <button
                onClick={() => setShowShortcutsHelp(false)}
                data-qa="kanban-shortcuts-close-btn"
                className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
              >
                <X size={12} />
              </button>
            </div>
            <div className="flex flex-col gap-2">
              {[
                { key: '/', description: 'Focus search' },
                { key: 'Esc', description: 'Close / deselect' },
                { key: '?', description: 'Toggle this panel' },
              ].map(({ key, description }) => (
                <div key={key} className="flex items-center justify-between gap-6">
                  <span className="text-[var(--text-muted)] text-xs font-['Inter']">{description}</span>
                  <kbd className="inline-flex items-center justify-center px-1.5 py-0.5 bg-[var(--bg-tertiary)] border border-[var(--border-secondary)] rounded text-[10px] font-['JetBrains_Mono'] text-[var(--text-secondary)] min-w-[28px] text-center">
                    {key}
                  </kbd>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
