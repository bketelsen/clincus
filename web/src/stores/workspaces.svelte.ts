import { api } from '../lib/api';
import type { Workspace } from '../lib/types';

const store = $state<{ workspaces: Workspace[]; roots: string[]; expandedRoots: string[] }>({
  workspaces: [],
  roots: [],
  expandedRoots: [],
});

export function getWorkspaces(): Workspace[] {
  return store.workspaces;
}

export function getRoots(): string[] {
  return store.roots;
}

/** Return expanded root paths in config order. Includes roots with no projects. */
export function getExpandedRoots(): string[] {
  return store.expandedRoots;
}

/** Return workspaces for a given expanded root path, sorted alphabetically by name. */
export function getWorkspacesForRoot(root: string): Workspace[] {
  return store.workspaces
    .filter((ws) => ws.root === root)
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));
}

export async function loadWorkspaces() {
  const data = await api.listWorkspaces();
  store.workspaces = data.workspaces;
  store.roots = data.roots;
  store.expandedRoots = data.expanded_roots;
}
