<script lang="ts">
  import Layout from './components/Layout.svelte';
  import SessionList from './components/SessionList.svelte';
  import Dashboard from './routes/Dashboard.svelte';
  import Terminal from './routes/Terminal.svelte';
  import Settings from './routes/Settings.svelte';
  import { connectEvents } from './lib/ws';
  import { loadSessions, removeSession } from './stores/sessions.svelte';
  import { loadWorkspaces } from './stores/workspaces.svelte';
  import { loadConfig } from './stores/config.svelte';

  let route = $state(location.hash || '#/');
  let routeParam = $state('');

  function parseRoute() {
    const hash = location.hash || '#/';
    route = hash;
    const termMatch = hash.match(/^#\/terminal\/(.+)$/);
    routeParam = termMatch ? termMatch[1] : '';
  }

  $effect(() => {
    parseRoute();
    window.addEventListener('hashchange', parseRoute);
    loadSessions();
    loadWorkspaces();
    loadConfig();

    // AC4: on reconnect, re-fetch all state so the UI is current even if
    // events were missed while the WebSocket was disconnected.
    function refreshAll() {
      loadSessions();
      loadWorkspaces();
      loadConfig();
    }

    const events = connectEvents((evt) => {
      if (evt.type === 'session.started') {
        loadSessions();
        loadWorkspaces();
      } else if (evt.type === 'session.stopped' && evt.id) {
        removeSession(evt.id);
        loadWorkspaces();
      } else if (evt.type === 'config.reloaded') {
        // AC3 (E01S02): config changed on disk — re-fetch config + workspaces.
        loadConfig();
        loadWorkspaces();
      }
    }, refreshAll);

    return () => {
      window.removeEventListener('hashchange', parseRoute);
      events.close();
    };
  });
</script>

<Layout>
  {#snippet sidebar()}
    <SessionList />
  {/snippet}
  {#snippet main()}
    {#if route === '#/' || route === '#/dashboard'}
      <Dashboard />
    {:else if routeParam}
      <Terminal containerId={routeParam} />
    {:else if route === '#/settings'}
      <Settings />
    {:else}
      <Dashboard />
    {/if}
  {/snippet}
</Layout>
