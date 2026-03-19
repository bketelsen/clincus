# Web Dashboard

Clincus includes a built-in web dashboard — a Svelte 5 single-page application (SPA) embedded
directly in the `clincus` binary. No separate server process or network dependency is required.

---

## Starting the Dashboard

```bash
clincus serve
```

The server starts at `http://127.0.0.1:3000` by default. Open that URL in your browser.

### Open Browser Automatically

```bash
clincus serve --open
```

This runs `xdg-open` to launch your default browser.

### Custom Port

```bash
clincus serve --port 8080
```

Or set a default in config:

```toml
# ~/.config/clincus/config.toml
[dashboard]
port = 8080
```

---

## Dashboard Features

### Session List

The main view shows all active Clincus containers with:

- Container name and slot number
- Status (running / stopped)
- Workspace path
- AI tool in use
- Persistent / ephemeral indicator

### Launch a New Session

Click **New Session** to open a dialog. Select:

- **Workspace** — choose from your configured workspace roots or type a path
- **Tool** — Claude Code, opencode, Aider, or any configured tool
- **Persistent** — toggle persistent mode

The dashboard calls `POST /api/sessions` to create the container and start the tool.

### Terminal in Browser

Click the terminal icon next to any running session to open an in-browser terminal connected
to the session's tmux window. This uses a WebSocket connection (`ws://127.0.0.1:3000/ws/terminal/{id}`)
and a PTY bridge running in the container.

The terminal supports full ANSI color, cursor control, and keyboard input — the same experience
as attaching from the CLI.

### Stop a Session

Click the stop icon (or the trash icon to force-delete) next to a session.

- **Stop** — sends `DELETE /api/sessions/{id}` which gracefully stops the container
- **Force** — sends `DELETE /api/sessions/{id}?force=true` which force-deletes the container

### Session History

The **History** tab shows past sessions loaded from `~/.clincus/history.jsonl`. Each entry
includes the workspace, tool, start time, and session ID.

### Workspaces

The **Workspaces** tab shows configured workspace roots. Add directories here to make them
available in the New Session dialog without typing full paths.

### Settings / Configuration View

The **Settings** page displays the full Clincus configuration in a structured, read-only layout.
All config sections are shown: defaults, paths, incus, tool, mounts, limits, git, security,
profiles, and dashboard. Values are rendered with clear labels grouped by section.

- **Workspace Roots** at the top remains editable (add/remove paths)
- All other config values are read-only — edit your TOML config files to change them
- The **Mounts** and **Profiles** sections are collapsible for large configurations
- Empty or default values display a dash to indicate no override is set
- The display updates automatically when config files change on disk (via `config.reloaded` events)

---

## Workspace Roots

Configure workspace roots in your config so they appear as suggestions in the dashboard:

```toml
# ~/.config/clincus/config.toml
[dashboard]
workspace_roots = [
  "~/projects",
  "~/work",
  "/srv/repos",
]
```

---

## Real-Time Updates

The dashboard subscribes to `GET /ws/events` — a WebSocket event stream that pushes container
state changes from Incus in real time. Session cards update automatically when containers
start, stop, or change state without requiring a page refresh.

---

## Config Hot-Reload

The dashboard server watches config files for changes and reloads automatically. If you edit
`~/.config/clincus/config.toml` or `/etc/clincus/config.toml` while the dashboard is running,
changes take effect within a few seconds without restarting the server.

When a config reload succeeds, the server broadcasts a `config.reloaded` event over the
`/ws/events` WebSocket. The frontend listens for this event and re-fetches the configuration
from `GET /api/config`, ensuring the dashboard always displays the current active settings.

If you change the `[dashboard] port` in the config, the listener is restarted on the new port.
The frontend WebSocket connection includes automatic reconnection — if the connection drops
during a port change, the client reconnects and fetches the latest state. Active tmux sessions
inside containers continue running and clients can reconnect on the new port.

---

## Stopping the Dashboard

Press `Ctrl+C` in the terminal where `clincus serve` is running.

The dashboard server is intentionally bound to `127.0.0.1` only and is not suitable for
exposure to a network without an authenticating reverse proxy in front of it.
