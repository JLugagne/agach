import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Plus,
  X,
  Loader2,
  Trash2,
  Pencil,
  Check,
} from 'lucide-react';
import { listRoles, createRole, updateRole, deleteRole } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { RoleResponse, CreateRoleRequest, UpdateRoleRequest } from '../lib/types';
import MarkdownContent from '../components/ui/MarkdownContent';

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
  const [roles, setRoles] = useState<RoleResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<RoleResponse | null>(null);

  const fetchRoles = useCallback(async () => {
    try {
      const data = await listRoles();
      setRoles(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRoles();
  }, [fetchRoles]);

  useWebSocket(
    useCallback(
      (event) => {
        if (
          event.type === 'role_created' ||
          event.type === 'role_updated' ||
          event.type === 'role_deleted'
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
            <p className="text-sm text-[#555555]">
              {roles.length} role{roles.length !== 1 ? 's' : ''} defined
            </p>
          </div>
          <button
            onClick={openCreate}
            className="flex items-center gap-1.5 px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 transition-colors"
          >
            <Plus size={15} />
            New Role
          </button>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[#555555]" size={24} />
          </div>
        ) : roles.length === 0 ? (
          <div className="text-center py-24">
            <p className="text-[#555555] text-sm mb-4">No roles defined yet.</p>
            <button
              onClick={openCreate}
              className="text-sm text-[#00C896] hover:text-[#00C896]/80 transition-colors"
            >
              Create your first role
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {roles.map((role) => (
              <RoleCard key={role.id} role={role} onClick={() => openEdit(role)} />
            ))}
          </div>
        )}
      </div>

      {/* Role Modal */}
      {modalOpen && (
        <RoleModal
          role={editingRole}
          onClose={closeModal}
          onSaved={handleSaved}
          onDeleted={handleDeleted}
        />
      )}
    </div>
  );
}

function RoleCard({ role, onClick }: { role: RoleResponse; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="rounded-lg bg-[#111111] border border-[#1E1E1E] p-5 text-left hover:border-[#252525] transition-colors cursor-pointer w-full"
    >
      <div className="flex items-start gap-3 mb-3">
        <span className="text-xl">{role.icon || '\u2B22'}</span>
        <div className="flex-1 min-w-0">
          <h3 className="font-heading text-[15px] text-[#F0F0F0] truncate">{role.name}</h3>
          <p className="font-mono text-[11px] text-[#444444]">{role.slug}</p>
        </div>
        <div
          className="w-3 h-3 rounded-full shrink-0 mt-1"
          style={{ backgroundColor: role.color || '#6B7280' }}
        />
      </div>

      {role.description && (
        <p className="text-xs text-[#888888] mb-3 line-clamp-2">{role.description}</p>
      )}

      {role.tech_stack && role.tech_stack.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {role.tech_stack.slice(0, 4).map((tech) => (
            <span
              key={tech}
              className="px-2 py-0.5 bg-[#1A1A1A] border border-[#252525] rounded text-[10px] font-mono text-[#888888]"
            >
              {tech}
            </span>
          ))}
          {role.tech_stack.length > 4 && (
            <span className="px-2 py-0.5 text-[10px] text-[#555555]">
              +{role.tech_stack.length - 4}
            </span>
          )}
        </div>
      )}
    </button>
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
        <label className="text-xs font-mono text-[#555555]">{label}</label>
        {!editing && (
          <button
            onClick={handleEdit}
            className="flex items-center gap-1 text-[#444444] hover:text-[#888888] transition-colors"
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
              className="flex items-center gap-1 text-xs text-[#555555] hover:text-[#888888] transition-colors disabled:opacity-50"
            >
              <X size={12} />
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
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
          className={`w-full bg-[#1A1A1A] border border-[#00C896]/40 rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/60 resize-none overflow-hidden ${monoFont ? 'font-mono text-xs' : ''}`}
        />
      ) : value ? (
        <div className="rounded-md bg-[#0D0D0D] border border-[#1A1A1A] px-3 py-2.5 min-h-[2.5rem]">
          <MarkdownContent content={value} />
        </div>
      ) : (
        <button
          onClick={handleEdit}
          className="w-full text-left px-3 py-2.5 rounded-md border border-dashed border-[#252525] text-xs text-[#444444] hover:text-[#555555] hover:border-[#333333] transition-colors"
        >
          {placeholder ?? `Add ${label.toLowerCase()}...`}
        </button>
      )}
    </div>
  );
}

interface RoleModalProps {
  role: RoleResponse | null;
  onClose: () => void;
  onSaved: () => void;
  onDeleted: () => void;
}

function RoleModal({ role, onClose, onSaved, onDeleted }: RoleModalProps) {
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

  // Inline save handlers for description and prompt_hint (edit mode only)
  const handleSaveDescription = async (newValue: string) => {
    if (!localRole) return;
    const updated = await updateRole(localRole.slug, { description: newValue });
    setLocalRole(updated);
    setDescription(newValue);
  };

  const handleSavePromptHint = async (newValue: string) => {
    if (!localRole) return;
    const updated = await updateRole(localRole.slug, { prompt_hint: newValue });
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
        await updateRole(role.slug, data);
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
        await createRole(data);
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
      await deleteRole(role.slug);
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
            className="text-[#555555] hover:text-[#888888] transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-5">
          {/* Name & Slug row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-mono text-[#555555] mb-1.5">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="e.g. Backend Developer"
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/50"
                autoFocus
              />
            </div>
            <div>
              <label className="block text-xs font-mono text-[#555555] mb-1.5">Slug</label>
              <input
                type="text"
                value={slug}
                onChange={(e) => handleSlugChange(e.target.value)}
                placeholder="backenddev"
                disabled={isEdit}
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/50 disabled:opacity-50 font-mono"
              />
            </div>
          </div>

          {/* Icon & Color row */}
          <div className="flex items-end gap-6">
            <div>
              <label className="block text-xs font-mono text-[#555555] mb-1.5">Icon (emoji)</label>
              <input
                type="text"
                value={icon}
                onChange={(e) => setIcon(e.target.value)}
                placeholder="e.g. \uD83D\uDE80"
                maxLength={10}
                className="w-24 bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-center text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/50"
              />
            </div>
            <div>
              <label className="block text-xs font-mono text-[#555555] mb-1.5">Color</label>
              <div className="flex items-center gap-2">
                {PRESET_COLORS.map((c) => (
                  <button
                    key={c}
                    onClick={() => setColor(c)}
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
              <label className="block text-xs font-mono text-[#555555] mb-1.5">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe this role..."
                rows={4}
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/50 resize-y"
              />
            </div>
          )}

          {/* Tech Stack */}
          <div>
            <label className="block text-xs font-mono text-[#555555] mb-1.5">Tech Stack</label>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {techStack.map((tech) => (
                <span
                  key={tech}
                  className="inline-flex items-center gap-1 px-2 py-0.5 bg-[#1A1A1A] border border-[#252525] rounded text-xs font-mono text-[#888888]"
                >
                  {tech}
                  <button
                    onClick={() => removeTech(tech)}
                    className="text-[#555555] hover:text-[#F06060] transition-colors"
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
              className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/50"
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
              <label className="block text-xs font-mono text-[#555555] mb-1.5">Prompt Hint</label>
              <textarea
                value={promptHint}
                onChange={(e) => setPromptHint(e.target.value)}
                placeholder="System prompt context for this role..."
                rows={6}
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[#333333] focus:outline-none focus:border-[#00C896]/50 resize-y font-mono text-xs"
              />
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[#1E1E1E]">
          <div>
            {isEdit && (
              <button
                onClick={handleDelete}
                disabled={deleting}
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
              className="px-4 py-2 text-sm text-[#888888] hover:text-[#E0E0E0] transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={!name.trim() || !slug.trim() || saving}
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
