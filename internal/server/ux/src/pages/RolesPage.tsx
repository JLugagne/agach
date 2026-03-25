import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Plus,
  X,
  Loader2,
  Trash2,
  Pencil,
  Check,
  Star,
  Copy,
  Info,
  ChevronDown,
  ChevronUp,
  ChevronRight,
  Bot,
} from 'lucide-react';
import {
  listAgents, createAgent, updateAgent, deleteAgent,
  listProjectAgents, createProjectAgent, updateProjectAgent, deleteProjectAgent,
  getProject, updateProject,
  listSpecializedAgents,
} from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { AgentResponse, CreateAgentRequest, UpdateAgentRequest, SpecializedAgentResponse } from '../lib/types';
import MarkdownContent from '../components/ui/MarkdownContent';
import CloneAgentDialog from '../components/CloneAgentDialog';
import AgentSkillsPanel from '../components/AgentSkillsPanel';
import EditSpecializedAgentDialog from '../components/EditSpecializedAgentDialog';

const PRESET_COLORS = [
  '#7C3AED',
  '#F09060',
  '#6C63FF',
  '#FF6B9D',
  '#FFD060',
  '#00B4D8',
  '#FF3B30',
];

const PRESET_ICONS = [
  '\u{1F680}', '\u{1F4BB}', '\u{1F527}', '\u{2699}\uFE0F', '\u{1F50D}', '\u{1F3AF}',
  '\u{1F4E6}', '\u{1F9EA}', '\u{1F41B}', '\u{1F6E0}\uFE0F', '\u{1F4CA}', '\u{1F512}',
  '\u{1F310}', '\u{2B50}', '\u{26A1}', '\u{1F4DD}', '\u{1F916}', '\u{1F9D1}\u200D\u{1F4BB}',
  '\u{1F3D7}\uFE0F', '\u{1F4D0}', '\u{1F4A1}', '\u{1F4AC}', '\u{1F50C}', '\u{1F4C1}',
  '\u{1F3A8}', '\u{1F9F9}', '\u{1F50E}', '\u{1F4E1}', '\u{1F9E9}', '\u{2B22}',
];

