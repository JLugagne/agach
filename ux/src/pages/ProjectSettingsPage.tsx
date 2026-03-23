import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { Loader2, AlertTriangle, Star } from 'lucide-react';
import { getProject, updateProject, deleteProject, listProjectAgents, listColumns, updateColumnWIPLimit } from '../lib/api';
import SettingsLayout from '../components/settings/SettingsLayout';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import AddAgentToProjectDialog from '../components/AddAgentToProjectDialog';
import RemoveAgentDialog from '../components/RemoveAgentDialog';
import { useWebSocket } from '../hooks/useWebSocket';
import type { ProjectResponse, ColumnResponse, AgentResponse, WSEvent } from '../lib/types';

function ProjectAgentsSection({ projectId, defaultRole, onDefaultRoleChange }: { projectId: string; defaultRole: string; onDefaultRoleChange: (slug: string) => void }) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<AgentResponse | null>(null);

  const fetchAgents = useCallback(async () => {
    try {
      const data = await listProjectAgents(projectId);
      setAgents(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => { fetchAgents(); }, [fetchAgents]);

  useWebSocket(useCallback((event: WSEvent) => {
    if (event.type === 'agent_assigned_to_project' || event.type === 'agent_removed_from_project') {
      fetchAgents();
    }
  }, [fetchAgents]));

  const handleSetDefault = async (slug: string) => {
    const newDefault = slug === defaultRole ? '' : slug;
    onDefaultRoleChange(newDefault);
    try {
      await updateProject(projectId, { default_role: newDefault });
    } catch {
      // ignore
    }
  };

  return (
    <div className="mb-12">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Agents</h2>
        <button
          onClick={() => setAddDialogOpen(true)}
          data-qa="add-agent-btn"
          className="px-3 py-1.5 bg-[var(--bg-secondary)] border border-[var(--border-primary)] text-sm text-[var(--text-muted)] rounded-md hover:text-[var(--text-primary)] hover:border-[var(--border-secondary)] transition-colors"
        >
          + Add Agent
        </button>
      </div>

      {loading ? (
        <div className="flex items-center gap-2 text-[var(--text-dim)] text-sm">
          <Loader2 className="animate-spin" size={14} />
          <span>Loading agents...</span>
        </div>
      ) : agents.length === 0 ? (
        <p className="text-sm text-[var(--text-dim)] italic">No agents assigned yet.</p>
      ) : (
        <div className="space-y-1">
          {agents.map(agent => (
            <div key={agent.slug} className="flex items-center justify-between py-2.5 px-3 rounded-md bg-[var(--bg-primary)] border border-[var(--border-primary)]">
              <div className="flex items-center gap-2.5">
                <span
                  className="w-2.5 h-2.5 rounded-full shrink-0"
                  style={{ backgroundColor: agent.color || '#6B7280' }}
                />
                <span className="text-sm text-[var(--text-primary)]">{agent.name}</span>
                <span className="text-xs text-[var(--text-dim)] font-mono">{agent.slug}</span>
                {agent.slug === defaultRole && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--primary)]/15 text-[var(--primary)] font-mono">
                    default
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleSetDefault(agent.slug)}
                  data-qa="set-default-agent-btn"
                  title={agent.slug === defaultRole ? 'Remove as default' : 'Set as default'}
                  className={`p-1 rounded transition-colors ${
                    agent.slug === defaultRole
                      ? 'text-[var(--primary)]'
                      : 'text-[var(--text-dim)] hover:text-[var(--primary)]'
                  }`}
                >
                  <Star size={14} fill={agent.slug === defaultRole ? 'currentColor' : 'none'} />
                </button>
                <button
                  onClick={() => setRemoveTarget(agent)}
                  data-qa="remove-agent-btn"
                  className="text-xs text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors px-2 py-0.5 rounded"
                >
                  Remove
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {addDialogOpen && (
        <AddAgentToProjectDialog
          projectId={projectId}
          assignedSlugs={new Set(agents.map(a => a.slug))}
          onClose={() => setAddDialogOpen(false)}
          onSuccess={fetchAgents}
        />
      )}

      {removeTarget && (
        <RemoveAgentDialog
          projectId={projectId}
          agent={removeTarget}
          projectAgents={agents}
          onClose={() => setRemoveTarget(null)}
          onSuccess={() => {
            setRemoveTarget(null);
            fetchAgents();
          }}
        />
      )}
    </div>
  );
}

export default function ProjectSettingsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const isAgentsTab = location.pathname.endsWith('/agents');
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [gitUrl, setGitUrl] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [defaultRole, setDefaultRole] = useState<string>('');
  const [columns, setColumns] = useState<ColumnResponse[]>([]);
  const [wipLimits, setWipLimits] = useState<Record<string, number>>({});
  const [wipSaving, setWipSaving] = useState(false);
  const [wipSaved, setWipSaved] = useState(false);

  const fetchProject = useCallback(async () => {
    if (!projectId) return;
    try {
      const [p, cols] = await Promise.all([
        getProject(projectId),
        listColumns(projectId).catch(() => [] as ColumnResponse[]),
      ]);
      setProject(p);
      setName(p.name);
      setDescription(p.description);
      setGitUrl(p.git_url || '');
      setDefaultRole(p.default_role ?? '');
      setColumns(cols);
      const limits: Record<string, number> = {};
      for (const col of cols) {
        limits[col.slug] = col.wip_limit;
      }
      setWipLimits(limits);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchProject();
  }, [fetchProject]);

  const handleSave = async () => {
    if (!projectId || !name.trim()) return;
    setSaving(true);
    setSaved(false);
    try {
      await updateProject(projectId, {
        name: name.trim(),
        description: description.trim(),
        git_url: gitUrl.trim(),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      fetchProject();
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleSaveWipLimits = async () => {
    if (!projectId) return;
    setWipSaving(true);
    setWipSaved(false);
    try {
      await Promise.all(
        columns
          .filter((col) => wipLimits[col.slug] !== col.wip_limit)
          .map((col) => updateColumnWIPLimit(projectId, col.slug, wipLimits[col.slug] ?? 0)),
      );
      setWipSaved(true);
      setTimeout(() => setWipSaved(false), 2000);
      fetchProject();
    } catch {
      // ignore
    } finally {
      setWipSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!projectId) return;
    setDeleting(true);
    try {
      await deleteProject(projectId);
      navigate('/');
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--bg-secondary)] flex items-center justify-center">
        <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="min-h-screen bg-[var(--bg-secondary)] flex items-center justify-center">
        <p className="text-[var(--text-dim)]">Project not found</p>
      </div>
    );
  }

  return (
    <SettingsLayout projectName={project.name}>
      {isAgentsTab ? (
        <>
          <h1 className="text-xl font-semibold text-[var(--text-primary)] mb-8" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Agents</h1>
          {projectId && (
            <ProjectAgentsSection
              projectId={projectId}
              defaultRole={defaultRole}
              onDefaultRoleChange={setDefaultRole}
            />
          )}
        </>
      ) : (
        <>
          <h1 className="text-xl font-semibold text-[var(--text-primary)] mb-8" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Project Settings</h1>

          {/* Project Definition Section */}
          <section className="mb-12">
            <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Project</h2>

            {/* Name */}
            <div className="mb-5">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                data-qa="project-name-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
              />
            </div>

            {/* Description */}
            <div className="mb-5">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this project..."
                rows={5}
                data-qa="project-description-textarea"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
              />
            </div>

            {/* Git URL */}
            <div className="mb-6">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Git URL</label>
              <input
                type="text"
                value={gitUrl}
                onChange={(e) => setGitUrl(e.target.value)}
                placeholder="https://github.com/org/repo.git"
                data-qa="project-git-url-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 font-mono text-xs"
              />
            </div>

            {/* Save button */}
            <button
              onClick={handleSave}
              disabled={!name.trim() || saving}
              data-qa="save-project-settings-btn"
              className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving...' : saved ? 'Saved' : 'Save Changes'}
            </button>
          </section>

          {/* WIP Limits */}
          {columns.length > 0 && (
            <section className="mb-12">
              <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Column WIP Limits</h2>
              <p className="text-xs text-[var(--text-muted)] mb-4">
                Set to 0 for no limit. Agents will be prevented from moving tasks into a column that has reached its limit.
              </p>
              <div className="grid grid-cols-2 gap-4 mb-4">
                {columns.map((col) => (
                  <div key={col.slug}>
                    <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">
                      {col.name}
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={wipLimits[col.slug] ?? 0}
                      onChange={(e) =>
                        setWipLimits((prev) => ({
                          ...prev,
                          [col.slug]: Math.max(0, parseInt(e.target.value) || 0),
                        }))
                      }
                      data-qa={`wip-limit-${col.slug}-input`}
                      className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
                    />
                  </div>
                ))}
              </div>
              <button
                onClick={handleSaveWipLimits}
                disabled={wipSaving}
                data-qa="save-wip-limits-btn"
                className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
              >
                {wipSaving ? 'Saving...' : wipSaved ? 'Saved' : 'Save WIP Limits'}
              </button>
            </section>
          )}

          {/* Danger Zone */}
          <section className="border border-[#FF3B30]/30 rounded-lg p-5">
            <div className="flex items-start gap-3">
              <AlertTriangle size={18} className="text-[#FF3B30] mt-0.5 shrink-0" />
              <div className="flex-1">
                <h3 className="text-sm text-[var(--text-primary)] font-medium mb-1">Delete this project</h3>
                <p className="text-xs text-[var(--text-muted)] mb-4">
                  Once you delete a project, there is no going back. All tasks, comments, and data will
                  be permanently removed.
                </p>
                <button
                  onClick={() => setDeleteOpen(true)}
                  data-qa="delete-project-btn"
                  className="px-4 py-2 bg-[#FF3B30] text-white text-sm font-medium rounded-md hover:bg-[#FF3B30]/80 transition-colors"
                >
                  Delete Project
                </button>
              </div>
            </div>
          </section>

          <DeleteConfirmModal
            open={deleteOpen}
            title="Delete Project?"
            description={`This will permanently delete "${project.name}" and all its tasks, comments, and data. This action cannot be undone.`}
            confirmLabel="Delete Project"
            onConfirm={handleDelete}
            onCancel={() => setDeleteOpen(false)}
            loading={deleting}
          />
        </>
      )}
    </SettingsLayout>
  );
}
