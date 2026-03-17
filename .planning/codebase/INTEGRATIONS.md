# External Integrations

**Analysis Date:** 2026-03-17

## APIs & External Services

**Container Orchestration:**
- Incus - Container runtime and lifecycle management
  - Interaction: Via `incus` CLI commands (exec, list, config, device, etc.)
  - Integration points: `internal/container/manager.go` (`IncusExec`, `IncusOutput`)
  - No SDK/authentication required — uses host's Incus credentials via socket

**GitHub/Version Control:**
- GitHub - Optional integration for AI tool token passing
  - SDK/Client: `gh` CLI (GitHub CLI)
  - Token resolution: `internal/tool/github.go` (`ResolveGHToken`)
  - Purpose: Resolves `GH_TOKEN` for Claude and Copilot tools running in containers
  - Auth: Priority: `GH_TOKEN` env var → `GITHUB_TOKEN` env var → `gh auth token` output
  - Passed to container via `AutoEnv()` interface

**AI Coding Tools:**
- Claude Code (Anthropic) - Configurable tool
  - Binary: `claude`
  - Config location: `~/.claude/` (synced to container)
  - Implementation: `internal/tool/tool.go` (ClaudeTool)
  - Session support: Yes (via `-session-id` / `-resume` flags)
  - Environment injection: `GH_TOKEN` if available
  - Sandbox settings injection: `bypassPermissions` mode settings to `.claude/settings.json`

- GitHub Copilot CLI - Configurable tool
  - Binary: `copilot`
  - Config location: `~/.copilot/` (synced to container)
  - Implementation: `internal/tool/copilot.go` (CopilotTool)
  - Session support: No CLI-based resume (starts fresh each time)
  - Environment injection: `GH_TOKEN` if available
  - Essential files: `config.json`, `mcp-config.json`
  - Essential dirs: `agents/`

- OpenCode (Supabase) - Configurable tool
  - Binary: `opencode`
  - Config file: `~/.opencode.json` (single home config file)
  - Implementation: `internal/tool/opencode.go`
  - Session support: Yes (via session ID files)
  - No environment injection required

## Data Storage

**Databases:**
- None - No persistent database required

**File Storage:**
- Local filesystem only - Session state stored in user home directory
  - Location: `~/.clincus/` (configurable via `paths.sessions_dir`)
  - Contents: Session metadata, history.jsonl (JSONL format)
  - Tool config synced from host: `~/.claude/`, `~/.copilot/`, `~/.opencode.json`, etc.
  - Workspace snapshots: Stored in Incus via container snapshots API

**Caching:**
- None - Config loaded on startup and watched for hot-reload via fsnotify

## Authentication & Identity

**Auth Provider:**
- None required for core functionality
- GitHub token resolution (optional) - `internal/tool/github.go`
  - Resolves from host environment variables for tool integration
  - No centralized auth server

**User Identity:**
- UID/GID mapping: Configurable in `config.toml` (`incus.code_uid`, `incus.code_gid`)
- Default: Run tools as unprivileged user in container
- Workspace mounting: Permissions mapped via Incus `shift` setting

## Monitoring & Observability

**Error Tracking:**
- None - No external error tracking service

**Logs:**
- Stdout/stderr: CLI logs printed to terminal
- Log files: Configured via `paths.logs_dir` in config
- Approach: Direct file writes to `~/.clincus/logs/` or custom path
- Format: Plain text logs (tool-specific formats preserved)
- No centralized logging or aggregation

## CI/CD & Deployment

**Hosting:**
- GitHub (repository only)
- Binary distributions via GitHub Releases
- Documentation hosted on GitHub Pages (MkDocs Material)

**CI Pipeline:**
- GitHub Actions
  - `.github/workflows/ci.yml` - Build, test, lint on push/PR to main/staging
    - Go build with golangci-lint v2.11
    - Node.js frontend build
    - Go unit tests with coverage
    - Python ruff linting/formatting
  - `.github/workflows/release.yml` - GoReleaser on version tags (`v*`)
    - Multi-platform builds (linux/darwin, amd64/arm64)
    - Archive creation with completions and man pages
    - Binary signing via cosign
    - Package distribution: deb, rpm, apk via nfpm
  - `.github/workflows/docs.yml` - MkDocs deploy to GitHub Pages

**Binary Distribution:**
- GoReleaser Pro with nfpm packaging
  - Formats: .deb (Debian/Ubuntu), .rpm (RHEL/Fedora), .apk (Alpine)
  - Includes completions and man pages
  - Cosign signing for binary verification

## Environment Configuration

**Required env vars:**
- None strictly required - All core functionality works with defaults
- Optional for development:
  - `GH_TOKEN` / `GITHUB_TOKEN` - GitHub token (auto-injected to tools)
  - `CLINCUS_CONFIG` - Override config file location
  - `CLINCUS_CONTAINER_PREFIX` - Override container naming prefix (default: "clincus-")

**Secrets location:**
- Tool credentials stored in user's home directory (e.g., `~/.claude/`, `~/.copilot/`)
- Not managed by Clincus — pre-existing on host
- Synced into container via workspace mount
- No secrets stored in config files or environment by Clincus itself

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None - Event broadcast is WebSocket-based for real-time dashboard updates
  - `internal/server/ws_events.go` - Monitors `incus monitor --type lifecycle` for container events
  - Events: session.started, session.stopped, config.reloaded
  - Transport: Gorilla WebSocket (ws:// protocol)

## Container Lifecycle Integration

**Incus Integration Points:**
- Launch: `incus launch [image] [container]` with optional `--ephemeral`
- Stop: `incus stop [container]` (graceful) or force delete
- Delete: `incus delete [container]`
- Config: `incus config set/get [container] [key] [value]`
- Devices: `incus config device add [container] [name] disk [properties]`
- Snapshots: `incus snapshot` commands for session preservation
- Monitoring: `incus monitor --type lifecycle --format json` (event stream)
- Project/Groups: Optional project isolation, resource group management

## Security & Permission Isolation

**Workspace Protection:**
- Read-only mounting of sensitive paths:
  - `.git/hooks` - Prevent git hook modification
  - `.git/config` - Prevent git config manipulation
  - `.husky` - Husky hooks directory
  - `.vscode` - VS Code settings that can execute code
  - Configurable via `security.protected_paths` in config
  - Disable via `security.disable_protection: true`

**Container Isolation:**
- Unprivileged container execution
- UID/GID mapping for workspace access
- Ephemeral or persistent container options
- Resource limits: CPU, memory, disk (Incus config)

---

*Integration audit: 2026-03-17*
