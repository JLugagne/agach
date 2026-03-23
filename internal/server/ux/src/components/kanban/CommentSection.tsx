import { useState, useEffect, useRef } from 'react';
import { Send, ImageIcon, Loader2 } from 'lucide-react';
import { listComments, createComment, uploadImage } from '../../lib/api';
import type { CommentResponse } from '../../lib/types';
import MarkdownContent from '../ui/MarkdownContent';

interface CommentSectionProps {
  projectId: string;
  taskId: string;
  refreshKey?: number;
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function authorColor(authorRole: string): string {
  let hash = 0;
  for (let i = 0; i < authorRole.length; i++) {
    hash = authorRole.charCodeAt(i) + ((hash << 5) - hash);
  }
  const colors = ['#7C3AED', '#F09060', '#6B8AFF', '#F06060', '#FFD060', '#A78BFA', '#F472B6'];
  return colors[Math.abs(hash) % colors.length];
}

export default function CommentSection({ projectId, taskId, refreshKey }: CommentSectionProps) {
  const [comments, setComments] = useState<CommentResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [content, setContent] = useState('');
  const [authorRole, setAuthorRole] = useState('human');
  const [submitting, setSubmitting] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState('');
  const [dragOver, setDragOver] = useState(false);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchComments = async () => {
    try {
      const data = await listComments(projectId, taskId);
      setComments(data || []);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchComments();
  }, [projectId, taskId, refreshKey]);

  const handleDragOver = (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => {
    setDragOver(false);
  };

  const handleDrop = async (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (!file || !file.type.startsWith('image/')) return;

    setUploading(true);
    setUploadError('');

    try {
      const { url } = await uploadImage(projectId, file);
      const markdown = `![](${url})`;

      const textarea = textareaRef.current;
      if (textarea) {
        const start = textarea.selectionStart ?? content.length;
        const end = textarea.selectionEnd ?? content.length;
        const before = content.slice(0, start);
        const after = content.slice(end);
        const separator = before.length > 0 && !before.endsWith('\n') ? '\n' : '';
        const newContent = before + separator + markdown + after;
        setContent(newContent);

        requestAnimationFrame(() => {
          textarea.focus();
          const newPos = start + separator.length + markdown.length;
          textarea.setSelectionRange(newPos, newPos);
        });
      } else {
        setContent((prev) => (prev ? prev + '\n' + markdown : markdown));
      }
    } catch {
      setUploadError('Image upload failed. Please try again.');
      setTimeout(() => setUploadError(''), 4000);
    } finally {
      setUploading(false);
    }
  };

  const handleSubmit = async () => {
    if (!content.trim()) return;
    setSubmitting(true);
    try {
      await createComment(projectId, taskId, {
        author_role: authorRole,
        author_name: authorRole === 'human' ? 'Human' : authorRole,
        content: content.trim(),
      });
      setContent('');
      await fetchComments();
    } catch {
      /* ignore */
    } finally {
      setSubmitting(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleImageButtonClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Reset the input so the same file can be selected again if needed
    e.target.value = '';

    setUploading(true);
    setUploadError('');

    try {
      const { url } = await uploadImage(projectId, file);
      const markdown = `![](${url})`;

      const textarea = textareaRef.current;
      if (textarea) {
        const start = textarea.selectionStart ?? content.length;
        const end = textarea.selectionEnd ?? content.length;
        const before = content.slice(0, start);
        const after = content.slice(end);
        const separator = before.length > 0 && !before.endsWith('\n') ? '\n' : '';
        const newContent = before + separator + markdown + after;
        setContent(newContent);

        // Restore focus and move cursor after inserted text
        requestAnimationFrame(() => {
          textarea.focus();
          const newPos = start + separator.length + markdown.length;
          textarea.setSelectionRange(newPos, newPos);
        });
      } else {
        setContent((prev) => (prev ? prev + '\n' + markdown : markdown));
      }
    } catch {
      setUploadError('Image upload failed. Please try again.');
      setTimeout(() => setUploadError(''), 4000);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">
        Comments {comments.length > 0 && `(${comments.length})`}
      </h3>

      {loading ? (
        <p className="text-[var(--text-dim)] text-sm font-['Inter']">Loading comments...</p>
      ) : comments.length === 0 ? (
        <p className="text-[var(--text-muted)] text-sm font-['Inter']">No comments yet.</p>
      ) : (
        <div className="flex flex-col gap-3 max-h-[300px] overflow-y-auto pr-1">
          {comments.map((c) => (
            <div key={c.id} className="flex gap-3">
              <div
                className="w-7 h-7 rounded-full flex items-center justify-center flex-shrink-0 text-[10px] font-['JetBrains_Mono'] font-bold text-[var(--primary-text)]"
                style={{ backgroundColor: authorColor(c.author_role) }}
              >
                {(c.author_name || c.author_role).charAt(0).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-0.5">
                  <span className="text-[var(--text-primary)] text-xs font-['Inter'] font-medium">
                    {c.author_name || c.author_role}
                  </span>
                  <span className="text-[var(--text-dim)] text-[10px] font-['JetBrains_Mono']">
                    {c.author_type}
                  </span>
                  <span className="text-[var(--text-muted)] text-[10px] font-['Inter']">
                    {timeAgo(c.created_at)}
                  </span>
                </div>
                <MarkdownContent content={c.content} />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Compose area */}
      <div className="border-t border-[var(--border-primary)] pt-4 flex flex-col gap-2">
        <div className="flex items-center gap-2 mb-1">
          <label className="text-[var(--text-muted)] text-xs font-['Inter']">Post as:</label>
          <select
            data-qa="comment-author-select"
            value={authorRole}
            onChange={(e) => setAuthorRole(e.target.value)}
            className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded px-2 py-0.5 text-[var(--text-primary)] text-xs font-['Inter'] focus:outline-none focus:border-[var(--primary)]"
          >
            <option value="human">Human</option>
          </select>
        </div>
        <div className="flex gap-2">
          <textarea
            data-qa="comment-content-input"
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            placeholder="Write a comment..."
            rows={2}
            className={`flex-1 bg-[var(--bg-secondary)] border rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none transition-colors ${dragOver ? 'border-[var(--primary)]' : 'border-[var(--border-primary)] focus:border-[var(--primary)]'}`}
          />
          <button
            data-qa="comment-submit-btn"
            onClick={handleSubmit}
            disabled={submitting || !content.trim()}
            className="self-end px-3 py-2 bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            <Send size={14} className="text-[var(--primary-text)]" />
          </button>
        </div>

        {/* Image upload toolbar */}
        <div className="flex items-center gap-2">
          <button
            data-qa="comment-upload-image-btn"
            type="button"
            onClick={handleImageButtonClick}
            disabled={uploading}
            title="Upload image"
            className="flex items-center gap-1.5 px-2 py-1 text-[var(--text-muted)] hover:text-[var(--primary)] disabled:opacity-40 disabled:cursor-not-allowed rounded transition-colors text-xs font-['Inter']"
          >
            {uploading ? (
              <Loader2 size={14} className="animate-spin" />
            ) : (
              <ImageIcon size={14} />
            )}
            {uploading ? 'Uploading...' : 'Image'}
          </button>

          {uploadError && (
            <span className="text-[var(--status-blocked)] text-[10px] font-['Inter']">{uploadError}</span>
          )}
        </div>

        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={handleFileChange}
        />

        <p className="text-[var(--text-muted)] text-[10px] font-['Inter']">Ctrl+Enter to send</p>
      </div>
    </div>
  );
}
