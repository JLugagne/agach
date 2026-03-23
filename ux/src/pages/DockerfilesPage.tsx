import { useState, useEffect, useCallback } from 'react';
import { Plus, X, Loader2, Trash2, Pencil, Tag, Container } from 'lucide-react';
import { listDockerfiles, createDockerfile, updateDockerfile, deleteDockerfile } from '../lib/api';
import type { DockerfileResponse, CreateDockerfileRequest, UpdateDockerfileRequest } from '../lib/types';

export default function DockerfilesPage() {
  const [dockerfiles, setDockerfiles] = useState<DockerfileResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingDockerfile, setEditingDockerfile] = useState<DockerfileResponse | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<DockerfileResponse | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const fetchDockerfiles = useCallback(async () => {
    try {
      const data = await listDockerfiles();
      setDockerfiles(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDockerfiles();
  }, [fetchDockerfiles]);

  const openCreate = () => {
    setEditingDockerfile(null);
    setModalOpen(true);
  };

  const openEdit = (dockerfile: DockerfileResponse) => {
    setEditingDockerfile(dockerfile);
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditingDockerfile(null);
  };

  const handleSaved = () => {
    closeModal();
    fetchDockerfiles();
  };

  const handleDeleteClick = (dockerfile: DockerfileResponse) => {
    setDeleteConfirm(dockerfile);
    setDeleteError(null);
  };

  const handleDeleteConfirm = async () => {
    if (!deleteConfirm) return;
    try {
      await deleteDockerfile(deleteConfirm.id);
      setDeleteConfirm(null);
      setDeleteError(null);
      fetchDockerfiles();
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg.toLowerCase().includes('in use') || msg.toLowerCase().includes('409')) {
        setDeleteError('Cannot delete: dockerfile is still assigned to one or more projects.');
      } else {
        setDeleteError(msg || 'Failed to delete dockerfile.');
      }
    }
  };

  const handleDeleteCancel = () => {
    setDeleteConfirm(null);
    setDeleteError(null);
  };

  // Group dockerfiles by slug for display
  const grouped = dockerfiles.reduce<Record<string, DockerfileResponse[]>>((acc, d) => {
    if (!acc[d.slug]) acc[d.slug] = [];
    acc[d.slug].push(d);
    return acc;
  }, {});

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-5xl mx-auto px-8 py-12">
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            Dockerfiles
          </h1>
          <button
            onClick={openCreate}
            data-qa="new-dockerfile-btn"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Dockerfile
          </button>
        </div>
        <p className="text-sm text-[var(--text-muted)] mb-10" style={{ fontFamily: 'Inter, sans-serif' }}>
          {dockerfiles.length} version{dockerfiles.length !== 1 ? 's' : ''} across {Object.keys(grouped).length} dockerfile{Object.keys(grouped).length !== 1 ? 's' : ''}
        </p>

        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
          </div>
        ) : dockerfiles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-5">
            <div className="w-20 h-20 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
              <Container size={36} className="text-[var(--text-muted)]" />
            </div>
            <p className="text-lg font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              No dockerfiles yet.
            </p>
            <p className="text-sm text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Get started by creating your first dockerfile
            </p>
            <button
              onClick={openCreate}
              data-qa="create-first-dockerfile-btn"
              className="flex items-center gap-2 px-6 py-3 rounded-lg text-sm font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              <Plus size={16} />
              Create your first dockerfile
            </button>
          </div>
        ) : (
          <div className="space-y-6">
            {Object.entries(grouped).map(([slug, versions]) => (
              <div key={slug}>
                <div className="flex items-center gap-2 mb-3">
                  <span className="font-mono text-xs text-[var(--text-dim)] bg-[var(--bg-secondary)] border border-[var(--border-primary)] px-2 py-0.5 rounded">
                    {slug}
                  </span>
                  <span className="text-xs text-[var(--text-dim)]">{versions.length} version{versions.length !== 1 ? 's' : ''}</span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {versions.map((dockerfile) => (
                    <div key={dockerfile.id}>
                      <DockerfileCard
                        dockerfile={dockerfile}
                        onEdit={() => openEdit(dockerfile)}
                        onDelete={() => handleDeleteClick(dockerfile)}
                      />
                      {deleteConfirm?.id === dockerfile.id && (
                        <div className="mt-2 p-3 rounded-md bg-[var(--bg-secondary)] border border-[#F06060]/30">
                          <p className="text-xs text-[var(--text-muted)] mb-2">
                            Are you sure? This will permanently delete this dockerfile version.
                          </p>
                          {deleteError && (
                            <p className="text-xs text-[#F06060] mb-2">{deleteError}</p>
                          )}
                          <div className="flex items-center gap-2">
                            <button
                              onClick={handleDeleteConfirm}
                              data-qa="confirm-delete-dockerfile-btn"
                              className="px-3 py-1 bg-[#F06060] text-white text-xs rounded-md hover:bg-[#FF3B30] transition-colors"
                            >
                              Confirm
                            </button>
                            <button
                              onClick={handleDeleteCancel}
                              data-qa="cancel-delete-dockerfile-btn"
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
              </div>
            ))}
          </div>
        )}
      </div>

      {modalOpen && (
        <DockerfileModal
          dockerfile={editingDockerfile}
          onClose={closeModal}
          onSaved={handleSaved}
        />
      )}
    </div>
  );
}

function DockerfileCard({
  dockerfile,
  onEdit,
  onDelete,
}: {
  dockerfile: DockerfileResponse;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="rounded-lg bg-[var(--bg-primary)] border border-[var(--border-primary)] p-5 text-left transition-colors hover:border-[var(--border-secondary)] w-full">
      <div className="flex items-start gap-3 mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="font-heading text-[15px] text-[var(--text-primary)] truncate">{dockerfile.name}</h3>
            <div className="flex items-center gap-1.5 shrink-0">
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono bg-[var(--bg-secondary)] text-[var(--text-muted)] border border-[var(--border-primary)]">
                <Tag size={9} />
                {dockerfile.version}
              </span>
              {dockerfile.is_latest && (
                <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono bg-[var(--primary)]/10 text-[var(--primary)] border border-[var(--primary)]/20">
                  latest
                </span>
              )}
            </div>
          </div>
          <p className="font-mono text-[11px] text-[var(--text-dim)] mt-0.5">{dockerfile.slug}</p>
        </div>
        <div className="flex items-center gap-2 shrink-0 mt-0.5">
          <button
            onClick={onEdit}
            title="Edit dockerfile"
            data-qa="dockerfile-edit-btn"
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
          >
            <Pencil size={13} />
          </button>
          <button
            onClick={onDelete}
            title="Delete dockerfile"
            data-qa="dockerfile-delete-btn"
            className="text-[var(--text-dim)] hover:text-[#F06060] transition-colors"
          >
            <Trash2 size={13} />
          </button>
        </div>
      </div>
      {dockerfile.description && (
        <p className="text-xs text-[var(--text-muted)] line-clamp-2">{dockerfile.description}</p>
      )}
      {dockerfile.content && (
        <p className="text-[10px] font-mono text-[var(--text-dim)] mt-2 bg-[#0D0D0D] rounded px-2 py-1 line-clamp-2 whitespace-pre">
          {dockerfile.content}
        </p>
      )}
    </div>
  );
}

interface DockerfileModalProps {
  dockerfile: DockerfileResponse | null;
  onClose: () => void;
  onSaved: () => void;
}

function DockerfileModal({ dockerfile, onClose, onSaved }: DockerfileModalProps) {
  const isEdit = !!dockerfile;
  const [slug, setSlug] = useState(dockerfile?.slug ?? '');
  const [name, setName] = useState(dockerfile?.name ?? '');
  const [description, setDescription] = useState(dockerfile?.description ?? '');
  const [version, setVersion] = useState(dockerfile?.version ?? '');
  const [content, setContent] = useState(dockerfile?.content ?? '');
  const [isLatest, setIsLatest] = useState(dockerfile?.is_latest ?? true);
  const [sortOrder, setSortOrder] = useState(dockerfile?.sort_order ?? 0);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [autoSlug, setAutoSlug] = useState(!isEdit);

  const generateSlug = (n: string) =>
    n
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '')
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
    if (!name.trim() || !slug.trim() || !version.trim()) return;
    setSaving(true);
    setError(null);
    try {
      if (isEdit) {
        const data: UpdateDockerfileRequest = {
          name: name.trim(),
          description: description.trim(),
          content,
          is_latest: isLatest,
          sort_order: sortOrder,
        };
        await updateDockerfile(dockerfile.id, data);
      } else {
        const data: CreateDockerfileRequest = {
          slug: slug.trim(),
          name: name.trim(),
          description: description.trim(),
          version: version.trim(),
          content,
          is_latest: isLatest,
          sort_order: sortOrder,
        };
        await createDockerfile(data);
      }
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save dockerfile.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="flex-1 bg-black/50" onClick={onClose} />
      <div className="w-[720px] h-full bg-[var(--bg-primary)] border-l border-[var(--border-primary)] flex flex-col animate-[slide-in-right_0.2s_ease-out]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-primary)]">
          <h2 className="text-lg text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            {isEdit ? 'Edit Dockerfile' : 'New Dockerfile'}
          </h2>
          <button
            onClick={onClose}
            data-qa="cancel-dockerfile-modal-btn"
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-5">
          {error && (
            <div className="p-3 bg-[#F06060]/10 border border-[#F06060]/30 rounded-md text-xs text-[#F06060]">
              {error}
            </div>
          )}

          {/* Name & Slug row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="e.g. Production API"
                data-qa="dockerfile-name-input"
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
                placeholder="production-api"
                disabled={isEdit}
                data-qa="dockerfile-slug-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 disabled:opacity-50 font-mono"
              />
            </div>
          </div>

          {/* Version & Latest row */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Version</label>
              <input
                type="text"
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                placeholder="e.g. 1.0.0 or 2024-01"
                disabled={isEdit}
                data-qa="dockerfile-version-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 disabled:opacity-50 font-mono"
              />
            </div>
            <div className="flex flex-col justify-end">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={isLatest}
                  onChange={(e) => setIsLatest(e.target.checked)}
                  data-qa="dockerfile-is-latest-checkbox"
                  className="w-4 h-4 rounded border-[var(--border-primary)] bg-[var(--bg-secondary)] accent-[var(--primary)]"
                />
                <span className="text-sm text-[var(--text-primary)]">Mark as latest</span>
              </label>
              <p className="text-xs text-[var(--text-dim)] mt-1">
                When a project is assigned this dockerfile, it will use the latest version by default.
              </p>
            </div>
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe this dockerfile configuration..."
              rows={2}
              data-qa="dockerfile-description-textarea"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
            />
          </div>

          {/* Content */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Docker Compose Content</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={'services:\n  app:\n    image: myapp:latest\n    ports:\n      - "8080:8080"'}
              rows={16}
              data-qa="dockerfile-content-textarea"
              className="w-full bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y font-mono text-xs"
            />
          </div>

          {/* Sort Order */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Sort Order</label>
            <input
              type="number"
              value={sortOrder}
              onChange={(e) => setSortOrder(Number(e.target.value))}
              data-qa="dockerfile-sort-order-input"
              className="w-24 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end px-6 py-4 border-t border-[var(--border-primary)] gap-3">
          <button
            onClick={onClose}
            data-qa="cancel-dockerfile-modal-footer-btn"
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!name.trim() || !slug.trim() || !version.trim() || saving}
            data-qa="save-dockerfile-btn"
            className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Dockerfile'}
          </button>
        </div>
      </div>
    </div>
  );
}
