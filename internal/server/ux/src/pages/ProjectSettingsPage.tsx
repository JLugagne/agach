import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { Loader2, AlertTriangle, Star, Users, User, X, Plus, Search, ChevronDown, UserPlus, MoreHorizontal } from 'lucide-react';
import {
  getProject, updateProject, deleteProject, listProjectAgents, listSpecializedAgents,
  listTeams, listUsers,
  listProjectUserAccess, grantUserAccess, revokeUserAccess, updateUserAccessRole,
  listProjectTeamAccess, grantTeamAccess, revokeTeamAccess,
} from '../lib/api';
import SettingsLayout from '../components/settings/SettingsLayout';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import AddAgentToProjectDialog from '../components/AddAgentToProjectDialog';
import RemoveAgentDialog from '../components/RemoveAgentDialog';
import { useWebSocket } from '../hooks/useWebSocket';
import { useAuth } from '../components/AuthContext';
import type { ProjectResponse, AgentResponse, SpecializedAgentResponse, WSEvent, TeamResponse, UserResponse, ProjectUserAccessResponse, ProjectTeamAccessResponse } from '../lib/types';

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

// ---------- Avatar helper ----------

const AVATAR_COLORS = ['#8B5CF6', '#3B82F6', '#F59E0B', '#10B981', '#EF4444', '#EC4899', '#06B6D4', '#1E6B4F'];

function avatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash);
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

function UserAvatar({ name, size = 36 }: { name: string; size?: number }) {
  const initial = (name || '?')[0].toUpperCase();
  return (
    <div
      className="rounded-full flex items-center justify-center shrink-0"
      style={{ width: size, height: size, backgroundColor: avatarColor(name) }}
    >
      <span className="text-white font-semibold" style={{ fontSize: size * 0.39 }}>{initial}</span>
    </div>
  );
}

// ---------- Role badge helper ----------

const ROLE_STYLES: Record<string, { bg: string; text: string }> = {
  admin: { bg: 'rgba(124,58,237,0.13)', text: '#8B5CF6' },
  member: { bg: 'rgba(59,130,246,0.13)', text: '#3B82F6' },
  viewer: { bg: 'rgba(16,185,129,0.13)', text: '#10B981' },
};

function RoleBadge({ role }: { role: string }) {
  const style = ROLE_STYLES[role] ?? ROLE_STYLES.member;
  return (
    <span
      className="inline-flex items-center justify-center rounded-md px-2.5 py-1 text-xs font-medium capitalize w-[120px] text-center"
      style={{ backgroundColor: style.bg, color: style.text }}
    >
      {role}
    </span>
  );
}

// ---------- Autocomplete input ----------

