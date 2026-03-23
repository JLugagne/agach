import { useState, useEffect, useCallback, useRef } from 'react';
import { X, Plus, XCircle, Paperclip, Loader2 } from 'lucide-react';
import { createTask, listProjectAgents } from '../../lib/api';
import type { AgentResponse, FeatureResponse } from '../../lib/types';
import { useImageUpload } from '../../hooks/useImageUpload';

interface NewTaskModalProps {
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
  defaultRole?: string;
  features?: FeatureResponse[];
}

export default function NewTaskModal({ projectId, onClose, onSuccess, defaultRole, features }: NewTaskModalProps) {
  const [title, setTitle] = useState('');
  const [summary, setSummary] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState('medium');
  const [assignedRole, setAssignedRole] = useState(defaultRole ?? '');
  const [selectedFeatureId, setSelectedFeatureId] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  const [contextFiles, setContextFiles] = useState<string[]>([]);
  const [fileInput, setFileInput] = useState('');
  const [roles, setRoles] = useState<AgentResponse[]>([]);
  const [addToBacklog, setAddToBacklog] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [descDragOver, setDescDragOver] = useState(false);
  const descTextareaRef = useRef<HTMLTextAreaElement>(null);
  const descFileInputRef = useRef<HTMLInputElement>(null);
  const { upload: uploadImg, uploading: imgUploading, error: imgError } = useImageUpload(projectId);

  useEffect(() => {
    listProjectAgents(projectId)
      .then((data) => setRoles(data || []))
      .catch(() => {});
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    },
    [onClose],
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  const addTag = () => {
    const t = tagInput.trim();
    if (t && !tags.includes(t)) {
      setTags([...tags, t]);
    }
    setTagInput('');
  };

  const removeTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag));
  };

  const addFile = () => {
    const f = fileInput.trim();
    if (f && !contextFiles.includes(f)) {
      setContextFiles([...contextFiles, f]);
    }
    setFileInput('');
  };

  const removeFile = (file: string) => {
    setContextFiles(contextFiles.filter((f) => f !== file));
  };

  const insertImageMarkdown = useCallback(
    async (file: File) => {
      const url = await uploadImg(file);
      if (!url) return;
      const markdown = `![](${url})`;
      const textarea = descTextareaRef.current;
      if (textarea) {
        const start = textarea.selectionStart ?? description.length;
        const end = textarea.selectionEnd ?? description.length;
        const newDesc = description.slice(0, start) + markdown + description.slice(end);
        setDescription(newDesc);
      } else {
        setDescription((prev) => prev + markdown);
      }
    },
    [uploadImg, description],
  );

  const handleDescDragOver = (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setDescDragOver(true);
  };

  const handleDescDragLeave = () => {
    setDescDragOver(false);
  };

  const handleDescDrop = async (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setDescDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) {
      await insertImageMarkdown(file);
    }
  };

  const handleDescFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      await insertImageMarkdown(file);
    }
    e.target.value = '';
  };

  const handleSubmit = async () => {
    if (!title.trim()) {
      setError('Title is required.');
      return;
    }
    if (!summary.trim()) {
      setError('Summary is required.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await createTask(projectId, {
        title: title.trim(),
        summary: summary.trim(),
        description: description.trim() || undefined,
        priority,
        assigned_role: assignedRole || undefined,
        tags: tags.length > 0 ? tags : undefined,
        context_files: contextFiles.length > 0 ? contextFiles : undefined,
        start_in_backlog: addToBacklog,
        feature_id: selectedFeatureId || undefined,
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create task');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      data-qa="new-task-modal"
      className="fixed inset-0 z-50 flex items-center justify-center"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="absolute inset-0 bg-[#00000060]" />
      <div className="relative w-[520px] max-h-[90vh] rounded-xl bg-[var(--bg-secondary)] border border-[#2A2A2A] shadow-2xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#2A2A2A] flex-shrink-0">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            New Task
          </h2>
          <button
            data-qa="new-task-close-btn"
            onClick={onClose}
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4 overflow-y-auto flex-1">
          {/* Title */}
          <div>
            <label className="block text-[#E0E0E0] text-sm font-['Inter'] font-medium mb-1.5">
              Title <span className="text-[#F06060]">*</span>
            </label>
            <input
              data-qa="new-task-title-input"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Task title"
              className="w-full bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
          </div>

          {/* Summary */}
          <div>
            <label className="block text-[#E0E0E0] text-sm font-['Inter'] font-medium mb-1.5">
              Summary <span className="text-[#F06060]">*</span>
            </label>
            <textarea
              data-qa="new-task-summary-input"
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
              placeholder="Brief description of what needs to be done"
              rows={2}
              className="w-full bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
          </div>

          {/* Description */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-[#E0E0E0] text-sm font-['Inter'] font-medium">
                Description
              </label>
              <div className="flex items-center gap-1.5">
                {imgUploading && (
                  <Loader2 size={13} className="text-[var(--primary)] animate-spin" />
                )}
                <button
                  data-qa="new-task-attach-image-btn"
                  type="button"
                  onClick={() => descFileInputRef.current?.click()}
                  title="Attach image"
                  className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
                >
                  <Paperclip size={14} />
                </button>
                <input
                  ref={descFileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={handleDescFileSelect}
                />
              </div>
            </div>
            <textarea
              data-qa="new-task-description-input"
              ref={descTextareaRef}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              onDragOver={handleDescDragOver}
              onDragLeave={handleDescDragLeave}
              onDrop={handleDescDrop}
              placeholder="Detailed description (optional)"
              rows={4}
              className={`w-full bg-[#0D0D0D] border rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none transition-colors ${descDragOver ? 'border-[var(--primary)]' : 'border-[var(--border-primary)] focus:border-[var(--primary)]'}`}
            />
            {imgError && (
              <p className="text-[#F06060] text-xs font-['Inter'] mt-1">{imgError}</p>
            )}
          </div>

          {/* Priority & Assigned Role row */}
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="block text-[#E0E0E0] text-sm font-['Inter'] font-medium mb-1.5">
                Priority
              </label>
              <select
                data-qa="new-task-priority-select"
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
                className="w-full bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] focus:outline-none focus:border-[var(--primary)] transition-colors"
              >
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>
            </div>
            <div className="flex-1">
              <label className="block text-[#E0E0E0] text-sm font-['Inter'] font-medium mb-1.5">
                Assigned Role
              </label>
              <select
                data-qa="new-task-role-select"
                value={assignedRole}
                onChange={(e) => setAssignedRole(e.target.value)}
                className="w-full bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] focus:outline-none focus:border-[var(--primary)] transition-colors"
              >
                <option value="">Unassigned</option>
                {roles.map((r) => (
                  <option key={r.id} value={r.slug}>
                    {r.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Add to backlog */}
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <input
              data-qa="new-task-backlog-checkbox"
              type="checkbox"
              checked={addToBacklog}
              onChange={(e) => setAddToBacklog(e.target.checked)}
              className="w-4 h-4 rounded border border-[var(--border-primary)] bg-[#0D0D0D] accent-[var(--primary)]"
            />
            <span className="text-[#E0E0E0] text-sm font-['Inter']">Add to backlog</span>
          </label>

          {/* Feature selector */}
          {features && features.length > 0 && (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">
                Feature
              </label>
              <select
                data-qa="new-task-feature-select"
                value={selectedFeatureId}
                onChange={(e) => setSelectedFeatureId(e.target.value)}
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
              >
                <option value="">None</option>
                {features.map((feat) => (
                  <option key={feat.id} value={feat.id}>
                    {feat.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Tags */}
          <div>
            <label className="block text-[#E0E0E0] text-sm font-['Inter'] font-medium mb-1.5">
              Tags
            </label>
            <div className="flex gap-2 mb-2 flex-wrap">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="flex items-center gap-1 px-2 py-0.5 rounded bg-[#1E1E1E] text-[var(--text-muted)] text-xs font-['JetBrains_Mono']"
                >
                  {tag}
                  <button data-qa="new-task-remove-tag-btn" onClick={() => removeTag(tag)} className="text-[var(--text-dim)] hover:text-[#F06060]">
                    <XCircle size={12} />
                  </button>
                </span>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                data-qa="new-task-tag-input"
                type="text"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addTag();
                  }
                }}
                placeholder="Add tag..."
                className="flex-1 bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-1.5 text-[var(--text-primary)] text-xs font-['Inter'] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)] transition-colors"
              />
              <button
                data-qa="new-task-add-tag-btn"
                onClick={addTag}
                disabled={!tagInput.trim()}
                className="px-2 py-1.5 bg-[#1E1E1E] hover:bg-[var(--border-primary)] disabled:opacity-30 rounded-md transition-colors"
              >
                <Plus size={14} className="text-[var(--text-muted)]" />
              </button>
            </div>
          </div>

          {/* Context Files */}
          <div>
            <label className="block text-[#E0E0E0] text-sm font-['Inter'] font-medium mb-1.5">
              Context Files
            </label>
            <div className="flex flex-col gap-1 mb-2">
              {contextFiles.map((f) => (
                <div
                  key={f}
                  className="flex items-center gap-2 px-2 py-1 rounded bg-[#0D0D0D] border border-[var(--border-primary)]"
                >
                  <span className="text-[var(--text-dim)] text-xs font-['JetBrains_Mono'] truncate flex-1">
                    {f}
                  </span>
                  <button data-qa="new-task-remove-file-btn" onClick={() => removeFile(f)} className="text-[var(--text-dim)] hover:text-[#F06060]">
                    <XCircle size={12} />
                  </button>
                </div>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                data-qa="new-task-file-input"
                type="text"
                value={fileInput}
                onChange={(e) => setFileInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addFile();
                  }
                }}
                placeholder="path/to/file.go"
                className="flex-1 bg-[#0D0D0D] border border-[var(--border-primary)] rounded-md px-3 py-1.5 text-[var(--text-primary)] text-xs font-['JetBrains_Mono'] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)] transition-colors"
              />
              <button
                data-qa="new-task-add-file-btn"
                onClick={addFile}
                disabled={!fileInput.trim()}
                className="px-2 py-1.5 bg-[#1E1E1E] hover:bg-[var(--border-primary)] disabled:opacity-30 rounded-md transition-colors"
              >
                <Plus size={14} className="text-[var(--text-muted)]" />
              </button>
            </div>
          </div>

          {error && <p className="text-[#F06060] text-sm font-['Inter']">{error}</p>}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[#2A2A2A] flex-shrink-0">
          <button
            data-qa="new-task-cancel-btn"
            onClick={onClose}
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            data-qa="new-task-submit-btn"
            onClick={handleSubmit}
            disabled={loading || !title.trim() || !summary.trim()}
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[#00B886] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Creating...' : 'Create Task'}
          </button>
        </div>
      </div>
    </div>
  );
}
