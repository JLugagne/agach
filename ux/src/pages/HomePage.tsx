import { useState, useEffect, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { FolderOpen, Loader2, ArrowRight, ChevronDown, ChevronRight } from 'lucide-react';
import { listProjects, listRoles } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { ProjectWithSummary, RoleResponse, ProjectSummaryResponse } from '../lib/types';

function getSummary(p: ProjectWithSummary): ProjectSummaryResponse {
  return p.task_summary || p.summary || { todo_count: 0, in_progress_count: 0, done_count: 0, blocked_count: 0 };
}

function getChildrenCount(p: ProjectWithSummary): number {
  return p.children_count ?? 0;
}

function shortenPath(path: string): string {
  const home = '/home/';
  const idx = path.indexOf(home);
  if (idx >= 0) {
    const afterHome = path.substring(idx + home.length);
    const slashIdx = afterHome.indexOf('/');
    if (slashIdx >= 0) {
      return '~' + afterHome.substring(slashIdx);
    }
    return '~/' + afterHome;
  }
  return path;
}

function getStatus(summary: ProjectSummaryResponse): { label: string; color: string; bg: string } {
  if (summary.in_progress_count > 0) {
    return { label: 'Active', color: 'var(--status-progress)', bg: 'var(--status-progress-bg)' };
  }
  if (summary.blocked_count > 0) {
    return { label: 'Blocked', color: 'var(--status-blocked)', bg: 'var(--status-blocked-bg)' };
  }
  if (summary.todo_count > 0) {
    return { label: 'Pending', color: 'var(--status-todo)', bg: 'var(--status-todo-bg)' };
  }
  if (summary.done_count > 0) {
    return { label: 'Done', color: 'var(--status-done)', bg: 'var(--status-done-bg)' };
  }
  return { label: 'Empty', color: 'var(--text-muted)', bg: 'var(--bg-tertiary)' };
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export default function HomePage() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectWithSummary[]>([]);
  const [roles, setRoles] = useState<RoleResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [collapsedDirs, setCollapsedDirs] = useState<Set<string>>(new Set());

  const fetchData = useCallback(async () => {
    try {
      const [p, r] = await Promise.all([listProjects(), listRoles()]);
      setProjects(p ?? []);
      setRoles(r ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useWebSocket(
    useCallback(
      (event) => {
        if (
          event.type === 'project_created' ||
          event.type === 'project_updated' ||
          event.type === 'project_deleted'
        ) {
          fetchData();
        }
      },
      [fetchData],
    ),
  );

  // Group projects by work_dir
  const grouped = useMemo(() => {
    const map = new Map<string, ProjectWithSummary[]>();
    for (const p of projects) {
      const dir = p.work_dir || '(no directory)';
      if (!map.has(dir)) map.set(dir, []);
      map.get(dir)!.push(p);
    }
    // Sort groups by directory name
    return Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b));
  }, [projects]);

  const totalProjects = projects.length;
  const totalDirs = grouped.length;

  const toggleDir = (dir: string) => {
    setCollapsedDirs((prev) => {
      const next = new Set(prev);
      if (next.has(dir)) next.delete(dir);
      else next.add(dir);
      return next;
    });
  };

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-5xl mx-auto px-8 py-12">
        {/* Header */}
        <div className="mb-2">
          <h1 className="text-[28px] font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            My Projects
          </h1>
        </div>
        <p className="text-xs text-[var(--text-muted)] mb-10" style={{ fontFamily: 'Inter, sans-serif' }}>
          {totalDirs} folder{totalDirs !== 1 ? 's' : ''}
          {' \u00B7 '}
          {totalProjects} project{totalProjects !== 1 ? 's' : ''}
        </p>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
          </div>
        ) : projects.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-3">
            <FolderOpen size={32} className="text-[var(--text-dim)]" />
            <p className="text-sm text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              No projects yet. Create one to get started.
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-6">
            {grouped.map(([dir, dirProjects]) => (
              <FolderGroup
                key={dir}
                dir={dir}
                projects={dirProjects}
                roles={roles}
                collapsed={collapsedDirs.has(dir)}
                onToggle={() => toggleDir(dir)}
                onOpen={(id) => navigate(`/projects/${id}/board`)}
              />
            ))}
          </div>
        )}
      </div>

    </div>
  );
}

