<script lang="ts">
  import SessionCard from './SessionCard.svelte';
  import { getSessions } from '../stores/sessions.svelte';
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
</div>

<style>
  .list { padding: 0; }
  .header { display: flex; justify-content: space-between; align-items: center; padding: 12px; }
  h3 { margin: 0; font-size: 14px; color: #ccc; }
  .home { color: #888; text-decoration: none; font-size: 20px; }
  .empty { color: #666; font-size: 12px; padding: 12px; }
</style>
