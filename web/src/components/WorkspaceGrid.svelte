<script lang="ts">
  import WorkspaceCard from './WorkspaceCard.svelte';
  import { getWorkspaces } from '../stores/workspaces.svelte';
</script>

<div class="grid">
  {#each getWorkspaces() as ws (ws.path)}
    <WorkspaceCard workspace={ws} />
  {:else}
    <p class="empty">No workspaces found. Add workspace roots in <a href="#/settings">Settings</a>.</p>
  {/each}
</div>

<style>
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
          gap: 12px; padding: 16px; }
  .empty { color: #666; grid-column: 1 / -1; }
  a { color: #88f; }
</style>
