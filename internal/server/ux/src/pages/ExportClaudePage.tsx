import { Link, useParams } from 'react-router-dom';
import { ChevronLeft, Bot } from 'lucide-react';

export default function ExportClaudePage() {
  const { projectId } = useParams<{ projectId: string }>();

  return (
    <div className="min-h-screen bg-[var(--bg-secondary)] flex flex-col items-center justify-center px-8">
      <div className="text-center max-w-md">
        <div className="mx-auto w-16 h-16 rounded-full bg-[var(--bg-secondary)] flex items-center justify-center mb-6">
          <Bot size={28} className="text-[#F09060]" />
        </div>
        <h1 className="font-heading text-2xl text-[var(--text-primary)] mb-3">Export to Claude Code</h1>
        <p className="text-sm text-[var(--text-muted)] mb-8">
          This feature is coming soon. It will allow you to export your project context and tasks
          for use with Anthropic's Claude Code.
        </p>
        <Link
          to={projectId ? `/projects/${projectId}` : '/'}
          data-qa="back-to-project-link"
          className="inline-flex items-center gap-1.5 text-sm text-[var(--primary)] hover:text-[var(--primary)]/80 transition-colors"
        >
          <ChevronLeft size={14} />
          Back to Project
        </Link>
      </div>
    </div>
  );
}
