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
- **Resize on tab switch:** When a terminal pane (Session or Shell) becomes visible, trigger `fitAddon.fit()` since the container dimensions may have changed while the tab was hidden. Similarly, trigger Monaco's `editor.layout()` when the Editor tab becomes visible.
- `SessionView` tracks `activeTab: 'session' | 'shell' | 'editor'` and boolean flags `shellInitialized`, `editorInitialized` that flip on first visit.

### SessionHeader Data

All data comes from the existing sessions store — no new API calls. The store already has workspace name (host path), tool, status, and container ID. The `workspace` field contains the host-side path (e.g., `/home/user/projects/myapp`) which is what we display in the header — it's more meaningful to the user than the container-side mount path.

## Shell Backend

### New Endpoint: `GET /ws/shell/{id}`

WebSocket endpoint for an independent bash session in the container.

**Implementation** (`ws_shell.go`):
- Mirrors `ws_terminal.go` — upgrades to WebSocket, creates a `Bridge`
- Bridge command: `incus exec --force-interactive --env TERM=xterm-256color --user {uid} {id} -- bash -c 'cd /workspace && exec bash'`
- Same `WSMessage` protocol: `input`, `output`, `resize`, `exit`, `error`
- Same PTY-to-WebSocket goroutine pair from `bridge.go`

**Bridge refactor:** The current `NewBridge` hardcodes tmux attach as the command. Refactor to accept the incus exec arguments as a generic slice:

```go
// NewBridge now accepts incus exec subcommand args (everything after "incus").
// Bridge continues to handle platform-specific sg wrapping internally.
func NewBridge(ws *websocket.Conn, containerName string, execArgs []string, uid int) (*Bridge, error)
```

`execArgs` represents the `incus exec ...` arguments (e.g., `["exec", "--force-interactive", "--env", "TERM=xterm-256color", "--user", "1000", containerName, "--", "bash"]`). Bridge continues to handle the platform-specific `sg incus-admin -c` wrapping on Linux vs direct execution on macOS — callers don't deal with that.

Update `ws_terminal.go` to pass the tmux attach args. `ws_shell.go` passes the bash args.

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

**Implementation:** Runs `incus exec {id} -- find {workspacePath}/{path} -maxdepth 1 -not -name '.' -printf '%f\t%y\t%s\n'` to get name, type, and size in a single command with tab-delimited output (safe for filenames with spaces). Type is `d` for directory, `f` for regular file, etc. Fetches one directory level at a time — no recursion. Hidden files (dotfiles) included. Empty directories return an empty entries list (find handles this gracefully, unlike stat with shell glob expansion). Uses existing `ExecArgsCapture()` method for command execution and output capture.

**Path safety:** All requested paths are cleaned with `filepath.Clean()` and then validated:
1. Reject paths containing `..` after cleaning
2. Reject absolute paths (must be relative to workspace)
3. Join with workspace root and verify the resolved path is still a prefix of the workspace root
4. No symlink resolution — symlinks within the container workspace are followed normally by the container's filesystem. The path validation ensures the *requested* path doesn't escape; if a symlink inside the container points outside `/workspace`, that's the container's concern (Incus already isolates the container filesystem).

### `GET /api/sessions/{id}/files/content?path=src/main.go`

Returns file content.

```json
{
  "path": "src/main.go",
  "content": "package main\n...",
  "size": 2048
}
```

**Implementation:** Two-step read:
1. Pre-check: `incus exec {id} -- stat -c '%s' {workspacePath}/{path}` to get file size. If > 5MB, return HTTP 413 without reading content.
2. Read: `incus exec {id} -- head -c 5242880 {workspacePath}/{path}` to read up to 5MB (safety cap even if stat was OK).
3. Binary detection: check the first 512 bytes of the read content for null bytes. If found, return HTTP 422 with `"Binary file, cannot display"`.

**Guards:**
- Max file size: 5MB. Larger files return HTTP 413 with an error message.
- Binary detection: post-read check on first 512 bytes. Binary files return HTTP 422.

### `PUT /api/sessions/{id}/files/content?path=src/main.go`

Saves file content.

```json
{ "content": "package main\n..." }
```

**Implementation:** Write content to a temp file on the host, then push it into the container via `PushFile()` followed by `Chown()` to set ownership to the `code` user (UID 1000). Both methods already exist in `manager.go`. The existing `PushFile()` does not support `--uid`/`--gid` flags directly, but `Chown()` handles ownership after the push.

**Request body limit:** Enforce `http.MaxBytesReader` capping request body at 5MB, matching the read limit.

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
- Vite integration: use `@monaco-editor/loader` or manual ESM worker imports (`monaco-editor/esm/vs/editor/editor.worker`) rather than `vite-plugin-monaco-editor` (unmaintained, may not support Vite 6). Verify compatibility during implementation and pick whichever approach works cleanly.
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
| `@monaco-editor/loader` (or manual ESM worker config) | Vite worker config | Dev dependency only |

No other new dependencies. xterm.js already installed for shell reuse.

### Backend (Go)

No new dependencies. Shell WebSocket reuses gorilla/websocket. File API reuses `incus exec` patterns. Path validation uses stdlib `path/filepath`.

### Build Pipeline

- `make web` (Vite) — Monaco workers bundled as separate chunks automatically
- `go:embed` in `webui/` — picks up new chunks, no changes needed
- No changes to GoReleaser, CI, or `Makefile`
- **Binary size impact:** Monaco adds ~2-3MB to the embedded frontend assets, increasing the Go binary size accordingly. Acceptable for a locally-served tool.

## Testing

### Backend (Go)

- **`ws_shell.go`**: Unit test that handler upgrades WebSocket and creates bridge with correct command slice (mock WebSocket, verify command args)
- **`api_files.go`**: Unit tests for:
  - `handleListFiles` — directory listing parsing, path validation (reject `..`, absolute paths)
  - `handleReadFile` — content retrieval, 5MB size guard, binary detection
  - `handleWriteFile` — content write, ownership verification
- **`bridge.go` refactor**: Verify existing terminal tests still pass after `NewBridge` signature change

### Frontend

- No frontend unit test framework currently in use beyond Vitest (added in recent work). Add tests for:
  - File API functions in `api.ts` (mock fetch, verify request construction)
  - Path validation logic if any is client-side

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
| `internal/server/bridge.go` | Refactor `NewBridge` to accept generic command slice |
| `internal/server/ws_terminal.go` | Update `NewBridge` call to pass tmux command as slice |
| `web/src/lib/ws.ts` | Add `connectShell()` function (or refactor `connectTerminal` to accept URL path) |
| `web/src/lib/api.ts` | Add `put` helper and file API functions (`listFiles`, `readFile`, `writeFile`) |
| `web/src/lib/types.ts` | Add types for file listing/content API responses |
| `web/package.json` | Add monaco-editor, @monaco-editor/loader |
| `web/vite.config.ts` | Configure Monaco worker loading |

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
