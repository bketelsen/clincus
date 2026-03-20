<script lang="ts">
  import WorkspaceCard from './WorkspaceCard.svelte';
  import type { Workspace } from '../lib/types';

  let { workspaces }: { workspaces: Workspace[] } = $props();
</script>

{#each workspaces as ws (ws.path)}
  <WorkspaceCard workspace={ws} />
{/each}
