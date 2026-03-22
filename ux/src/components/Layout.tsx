import { type ReactNode, useCallback, useEffect, useRef, useState } from 'react';
import { useParams, useNavigate, useLocation, Link } from 'react-router-dom';
import { LayoutGrid, Users, Settings, Plus, AlertTriangle, Sun, Moon, BarChart3, Inbox, BookOpen, Container, Key, LogOut, UserCircle, ChevronUp } from 'lucide-react';
import { listFeaturesActiveOnly, getProject, getProjectSummary } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import { useTheme } from './ThemeContext';
import { useAuth } from './AuthContext';
import type { ProjectWithSummary, ProjectResponse, ProjectSummaryResponse } from '../lib/types';

interface LayoutProps {
  children: ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const { projectId } = useParams<{ projectId?: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, toggleTheme } = useTheme();
  const { user, logout } = useAuth();
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [parentProject, setParentProject] = useState<ProjectResponse | null>(null);
  const [activeFeatures, setActiveFeatures] = useState<ProjectWithSummary[]>([]);
  const [projectSummary, setProjectSummary] = useState<ProjectSummaryResponse | null>(null);
  // Track which features had updates since the user last visited them
  const [updatedFeatures, setUpdatedFeatures] = useState<Set<string>>(new Set());
  // The parent project ID to use for fetching features
  const [parentId, setParentId] = useState<string | null>(null);

  useEffect(() => {
    if (!projectId) {
      setProject(null);
      setParentProject(null);
      setActiveFeatures([]);
      setParentId(null);
      setProjectSummary(null);
      return;
    }

    getProject(projectId).then((proj) => {
      setProject(proj);
      // Always fetch active features from the ROOT project
      const rootId = proj.parent_id || projectId;
      setParentId(proj.parent_id ?? null);
      listFeaturesActiveOnly(rootId).then(setActiveFeatures).catch(() => setActiveFeatures([]));
      // Fetch parent project for breadcrumb title
      if (proj.parent_id) {
        getProject(proj.parent_id).then(setParentProject).catch(() => setParentProject(null));
      } else {
        setParentProject(null);
      }
    }).catch(() => {});
    getProjectSummary(projectId).then(setProjectSummary).catch(() => setProjectSummary(null));
  }, [projectId]);

  // Update document title based on breadcrumb
  useEffect(() => {
    if (project) {
      const parts: string[] = [];
      if (parentProject) parts.push(parentProject.name);
      parts.push(project.name);
      parts.push('Agach');
      document.title = parts.join(' - ');
    } else {
      document.title = 'Agach';
    }
  }, [project, parentProject]);

  // Clear update indicator when navigating to a feature
  useEffect(() => {
    if (projectId) {
      setUpdatedFeatures((prev) => {
        if (!prev.has(projectId)) return prev;
        const next = new Set(prev);
        next.delete(projectId);
        return next;
      });
    }
  }, [projectId]);

  // Listen for WebSocket events and mark features with updates
  useWebSocket(
    useCallback(
      (event) => {
        const eventProjectId = event.project_id;
        if (!eventProjectId) return;
        const type = event.type || '';
        if (!type.startsWith('task_') && !type.startsWith('comment_')) return;
        // Refresh summary if event is for the current project (backlog count may change)
        if (eventProjectId === projectId) {
          getProjectSummary(projectId).then(setProjectSummary).catch(() => {});
          return;
        }
        setActiveFeatures((features) => {
          if (features.some((f) => f.id === eventProjectId)) {
            setUpdatedFeatures((prev) => {
              if (prev.has(eventProjectId)) return prev;
              const next = new Set(prev);
              next.add(eventProjectId);
              return next;
            });
            const rootId = parentId || projectId;
            if (rootId) {
              listFeaturesActiveOnly(rootId).then(setActiveFeatures).catch(() => {});
            }
          }
          return features;
        });
      },
      [projectId, parentId],
    ),
  );

