# Session Lifecycle

Sessions are the core abstraction in clincus. Each session represents an AI coding tool running inside an Incus container with a mounted workspace.

## Naming Convention

Container name format: `<prefix><workspace-hash>-<slot>`

- **Prefix**: `clincus-` (configurable via `CLINCUS_CONTAINER_PREFIX`)
- **Hash**: First 8 characters of SHA256 of the absolute workspace path
- **Slot**: Integer 1-10, auto-allocated to first available

Example: workspace `/home/user/myproject` → container `clincus-a1b2c3d4-1`

Functions in `internal/session/naming.go`:
- `ContainerName(workspace, slot)` — generate name
- `WorkspaceHash(path)` — SHA256 first 8 chars
- `AllocateSlot()` — find first available slot (1-10)
- `ParseContainerName()` — extract hash and slot
- `ListWorkspaceSessions()` — all slots for a workspace

## Session Setup (11 Steps)

Defined across `internal/session/setup.go` and helper files.

### Phase 1: Resolution (`resolveContainer`)

1. **Generate container name** from workspace path hash + slot
2. **Determine image** — default "clincus", overridable by flag/config
3. **Determine exec context** — non-root user (`code`, UID 1000) for clincus image, root for others

### Phase 2: Container Creation (`createAndConfigureContainer`)

4. **Check for existing container** — reuse if persistent and already running
5. **Create and configure container**:
   - UID/GID mapping: `shift=true` for local, `raw.idmap` for CI, disabled for Colima/Lima
   - Mount workspace as disk device at `/workspace` (or preserved host path if `preserve_workspace_path=true`)
   - Configure `/tmp` tmpfs size if set
   - Mount additional directories from config `[mounts.default]`
   - Mount security paths read-only (`.git/hooks`, `.git/config`, `.husky`, `.vscode`)
   - Apply resource limits via `limits.ApplyResourceLimits()`
   - Start container

### Phase 3: Post-Launch (`postLaunchSetup`)

6. **Wait for readiness** — poll until container responds, set metadata labels
7. **Start timeout monitor** — background goroutine if `max_duration` configured

### Phase 4: Tool Configuration (`configureToolAccess`)

9. **Restore session data** if resuming (non-persistent sessions): pull tool config from saved session directory
10. _(workspace already mounted in step 5)_
11. **Setup CLI tool config**:
    - **Directory-based tools** (Claude, Copilot): Copy config directory, inject credentials, merge settings
    - **File-based tools** (Opencode): Write single config file to container home
    - **ENV-based tools**: No file config needed
    - Path rewriting: host home paths → container home paths
    - Settings merge: Python JSON manipulation for safe `.claude/settings.json` injection

## Tool Configuration Details

### Claude (`internal/tool/tool.go`)

- Config dir: `~/.claude/`
- Sessions dir: `~/.clincus/sessions-claude/`
- Resume: discovers `.jsonl` files in `projects/-workspace/`
- Command: `claude --verbose --permission-mode bypassPermissions --session-id <id>`
- Sandbox settings injected into `settings.json` with effort level
- Essential files: `.credentials.json`, `config.yml`, `settings.json`, `plugins/`, `hooks/`
- Auto-env: `GH_TOKEN`

### Copilot (`internal/tool/copilot.go`)

- Config dir: `~/.copilot/`
- Command: `copilot --allow-all-tools`
- No CLI resume support
- Essential files: `config.json`, `mcp-config.json`, `agents/`
- Auto-env: `GH_TOKEN`

### Opencode (`internal/tool/opencode.go`)

- Config file: `~/.opencode.json` (file-based, not directory)
- Command: `opencode` (always fresh session)
- No CLI resume (SQLite-based)
- Sandbox: `{ "permission": { "*": "allow" } }`

### GitHub Token Resolution (`internal/tool/github.go`)

Priority: `GH_TOKEN` env → `GITHUB_TOKEN` env → `gh auth token` CLI → empty

## Session Persistence

### Persistent Mode (`--persistent`)

- Container kept running after session ends
- Reused on next session start for same workspace+slot
- Tool state lives in the running container

### Non-Persistent Mode (Default)

- On cleanup: tool config pulled to `~/.clincus/sessions-<tool>/<session-id>/`
- Container deleted after stop
- On resume: session data restored from saved directory, fresh credentials injected

### Session Metadata

Stored in `~/.clincus/sessions-<tool>/<session-id>/metadata.json`:
```json
{
  "id": "uuid",
  "container": "clincus-a1b2c3d4-1",
  "workspace": "/home/user/project",
  "persistent": false,
  "saved_at": "2026-03-22T10:00:00Z"
}
```

## Cleanup (`internal/session/cleanup.go`)

1. Save session data (pull tool config directory to local storage)
2. Check container status with exponential backoff (detect shutdown completion)
3. **Persistent**: keep container running
4. **Non-persistent**: delete if stopped; keep if still running (for re-attach)
5. Record stop in history

## History (`internal/session/history.go`)

- JSONL-based append-only log at `~/.clincus/history.jsonl`
- Records: `RecordStart(id, workspace, tool, persistent)`, `RecordStop(id, exitCode)`
- File locking prevents concurrent writes
- `ListHistory()` merges start/stop records, returns newest first

## Mount Configuration Sharing

Config-defined mounts are parsed by `session.MountConfigFromConfig()` (`internal/session/types.go`), which handles tilde expansion and path validation. Both the CLI (`internal/cli/mount_parser.go`) and the web dashboard (`internal/server/api_sessions.go`) use this shared function to ensure config mounts (e.g., `~/.ssh` for git-over-SSH) are applied consistently. CLI-specific `--mount HOST:CONTAINER` flag parsing adds to the mount list but remains in the cli package.

## Mount Strategy

| Mount Type | Source | Target | Shift | Read-Only |
|------------|--------|--------|-------|-----------|
| Workspace | Host workspace dir | `/workspace` or preserved path | Yes | No |
| Config mounts | `[mounts.default]` entries | Specified container path | Yes | No |
| Security mounts | `.git/hooks`, `.vscode`, etc. | Same relative path | Yes | **Yes** |
| CLI flag mounts | `--mount HOST:CONTAINER` | Specified container path | Yes | No |

### Security Protections

- Protected paths from config `[security]`: `.git/hooks`, `.git/config`, `.husky`, `.vscode`
- Symlink detection: refuses to mount symlinks as protected paths
- `--writable-git-hooks` flag overrides `.git/hooks` protection
- Additional paths via `[security].additional_protected_paths`
- All protection disableable via `[security].disable_protection`

## Resource Limits (`internal/limits/`)

Applied via Incus config keys before container start.

| Category | Config Keys | Validation |
|----------|-------------|------------|
| CPU | `limits.cpu`, `limits.cpu.allowance`, `limits.cpu.priority` | Regex: count, range, list, percentage, time slice |
| Memory | `limits.memory`, `limits.memory.enforce`, `limits.memory.swap` | Size or percentage |
| Disk | `limits.read`, `limits.write`, `limits.max`, `limits.disk.priority` | Rate or IOPS |
| Runtime | `limits.processes` + `TimeoutMonitor` goroutine | Duration parsing |

`TimeoutMonitor` runs as a background goroutine with context cancellation. When duration expires, stops container (gracefully or forcefully based on config).
