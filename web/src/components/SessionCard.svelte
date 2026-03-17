<script lang="ts">
  import type { Session } from '../lib/types';
  import { api } from '../lib/api';
  import { removeSession, loadSessions } from '../stores/sessions.svelte';

  let { session }: { session: Session } = $props();

  function detach(e: Event) {
    e.preventDefault();
    e.stopPropagation();
    // Navigate away from terminal if viewing this session
    if (location.hash === `#/terminal/${session.id}`) {
      location.hash = '#/';
    }
  }

  async function stop(e: Event) {
    e.preventDefault();
    e.stopPropagation();
    try {
      await api.stopSession(session.id);
      removeSession(session.id);
      if (location.hash === `#/terminal/${session.id}`) {
        location.hash = '#/';
      }
    } catch {
      // Refresh list to get accurate state
      await loadSessions();
    }
  }

  async function kill(e: Event) {
    e.preventDefault();
    e.stopPropagation();
    try {
      await api.stopSession(session.id, true);
      removeSession(session.id);
      if (location.hash === `#/terminal/${session.id}`) {
        location.hash = '#/';
      }
    } catch {
      await loadSessions();
    }
  }
</script>

<div class="card">
  <a class="info" href="#/terminal/{session.id}">
    <div class="name">{session.workspace ? session.workspace.split('/').pop() : session.id}</div>
    <div class="meta">
      {session.tool || 'unknown'} &middot; {session.status} &middot; {session.id}
    </div>
  </a>
  <div class="actions">
    <button class="btn" onclick={detach} title="Detach (go to dashboard)">&#x2190;</button>
    <button class="btn" onclick={stop} title="Stop session">&#x25A0;</button>
    <button class="btn btn-danger" onclick={kill} title="Force kill">&#x2715;</button>
  </div>
</div>

<style>
  .card { display: flex; align-items: center; border-bottom: 1px solid #2a2a3e; }
  .card:hover { background: #2a2a3e; }
  .info { flex: 1; display: block; padding: 8px 8px 8px 12px; text-decoration: none; color: #ccc;
          cursor: pointer; min-width: 0; }
  .name { font-weight: 600; font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .meta { font-size: 11px; color: #888; margin-top: 2px; }
  .actions { display: flex; gap: 2px; padding-right: 8px; flex-shrink: 0; }
  .btn { background: none; border: none; color: #666; cursor: pointer;
         font-size: 12px; padding: 4px 5px; border-radius: 3px; line-height: 1; }
  .btn:hover { background: #333; color: #ccc; }
  .btn-danger:hover { background: #5a2020; color: #f88; }
</style>
