# Web Dashboard & API

The web dashboard provides a browser-based interface for managing AI coding sessions. Backend is in `internal/server/`, frontend in `web/`, embedded via `webui/`.

## REST API

### Session Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sessions` | List running sessions with container metadata |
| POST | `/api/sessions` | Create session (body: workspace, tool, persistent). Applies config mounts via `session.MountConfigFromConfig()` |
| DELETE | `/api/sessions/{id}` | Stop session (`?force=true` to kill) |
| POST | `/api/sessions/{id}/resume` | Resume stopped persistent session |
| GET | `/api/sessions/history` | Session history (query: limit, offset) |

### Workspace Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/workspaces` | List workspaces from configured roots |
| POST | `/api/workspaces` | Add workspace root |
| DELETE | `/api/workspaces?path=...` | Remove workspace root |
| POST | `/api/workspaces/folder` | Create project folder in a root |

### Config Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/config` | Get full config |
| PUT | `/api/config` | Update config (port, workspace roots) |
| GET | `/api/tools` | List supported tools |

### File Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sessions/{id}/files?path=...` | List directory contents |
| GET | `/api/sessions/{id}/files/content?path=...` | Read file (max 5MB) |
| PUT | `/api/sessions/{id}/files/content?path=...` | Write file |

File security: path traversal rejected, binary detection (null byte in first 512 bytes), operations run as `CodeUID` (default 1000, configurable via `incus.code_uid`). The `Server.codeUID()` helper centralizes UID resolution across terminal, shell, session, and file handlers.

## WebSocket Endpoints

### Terminal (`/ws/terminal/{id}`)

Connects to the tmux session running the AI tool inside the container.

Flow:
1. Validates container is running
2. Executes: `incus exec --force-interactive --env TERM=xterm-256color <container> -- tmux attach-session -t clincus-<id>`
3. PTY ↔ WebSocket bridge via `creack/pty` (16KB read buffer)

### Shell (`/ws/shell/{id}`)

Independent bash shell in the container (separate from tmux).

Flow:
1. Launches: `bash --login -c "cd <workspace> && exec bash --login"`
2. Sets HOME and TERM env vars
3. Same PTY bridge as terminal

### Events (`/ws/events`)

Persistent event stream for real-time updates.

- Server watches `incus monitor --type lifecycle --format json`
- Broadcasts: `session.started`, `session.stopped`, `config.reloaded`
- Client auto-reconnects with exponential backoff (2s → 4s → 8s → 30s max)
- On reconnect, frontend refetches all state

### WebSocket Message Protocol

```json
{
  "type": "output|input|resize|exit|error",
  "data": "terminal content or input",
  "cols": 80,
  "rows": 24,
  "code": 0,
  "message": "error description"
}
```

| Type | Direction | Purpose |
|------|-----------|---------|
| `output` | Server→Client | Terminal data from PTY |
| `input` | Client→Server | Keyboard input |
| `resize` | Client→Server | Terminal dimensions changed |
| `exit` | Server→Client | Process exited with code |
| `error` | Server→Client | Error notification |

## Frontend Architecture

### Tech Stack

- **Framework**: Svelte 5 (runes-based reactivity)
- **Terminal**: xterm.js 6.0 with fit addon
- **Editor**: Monaco Editor 0.55
- **Bundler**: Vite 6
- **Routing**: Hash-based (`#/`, `#/dashboard`, `#/terminal/{id}`, `#/settings`)

### Component Hierarchy

```
App.svelte (router + event subscription)
├── Layout.svelte (sidebar + main content)
│   ├── SessionList.svelte (sidebar session cards)
│   │   └── SessionCard.svelte (stop/kill/detach actions)
│   ├── Dashboard.svelte (workspace grid)
│   │   ├── WorkspaceCard.svelte (project cards)
│   │   ├── LaunchDialog.svelte (create session modal)
│   │   └── NewFolderDialog.svelte
│   ├── SessionView.svelte (tabbed view)
│   │   ├── TabBar.svelte (Terminal/Shell/Editor tabs)
│   │   ├── TerminalPane.svelte (xterm.js → /ws/terminal)
│   │   ├── ShellPane.svelte (xterm.js → /ws/shell + restart)
│   │   └── EditorPane.svelte
│   │       ├── FileTree.svelte (lazy-loaded tree)
│   │       └── MonacoEditor.svelte (syntax highlighting)
│   └── Settings.svelte (config UI)
```

### State Management (Svelte Stores)

| Store | File | State |
|-------|------|-------|
| Sessions | `stores/sessions.svelte.ts` | Active sessions array |
| Workspaces | `stores/workspaces.svelte.ts` | Workspaces, roots, expanded state |
| Config | `stores/config.svelte.ts` | Full ClincusConfig mirror |

### API Client (`lib/api.ts`)

Retry strategy: exponential backoff 1s→2s→4s, max 3 retries. Retries on network errors and 5xx. 4xx thrown immediately.

### Terminal Configuration

- Font: 14px JetBrains Mono
- Theme: Dark (#1a1a2e background), 256-color
- Cursor blink enabled
- FitAddon for responsive sizing
- Lazy initialization (only when pane visible)

## Config Hot-Reload

1. `ConfigManager` watches system and user config files (`fsnotify`)
2. 1-second debounce, failed reloads retain previous config
3. `handleUpdateConfig` uses atomic config swap: clones current config, mutates clone, swaps atomically via `UpdateConfig()` under `sync.RWMutex`
4. Server broadcasts `config.reloaded` via WebSocket events
5. Frontend refetches config + workspaces on event receipt

## Session Container Metadata

Sessions stored as Incus container config keys:
- `user.clincus.workspace` — workspace path
- `user.clincus.tool` — tool name
- `user.clincus.persistent` — boolean flag