function UserAutocomplete({
  users,
  value,
  onChange,
  onSelect,
}: {
  users: UserResponse[];
  value: string;
  onChange: (v: string) => void;
  onSelect: (user: UserResponse) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = React.useRef<HTMLDivElement>(null);

  const filtered = value.trim()
    ? users.filter(u => {
        const q = value.toLowerCase();
        return (u.display_name?.toLowerCase().includes(q) || u.email.toLowerCase().includes(q));
      })
    : [];

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  return (
    <div ref={ref} className="relative flex-1 min-w-0">
      <div className="flex items-center gap-2 bg-[var(--bg-primary)] border border-[var(--border-primary)] focus-within:border-[var(--primary)] rounded-lg px-3.5 min-h-[46px]">
        <Search size={16} className="text-[var(--text-dim)] shrink-0" />
        <input
          type="text"
          value={value}
          onChange={e => { onChange(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          placeholder="Search users by name or email..."
          data-qa="user-search-input"
          className="flex-1 bg-transparent text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] outline-none"
        />
      </div>
      {open && filtered.length > 0 && (
        <div className="absolute z-50 left-0 right-0 mt-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg shadow-lg overflow-hidden p-1">
          {filtered.slice(0, 6).map(user => (
            <button
              key={user.id}
              onClick={() => { onSelect(user); setOpen(false); onChange(''); }}
              data-qa="autocomplete-suggestion"
              className="flex items-center gap-3 w-full px-3 py-2.5 rounded-md text-left hover:bg-[var(--bg-tertiary)] transition-colors"
            >
              <UserAvatar name={user.display_name || user.email} size={32} />
              <div className="flex flex-col gap-px min-w-0">
                <span className="text-[13px] font-medium text-[var(--text-primary)] truncate">{user.display_name || user.email}</span>
                <span className="text-xs text-[var(--text-dim)] truncate">{user.email}</span>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ---------- Team Autocomplete ----------

function TeamAutocomplete({
  teams,
  allUsers: users,
  value,
  onChange,
  onSelect,
}: {
  teams: TeamResponse[];
  allUsers: UserResponse[];
  value: string;
  onChange: (v: string) => void;
  onSelect: (team: TeamResponse) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = React.useRef<HTMLDivElement>(null);

  const filtered = value.trim()
    ? teams.filter(t => {
        const q = value.toLowerCase();
        return (t.name.toLowerCase().includes(q) || t.slug.toLowerCase().includes(q));
      })
    : [];

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  return (
    <div ref={ref} className="relative flex-1 min-w-0">
      <div className="flex items-center gap-2 bg-[var(--bg-primary)] border border-[var(--border-primary)] focus-within:border-[var(--primary)] rounded-lg px-3.5 py-2.5">
        <Search size={16} className="text-[var(--text-dim)] shrink-0" />
        <input
          type="text"
          value={value}
          onChange={e => { onChange(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          placeholder="Search teams by name..."
          data-qa="team-search-input"
          className="flex-1 bg-transparent text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] outline-none"
        />
      </div>
      {open && filtered.length > 0 && (
        <div className="absolute z-50 left-0 right-0 mt-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg shadow-lg overflow-hidden p-1">
          {filtered.slice(0, 6).map(team => {
            const memberCount = users.filter(u => u.team_ids.includes(team.id)).length;
            return (
              <button
                key={team.id}
                onClick={() => { onSelect(team); setOpen(false); onChange(''); }}
                data-qa="autocomplete-suggestion"
                className="flex items-center gap-3 w-full px-3 py-2.5 rounded-md text-left hover:bg-[var(--bg-tertiary)] transition-colors"
              >
                <div className="w-8 h-8 rounded-full bg-[var(--primary)] flex items-center justify-center shrink-0">
                  <Users size={14} className="text-white" />
                </div>
                <div className="flex flex-col gap-px min-w-0">
                  <span className="text-[13px] font-medium text-[var(--text-primary)] truncate">{team.name}</span>
                  <span className="text-xs text-[var(--text-dim)] truncate">{team.slug} · {memberCount} member{memberCount !== 1 ? 's' : ''}</span>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ---------- Role Dropdown ----------

function RoleDropdown({ value, onChange }: { value: 'admin' | 'member'; onChange: (v: 'admin' | 'member') => void }) {
  const [open, setOpen] = useState(false);
  const ref = React.useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const options: { value: 'admin' | 'member'; label: string }[] = [
    { value: 'member', label: 'Member' },
    { value: 'admin', label: 'Admin' },
  ];

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(o => !o)}
        data-qa="role-select"
        className="flex items-center gap-2 bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg px-3.5 min-h-[46px] hover:border-[var(--border-secondary)] transition-colors"
      >
        <span className="text-sm text-[var(--text-primary)] capitalize">{value}</span>
        <ChevronDown size={14} className="text-[var(--text-dim)]" />
      </button>
      {open && (
        <div className="absolute z-50 right-0 mt-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg shadow-lg overflow-hidden p-1 min-w-[120px]">
          {options.map(opt => (
            <button
              key={opt.value}
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className={`flex items-center w-full px-3 py-2 rounded-md text-sm transition-colors ${
                value === opt.value
                  ? 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] font-medium'
                  : 'text-[var(--text-muted)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ---------- Members Section ----------

function MembersSection({ projectId }: { projectId: string }) {
  const [subTab, setSubTab] = useState<'users' | 'teams'>('users');
  const [allUsers, setAllUsers] = useState<UserResponse[]>([]);
  const [allTeams, setAllTeams] = useState<TeamResponse[]>([]);
  const [userAccess, setUserAccess] = useState<ProjectUserAccessResponse[]>([]);
  const [teamAccess, setTeamAccess] = useState<ProjectTeamAccessResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRole, setSelectedRole] = useState<'admin' | 'member'>('member');
  const [selectedUser, setSelectedUser] = useState<UserResponse | null>(null);
  const [teamSearchQuery, setTeamSearchQuery] = useState('');

  const fetchData = useCallback(async () => {
    try {
      const [users, teams, ua, ta] = await Promise.all([
        listUsers(), listTeams(),
        listProjectUserAccess(projectId), listProjectTeamAccess(projectId),
      ]);
      setAllUsers(users ?? []);
      setAllTeams(teams ?? []);
      setUserAccess(ua ?? []);
      setTeamAccess(ta ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useWebSocket(useCallback((event: WSEvent) => {
    if (event.type === 'project_access_updated') fetchData();
  }, [fetchData]));

  const handleGrantUser = async (userId: string, role: 'admin' | 'member') => {
    try {
      await grantUserAccess(projectId, userId, role);
      setSelectedUser(null);
      setSearchQuery('');
      fetchData();
    } catch { /* ignore */ }
  };

  const handleRevokeUser = async (userId: string) => {
    try {
      await revokeUserAccess(projectId, userId);
      fetchData();
    } catch { /* ignore */ }
  };

  const handleUpdateUserRole = async (userId: string, role: 'admin' | 'member') => {
    try {
      await updateUserAccessRole(projectId, userId, role);
      fetchData();
    } catch { /* ignore */ }
  };

  const handleGrantTeam = async (teamId: string) => {
    try {
      await grantTeamAccess(projectId, teamId);
      setTeamSearchQuery('');
      fetchData();
    } catch { /* ignore */ }
  };

  const handleRevokeTeam = async (teamId: string) => {
    try {
      await revokeTeamAccess(projectId, teamId);
      fetchData();
    } catch { /* ignore */ }
  };

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-[var(--text-dim)] text-sm py-8">
        <Loader2 className="animate-spin" size={14} />
        <span>Loading access...</span>
      </div>
    );
  }

  const grantedUserIDs = new Set(userAccess.map(a => a.user_id));
  const grantedTeamIDs = new Set(teamAccess.map(a => a.team_id));
  const availableUsers = allUsers.filter(u => !grantedUserIDs.has(u.id));
  const availableTeams = allTeams.filter(t => !grantedTeamIDs.has(t.id));

  return (
    <div className="flex flex-col gap-6">
      {/* Sub-tabs */}
      <div className="flex items-center border-b border-[var(--border-primary)]">
        <button
          onClick={() => setSubTab('users')}
          className={`flex items-center gap-2 px-6 py-3 text-sm transition-colors -mb-px border-b-2 ${
            subTab === 'users'
              ? 'border-[var(--primary)] text-[var(--text-primary)] font-medium'
              : 'border-transparent text-[var(--text-dim)] hover:text-[var(--text-primary)]'
          }`}
        >
          <User size={16} />
          Users
        </button>
        <button
          onClick={() => setSubTab('teams')}
          className={`flex items-center gap-2 px-6 py-3 text-sm transition-colors -mb-px border-b-2 ${
            subTab === 'teams'
              ? 'border-[var(--primary)] text-[var(--text-primary)] font-medium'
              : 'border-transparent text-[var(--text-dim)] hover:text-[var(--text-primary)]'
          }`}
        >
          <Users size={16} />
          Teams
        </button>
      </div>

      {subTab === 'users' ? (
        <>
          {/* Add User Card */}
          <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-xl p-6 flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <UserPlus size={20} className="text-[var(--primary)]" />
              <h2 className="text-lg font-semibold text-[var(--text-primary)]">Add User</h2>
            </div>
            <p className="text-[13px] text-[var(--text-dim)]">Grant project access to an existing user.</p>
            <div className="flex items-center gap-3">
              {selectedUser ? (
                <div className="flex-1 flex items-center gap-3 bg-[var(--bg-primary)] border border-[var(--primary)] rounded-lg px-3.5 min-h-[46px]">
                  <UserAvatar name={selectedUser.display_name || selectedUser.email} size={28} />
                  <span className="text-sm text-[var(--text-primary)] font-medium truncate">{selectedUser.display_name || selectedUser.email}</span>
                  <span className="text-xs text-[var(--text-dim)] truncate">{selectedUser.email}</span>
                  <button onClick={() => { setSelectedUser(null); setSearchQuery(''); }} className="ml-auto text-[var(--text-dim)] hover:text-[var(--text-primary)]">
                    <X size={14} />
                  </button>
                </div>
              ) : (
                <UserAutocomplete
                  users={availableUsers}
                  value={searchQuery}
                  onChange={setSearchQuery}
                  onSelect={setSelectedUser}
                />
              )}
              <RoleDropdown value={selectedRole} onChange={setSelectedRole} />
              <button
                onClick={() => selectedUser && handleGrantUser(selectedUser.id, selectedRole)}
                disabled={!selectedUser}
                data-qa="add-user-btn"
                className="flex items-center gap-2 px-5 py-2.5 bg-[var(--primary)] text-white text-[13px] font-medium rounded-lg hover:bg-[var(--primary-hover)] disabled:opacity-40 transition-colors"
              >
                <Plus size={14} />
                Add
              </button>
            </div>
          </div>

          {/* Users Table */}
          {userAccess.length === 0 ? (
            <p className="text-sm text-[var(--text-dim)] italic py-4">No users have been granted access yet.</p>
          ) : (
            <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-xl overflow-hidden">
              {/* Table Header */}
              <div className="flex items-center px-5 py-3 bg-[var(--bg-primary)]/50 border-b border-[var(--border-primary)]">
                <span className="w-[240px] text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase">User</span>
                <span className="flex-1 text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase">Email</span>
                <span className="w-[120px] text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase">Role</span>
                <span className="w-[80px] text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase text-right">Actions</span>
              </div>
              {/* Table Rows */}
              {userAccess.map((access, i) => {
                const user = allUsers.find(u => u.id === access.user_id);
                const displayName = user?.display_name || user?.email || access.user_id;
                return (
                  <div
                    key={access.id}
                    className={`flex items-center px-5 py-3.5 ${i < userAccess.length - 1 ? 'border-b border-[var(--border-primary)]' : ''}`}
                  >
                    <div className="w-[240px] flex items-center gap-3">
                      <UserAvatar name={displayName} />
                      <div className="flex flex-col gap-0.5 min-w-0">
                        <span className="text-sm font-medium text-[var(--text-primary)] truncate">{displayName}</span>
                        {user?.role === 'admin' && <span className="text-xs text-[var(--text-dim)]">Owner</span>}
                      </div>
                    </div>
                    <span className="flex-1 text-[13px] text-[var(--text-muted)] truncate">{user?.email || ''}</span>
                    <div className="w-[120px]">
                      <button
                        onClick={() => handleUpdateUserRole(access.user_id, access.role === 'admin' ? 'member' : 'admin')}
                        data-qa="toggle-user-role"
                        title={`Click to change role to ${access.role === 'admin' ? 'member' : 'admin'}`}
                      >
                        <RoleBadge role={access.role} />
                      </button>
                    </div>
                    <div className="w-[80px] flex justify-end">
                      <button
                        onClick={() => handleRevokeUser(access.user_id)}
                        data-qa="revoke-user-btn"
                        className="text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors p-1"
                        title="Revoke access"
                      >
                        <MoreHorizontal size={18} />
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      ) : (
        <>
          {/* Add Team Card */}
          <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-xl p-6 flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <Users size={20} className="text-[var(--primary)]" />
              <h2 className="text-lg font-semibold text-[var(--text-primary)]">Add Team</h2>
            </div>
            <p className="text-[13px] text-[var(--text-dim)]">Grant project access to an existing team.</p>
            <div className="flex items-center gap-3">
              <TeamAutocomplete
                teams={availableTeams}
                allUsers={allUsers}
                value={teamSearchQuery}
                onChange={setTeamSearchQuery}
                onSelect={team => handleGrantTeam(team.id)}
              />
            </div>
          </div>

          {/* Teams Table */}
          {teamAccess.length === 0 ? (
            <p className="text-sm text-[var(--text-dim)] italic py-4">No teams have been granted access yet.</p>
          ) : (
            <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-xl overflow-hidden">
              <div className="flex items-center px-5 py-3 bg-[var(--bg-primary)]/50 border-b border-[var(--border-primary)]">
                <span className="w-[240px] text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase">Team</span>
                <span className="flex-1 text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase">Slug</span>
                <span className="w-[120px] text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase">Members</span>
                <span className="w-[80px] text-[11px] font-semibold text-[var(--text-dim)] tracking-wider uppercase text-right">Actions</span>
              </div>
              {teamAccess.map((access, i) => {
                const team = allTeams.find(t => t.id === access.team_id);
                const memberCount = team ? allUsers.filter(u => u.team_ids.includes(team.id)).length : 0;
                return (
                  <div
                    key={access.id}
                    className={`flex items-center px-5 py-3.5 ${i < teamAccess.length - 1 ? 'border-b border-[var(--border-primary)]' : ''}`}
                  >
                    <div className="w-[240px] flex items-center gap-3">
                      <div className="w-9 h-9 rounded-full bg-[var(--primary)] flex items-center justify-center shrink-0">
                        <Users size={16} className="text-white" />
                      </div>
                      <span className="text-sm font-medium text-[var(--text-primary)] truncate">{team?.name || access.team_id}</span>
                    </div>
                    <span className="flex-1 text-[13px] text-[var(--text-muted)] font-mono truncate">{team?.slug || ''}</span>
                    <span className="w-[120px] text-[13px] text-[var(--text-dim)]">{memberCount} member{memberCount !== 1 ? 's' : ''}</span>
                    <div className="w-[80px] flex justify-end">
                      <button
                        onClick={() => handleRevokeTeam(access.team_id)}
                        data-qa="revoke-team-btn"
                        className="text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors p-1"
                        title="Revoke access"
                      >
                        <MoreHorizontal size={18} />
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default function ProjectSettingsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { user } = useAuth();
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
  const [isProjectAdmin, setIsProjectAdmin] = useState(false);

  const isAgachAdmin = user?.role === 'admin';

  const fetchProject = useCallback(async () => {
    if (!projectId) return;
    try {
      const p = await getProject(projectId);
      setProject(p);
      setName(p.name);
      setDescription(p.description);
      setGitUrl(p.git_url || '');
      setDefaultRole(p.default_role ?? '');

      try {
        const userAccess = await listProjectUserAccess(projectId);
        const myAccess = (userAccess ?? []).find(a => a.user_id === user?.id);
        setIsProjectAdmin(myAccess?.role === 'admin');
      } catch {
        // access table may not exist yet — fall back to agach admin check only
      }
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId, user?.id]);

  useEffect(() => {
    fetchProject();
  }, [fetchProject]);

  const canManageSharing = isAgachAdmin || isProjectAdmin;

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
    <SettingsLayout projectName={project.name} showSharing={canManageSharing}>
      {isMembersTab ? (
        canManageSharing ? (
          <MembersSection projectId={projectId!} />
        ) : (
          <div className="py-8">
            <p className="text-sm text-[var(--text-dim)]">You don't have permission to manage sharing for this project.</p>
          </div>
        )
      ) : isAgentsTab ? (
        <>
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
