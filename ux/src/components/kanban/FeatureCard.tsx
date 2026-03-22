import { useState } from 'react';
import { FolderGit2 } from 'lucide-react';
import type { ProjectWithSummary } from '../../lib/types';

interface FeatureCardProps {
  feature: ProjectWithSummary;
  onClick: () => void;
}

const defaultBorder = 'color-mix(in srgb, var(--primary) 30%, transparent)';
const hoverBorder = 'var(--primary)';

const segments = [
  {
    key: 'todo' as const,
    label: 'todo',
    dot: 'var(--status-todo)',
    bg: 'color-mix(in srgb, var(--status-todo) 12%, transparent)',
  },
  {
    key: 'in_progress' as const,
    label: 'in progress',
    dot: 'var(--status-progress)',
    bg: 'color-mix(in srgb, var(--status-progress) 12%, transparent)',
  },
  {
    key: 'done' as const,
    label: 'done',
    dot: 'var(--status-done)',
    bg: 'color-mix(in srgb, var(--status-done) 12%, transparent)',
  },
  {
    key: 'blocked' as const,
    label: 'blocked',
    dot: 'var(--status-blocked)',
    bg: 'color-mix(in srgb, var(--status-blocked) 12%, transparent)',
  },
];

function getCount(
  feature: ProjectWithSummary,
  key: 'todo_count' | 'in_progress_count' | 'done_count' | 'blocked_count',
): number {
  const summary = feature.summary ?? feature.task_summary;
  if (!summary) return 0;
  return summary[key] ?? 0;
}

const countKeys = {
  todo: 'todo_count',
  in_progress: 'in_progress_count',
  done: 'done_count',
  blocked: 'blocked_count',
} as const;

export default function FeatureCard({ feature, onClick }: FeatureCardProps) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <div
      data-qa="feature-card"
      onClick={onClick}
      className="rounded-md p-[10px_12px] cursor-pointer transition-all duration-150 border bg-[var(--bg-elevated)]"
      style={{
        borderColor: isHovered ? hoverBorder : defaultBorder,
      }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* Header row */}
      <div className="flex items-center gap-1.5 mb-2">
        <FolderGit2 size={12} className="text-[var(--primary)] flex-shrink-0" />
        <p className="text-[var(--text-primary)] text-[13px] font-['Newsreader'] font-medium leading-snug truncate">
          {feature.name}
        </p>
      </div>

      {/* Status bar */}
      <div className="flex items-center gap-1 flex-wrap">
        {segments.map((seg) => {
          const count = getCount(feature, countKeys[seg.key]);
          return (
            <div
              key={seg.key}
              className="flex items-center gap-1 px-1.5 py-[2px] rounded"
              style={{ backgroundColor: seg.bg }}
            >
              <div
                className="w-1.5 h-1.5 rounded-full flex-shrink-0"
                style={{ backgroundColor: seg.dot }}
              />
              <span
                className="font-['JetBrains_Mono'] text-[9px] text-[var(--text-secondary)]"
              >
                {count} {seg.label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
