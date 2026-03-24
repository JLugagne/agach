import type { WSEvent } from './types';

type WSCallback = (event: WSEvent) => void;

class WebSocketClient {
  private ws: WebSocket | null = null;
  private listeners: Set<WSCallback> = new Set();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private connected = false;
  private consecutiveFailures = 0;

  connect() {
    if (this.connected || this.ws) return;
    const token = localStorage.getItem('agach_access_token');
    if (!token) return;
    // Stop trying after repeated failures (likely invalid token / 401)
    if (this.consecutiveFailures >= 3) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const query = `?token=${encodeURIComponent(token)}`;
    this.ws = new WebSocket(`${protocol}//${window.location.host}/ws${query}`);

    this.ws.onopen = () => {
      this.connected = true;
      this.consecutiveFailures = 0;
    };

    this.ws.onmessage = (msg) => {
      try {
        const event: WSEvent = JSON.parse(msg.data);
        this.listeners.forEach((cb) => cb(event));
      } catch { /* ignore parse errors */ }
    };

    this.ws.onclose = () => {
      const wasConnected = this.connected;
      this.connected = false;
      this.ws = null;
      if (!wasConnected) {
        this.consecutiveFailures++;
        if (this.consecutiveFailures >= 3) return;
      }
      const delay = wasConnected ? 2000 : 2000 * Math.pow(2, this.consecutiveFailures);
      this.reconnectTimer = setTimeout(() => this.connect(), delay);
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  /** Reset failure count (call after successful login) */
  reset() {
    this.consecutiveFailures = 0;
  }

  subscribe(cb: WSCallback): () => void {
    this.listeners.add(cb);
    return () => { this.listeners.delete(cb); };
  }

  /** Send a message to the server via WebSocket */
  send(msg: unknown) {
    if (this.ws && this.connected) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.ws = null;
    this.connected = false;
  }
}

export const wsClient = new WebSocketClient();
