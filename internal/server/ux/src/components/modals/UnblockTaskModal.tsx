import { useState, useEffect, useCallback } from 'react';
import { X, TriangleAlert } from 'lucide-react';
import { unblockTask } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface UnblockTaskModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function UnblockTaskModal({ task, projectId, onClose, onSuccess }: UnblockTaskModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);

    try {
      await unblockTask(projectId, task.id);
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unblock task');
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
            Unblock Task
          </h2>
          <button
            onClick={onClose}
            data-qa="unblock-task-close-btn"
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5">
              <TriangleAlert size={24} className="text-[var(--status-progress)]" />
            </div>
            <div>
              <h3 className="text-[var(--text-primary)] text-base font-['Inter'] font-semibold mb-1">
                Unblock this task?
              </h3>
              <p className="text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed">
                This will move the task back to the Todo column and make it available for agents to pick up again.
              </p>
            </div>
          </div>

          {task.blocked_reason && (
            <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3">
              <p className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-1">
                Blocked Reason
              </p>
              <p className="text-[var(--text-primary)] text-sm font-['Inter'] leading-relaxed">
                {task.blocked_reason}
              </p>
            </div>
          )}

          {error && (
            <p className="text-[var(--status-blocked)] text-sm font-['Inter']">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            data-qa="unblock-task-cancel-btn"
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            data-qa="unblock-task-submit-btn"
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--status-progress)] hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Unblocking...' : 'Unblock Task'}
          </button>
        </div>
      </div>
    </div>
  );
}
