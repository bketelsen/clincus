# Architecture

**Analysis Date:** 2026-03-17

## Pattern Overview

**Overall:** Layered CLI + Web Dashboard architecture with separation between CLI command orchestration, container lifecycle management, session persistence, and web server APIs.

**Key Characteristics:**
- CLI commands delegate to internal packages (config, container, session, tool)
- Container management abstracted through `Manager` interface wrapping Incus CLI
- Sessions are persistent entities with unique IDs, stored in filesystem
- Web dashboard provides alternative UI with REST API + WebSocket bridges to container terminal
- AI tool abstraction enables swappable tools (Claude, Aider, OpenCode, etc.)
- Configuration is hierarchical: system → user → project → profile → CLI flags

## Layers

**CLI Layer:**
- Purpose: Cobra command definitions and orchestration
- Location: `internal/cli/`
- Contains: Root command, subcommands (shell, run, list, build, clean, etc.), flag parsing, error handling
- Depends on: config, container, session, server, tool, limits, health, cleanup
- Used by: End users via `clincus` binary; Web dashboard via serve command

**Configuration Layer:**
- Purpose: Load, validate, and manage TOML configuration from multiple sources
- Location: `internal/config/`
- Contains: Config structs, loader, watcher for hot-reload, profile application
- Depends on: standard library
- Used by: CLI root command, Server, Session setup

