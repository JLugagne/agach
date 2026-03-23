import { useState, useEffect, useCallback } from 'react';
import { X, Trash2 } from 'lucide-react';
import { deleteTask } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface DeleteTaskModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function DeleteTaskModal({ task, projectId, onClose, onSuccess }: DeleteTaskModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);

    try {
      await deleteTask(projectId, task.id);
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete task');
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
  }, [onClose]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="absolute inset-0 bg-black/60" />
      <div className="relative w-[400px] rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-primary)] shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            Delete Task
          </h2>
          <button
            onClick={onClose}
            data-qa="delete-task-close-btn"
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 p-2 bg-[var(--status-blocked-bg)] rounded-lg">
              <Trash2 size={20} className="text-[var(--status-blocked)]" />
            </div>
            <div>
              <h3 className="text-[var(--text-primary)] text-base font-['Inter'] font-semibold mb-1">
                Delete this task?
              </h3>
              <p className="text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed">
                This action cannot be undone. The task, its comments, and dependencies will be permanently removed.
              </p>
            </div>
          </div>

          <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3">
            <p className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-1">
              Task
            </p>
            <p className="text-[var(--text-primary)] text-sm font-['Inter'] font-medium">
              {task.title}
            </p>
          </div>

          {error && (
            <p className="text-[var(--status-blocked)] text-sm font-['Inter']">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            data-qa="delete-task-cancel-btn"
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            data-qa="delete-task-confirm-btn"
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-white bg-[var(--status-blocked)] hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Deleting...' : 'Delete Task'}
          </button>
        </div>
      </div>
    </div>
  );
}
