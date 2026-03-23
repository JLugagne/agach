import { useEffect, useState } from 'react';
import { X, Loader2, ExternalLink, MessageCircle } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { FeatureWithSummaryResponse, FeatureStatus, TaskWithDetailsResponse } from '../../lib/types';
import { listTasks } from '../../lib/api';

interface FeatureDetailDrawerProps {
  feature: FeatureWithSummaryResponse;
  projectId: string;
  onClose: () => void;
  onTaskClick: (taskId: string, taskProjectId: string) => void;
}

const STATUS_BADGE_COLORS: Record<FeatureStatus, string> = {
  draft: 'var(--text-muted)',
  ready: 'var(--status-todo)',
  in_progress: 'var(--status-progress)',
  done: 'var(--status-done)',
  blocked: '#FF3B30',
};

// Status bar segments
const segments = [
  {
    key: 'todo' as const,
    label: 'todo',
    dot: 'var(--status-todo)',
    bg: 'color-mix(in srgb, var(--status-todo) 12%, transparent)',
  },
  {
    key: 'in_progress' as const,
    label: 'in progress',
    dot: 'var(--status-progress)',
    bg: 'color-mix(in srgb, var(--status-progress) 12%, transparent)',
  },
  {
    key: 'done' as const,
    label: 'done',
    dot: 'var(--status-done)',
    bg: 'color-mix(in srgb, var(--status-done) 12%, transparent)',
  },
  {
    key: 'blocked' as const,
    label: 'blocked',
    dot: 'var(--status-blocked)',
    bg: 'color-mix(in srgb, var(--status-blocked) 12%, transparent)',
  },
];

const countKeys = {
  todo: 'todo_count',
  in_progress: 'in_progress_count',
  done: 'done_count',
  blocked: 'blocked_count',
} as const;

// Priority pill vars
const priorityPillVars: Record<string, { text: string; bg: string }> = {
  critical: { text: 'var(--priority-critical)', bg: 'var(--priority-critical-bg)' },
  high: { text: 'var(--priority-high)', bg: 'var(--priority-high-bg)' },
  medium: { text: 'var(--priority-medium)', bg: 'var(--priority-medium-bg)' },
  low: { text: 'var(--priority-low)', bg: 'var(--priority-low-bg)' },
};

// Task groups in display order
const TASK_GROUPS: { slug: string; label: string; dot: string }[] = [
  { slug: 'in_progress', label: 'in progress', dot: 'var(--status-progress)' },
  { slug: 'todo', label: 'todo', dot: 'var(--status-todo)' },
  { slug: 'blocked', label: 'blocked', dot: 'var(--status-blocked)' },
  { slug: 'done', label: 'done', dot: 'var(--status-done)' },
];

// Derive column slug from task fields
function getTaskSlug(task: TaskWithDetailsResponse): string {
  if (task.is_blocked) return 'blocked';
  if (task.completed_at) return 'done';
  if (task.started_at) return 'in_progress';
  return 'todo';
}

