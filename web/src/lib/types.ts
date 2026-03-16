export interface Session {
  id: string;
  workspace: string;
  tool: string;
  slot: number;
  status: string;
  persistent: boolean;
}

export interface Workspace {
  path: string;
  name: string;
  has_config: boolean;
  active_sessions: number;
}

export interface HistoryEntry {
  id: string;
  workspace: string;
  tool: string;
  started: string;
  stopped: string;
  persistent: boolean;
  exit_code: number;
}

export interface WSMessage {
  type: string;
  data?: string;
  cols?: number;
  rows?: number;
  code?: number;
  message?: string;
}
