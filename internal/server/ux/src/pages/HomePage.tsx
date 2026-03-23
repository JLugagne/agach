import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { FolderOpen, Loader2, ArrowRight, Plus } from 'lucide-react';
import { listProjects, listAgents } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { ProjectWithSummary, AgentResponse, ProjectSummaryResponse } from '../lib/types';
import CreateProjectDialog from '../components/CreateProjectDialog';

function getSummary(p: ProjectWithSummary): ProjectSummaryResponse {
  return p.task_summary || p.summary || { todo_count: 0, in_progress_count: 0, done_count: 0, blocked_count: 0 };
}

function getChildrenCount(p: ProjectWithSummary): number {
  return p.children_count ?? 0;
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
  const [roles, setRoles] = useState<AgentResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [p, r] = await Promise.all([listProjects(), listAgents()]);
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

  const totalProjects = projects.length;

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-5xl mx-auto px-4 sm:px-8 py-8 sm:py-12">
        {/* Header */}
        <div className="flex items-center justify-between mb-2 gap-4">
          <h1 className="text-xl sm:text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            My Projects
          </h1>
          <button
            onClick={() => setShowCreateDialog(true)}
            data-qa="create-project-btn"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Project
          </button>
        </div>
        <p className="text-sm text-[var(--text-muted)] mb-10" style={{ fontFamily: 'Inter, sans-serif' }}>
          {totalProjects} project{totalProjects !== 1 ? 's' : ''}
        </p>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
          </div>
        ) : projects.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-5">
            <div className="w-20 h-20 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
              <FolderOpen size={36} className="text-[var(--text-muted)]" />
            </div>
            <p className="text-lg font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              No projects yet.
            </p>
            <p className="text-sm text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Get started by creating your first project
            </p>
            <button
              onClick={() => setShowCreateDialog(true)}
              data-qa="create-project-empty-btn"
              className="flex items-center gap-2 px-6 py-3 rounded-lg text-sm font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              <Plus size={16} />
              Create your first project
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                roles={roles}
                onOpen={() => navigate(`/projects/${project.id}/board`)}
              />
            ))}
          </div>
        )}
      </div>

      {showCreateDialog && (
        <CreateProjectDialog
          onClose={() => setShowCreateDialog(false)}
          onCreated={fetchData}
        />
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
  roles: AgentResponse[];
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
    <div data-qa="project-card" className="rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-primary)] p-5 flex flex-col gap-3 hover:border-[var(--border-secondary)] transition-colors group min-w-0 overflow-hidden">
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
      <p className="text-[11px] text-[var(--text-dim)] break-words" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
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
        data-qa="project-open-btn"
        className="flex items-center gap-1 text-xs text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors cursor-pointer self-start mt-auto"
        style={{ fontFamily: 'Inter, sans-serif' }}
      >
        Open <ArrowRight size={12} />
      </button>
    </div>
  );
}
