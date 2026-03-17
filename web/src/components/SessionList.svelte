<script lang="ts">
  import SessionCard from './SessionCard.svelte';
  import { getSessions } from '../stores/sessions.svelte';

  let currentHash = $state(location.hash || '#/');

  $effect(() => {
    function onHashChange() {
      currentHash = location.hash || '#/';
    }
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  });
</script>

<div class="list">
  <div class="header">
    <h3>Sessions</h3>
    <a href="#/" class="home">+</a>
  </div>
  {#each getSessions() as session (session.id)}
    <SessionCard {session} />
  {:else}
    <p class="empty">No active sessions</p>
  {/each}
  <div class="nav-section">
    <a
      href="#/settings"
      class="nav-link"
      class:active={currentHash === '#/settings'}
    >
      <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="3"></circle>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
      </svg>
      Settings
    </a>
  </div>
</div>

<style>
  .list { padding: 0; }
  .header { display: flex; justify-content: space-between; align-items: center; padding: 12px; }
  h3 { margin: 0; font-size: 14px; color: #ccc; }
  .home { color: #888; text-decoration: none; font-size: 20px; }
  .empty { color: #666; font-size: 12px; padding: 12px; }
  .nav-section { border-top: 1px solid #333; margin-top: 8px; padding: 8px 12px; }
  .nav-link {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #888;
    text-decoration: none;
    font-size: 13px;
    padding: 8px;
    border-radius: 6px;
    transition: background-color 0.15s, color 0.15s;
  }
  .nav-link:hover { background-color: #2a2a2a; color: #ccc; }
  .nav-link.active { background-color: #2a2a2a; color: #fff; }
  .nav-icon { width: 16px; height: 16px; flex-shrink: 0; }
</style>
