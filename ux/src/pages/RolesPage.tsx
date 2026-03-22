import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams } from 'react-router-dom';
import {
  Plus,
  X,
  Loader2,
  Trash2,
  Pencil,
  Check,
  Star,
  Copy,
} from 'lucide-react';
import {
  listRoles, createRole, updateRole, deleteRole,
  listProjectRoles, createProjectRole, updateProjectRole, deleteProjectRole,
  getProject, updateProject,
} from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { RoleResponse, CreateRoleRequest, UpdateRoleRequest } from '../lib/types';
import MarkdownContent from '../components/ui/MarkdownContent';
import CloneAgentDialog from '../components/CloneAgentDialog';
import AgentSkillsPanel from '../components/AgentSkillsPanel';

const PRESET_COLORS = [
  '#00C896',
  '#F09060',
  '#6C63FF',
  '#FF6B9D',
  '#FFD060',
  '#00B4D8',
  '#FF3B30',
];

export default function RolesPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [roles, setRoles] = useState<RoleResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<RoleResponse | null>(null);
  const [defaultRole, setDefaultRole] = useState<string>('');
  const [cloningRole, setCloningRole] = useState<RoleResponse | null>(null);

  const fetchRoles = useCallback(async () => {
    try {
      const data = projectId ? await listProjectRoles(projectId) : await listRoles();
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
          event.type === 'role_created' ||
          event.type === 'role_updated' ||
          event.type === 'role_deleted' ||
          event.type === 'agent_cloned'
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

  const openEdit = (role: RoleResponse) => {
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
      <div className="max-w-5xl mx-auto px-8 py-12">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-[28px] text-[#F0F0F0] mb-1" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Roles</h1>
            <p className="text-sm text-[var(--text-dim)]">
              {roles.length} role{roles.length !== 1 ? 's' : ''} defined
            </p>
          </div>
          <button
            onClick={openCreate}
            data-qa="new-role-btn"
            className="flex items-center gap-1.5 px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 transition-colors"
          >
            <Plus size={15} />
            New Role
          </button>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
          </div>
        ) : roles.length === 0 ? (
          <div className="text-center py-24">
            <p className="text-[var(--text-dim)] text-sm mb-4">No roles defined yet.</p>
            <button
              onClick={openCreate}
              data-qa="roles-create-first-role-btn"
              className="text-sm text-[#00C896] hover:text-[#00C896]/80 transition-colors"
            >
              Create your first role
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
                onClick={() => openEdit(role)}
                onClone={() => setCloningRole(role)}
              />
            ))}
          </div>
        )}
      </div>

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
  role: RoleResponse;
  isDefault: boolean;
  onSetDefault?: () => void;
  onClick: () => void;
  onClone: () => void;
}) {
  return (
    <div
      data-qa="role-card"
      className={`rounded-lg bg-[#111111] border p-5 text-left transition-colors w-full ${
        isDefault ? 'border-[#00C896]/40' : 'border-[#1E1E1E] hover:border-[#252525]'
      }`}
    >
      <div className="flex items-start gap-3 mb-3">
        <button onClick={onClick} data-qa="role-card-icon-btn" className="text-xl cursor-pointer">{role.icon || '\u2B22'}</button>
        <div className="flex-1 min-w-0 cursor-pointer" onClick={onClick}>
          <div className="flex items-center gap-2">
            <h3 className="font-heading text-[15px] text-[#F0F0F0] truncate">{role.name}</h3>
            {isDefault && (
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono bg-[#00C896]/10 text-[#00C896] border border-[#00C896]/20 shrink-0">
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
              data-qa="role-card-set-default-btn"
              title={isDefault ? 'Unset default' : 'Set as default'}
              className={`transition-colors ${
                isDefault
                  ? 'text-[#00C896] hover:text-[#00C896]/60'
                  : 'text-[var(--text-dim)] hover:text-[#00C896]'
              }`}
            >
              <Star size={14} fill={isDefault ? 'currentColor' : 'none'} />
            </button>
          )}
          <button
            onClick={(e) => { e.stopPropagation(); onClone(); }}
            data-qa="role-card-clone-btn"
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

        {role.tech_stack && role.tech_stack.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {role.tech_stack.slice(0, 4).map((tech) => (
              <span
                key={tech}
                className="px-2 py-0.5 bg-[#1A1A1A] border border-[#252525] rounded text-[10px] font-mono text-[var(--text-muted)]"
              >
                {tech}
              </span>
            ))}
            {role.tech_stack.length > 4 && (
              <span className="px-2 py-0.5 text-[10px] text-[var(--text-dim)]">
                +{role.tech_stack.length - 4}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
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
              className="flex items-center gap-1 text-xs text-[#00C896] hover:text-[#00C896]/80 transition-colors disabled:opacity-50"
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
          className={`w-full bg-[#1A1A1A] border border-[#00C896]/40 rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/60 resize-none overflow-hidden ${monoFont ? 'font-mono text-xs' : ''}`}
        />
      ) : value ? (
        <div className="rounded-md bg-[#0D0D0D] border border-[#1A1A1A] px-3 py-2.5 min-h-[2.5rem]">
          <MarkdownContent content={value} />
        </div>
      ) : (
        <button
          onClick={handleEdit}
          className="w-full text-left px-3 py-2.5 rounded-md border border-dashed border-[#252525] text-xs text-[var(--text-dim)] hover:text-[var(--text-dim)] hover:border-[#333333] transition-colors"
        >
          {placeholder ?? `Add ${label.toLowerCase()}...`}
        </button>
      )}
    </div>
  );
}

interface RoleModalProps {
  role: RoleResponse | null;
  projectId?: string;
  onClose: () => void;
  onSaved: () => void;
  onDeleted: () => void;
}

function RoleModal({ role, projectId, onClose, onSaved, onDeleted }: RoleModalProps) {
  const isEdit = !!role;
  // Local role state for optimistic updates of description/prompt_hint
  const [localRole, setLocalRole] = useState<RoleResponse | null>(role);
  const [name, setName] = useState(role?.name ?? '');
  const [slug, setSlug] = useState(role?.slug ?? '');
  const [icon, setIcon] = useState(role?.icon ?? '');
  const [color, setColor] = useState(role?.color ?? PRESET_COLORS[0]);
  const [description, setDescription] = useState(role?.description ?? '');
  const [techStack, setTechStack] = useState<string[]>(role?.tech_stack ?? []);
  const [techInput, setTechInput] = useState('');
  const [promptHint, setPromptHint] = useState(role?.prompt_hint ?? '');
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  const apiUpdate = (slug: string, data: UpdateRoleRequest) =>
    projectId ? updateProjectRole(projectId, slug, data) : updateRole(slug, data);
  const apiCreate = (data: CreateRoleRequest) =>
    projectId ? createProjectRole(projectId, data) : createRole(data);
  const apiDelete = (slug: string) =>
    projectId ? deleteProjectRole(projectId, slug) : deleteRole(slug);

  // Inline save handlers for description and prompt_hint (edit mode only)
  const handleSaveDescription = async (newValue: string) => {
    if (!localRole) return;
    const updated = await apiUpdate(localRole.slug, { description: newValue });
    setLocalRole(updated);
    setDescription(newValue);
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

  const addTech = () => {
    const val = techInput.trim();
    if (val && !techStack.includes(val)) {
      setTechStack([...techStack, val]);
    }
    setTechInput('');
  };

  const removeTech = (tech: string) => {
    setTechStack(techStack.filter((t) => t !== tech));
  };

  const handleSave = async () => {
    if (!name.trim() || !slug.trim()) return;
    setSaving(true);
    try {
      if (isEdit) {
        const data: UpdateRoleRequest = {
          name: name.trim(),
          icon: icon.trim(),
          color,
          description: description.trim(),
          tech_stack: techStack,
          prompt_hint: promptHint.trim(),
        };
        await apiUpdate(role.slug, data);
      } else {
        const data: CreateRoleRequest = {
          slug: slug.trim(),
          name: name.trim(),
          icon: icon.trim(),
          color,
          description: description.trim(),
          tech_stack: techStack,
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
        className="w-[1040px] h-full bg-[#111111] border-l border-[#1E1E1E] flex flex-col animate-[slide-in-right_0.2s_ease-out]"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[#1E1E1E]">
          <h2 className="text-lg text-[#F0F0F0]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            {isEdit ? 'Edit Role' : 'New Role'}
          </h2>
          <button
            onClick={onClose}
            data-qa="role-modal-close-btn"
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
                data-qa="role-name-input"
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
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
                data-qa="role-slug-input"
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 disabled:opacity-50 font-mono"
              />
            </div>
          </div>

          {/* Icon & Color row */}
          <div className="flex items-end gap-6">
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Icon (emoji)</label>
              <input
                type="text"
                value={icon}
                onChange={(e) => setIcon(e.target.value)}
                placeholder="e.g. \uD83D\uDE80"
                maxLength={10}
                data-qa="role-modal-icon-input"
                className="w-24 bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-center text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
              />
            </div>
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Color</label>
              <div className="flex items-center gap-2">
                {PRESET_COLORS.map((c) => (
                  <button
                    key={c}
                    onClick={() => setColor(c)}
                    data-qa="role-modal-color-btn"
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
              placeholder="Describe this role..."
              onSave={handleSaveDescription}
            />
          ) : (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this role..."
                rows={4}
                data-qa="role-modal-description-textarea"
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 resize-y"
              />
            </div>
          )}

          {/* Tech Stack */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Tech Stack</label>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {techStack.map((tech) => (
                <span
                  key={tech}
                  className="inline-flex items-center gap-1 px-2 py-0.5 bg-[#1A1A1A] border border-[#252525] rounded text-xs font-mono text-[var(--text-muted)]"
                >
                  {tech}
                  <button
                    onClick={() => removeTech(tech)}
                    data-qa="role-modal-remove-tech-btn"
                    className="text-[var(--text-dim)] hover:text-[#F06060] transition-colors"
                  >
                    <X size={10} />
                  </button>
                </span>
              ))}
            </div>
            <input
              type="text"
              value={techInput}
              onChange={(e) => setTechInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  addTech();
                }
              }}
              placeholder="Type and press Enter"
              data-qa="role-modal-tech-input"
              className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
            />
          </div>

          {/* Prompt Hint */}
          {isEdit ? (
            <InlineEditField
              label="Prompt Hint"
              value={localRole?.prompt_hint ?? promptHint}
              placeholder="System prompt context for this role..."
              monoFont
              onSave={handleSavePromptHint}
            />
          ) : (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Prompt Hint</label>
              <textarea
                value={promptHint}
                onChange={(e) => setPromptHint(e.target.value)}
                placeholder="System prompt context for this role..."
                rows={6}
                data-qa="role-modal-prompt-hint-textarea"
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 resize-y font-mono text-xs"
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
        <div className="flex items-center justify-between px-6 py-4 border-t border-[#1E1E1E]">
          <div>
            {isEdit && (
              <button
                onClick={handleDelete}
                disabled={deleting}
                data-qa="role-delete-btn"
                className="flex items-center gap-1.5 text-sm text-[#F06060] hover:text-[#FF3B30] transition-colors disabled:opacity-50"
              >
                <Trash2 size={14} />
                {deleting ? 'Deleting...' : 'Delete Role'}
              </button>
            )}
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              data-qa="role-cancel-btn"
              className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={!name.trim() || !slug.trim() || saving}
              data-qa="role-save-btn"
              className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Role'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
