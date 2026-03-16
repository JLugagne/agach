import { useState, useEffect, useCallback } from 'react';
import { X } from 'lucide-react';
import { approveWontDo, rejectWontDo } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface ApproveWontDoModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function ApproveWontDoModal({ task, projectId, onClose, onSuccess }: ApproveWontDoModalProps) {
  const [rejecting, setRejecting] = useState(false);
  const [rejectionReason, setRejectionReason] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleApprove = async () => {
    setLoading(true);
    setError(null);

    try {
      await approveWontDo(projectId, task.id);
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve');
    } finally {
      setLoading(false);
    }
  };

  const handleReject = async () => {
    if (!rejecting) {
      setRejecting(true);
      return;
    }

    if (rejectionReason.trim().length < 10) {
      setError('Rejection reason must be at least 10 characters.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await rejectWontDo(projectId, task.id, { reason: rejectionReason.trim() });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject');
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
      <div className="relative w-[480px] rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-primary)] shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            Won't Do Requested
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

          {task.wont_do_requested_by && (
            <div>
              <p className="text-[var(--text-secondary)] text-sm font-['Inter'] mb-1">Agent Role</p>
              <p className="text-[var(--text-primary)] text-sm font-['JetBrains_Mono']">
                {task.wont_do_requested_by}
              </p>
            </div>
          )}

          {task.wont_do_reason && (
            <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3">
              <p className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-1">
                Agent's Reason
              </p>
              <p className="text-[var(--text-primary)] text-sm font-['Inter'] leading-relaxed">
                {task.wont_do_reason}
              </p>
            </div>
          )}

          {rejecting && (
            <div>
              <label className="block text-[var(--text-primary)] text-sm font-['Inter'] font-medium mb-2">
                Rejection reason required
              </label>
              <textarea
                value={rejectionReason}
                onChange={(e) => setRejectionReason(e.target.value)}
                placeholder="Explain why this request is being rejected..."
                rows={3}
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
              />
              <p className="text-[var(--text-dim)] text-xs font-['Inter'] mt-1">
                {rejectionReason.length} / 10 minimum characters
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
            onClick={handleReject}
            disabled={loading}
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--status-blocked)] border border-[var(--status-blocked)] hover:bg-[var(--status-blocked)] hover:text-white disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading && rejecting ? 'Rejecting...' : 'Reject'}
          </button>
          {!rejecting && (
            <button
              onClick={handleApprove}
              disabled={loading}
              className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
            >
              {loading ? 'Approving...' : "Approve Won't Do"}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
