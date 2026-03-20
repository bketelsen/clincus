<script lang="ts">
  import { api } from '../lib/api';
  import { onMount } from 'svelte';

  let { root, onclose }: { root: string; onclose: () => void } = $props();

  let tools = $state<string[]>([]);
  let selectedTool = $state('claude');
  let folderName = $state('');
  let creating = $state(false);
  let error = $state('');
  let nameInput: HTMLInputElement;

  const VALID_NAME = /^[a-z0-9]+(-[a-z0-9]+)*$/;

  $effect(() => {
    // Auto-focus input when dialog opens
    if (nameInput) nameInput.focus();
  });

  onMount(async () => {
    tools = await api.getTools();
    if (tools.length > 0) selectedTool = tools[0];
  });

  function basename(path: string): string {
    const parts = path.split('/');
    return parts[parts.length - 1] || path;
  }

  let nameValid = $derived(VALID_NAME.test(folderName));
  let nameError = $derived(
    folderName.length === 0
      ? ''
      : nameValid
        ? ''
        : 'Lowercase letters, numbers, and hyphens only (e.g., my-project)',
  );

  async function create() {
    if (!nameValid || creating) return;
    creating = true;
    error = '';
    try {
      const folder = await api.createFolder(root, folderName);
      const session = await api.createSession(folder.path, selectedTool);
      location.hash = `#/terminal/${session.id}`;
      onclose();
    } catch (e: any) {
      error = e.message || 'Failed to create project';
      creating = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
    if (e.key === 'Enter' && nameValid && !creating) create();
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="overlay" onclick={onclose} onkeydown={handleKeydown} role="dialog">
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="dialog" onclick={(e) => e.stopPropagation()} onkeydown={handleKeydown}>
    <h3>New Project in {basename(root)}</h3>

    <label>
      Folder name
      <input
        bind:this={nameInput}
        bind:value={folderName}
        type="text"
        placeholder="my-project"
        class:invalid={nameError}
        disabled={creating}
      />
      {#if nameError}
        <span class="validation">{nameError}</span>
      {/if}
    </label>

    <label>
      Tool
      <select bind:value={selectedTool} disabled={creating}>
        {#each tools as t}
          <option value={t}>{t}</option>
        {/each}
      </select>
    </label>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <div class="actions">
      <button onclick={onclose} disabled={creating}>Cancel</button>
      <button class="primary" onclick={create} disabled={!nameValid || creating}>
        {creating ? 'Creating...' : 'Create & Launch'}
      </button>
    </div>
  </div>
</div>

<style>
  .overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6);
             display: flex; align-items: center; justify-content: center; z-index: 100; }
  .dialog { background: #1e1e30; padding: 24px; border-radius: 8px; min-width: 360px;
            color: #ccc; border: 1px solid #333; }
  h3 { margin: 0 0 16px; }
  label { display: block; margin-bottom: 16px; font-size: 13px; color: #999; }
  input, select { display: block; width: 100%; margin-top: 4px; padding: 8px;
           background: #252540; border: 1px solid #333; color: #ccc; border-radius: 4px;
           box-sizing: border-box; }
  input:focus, select:focus { outline: none; border-color: #555; }
  input.invalid { border-color: #f04040; }
  .validation { font-size: 11px; color: #f04040; margin-top: 4px; display: block; }
  .error { font-size: 12px; color: #f04040; margin: 0 0 12px; }
  .actions { display: flex; gap: 8px; justify-content: flex-end; }
  button { padding: 8px 12px; background: #333; border: none; color: #ccc;
           border-radius: 4px; cursor: pointer; }
  button:hover:not(:disabled) { background: #444; }
  button:disabled { opacity: 0.5; cursor: not-allowed; }
  .primary { background: #4a5568; }
  .primary:hover:not(:disabled) { background: #5a6578; }
</style>