**Container Management Layer:**
- Purpose: Abstraction over Incus container lifecycle (launch, stop, delete, exec, mount, config)
- Location: `internal/container/`
- Contains: Manager struct wrapping incus CLI, command execution, container status checks
- Depends on: Incus CLI (exec'd as subprocess)
- Used by: Session setup, CLI commands (run, shell, etc.)

**Session Layer:**
- Purpose: Session lifecycle management—naming, persistence, history, cleanup
- Location: `internal/session/`
- Contains: Session ID generation, container naming, setup (mounts, tool config), history tracking, cleanup
- Depends on: config, container, limits, tool
- Used by: CLI shell/run commands, Server session APIs

**Tool Abstraction Layer:**
- Purpose: Pluggable AI coding tools with unified interface
- Location: `internal/tool/`
- Contains: Tool interface, implementations (Claude, Aider, OpenCode), registry, command building
- Depends on: standard library
- Used by: Session setup (to build tool-specific commands and discover session IDs)

**Resource Limits Layer:**
- Purpose: CPU, memory, disk, process, and runtime duration limits
- Location: `internal/limits/`
- Contains: Limit structs, validator, applier (translates to incus config), duration timer
- Depends on: container, standard library
- Used by: Session setup, CLI commands

**Server Layer (Web Dashboard):**
- Purpose: HTTP REST API and WebSocket server for dashboard UI
- Location: `internal/server/`
- Contains: Server struct, route handlers, event hub, terminal WebSocket bridge, config hot-reload
- Depends on: config, container
- Used by: Web dashboard frontend via fetch/WebSocket

**Terminal/PTY Layer:**
- Purpose: PTY and tmux bridging for interactive terminal sessions
- Location: `internal/terminal/`
- Contains: tmux session management, PTY allocation, shell execution
- Depends on: container, standard library
- Used by: Shell command, Server WebSocket terminal handler

**Health & Monitoring:**
- Purpose: System health checks (Incus, permissions, image availability)
- Location: `internal/health/`
- Contains: Health check runners, status enums, result structs
- Depends on: config, container, standard library
- Used by: Health CLI command

**Cleanup Layer:**
- Purpose: Orphan container and resource detection/cleanup
- Location: `internal/cleanup/`
- Contains: Orphan detection logic, cleanup procedures
- Depends on: container, session
- Used by: Clean CLI command

**Image Building:**
- Purpose: Custom image creation for Clincus sessions
- Location: `internal/image/`
- Contains: Image builder using Incus publish API
- Depends on: container, standard library
- Used by: Build CLI command

**Web Frontend:**
- Purpose: Dashboard UI for session and workspace management
- Location: `web/src/`
- Contains: Svelte components, API client, stores, WebSocket client
- Depends on: REST API endpoints and WebSocket handlers in server layer
- Used by: Web browsers connecting to dashboard

## Data Flow

**Interactive Shell Session (clincus shell):**

1. User runs `clincus shell --workspace /path/to/project`
2. Root command loads config, applies profile if specified
3. Shell command resolves workspace path, checks Incus availability
4. Allocates slot number (auto or user-specified)
5. Generates container name: `clincus-<workspace-hash>-<slot>`
6. Session setup creates container:
   - Launches ephemeral or persistent container from image
   - Mounts workspace directory and any configured mounts
   - Sets resource limits (CPU, memory, disk, process count)
   - Applies tool-specific configuration (Claude session ID, Aider keys, etc.)
7. Tool abstraction builds command with appropriate flags
8. Terminal layer creates tmux session: `clincus-<container-id>`
9. Tool binary executes in container via `incus exec`
10. User interacts directly with terminal session
11. On exit: container cleaned up (ephemeral) or stopped (persistent)

**Web Dashboard Session:**

1. Browser connects to `/api/sessions` GET
2. Server lists active sessions by querying Incus
3. User clicks "Launch" → POST `/api/sessions` with workspace + tool
4. Server spawns background session (via existing CLI logic)
5. Session added to list; dashboard refreshes every 2 seconds
6. User clicks terminal icon → upgrade to `/ws/terminal/{id}` WebSocket
7. Bridge opens tmux session capture and relays output/input
8. Terminal display updates in real-time via WebSocket messages
9. Server also listens to `/ws/events` for Incus container lifecycle events

**Configuration Hot-Reload (Dashboard):**

1. User edits `~/.config/clincus/config.toml`
2. Watcher detects file change
3. New config loaded and validated
4. Server's config reference updated (thread-safe via RWMutex)
5. Broadcast `config.reloaded` event to WebSocket clients
6. Dashboard re-fetches `/api/config` and updates UI

**State Management:**

- **Session State:** Persisted in filesystem:
  - Session metadata: `~/.config/clincus/sessions.json` or similar
  - Tool session IDs: `~/.claude/sessions-claude/` or equivalent per tool
  - History: Tracked in logs directory
- **Container State:** Managed by Incus daemon (persistent storage in Incus DB)
- **Web Frontend State:** Client-side stores (Svelte `.svelte.ts` stores) synced to server via API

## Key Abstractions

**Container Manager:**
- Purpose: Encapsulates all Incus CLI interactions into typed methods
- Examples: `internal/container/manager.go`
- Pattern: Wrapper interface with methods like `Launch()`, `Stop()`, `MountDisk()`, `Exec()`, `SetConfig()`
- Rationale: Makes container operations testable and replaceable; centralizes Incus command building

**Session:**
- Purpose: Represents a user's interactive or background work session
- Examples: Container naming, session IDs, history tracking
- Pattern: Immutable session metadata + mutable lifecycle state (running → stopped)
- Rationale: Supports both ephemeral and persistent containers; enables resume functionality

**Tool Interface:**
- Purpose: Abstracts AI coding tools behind a common interface
- Examples: `internal/tool/tool.go` with implementations for Claude, Aider, OpenCode
- Pattern: Interface defines `BuildCommand()`, `DiscoverSessionID()`, `GetSandboxSettings()`
- Rationale: Enables swappable tools without changing session/container code

**EventHub:**
- Purpose: Pub-sub for WebSocket events (Incus lifecycle, config changes)
- Examples: `internal/server/server.go`
- Pattern: Central event broadcaster to all connected clients
- Rationale: Keeps dashboard in sync across multiple browser tabs/clients

**Config Watcher:**
- Purpose: Monitors config file changes and triggers hot-reload
- Examples: `internal/config/watcher.go`
- Pattern: File system watcher + callback mechanism
- Rationale: Enables configuration updates without restarting server

## Entry Points

**CLI Binary (`clincus`):**
- Location: `cmd/clincus/main.go`
- Triggers: User execution in terminal
- Responsibilities: Initialize context, call `cli.Execute()` which runs Cobra root command

**Cobra Root Command:**
- Location: `internal/cli/root.go` → `Execute(ctx context.Context)`
- Triggers: Every CLI invocation
- Responsibilities: Parse global flags, load config, initialize container module, dispatch to subcommand

**Subcommands:**
- `shell` (`internal/cli/shell.go`): Interactive AI coding session in tmux
- `run` (`internal/cli/run.go`): Execute single command, capture output, cleanup
- `list` (`internal/cli/list.go`): List active sessions
- `build` (`internal/cli/build.go`): Build clincus image
- `clean` (`internal/cli/clean.go`): Cleanup orphan containers
- `serve` (`internal/cli/serve.go`): Start web dashboard server
- See `internal/cli/root.go` lines 142-159 for full command list

**Web Server:**
- Location: `internal/cli/serve.go`
- Triggers: `clincus serve` command
- Responsibilities: Create `internal/server/Server`, attach routes, start HTTP listener
- Routes: Listed in `internal/server/server.go` lines 70-83

**WebSocket Terminal Handler:**
- Location: `internal/server/ws_terminal.go` → `handleTerminalWS()`
- Triggers: Browser upgrade to `/ws/terminal/{id}`
- Responsibilities: Validate container, create tmux capture bridge, relay I/O

## Error Handling

**Strategy:** Explicit error propagation with context wrapping using `fmt.Errorf(...%w)`, exiting with code 1 on fatal errors.

**Patterns:**
- **Container Operations:** Wrapped in `ExitError` struct (see `internal/container/manager.go`) to distinguish exit codes from system errors
- **Session Setup:** Early validation (config, image existence, workspace path) before container launch
- **Config Loading:** Validates TOML syntax and required fields; returns error that stops root command
- **WebSocket:** Returns 400/404 HTTP errors before upgrade; uses error JSON messages over WebSocket for runtime failures
- **Limits Validation:** Pre-flight validation in `internal/limits/validator.go` before applying to container

## Cross-Cutting Concerns

**Logging:** Printf-style logging to stderr (no structured logger); verbose flags control detail level (e.g., tool binary runs with `--verbose`)

**Validation:**
- Config: Validated on load by TOML unmarshal + manual checks
- Paths: Resolved to absolute paths, checked for existence
- Limits: Validated by `limits.ValidateAll()` with user-friendly error messages
- Mount paths: Validated for traversal attacks in `internal/session/mount_validator.go`

**Authentication:**
- CLI: No authentication (tool assumes host user is authorized)
- Web Dashboard: No built-in auth (assumes localhost access or proxy auth); WebSocket accepts all origins (`CheckOrigin: true`)

**Resource Cleanup:**
- Ephemeral containers: Deleted automatically on session exit
- Persistent containers: Kept for resume; `clincus kill` forces stop; `clincus clean` removes orphans
- Mount directories: Created if missing, never auto-deleted
- Tmux sessions: Cleaned up when container stops

**Concurrency:**
- Server config: Protected by RWMutex for hot-reload
- WebSocket broadcasting: EventHub uses mutex-protected client list
- Container operations: Incus daemon handles concurrency (assumed single or clustered incus daemon)

**Security:**
- Protected paths (e.g., `.git/hooks`) mounted read-only to prevent shell injection
- Container runs as unprivileged user (configurable, default 1000)
- UID shifting enabled by default (disabled for Colima/Lima)
- Git hooks writing disabled by default (requires `--writable-git-hooks` flag)
