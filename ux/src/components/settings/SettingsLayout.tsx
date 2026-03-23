import { Link, useParams, useLocation } from 'react-router-dom';
import { FileText, Users, ChevronLeft } from 'lucide-react';
import type { ReactNode } from 'react';

interface SettingsLayoutProps {
  projectName: string;
  children: ReactNode;
  rightDrawer?: ReactNode;
}

const tabs = [
  { label: 'Project Settings', path: '', icon: FileText },
  { label: 'Agents', path: '/agents', icon: Users },
];

export default function SettingsLayout({ projectName, children, rightDrawer }: SettingsLayoutProps) {
  const { projectId } = useParams<{ projectId: string }>();
  const location = useLocation();

  return (
    <div className="min-h-screen bg-[var(--bg-secondary)] flex">
      {/* Left sidebar */}
      <aside className="w-56 bg-[#0D0D0D] border-r border-[#2A2A2A] flex flex-col shrink-0">
        <div className="p-4 border-b border-[var(--border-primary)]">
          <Link
            to={`/`}
            className="flex items-center gap-2 text-[var(--text-muted)] hover:text-[#E0E0E0] text-sm transition-colors"
          >
            <ChevronLeft size={14} />
            <span>Back to Projects</span>
          </Link>
        </div>

        <div className="p-4">
          <p className="font-heading text-sm text-[var(--text-primary)] truncate mb-1">{projectName}</p>
          <p className="text-xs text-[var(--text-dim)]">Settings</p>
        </div>

        <nav className="flex-1 px-2">
          {tabs.map((tab) => {
            const fullPath = `/projects/${projectId}/settings${tab.path}`;
            const isActive = location.pathname === fullPath;
            const Icon = tab.icon;
            return (
              <Link
                key={tab.path}
                to={fullPath}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-md text-sm mb-0.5 transition-colors ${
                  isActive
                    ? 'bg-[var(--bg-secondary)] text-[var(--text-primary)]'
                    : 'text-[var(--text-muted)] hover:text-[#E0E0E0] hover:bg-[var(--bg-secondary)]/50'
                }`}
              >
                <Icon size={15} />
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex min-w-0">
        <main className="flex-1 p-12 max-w-2xl">{children}</main>

        {/* Optional right drawer */}
        {rightDrawer && (
          <aside className="w-[680px] bg-[var(--bg-primary)] border-l border-[var(--border-primary)] shrink-0 overflow-y-auto">
            {rightDrawer}
          </aside>
        )}
      </div>
    </div>
  );
}
