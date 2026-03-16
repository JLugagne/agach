import { useEffect, useRef } from 'react';
import { wsClient } from '../lib/ws';
import type { WSEvent } from '../lib/types';

export function useWebSocket(callback: (event: WSEvent) => void) {
  const cbRef = useRef(callback);
  cbRef.current = callback;

  useEffect(() => {
    wsClient.connect();
    const unsub = wsClient.subscribe((e) => cbRef.current(e));
    return () => { unsub(); };
  }, []);
}
