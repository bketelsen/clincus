# Architecture Overview

This document explains how Clincus works internally — the container lifecycle, how the web
dashboard is delivered, how PTY bridging works, and the Go package structure.

---

## High-Level Design

Clincus is a single static binary that:

1. Wraps the `incus` CLI to manage containers
2. Embeds a compiled Svelte SPA as a Go `embed.FS`
3. Provides an HTTP+WebSocket server for the web dashboard
4. Bridges WebSocket connections to PTY sessions inside containers

There are no daemons, no external databases, and no services to install. Everything runs
as the current user (modulo `incus-admin` group membership).

```
┌──────────────────────────────────────────────────────┐
│                  clincus binary                      │
│                                                      │
│  ┌────────────────┐   ┌──────────────────────────┐  │
│  │   CLI (cobra)  │   │    HTTP Server           │  │
│  │  shell/attach  │   │  REST API + WebSocket    │  │
│  │  build/list    │   │  Embedded Svelte SPA     │  │
│  │  snapshot/file │   └──────────────────────────┘  │
│  └────────────────┘                                  │
│         │                        │                   │
│  ┌──────▼──────────────────────▼──────────┐         │
│  │           internal packages             │         │
│  │  session  container  config  tool       │         │
│  │  image    health     limits  terminal   │         │
│  └─────────────────────────────────────────┘         │
│         │                                            │
└─────────┼────────────────────────────────────────────┘
          │ exec "incus ..."
          ▼
   ┌─────────────┐
   │   Incus     │
   │  (system)   │
   └─────────────┘
```

---

## Container Lifecycle

### Ephemeral Session (default)

```
clincus shell
  │
  ├─ session.Resolve()       — allocate slot, generate session ID + container name
  ├─ session.Setup()
  │    ├─ container.Launch()    — incus launch <image> <name>
  │    ├─ container.MountDisk() — mount workspace + extra mounts
  │    ├─ Copy tool config into container (e.g., ~/.claude/ → container home)
  │    ├─ Apply resource limits via incus config device set
  │    └─ Apply security mounts (read-only paths)
  │
  ├─ tmux new-session -d ...  — start AI tool in detached tmux inside container
  ├─ tmux attach              — attach current terminal to that tmux session
  │
  [user works, detaches with Ctrl+B d, or AI tool exits]
  │
  └─ session.Cleanup()
       ├─ Copy tool config back to host (save session state)
       ├─ Write metadata.json and session history entry
       └─ container.Delete()  — incus delete --force <name>
```

### Persistent Session

The lifecycle is the same except:

- The container is created with `persistent: true` in Incus metadata
- On exit, `container.Stop()` is called instead of `container.Delete()`
- On resume, the existing container is started (`container.Start()`) rather than
  re-created

---

## Session Naming and Slot Allocation

Container names encode the workspace and slot:

```
clincus-<workspace-hash>-<slot>
```

The workspace hash is a 6-character lowercase hex prefix of the SHA-256 of the absolute
workspace path. This makes container names stable across renames of parent directories
but unique per workspace.

Slot allocation (`session.AllocateSlot`) queries Incus for containers matching the workspace
hash prefix and picks the lowest integer (1–10) not already in use.

---

## Session Persistence on Disk

Session state is stored in `~/.clincus/sessions-<tool>/<session-id>/`:

```
~/.clincus/
  sessions-claude/
    abc123xyz/
      .claude/            # full copy of ~/.claude/ from the container
      metadata.json       # workspace, container name, timestamp, persistent flag
  sessions-opencode/
    def456uvw/
      metadata.json
  history.jsonl           # append-only log of all session starts/ends
```

At session start, the tool config directory is copied **into** the container so the AI tool
sees its credentials and conversation history. At session end, it is copied back to update
the saved state.

---

## Workspace Mounting

Clincus uses Incus's `disk` device type to bind-mount the host workspace:

```
incus config device add <container> workspace disk source=<host-path> path=/workspace shift=true
```

