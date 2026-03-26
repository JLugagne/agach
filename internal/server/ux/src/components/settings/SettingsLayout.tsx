import { Link, useParams, useLocation } from 'react-router-dom';
import { FileText, Users, ChevronLeft, UserPlus } from 'lucide-react';
import type { ReactNode } from 'react';

interface SettingsLayoutProps {
  projectName: string;
  children: ReactNode;
  rightDrawer?: ReactNode;
}

const tabs = [
  { label: 'Project Settings', path: '', icon: FileText },
  { label: 'Agents', path: '/agents', icon: Users },
  { label: 'Members', path: '/members', icon: UserPlus },
];

export default function SettingsLayout({ projectName, children, rightDrawer }: SettingsLayoutProps) {
  const { projectId } = useParams<{ projectId: string }>();
  const location = useLocation();

  return (
    <div className="h-full bg-[var(--bg-secondary)] flex flex-col md:flex-row overflow-hidden">
      {/* Left sidebar — horizontal tabs on mobile, vertical sidebar on desktop */}
      <aside className="md:w-56 bg-[#0D0D0D] border-b md:border-b-0 md:border-r border-[#2A2A2A] flex flex-col shrink-0">
        <div className="p-4 border-b border-[var(--border-primary)]">
          <Link
            to={`/`}
            className="flex items-center gap-2 text-[var(--text-muted)] hover:text-[#E0E0E0] text-sm transition-colors"
          >
            <ChevronLeft size={14} />
            <span>Back to Projects</span>
          </Link>
        </div>

        <div className="hidden md:block p-4">
          <p className="font-heading text-sm text-[var(--text-primary)] truncate mb-1">{projectName}</p>
          <p className="text-xs text-[var(--text-dim)]">Settings</p>
        </div>

        <nav className="flex md:flex-col md:flex-1 px-2 py-1 md:py-0 overflow-x-auto">
          {tabs.map((tab) => {
            const fullPath = `/projects/${projectId}/settings${tab.path}`;
            const isActive = location.pathname === fullPath;
            const Icon = tab.icon;
            return (
              <Link
                key={tab.path}
                to={fullPath}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-md text-sm mb-0.5 transition-colors whitespace-nowrap ${
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
      <div className="flex-1 flex min-w-0 overflow-hidden">
        <main className="flex-1 p-4 sm:p-8 md:p-12 max-w-2xl overflow-y-auto">{children}</main>

        {/* Optional right drawer — hidden on mobile */}
        {rightDrawer && (
          <aside className="hidden lg:block w-[680px] bg-[var(--bg-primary)] border-l border-[var(--border-primary)] shrink-0 overflow-y-auto">
            {rightDrawer}
          </aside>
        )}
      </div>
    </div>
  );
}
