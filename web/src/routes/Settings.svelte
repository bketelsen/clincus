<script lang="ts">
  import { getRoots, loadWorkspaces } from '../stores/workspaces.svelte';
  import { getConfig, getConfigLoading, getConfigError } from '../stores/config.svelte';
  import { api } from '../lib/api';
  import type { ClincusConfig } from '../lib/types';

  let newPath = $state('');

  // Track which sections are collapsed (profiles/mounts can be large)
  let collapsed = $state<Record<string, boolean>>({});

  function toggle(section: string) {
    collapsed[section] = !collapsed[section];
  }

  async function addRoot() {
    if (!newPath.trim()) return;
    await api.addWorkspace(newPath.trim());
    newPath = '';
    await loadWorkspaces();
  }

  async function removeRoot(path: string) {
    await api.removeWorkspace(path);
    await loadWorkspaces();
  }

  /** Format a value for display, handling empty/null/false gracefully (AC5). */
  function displayValue(val: unknown): string {
    if (val === null || val === undefined) return '\u2014';
    if (val === '') return '\u2014';
    if (typeof val === 'boolean') return val ? 'Yes' : 'No';
    if (typeof val === 'number') return String(val);
    if (Array.isArray(val)) return val.length === 0 ? '\u2014' : val.join(', ');
    return String(val);
  }
</script>