The `shift=true` parameter enables UID-shifting (idmapped mounts), so files owned by your
host UID appear as `code` (UID 1000) inside the container. On macOS VMs where this is not
supported, `shift=false` with `disable_shift = true` in config.

Security-sensitive subdirectories (`.git/hooks`, `.vscode`, etc.) are overlaid with
additional read-only disk devices on top of the workspace mount.

---

## tmux as Session Manager

All sessions run inside a tmux session named `clincus-<container-name>`. Using tmux provides:

- **Detach/reattach** — `Ctrl+B d` detaches without killing the AI tool
- **Background sessions** — `--background` creates a detached tmux session
- **Terminal capture** — `clincus tmux capture` reads tmux scrollback
- **Command injection** — `clincus tmux send` sends text to the running session

When `clincus shell` or `clincus attach` is called on an already-running container, it
detects the existing tmux session and attaches to it directly.

The AI tool runs inside a bash wrapper that traps `SIGINT` (so `Ctrl+C` goes to the tool,
not to bash) and falls back to an interactive bash prompt when the tool exits. This lets
you stay in the container without restarting.

---

## Web Dashboard and Embedded SPA

The Svelte 5 app in `web/` is built by `make web` (runs `npm run build` in `web/`) and
outputs compiled assets to `webui/dist/`. The `webui` Go package embeds that directory:

```go
//go:embed dist
var Dist embed.FS
```

When `clincus serve` starts, the server serves the embedded SPA from the root path and all
API/WebSocket routes under `/api/` and `/ws/`.

SPA routing: any path that does not match a known static asset falls back to `index.html`,
allowing client-side navigation.

---

## PTY Bridging for Terminal in Browser

The WebSocket terminal (`/ws/terminal/{id}`) works as follows:

1. A WebSocket connection is opened from the browser
2. The server calls `incus exec <container> --user 1000 -- tmux attach -t clincus-<name>`
  with a PTY allocated
3. The server pumps bytes between the WebSocket and the PTY in both directions
4. The browser renders the byte stream using xterm.js

This gives the browser-based terminal identical behavior to `clincus attach`.

---

## Real-Time Events

The server subscribes to the Incus Unix socket event API and filters events for containers
whose names start with the Clincus prefix. These events are broadcast to all connected
`/ws/events` WebSocket clients, enabling the dashboard to update without polling.

---

## Package Structure

```
cmd/clincus/          — main.go: entry point, calls cli.Execute()
internal/
  cli/                — cobra commands: shell, attach, build, list, etc.
  config/             — config.toml loading, merging, defaults
  container/          — Incus wrapper: launch, stop, delete, exec, mount, snapshot
  session/            — session lifecycle: resolve, setup, cleanup, history, naming
  tool/               — tool abstraction: claude, copilot, opencode, registry
  image/              — image build: clincus image and custom images
  health/             — health checks for all dependencies
  limits/             — resource limit application via incus CLI
  server/             — HTTP server: REST API, WebSocket bridge
  terminal/           — TERM sanitization, PTY utilities
  cleanup/            — orphaned resource cleanup
web/                  — Svelte 5 frontend source
webui/                — Go package that embeds web/dist
```

### Key Abstractions

**`tool.Tool` interface** (in `internal/tool/tool.go`):

```go
type Tool interface {
    Name() string
    ConfigDirName() string
    BuildCommand(sessionID string, resume bool, cliSessionID string) []string
    DiscoverSessionID(statePath string) string
}
```

Each supported tool implements this interface. The CLI uses the interface to build the
in-container command, locate config directories to copy in/out, and discover the tool's
internal session ID for resumption.

**`container.Manager`** (in `internal/container/manager.go`):

Thin wrapper around `incus exec`, `incus launch`, `incus delete`, and related commands.
All Incus operations go through this type, making it straightforward to mock in tests.

**`session.Setup` / `session.Cleanup`**:

The two main operations that orchestrate the full container lifecycle. `Setup` launches the
container, mounts disks, copies credentials, and applies limits. `Cleanup` copies state
back, writes metadata, and deletes (or stops) the container.
