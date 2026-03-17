<script lang="ts">
  import Layout from './components/Layout.svelte';
  import SessionList from './components/SessionList.svelte';
  import Dashboard from './routes/Dashboard.svelte';
  import Terminal from './routes/Terminal.svelte';
  import Settings from './routes/Settings.svelte';
  import { connectEvents } from './lib/ws';
  import { loadSessions, removeSession } from './stores/sessions.svelte';
  import { loadWorkspaces } from './stores/workspaces.svelte';

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

    const events = connectEvents((evt) => {
      if (evt.type === 'session.started') {
        loadSessions();
        loadWorkspaces();
      } else if (evt.type === 'session.stopped' && evt.id) {
        removeSession(evt.id);
        loadWorkspaces();
      }
    });

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
