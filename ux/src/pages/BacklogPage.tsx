import { useState, useEffect, useCallback, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { Loader2, ArrowRight, ChevronDown } from 'lucide-react';
import { listTasks, moveTask, listSubProjects, getProjectSummary, listColumns } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { TaskWithDetailsResponse, ProjectWithSummary, ProjectSummaryResponse, ColumnWithTasksResponse } from '../lib/types';
import TaskDrawer from '../components/kanban/TaskDrawer';

export default function BacklogPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [tasks, setTasks] = useState<TaskWithDetailsResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [subProjects, setSubProjects] = useState<ProjectWithSummary[]>([]);
  const [selectedSubProject, setSelectedSubProject] = useState<string>('');
  const [summary, setSummary] = useState<ProjectSummaryResponse | null>(null);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [columns, setColumns] = useState<ColumnWithTasksResponse[]>([]);
  const [movingTaskId, setMovingTaskId] = useState<string | null>(null);

  const fetchTasks = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const params: Record<string, string> = { column: 'backlog', include_children: 'true' };
      const result = await listTasks(projectId, params);
      setTasks(result);
    } catch {
      setTasks([]);
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  const fetchSubProjects = useCallback(async () => {
    if (!projectId) return;
    try {
      const sps = await listSubProjects(projectId);
      setSubProjects(sps);
    } catch {
      setSubProjects([]);
    }
  }, [projectId]);

  const fetchSummary = useCallback(async () => {
    if (!projectId) return;
    try {
      const s = await getProjectSummary(projectId);
      setSummary(s);
    } catch {
      setSummary(null);
    }
  }, [projectId]);

  const fetchColumns = useCallback(async () => {
    if (!projectId) return;
    try {
      const cols = await listColumns(projectId);
      setColumns(cols.map((c) => ({ ...c, tasks: [] })));
    } catch {
      setColumns([]);
    }
  }, [projectId]);

  useEffect(() => {
    fetchTasks();
    fetchSubProjects();
    fetchSummary();
    fetchColumns();
  }, [fetchTasks, fetchSubProjects, fetchSummary, fetchColumns]);

  useWebSocket(
    useCallback(
      (event) => {
        if (event.type === 'task_created' || event.type === 'task_updated' || event.type === 'task_moved' || event.type === 'task_deleted') {
          fetchTasks();
          fetchSummary();
        }
      },
      [fetchTasks, fetchSummary],
    ),
  );

  const handleMoveToTodo = async (taskId: string) => {
    if (!projectId) return;
    setMovingTaskId(taskId);
    try {
      await moveTask(projectId, taskId, { target_column: 'todo' });
      fetchTasks();
      fetchSummary();
    } catch {
      // error handled silently
    } finally {
      setMovingTaskId(null);
    }
  };

  // Filter by selected sub-project
  const filteredTasks = useMemo(() => {
    if (!selectedSubProject) return tasks;
    return tasks.filter((t) => t.project_id === selectedSubProject);
  }, [tasks, selectedSubProject]);

  // Priority color
  const priorityColor = (priority: string) => {
    switch (priority) {
      case 'critical': return '#FF3B30';
      case 'high': return '#FF9500';
      case 'medium': return '#007AFF';
      case 'low': return 'var(--text-muted)';
      default: return 'var(--text-muted)';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-6 h-[60px] border-b border-[var(--border-primary)] flex-shrink-0">
        <div className="flex items-center gap-3">
          <h1 className="text-[17px] font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            Backlog
          </h1>
          <span className="text-[12px] px-2 py-0.5 rounded-full bg-[var(--bg-secondary)] text-[var(--text-muted)]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
            {filteredTasks.length}
          </span>
        </div>

        {/* Sub-project filter */}
        {subProjects.length > 0 && (
          <div className="relative">
            <select
              value={selectedSubProject}
              onChange={(e) => setSelectedSubProject(e.target.value)}
              className="appearance-none bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-1.5 pr-8 text-[13px] text-[var(--text-primary)] cursor-pointer focus:outline-none focus:ring-1 focus:ring-[var(--primary)]"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              <option value="">
                All projects {summary ? `(${summary.backlog_count})` : ''}
              </option>
              {subProjects.map((sp) => {
                const spSummary = sp.task_summary ?? sp.summary;
                const backlogCount = spSummary?.backlog_count ?? 0;
                return (
                  <option key={sp.id} value={sp.id}>
                    {sp.name} ({backlogCount})
                  </option>
                );
              })}
            </select>
            <ChevronDown size={14} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] pointer-events-none" />
          </div>
        )}
      </div>

      {/* Task list */}
      <div className="flex-1 overflow-y-auto">
        {filteredTasks.length === 0 ? (
          <div className="flex items-center justify-center h-full text-[var(--text-muted)] text-[14px]" style={{ fontFamily: 'Inter, sans-serif' }}>
            No tasks in backlog
          </div>
        ) : (
          <div className="divide-y divide-[var(--border-primary)]">
            {filteredTasks.map((task) => (
              <div
                key={task.id}
                className="flex items-center gap-4 px-6 py-3 hover:bg-[var(--bg-secondary)]/50 transition-colors group"
              >
                {/* Priority indicator */}
                <div
                  className="w-2 h-2 rounded-full flex-shrink-0"
                  style={{ backgroundColor: priorityColor(task.priority) }}
                  title={task.priority}
                />

                {/* Task info */}
                <button
                  className="flex-1 min-w-0 text-left cursor-pointer"
                  onClick={() => setSelectedTaskId(task.id)}
                >
                  <div className="text-[14px] text-[var(--text-primary)] truncate" style={{ fontFamily: 'Inter, sans-serif' }}>
                    {task.title}
                  </div>
                  <div className="text-[12px] text-[var(--text-muted)] truncate mt-0.5" style={{ fontFamily: 'Inter, sans-serif' }}>
                    {task.summary}
                  </div>
                </button>

                {/* Assigned role */}
                {task.assigned_role && (
                  <span className="text-[11px] px-2 py-0.5 rounded-full bg-[var(--bg-secondary)] text-[var(--text-muted)] flex-shrink-0" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                    {task.assigned_role}
                  </span>
                )}

                {/* Dependencies indicator */}
                {task.has_unresolved_deps && (
                  <span className="text-[11px] px-2 py-0.5 rounded-full bg-[var(--bg-secondary)] text-[var(--text-muted)] flex-shrink-0" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                    deps
                  </span>
                )}

                {/* Move to Todo button */}
                <button
                  onClick={() => handleMoveToTodo(task.id)}
                  disabled={movingTaskId === task.id}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium rounded-md bg-[var(--primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-50 opacity-0 group-hover:opacity-100 flex-shrink-0 cursor-pointer"
                  style={{ fontFamily: 'Inter, sans-serif' }}
                  title="Move to Todo"
                >
                  {movingTaskId === task.id ? (
                    <Loader2 size={12} className="animate-spin" />
                  ) : (
                    <ArrowRight size={12} />
                  )}
                  Todo
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Task drawer */}
      {selectedTaskId && projectId && (
        <TaskDrawer
          projectId={projectId}
          taskId={selectedTaskId}
          columns={columns}
          onClose={() => setSelectedTaskId(null)}
          onAction={() => { fetchTasks(); fetchSummary(); }}
        />
      )}
    </div>
  );
}
