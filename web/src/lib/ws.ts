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

export function connectEvents(
  onEvent: (event: { type: string; id?: string; workspace?: string }) => void,
): { close: () => void } {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${proto}//${location.host}/ws/events`);
  ws.onmessage = (evt) => onEvent(JSON.parse(evt.data));
  return { close: () => ws.close() };
}