export default function FeatureDetailDrawer({ feature, onClose, onTaskClick, projectId }: FeatureDetailDrawerProps) {
  const [tasks, setTasks] = useState<TaskWithDetailsResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const statusColor = STATUS_BADGE_COLORS[feature.status] ?? 'var(--text-muted)';

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    listTasks(projectId, { feature_id: feature.id })
      .then((data) => {
        if (!cancelled) {
          setTasks(data);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load tasks');
          setLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, [feature.id, projectId]);

  // Group tasks by status slug
  const grouped: Record<string, TaskWithDetailsResponse[]> = {};
  for (const task of tasks) {
    const slug = getTaskSlug(task);
    if (!grouped[slug]) grouped[slug] = [];
    grouped[slug].push(task);
  }

  const summary = feature.task_summary;

  return (
    <div
      className="fixed top-0 right-0 h-full w-[500px] flex flex-col z-40 shadow-xl border-l"
      style={{
        backgroundColor: 'var(--bg-primary)',
        borderColor: 'var(--border-subtle)',
      }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 py-3 border-b flex-shrink-0"
        style={{ borderColor: 'var(--border-subtle)' }}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0 mr-2">
          <h2
            className="font-['Newsreader'] text-base font-semibold text-[var(--text-primary)] truncate"
          >
            {feature.name}
          </h2>
          <span
            className="text-[9px] px-1.5 py-0.5 rounded-full font-['JetBrains_Mono'] font-bold uppercase tracking-wider flex-shrink-0"
            style={{
              color: statusColor,
              backgroundColor: `color-mix(in srgb, ${statusColor} 15%, transparent)`,
            }}
            data-qa="feature-drawer-status-badge"
          >
            {feature.status.replace('_', ' ')}
          </span>
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          <Link
            to={`/projects/${projectId}/features/${feature.id}`}
            data-qa="feature-open-full-page-link"
            className="p-1 rounded hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
            title="Open full page"
          >
            <ExternalLink size={14} />
          </Link>
          <button
            data-qa="feature-chat-btn"
            className="p-1 rounded hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
            title="Chat"
          >
            <MessageCircle size={14} />
          </button>
          <button
            onClick={onClose}
            className="flex-shrink-0 p-1 rounded hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">

        {/* Description */}
        {feature.description && (
          <p className="text-sm text-[var(--text-secondary)] leading-relaxed">
            {feature.description}
          </p>
        )}

        {/* Status summary bar */}
        {summary && (
          <div className="flex items-center gap-1 flex-wrap">
            {segments.map((seg) => {
              const count = summary[countKeys[seg.key]] ?? 0;
              return (
                <div
                  key={seg.key}
                  className="flex items-center gap-1 px-1.5 py-[2px] rounded"
                  style={{ backgroundColor: seg.bg }}
                >
                  <div
                    className="w-1.5 h-1.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: seg.dot }}
                  />
                  <span className="font-['JetBrains_Mono'] text-[9px] text-[var(--text-secondary)]">
                    {count} {seg.label}
                  </span>
                </div>
              );
            })}
          </div>
        )}

        {/* Task list */}
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 size={20} className="animate-spin text-[var(--text-muted)]" />
          </div>
        ) : error ? (
          <p className="text-sm text-[var(--status-blocked)] py-4 text-center">{error}</p>
        ) : tasks.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-4 text-center">No tasks</p>
        ) : (
          <div className="space-y-4">
            {TASK_GROUPS.map((group) => {
              const groupTasks = grouped[group.slug];
              if (!groupTasks || groupTasks.length === 0) return null;
              return (
                <div key={group.slug}>
                  {/* Group header */}
                  <div className="flex items-center gap-1.5 mb-1.5">
                    <div
                      className="w-1.5 h-1.5 rounded-full flex-shrink-0"
                      style={{ backgroundColor: group.dot }}
                    />
                    <span className="text-[10px] font-['JetBrains_Mono'] uppercase tracking-wider text-[var(--text-secondary)]">
                      {group.label}
                    </span>
                  </div>
                  {/* Task rows */}
                  <div className="space-y-0.5">
                    {groupTasks.map((task) => {
                      const prio = priorityPillVars[task.priority] || priorityPillVars.medium;
                      return (
                        <div
                          key={task.id}
                          onClick={() => onTaskClick(task.id, projectId)}
                          className="flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer transition-colors"
                          style={{ backgroundColor: 'transparent' }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--bg-tertiary)';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLElement).style.backgroundColor = 'transparent';
                          }}
                        >
                          {/* Title */}
                          <p className="font-['Newsreader'] text-sm text-[var(--text-primary)] truncate flex-1 min-w-0">
                            {task.title}
                          </p>
                          {/* Priority pill */}
                          <span
                            className="flex-shrink-0 px-1.5 py-[1px] rounded text-[9px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider"
                            style={{ color: prio.text, backgroundColor: prio.bg }}
                          >
                            {task.priority}
                          </span>
                          {/* Assigned role badge */}
                          {task.assigned_role && (
                            <span className="flex-shrink-0 text-[10px] font-['JetBrains_Mono'] px-1.5 py-0.5 rounded font-medium text-[var(--text-secondary)] bg-[var(--bg-tertiary)] truncate max-w-[80px]">
                              @{task.assigned_role}
                            </span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