function FolderGroup({
  dir,
  projects,
  roles,
  collapsed,
  onToggle,
  onOpen,
}: {
  dir: string;
  projects: ProjectWithSummary[];
  roles: RoleResponse[];
  collapsed: boolean;
  onToggle: () => void;
  onOpen: (id: string) => void;
}) {
  const displayDir = dir === '(no directory)' ? dir : shortenPath(dir);

  return (
    <div>
      {/* Folder header */}
      <button
        onClick={onToggle}
        className="flex items-center gap-3 w-full px-4 py-3 rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-primary)] hover:border-[var(--border-secondary)] transition-colors cursor-pointer"
      >
        <FolderOpen size={16} className="text-[var(--text-secondary)] flex-shrink-0" />
        <span className="text-[13px] text-[var(--text-primary)] opacity-80" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
          {displayDir}
        </span>
        <span
          className="text-[10px] font-medium px-2 py-0.5 rounded-full flex-shrink-0"
          style={{
            fontFamily: 'Inter, sans-serif',
            color: 'var(--primary)',
            backgroundColor: 'color-mix(in srgb, var(--primary) 15%, transparent)',
          }}
        >
          {projects.length} project{projects.length !== 1 ? 's' : ''}
        </span>
        <div className="flex-1" />
        {collapsed ? (
          <ChevronRight size={14} className="text-[var(--text-muted)]" />
        ) : (
          <ChevronDown size={14} className="text-[var(--text-muted)]" />
        )}
      </button>

      {/* Project cards */}
      {!collapsed && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4 pl-2">
          {projects.map((project) => (
            <ProjectCard
              key={project.id}
              project={project}
              roles={roles}
              onOpen={() => onOpen(project.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ProjectCard({
  project,
  roles,
  onOpen,
}: {
  project: ProjectWithSummary;
  roles: RoleResponse[];
  onOpen: () => void;
}) {
  const summary = getSummary(project);
  const childrenCount = getChildrenCount(project);
  const status = getStatus(summary);

  // Find role by slug
  const role = project.created_by_role
    ? roles.find((r) => r.slug === project.created_by_role)
    : null;

  return (
    <div className="rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-primary)] p-5 flex flex-col gap-3 hover:border-[var(--border-secondary)] transition-colors group">
      {/* Top row: role badge + status */}
      <div className="flex items-center justify-between">
        {role ? (
          <span
            className="text-[10px] font-medium px-2 py-0.5 rounded-full"
            style={{
              fontFamily: 'Inter, sans-serif',
              color: role.color || 'var(--text-secondary)',
              backgroundColor: `color-mix(in srgb, ${role.color || 'var(--text-secondary)'} 15%, transparent)`,
            }}
          >
            {role.name}
          </span>
        ) : (
          <div />
        )}
        <span
          className="text-[10px] font-medium px-2 py-0.5 rounded-full"
          style={{
            fontFamily: 'Inter, sans-serif',
            color: status.color,
            backgroundColor: status.bg,
          }}
        >
          {status.label}
        </span>
      </div>

      {/* Name + description */}
      <div>
        <h3 className="text-[15px] font-medium text-[var(--text-primary)] truncate group-hover:text-[var(--primary)] transition-colors" style={{ fontFamily: 'Inter, sans-serif' }}>
          {project.name}
        </h3>
        {project.description && (
          <p className="text-xs text-[var(--text-secondary)] mt-1 line-clamp-2" style={{ fontFamily: 'Inter, sans-serif' }}>
            {project.description}
          </p>
        )}
      </div>

      {/* Meta row */}
      <p className="text-[11px] text-[var(--text-dim)]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
        Created {formatDate(project.created_at)}
        {' \u00B7 '}
        Modified {formatDate(project.updated_at)}
        {childrenCount > 0 && (
          <>
            {' \u00B7 '}
            {childrenCount} sub-project{childrenCount !== 1 ? 's' : ''}
          </>
        )}
      </p>

      {/* Open link */}
      <button
        onClick={onOpen}
        className="flex items-center gap-1 text-xs text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors cursor-pointer self-start mt-auto"
        style={{ fontFamily: 'Inter, sans-serif' }}
      >
        Open <ArrowRight size={12} />
      </button>
    </div>
  );
}
