import { useState, useEffect, useRef, useCallback } from 'react';
import { wsClient } from '../lib/ws';
import type { WSEvent, ChatMessage, ChatStats } from '../lib/types';

const EMPTY_STATS: ChatStats = {
  messageCount: 0,
  inputTokens: 0,
  outputTokens: 0,
  cacheReadTokens: 0,
  totalCost: 0,
  durationSeconds: 0,
  model: '--',
};

interface UseChatOptions {
  projectId: string | undefined;
  featureId: string | undefined;
  sessionId: string | undefined;
  nodeId?: string;
  onSessionStarted?: (sessionId: string) => void;
  onSessionEnded?: () => void;
  onError?: (error: string) => void;
}

interface UseChatReturn {
  messages: ChatMessage[];
  stats: ChatStats;
  isConnected: boolean;
  isThinking: boolean;
  sendMessage: (content: string) => void;
  endSession: () => void;
  refreshActivity: () => void;
}

/**
 * Parse assistant message content from Claude's streaming-json format.
 * Looks for .text or .content fields in the data payload.
 */
function parseAssistantContent(data: unknown): string {
  if (!data || typeof data !== 'object') return '';
  const d = data as Record<string, unknown>;
  if (typeof d.text === 'string') return d.text;
  if (typeof d.content === 'string') return d.content;
  // Nested content block (e.g. {content: {text: "..."}})
  if (d.content && typeof d.content === 'object') {
    const inner = d.content as Record<string, unknown>;
    if (typeof inner.text === 'string') return inner.text;
  }
  return JSON.stringify(data);
}

export function useChat(options: UseChatOptions): UseChatReturn {
  const { projectId, featureId, sessionId, nodeId, onSessionStarted, onSessionEnded, onError } = options;

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [stats, setStats] = useState<ChatStats>(EMPTY_STATS);
  const [isConnected, setIsConnected] = useState(false);
  const [isThinking, setIsThinking] = useState(false);
  const msgIdRef = useRef(0);

  // Keep callbacks in refs to avoid re-subscribing on every render
  const onSessionStartedRef = useRef(onSessionStarted);
  onSessionStartedRef.current = onSessionStarted;
  const onSessionEndedRef = useRef(onSessionEnded);
  onSessionEndedRef.current = onSessionEnded;
  const onErrorRef = useRef(onError);
  onErrorRef.current = onError;
  const sessionIdRef = useRef(sessionId);
  sessionIdRef.current = sessionId;

  const nextId = useCallback(() => {
    msgIdRef.current += 1;
    return `msg-${msgIdRef.current}`;
  }, []);

  // Subscribe to WebSocket events
  useEffect(() => {
    wsClient.connect();
    setIsConnected(true);

    const unsub = wsClient.subscribe((event: WSEvent) => {
      switch (event.type) {
        case 'chat.start': {
          const d = event.data as Record<string, unknown> | undefined;
          if (d && typeof d.session_id === 'string') {
            onSessionStartedRef.current?.(d.session_id);
          }
          break;
        }
        case 'chat.message': {
          const d = event.data as Record<string, unknown> | undefined;
          if (sessionIdRef.current && d?.session_id !== sessionIdRef.current) break;
          if (d?.is_final) { setIsThinking(false); break; }
          const msgType = (d?.type as string) ?? 'assistant';
          const content = parseAssistantContent(event.data);
          if (content) {
            const msg: ChatMessage = {
              id: nextId(),
              type: msgType as ChatMessage['type'],
              content,
              timestamp: new Date().toISOString(),
              raw: event.data,
            };
            setMessages((prev) => [...prev, msg]);
          }
          break;
        }
        case 'chat.stats': {
          const d = event.data as Record<string, unknown> | undefined;
          if (d && (!sessionIdRef.current || d.session_id === sessionIdRef.current)) {
            setStats((prev) => ({
              messageCount: asNumber(d.message_count ?? d.messageCount) ?? prev.messageCount,
              inputTokens: asNumber(d.input_tokens ?? d.inputTokens) ?? prev.inputTokens,
              outputTokens: asNumber(d.output_tokens ?? d.outputTokens) ?? prev.outputTokens,
              cacheReadTokens: asNumber(d.cache_read_tokens ?? d.cacheReadTokens) ?? prev.cacheReadTokens,
              totalCost: asNumber(d.total_cost ?? d.totalCost) ?? prev.totalCost,
              durationSeconds: asNumber(d.duration_seconds ?? d.durationSeconds) ?? prev.durationSeconds,
              model: (typeof d.model === 'string' ? d.model : undefined) ?? prev.model,
            }));
          }
          break;
        }
        case 'chat.end': {
          onSessionEndedRef.current?.();
          break;
        }
        case 'chat.ttl_warning': {
          const d = event.data as Record<string, unknown> | undefined;
          const text = typeof d?.message === 'string' ? d.message : 'Session will expire soon due to inactivity.';
          setMessages((prev) => [
            ...prev,
            {
              id: nextId(),
              type: 'system',
              content: text,
              timestamp: new Date().toISOString(),
            },
          ]);
          break;
        }
        case 'chat.error': {
          const d = event.data as Record<string, unknown> | undefined;
          const errMsg = typeof d?.message === 'string' ? d.message : 'An error occurred.';
          onErrorRef.current?.(errMsg);
          setMessages((prev) => [
            ...prev,
            {
              id: nextId(),
              type: 'system',
              content: `Error: ${errMsg}`,
              timestamp: new Date().toISOString(),
            },
          ]);
          break;
        }
      }
    });

    return () => {
      unsub();
      setIsConnected(false);
    };
  }, [nextId]);

  const sendMessage = useCallback(
    (content: string) => {
      if (!projectId || !featureId) return;

      // Optimistic update: add user message immediately
      const msg: ChatMessage = {
        id: nextId(),
        type: 'user',
        content,
        timestamp: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, msg]);
      setStats((prev) => ({ ...prev, messageCount: prev.messageCount + 1 }));
      setIsThinking(true);

      wsClient.send({
        type: 'chat.user_message',
        data: {
          project_id: projectId,
          feature_id: featureId,
          session_id: sessionId,
          node_id: nodeId,
          content,
        },
      });
    },
    [projectId, featureId, sessionId, nodeId, nextId],
  );

  const endSession = useCallback(() => {
    if (!projectId || !featureId) return;
    wsClient.send({
      type: 'chat.end',
      data: {
        project_id: projectId,
        feature_id: featureId,
        session_id: sessionId,
      },
    });
  }, [projectId, featureId, sessionId]);

  const refreshActivity = useCallback(() => {
    if (!projectId || !featureId) return;
    wsClient.send({
      type: 'chat.ping',
      data: {
        project_id: projectId,
        feature_id: featureId,
        session_id: sessionId,
      },
    });
  }, [projectId, featureId, sessionId]);

  return { messages, stats, isConnected, isThinking, sendMessage, endSession, refreshActivity };
}

function asNumber(v: unknown): number | undefined {
  if (typeof v === 'number') return v;
  return undefined;
}
