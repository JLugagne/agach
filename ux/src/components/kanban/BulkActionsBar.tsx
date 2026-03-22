import { X, ArrowRight, Trash2, Ban, CheckCircle2, Unlock } from 'lucide-react';
import type { TaskWithDetailsResponse } from '../../lib/types';
import { moveTask, deleteTask, blockTask, unblockTask, completeTask } from '../../lib/api';

// Determine the single column slug if all tasks are in the same column, or 'mixed'
function getColumnContext(tasks: TaskWithDetailsResponse[], columnMap: Map<string, string>): string {
  const slugs = new Set<string>();
  for (const t of tasks) {
    const slug = columnMap.get(t.id);
    if (slug) slugs.add(slug);
  }
  if (slugs.size === 1) return [...slugs][0];
  return 'mixed';
}

interface BulkActionsBarInternalProps {
  selectedTasks: TaskWithDetailsResponse[];
  columnMap: Map<string, string>; // taskId -> columnSlug
  projectId: string;
  onClear: () => void;
  onRefresh: () => void;
}

export default function BulkActionsBar({
  selectedTasks,
  columnMap,
  projectId,
  onClear,
  onRefresh,
}: BulkActionsBarInternalProps) {
  const count = selectedTasks.length;
  if (count === 0) return null;

  const context = getColumnContext(selectedTasks, columnMap);

  const runAll = async (fn: (task: TaskWithDetailsResponse) => Promise<unknown>) => {
    await Promise.all(selectedTasks.map((t) => fn(t).catch(() => { /* ignore individual failures */ })));
    onClear();
    onRefresh();
  };

  const handleMoveInProgress = () => runAll((t) => {
    const pid = t.project_id || projectId;
    return moveTask(pid, t.id, { target_column: 'in_progress' });
  });

  const handleMoveTodo = () => runAll((t) => {
    const pid = t.project_id || projectId;
    return moveTask(pid, t.id, { target_column: 'todo' });
  });

  const handleDelete = () => runAll((t) => {
    const pid = t.project_id || projectId;
    return deleteTask(pid, t.id);
  });

  const handleBlock = () => runAll((t) => {
    const pid = t.project_id || projectId;
    return blockTask(pid, t.id, { blocked_reason: 'Blocked via bulk action', blocked_by_agent: 'human' });
  });

  const handleUnblock = () => runAll((t) => {
    const pid = t.project_id || projectId;
    return unblockTask(pid, t.id);
  });

  const handleComplete = () => runAll((t) => {
    const pid = t.project_id || projectId;
    return completeTask(pid, t.id, { completion_summary: 'Completed via bulk action', completed_by_agent: 'human' });
  });

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-2 px-4 py-2.5 rounded-xl bg-[#1A1A1A] border border-[#2A2A2A] shadow-2xl">
      {/* Count badge */}
      <span className="px-2 py-0.5 rounded bg-[var(--primary)]/20 text-[var(--primary)] text-[11px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider">
        {count} selected
      </span>

      <div className="w-px h-5 bg-[#2A2A2A]" />

      {/* Context-aware action buttons */}
      {context === 'todo' && (
        <>
          <button
            data-qa="bulk-move-in-progress-btn"
            onClick={handleMoveInProgress}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-progress-bg)] text-[var(--status-progress)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
          >
            <ArrowRight size={13} />
            Move to In Progress
          </button>
          <button
            data-qa="bulk-block-btn"
            onClick={handleBlock}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-blocked-bg)] text-[var(--status-blocked)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
          >
            <Ban size={13} />
            Block
          </button>
        </>
      )}

      {context === 'in_progress' && (
        <>
          <button
            data-qa="bulk-move-todo-btn"
            onClick={handleMoveTodo}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-todo-bg)] text-[var(--status-todo)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
          >
            <ArrowRight size={13} />
            Move to Todo
          </button>
          <button
            data-qa="bulk-complete-btn"
            onClick={handleComplete}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-done-bg)] text-[var(--status-done)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
          >
            <CheckCircle2 size={13} />
            Complete
          </button>
          <button
            data-qa="bulk-block-btn"
            onClick={handleBlock}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-blocked-bg)] text-[var(--status-blocked)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
          >
            <Ban size={13} />
            Block
          </button>
        </>
      )}

      {context === 'blocked' && (
        <button
          data-qa="bulk-unblock-btn"
          onClick={handleUnblock}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-todo-bg)] text-[var(--status-todo)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
        >
          <Unlock size={13} />
          Unblock
        </button>
      )}

      {context === 'done' && (
        <button
          data-qa="bulk-move-todo-btn"
          onClick={handleMoveTodo}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[var(--status-todo-bg)] text-[var(--status-todo)] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
        >
          <ArrowRight size={13} />
          Move to Todo
        </button>
      )}

      {/* Delete — always shown */}
      <button
        data-qa="bulk-delete-btn"
        onClick={handleDelete}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#F0606015] text-[#F06060] text-xs font-['Inter'] font-medium hover:opacity-80 transition-opacity"
      >
        <Trash2 size={13} />
        Delete
      </button>

      <div className="w-px h-5 bg-[#2A2A2A]" />

      {/* Cancel */}
      <button
        data-qa="bulk-cancel-btn"
        onClick={onClear}
        className="flex items-center gap-1 text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors text-xs font-['Inter']"
      >
        <X size={14} />
        Cancel
      </button>
    </div>
  );
}

