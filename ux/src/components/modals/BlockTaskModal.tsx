import { useState, useEffect, useCallback } from 'react';
import { X } from 'lucide-react';
import { blockTask } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface BlockTaskModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function BlockTaskModal({ task, projectId, onClose, onSuccess }: BlockTaskModalProps) {
  const [reason, setReason] = useState('');
  const [agentName, setAgentName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (reason.trim().length < 50) {
      setError('Blocked reason must be at least 50 characters.');
      return;
    }
    if (agentName.trim().length === 0) {
      setError('Agent name is required.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await blockTask(projectId, task.id, {
        blocked_reason: reason.trim(),
        blocked_by_agent: agentName.trim(),
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to block task');
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
      <div className="relative w-[440px] rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-primary)] shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            Block Task
          </h2>
          <button
            onClick={onClose}
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <div>
            <p className="text-[var(--text-secondary)] text-sm font-['Inter'] mb-1">Task</p>
            <p className="text-[var(--text-primary)] text-sm font-['Inter'] font-medium">{task.title}</p>
          </div>

          <div>
            <label className="block text-[var(--text-primary)] text-sm font-['Inter'] font-medium mb-2">
              Blocked Reason
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Describe why this task is blocked (min 50 chars)..."
              rows={4}
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
            <p className="text-[var(--text-dim)] text-xs font-['Inter'] mt-1">
              {reason.length} / 50 minimum characters
            </p>
          </div>

          <div>
            <label className="block text-[var(--text-primary)] text-sm font-['Inter'] font-medium mb-2">
              Agent Name
            </label>
            <input
              type="text"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              placeholder="e.g. human or agent identifier"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
          </div>

          {error && (
            <p className="text-[var(--status-blocked)] text-sm font-['Inter']">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || reason.trim().length < 50 || agentName.trim().length === 0}
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-white bg-[var(--status-blocked)] hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Blocking...' : 'Block Task'}
          </button>
        </div>
      </div>
    </div>
  );
}
