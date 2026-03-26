# Clincus — Developer Documentation

## Purpose

Clincus is a CLI tool and web dashboard for running AI coding sessions (Claude, Copilot, Opencode) inside isolated Incus containers. It handles container lifecycle, workspace mounting, session persistence, resource limits, and provides both a Cobra-based CLI and a Svelte 5 web dashboard for managing sessions.

Derived from [code-on-incus](https://github.com/mensfeld/code-on-incus). Web dashboard inspired by [wingthing](https://github.com/ehrlich-b/wingthing).

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  User Interface                                         │
│  ┌──────────────┐  ┌────────────────────────────────┐   │
│  │  CLI (Cobra)  │  │  Web Dashboard (Svelte 5)      │   │
│  │  cmd/clincus/ │  │  web/ → webui/ (go:embed)      │   │
│  └──────┬───────┘  └──────────────┬─────────────────┘   │
│         │                         │                      │
│  ┌──────┴───────┐  ┌─────────────┴──────────────────┐   │
│  │ internal/cli/ │  │ internal/server/               │   │
│  │ 21 commands   │  │ REST API + WebSocket + Bridge  │   │
│  └──────┬───────┘  └──────────────┬─────────────────┘   │
│         │                         │                      │
│  ┌──────┴─────────────────────────┴─────────────────┐   │
│  │  Core Packages                                    │   │
│  │  ┌────────────┐ ┌──────────┐ ┌────────────────┐  │   │
│  │  │ session/   │ │ config/  │ │ container/     │  │   │
│  │  │ lifecycle  │ │ TOML +   │ │ Incus mgmt    │  │   │
│  │  │ + naming   │ │ reload   │ │ + commands     │  │   │
│  │  └────────────┘ └──────────┘ └────────────────┘  │   │
│  │  ┌────────────┐ ┌──────────┐ ┌────────────────┐  │   │
│  │  │ tool/      │ │ limits/  │ │ image/         │  │   │
│  │  │ AI tool    │ │ CPU/mem/ │ │ builder +      │  │   │
│  │  │ abstraction│ │ disk/run │ │ versioning     │  │   │
│  │  └────────────┘ └──────────┘ └────────────────┘  │   │
│  │  ┌────────────┐ ┌──────────┐ ┌────────────────┐  │   │
│  │  │ terminal/  │ │ health/  │ │ cleanup/       │  │   │
│  │  │ TERM + PTY │ │ checks   │ │ orphan detect  │  │   │
│  │  └────────────┘ └──────────┘ └────────────────┘  │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                          │
                    ┌─────┴─────┐
                    │   Incus   │
                    │ Containers│
                    └───────────┘
```

### Package Responsibilities

| Package | Path | Role |
|---------|------|------|
| **cli** | `internal/cli/` | 21 Cobra commands (19 in root.go + attach/shutdown via init()) with global flags for workspace, slot, image, limits |
| **server** | `internal/server/` | HTTP API, WebSocket terminal/shell/events, PTY bridge |
| **session** | `internal/session/` | Setup (11-step flow), cleanup, naming, history, persistence |
| **container** | `internal/container/` | Incus operations: launch, exec, mount, snapshot, file transfer |
| **config** | `internal/config/` | TOML loading (system→user→project→env), hot-reload, profiles |
| **tool** | `internal/tool/` | AI tool interface: Claude, Copilot, Opencode with config injection |
| **limits** | `internal/limits/` | Resource limits: CPU, memory, disk, processes, duration timeout |
| **image** | `internal/image/` | Image building with versioning, network waits, DNS fixes |
| **terminal** | `internal/terminal/` | TERM sanitization (exotic → xterm-256color) |
| **health** | `internal/health/` | System checks: Incus, storage, network, permissions |
| **cleanup** | `internal/cleanup/` | Orphan detection: `IsOrphanedWorkspace()` checks if a container's workspace path still exists on host |
| **webui** | `webui/` | `go:embed` wrapper for built Svelte frontend assets |

## Key Patterns

### Session Naming & Slot Allocation

Container names follow `<prefix><workspace-hash>-<slot>`:
- Prefix: `clincus-` (configurable via `CLINCUS_CONTAINER_PREFIX`)
- Hash: first 8 chars of SHA256 of absolute workspace path
- Slot: integer 1-10, auto-allocated to first available

### Session Setup Flow (11 Steps)

1. Generate/resolve container name from workspace hash
2. Determine image (default: "clincus")
3. Determine exec context (non-root for clincus image, root otherwise)
4. Check for existing container to reuse
5. Create container: UID mapping, workspace mount, tmpfs, additional mounts, security mounts, resource limits, start
6. Wait for readiness, set metadata labels
7. Start timeout monitor if `max_duration` configured
8. _(reserved)_
9. Restore session data if resuming
10. _(workspace already mounted in step 5)_
11. Setup CLI tool config (credentials injection, settings merge)

### Tool Abstraction

Tools implement a `Tool` interface with optional capabilities:

| Tool | Config Type | Resume Support | Config Location |
|------|------------|----------------|-----------------|
| Claude | Directory-based | Yes (JSONL discovery) | `~/.claude/` |
| Copilot | Directory-based | No | `~/.copilot/` |
| Opencode | File-based | No (SQLite) | `~/.opencode.json` |

All tools run inside tmux sessions for monitoring and reattachment. Credentials are injected fresh on resume.

### Security Model

- **UID shifting**: Bind mounts use Incus UID/GID shifting (disabled for Colima/Lima)
- **Protected paths**: `.git/hooks`, `.git/config`, `.husky`, `.vscode` mounted read-only
- **Symlink rejection**: Protected path setup refuses symlinks
- **File API**: Path traversal prevention, 5MB size limit, binary detection

### Config Hierarchy (Lowest → Highest Precedence)

1. Built-in defaults
2. System: `/etc/clincus/config.toml`
3. User: `~/.config/clincus/config.toml`
4. Project: `./.clincus.toml`
5. Environment variables (`CLINCUS_*`)
6. CLI flags

Config supports hot-reload via file watcher with 1-second debounce. Failed reloads retain previous valid config.

### Web Dashboard Architecture

- **Backend**: Go HTTP server with REST API + 3 WebSocket endpoints
- **Frontend**: Svelte 5 SPA with hash routing, xterm.js terminals, Monaco editor
- **Terminal bridge**: PTY ↔ WebSocket relay via `creack/pty`
- **Events**: Incus lifecycle monitoring (`incus monitor`) broadcasts session.started/stopped
- **Assets**: Embedded via `go:embed` from `webui/dist/`

### Mount Configuration

Mount parsing is shared between the CLI and web dashboard via `session.MountConfigFromConfig()` in `internal/session/types.go`. This ensures config-defined mounts (e.g., `~/.ssh` for git-over-SSH) are applied consistently whether a session is created from the CLI or the dashboard. CLI-specific `--mount` flag parsing remains in `internal/cli/mount_parser.go`.

### Server Internals

- **Config hot-reload**: `ConfigManager` uses `fsnotify` with 1-second debounce. `handleUpdateConfig` clones the current config, mutates the clone, then atomically swaps via `UpdateConfig()` under a `sync.RWMutex`
- **CodeUID resolution**: `Server.codeUID()` helper centralizes container user ID lookup (defaults to 1000, overridable via `incus.code_uid` config). Used by terminal, shell, session, and file API handlers
- **Clean command**: Stopped-container and orphaned-container cleanup share `confirmAndDeleteContainers()` helper for confirmation prompts and deletion logic

### CLI Framework

The CLI uses Cobra for command structure with `charm.land/fang/v2` for styled output and version display. The `Execute()` function wraps Cobra's execution with fang options (`WithVersion`, `WithCommit`). Config is loaded in `PersistentPreRunE` so it's available to all subcommands. CLI limit flags are merged with config-file limits in `mergeLimitsConfig()`, with CLI flags taking precedence.

### Build Pipeline

- `make web` builds Svelte frontend → `webui/dist/`
- `make build` compiles Go binary with embedded frontend + version ldflags
- GoReleaser Pro produces Linux/Darwin binaries + deb/rpm/apk packages
- Cosign signs all release artifacts

## Configuration Reference

See [configuration.md](configuration.md) for the complete config reference with all TOML fields, defaults, environment variables, and profiles.

## Detailed Documentation

- [CLI Commands](cli-commands.md) — All 21 commands with flags and behavior
- [Web Dashboard & API](web-dashboard.md) — REST API, WebSocket protocol, frontend architecture
- [Session Lifecycle](session-lifecycle.md) — Setup flow, persistence, cleanup, naming
- [Configuration](configuration.md) — Complete TOML reference with defaults and env vars
