import type { Session, Workspace, HistoryEntry, ClincusConfig } from './types';

const BASE = '';

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function del(path: string): Promise<void> {
  const res = await fetch(`${BASE}${path}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(await res.text());
}

export const api = {
  listSessions: () => get<Session[]>('/api/sessions'),
  createSession: (workspace: string, tool: string) =>
    post<Session>('/api/sessions', { workspace, tool }),
  stopSession: (id: string, force = false) =>
    del(`/api/sessions/${id}${force ? '?force=true' : ''}`),
  resumeSession: (id: string) =>
    post<{ id: string }>(`/api/sessions/${id}/resume`),
  sessionHistory: () => get<HistoryEntry[]>('/api/sessions/history'),
  listWorkspaces: () =>
    get<{ roots: string[]; workspaces: Workspace[] }>('/api/workspaces'),
  addWorkspace: (path: string) =>
    post('/api/workspaces', { path }),
  removeWorkspace: (path: string) =>
    del(`/api/workspaces?path=${encodeURIComponent(path)}`),
  getTools: () => get<string[]>('/api/tools'),
  getConfig: () => get<ClincusConfig>('/api/config'),
};
