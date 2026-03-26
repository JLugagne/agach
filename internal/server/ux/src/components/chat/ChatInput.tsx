import { useState, useRef, useCallback } from 'react';
import { Send, Paperclip, Smile } from 'lucide-react';

interface ChatInputProps {
  onSend: (message: string) => void;
  disabled?: boolean;
}

export default function ChatInput({ onSend, disabled }: ChatInputProps) {
  const [text, setText] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSend = useCallback(() => {
    const trimmed = text.trim();
    if (!trimmed || disabled) return;
    onSend(trimmed);
    setText('');
    if (textareaRef.current) {
      textareaRef.current.style.height = '40px';
    }
  }, [text, disabled, onSend]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend],
  );

  const handleInput = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = '40px';
    el.style.height = Math.min(el.scrollHeight, 120) + 'px';
  }, []);

  const canSend = text.trim().length > 0 && !disabled;

  return (
    <div
      data-qa="chat-input"
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        gap: '8px',
        padding: '12px 16px',
        borderTop: '1px solid var(--border-subtle)',
        backgroundColor: 'var(--bg-primary)',
        fontFamily: 'Inter, sans-serif',
      }}
    >
      {/* Attach button (placeholder, not functional) */}
      <button
        data-qa="chat-attach-btn"
        disabled={disabled}
        style={{
          background: 'none',
          border: 'none',
          padding: '8px',
          cursor: disabled ? 'not-allowed' : 'pointer',
          color: disabled ? 'var(--text-disabled)' : 'var(--text-muted)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderRadius: '50%',
          transition: 'color 0.15s',
          flexShrink: 0,
        }}
        title="Attach file"
      >
        <Paperclip size={18} />
      </button>

      {/* Text input pill */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          alignItems: 'flex-end',
          borderRadius: '20px',
          border: '1px solid var(--border-subtle)',
          backgroundColor: 'var(--bg-secondary)',
          padding: '4px 12px',
          transition: 'border-color 0.15s',
        }}
      >
        <textarea
          ref={textareaRef}
          data-qa="chat-input-field"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          onInput={handleInput}
          disabled={disabled}
          placeholder={disabled ? 'No active session' : 'Type a message...'}
          rows={1}
          style={{
            flex: 1,
            border: 'none',
            outline: 'none',
            background: 'none',
            resize: 'none',
            fontSize: '14px',
            lineHeight: '20px',
            height: '40px',
            maxHeight: '120px',
            padding: '10px 0',
            color: 'var(--text-primary)',
            fontFamily: 'Inter, sans-serif',
          }}
        />
      </div>

      {/* Emoji button (placeholder, not functional) */}
      <button
        data-qa="chat-emoji-btn"
        disabled={disabled}
        style={{
          background: 'none',
          border: 'none',
          padding: '8px',
          cursor: disabled ? 'not-allowed' : 'pointer',
          color: disabled ? 'var(--text-disabled)' : 'var(--text-muted)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderRadius: '50%',
          transition: 'color 0.15s',
          flexShrink: 0,
        }}
        title="Emoji"
      >
        <Smile size={18} />
      </button>

      {/* Send button */}
      <button
        data-qa="chat-send-btn"
        onClick={handleSend}
        disabled={!canSend}
        style={{
          background: canSend ? 'var(--primary)' : 'var(--bg-tertiary)',
          border: 'none',
          padding: '8px',
          cursor: canSend ? 'pointer' : 'not-allowed',
          color: canSend ? 'var(--primary-text)' : 'var(--text-disabled)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderRadius: '50%',
          width: '36px',
          height: '36px',
          transition: 'background-color 0.15s, color 0.15s',
          flexShrink: 0,
        }}
        title="Send message"
      >
        <Send size={16} />
      </button>
    </div>
  );
}
