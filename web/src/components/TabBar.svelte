<script lang="ts">
  let { activeTab, onTabChange }: {
    activeTab: string;
    onTabChange: (tab: string) => void;
  } = $props();

  const tabs = [
    { id: 'session', label: 'Session' },
    { id: 'shell', label: 'Shell' },
    { id: 'editor', label: 'Editor' },
  ];
</script>

<div class="tab-bar">
  {#each tabs as tab}
    <button
      class="tab"
      class:active={activeTab === tab.id}
      onclick={() => onTabChange(tab.id)}
    >
      {tab.label}
    </button>
  {/each}
</div>

<style>
  .tab-bar {
    display: flex;
    align-items: center;
    padding: 0 12px;
    gap: 2px;
    height: 32px;
    background: #1a1a2e;
    border-bottom: 1px solid #333;
  }
  .tab {
    padding: 5px 14px;
    color: #888;
    font-size: 12px;
    font-family: inherit;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s;
  }
  .tab:hover {
    color: #bbb;
  }
  .tab.active {
    color: #eee;
    border-bottom-color: #7c6fe0;
  }
</style>
