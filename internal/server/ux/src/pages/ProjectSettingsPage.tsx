import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useLocation, Link } from 'react-router-dom';
import { Loader2, AlertTriangle, Star, Users, User, ExternalLink } from 'lucide-react';
import {
  getProject, updateProject, deleteProject, listProjectAgents, listSpecializedAgents,
  listTeams, listUsers,
} from '../lib/api';
import SettingsLayout from '../components/settings/SettingsLayout';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import AddAgentToProjectDialog from '../components/AddAgentToProjectDialog';
import RemoveAgentDialog from '../components/RemoveAgentDialog';
import { useWebSocket } from '../hooks/useWebSocket';
import type { ProjectResponse, AgentResponse, SpecializedAgentResponse, WSEvent, TeamResponse, UserResponse } from '../lib/types';

interface ProjectSpecializedAgent {
  slug: string;
  name: string;
  color: string;
  parentSlug: string;
}

function ProjectAgentsSection({ projectId, defaultRole, onDefaultRoleChange }: { projectId: string; defaultRole: string; onDefaultRoleChange: (slug: string) => void }) {
  const [parentAgents, setParentAgents] = useState<AgentResponse[]>([]);
  const [specAgents, setSpecAgents] = useState<ProjectSpecializedAgent[]>([]);
  const [loading, setLoading] = useState(true);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<AgentResponse | null>(null);

  const fetchAgents = useCallback(async () => {
    try {
      const parents = (await listProjectAgents(projectId)) ?? [];
      setParentAgents(parents);

      const specResults = await Promise.all(
        parents.map(async (agent) => {
          try {
            const specs = (await listSpecializedAgents(agent.slug)) ?? [];
            return specs.map((spec: SpecializedAgentResponse) => ({
              slug: spec.slug,
              name: spec.name,
              color: agent.color,
              parentSlug: agent.slug,
            }));
          } catch {
            return [];
          }
        })
      );
      setSpecAgents(specResults.flat());
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
      ) : specAgents.length === 0 ? (
        <p className="text-sm text-[var(--text-dim)] italic">No agents assigned yet.</p>
      ) : (
        <div className="space-y-1">
          {specAgents.map(agent => (
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
                  onClick={() => setRemoveTarget(parentAgents.find(p => p.slug === agent.parentSlug) ?? null)}
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
          assignedSlugs={new Set(specAgents.map(a => a.slug))}
          onClose={() => setAddDialogOpen(false)}
          onSuccess={fetchAgents}
        />
      )}

      {removeTarget && (
        <RemoveAgentDialog
          projectId={projectId}
          agent={removeTarget}
          projectAgents={parentAgents}
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

// ---------- Members Section ----------

function MembersSection() {
  const [teams, setTeams] = useState<TeamResponse[]>([]);
  const [users, setUsers] = useState<UserResponse[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const [t, u] = await Promise.all([listTeams(), listUsers()]);
      setTeams(t ?? []);
      setUsers(u ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-[var(--text-dim)] text-sm py-8">
        <Loader2 className="animate-spin" size={14} />
        <span>Loading members...</span>
      </div>
    );
  }

  return (
    <>
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-xl font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Members</h1>
        <Link
          to="/teams"
          data-qa="manage-teams-link"
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
        >
          Manage Teams
          <ExternalLink size={12} />
        </Link>
      </div>

      {/* Teams */}
      {teams.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 gap-3">
          <Users size={24} className="text-[var(--text-dim)]" />
          <p className="text-sm text-[var(--text-dim)]">No teams yet.</p>
          <Link
            to="/teams"
            className="text-sm text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
          >
            Create teams in admin settings
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {teams.map((team) => {
            const members = users.filter((u) => u.team_ids.includes(team.id));
            return (
              <div key={team.id} className="border border-[var(--border-primary)] rounded-lg overflow-hidden">
                <div className="flex items-center justify-between px-4 py-3 bg-[var(--bg-primary)]">
                  <div className="flex items-center gap-2.5">
                    <Users size={14} className="text-[var(--primary)]" />
                    <span className="text-sm font-medium text-[var(--text-primary)]">{team.name}</span>
                    <span className="text-xs text-[var(--text-dim)] font-mono">{team.slug}</span>
                    <span className="text-[10px] text-[var(--text-dim)]">
                      {members.length} member{members.length !== 1 ? 's' : ''}
                    </span>
                  </div>
                </div>
                {members.length > 0 && (
                  <div className="divide-y divide-[var(--border-primary)]">
                    {members.map((u) => (
                      <div key={u.id} className="flex items-center gap-2.5 px-4 py-2.5">
                        <User size={13} className="text-[var(--text-dim)]" />
                        <span className="text-sm text-[var(--text-primary)]">{u.display_name || u.email}</span>
                        <span className="text-xs text-[var(--text-dim)]">{u.email}</span>
                        <span
                          className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono ${
                            u.role === 'admin'
                              ? 'bg-[var(--primary)]/15 text-[var(--primary)]'
                              : 'bg-[var(--bg-tertiary)] text-[var(--text-dim)]'
                          }`}
                        >
                          {u.role}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Unassigned users */}
      {users.filter((u) => u.team_ids.length === 0).length > 0 && (
        <section className="mt-8">
          <h2 className="text-sm font-medium text-[var(--text-muted)] mb-3">Unassigned Users</h2>
          <div className="border border-[var(--border-primary)] rounded-lg overflow-hidden divide-y divide-[var(--border-primary)]">
            {users.filter((u) => u.team_ids.length === 0).map((u) => (
              <div key={u.id} className="flex items-center gap-2.5 px-4 py-2.5">
                <User size={13} className="text-[var(--text-dim)]" />
                <span className="text-sm text-[var(--text-primary)]">{u.display_name || u.email}</span>
                <span className="text-xs text-[var(--text-dim)]">{u.email}</span>
                <span
                  className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono ${
                    u.role === 'admin'
                      ? 'bg-[var(--primary)]/15 text-[var(--primary)]'
                      : 'bg-[var(--bg-tertiary)] text-[var(--text-dim)]'
                  }`}
                >
                  {u.role}
                </span>
              </div>
            ))}
          </div>
        </section>
      )}
    </>
  );
}

export default function ProjectSettingsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const isAgentsTab = location.pathname.endsWith('/agents');
  const isMembersTab = location.pathname.endsWith('/members');
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

  const fetchProject = useCallback(async () => {
    if (!projectId) return;
    try {
      const p = await getProject(projectId);
      setProject(p);
      setName(p.name);
      setDescription(p.description);
      setGitUrl(p.git_url || '');
      setDefaultRole(p.default_role ?? '');
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
      {isMembersTab ? (
        <MembersSection />
      ) : isAgentsTab ? (
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
