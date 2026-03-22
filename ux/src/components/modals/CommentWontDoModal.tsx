import { useState, useEffect, useCallback } from 'react';
import { X } from 'lucide-react';
import { createComment } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface CommentWontDoModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

const MAX_CHARS = 256;

export default function CommentWontDoModal({ task, projectId, onClose, onSuccess }: CommentWontDoModalProps) {
  const [content, setContent] = useState('');
  const [markAsWontDo, setMarkAsWontDo] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (content.trim().length === 0) {
      setError('Comment cannot be empty.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await createComment(projectId, task.id, {
        content: content.trim(),
        mark_as_wont_do: markAsWontDo,
        author_role: 'human',
        author_name: '',
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add comment');
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
      <div className="relative w-[420px] rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-primary)] shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            Add Comment
          </h2>
          <div className="flex items-center gap-3">
            <span className="text-[var(--text-dim)] text-xs font-['JetBrains_Mono']">
              {content.length} / {MAX_CHARS}
            </span>
            <button
              onClick={onClose}
              data-qa="comment-wont-do-close-btn"
              className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
            >
              <X size={20} />
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <div>
            <textarea
              value={content}
              onChange={(e) => {
                if (e.target.value.length <= MAX_CHARS) {
                  setContent(e.target.value);
                }
              }}
              placeholder="Write your comment..."
              rows={4}
              data-qa="comment-content-input"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
          </div>

          <label className="flex items-center gap-3 cursor-pointer select-none">
            <div className="relative">
              <input
                type="checkbox"
                checked={markAsWontDo}
                onChange={(e) => setMarkAsWontDo(e.target.checked)}
                data-qa="comment-mark-wont-do-checkbox"
                className="sr-only peer"
              />
              <div className="w-4 h-4 border border-[var(--border-primary)] rounded bg-[var(--bg-secondary)] peer-checked:bg-[var(--status-progress)] peer-checked:border-[var(--status-progress)] transition-colors flex items-center justify-center">
                {markAsWontDo && (
                  <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                    <path d="M1 4L3.5 6.5L9 1" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </div>
            </div>
            <span className="text-[var(--text-primary)] text-sm font-['Inter']">
              Mark as Won't Do
            </span>
          </label>

          {error && (
            <p className="text-[var(--status-blocked)] text-sm font-['Inter']">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            data-qa="comment-wont-do-cancel-btn"
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || content.trim().length === 0}
            data-qa="comment-wont-do-submit-btn"
            className={`px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
              markAsWontDo
                ? 'bg-[var(--status-progress)] hover:opacity-90'
                : 'bg-[var(--primary)] hover:bg-[var(--primary-hover)]'
            }`}
          >
            {loading
              ? 'Submitting...'
              : markAsWontDo
                ? "Comment & Mark Won't Do"
                : 'Comment'}
          </button>
        </div>
      </div>
    </div>
  );
}
