import type { WSMessage } from './types';

export function connectTerminal(
  containerId: string,
  onOutput: (data: string) => void,
  onExit: (code: number) => void,
  onError: (msg: string) => void,
): { send: (msg: WSMessage) => void; close: () => void } {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${proto}//${location.host}/ws/terminal/${containerId}`);

  ws.onmessage = (evt) => {
    const msg: WSMessage = JSON.parse(evt.data);
    switch (msg.type) {
      case 'output':
        if (msg.data) onOutput(msg.data);
        break;
      case 'exit':
        onExit(msg.code ?? 0);
        break;
      case 'error':
        onError(msg.message ?? 'unknown error');
        break;
    }
  };

  ws.onerror = () => onError('WebSocket connection error');

  return {
    send: (msg) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(msg));
      }
    },
    close: () => ws.close(),
  };
}

/**
 * Connect to the /ws/events WebSocket endpoint with automatic reconnection.
 * When the connection drops and is re-established, onReconnect is called so
 * the caller can re-fetch current state (AC4).
 *
 * BUG-02 fix: onReconnect now fires inside onopen (after connection is
 * established), not immediately after calling connect(). Old WebSocket is
 * explicitly closed with onclose nulled before creating a new connection.
 * Reconnection uses exponential backoff: 2s -> 4s -> 8s -> 30s cap.
 */
export function connectEvents(
  onEvent: (event: { type: string; id?: string; timestamp?: string }) => void,
  onReconnect?: () => void,
): { close: () => void } {
  let ws: WebSocket | null = null;
  let closed = false;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let delay = 2000;
  const MAX_DELAY = 30000;

  function connect() {
    if (closed) return;

    // Close old connection explicitly (null onclose before close to prevent re-trigger)
    if (ws) {
      ws.onclose = null;
      ws.close();
      ws = null;
    }

    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${proto}//${location.host}/ws/events`);

    ws.onopen = () => {
      delay = 2000; // reset on success
      if (onReconnect) onReconnect(); // fires AFTER connection established
    };

    ws.onmessage = (evt) => onEvent(JSON.parse(evt.data));

    ws.onclose = () => {
      if (closed) return;
      reconnectTimer = setTimeout(() => {
        connect();
      }, delay);
      delay = Math.min(delay * 2, MAX_DELAY);
    };

    ws.onerror = () => {
      // onerror is always followed by onclose -- reconnect handled there.
    };
  }

  connect();

  return {
    close: () => {
      closed = true;
      if (reconnectTimer !== null) clearTimeout(reconnectTimer);
      if (ws) {
        ws.onclose = null;
        ws.close();
      }
    },
  };
}
