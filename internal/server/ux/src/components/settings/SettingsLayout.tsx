import { Link, useParams, useLocation } from 'react-router-dom';
import { ChevronLeft, ChevronRight, Settings, Bot, Users } from 'lucide-react';
import type { ReactNode } from 'react';

interface SettingsLayoutProps {
  projectName: string;
  children: ReactNode;
  rightDrawer?: ReactNode;
  showSharing?: boolean;
}

const baseTabs = [
  { label: 'Project Settings', path: '', icon: Settings },
  { label: 'Agents', path: '/agents', icon: Bot },
];

const sharingTab = { label: 'Sharing', path: '/members', icon: Users };

function activeTabLabel(pathname: string): string {
  if (pathname.endsWith('/agents')) return 'Agents';
  if (pathname.endsWith('/members')) return 'Sharing';
  return 'Project Settings';
}

export default function SettingsLayout({ projectName, children, rightDrawer, showSharing = true }: SettingsLayoutProps) {
  const { projectId } = useParams<{ projectId: string }>();
  const location = useLocation();
  const tabs = showSharing ? [...baseTabs, sharingTab] : baseTabs;
  const currentLabel = activeTabLabel(location.pathname);

  return (
    <div className="h-full bg-[var(--bg-secondary)] flex overflow-hidden">
      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <main className="flex-1 overflow-y-auto px-6 sm:px-10 py-8">
          <div className="max-w-5xl mx-auto flex flex-col gap-7">
            {/* Breadcrumb */}
            <nav className="flex items-center gap-2 text-[13px]">
              <span className="text-[var(--text-dim)]">Settings</span>
              <ChevronRight size={14} className="text-[var(--text-dim)]" />
              <span className="text-[var(--primary)] font-medium">{currentLabel}</span>
            </nav>

            {/* Title */}
            <div className="flex flex-col gap-1">
              <h1 className="text-[26px] font-semibold text-[var(--text-primary)]">Project Settings</h1>
              <p className="text-sm text-[var(--text-dim)]">{projectName}</p>
            </div>

            {/* Divider */}
            <div className="h-px bg-[var(--border-primary)]" />

            {/* SubNav: back link | project path | tab pills */}
            <div className="flex items-center gap-4">
              <Link
                to="/"
                className="flex items-center gap-1.5 text-[var(--primary)] text-[13px] font-medium hover:opacity-80 transition-opacity shrink-0"
              >
                <ChevronLeft size={16} />
                Back to Projects
              </Link>

              <div className="flex-1" />

              <span className="text-[13px] text-[var(--text-dim)] hidden sm:block truncate">{projectName} / Settings</span>

              <div className="flex-1" />

              <div className="flex items-center gap-1 shrink-0">
                {tabs.map((tab) => {
                  const fullPath = `/projects/${projectId}/settings${tab.path}`;
                  const isActive = location.pathname === fullPath;
                  const Icon = tab.icon;
                  return (
                    <Link
                      key={tab.path}
                      to={fullPath}
                      className={`flex items-center gap-2 px-3.5 py-2 rounded-lg text-[13px] transition-colors ${
                        isActive
                          ? 'bg-[var(--primary)] text-white font-medium'
                          : 'text-[var(--text-dim)] hover:text-[var(--text-primary)]'
                      }`}
                    >
                      <Icon size={16} />
                      {tab.label}
                    </Link>
                  );
                })}
              </div>
            </div>

            {/* Page content */}
            {children}
          </div>
        </main>
      </div>

      {/* Optional right drawer */}
      {rightDrawer && (
        <aside className="hidden lg:block w-[680px] bg-[var(--bg-primary)] border-l border-[var(--border-primary)] shrink-0 overflow-y-auto">
          {rightDrawer}
        </aside>
      )}
    </div>
  );
}
