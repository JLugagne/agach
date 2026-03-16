import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { Plus, ChevronRight, FolderTree, X, Search } from 'lucide-react';
import { getBoard, getProject, listRoles, listSubProjects, moveTask as apiMoveTask, markTaskSeen } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type {
  BoardResponse,
  ColumnWithTasksResponse,
  ProjectResponse,
  RoleResponse,
  TaskWithDetailsResponse,
} from '../lib/types';
import Column from '../components/kanban/Column';
import TaskDrawer from '../components/kanban/TaskDrawer';
import TaskContextMenu from '../components/kanban/TaskContextMenu';
import TaskActions from '../components/kanban/TaskActions';
import NewTaskModal from '../components/kanban/NewTaskModal';

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

  // Filter state
  const [includeChildren, setIncludeChildren] = useState(true);
  const [roles, setRoles] = useState<RoleResponse[]>([]);
  const [selectedRoles, setSelectedRoles] = useState<Set<string>>(new Set());
  const [childProjectIds, setChildProjectIds] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  // Track task column positions to detect moves and highlight
  const [highlightedTaskIds, setHighlightedTaskIds] = useState<Set<string>>(new Set());
  const prevTaskColumnsRef = useRef<Map<string, string>>(new Map());

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
    listRoles().then(setRoles).catch(() => {});
  }, []);

  useEffect(() => {
    if (!projectId) return;
    listSubProjects(projectId).then((children) => {
      setChildProjectIds(new Set(children.map((c) => c.id)));
    }).catch(() => setChildProjectIds(new Set()));
  }, [projectId]);

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
        }
      },
      [projectId, fetchBoard, childProjectIds],
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
      columns: board.columns.map((col): ColumnWithTasksResponse => ({
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

    // Actions that open modals
    if (['block', 'unblock', 'wontdo', 'delete', 'complete'].includes(action)) {
      setActionState({ task, action });
      return;
    }

    // Actions that open the drawer in edit mode (for now just open drawer)
    if (action === 'edit') {
      setSelectedTaskId(task.id);
      return;
    }

    // Move actions — use task's actual project ID
    const taskProjId = task.project_id || projectId;

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
      <div className="flex items-center justify-between px-6 h-[60px] bg-[var(--bg-secondary)] border-b border-[var(--border-primary)] flex-shrink-0">
        <div className="flex items-center gap-2">
          {parentProject ? (
            <>
              <Link
                to={`/projects/${parentProject.id}`}
                className="text-[var(--text-secondary)] text-lg font-['Newsreader'] hover:text-[var(--text-primary)] transition-colors"
              >
                {parentProject.name}
              </Link>
              <ChevronRight size={16} className="text-[var(--text-muted)]" />
              <span className="text-[var(--primary)] text-base font-['Newsreader']">
                {project?.name || 'Loading...'}
              </span>
            </>
          ) : (
            <span className="text-[var(--text-primary)] text-lg font-['Newsreader']">
              {project?.name || 'Loading...'}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {/* Search */}
          <div className="relative">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search tasks..."
              className="w-48 pl-8 pr-8 py-1.5 bg-[var(--bg-elevated)] border border-[var(--border-secondary)] text-[var(--text-primary)] text-xs font-['Inter'] rounded-md focus:outline-none focus:border-[var(--primary)] placeholder:text-[var(--text-muted)] transition-colors"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
              >
                <X size={12} />
              </button>
            )}
          </div>

          {/* Sub-projects toggle */}
          {hasChildren && (
            <button
              onClick={() => setIncludeChildren((v) => !v)}
              className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-['JetBrains_Mono'] uppercase tracking-wider transition-colors ${
                includeChildren
                  ? 'bg-[#8B5CF615] text-[#8B5CF6] border border-[#8B5CF630]'
                  : 'bg-[var(--bg-elevated)] text-[var(--text-muted)] border border-[var(--border-secondary)]'
              }`}
              title={includeChildren ? 'Showing sub-project tasks' : 'Sub-project tasks hidden'}
            >
              <FolderTree size={12} />
              Subs
            </button>
          )}

          {/* Role filters */}
          <div className="flex items-center gap-1">
            {roles.map((role) => {
              const isActive = selectedRoles.has(role.slug);
              return (
                <button
                  key={role.slug}
                  onClick={() => toggleRole(role.slug)}
                  className={`flex items-center gap-1 px-2 py-1.5 rounded-md text-[10px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider transition-colors border ${
                    isActive
                      ? 'border-current bg-[var(--bg-elevated)]'
                      : 'border-transparent hover:bg-[var(--bg-tertiary)]'
                  }`}
                  style={{ color: isActive ? role.color : 'var(--text-secondary)' }}
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
                className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors ml-1"
                title="Clear all filters"
              >
                <X size={14} />
              </button>
            )}
          </div>

          {/* Done filter */}
          <div className="flex items-center gap-2">
            <span className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">Done</span>
            <select
              value={doneSince}
              onChange={(e) => setDoneSince(e.target.value)}
              className="bg-[var(--bg-elevated)] border border-[var(--border-secondary)] text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] rounded px-2 py-1.5 focus:outline-none focus:border-[var(--primary)] cursor-pointer"
            >
              {DONE_FILTER_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <button
            onClick={() => setShowNewTask(true)}
            className="flex items-center gap-1.5 px-4 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--primary-text)] text-sm font-['Inter'] font-medium rounded-md transition-colors"
          >
            <Plus size={16} />
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
        <div className="flex gap-4 p-[20px_24px] flex-1 overflow-x-auto overflow-y-hidden min-h-0">
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
                onTaskClick={handleTaskClick}
                onTaskContextMenu={handleTaskContextMenu}
                isTaskNew={col.slug === 'done' ? (taskId) => {
                  // Check from unfiltered board to get correct seen_at
                  const origCol = board?.columns.find((c) => c.slug === 'done');
                  const task = origCol?.tasks?.find((t) => t.id === taskId);
                  return task?.seen_at === null;
                } : undefined}
                isTaskHighlighted={(taskId) => highlightedTaskIds.has(taskId)}
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
            onClose={() => setSelectedTaskId(null)}
            onAction={handleDrawerAction}
            onTaskNavigate={(taskId) => setSelectedTaskId(taskId)}
          />
        );
      })()}

      {/* New task modal */}
      {showNewTask && (
        <NewTaskModal
          projectId={projectId}
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
    </div>
  );
}
