import { type ReactNode, useCallback, useEffect, useState } from 'react';
import { useParams, useNavigate, useLocation, Link } from 'react-router-dom';
import { LayoutGrid, Users, Settings, Plus, AlertTriangle, Sun, Moon, BarChart3, Inbox, BookOpen } from 'lucide-react';
import { listSubProjects, getProject, getProjectSummary } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import { useTheme } from './ThemeContext';
import type { ProjectWithSummary, ProjectResponse, ProjectSummaryResponse } from '../lib/types';

interface LayoutProps {
  children: ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const { projectId } = useParams<{ projectId?: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, toggleTheme } = useTheme();
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [parentProject, setParentProject] = useState<ProjectResponse | null>(null);
  const [subProjects, setSubProjects] = useState<ProjectWithSummary[]>([]);
  const [projectSummary, setProjectSummary] = useState<ProjectSummaryResponse | null>(null);
  // Track which sub-projects had updates since the user last visited them
  const [updatedSubProjects, setUpdatedSubProjects] = useState<Set<string>>(new Set());
  // The parent project ID to use for fetching sub-projects
  const [parentId, setParentId] = useState<string | null>(null);

  useEffect(() => {
    if (!projectId) {
      setProject(null);
      setParentProject(null);
      setSubProjects([]);
      setParentId(null);
      setProjectSummary(null);
      return;
    }

    getProject(projectId).then((proj) => {
      setProject(proj);
      // If this project has a parent, fetch siblings (parent's children)
      const listFrom = proj.parent_id || projectId;
      setParentId(proj.parent_id);
      listSubProjects(listFrom).then(setSubProjects).catch(() => setSubProjects([]));
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

  // Clear update indicator when navigating to a sub-project
  useEffect(() => {
    if (projectId) {
      setUpdatedSubProjects((prev) => {
        if (!prev.has(projectId)) return prev;
        const next = new Set(prev);
        next.delete(projectId);
        return next;
      });
    }
  }, [projectId]);

  // Listen for WebSocket events and mark sub-projects with updates
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
        setSubProjects((sps) => {
          if (sps.some((sp) => sp.id === eventProjectId)) {
            setUpdatedSubProjects((prev) => {
              if (prev.has(eventProjectId)) return prev;
              const next = new Set(prev);
              next.add(eventProjectId);
              return next;
            });
            // Refresh sub-project summaries
            const listFrom = parentId || projectId;
            if (listFrom) {
              listSubProjects(listFrom).then(setSubProjects).catch(() => {});
            }
          }
          return sps;
        });
      },
      [projectId, parentId],
    ),
  );

  const isActive = (path: string) => location.pathname === path;
  const isActivePrefix = (prefix: string) => location.pathname.startsWith(prefix);

  // Determine the "root" project ID for nav links (parent if we're in a sub-project)
  const navProjectId = parentId || projectId;

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-[220px] flex-shrink-0 bg-[var(--bg-secondary)] border-r border-[var(--border-primary)] flex flex-col">
        {/* Logo */}
        <Link to="/" className="flex items-center gap-2.5 h-[60px] px-4 border-b border-[var(--border-primary)] hover:opacity-80 transition-opacity flex-shrink-0">
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
              />
              <NavItem
                icon={<Inbox size={15} />}
                label="Backlog"
                active={isActive(`/projects/${projectId}/backlog`)}
                onClick={() => navigate(`/projects/${projectId}/backlog`)}
                badge={(() => {
                  const own = projectSummary?.backlog_count ?? 0;
                  // When at root project (no parentId), add children's backlog counts
                  const childrenCount = !parentId
                    ? subProjects.reduce((sum, sp) => sum + ((sp.task_summary ?? sp.summary)?.backlog_count ?? 0), 0)
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
              />
              <NavItem
                icon={<BarChart3 size={15} />}
                label="Statistics"
                active={isActive(`/projects/${projectId}/statistics`)}
                onClick={() => navigate(`/projects/${projectId}/statistics`)}
              />
              <NavItem
                icon={<Settings size={15} />}
                label="Settings"
                active={isActivePrefix(`/projects/${projectId}/settings`)}
                onClick={() => navigate(`/projects/${projectId}/settings`)}
              />
            </>
          ) : (
            <>
              <NavItem
                icon={<LayoutGrid size={15} />}
                label="Projects"
                active={isActive('/')}
                onClick={() => navigate('/')}
              />
              <NavItem
                icon={<Users size={15} />}
                label="Roles"
                active={isActive('/roles')}
                onClick={() => navigate('/roles')}
              />
              <NavItem
                icon={<BookOpen size={15} />}
                label="Skills"
                active={isActive('/skills')}
                onClick={() => navigate('/skills')}
              />
              <NavItem
                icon={<Settings size={15} />}
                label="Settings"
                active={false}
                onClick={() => {}}
              />
            </>
          )}
        </nav>

        {/* Sub-projects section */}
        {projectId && subProjects.length > 0 && (
          <>
            <div className="px-4 pt-4 pb-2">
              <span className="text-[10px] font-semibold tracking-[2px] text-[var(--text-muted)]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                SUB-PROJECTS
              </span>
            </div>
            <div className="flex flex-col gap-0.5 px-2.5 pb-2">
              {subProjects.map((sp) => {
                const isCurrentSubProject = sp.id === projectId;
                const hasUpdate = updatedSubProjects.has(sp.id);
                const summary = sp.task_summary ?? sp.summary;
                const inProgress = summary?.in_progress_count ?? 0;
                const blocked = summary?.blocked_count ?? 0;
                const total = (summary?.todo_count ?? 0) + inProgress + (summary?.done_count ?? 0) + blocked;
                
                let dotColor = 'var(--text-muted)';
                if (isCurrentSubProject) dotColor = 'var(--primary)';
                else if (hasUpdate) dotColor = theme === 'dark' ? '#F59E0B' : '#E07B54';
                else if (blocked > 0) dotColor = theme === 'dark' ? '#F06060' : '#D94040';
                else if (inProgress > 0) dotColor = 'var(--primary)';

                return (
                  <button
                    key={sp.id}
                    onClick={() => navigate(`/projects/${sp.id}`)}
                    className={`flex items-center gap-2.5 h-10 px-2.5 rounded-md w-full text-left transition-colors cursor-pointer ${
                      isCurrentSubProject ? 'bg-[var(--nav-bg-active)]' : 'hover:bg-[var(--nav-bg-active)]/50'
                    }`}
                  >
                    <div className="relative flex-shrink-0">
                      <div
                        className={`w-[7px] h-[7px] rounded-full ${hasUpdate && !isCurrentSubProject ? 'animate-pulse' : ''}`}
                        style={{ backgroundColor: dotColor }}
                      />
                    </div>
                    <span
                      className={`text-[13px] flex-1 truncate ${
                        isCurrentSubProject ? 'text-[var(--nav-text-active)]' : 'text-[var(--text-secondary)]'
                      }`}
                      style={{ fontFamily: 'Inter, sans-serif' }}
                    >
                      {sp.name}
                    </span>
                    {blocked > 0 && (
                      <AlertTriangle size={12} className="text-[#FF3B30] shrink-0" />
                    )}
                    <span className="text-[10px] text-[var(--text-muted)]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                      {total}
                    </span>
                  </button>
                );
              })}
              <button
                className="flex items-center gap-2 h-9 px-2.5 rounded-md w-full text-left hover:bg-[var(--nav-bg-active)]/50 transition-colors cursor-pointer group"
                onClick={() => navigate(`/projects/${navProjectId}/settings/sub-projects`)}
              >
                <Plus size={13} className="text-[var(--text-muted)] group-hover:text-[var(--text-secondary)]" />
                <span className="text-[12px] text-[var(--text-muted)] group-hover:text-[var(--text-secondary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                  Add sub-project
                </span>
              </button>
            </div>
          </>
        )}

        </div>

        {/* Theme Switcher */}
        <div className="p-[12px_10px] border-t border-[var(--border-primary)]">
          <button
            onClick={toggleTheme}
            className="flex items-center gap-2.5 h-9 px-2.5 rounded-md w-full text-left hover:bg-[var(--nav-bg-active)]/50 transition-colors cursor-pointer text-[var(--text-secondary)]"
          >
            {theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}
            <span className="text-[13px] font-medium" style={{ fontFamily: 'Inter, sans-serif' }}>
              {theme === 'dark' ? 'Light Mode' : 'Dark Mode'}
            </span>
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

function NavItem({ icon, label, active, onClick, badge }: { icon: ReactNode; label: string; active: boolean; onClick: () => void; badge?: number }) {
  return (
    <button
      onClick={onClick}
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
