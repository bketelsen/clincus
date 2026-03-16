<script lang="ts">
  import WorkspaceGrid from '../components/WorkspaceGrid.svelte';
</script>

<div class="dashboard">
  <h2>Workspaces</h2>
  <WorkspaceGrid />
</div>

<style>
  .dashboard { padding: 16px; color: #ccc; }
  h2 { margin: 0 0 12px; font-size: 18px; }
</style>