<div class="settings">
  <h2>Settings</h2>

  <!-- Workspace Roots (editable) -->
  <section class="config-section">
    <h3>Workspace Roots</h3>
    <ul class="roots-list">
      {#each getRoots() as root}
        <li>
          <span>{root}</span>
          <button onclick={() => removeRoot(root)}>Remove</button>
        </li>
      {/each}
    </ul>
    <form onsubmit={(e) => { e.preventDefault(); addRoot(); }}>
      <input type="text" bind:value={newPath} placeholder="/path/to/projects" />
      <button type="submit">Add</button>
    </form>
  </section>

  <!-- Read-only config display (AC1, AC2) -->
  {#if getConfigLoading()}
    <p class="status">Loading configuration...</p>
  {:else if getConfigError()}
    <p class="status error">Error: {getConfigError()}</p>
  {:else if getConfig()}
    {@const cfg = getConfig() as ClincusConfig}

    <!-- Defaults -->
    <section class="config-section">
      <h3>Defaults</h3>
      <dl>
        <dt>Image</dt><dd>{displayValue(cfg.defaults.image)}</dd>
        <dt>Persistent</dt><dd>{displayValue(cfg.defaults.persistent)}</dd>
        <dt>Model</dt><dd>{displayValue(cfg.defaults.model)}</dd>
      </dl>
    </section>

    <!-- Paths -->
    <section class="config-section">
      <h3>Paths</h3>
      <dl>
        <dt>Sessions Dir</dt><dd>{displayValue(cfg.paths.sessions_dir)}</dd>
        <dt>Storage Dir</dt><dd>{displayValue(cfg.paths.storage_dir)}</dd>
        <dt>Logs Dir</dt><dd>{displayValue(cfg.paths.logs_dir)}</dd>
        <dt>Preserve Workspace Path</dt><dd>{displayValue(cfg.paths.preserve_workspace_path)}</dd>
      </dl>
    </section>

    <!-- Incus -->
    <section class="config-section">
      <h3>Incus</h3>
      <dl>
        <dt>Project</dt><dd>{displayValue(cfg.incus.project)}</dd>
        <dt>Group</dt><dd>{displayValue(cfg.incus.group)}</dd>
        <dt>Code UID</dt><dd>{displayValue(cfg.incus.code_uid)}</dd>
        <dt>Code User</dt><dd>{displayValue(cfg.incus.code_user)}</dd>
        <dt>Disable Shift</dt><dd>{displayValue(cfg.incus.disable_shift)}</dd>
      </dl>
    </section>

    <!-- Tool -->
    <section class="config-section">
      <h3>Tool</h3>
      <dl>
        <dt>Name</dt><dd>{displayValue(cfg.tool.name)}</dd>
        <dt>Binary</dt><dd>{displayValue(cfg.tool.binary)}</dd>
        <dt>Claude Effort Level</dt><dd>{displayValue(cfg.tool.claude?.effort_level)}</dd>
      </dl>
    </section>

    <!-- Mounts (collapsible) -->
    <section class="config-section">
      <h3>
        <button class="collapse-toggle" onclick={() => toggle('mounts')}>
          {collapsed['mounts'] ? '+' : '-'} Mounts
        </button>
      </h3>
      {#if !collapsed['mounts']}
        {#if cfg.mounts.default && cfg.mounts.default.length > 0}
          <table>
            <thead><tr><th>Host</th><th>Container</th></tr></thead>
            <tbody>
              {#each cfg.mounts.default as mount}
                <tr><td>{mount.host}</td><td>{mount.container}</td></tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="empty">No default mounts configured.</p>
        {/if}
      {/if}
    </section>

    <!-- Limits -->
    <section class="config-section">
      <h3>Limits</h3>

      <h4>CPU</h4>
      <dl>
        <dt>Count</dt><dd>{displayValue(cfg.limits.cpu.count)}</dd>
        <dt>Allowance</dt><dd>{displayValue(cfg.limits.cpu.allowance)}</dd>
        <dt>Priority</dt><dd>{displayValue(cfg.limits.cpu.priority)}</dd>
      </dl>

      <h4>Memory</h4>
      <dl>
        <dt>Limit</dt><dd>{displayValue(cfg.limits.memory.limit)}</dd>
        <dt>Enforce</dt><dd>{displayValue(cfg.limits.memory.enforce)}</dd>
        <dt>Swap</dt><dd>{displayValue(cfg.limits.memory.swap)}</dd>
      </dl>

      <h4>Disk</h4>
      <dl>
        <dt>Read</dt><dd>{displayValue(cfg.limits.disk.read)}</dd>
        <dt>Write</dt><dd>{displayValue(cfg.limits.disk.write)}</dd>
        <dt>Max</dt><dd>{displayValue(cfg.limits.disk.max)}</dd>
        <dt>Priority</dt><dd>{displayValue(cfg.limits.disk.priority)}</dd>
        <dt>Tmpfs Size</dt><dd>{displayValue(cfg.limits.disk.tmpfs_size)}</dd>
      </dl>

      <h4>Runtime</h4>
      <dl>
        <dt>Max Duration</dt><dd>{displayValue(cfg.limits.runtime.max_duration)}</dd>
        <dt>Max Processes</dt><dd>{displayValue(cfg.limits.runtime.max_processes)}</dd>
        <dt>Auto Stop</dt><dd>{displayValue(cfg.limits.runtime.auto_stop)}</dd>
        <dt>Stop Graceful</dt><dd>{displayValue(cfg.limits.runtime.stop_graceful)}</dd>
      </dl>
    </section>

    <!-- Git -->
    <section class="config-section">
      <h3>Git</h3>
      <dl>
        <dt>Writable Hooks</dt><dd>{displayValue(cfg.git.writable_hooks)}</dd>
      </dl>
    </section>

    <!-- Security -->
    <section class="config-section">
      <h3>Security</h3>
      <dl>
        <dt>Disable Protection</dt><dd>{displayValue(cfg.security.disable_protection)}</dd>
        <dt>Protected Paths</dt>
        <dd>
          {#if cfg.security.protected_paths && cfg.security.protected_paths.length > 0}
            <ul class="value-list">
              {#each cfg.security.protected_paths as p}
                <li>{p}</li>
              {/each}
            </ul>
          {:else}
            {'\u2014'}
          {/if}
        </dd>
        <dt>Additional Protected Paths</dt>
        <dd>
          {#if cfg.security.additional_protected_paths && cfg.security.additional_protected_paths.length > 0}
            <ul class="value-list">
              {#each cfg.security.additional_protected_paths as p}
                <li>{p}</li>
              {/each}
            </ul>
          {:else}
            {'\u2014'}
          {/if}
        </dd>
      </dl>
    </section>

    <!-- Profiles (collapsible) -->
    <section class="config-section">
      <h3>
        <button class="collapse-toggle" onclick={() => toggle('profiles')}>
          {collapsed['profiles'] ? '+' : '-'} Profiles
        </button>
      </h3>
      {#if !collapsed['profiles']}
        {#if cfg.profiles && Object.keys(cfg.profiles).length > 0}
          {#each Object.entries(cfg.profiles) as [name, profile]}
            <div class="profile-entry">
              <h4>{name}</h4>
              <dl>
                <dt>Image</dt><dd>{displayValue(profile.image)}</dd>
                <dt>Persistent</dt><dd>{displayValue(profile.persistent)}</dd>
                {#if profile.environment && Object.keys(profile.environment).length > 0}
                  <dt>Environment</dt>
                  <dd>
                    <ul class="value-list">
                      {#each Object.entries(profile.environment) as [k, v]}
                        <li><code>{k}={v}</code></li>
                      {/each}
                    </ul>
                  </dd>
                {/if}
              </dl>
            </div>
          {/each}
        {:else}
          <p class="empty">No profiles configured.</p>
        {/if}
      {/if}
    </section>

    <!-- Dashboard -->
    <section class="config-section">
      <h3>Dashboard</h3>
      <dl>
        <dt>Port</dt><dd>{displayValue(cfg.dashboard.port)}</dd>
        <dt>Workspace Roots</dt><dd>{displayValue(cfg.dashboard.workspace_roots)}</dd>
      </dl>
    </section>
  {/if}
</div>

<style>
  .settings { padding: 16px; color: #ccc; max-width: 700px; overflow-y: auto; max-height: 100vh; }
  h2 { margin: 0 0 16px; }
  h3 { margin: 16px 0 8px; font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; color: #888; }
  h4 { margin: 12px 0 4px; font-size: 13px; color: #aaa; }

  .config-section { margin-bottom: 8px; padding: 12px; background: #1a1a2e; border-radius: 6px; }

  dl { display: grid; grid-template-columns: 180px 1fr; gap: 4px 12px; margin: 0; font-size: 13px; }
  dt { color: #888; padding: 2px 0; }
  dd { margin: 0; padding: 2px 0; word-break: break-all; }

  .roots-list { list-style: none; padding: 0; }
  .roots-list li { display: flex; justify-content: space-between; align-items: center;
       padding: 8px; background: #12122a; margin-bottom: 4px; border-radius: 4px; }
  form { display: flex; gap: 8px; margin-top: 12px; }
  input { flex: 1; padding: 8px; background: #12122a; border: 1px solid #333;
          color: #ccc; border-radius: 4px; }
  button { padding: 8px 12px; background: #333; border: none; color: #ccc;
           border-radius: 4px; cursor: pointer; }
  button:hover { background: #444; }

  .collapse-toggle { background: none; border: none; color: #888; cursor: pointer;
                     font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; padding: 0; }
  .collapse-toggle:hover { color: #ccc; }

  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th { text-align: left; color: #888; padding: 4px 8px; border-bottom: 1px solid #333; }
  td { padding: 4px 8px; }

  .value-list { list-style: none; padding: 0; margin: 0; }
  .value-list li { padding: 1px 0; }
  .value-list code { background: #12122a; padding: 1px 4px; border-radius: 2px; font-size: 12px; }

  .profile-entry { margin: 8px 0; padding: 8px; background: #12122a; border-radius: 4px; }
  .profile-entry h4 { margin: 0 0 4px; color: #ccc; }

  .empty { color: #666; font-style: italic; font-size: 13px; margin: 4px 0; }
  .status { color: #888; font-size: 13px; padding: 16px; }
  .status.error { color: #e55; }
</style>
