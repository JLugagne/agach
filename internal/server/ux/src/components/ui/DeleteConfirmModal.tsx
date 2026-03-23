import { Trash2, X } from 'lucide-react';

interface DeleteConfirmModalProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
  loading?: boolean;
}

export default function DeleteConfirmModal({
  open,
  title,
  description,
  confirmLabel = 'Delete',
  onConfirm,
  onCancel,
  loading = false,
}: DeleteConfirmModalProps) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onCancel} />
      <div className="relative bg-[var(--bg-elevated)] border border-[var(--border-primary)] rounded-lg w-full max-w-[400px] p-6 text-center">
        <button
          onClick={onCancel}
          data-qa="delete-confirm-close-btn"
          className="absolute top-3 right-3 text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
        >
          <X size={18} />
        </button>

        <div className="mx-auto w-14 h-14 rounded-full bg-[var(--status-blocked-bg)] flex items-center justify-center mb-4">
          <Trash2 size={24} className="text-[var(--status-blocked)]" />
        </div>

        <h3 className="font-heading text-lg text-[var(--text-primary)] mb-2">{title}</h3>
        <p className="text-sm text-[var(--text-secondary)] mb-6">{description}</p>

        <div className="flex items-center justify-center gap-4">
          <button
            onClick={onCancel}
            data-qa="delete-confirm-cancel-btn"
            className="text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            data-qa="delete-confirm-submit-btn"
            className="px-4 py-2 bg-[var(--status-blocked)] text-white text-sm font-medium rounded-md hover:opacity-90 disabled:opacity-50 transition-colors"
          >
            {loading ? 'Deleting...' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
