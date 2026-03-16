import type { TaskWithDetailsResponse } from '../../lib/types';
import MarkWontDoModal from '../modals/MarkWontDoModal';
import UnblockTaskModal from '../modals/UnblockTaskModal';
import ApproveWontDoModal from '../modals/ApproveWontDoModal';
import CommentWontDoModal from '../modals/CommentWontDoModal';
import BlockTaskModal from '../modals/BlockTaskModal';
import DeleteTaskModal from '../modals/DeleteTaskModal';
import CompleteTaskModal from '../modals/CompleteTaskModal';

interface TaskActionsProps {
  projectId: string;
  task: TaskWithDetailsResponse | null;
  action: string | null; // 'block', 'unblock', 'wontdo', 'approve_wontdo', 'delete', 'comment_wontdo', 'complete'
  onClose: () => void;
  onSuccess: () => void;
}

export default function TaskActions({ projectId, task, action, onClose, onSuccess }: TaskActionsProps) {
  if (!task || !action) return null;

  switch (action) {
    case 'wontdo':
      return (
        <MarkWontDoModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    case 'unblock':
      return (
        <UnblockTaskModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    case 'approve_wontdo':
      return (
        <ApproveWontDoModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    case 'comment_wontdo':
      return (
        <CommentWontDoModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    case 'block':
      return (
        <BlockTaskModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    case 'delete':
      return (
        <DeleteTaskModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    case 'complete':
      return (
        <CompleteTaskModal
          task={task}
          projectId={projectId}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      );

    default:
      return null;
  }
}
