import { X } from 'lucide-react';
import MarkdownContent from '../ui/MarkdownContent';

interface FeatureChangelogDrawerProps {
  open: boolean;
  onClose: () => void;
  type: 'user' | 'tech';
  content: string;
  featureName: string;
}

export default function FeatureChangelogDrawer({ open, onClose, type, content, featureName }: FeatureChangelogDrawerProps) {
  if (!open) return null;

  const title = type === 'user' ? 'User Changelog' : 'Technical Changelog';

  return (
    <>
      {/* Overlay */}
      <div className="fixed inset-0 z-40 bg-[rgba(0,0,0,0.5)]" onClick={onClose} />

      {/* Drawer */}
      <div
        className="fixed top-0 right-0 z-50 h-full w-[720px] max-w-full bg-[var(--card-bg)] border-l border-[var(--border-primary)] shadow-2xl flex flex-col overflow-hidden animate-slide-in"
        data-qa="feature-changelog-drawer"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <div className="flex flex-col gap-0.5">
            <h2 className="text-[var(--text-primary)] text-lg font-['Newsreader'] font-medium">{title}</h2>
            <span className="text-[var(--text-muted)] text-xs font-['Inter']">{featureName}</span>
          </div>
          <button
            onClick={onClose}
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
            aria-label="Close"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {content.trim() ? (
            <MarkdownContent content={content} />
          ) : (
            <p className="text-[var(--text-dim)] text-sm font-['Inter'] italic">No changelog available</p>
          )}
        </div>
      </div>
    </>
  );
}
