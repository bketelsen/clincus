<script lang="ts">
  import type { Workspace } from '../lib/types';
  import { api } from '../lib/api';
  import { onMount } from 'svelte';

  let { workspace, onclose }: { workspace: Workspace; onclose: () => void } = $props();

  let tools = $state<string[]>([]);
  let selectedTool = $state('claude');

  onMount(async () => {
    tools = await api.getTools();
    if (tools.length > 0) selectedTool = tools[0];
  });

  async function launch() {
    const session = await api.createSession(workspace.path, selectedTool);
    location.hash = `#/terminal/${session.id}`;
    onclose();
  }
</script>

<div class="overlay" onclick={onclose} role="dialog">
  <div class="dialog" onclick={(e) => e.stopPropagation()}>
    <h3>Launch Session</h3>
    <p class="workspace">{workspace.path}</p>
    <label>
      Tool:
      <select bind:value={selectedTool}>
        {#each tools as t}
          <option value={t}>{t}</option>
        {/each}
      </select>
    </label>
    <div class="actions">
      <button onclick={onclose}>Cancel</button>
      <button class="primary" onclick={launch}>Launch</button>
    </div>
  </div>
</div>

<style>
  .overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6);
             display: flex; align-items: center; justify-content: center; z-index: 100; }
  .dialog { background: #1e1e30; padding: 24px; border-radius: 8px; min-width: 360px;
            color: #ccc; border: 1px solid #333; }
  h3 { margin: 0 0 12px; }
  .workspace { font-size: 12px; color: #888; margin: 0 0 16px; }
  label { display: block; margin-bottom: 16px; }
  select { display: block; width: 100%; margin-top: 4px; padding: 8px;
           background: #252540; border: 1px solid #333; color: #ccc; border-radius: 4px; }
  .actions { display: flex; gap: 8px; justify-content: flex-end; }
  button { padding: 8px 12px; background: #333; border: none; color: #ccc;
           border-radius: 4px; cursor: pointer; }
  button:hover { background: #444; }
  .primary { background: #4a5568; }
</style>
