# Session Tabbed Workspace

**Date:** 2026-03-19
**Status:** Approved

## Summary

Add a tabbed workspace UI to the web dashboard's session view. When a session is open, a header shows session metadata (name, tool, status, workspace path) with a tab bar below offering three views: **Session** (existing tmux terminal), **Shell** (independent bash in the container), and **Editor** (Monaco-based file editor for the workspace).

## Motivation

Currently the session view is a full-screen tmux terminal with no chrome. Users who want to run commands alongside the AI tool or quickly edit a file must either use a separate terminal or SSH into the container. A tabbed workspace brings shell access and file editing into the same interface without leaving the dashboard.

## Component Architecture

### Current

```
App.svelte → Terminal.svelte → TerminalPane.svelte
```

### New

```
App.svelte → SessionView.svelte
               ├── SessionHeader.svelte  (name, tool, status, workspace path)
               ├── TabBar.svelte         (Session | Shell | Editor)
               └── Tab content (conditional):
                   ├── TerminalPane.svelte   (existing, unchanged)
                   ├── ShellPane.svelte      (new xterm.js + /ws/shell/{id})
                   └── EditorPane.svelte     (new)
                         ├── FileTree.svelte
                         └── MonacoEditor.svelte
```

### Layout

Two-line header above the content area:
- **Line 1 (SessionHeader):** Session name, tool name, running status badge, workspace path (right-aligned)
- **Line 2 (TabBar):** Session | Shell | Editor tabs, active tab has bottom accent border

The hash route stays `#/terminal/{id}`. `Terminal.svelte` is replaced by `SessionView.svelte` in the router.

### Tab Loading Strategy

- **Lazy initialization:** Shell and Editor only initialize when their tab is first clicked. Session (tmux) initializes immediately since it's the default tab.
- **State preservation:** Once initialized, panes are hidden via CSS `display:none` rather than unmounted. This preserves xterm.js terminal scrollback/cursor and Monaco editor state without reconnecting WebSockets.
- `SessionView` tracks `activeTab: 'session' | 'shell' | 'editor'` and boolean flags `shellInitialized`, `editorInitialized` that flip on first visit.

### SessionHeader Data

All data comes from the existing sessions store — no new API calls. The store already has workspace name, tool, status, and container ID. The workspace path is available from the session metadata (`user.clincus.workspace` container config key).

## Shell Backend

### New Endpoint: `GET /ws/shell/{id}`

WebSocket endpoint for an independent bash session in the container.

**Implementation** (`ws_shell.go`):
- Mirrors `ws_terminal.go` — upgrades to WebSocket, creates a `Bridge`
- Bridge command: `incus exec --force-interactive --env TERM=xterm-256color --user {uid} {id} -- bash -c 'cd /workspace && exec bash'`
- Same `WSMessage` protocol: `input`, `output`, `resize`, `exit`, `error`
- Same PTY-to-WebSocket goroutine pair from `bridge.go`

The only difference from the terminal handler is the exec command (bash instead of tmux attach) and the working directory (`/workspace`).

### ShellPane Component

- New xterm.js `Terminal` instance, same configuration as `TerminalPane`
- Connects to `/ws/shell/{containerId}` instead of `/ws/terminal/{containerId}`
- On shell exit (user types `exit`): shows `[Shell exited - click to restart]`, re-initializes on next tab click
- On navigation away: closes WebSocket, shell process dies

### Route Registration

Add to `server.go` route table:
```go
mux.HandleFunc("GET /ws/shell/{id}", srv.handleShellWS)
```

## File REST API

Three new endpoints for the editor. All file paths are relative to the container's workspace mount point.

### `GET /api/sessions/{id}/files?path=/`

Returns directory listing for the given path.

```json
{
  "path": "/",
  "entries": [
    { "name": "src", "type": "dir" },
    { "name": "main.go", "type": "file", "size": 2048 },
    { "name": "go.mod", "type": "file", "size": 512 }
  ]
}
```

**Implementation:** Runs `incus exec {id} -- ls -1pLA {workspacePath}/{path}` and parses output. Directories identified by trailing `/` from `ls -p`. Fetches one directory level at a time — no recursion.

**Path safety:** Reject paths containing `..`, absolute paths, symlinks escaping workspace root. Resolve the workspace path from the container's device config via existing `GetWorkspacePath()`.

### `GET /api/sessions/{id}/files/content?path=src/main.go`

Returns file content.

```json
{
  "path": "src/main.go",
  "content": "package main\n...",
  "size": 2048
}
```

**Implementation:** `incus exec {id} -- cat {workspacePath}/{path}` with output capture.

**Guards:**
- Max file size: 5MB. Larger files return HTTP 413 with an error message.
- Binary detection: check first 512 bytes for null bytes. Binary files return HTTP 422 with `"Binary file, cannot display"`.

### `PUT /api/sessions/{id}/files/content?path=src/main.go`

Saves file content.

```json
{ "content": "package main\n..." }
```

**Implementation:** Uses the existing `PushFile()` method (wraps `incus file push`) or pipes content via `incus exec {id} -- tee {workspacePath}/{path}`. File is written atomically as the `code` user (UID 1000).

### Route Registration

