import type { ColumnWithTasksResponse, TaskWithDetailsResponse } from '../../lib/types';
import TaskCard from './TaskCard';

interface ColumnProps {
  column: ColumnWithTasksResponse;
  onTaskClick: (task: TaskWithDetailsResponse) => void;
  onTaskContextMenu: (task: TaskWithDetailsResponse, e: React.MouseEvent) => void;
  isTaskNew?: (taskId: string) => boolean;
  isTaskHighlighted?: (taskId: string) => boolean;
}

const statusColorVars: Record<string, { dot: string; label: string; countBg: string; countText: string }> = {
  todo: { dot: 'var(--status-todo)', label: 'var(--status-todo)', countBg: 'var(--status-todo-bg)', countText: 'var(--status-todo)' },
  in_progress: { dot: 'var(--status-progress)', label: 'var(--status-progress)', countBg: 'var(--status-progress-bg)', countText: 'var(--status-progress)' },
  done: { dot: 'var(--status-done)', label: 'var(--status-done)', countBg: 'var(--status-done-bg)', countText: 'var(--status-done)' },
  blocked: { dot: 'var(--status-blocked)', label: 'var(--status-blocked)', countBg: 'var(--status-blocked-bg)', countText: 'var(--status-blocked)' },
};

export default function Column({ column, onTaskClick, onTaskContextMenu, isTaskNew, isTaskHighlighted }: ColumnProps) {
  const colors = statusColorVars[column.slug] || statusColorVars.todo;
  const tasks = column.tasks || [];
  const wipLimit = column.wip_limit;
  const isOverWip = wipLimit > 0 && tasks.length > wipLimit;

  return (
    <div className="flex flex-col rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-primary)] min-w-0 flex-1 h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-4 h-[44px] bg-[var(--bg-secondary)] rounded-t-lg flex-shrink-0">
        <div className="flex items-center gap-2">
          <div
            className="w-2 h-2 rounded-full"
            style={{ backgroundColor: colors.dot }}
          />
          <span
            className="text-xs font-['JetBrains_Mono'] font-bold uppercase tracking-wider"
            style={{ color: colors.label }}
          >
            {column.name}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <span
            className="px-1.5 py-0.5 rounded text-[10px] font-['JetBrains_Mono'] font-bold min-w-[20px] text-center"
            style={{ backgroundColor: colors.countBg, color: colors.countText }}
          >
            {tasks.length}
          </span>
          {wipLimit > 0 && (
            <span
              className={`text-[10px] font-['JetBrains_Mono'] ${
                isOverWip ? 'text-[var(--status-blocked)]' : 'var(--text-dim)'
              }`}
              style={!isOverWip ? { color: 'var(--text-dim)' } : {}}
            >
              /{wipLimit}
            </span>
          )}
        </div>
      </div>

      {/* Cards */}
      <div className="flex flex-col gap-2 p-3 overflow-y-auto flex-1">
        {tasks.length === 0 ? (
          <p className="text-[var(--text-muted)] text-xs font-['Inter'] text-center py-6">
            No tasks
          </p>
        ) : (
          tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              columnSlug={column.slug}
              isNew={isTaskNew ? isTaskNew(task.id) : false}
              isHighlighted={isTaskHighlighted ? isTaskHighlighted(task.id) : false}
              onClick={() => onTaskClick(task)}
              onContextMenu={(e) => onTaskContextMenu(task, e)}
            />
          ))
        )}
      </div>
    </div>
  );
}
