import { useState, useEffect, useCallback } from 'react';
import { X } from 'lucide-react';
import { markWontDo } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface MarkWontDoModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function MarkWontDoModal({ task, projectId, onClose, onSuccess }: MarkWontDoModalProps) {
  const [reason, setReason] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (reason.trim().length < 50) {
      setError('Reason must be at least 50 characters.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await markWontDo(projectId, task.id, {
        wont_do_reason: reason.trim(),
        wont_do_requested_by: 'human',
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to mark as won\'t do');
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
            Mark as Won't Do
          </h2>
          <button
            onClick={onClose}
            data-qa="mark-wont-do-close-btn"
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
              Reason Required
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Briefly explain why this won't be done..."
              rows={4}
              data-qa="wont-do-reason-input"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
            <p className="text-[var(--text-dim)] text-xs font-['Inter'] mt-1">
              {reason.length} / 50 minimum characters
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
            data-qa="mark-wont-do-cancel-btn"
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || reason.trim().length < 50}
            data-qa="mark-wont-do-submit-btn"
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--status-progress)] hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Processing...' : "Mark as Won't Do"}
          </button>
        </div>
      </div>
    </div>
  );
}