  // Close user menu on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false);
      }
    }
    if (userMenuOpen) document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [userMenuOpen]);

  const isActive = (path: string) => location.pathname === path;
  const isActivePrefix = (prefix: string) => location.pathname.startsWith(prefix);

  // Determine the "root" project ID for nav links (parent if we're in a feature)
  const navProjectId = parentId || projectId;

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-[220px] flex-shrink-0 bg-[var(--bg-secondary)] border-r border-[var(--border-primary)] flex flex-col">
        {/* Logo */}
        <Link to="/" data-qa="logo-home-link" className="flex items-center gap-2.5 h-[60px] px-4 border-b border-[var(--border-primary)] hover:opacity-80 transition-opacity flex-shrink-0">
          <div className="w-8 h-8 flex items-center justify-center">
            <img src={theme === 'dark' ? "/logo-dark.svg" : "/logo-light.svg"} alt="Agach" className="w-full h-auto" />
          </div>
          <span className="font-heading text-[17px] font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            Agach
          </span>
        </Link>

        {/* Scrollable middle section */}
        <div className="flex-1 overflow-y-auto">
        {/* Nav */}
        <nav className="flex flex-col gap-0.5 p-[12px_10px]">
          {projectId ? (
            <>
              <NavItem
                icon={<LayoutGrid size={15} />}
                label="Kanban"
                active={isActive(`/projects/${projectId}`) || isActive(`/projects/${projectId}/board`)}
                onClick={() => navigate(`/projects/${projectId}`)}
                data-qa="nav-kanban-btn"
              />
              <NavItem
                icon={<Inbox size={15} />}
                label="Backlog"
                active={isActive(`/projects/${projectId}/backlog`)}
                onClick={() => navigate(`/projects/${projectId}/backlog`)}
                data-qa="nav-backlog-btn"
                badge={(() => {
                  const own = projectSummary?.backlog_count ?? 0;
                  // When at root project (no parentId), add features' backlog counts
                  const childrenCount = !parentId
                    ? activeFeatures.reduce(
                        (sum, f) => sum + ((f.task_summary ?? f.summary)?.backlog_count ?? 0),
                        0
                      )
                    : 0;
                  const total = own + childrenCount;
                  return total > 0 ? total : undefined;
                })()}
              />
              <NavItem
                icon={<Users size={15} />}
                label="Roles"
                active={isActive(`/projects/${projectId}/roles`)}
                onClick={() => navigate(`/projects/${projectId}/roles`)}
                data-qa="nav-roles-btn"
              />
              <NavItem
                icon={<BarChart3 size={15} />}
                label="Statistics"
                active={isActive(`/projects/${projectId}/statistics`)}
                onClick={() => navigate(`/projects/${projectId}/statistics`)}
                data-qa="nav-statistics-btn"
              />
              <NavItem
                icon={<Settings size={15} />}
                label="Settings"
                active={isActivePrefix(`/projects/${projectId}/settings`)}
                onClick={() => navigate(`/projects/${projectId}/settings`)}
                data-qa="nav-settings-btn"
              />
            </>
          ) : (
            <>
              <NavItem
                icon={<LayoutGrid size={15} />}
                label="Projects"
                active={isActive('/')}
                onClick={() => navigate('/')}
                data-qa="nav-projects-btn"
              />
              <NavItem
                icon={<Users size={15} />}
                label="Roles"
                active={isActive('/roles')}
                onClick={() => navigate('/roles')}
                data-qa="nav-roles-btn"
              />
              <NavItem
                icon={<BookOpen size={15} />}
                label="Skills"
                active={isActive('/skills')}
                onClick={() => navigate('/skills')}
                data-qa="nav-skills-btn"
              />
              <NavItem
                icon={<Container size={15} />}
                label="Dockerfiles"
                active={isActive('/dockerfiles')}
                onClick={() => navigate('/dockerfiles')}
                data-qa="nav-dockerfiles-btn"
              />
            </>
          )}
        </nav>

        {/* Features section — only shown when there are active features */}
        {projectId && activeFeatures.length > 0 && (
          <>
            <div className="px-4 pt-4 pb-2">
              <span
                className="text-[10px] font-semibold tracking-[2px] text-[var(--text-muted)]"
                style={{ fontFamily: 'JetBrains Mono, monospace' }}
              >
                FEATURES
              </span>
            </div>
            <div className="flex flex-col gap-0.5 px-2.5 pb-2">
              {activeFeatures.map((feat) => {
                const isCurrentFeature = feat.id === projectId;
                const hasUpdate = updatedFeatures.has(feat.id);
                const summary = feat.task_summary ?? feat.summary;
                const inProgress = summary?.in_progress_count ?? 0;
                const blocked = summary?.blocked_count ?? 0;
                const total =
                  (summary?.todo_count ?? 0) +
                  inProgress +
                  (summary?.done_count ?? 0) +
                  blocked;

                let dotColor = 'var(--text-muted)';
                if (isCurrentFeature) dotColor = 'var(--primary)';
                else if (hasUpdate) dotColor = theme === 'dark' ? '#F59E0B' : '#E07B54';
                else if (blocked > 0) dotColor = theme === 'dark' ? '#F06060' : '#D94040';
                else if (inProgress > 0) dotColor = 'var(--primary)';

                return (
                  <button
                    key={feat.id}
                    onClick={() => navigate(`/projects/${feat.id}`)}
                    data-qa="nav-feature-btn"
                    className={`flex items-center gap-2.5 h-10 px-2.5 rounded-md w-full text-left transition-colors cursor-pointer ${
                      isCurrentFeature
                        ? 'bg-[var(--nav-bg-active)]'
                        : 'hover:bg-[var(--nav-bg-active)]/50'
                    }`}
                  >
                    <div className="relative flex-shrink-0">
                      <div
                        className={`w-[7px] h-[7px] rounded-full ${
                          hasUpdate && !isCurrentFeature ? 'animate-pulse' : ''
                        }`}
                        style={{ backgroundColor: dotColor }}
                      />
                    </div>
                    <span
                      className={`text-[13px] flex-1 truncate ${
                        isCurrentFeature
                          ? 'text-[var(--nav-text-active)]'
                          : 'text-[var(--text-secondary)]'
                      }`}
                      style={{ fontFamily: 'Inter, sans-serif' }}
                    >
                      {feat.name}
                    </span>
                    {blocked > 0 && (
                      <AlertTriangle size={12} className="text-[#FF3B30] shrink-0" />
                    )}
                    <span
                      className="text-[10px] text-[var(--text-muted)]"
                      style={{ fontFamily: 'JetBrains Mono, monospace' }}
                    >
                      {total}
                    </span>
                  </button>
                );
              })}
              <button
                className="flex items-center gap-2 h-9 px-2.5 rounded-md w-full text-left hover:bg-[var(--nav-bg-active)]/50 transition-colors cursor-pointer group"
                onClick={() => navigate(`/projects/${navProjectId}/features`)}
                data-qa="nav-add-feature-btn"
              >
                <Plus
                  size={13}
                  className="text-[var(--text-muted)] group-hover:text-[var(--text-secondary)]"
                />
                <span
                  className="text-[12px] text-[var(--text-muted)] group-hover:text-[var(--text-secondary)]"
                  style={{ fontFamily: 'Inter, sans-serif' }}
                >
                  Add feature
                </span>
              </button>
            </div>
          </>
        )}

        </div>

        {/* User Menu */}
        <div ref={userMenuRef} className="p-[12px_10px] border-t border-[var(--border-primary)] relative">
          {/* Popup menu */}
          {userMenuOpen && (
            <div className="absolute bottom-full left-2 right-2 mb-1 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-secondary)] shadow-lg overflow-hidden z-50">
              {/* User info header */}
              <div className="px-3 py-2.5 border-b border-[var(--border-primary)]">
                <p className="text-[12px] font-medium text-[var(--text-primary)] truncate" style={{ fontFamily: 'Inter, sans-serif' }}>
                  {user?.display_name || user?.email}
                </p>
                <p className="text-[11px] text-[var(--text-muted)] truncate" style={{ fontFamily: 'Inter, sans-serif' }}>
                  {user?.email}
                </p>
              </div>
              {/* Theme toggle */}
              <button
                onClick={() => { toggleTheme(); setUserMenuOpen(false); }}
                data-qa="theme-toggle-btn"
                className="flex items-center gap-2.5 w-full px-3 py-2 text-[13px] text-[var(--text-secondary)] hover:bg-[var(--nav-bg-active)]/50 hover:text-[var(--text-primary)] transition-colors cursor-pointer"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
                {theme === 'dark' ? 'Light Mode' : 'Dark Mode'}
              </button>
              {/* Account */}
              <button
                onClick={() => { navigate('/account'); setUserMenuOpen(false); }}
                data-qa="user-menu-account-btn"
                className="flex items-center gap-2.5 w-full px-3 py-2 text-[13px] text-[var(--text-secondary)] hover:bg-[var(--nav-bg-active)]/50 hover:text-[var(--text-primary)] transition-colors cursor-pointer"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                <UserCircle size={14} />
                Account
              </button>
              {/* API Keys */}
              <button
                onClick={() => { navigate('/account/api-keys'); setUserMenuOpen(false); }}
                data-qa="user-menu-api-keys-btn"
                className="flex items-center gap-2.5 w-full px-3 py-2 text-[13px] text-[var(--text-secondary)] hover:bg-[var(--nav-bg-active)]/50 hover:text-[var(--text-primary)] transition-colors cursor-pointer"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                <Key size={14} />
                API Keys
              </button>
              {/* Divider + Logout */}
              <div className="border-t border-[var(--border-primary)]">
                <button
                  onClick={async () => { setUserMenuOpen(false); await logout(); navigate('/login'); }}
                  data-qa="user-menu-logout-btn"
                  className="flex items-center gap-2.5 w-full px-3 py-2 text-[13px] text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer"
                  style={{ fontFamily: 'Inter, sans-serif' }}
                >
                  <LogOut size={14} />
                  Sign out
                </button>
              </div>
            </div>
          )}

          {/* Trigger button */}
          <button
            onClick={() => setUserMenuOpen(v => !v)}
            data-qa="user-menu-btn"
            className="flex items-center gap-2.5 h-9 px-2.5 rounded-md w-full text-left hover:bg-[var(--nav-bg-active)]/50 transition-colors cursor-pointer text-[var(--text-secondary)]"
          >
            <div className="w-6 h-6 rounded-full bg-[var(--primary)] flex items-center justify-center flex-shrink-0">
              <span className="text-[11px] font-bold text-white" style={{ fontFamily: 'Inter, sans-serif' }}>
                {(user?.display_name || user?.email || '?')[0].toUpperCase()}
              </span>
            </div>
            <span className="text-[13px] font-medium flex-1 truncate" style={{ fontFamily: 'Inter, sans-serif' }}>
              {user?.display_name || user?.email}
            </span>
            <ChevronUp size={13} className={`flex-shrink-0 transition-transform ${userMenuOpen ? '' : 'rotate-180'}`} />
          </button>
        </div>
      </aside>

      {/* Main area */}
      <main className="flex-1 bg-[var(--bg-primary)] overflow-hidden flex flex-col">
        {children}
      </main>
    </div>
  );
}

function NavItem({ icon, label, active, onClick, badge, 'data-qa': dataQa }: { icon: ReactNode; label: string; active: boolean; onClick: () => void; badge?: number; 'data-qa'?: string }) {
  return (
    <button
      onClick={onClick}
      data-qa={dataQa}
      className={`flex items-center gap-2.5 h-9 px-2.5 rounded-md w-full text-left transition-colors cursor-pointer ${
        active ? 'bg-[var(--nav-bg-active)] text-[var(--nav-text-active)]' : 'text-[var(--text-secondary)] hover:bg-[var(--nav-bg-active)]/50 hover:text-[var(--text-primary)]'
      }`}
    >
      <span className={active ? 'text-[var(--nav-text-active)]' : 'text-[var(--text-secondary)]'}>{icon}</span>
      <span className="text-[13px] font-medium flex-1" style={{ fontFamily: 'Inter, sans-serif' }}>
        {label}
      </span>
      {badge !== undefined && (
        <span className="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-[var(--border-primary)] text-[var(--text-muted)]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
          {badge}
        </span>
      )}
    </button>
  );
}
