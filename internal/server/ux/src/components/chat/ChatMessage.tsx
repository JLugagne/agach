import type { ChatMessage as ChatMessageType } from '../../lib/types';
import MarkdownContent from '../ui/MarkdownContent';

interface ChatMessageProps {
  message: ChatMessageType;
}

function formatTime(timestamp: string): string {
  try {
    const d = new Date(timestamp);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
}

export default function ChatMessage({ message }: ChatMessageProps) {
  const isUser = message.type === 'user';

  if (isUser) {
    return (
      <div
        data-qa="chat-message-user"
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          gap: '4px',
          fontFamily: 'Inter, sans-serif',
        }}
      >
        <span
          style={{
            fontSize: '11px',
            color: 'var(--text-muted)',
            fontWeight: 500,
          }}
        >
          You
        </span>
        <div
          style={{
            maxWidth: '75%',
            padding: '10px 14px',
            borderRadius: '16px 16px 4px 16px',
            backgroundColor: 'var(--primary)',
            color: 'var(--primary-text)',
            fontSize: '14px',
            lineHeight: '1.5',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {message.content}
        </div>
        <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
          {formatTime(message.timestamp)}
        </span>
      </div>
    );
  }

  return (
    <div
      data-qa="chat-message-assistant"
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: '8px',
        fontFamily: 'Inter, sans-serif',
      }}
    >
      {/* Avatar */}
      <div
        style={{
          width: '32px',
          height: '32px',
          borderRadius: '50%',
          backgroundColor: 'var(--severity-success)',
          color: 'var(--primary-text)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '12px',
          fontWeight: 700,
          flexShrink: 0,
        }}
      >
        A1
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', minWidth: 0, flex: 1 }}>
        <span
          style={{
            fontSize: '11px',
            color: 'var(--text-muted)',
            fontWeight: 500,
          }}
        >
          Assistant
        </span>
        <div
          className="chat-assistant-bubble"
          style={{
            padding: '10px 14px',
            borderRadius: '4px 16px 16px 16px',
            backgroundColor: 'var(--bg-elevated)',
            color: 'var(--text-primary)',
            fontSize: '14px',
            lineHeight: '1.5',
            wordBreak: 'break-word',
            overflowX: 'auto',
          }}
        >
          <MarkdownContent content={message.content} />
        </div>
        <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
          {formatTime(message.timestamp)}
        </span>
      </div>
    </div>
  );
}
