<script lang="ts">
  import type { Session } from '../lib/types';
  let { session }: { session: Session } = $props();
</script>

<div class="session-header">
  <div class="session-info">
    <span class="session-name">{session.workspace.split('/').pop()}</span>
    <span class="separator">&middot;</span>
    <span class="session-tool">{session.tool}</span>
    <span class="status-badge" class:running={session.status === 'Running'}>
      {session.status.toLowerCase()}
    </span>
  </div>
  <div class="session-path">{session.workspace}</div>
</div>

<style>
  .session-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    background: #1a1a2e;
    border-bottom: 1px solid #222;
  }
  .session-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .session-name {
    color: #eee;
    font-size: 13px;
    font-weight: 600;
  }
  .separator { color: #555; }
  .session-tool {
    color: #888;
    font-size: 12px;
  }
  .status-badge {
    padding: 1px 6px;
    border-radius: 8px;
    font-size: 10px;
    background: #333;
    color: #888;
  }
  .status-badge.running {
    background: #2d4a2d;
    color: #6d6;
  }
  .session-path {
    color: #666;
    font-size: 11px;
  }
</style>
