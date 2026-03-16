import { AlertCircle } from 'lucide-react';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface BlockedBannerProps {
  task: TaskWithDetailsResponse;
  onUnblock?: () => void;
}

export default function BlockedBanner({ task, onUnblock }: BlockedBannerProps) {
  if (!task.is_blocked) return null;

  return (
    <div className="rounded-lg border border-[var(--status-blocked)] bg-[var(--status-blocked-bg)] p-4">
      <div className="flex items-start gap-3">
        <AlertCircle size={20} className="text-[var(--status-blocked)] mt-0.5 flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[var(--status-blocked)] text-xs font-['JetBrains_Mono'] font-bold uppercase tracking-wider">
              BLOCKED
            </span>
            {task.wont_do_requested && (
              <span className="text-[var(--status-progress)] text-xs font-['JetBrains_Mono'] font-bold uppercase tracking-wider px-1.5 py-0.5 bg-[var(--status-progress-bg)] rounded">
                Won't Do Requested
              </span>
            )}
          </div>

          {task.blocked_reason && (
            <p className="text-[var(--text-primary)] text-sm font-['Inter'] leading-relaxed mb-2">
              {task.blocked_reason}
            </p>
          )}

          {task.blocked_by_agent && (
            <p className="text-[var(--text-muted)] text-xs font-['Inter'] mb-3">
              Blocked by <span className="text-[var(--text-secondary)] font-['JetBrains_Mono']">{task.blocked_by_agent}</span>
              {task.blocked_at && (
                <> on {new Date(task.blocked_at).toLocaleDateString()}</>
              )}
            </p>
          )}

          {task.wont_do_requested && task.wont_do_reason && (
            <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3 mb-3">
              <p className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-1">
                Won't Do Reason
              </p>
              <p className="text-[var(--text-primary)] text-sm font-['Inter'] leading-relaxed">
                {task.wont_do_reason}
              </p>
              {task.wont_do_requested_by && (
                <p className="text-[var(--text-dim)] text-xs font-['Inter'] mt-1">
                  Requested by {task.wont_do_requested_by}
                </p>
              )}
            </div>
          )}

          {onUnblock && (
            <button
              onClick={onUnblock}
              className="px-3 py-1.5 text-xs font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] rounded-md transition-colors"
            >
              Unblock Task
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
