# API Reference

`clincus serve` starts an HTTP server that exposes the REST API and WebSocket endpoints used
by the web dashboard. The server binds to `127.0.0.1:3000` by default (configurable with
`--port` or `[dashboard] port` in config).

All REST endpoints return `application/json`. Error responses use the shape:

```json
{"error": "description of the problem"}
```

---

## REST Endpoints

### `GET /api/config`

Return the full merged Clincus configuration (system + user defaults applied).

**Response:** a JSON object containing all top-level config sections. Keys use `snake_case`
to match `config.toml` field names.

```json
{
  "defaults": { "image": "clincus", "persistent": false, "model": "claude-sonnet-4-5" },
  "paths": { "sessions_dir": "...", "storage_dir": "...", "logs_dir": "...", "preserve_workspace_path": false },
  "incus": { "project": "default", "group": "incus-admin", "code_uid": 1000, "code_user": "code", "disable_shift": false },
  "tool": { "name": "claude", "binary": "", "claude": { "effort_level": "" } },
  "mounts": { "default": [] },
  "limits": {
    "cpu": { "count": "", "allowance": "", "priority": 0 },
    "memory": { "limit": "", "enforce": "soft", "swap": "true" },
    "disk": { "read": "", "write": "", "max": "", "priority": 0, "tmpfs_size": "" },
    "runtime": { "max_duration": "", "max_processes": 0, "auto_stop": true, "stop_graceful": true }
  },
  "git": { "writable_hooks": false },
  "security": { "protected_paths": [".git/hooks", ".git/config", ".husky", ".vscode"], "additional_protected_paths": [], "disable_protection": false },
  "profiles": {},
  "dashboard": { "port": 3000, "workspace_roots": [] }
}
```

---

### `PUT /api/config`

Update runtime dashboard settings (port and workspace roots).

**Request body:**

```json
{
  "port": 3001,
  "workspace_roots": ["/home/user/projects"]
}
```

Both fields are optional. Only provided fields are updated.

**Response:** `200 OK` with `{"status": "updated"}`, or `400 Bad Request`.

---

### `GET /api/tools`

Return the list of supported AI tool names.

**Response:**

```json
["claude", "copilot", "opencode"]
```

---

### `GET /api/sessions`

List all active Clincus containers.

**Response:** array of session objects.

```json
[
  {
    "id": "clincus-a1b2c3-1",
    "workspace": "/home/user/my-project",
    "tool": "claude",
    "slot": 1,
    "status": "running",
    "persistent": false
  }
]
```

#### Session Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Container name (used in all other API calls) |
| `workspace` | string | Host workspace path |
| `tool` | string | AI tool name |
| `slot` | int | Slot number |
| `status` | string | `running`, `stopped`, `starting`, `stopping` |
| `persistent` | bool | Whether the session is persistent |

---

### `POST /api/sessions`

Create and start a new session.

**Request body:**

```json
{
  "workspace": "/home/user/my-project",
  "tool": "claude",
  "persistent": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workspace` | string | yes | Absolute path to the workspace directory |
| `tool` | string | no | AI tool name (default: `"claude"`) |
| `persistent` | bool | no | Persistent mode (default: `false`) |

**Response:** `200 OK` with the created session object, or an error.

```json
{
  "id": "clincus-a1b2c3-1",
  "workspace": "/home/user/my-project",
  "tool": "claude",
  "slot": 1,
  "status": "running",
  "persistent": false
}
```

---

### `DELETE /api/sessions/{id}`

Stop or force-delete a session.

**Path parameter:** `id` — the container name.

**Query parameters:**

| Parameter | Values | Description |
|-----------|--------|-------------|
| `force` | `true` / `false` | Force-delete the container (default: graceful stop) |

**Response:** `204 No Content` on success, `404` if not found, `500` on error.

```bash
# Graceful stop
curl -X DELETE http://127.0.0.1:3000/api/sessions/clincus-a1b2c3-1

# Force delete
curl -X DELETE "http://127.0.0.1:3000/api/sessions/clincus-a1b2c3-1?force=true"
```

---

### `POST /api/sessions/{id}/resume`

Resume a stopped session: starts the container and relaunches the AI tool.

**Path parameter:** `id` — the container name.

**Response:** `200 OK` with `{"id": "...", "status": "running"}`, or an error.

---

### `GET /api/sessions/history`

Return session history from `~/.clincus/history.jsonl`.

**Query parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `limit` | `50` | Number of entries to return |
| `offset` | `0` | Offset for pagination |

**Response:** array of history entry objects.

