<script lang="ts">
  import { api } from '../lib/api';
  import type { FileEntry } from '../lib/types';
  import { onMount } from 'svelte';

  let { sessionId, onFileSelect }: {
    sessionId: string;
    onFileSelect: (path: string) => void;
  } = $props();

  interface TreeNode {
    name: string;
    path: string;
    type: string;
    size: number;
    children?: TreeNode[];
    expanded?: boolean;
    loading?: boolean;
  }

  let root = $state<TreeNode[]>([]);
  let error = $state('');

  async function loadDir(path: string): Promise<TreeNode[]> {
    const res = await api.listFiles(sessionId, path);
    return res.entries
      .sort((a, b) => {
        // Directories first, then alphabetical
        if (a.type === 'dir' && b.type !== 'dir') return -1;
        if (a.type !== 'dir' && b.type === 'dir') return 1;
        return a.name.localeCompare(b.name);
      })
      .map((e) => ({
        name: e.name,
        path: path === '/' ? e.name : `${path}/${e.name}`,
        type: e.type,
        size: e.size,
      }));
  }

  async function toggleDir(node: TreeNode) {
    if (node.expanded) {
      node.expanded = false;
      return;
    }
    node.loading = true;
    try {
      node.children = await loadDir(node.path);
      node.expanded = true;
    } catch {
      node.children = [];
    }
    node.loading = false;
  }

  function handleClick(node: TreeNode) {
    if (node.type === 'dir') {
      toggleDir(node);
    } else {
      onFileSelect(node.path);
    }
  }

  async function refresh() {
    error = '';
    try {
      root = await loadDir('/');
    } catch (e) {
      error = 'Failed to load file tree';
    }
  }

  onMount(() => { refresh(); });
</script>

<div class="file-tree">
  <div class="tree-header">
    <span class="tree-title">Files</span>
    <button class="refresh-btn" onclick={refresh} title="Refresh">&#x21bb;</button>
  </div>
  {#if error}
    <div class="tree-error">{error}</div>
  {:else}
    <div class="tree-content">
      {#each root as node}
        {@render treeNode(node, 0)}
      {/each}
    </div>
  {/if}
</div>

{#snippet treeNode(node: TreeNode, depth: number)}
  <button
    class="tree-item"
    class:dir={node.type === 'dir'}
    style="padding-left: {12 + depth * 16}px"
    onclick={() => handleClick(node)}
  >
    <span class="tree-icon">
      {#if node.type === 'dir'}
        {node.loading ? '...' : node.expanded ? '▼' : '▶'}
      {:else}
        &#x25A0;
      {/if}
    </span>
    <span class="tree-name">{node.name}</span>
  </button>
  {#if node.expanded && node.children}
    {#each node.children as child}
      {@render treeNode(child, depth + 1)}
    {/each}
  {/if}
{/snippet}

<style>
  .file-tree {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #1a1a2e;
    border-right: 1px solid #333;
    min-width: 200px;
    max-width: 300px;
    overflow-y: auto;
  }
  .tree-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-bottom: 1px solid #333;
  }
  .tree-title {
    color: #888;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .refresh-btn {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    font-size: 14px;
    padding: 2px 4px;
  }
  .refresh-btn:hover { color: #ccc; }
  .tree-content { padding: 4px 0; }
  .tree-error {
    padding: 12px;
    color: #e66;
    font-size: 12px;
  }
  .tree-item {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 3px 12px;
    background: none;
    border: none;
    color: #ccc;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 12px;
    cursor: pointer;
    text-align: left;
  }
  .tree-item:hover { background: #2a2a4a; }
  .tree-icon {
    font-size: 8px;
    width: 12px;
    text-align: center;
    color: #888;
  }
  .tree-item.dir .tree-name { color: #8be9fd; }
</style>
