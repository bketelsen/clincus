<script lang="ts">
  import { getRoots, loadWorkspaces } from '../stores/workspaces.svelte';
  import { api } from '../lib/api';

  let newPath = $state('');

  async function addRoot() {
    if (!newPath.trim()) return;
    await api.addWorkspace(newPath.trim());
    newPath = '';
    await loadWorkspaces();
  }

  async function removeRoot(path: string) {
    await api.removeWorkspace(path);
    await loadWorkspaces();
  }
</script>

<div class="settings">
  <h2>Settings</h2>
  <h3>Workspace Roots</h3>
  <ul>
    {#each getRoots() as root}
      <li>
        <span>{root}</span>
        <button onclick={() => removeRoot(root)}>Remove</button>
      </li>
    {/each}
  </ul>
  <form onsubmit={(e) => { e.preventDefault(); addRoot(); }}>
    <input type="text" bind:value={newPath} placeholder="/path/to/projects" />
    <button type="submit">Add</button>
  </form>
</div>

<style>
  .settings { padding: 16px; color: #ccc; max-width: 600px; }
  h2 { margin: 0 0 16px; }
  h3 { margin: 16px 0 8px; font-size: 14px; }
  ul { list-style: none; padding: 0; }
  li { display: flex; justify-content: space-between; align-items: center;
       padding: 8px; background: #1e1e30; margin-bottom: 4px; border-radius: 4px; }
  form { display: flex; gap: 8px; margin-top: 12px; }
  input { flex: 1; padding: 8px; background: #1e1e30; border: 1px solid #333;
          color: #ccc; border-radius: 4px; }
  button { padding: 8px 12px; background: #333; border: none; color: #ccc;
           border-radius: 4px; cursor: pointer; }
  button:hover { background: #444; }
</style>
