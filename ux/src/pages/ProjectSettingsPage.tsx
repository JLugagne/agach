import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Loader2, AlertTriangle } from 'lucide-react';
import { getProject, updateProject, deleteProject, getProjectInfo, listColumns, updateColumnWIPLimit, listProjectRoles, listProjectAgents } from '../lib/api';
import SettingsLayout from '../components/settings/SettingsLayout';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import AddAgentToProjectDialog from '../components/AddAgentToProjectDialog';
import RemoveAgentDialog from '../components/RemoveAgentDialog';
import { useWebSocket } from '../hooks/useWebSocket';
import type { ProjectResponse, ColumnResponse, RoleResponse, WSEvent } from '../lib/types';

function ProjectAgentsSection({ projectId }: { projectId: string }) {
  const [agents, setAgents] = useState<RoleResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<RoleResponse | null>(null);

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

  return (
    <div className="mb-12">
      <div className="flex items-center justify-between mb-4">
        <h2 className="font-heading text-lg text-[#F0F0F0]">Project Agents</h2>
        <button
          onClick={() => setAddDialogOpen(true)}
          className="px-3 py-1.5 bg-[#1A1A1A] border border-[#252525] text-sm text-[var(--text-muted)] rounded-md hover:text-[#F0F0F0] hover:border-[#333] transition-colors"
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
            <div key={agent.slug} className="flex items-center justify-between py-2 px-3 rounded-md bg-[#111] border border-[#1E1E1E]">
              <div className="flex items-center gap-2.5">
                <span
                  className="w-2.5 h-2.5 rounded-full shrink-0"
                  style={{ backgroundColor: agent.color || '#6B7280' }}
                />
                <span className="text-sm text-[#F0F0F0]">{agent.name}</span>
                <span className="text-xs text-[var(--text-dim)] font-mono">{agent.slug}</span>
              </div>
              <button
                onClick={() => setRemoveTarget(agent)}
                className="text-xs text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors px-2 py-0.5 rounded"
              >
                Remove
              </button>
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
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [projectInfo, setProjectInfo] = useState<unknown>(null);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [defaultRole, setDefaultRole] = useState<string>('');
  const [roles, setRoles] = useState<RoleResponse[]>([]);
  const [columns, setColumns] = useState<ColumnResponse[]>([]);
  const [wipLimits, setWipLimits] = useState<Record<string, number>>({});
  const [wipSaving, setWipSaving] = useState(false);
  const [wipSaved, setWipSaved] = useState(false);

  const fetchProject = useCallback(async () => {
    if (!projectId) return;
    try {
      const [p, info, cols] = await Promise.all([
        getProject(projectId),
        getProjectInfo(projectId).catch(() => null),
        listColumns(projectId).catch(() => [] as ColumnResponse[]),
      ]);
      setProject(p);
      setProjectInfo(info as unknown);
      setName(p.name);
      setDescription(p.description);
      setDefaultRole(p.default_role ?? '');
      listProjectRoles(projectId).then(setRoles).catch(() => {});
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
        default_role: defaultRole,
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
      <div className="min-h-screen bg-[#0F0F0F] flex items-center justify-center">
        <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="min-h-screen bg-[#0F0F0F] flex items-center justify-center">
        <p className="text-[var(--text-dim)]">Project not found</p>
      </div>
    );
  }

  const definitionDrawer = projectInfo ? (
    <div className="flex flex-col h-full">
      <div className="px-6 py-4 border-b border-[#1E1E1E]">
        <h3 className="font-heading text-sm text-[#F0F0F0]">Project Definition</h3>
        <p className="text-xs text-[var(--text-dim)] mt-0.5">Read-only view of project data</p>
      </div>
      <div className="flex-1 overflow-auto p-6">
        <pre className="font-mono text-xs text-[var(--text-muted)] whitespace-pre-wrap leading-relaxed">
          {JSON.stringify(projectInfo, null, 2)}
        </pre>
      </div>
    </div>
  ) : undefined;

  return (
    <SettingsLayout projectName={project.name} rightDrawer={definitionDrawer}>
      <h1 className="font-heading text-2xl text-[#F0F0F0] mb-8">Project Definition</h1>

      {/* Name */}
      <div className="mb-5">
        <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
        />
      </div>

      {/* Description */}
      <div className="mb-6">
        <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Describe this project..."
          rows={5}
          className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 resize-y"
        />
      </div>

      {/* Default Role */}
      <div className="mb-6">
        <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Default Role</label>
        <select
          value={defaultRole}
          onChange={(e) => setDefaultRole(e.target.value)}
          className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] focus:outline-none focus:border-[#00C896]/50"
        >
          <option value="">None (Unassigned)</option>
          {roles.map((r) => (
            <option key={r.id} value={r.slug}>{r.icon ? r.icon + ' ' : ''}{r.name}</option>
          ))}
        </select>
      </div>

      {/* Save button */}
      <div className="mb-12">
        <button
          onClick={handleSave}
          disabled={!name.trim() || saving}
          className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving...' : saved ? 'Saved' : 'Save Changes'}
        </button>
      </div>

      {/* WIP Limits */}
      {columns.length > 0 && (
        <div className="mb-12">
          <h2 className="font-heading text-lg text-[#F0F0F0] mb-4">Column WIP Limits</h2>
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
                  className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] focus:outline-none focus:border-[#00C896]/50"
                />
              </div>
            ))}
          </div>
          <button
            onClick={handleSaveWipLimits}
            disabled={wipSaving}
            className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
          >
            {wipSaving ? 'Saving...' : wipSaved ? 'Saved' : 'Save WIP Limits'}
          </button>
        </div>
      )}

      {/* Project Agents */}
      {projectId && <ProjectAgentsSection projectId={projectId} />}

      {/* Danger Zone */}
      <div className="border border-[#FF3B30]/30 rounded-lg p-5">
        <div className="flex items-start gap-3">
          <AlertTriangle size={18} className="text-[#FF3B30] mt-0.5 shrink-0" />
          <div className="flex-1">
            <h3 className="text-sm text-[#F0F0F0] font-medium mb-1">Delete this project</h3>
            <p className="text-xs text-[var(--text-muted)] mb-4">
              Once you delete a project, there is no going back. All tasks, comments, and data will
              be permanently removed.
            </p>
            <button
              onClick={() => setDeleteOpen(true)}
              className="px-4 py-2 bg-[#FF3B30] text-white text-sm font-medium rounded-md hover:bg-[#FF3B30]/80 transition-colors"
            >
              Delete Project
            </button>
          </div>
        </div>
      </div>

      <DeleteConfirmModal
        open={deleteOpen}
        title="Delete Project?"
        description={`This will permanently delete "${project.name}" and all its tasks, comments, and data. This action cannot be undone.`}
        confirmLabel="Delete Project"
        onConfirm={handleDelete}
        onCancel={() => setDeleteOpen(false)}
        loading={deleting}
      />
    </SettingsLayout>
  );
}
