import type { TaskWithDetailsResponse } from '../../lib/types';
import { MessageSquare, GitBranch } from 'lucide-react';

interface TaskCardProps {
  task: TaskWithDetailsResponse;
  columnSlug: string;
  isNew?: boolean;
  isHighlighted?: boolean;
  onClick: () => void;
  onContextMenu: (e: React.MouseEvent) => void;
}

const priorityPillVars: Record<string, { text: string; bg: string }> = {
  critical: { text: 'var(--priority-critical)', bg: 'var(--priority-critical-bg)' },
  high: { text: 'var(--priority-high)', bg: 'var(--priority-high-bg)' },
  medium: { text: 'var(--priority-medium)', bg: 'var(--priority-medium-bg)' },
  low: { text: 'var(--priority-low)', bg: 'var(--priority-low-bg)' },
};

interface CardStyle {
  bg: string;
  border: string;
  hoverBorder: string;
  opacity?: string;
}

function getCardStyleVars(columnSlug: string): CardStyle {
  switch (columnSlug) {
    case 'in_progress':
      return { bg: 'var(--status-progress-bg)', border: 'var(--status-progress-bg)', hoverBorder: 'var(--status-progress)' };
    case 'done':
      return { bg: 'var(--status-done-bg)', border: 'var(--status-done-bg)', hoverBorder: 'var(--status-done)', opacity: '0.65' };
    case 'blocked':
      return { bg: 'var(--status-blocked-bg)', border: 'var(--status-blocked-bg)', hoverBorder: 'var(--status-blocked)' };
    default:
      return { bg: 'var(--bg-elevated)', border: 'var(--border-subtle)', hoverBorder: 'var(--text-muted)' };
  }
}

export default function TaskCard({ task, columnSlug, isNew, isHighlighted, onClick, onContextMenu }: TaskCardProps) {
  const style = getCardStyleVars(columnSlug);
  const prio = priorityPillVars[task.priority] || priorityPillVars.medium;

  return (
    <div
      onClick={onClick}
      onContextMenu={(e) => {
        e.preventDefault();
        onContextMenu(e);
      }}
      className={`group rounded-md p-[10px_12px] cursor-pointer transition-all duration-150 bg-[var(--card-bg)] border border-[var(--card-border)]${isHighlighted ? ' animate-task-highlight' : ''}`}
      style={{
        backgroundColor: style.bg,
        borderColor: style.border,
        opacity: style.opacity || '1',
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = style.hoverBorder;
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = style.border;
      }}
    >
      {/* Project name (for sub-project tasks) */}
      {task.project_name && (
        <div className="mb-1.5">
          <span className="px-1.5 py-[1px] rounded bg-[var(--nav-bg-active)]/10 text-[var(--nav-text-active)] text-[9px] font-['JetBrains_Mono'] font-medium uppercase tracking-wider truncate inline-block max-w-full">
            {task.project_name}
          </span>
        </div>
      )}

      {/* Title row */}
      <div className="flex items-start justify-between gap-2 mb-2">
        <p className="text-[var(--text-primary)] text-[13px] font-['Newsreader'] font-medium leading-snug break-words flex-1">
          {task.title}
        </p>
        {isNew && columnSlug === 'done' && (
          <span className="flex-shrink-0 px-1.5 py-[1px] rounded bg-[var(--status-done)]/20 text-[var(--status-done)] text-[9px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider">
            New
          </span>
        )}
      </div>

      {/* Meta row */}
      <div className="flex items-center gap-1.5 flex-wrap">
        {/* Priority pill */}
        <span
          className="px-1.5 py-[1px] rounded text-[9px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider"
          style={{ color: prio.text, backgroundColor: prio.bg }}
        >
          {task.priority}
        </span>

        {/* Tags (show max 2) */}
        {task.tags?.slice(0, 2).map((tag) => (
          <span
            key={tag}
            className="px-1.5 py-[1px] rounded bg-[var(--bg-tertiary)] text-[var(--text-secondary)] text-[9px] font-['JetBrains_Mono'] truncate max-w-[80px]"
          >
            {tag}
          </span>
        ))}
        {task.tags && task.tags.length > 2 && (
          <span className="text-[var(--text-dim)] text-[9px] font-['JetBrains_Mono']">
            +{task.tags.length - 2}
          </span>
        )}

        {/* Spacer */}
        <div className="flex-1" />

        {/* Assigned role */}
        {task.assigned_role && (
          <span className="text-[var(--text-muted)] text-[9px] font-['JetBrains_Mono'] truncate max-w-[60px]">
            @{task.assigned_role}
          </span>
        )}

        {/* Unresolved deps indicator */}
        {task.has_unresolved_deps && (
          <GitBranch size={10} className="text-[var(--status-progress)]" />
        )}

        {/* Comment count */}
        {task.comment_count > 0 && (
          <div className="flex items-center gap-0.5 text-[var(--text-dim)]">
            <MessageSquare size={10} />
            <span className="text-[9px] font-['JetBrains_Mono']">{task.comment_count}</span>
          </div>
        )}
      </div>

      {/* Won't-do requested badge */}
      {task.wont_do_requested && (
        <div className="mt-2 px-1.5 py-0.5 rounded bg-[var(--status-progress)]/10 text-[var(--status-progress)] text-[9px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider inline-block">
          Won't Do Requested
        </div>
      )}
    </div>
  );
}