```go
mux.HandleFunc("GET /api/sessions/{id}/files", srv.handleListFiles)
mux.HandleFunc("GET /api/sessions/{id}/files/content", srv.handleReadFile)
mux.HandleFunc("PUT /api/sessions/{id}/files/content", srv.handleWriteFile)
```

## Editor UI

### Layout

```
┌─────────────┬──────────────────────────────────┐
│  File Tree   │  Monaco Editor                    │
│  (resizable) │                                   │
│              │  filename.go                       │
│  ▼ src/      │  ──────────────────────────────── │
│    main.go   │  package main                     │
│    util.go   │                                   │
│  ▶ tests/    │  import "fmt"                     │
│  go.mod      │                                   │
└─────────────┴──────────────────────────────────┘
```

### FileTree Component

- Fetches root directory on init via `GET /api/sessions/{id}/files?path=/`
- Folders expand/collapse on click, fetching children lazily
- Clicking a file opens it in the Monaco editor
- Indentation-based tree rendering
- Manual refresh button at the top (no auto-refresh, no polling)
- No drag-and-drop, no multi-select

### MonacoEditor Component

- Single `monaco.editor.create()` instance, swap content via `editor.setModel()` when files change
- Language auto-detection from file extension (Monaco built-in)
- Full default language set: Go, TypeScript, JavaScript, Python, Rust, Ruby, Java, C/C++, C#, PHP, Swift, HTML, CSS, SCSS, JSON, YAML, TOML, XML, Markdown, Shell, Dockerfile, SQL, and all other Monaco-bundled languages. No filtering — include everything Monaco ships.
- Save: `Ctrl+S` / `Cmd+S` triggers `PUT /api/sessions/{id}/files/content`
- Unsaved indicator (dot/marker) on file name when buffer is dirty
- Theme: dark theme matching app (`#1a1a2e` background)
- Single file open at a time (no editor tabs)
- Cursor position and scroll position preserved across tab switches (Monaco instance stays alive in hidden DOM)

### Monaco Integration

- npm package: `monaco-editor`
- Vite integration: `vite-plugin-monaco-editor` for worker configuration
- Workers bundled as separate chunks by Vite, picked up automatically by `go:embed`

## Error Handling

### Container stops mid-session

- Terminal and shell WebSockets receive `exit`/close events
- File API calls return errors
- `SessionHeader` reflects status via existing event hub (`session.stopped`)
- Each pane shows inline message: `[Container stopped]`
- No auto-navigation — user may want to read terminal scrollback

### Large / binary files

- Files > 5MB: HTTP 413, editor shows "File too large to edit"
- Binary files (null bytes in first 512 bytes): HTTP 422, editor shows "Binary file, cannot display"

### Shell exits

- `exit` or process death → `[Shell exited - click to restart]`
- Re-clicking the shell tab re-initializes: new WebSocket, fresh bash

### File save conflicts

- No conflict detection in v1. Last write wins.
- Matches natural terminal + editor workflow — user coordinates
- Modification-time checking can be added later if needed

### Network interruption

- Shell WebSocket: same behavior as terminal WebSocket (no reconnect on drop, consistent)
- File API calls: use existing `ApiError` retry logic

## New Dependencies

### Frontend (npm)

| Package | Purpose | Bundle Impact |
|---------|---------|---------------|
| `monaco-editor` | Code editor | ~2.5MB raw, full language set |
| `vite-plugin-monaco-editor` | Vite worker config | Dev dependency only |

No other new dependencies. xterm.js already installed for shell reuse.

### Backend (Go)

No new dependencies. Shell WebSocket reuses gorilla/websocket. File API reuses `incus exec` patterns. Path validation uses stdlib `path/filepath`.

### Build Pipeline

- `make web` (Vite) — Monaco workers bundled as separate chunks automatically
- `go:embed` in `webui/` — picks up new chunks, no changes needed
- No changes to GoReleaser, CI, or `Makefile`

## Files Changed

### New files

| File | Purpose |
|------|---------|
| `web/src/routes/SessionView.svelte` | Tab container replacing Terminal.svelte |
| `web/src/components/SessionHeader.svelte` | Session info header bar |
| `web/src/components/TabBar.svelte` | Tab navigation component |
| `web/src/components/ShellPane.svelte` | Independent bash terminal |
| `web/src/components/EditorPane.svelte` | Editor layout (tree + Monaco) |
| `web/src/components/FileTree.svelte` | Workspace file tree |
| `web/src/components/MonacoEditor.svelte` | Monaco editor wrapper |
| `internal/server/ws_shell.go` | Shell WebSocket handler |
| `internal/server/api_files.go` | File listing/read/write endpoints |

### Modified files

| File | Change |
|------|--------|
| `web/src/App.svelte` | Import SessionView instead of Terminal |
| `internal/server/server.go` | Register new routes |
| `web/package.json` | Add monaco-editor, vite-plugin-monaco-editor |
| `web/vite.config.ts` | Configure Monaco plugin |

### Removed files

| File | Reason |
|------|--------|
| `web/src/routes/Terminal.svelte` | Replaced by SessionView.svelte |

## Out of Scope

- Multi-tab editor (multiple files open simultaneously)
- File upload/download
- Git integration in editor
- Auto-refresh file tree
- Terminal WebSocket reconnection (neither existing terminal nor new shell reconnect today)
- File search / find-in-files
- Editor tabs or split panes
