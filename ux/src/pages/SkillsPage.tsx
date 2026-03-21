import { useState, useEffect, useCallback } from 'react';
import { Plus, X, Loader2, Trash2, Pencil } from 'lucide-react';
import { listSkills, createSkill, updateSkill, deleteSkill } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { SkillResponse, CreateSkillRequest, UpdateSkillRequest } from '../lib/types';

const PRESET_COLORS = [
  '#00C896',
  '#F09060',
  '#6C63FF',
  '#FF6B9D',
  '#FFD060',
  '#00B4D8',
  '#FF3B30',
];

export default function SkillsPage() {
  const [skills, setSkills] = useState<SkillResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingSkill, setEditingSkill] = useState<SkillResponse | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<SkillResponse | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const fetchSkills = useCallback(async () => {
    try {
      const data = await listSkills();
      setSkills(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSkills();
  }, [fetchSkills]);

  useWebSocket(
    useCallback(
      (event) => {
        if (
          event.type === 'skill_created' ||
          event.type === 'skill_updated' ||
          event.type === 'skill_deleted'
        ) {
          fetchSkills();
        }
      },
      [fetchSkills],
    ),
  );

  const openCreate = () => {
    setEditingSkill(null);
    setModalOpen(true);
  };

  const openEdit = (skill: SkillResponse) => {
    setEditingSkill(skill);
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditingSkill(null);
  };

  const handleSaved = () => {
    closeModal();
    fetchSkills();
  };

  const handleDeleteClick = (skill: SkillResponse) => {
    setDeleteConfirm(skill);
    setDeleteError(null);
  };

  const handleDeleteConfirm = async () => {
    if (!deleteConfirm) return;
    try {
      await deleteSkill(deleteConfirm.slug);
      setDeleteConfirm(null);
      setDeleteError(null);
      fetchSkills();
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg.toLowerCase().includes('in use') || msg.toLowerCase().includes('409') || msg.toLowerCase().includes('assigned')) {
        setDeleteError('Cannot delete: skill is still assigned to one or more agents.');
      } else {
        setDeleteError(msg || 'Failed to delete skill.');
      }
    }
  };

  const handleDeleteCancel = () => {
    setDeleteConfirm(null);
    setDeleteError(null);
  };

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-5xl mx-auto px-8 py-12">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-[28px] text-[#F0F0F0] mb-1" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Skills</h1>
            <p className="text-sm text-[var(--text-dim)]">
              {skills.length} skill{skills.length !== 1 ? 's' : ''} defined
            </p>
          </div>
          <button
            onClick={openCreate}
            className="flex items-center gap-1.5 px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 transition-colors"
          >
            <Plus size={15} />
            New Skill
          </button>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
          </div>
        ) : skills.length === 0 ? (
          <div className="text-center py-24">
            <p className="text-[var(--text-dim)] text-sm mb-4">No skills yet. Create your first skill to get started.</p>
            <button
              onClick={openCreate}
              className="text-sm text-[#00C896] hover:text-[#00C896]/80 transition-colors"
            >
              Create your first skill
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {skills.map((skill) => (
              <div key={skill.id}>
                <SkillCard
                  skill={skill}
                  onEdit={() => openEdit(skill)}
                  onDelete={() => handleDeleteClick(skill)}
                />
                {deleteConfirm?.id === skill.id && (
                  <div className="mt-2 p-3 rounded-md bg-[#1A1A1A] border border-[#F06060]/30">
                    <p className="text-xs text-[var(--text-muted)] mb-2">
                      Are you sure? This skill will be removed from all agents.
                    </p>
                    {deleteError && (
                      <p className="text-xs text-[#F06060] mb-2">{deleteError}</p>
                    )}
                    <div className="flex items-center gap-2">
                      <button
                        onClick={handleDeleteConfirm}
                        className="px-3 py-1 bg-[#F06060] text-white text-xs rounded-md hover:bg-[#FF3B30] transition-colors"
                      >
                        Confirm
                      </button>
                      <button
                        onClick={handleDeleteCancel}
                        className="px-3 py-1 text-xs text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {modalOpen && (
        <SkillModal
          skill={editingSkill}
          onClose={closeModal}
          onSaved={handleSaved}
        />
      )}
    </div>
  );
}

function SkillCard({
  skill,
  onEdit,
  onDelete,
}: {
  skill: SkillResponse;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="rounded-lg bg-[#111111] border border-[#1E1E1E] p-5 text-left transition-colors hover:border-[#252525] w-full">
      <div className="flex items-start gap-3 mb-3">
        <span className="text-xl">{skill.icon || '\u2B22'}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="font-heading text-[15px] text-[#F0F0F0] truncate">{skill.name}</h3>
            {skill.content && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono bg-[#00C896]/10 text-[#00C896] border border-[#00C896]/20 shrink-0">
                Has content
              </span>
            )}
          </div>
          <p className="font-mono text-[11px] text-[var(--text-dim)]">{skill.slug}</p>
        </div>
        <div className="flex items-center gap-2 shrink-0 mt-0.5">
          <button
            onClick={onEdit}
            title="Edit skill"
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
          >
            <Pencil size={13} />
          </button>
          <button
            onClick={onDelete}
            title="Delete skill"
            className="text-[var(--text-dim)] hover:text-[#F06060] transition-colors"
          >
            <Trash2 size={13} />
          </button>
          <div
            className="w-3 h-3 rounded-full"
            style={{ backgroundColor: skill.color || '#6B7280' }}
          />
        </div>
      </div>
      {skill.description && (
        <p className="text-xs text-[var(--text-muted)] line-clamp-2">{skill.description}</p>
      )}
    </div>
  );
}

interface SkillModalProps {
  skill: SkillResponse | null;
  onClose: () => void;
  onSaved: () => void;
}

function SkillModal({ skill, onClose, onSaved }: SkillModalProps) {
  const isEdit = !!skill;
  const [slug, setSlug] = useState(skill?.slug ?? '');
  const [name, setName] = useState(skill?.name ?? '');
  const [description, setDescription] = useState(skill?.description ?? '');
  const [content, setContent] = useState(skill?.content ?? '');
  const [icon, setIcon] = useState(skill?.icon ?? '');
  const [color, setColor] = useState(skill?.color ?? PRESET_COLORS[0]);
  const [sortOrder, setSortOrder] = useState(skill?.sort_order ?? 0);
  const [saving, setSaving] = useState(false);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

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
    setSlug(val.toLowerCase().replace(/[^a-z0-9-]/g, ''));
  };

  const handleSave = async () => {
    if (!name.trim() || !slug.trim()) return;
    setSaving(true);
    try {
      if (isEdit) {
        const data: UpdateSkillRequest = {
          name: name.trim(),
          description: description.trim(),
          content,
          icon: icon.trim(),
          color,
          sort_order: sortOrder,
        };
        await updateSkill(skill.slug, data);
      } else {
        const data: CreateSkillRequest = {
          slug: slug.trim(),
          name: name.trim(),
          description: description.trim(),
          content,
          icon: icon.trim(),
          color,
          sort_order: sortOrder,
        };
        await createSkill(data);
      }
      onSaved();
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="flex-1 bg-black/50" onClick={onClose} />
      <div className="w-[720px] h-full bg-[#111111] border-l border-[#1E1E1E] flex flex-col animate-[slide-in-right_0.2s_ease-out]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[#1E1E1E]">
          <h2 className="text-lg text-[#F0F0F0]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            {isEdit ? 'Edit Skill' : 'New Skill'}
          </h2>
          <button
            onClick={onClose}
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
                placeholder="e.g. Go Testing"
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
                placeholder="gotesting"
                disabled={isEdit}
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
                placeholder="e.g. \uD83D\uDCDA"
                maxLength={10}
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
                    className={`w-7 h-7 rounded-full border-2 transition-all ${
                      color === c ? 'border-white scale-110' : 'border-transparent'
                    }`}
                    style={{ backgroundColor: c }}
                  />
                ))}
              </div>
            </div>
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Sort Order</label>
              <input
                type="number"
                value={sortOrder}
                onChange={(e) => setSortOrder(Number(e.target.value))}
                className="w-20 bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] focus:outline-none focus:border-[#00C896]/50"
              />
            </div>
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe this skill..."
              rows={3}
              className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 resize-y"
            />
          </div>

          {/* Content */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Content (Markdown)</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Markdown content..."
              rows={10}
              className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 resize-y font-mono text-xs"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end px-6 py-4 border-t border-[#1E1E1E] gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!name.trim() || !slug.trim() || saving}
            className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Skill'}
          </button>
        </div>
      </div>
    </div>
  );
}