export default function RolesPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [roles, setRoles] = useState<AgentResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<AgentResponse | null>(null);
  const [defaultRole, setDefaultRole] = useState<string>('');
  const [cloningRole, setCloningRole] = useState<AgentResponse | null>(null);
  const [drawerRole, setDrawerRole] = useState<AgentResponse | null>(null);

  const fetchRoles = useCallback(async () => {
    try {
      const data = projectId ? await listProjectAgents(projectId) : await listAgents();
      setRoles(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  const fetchDefaultRole = useCallback(async () => {
    if (!projectId) return;
    try {
      const project = await getProject(projectId);
      setDefaultRole(project.default_role ?? '');
    } catch {
      // ignore
    }
  }, [projectId]);

  const handleSetDefault = async (slug: string) => {
    if (!projectId) return;
    const newDefault = defaultRole === slug ? '' : slug;
    setDefaultRole(newDefault);
    try {
      await updateProject(projectId, { default_role: newDefault });
    } catch {
      setDefaultRole(defaultRole);
    }
  };

  useEffect(() => {
    fetchRoles();
    fetchDefaultRole();
  }, [fetchRoles, fetchDefaultRole]);

  useWebSocket(
    useCallback(
      (event) => {
        if (
          event.type === 'agent_created' ||
          event.type === 'agent_updated' ||
          event.type === 'agent_deleted' ||
          event.type === 'agent_cloned' ||
          event.type === 'specialized_agent_created' ||
          event.type === 'specialized_agent_updated' ||
          event.type === 'specialized_agent_deleted'
        ) {
          fetchRoles();
        }
      },
      [fetchRoles],
    ),
  );

  const openCreate = () => {
    setEditingRole(null);
    setModalOpen(true);
  };

  const handleCardClick = (role: AgentResponse) => {
    if (projectId) {
      // In project mode, open the edit modal directly
      setEditingRole(role);
      setModalOpen(true);
    } else {
      // In global mode, open the drawer
      setDrawerRole(role);
    }
  };

  const openEditFromDrawer = () => {
    const role = drawerRole;
    setDrawerRole(null);
    setEditingRole(role);
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditingRole(null);
  };

  const handleSaved = () => {
    closeModal();
    fetchRoles();
  };

  const handleDeleted = () => {
    closeModal();
    fetchRoles();
  };

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-5xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            Agents
          </h1>
          <button
            onClick={openCreate}
            data-qa="new-agent-btn"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Agent
          </button>
        </div>
        <p className="text-sm text-[var(--text-muted)] mb-10" style={{ fontFamily: 'Inter, sans-serif' }}>
          {roles.length} agent{roles.length !== 1 ? 's' : ''} defined
        </p>

        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
          </div>
        ) : roles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-5">
            <div className="w-20 h-20 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
              <Bot size={36} className="text-[var(--text-muted)]" />
            </div>
            <p className="text-lg font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              No agents yet.
            </p>
            <p className="text-sm text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Get started by creating your first agent
            </p>
            <button
              onClick={openCreate}
              data-qa="agents-create-first-agent-btn"
              className="flex items-center gap-2 px-6 py-3 rounded-lg text-sm font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              <Plus size={16} />
              Create your first agent
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {roles.map((role) => (
              <RoleCard
                key={role.id}
                role={role}
                isDefault={defaultRole === role.slug}
                onSetDefault={projectId ? () => handleSetDefault(role.slug) : undefined}
                onClick={() => handleCardClick(role)}
                onClone={() => setCloningRole(role)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Agent Drawer */}
      {drawerRole && (
        <AgentDrawer
          role={drawerRole}
          onClose={() => setDrawerRole(null)}
          onEdit={openEditFromDrawer}
        />
      )}

      {/* Role Modal */}
      {modalOpen && (
        <RoleModal
          role={editingRole}
          projectId={projectId}
          onClose={closeModal}
          onSaved={handleSaved}
          onDeleted={handleDeleted}
        />
      )}

      {/* Clone Dialog */}
      {cloningRole && (
        <CloneAgentDialog
          sourceRole={cloningRole}
          onClose={() => setCloningRole(null)}
          onSuccess={(cloned) => {
            setRoles(prev => [...prev, cloned]);
            setCloningRole(null);
          }}
        />
      )}
    </div>
  );
}

function RoleCard({
  role,
  isDefault,
  onSetDefault,
  onClick,
  onClone,
}: {
  role: AgentResponse;
  isDefault: boolean;
  onSetDefault?: () => void;
  onClick: () => void;
  onClone: () => void;
}) {
  return (
    <div
      data-qa="agent-card"
      className={`rounded-lg bg-[var(--bg-primary)] border p-5 text-left transition-colors w-full ${
        isDefault ? 'border-[var(--primary)]/40' : 'border-[var(--border-primary)] hover:border-[var(--border-secondary)]'
      }`}
    >
      <div className="flex items-start gap-3 mb-3">
        <button onClick={onClick} data-qa="agent-card-icon-btn" className="text-xl cursor-pointer">{role.icon || '\u2B22'}</button>
        <div className="flex-1 min-w-0 cursor-pointer" onClick={onClick}>
          <div className="flex items-center gap-2">
            <h3 className="font-heading text-[15px] text-[var(--text-primary)] truncate">{role.name}</h3>
            {isDefault && (
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono bg-[var(--primary)]/10 text-[var(--primary)] border border-[var(--primary)]/20 shrink-0">
                <Star size={9} />
                default
              </span>
            )}
          </div>
          <p className="font-mono text-[11px] text-[var(--text-dim)]">{role.slug}</p>
        </div>
        <div className="flex items-center gap-2 shrink-0 mt-0.5">
          {onSetDefault && (
            <button
              onClick={(e) => { e.stopPropagation(); onSetDefault(); }}
              data-qa="agent-card-set-default-btn"
              title={isDefault ? 'Unset default' : 'Set as default'}
              className={`transition-colors ${
                isDefault
                  ? 'text-[var(--primary)] hover:text-[var(--primary)]/60'
                  : 'text-[var(--text-dim)] hover:text-[var(--primary)]'
              }`}
            >
              <Star size={14} fill={isDefault ? 'currentColor' : 'none'} />
            </button>
          )}
          <button
            onClick={(e) => { e.stopPropagation(); onClone(); }}
            data-qa="agent-card-clone-btn"
            title="Clone agent"
            className="p-1.5 rounded text-muted-foreground hover:text-blue-400 hover:bg-blue-400/10 transition-colors"
          >
            <Copy size={14} />
          </button>
          <div
            className="w-3 h-3 rounded-full"
            style={{ backgroundColor: role.color || '#6B7280' }}
          />
        </div>
      </div>

      <div onClick={onClick} className="cursor-pointer">
        {role.description && (
          <p className="text-xs text-[var(--text-muted)] mb-3 line-clamp-2">{role.description}</p>
        )}

        <div className="flex items-center gap-2 flex-wrap">
          {role.prompt_template && (
            <span className="px-2 py-0.5 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded text-[10px] font-mono text-[var(--text-muted)]">
              has template
            </span>
          )}
          {(role.specialized_count ?? 0) > 0 && (
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-[var(--primary)]/10 text-[var(--primary)]">
              {role.specialized_count} specialized
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

function AgentDrawer({
  role,
  onClose,
  onEdit,
}: {
  role: AgentResponse;
  onClose: () => void;
  onEdit: () => void;
}) {
  const navigate = useNavigate();
  const [specAgents, setSpecAgents] = useState<SpecializedAgentResponse[]>([]);
  const [loadingSpec, setLoadingSpec] = useState(true);
  const [hintExpanded, setHintExpanded] = useState(false);
  const [createSpecOpen, setCreateSpecOpen] = useState(false);

  const fetchSpecialized = useCallback(async () => {
    try {
      const data = await listSpecializedAgents(role.slug);
      setSpecAgents(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoadingSpec(false);
    }
  }, [role.slug]);

  useEffect(() => { fetchSpecialized(); }, [fetchSpecialized]);

  useWebSocket(useCallback((event) => {
    if (
      event.type === 'specialized_agent_created' ||
      event.type === 'specialized_agent_updated' ||
      event.type === 'specialized_agent_deleted'
    ) {
      fetchSpecialized();
    }
  }, [fetchSpecialized]));

  const promptHint = role.prompt_hint || '';
  const hintLines = promptHint.split('\n');
  const HINT_COLLAPSED_LINES = 6;
  const showToggle = hintLines.length > HINT_COLLAPSED_LINES;
  const displayedHint = hintExpanded ? promptHint : hintLines.slice(0, HINT_COLLAPSED_LINES).join('\n');

  return (
    <>
      <div className="fixed inset-0 z-50 flex">
        <div className="flex-1 bg-black/50" onClick={onClose} />
        <div className="w-[480px] h-full bg-[var(--bg-primary)] border-l border-[var(--border-primary)] flex flex-col animate-[slide-in-right_0.2s_ease-out]">
          {/* Header */}
          <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-primary)]">
            <div className="flex items-center gap-3">
              {role.icon && <span className="text-xl">{role.icon}</span>}
              <h2 className="text-lg font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                {role.name}
              </h2>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={onEdit}
                data-qa="agent-drawer-edit-btn"
                className="p-1.5 rounded text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
                title="Edit agent"
              >
                <Pencil size={16} />
              </button>
              <button
                onClick={onClose}
                data-qa="agent-drawer-close-btn"
                className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
              >
                <X size={18} />
              </button>
            </div>
          </div>

          {/* Body */}
          <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
            {/* Description */}
            <div>
              <label className="text-xs font-mono text-[var(--text-dim)] mb-2 block">Description</label>
              {role.description ? (
                <p className="text-sm text-[var(--text-muted)] leading-relaxed">{role.description}</p>
              ) : (
                <p className="text-sm text-[var(--text-dim)] italic">No description</p>
              )}
            </div>

            {/* Prompt Hint */}
            <div>
              <div className="flex items-center gap-2 mb-2">
                <label className="text-xs font-mono text-[var(--text-dim)]">Prompt Hint</label>
                {promptHint && (
                  <span className="text-[10px] font-mono text-[var(--text-dim)]">
                    {hintLines.length} line{hintLines.length !== 1 ? 's' : ''}
                  </span>
                )}
              </div>
              {promptHint ? (
                <div>
                  <div className="rounded-md bg-[#0D0D0D] border border-[#1A1A1A] px-3 py-2.5">
                    <pre className="text-xs font-mono text-[var(--text-muted)] whitespace-pre-wrap break-words leading-relaxed">
                      {displayedHint}
                    </pre>
                  </div>
                  {showToggle && (
                    <button
                      onClick={() => setHintExpanded(!hintExpanded)}
                      className="flex items-center gap-1 mt-2 text-xs text-[var(--primary)] hover:text-[var(--primary)]/80 transition-colors"
                    >
                      {hintExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                      {hintExpanded ? 'Show less' : 'Show more'}
                    </button>
                  )}
                </div>
              ) : (
                <p className="text-sm text-[var(--text-dim)] italic">No prompt hint</p>
              )}
            </div>

            {/* Specialized Agents */}
            <div>
              <div className="flex items-center gap-2 mb-3">
                <label className="text-xs font-mono text-[var(--text-dim)]">Specialized Agents</label>
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-[var(--primary)]/10 text-[var(--primary)]">
                  {specAgents.length}
                </span>
              </div>

              {loadingSpec ? (
                <div className="flex items-center gap-2 py-3">
                  <Loader2 size={14} className="animate-spin text-[var(--text-dim)]" />
                  <span className="text-sm text-[var(--text-dim)]">Loading...</span>
                </div>
              ) : (
                <>
                  {specAgents.length > 0 && (
                    <div className="space-y-1 mb-3">
                      {specAgents.map(spec => (
                        <button
                          key={spec.id}
                          onClick={() => navigate(`/agents/${role.slug}/specialized/${spec.slug}`)}
                          data-qa="agent-drawer-specialized-item"
                          className="w-full flex items-center justify-between px-3 py-2.5 rounded-md hover:bg-[var(--bg-secondary)] transition-colors text-left"
                        >
                          <div>
                            <p className="text-sm text-[var(--text-primary)]">{spec.name}</p>
                            <p className="text-[11px] text-[var(--text-dim)]">{spec.skill_count} skill{spec.skill_count !== 1 ? 's' : ''}</p>
                          </div>
                          <ChevronRight size={16} className="text-[var(--text-dim)]" />
                        </button>
                      ))}
                    </div>
                  )}

                  <button
                    onClick={() => setCreateSpecOpen(true)}
                    data-qa="agent-drawer-add-specialized-btn"
                    className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium border border-[var(--primary)] text-[var(--primary)] hover:bg-[var(--primary)]/10 transition-colors w-full justify-center"
                    style={{ fontFamily: 'Inter, sans-serif' }}
                  >
                    <Plus size={14} />
                    Add Specialized Agent
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Create Specialized Agent Dialog */}
      {createSpecOpen && (
        <EditSpecializedAgentDialog
          parentSlug={role.slug}
          onClose={() => setCreateSpecOpen(false)}
          onSaved={() => {
            setCreateSpecOpen(false);
            fetchSpecialized();
          }}
        />
      )}
    </>
  );
}

interface InlineEditFieldProps {
  label: string;
  value: string;
  placeholder?: string;
  monoFont?: boolean;
  onSave: (newValue: string) => Promise<void>;
}

function InlineEditField({ label, value, placeholder, monoFont, onSave }: InlineEditFieldProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const [saving, setSaving] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Sync draft when value changes from outside (e.g. optimistic update)
  useEffect(() => {
    if (!editing) setDraft(value);
  }, [value, editing]);

  // Auto-resize textarea
  useEffect(() => {
    if (editing && textareaRef.current) {
      const el = textareaRef.current;
      el.style.height = 'auto';
      el.style.height = `${el.scrollHeight}px`;
      el.focus();
      el.setSelectionRange(el.value.length, el.value.length);
    }
  }, [editing]);

  const handleEdit = () => {
    setDraft(value);
    setEditing(true);
  };

  const handleCancel = () => {
    setDraft(value);
    setEditing(false);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(draft);
      setEditing(false);
    } catch {
      // keep editing open on error
    } finally {
      setSaving(false);
    }
  };

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setDraft(e.target.value);
    const el = e.target;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <label className="text-xs font-mono text-[var(--text-dim)]">{label}</label>
        {!editing && (
          <button
            onClick={handleEdit}
            data-qa="inline-edit-field-edit-btn"
            className="flex items-center gap-1 text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
            title={`Edit ${label}`}
          >
            <Pencil size={12} />
          </button>
        )}
        {editing && (
          <div className="flex items-center gap-2">
            <button
              onClick={handleCancel}
              disabled={saving}
              data-qa="inline-edit-field-cancel-btn"
              className="flex items-center gap-1 text-xs text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors disabled:opacity-50"
            >
              <X size={12} />
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              data-qa="inline-edit-field-save-btn"
              className="flex items-center gap-1 text-xs text-[var(--primary)] hover:text-[var(--primary)]/80 transition-colors disabled:opacity-50"
            >
              <Check size={12} />
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        )}
      </div>

      {editing ? (
        <textarea
          ref={textareaRef}
          value={draft}
          onChange={handleTextareaChange}
          placeholder={placeholder}
          rows={6}
          data-qa="inline-edit-field-textarea"
          className={`w-full bg-[var(--bg-secondary)] border border-[var(--primary)]/40 rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/60 resize-none overflow-hidden ${monoFont ? 'font-mono text-xs' : ''}`}
        />
      ) : value ? (
        <div className="rounded-md bg-[#0D0D0D] border border-[#1A1A1A] px-3 py-2.5 min-h-[2.5rem]">
          <MarkdownContent content={value} />
        </div>
      ) : (
        <button
          onClick={handleEdit}
          className="w-full text-left px-3 py-2.5 rounded-md border border-dashed border-[var(--border-primary)] text-xs text-[var(--text-dim)] hover:text-[var(--text-dim)] hover:border-[#333333] transition-colors"
        >
          {placeholder ?? `Add ${label.toLowerCase()}...`}
        </button>
      )}
    </div>
  );
}

const TEMPLATE_VARIABLES = [
  { category: 'Task', vars: [
    { name: 'task.title', desc: 'Task title' },
    { name: 'task.summary', desc: 'Brief description' },
    { name: 'task.description', desc: 'Full description' },
    { name: 'task.priority', desc: 'Priority level (critical, high, medium, low)' },
    { name: 'task.assigned_role', desc: 'Assigned role slug' },
    { name: 'task.estimated_effort', desc: 'Effort estimate (XS, S, M, L, XL)' },
    { name: 'task.tags', desc: 'Array of tags (use {{join .task.tags ", "}})' },
    { name: 'task.test_command', desc: 'Inferred test command from role tech stack' },
    { name: 'task.test_file', desc: 'Test file extracted from red dependency' },
    { name: 'task.test_name', desc: 'Test function name from red dependency' },
  ]},
  { category: 'Project', vars: [
    { name: 'project.name', desc: 'Project name' },
  ]},
  { category: 'Dependencies', vars: [
    { name: 'dependencies.all', desc: 'All dependency summaries combined' },
    { name: 'dependencies.red', desc: 'Red dependency completion summary' },
    { name: 'dependencies.green', desc: 'Green dependency completion summary' },
  ]},
  { category: 'Dependency (by role)', vars: [
    { name: 'dependency.{role}.title', desc: 'Dependency task title' },
    { name: 'dependency.{role}.completion_summary', desc: 'Completion summary' },
    { name: 'dependency.{role}.task_id', desc: 'Task ID' },
    { name: 'dependency.{role}.files_modified', desc: 'Modified files (comma-separated)' },
    { name: 'dependency.{role}.failure_output', desc: 'Failure output (red only)' },
  ]},
  { category: 'Context Files', vars: [
    { name: 'context_files', desc: 'Combined contents of task context files' },
    { name: 'context_files_signatures', desc: 'Go function/type signatures from context files' },
  ]},
];

function TemplateVariablesPanel() {
  const [open, setOpen] = useState(false);

  return (
    <div className="rounded-md border border-[var(--border-primary)] bg-[var(--bg-secondary)]">
      <button
        onClick={() => setOpen(!open)}
        data-qa="template-variables-toggle"
        className="w-full flex items-center gap-2 px-3 py-2.5 text-left text-xs text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
      >
        <Info size={13} className="text-[var(--primary)] shrink-0" />
        <span className="font-mono">Template Variables Reference</span>
        <span className="ml-auto">
          {open ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
        </span>
      </button>
      {open && (
        <div className="px-3 pb-3 space-y-3 border-t border-[var(--border-primary)] pt-3">
          <p className="text-[11px] text-[var(--text-dim)]">
            Use <code className="px-1 py-0.5 bg-[var(--bg-primary)] rounded text-[var(--text-muted)]">{'{{variable.name}}'}</code> syntax.
            Available function: <code className="px-1 py-0.5 bg-[var(--bg-primary)] rounded text-[var(--text-muted)]">{'{{join .array ", "}}'}</code>
          </p>
          {TEMPLATE_VARIABLES.map((group) => (
            <div key={group.category}>
              <h4 className="text-[11px] font-mono font-medium text-[var(--text-muted)] mb-1.5">{group.category}</h4>
              <div className="space-y-0.5">
                {group.vars.map((v) => (
                  <div key={v.name} className="flex items-baseline gap-3 text-[11px]">
                    <code className="font-mono text-[var(--primary)] shrink-0">{`{{${v.name}}}`}</code>
                    <span className="text-[var(--text-dim)]">{v.desc}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface RoleModalProps {
  role: AgentResponse | null;
  projectId?: string;
  onClose: () => void;
  onSaved: () => void;
  onDeleted: () => void;
}

function RoleModal({ role, projectId, onClose, onSaved, onDeleted }: RoleModalProps) {
  const isEdit = !!role;
  // Local role state for optimistic updates of description/prompt_hint
  const [localRole, setLocalRole] = useState<AgentResponse | null>(role);
  const [name, setName] = useState(role?.name ?? '');
  const [slug, setSlug] = useState(role?.slug ?? '');
  const [icon, setIcon] = useState(role?.icon ?? '');
  const [color, setColor] = useState(role?.color ?? PRESET_COLORS[0]);
  const [description, setDescription] = useState(role?.description ?? '');
  const [promptTemplate, setPromptTemplate] = useState(role?.prompt_template ?? '');
  const [promptHint, setPromptHint] = useState(role?.prompt_hint ?? '');
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);
  const [iconPickerOpen, setIconPickerOpen] = useState(false);

  const apiUpdate = (slug: string, data: UpdateAgentRequest) =>
    projectId ? updateProjectAgent(projectId, slug, data) : updateAgent(slug, data);
  const apiCreate = (data: CreateAgentRequest) =>
    projectId ? createProjectAgent(projectId, data) : createAgent(data);
  const apiDelete = (slug: string) =>
    projectId ? deleteProjectAgent(projectId, slug) : deleteAgent(slug);

  // Inline save handlers for description and prompt_hint (edit mode only)
  const handleSaveDescription = async (newValue: string) => {
    if (!localRole) return;
    const updated = await apiUpdate(localRole.slug, { description: newValue });
    setLocalRole(updated);
    setDescription(newValue);
  };

  const handleSavePromptTemplate = async (newValue: string) => {
    if (!localRole) return;
    const updated = await apiUpdate(localRole.slug, { prompt_template: newValue });
    setLocalRole(updated);
    setPromptTemplate(newValue);
  };

  const handleSavePromptHint = async (newValue: string) => {
    if (!localRole) return;
    const updated = await apiUpdate(localRole.slug, { prompt_hint: newValue });
    setLocalRole(updated);
    setPromptHint(newValue);
  };

  const generateSlug = (n: string) =>
    n
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '')
      .slice(0, 50);

  const handleNameChange = (val: string) => {
    setName(val);
    if (autoSlug) {
      setSlug(generateSlug(val));
    }
  };

  const handleSlugChange = (val: string) => {
    setAutoSlug(false);
    setSlug(val.toLowerCase().replace(/[^a-z0-9]/g, ''));
  };

  const handleSave = async () => {
    if (!name.trim() || !slug.trim()) return;
    setSaving(true);
    try {
      if (isEdit) {
        const data: UpdateAgentRequest = {
          name: name.trim(),
          icon: icon.trim(),
          color,
          description: description.trim(),
          prompt_template: promptTemplate.trim(),
          prompt_hint: promptHint.trim(),
        };
        await apiUpdate(role.slug, data);
      } else {
        const data: CreateAgentRequest = {
          slug: slug.trim(),
          name: name.trim(),
          icon: icon.trim(),
          color,
          description: description.trim(),
          prompt_template: promptTemplate.trim(),
          prompt_hint: promptHint.trim(),
        };
        await apiCreate(data);
      }
      onSaved();
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!role) return;
    setDeleting(true);
    try {
      await apiDelete(role.slug);
      onDeleted();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="flex-1 bg-black/50" onClick={onClose} />
      <div
        className="w-[1040px] h-full bg-[var(--bg-primary)] border-l border-[var(--border-primary)] flex flex-col animate-[slide-in-right_0.2s_ease-out]"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-primary)]">
          <h2 className="text-lg text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            {isEdit ? 'Edit Agent' : 'New Agent'}
          </h2>
          <button
            onClick={onClose}
            data-qa="agent-modal-close-btn"
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-5">
          {/* Name & Slug row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="e.g. Backend Developer"
                data-qa="agent-name-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
                autoFocus
              />
            </div>
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Slug</label>
              <input
                type="text"
                value={slug}
                onChange={(e) => handleSlugChange(e.target.value)}
                placeholder="backenddev"
                disabled={isEdit}
                data-qa="agent-slug-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 disabled:opacity-50 font-mono"
              />
            </div>
          </div>

          {/* Icon & Color row */}
          <div className="flex items-start gap-6">
            <div className="relative">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Icon</label>
              <button
                type="button"
                onClick={() => setIconPickerOpen(!iconPickerOpen)}
                data-qa="agent-modal-icon-toggle"
                className="w-10 h-10 rounded-md text-xl flex items-center justify-center bg-[var(--bg-secondary)] border border-[var(--border-primary)] hover:border-[var(--primary)]/50 transition-all"
              >
                {icon || '?'}
              </button>
              {iconPickerOpen && (
                <div className="absolute top-full left-0 mt-1 z-10 p-2 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-primary)] shadow-lg">
                  <div className="grid grid-cols-10 gap-1">
                    {PRESET_ICONS.map((ic) => (
                      <button
                        key={ic}
                        onClick={() => { setIcon(ic); setIconPickerOpen(false); }}
                        data-qa="agent-modal-icon-btn"
                        className={`w-8 h-8 rounded-md text-base flex items-center justify-center transition-all ${
                          icon === ic
                            ? 'bg-[var(--primary)]/20 border border-[var(--primary)]/50 scale-110'
                            : 'bg-[var(--bg-secondary)] border border-transparent hover:border-[var(--border-secondary)] hover:bg-[var(--bg-tertiary)]'
                        }`}
                      >
                        {ic}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Color</label>
              <div className="flex items-center gap-2 h-10">
                {PRESET_COLORS.map((c) => (
                  <button
                    key={c}
                    onClick={() => setColor(c)}
                    data-qa="agent-modal-color-btn"
                    className={`w-7 h-7 rounded-full border-2 transition-all ${
                      color === c ? 'border-white scale-110' : 'border-transparent'
                    }`}
                    style={{ backgroundColor: c }}
                  />
                ))}
              </div>
            </div>
          </div>

          {/* Description */}
          {isEdit ? (
            <InlineEditField
              label="Description"
              value={localRole?.description ?? description}
              placeholder="Describe this agent..."
              onSave={handleSaveDescription}
            />
          ) : (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this agent..."
                rows={4}
                data-qa="agent-modal-description-textarea"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
              />
            </div>
          )}

          {/* Prompt Template */}
          {isEdit ? (
            <InlineEditField
              label="Prompt Template"
              value={localRole?.prompt_template ?? promptTemplate}
              placeholder="Go template for rendering agent prompts..."
              monoFont
              onSave={handleSavePromptTemplate}
            />
          ) : (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Prompt Template</label>
              <textarea
                value={promptTemplate}
                onChange={(e) => setPromptTemplate(e.target.value)}
                placeholder="Go template for rendering agent prompts..."
                rows={6}
                data-qa="agent-modal-prompt-template-textarea"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y font-mono text-xs"
              />
            </div>
          )}

          {/* Template Variables Info */}
          <TemplateVariablesPanel />

          {/* Prompt Hint */}
          {isEdit ? (
            <InlineEditField
              label="Prompt Hint"
              value={localRole?.prompt_hint ?? promptHint}
              placeholder="System prompt context for this agent..."
              monoFont
              onSave={handleSavePromptHint}
            />
          ) : (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Prompt Hint</label>
              <textarea
                value={promptHint}
                onChange={(e) => setPromptHint(e.target.value)}
                placeholder="System prompt context for this agent..."
                rows={6}
                data-qa="agent-modal-prompt-hint-textarea"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y font-mono text-xs"
              />
            </div>
          )}

          {/* Agent Skills */}
          {isEdit && localRole && (
            <AgentSkillsPanel
              agentSlug={localRole.slug}
              agentName={localRole.name}
            />
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-primary)]">
          <div>
            {isEdit && (
              <button
                onClick={handleDelete}
                disabled={deleting}
                data-qa="agent-delete-btn"
                className="flex items-center gap-1.5 text-sm text-[#F06060] hover:text-[#FF3B30] transition-colors disabled:opacity-50"
              >
                <Trash2 size={14} />
                {deleting ? 'Deleting...' : 'Delete Agent'}
              </button>
            )}
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              data-qa="agent-cancel-btn"
              className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={!name.trim() || !slug.trim() || saving}
              data-qa="agent-save-btn"
              className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Agent'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
