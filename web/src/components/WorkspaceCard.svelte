<script lang="ts">
  import type { Workspace } from '../lib/types';
  import { api } from '../lib/api';

  let { workspace }: { workspace: Workspace } = $props();
  let launching = $state(false);
  let error = $state('');

  async function launch() {
    launching = true;
    error = '';
    try {
      const session = await api.createSession(workspace.path, 'claude');
      location.hash = `#/terminal/${session.id}`;
    } catch (e: any) {
      error = e.message || 'Launch failed';
    } finally {
      launching = false;
    }
  }
</script>

<button class="card" onclick={launch} disabled={launching}>
  <div class="name">{workspace.name}</div>
  {#if launching}
    <span class="status">Launching...</span>
  {:else if error}
    <span class="error">{error}</span>
  {:else}
    {#if workspace.has_config}
      <span class="badge">coi.toml</span>
    {/if}
    {#if workspace.active_sessions > 0}
      <span class="active">{workspace.active_sessions} active</span>
    {/if}
  {/if}
</button>

<style>
  .card { display: block; width: 100%; text-align: left; padding: 16px;
          background: #1e1e30; border: 1px solid #333; border-radius: 8px;
          color: #ccc; cursor: pointer; }
  .card:hover { border-color: #555; background: #252540; }
  .card:disabled { opacity: 0.7; cursor: wait; }
  .name { font-weight: 600; font-size: 15px; }
  .badge { font-size: 10px; color: #888; background: #2a2a3e; padding: 2px 6px;
           border-radius: 4px; margin-top: 4px; display: inline-block; }
  .active { font-size: 11px; color: #4caf50; margin-left: 8px; }
  .status { font-size: 11px; color: #f0c040; margin-top: 4px; display: block; }
  .error { font-size: 11px; color: #f04040; margin-top: 4px; display: block; word-break: break-word; }
</style>
