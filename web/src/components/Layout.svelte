<script lang="ts">
  import type { Snippet } from 'svelte';
  let { sidebar, main }: { sidebar: Snippet; main: Snippet } = $props();
</script>

<div class="layout">
  <aside class="sidebar">
    {@render sidebar()}
  </aside>
  <main class="content">
    {@render main()}
  </main>
</div>

<style>
  .layout { display: flex; height: 100vh; background: #0f0f1a; }
  .sidebar { width: 260px; border-right: 1px solid #333; overflow-y: auto; background: #1a1a2e; }
  .content { flex: 1; overflow: hidden; }
</style>
