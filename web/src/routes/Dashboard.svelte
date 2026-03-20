<script lang="ts">
  import WorkspaceGrid from '../components/WorkspaceGrid.svelte';
  import NewFolderDialog from '../components/NewFolderDialog.svelte';
  import { getWorkspacesForRoot, getExpandedRoots } from '../stores/workspaces.svelte';

  let dialogRoot = $state<string | null>(null);

  function basename(path: string): string {
    const parts = path.split('/');
    return parts[parts.length - 1] || path;
  }
</script>

<div class="dashboard">
  <h2>Projects</h2>

  {#each getExpandedRoots() as root (root)}
    <div class="root-container">
      <div class="root-header" title={root}>
        {basename(root)}
      </div>
      <div class="root-body">
        <div class="grid-with-new">
          <WorkspaceGrid workspaces={getWorkspacesForRoot(root)} />
          <button class="new-folder" onclick={() => dialogRoot = root}>
            <span class="plus">+</span>
            <span class="label">New Project</span>
          </button>
        </div>
      </div>
    </div>
  {:else}
    <p class="empty">No workspaces found. Add workspace roots in <a href="#/settings">Settings</a>.</p>
  {/each}
</div>

{#if dialogRoot}
  <NewFolderDialog root={dialogRoot} onclose={() => dialogRoot = null} />
{/if}

<style>
  .dashboard { padding: 16px; color: #ccc; }
  h2 { margin: 0 0 16px; font-size: 18px; }

  .root-container {
    background: #1a1a2e;
    border: 1px solid #2a2a40;
    border-radius: 8px;
    margin-bottom: 16px;
    overflow: hidden;
  }

  .root-header {
    padding: 10px 16px;
    font-size: 14px;
    font-weight: 600;
    color: #999;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 1px solid #2a2a40;
  }

  .root-body {
    padding: 16px;
  }

  .grid-with-new {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 12px;
  }

  .new-folder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 80px;
    background: transparent;
    border: 2px dashed #333;
    border-radius: 8px;
    color: #666;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }

  .new-folder:hover {
    border-color: #555;
    color: #999;
  }

  .new-folder .plus {
    font-size: 24px;
    line-height: 1;
  }

  .new-folder .label {
    font-size: 12px;
    margin-top: 4px;
  }

  .empty { color: #666; }
  a { color: #88f; }
</style>
