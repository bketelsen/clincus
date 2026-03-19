<script lang="ts">
  import SessionHeader from '../components/SessionHeader.svelte';
  import TabBar from '../components/TabBar.svelte';
  import TerminalPane from '../components/TerminalPane.svelte';
  import { getSessions } from '../stores/sessions.svelte';
  import type { Session } from '../lib/types';

  let { containerId }: { containerId: string } = $props();

  let activeTab = $state('session');
  let shellInitialized = $state(false);
  let editorInitialized = $state(false);

  function onTabChange(tab: string) {
    activeTab = tab;
    if (tab === 'shell') shellInitialized = true;
    if (tab === 'editor') editorInitialized = true;
  }

  let session = $derived(
    getSessions().find((s: Session) => s.id === containerId)
  );
</script>

<div class="session-view">
  {#if session}
    <SessionHeader {session} />
  {/if}
  <TabBar {activeTab} {onTabChange} />
  <div class="pane-container">
    <div class="pane" class:hidden={activeTab !== 'session'}>
      <TerminalPane {containerId} visible={activeTab === 'session'} />
    </div>
    {#if shellInitialized}
      <div class="pane" class:hidden={activeTab !== 'shell'}>
        <!-- ShellPane added in Task 8 -->
        <div style="padding: 20px; color: #888;">Shell pane (coming soon)</div>
      </div>
    {/if}
    {#if editorInitialized}
      <div class="pane" class:hidden={activeTab !== 'editor'}>
        <!-- EditorPane added in Task 11 -->
        <div style="padding: 20px; color: #888;">Editor pane (coming soon)</div>
      </div>
    {/if}
  </div>
</div>

<style>
  .session-view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .pane-container {
    flex: 1;
    position: relative;
    overflow: hidden;
  }
  .pane {
    position: absolute;
    inset: 0;
  }
  .pane.hidden {
    display: none;
  }
</style>
