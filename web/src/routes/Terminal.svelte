<script lang="ts">
  import TerminalPane from '../components/TerminalPane.svelte';
  let { containerId }: { containerId: string } = $props();
</script>

<div class="terminal-view">
  <TerminalPane {containerId} />
</div>

<style>
  .terminal-view { width: 100%; height: 100%; }
</style>
