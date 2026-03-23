import type { TaskWithDetailsResponse } from '../../lib/types';
import { MessageSquare, GitBranch, GripVertical, Clock } from 'lucide-react';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { formatDuration } from '../../lib/utils';

interface TaskCardProps {
  task: TaskWithDetailsResponse;
  columnSlug: string;
  isNew?: boolean;
  isHighlighted?: boolean;
  selected?: boolean;
  roleColor?: string;
  onClick: () => void;
  onContextMenu: (e: React.MouseEvent) => void;
  onSelect?: (taskId: string, ctrlKey: boolean) => void;
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
      return { bg: 'var(--bg-tertiary)', border: 'var(--border-primary)', hoverBorder: 'var(--border-secondary)' };
  }
}

function formatShortDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export default function TaskCard({ task, columnSlug, isNew, isHighlighted, selected, roleColor, onClick, onContextMenu, onSelect }: TaskCardProps) {
  const style = getCardStyleVars(columnSlug);
  const prio = priorityPillVars[task.priority] || priorityPillVars.medium;

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: task.id });

  const dndStyle = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : undefined,
  };

  const handleClick = (e: React.MouseEvent) => {
    if (onSelect) {
      onSelect(task.id, e.ctrlKey || e.metaKey);
      if (!e.ctrlKey && !e.metaKey) {
        onClick();
      }
    } else {
      onClick();
    }
  };

  // Get initials for avatar
  const avatarLabel = task.assigned_role ? task.assigned_role[0].toUpperCase() : '?';

  return (
    <div
      ref={setNodeRef}
      style={{ ...dndStyle }}
    >
      <div
        data-qa="task-card"
        onClick={handleClick}
        onContextMenu={(e) => {
          e.preventDefault();
          onContextMenu(e);
        }}
        className={`group rounded-xl p-4 cursor-pointer transition-all duration-150 border${isHighlighted ? ' animate-task-highlight' : ''}`}
        style={{
          backgroundColor: style.bg,
          borderColor: selected ? 'var(--primary)' : style.border,
          opacity: style.opacity || '1',
          boxShadow: selected ? '0 0 0 2px var(--primary)' : undefined,
        }}
        onMouseEnter={(e) => {
          if (!selected) {
            (e.currentTarget as HTMLElement).style.borderColor = style.hoverBorder;
          }
        }}
        onMouseLeave={(e) => {
          if (!selected) {
            (e.currentTarget as HTMLElement).style.borderColor = style.border;
          }
        }}
      >
        {/* Project name (for sub-project tasks) */}
        {task.project_name && (
          <div className="mb-2">
            <span className="px-2 py-0.5 rounded-md bg-[var(--nav-bg-active)]/10 text-[var(--nav-text-active)] text-[9px] font-medium uppercase tracking-wider truncate inline-block max-w-full" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
              {task.project_name}
            </span>
          </div>
        )}

        {/* Top row: priority/tag pills */}
        <div className="flex items-center gap-1.5 mb-2.5 flex-wrap">
          <span
            className="px-2 py-0.5 rounded-md text-[10px] font-bold uppercase tracking-wider"
            style={{ color: prio.text, backgroundColor: prio.bg, fontFamily: 'JetBrains Mono, monospace' }}
          >
            {task.priority}
          </span>

          {/* Tags (show max 2) */}
          {task.tags?.slice(0, 2).map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 rounded-md bg-[var(--bg-tertiary)] text-[var(--text-secondary)] text-[10px] truncate max-w-[90px]"
              style={{ fontFamily: 'JetBrains Mono, monospace' }}
            >
              {tag}
            </span>
          ))}
          {task.tags && task.tags.length > 2 && (
            <span className="text-[var(--text-dim)] text-[10px]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
              +{task.tags.length - 2}
            </span>
          )}

          {/* New badge */}
          {isNew && columnSlug === 'done' && (
            <span className="px-2 py-0.5 rounded-md bg-[var(--status-done)]/20 text-[var(--status-done)] text-[9px] font-bold uppercase tracking-wider" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
              New
            </span>
          )}

          {/* Drag handle - far right */}
          <div className="flex-1" />
          <div
            {...attributes}
            {...listeners}
            data-qa="task-card-drag-handle"
            className="flex-shrink-0 text-[var(--text-dim)] opacity-0 group-hover:opacity-100 cursor-grab active:cursor-grabbing transition-opacity"
            onClick={(e) => e.stopPropagation()}
          >
            <GripVertical size={14} />
          </div>
        </div>

        {/* Title */}
        <p className="text-[var(--text-primary)] text-[14px] font-semibold leading-snug break-words mb-1.5" style={{ fontFamily: 'Inter, sans-serif' }}>
          {task.title}
        </p>

        {/* Summary */}
        {task.summary && (
          <p className="text-[var(--text-secondary)] text-[12px] leading-relaxed line-clamp-2 mb-3" style={{ fontFamily: 'Inter, sans-serif' }}>
            {task.summary}
          </p>
        )}

        {/* Blocked reason */}
        {task.is_blocked && task.blocked_reason && (
          <div className="mb-3 px-2.5 py-1.5 rounded-lg bg-[var(--status-blocked)]/10 text-[var(--status-blocked)] text-[11px] flex items-start gap-1.5" style={{ fontFamily: 'Inter, sans-serif' }}>
            <span className="shrink-0 mt-0.5">&#9888;</span>
            <span className="line-clamp-2">{task.blocked_reason}</span>
          </div>
        )}

        {/* Won't-do requested badge */}
        {task.wont_do_requested && (
          <div className="mb-3 px-2.5 py-1.5 rounded-lg bg-[var(--status-progress)]/10 text-[var(--status-progress)] text-[10px] font-bold uppercase tracking-wider inline-block" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
            Won't Do Requested
          </div>
        )}

        {/* Bottom row: avatar + metadata + date */}
        <div className="flex items-center gap-2">
          {/* Avatar */}
          {task.assigned_role && (
            <div
              className="w-7 h-7 rounded-full flex items-center justify-center text-[11px] font-bold flex-shrink-0"
              style={{
                backgroundColor: roleColor ?? '#6B7280',
                color: '#fff',
                fontFamily: 'Inter, sans-serif',
              }}
              title={task.assigned_role}
            >
              {avatarLabel}
            </div>
          )}

          {/* Meta indicators */}
          <div className="flex items-center gap-1.5 flex-1 min-w-0">
            {/* Feature dot */}
            {task.feature_id && (
              <div
                className="w-1.5 h-1.5 rounded-full bg-[var(--primary)]/60 shrink-0"
                title={`Feature: ${task.feature_id}`}
              />
            )}

            {/* Unresolved deps indicator */}
            {task.has_unresolved_deps && (
              <GitBranch size={12} className="text-[var(--status-progress)] shrink-0" />
            )}

            {/* Comment count */}
            {task.comment_count > 0 && (
              <div className="flex items-center gap-0.5 text-[var(--text-dim)]">
                <MessageSquare size={11} />
                <span className="text-[10px]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>{task.comment_count}</span>
              </div>
            )}
          </div>

          {/* Duration badge for completed tasks */}
          {columnSlug === 'done' && task.duration_seconds > 0 && (
            <div className="flex items-center gap-1 text-[var(--status-done)]">
              <Clock size={11} />
              <span className="text-[11px]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                {formatDuration(task.duration_seconds)}
              </span>
            </div>
          )}

          {/* Date */}
          <span className="text-[11px] text-[var(--text-dim)] shrink-0" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
            {formatShortDate(task.updated_at)}
          </span>
        </div>
      </div>
    </div>
  );
}
