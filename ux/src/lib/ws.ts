import type { WSEvent } from './types';

type WSCallback = (event: WSEvent) => void;

class WebSocketClient {
  private ws: WebSocket | null = null;
  private listeners: Set<WSCallback> = new Set();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private connected = false;

  connect() {
    if (this.connected || this.ws) return;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    this.ws.onopen = () => {
      this.connected = true;
    };

    this.ws.onmessage = (msg) => {
      try {
        const event: WSEvent = JSON.parse(msg.data);
        this.listeners.forEach((cb) => cb(event));
      } catch { /* ignore parse errors */ }
    };

    this.ws.onclose = () => {
      this.connected = false;
      this.ws = null;
      this.reconnectTimer = setTimeout(() => this.connect(), 2000);
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  subscribe(cb: WSCallback): () => void {
    this.listeners.add(cb);
    return () => { this.listeners.delete(cb); };
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.ws = null;
    this.connected = false;
  }
}

export const wsClient = new WebSocketClient();
