import { api } from '../lib/api';
import type { Workspace } from '../lib/types';

const store = $state<{ workspaces: Workspace[]; roots: string[] }>({ workspaces: [], roots: [] });

export function getWorkspaces(): Workspace[] {
  return store.workspaces;
}

export function getRoots(): string[] {
  return store.roots;
}

export async function loadWorkspaces() {
  const data = await api.listWorkspaces();
  store.workspaces = data.workspaces;
  store.roots = data.roots;
}