```json
[
  {
    "id": "abc123",
    "workspace": "/home/user/my-project",
    "tool": "claude",
    "started": "2026-01-01T12:00:00Z",
    "stopped": "2026-01-01T14:30:00Z",
    "persistent": false,
    "exit_code": 0
  }
]
```

---

### `GET /api/workspaces`

Return configured workspace roots and discovered project directories.

**Response:**

```json
{
  "roots": ["~/projects", "~/work"],
  "expanded_roots": ["/home/user/projects", "/home/user/work"],
  "workspaces": [
    {
      "path": "/home/user/projects/my-app",
      "name": "my-app",
      "root": "/home/user/projects",
      "has_config": true,
      "active_sessions": 1
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `roots` | string[] | Configured workspace roots (raw config values) |
| `expanded_roots` | string[] | Workspace roots with `~` expanded to absolute paths |
| `workspaces[].path` | string | Absolute path to the project directory |
| `workspaces[].name` | string | Directory basename |
| `workspaces[].root` | string | Expanded absolute path of the workspace root this project belongs to |
| `workspaces[].has_config` | bool | Whether `.clincus.toml` exists in the project |
| `workspaces[].active_sessions` | int | Number of active sessions for this project |

---

### `POST /api/workspaces`

Add a workspace root to the dashboard's workspace list.

**Request body:**

```json
{"path": "/home/user/new-project"}
```

**Response:** `200 OK` with the updated list, or `400 Bad Request`.

---

### `DELETE /api/workspaces`

Remove a workspace root from the list.

**Request body:**

```json
{"path": "/home/user/old-project"}
```

**Response:** `200 OK` with the updated list, or `400 Bad Request`.

---

### `POST /api/workspaces/folder`

Create a new project directory inside a configured workspace root.

**Request body:**

```json
{
  "root": "/home/user/projects",
  "name": "my-new-project"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root` | string | yes | Absolute path to a configured workspace root |
| `name` | string | yes | Folder name (kebab-case: lowercase alphanumeric with hyphens) |

**Folder name validation:** Must match `^[a-z0-9]+(-[a-z0-9]+)*$`. No spaces, uppercase, or special characters.

**Response:** `201 Created`

```json
{
  "path": "/home/user/projects/my-new-project"
}
```

**Errors:**

| Code | Condition |
|------|-----------|
| `400` | Invalid folder name or root not in configured workspace roots |
| `409` | Directory already exists |
| `500` | Filesystem error creating the directory |

---

## WebSocket Endpoints

### `GET /ws/terminal/{id}`

Open an interactive terminal session connected to the AI tool running in container `{id}`.

**Path parameter:** `id` — the container name.

**Protocol:** WebSocket, upgraded from HTTP.

The server bridges the WebSocket to a PTY attached to the container's tmux session. Messages
in both directions are raw bytes.

```
Client → Server: keyboard input bytes
Server → Client: terminal output bytes (ANSI sequences included)
```

**Connection example (JavaScript):**

```javascript
const ws = new WebSocket(`ws://127.0.0.1:3000/ws/terminal/clincus-a1b2c3-1`);
ws.onmessage = (event) => { term.write(event.data); };
ws.onopen = () => { /* terminal ready */ };
```

The Svelte dashboard uses [xterm.js](https://xtermjs.org/) as the terminal renderer.

---

### `GET /ws/events`

Subscribe to a real-time event stream of container state changes and system events.

**Protocol:** WebSocket.

The server broadcasts events to all connected clients. Each message is a JSON object with
a `type` field. The following event types are emitted:

#### Session Events

Forwarded from the Incus lifecycle monitor, filtered to Clincus containers:

```json
{
  "type": "session.started",
  "id": "clincus-a1b2c3-1"
}
```

| Event type | Description |
|------------|-------------|
| `session.started` | A Clincus container started |
| `session.stopped` | A Clincus container stopped or was deleted |

#### Config Reload Event

Broadcast when the configuration is successfully reloaded from disk (via the config file
watcher). Failed reloads do not produce an event.

```json
{
  "type": "config.reloaded",
  "timestamp": "2026-01-01T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"config.reloaded"` |
| `timestamp` | string | ISO 8601 UTC timestamp of the reload |

The dashboard listens for this event and re-fetches `GET /api/config` to pick up the new
configuration. The WebSocket connection includes automatic reconnection — if the connection
drops during a reload (e.g., a port change), the client reconnects and fetches the latest
state.

The dashboard uses this stream to update session cards and configuration in real time
without polling.

---

## Static Assets

All paths not matching an API or WebSocket route serve the embedded Svelte SPA. Unrecognized
paths fall back to `index.html` (standard SPA routing pattern).

```
GET /          → index.html (Svelte app)
GET /sessions  → index.html (client-side route)
GET /assets/*  → bundled JS/CSS
```
