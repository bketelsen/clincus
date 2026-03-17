# Codebase Concerns

**Analysis Date:** 2026-03-17

## Tech Debt

**Large functions with multiple responsibilities:**
- Issue: `Setup()` in `internal/session/setup.go` (886 lines) has high cyclomatic complexity (nolint:gocyclo) and handles container creation, mounting, user setup, CLI config copying, and limits validation sequentially. Changes to any step risk affecting all others.
- Files: `internal/session/setup.go:115-400+`
- Impact: Difficult to test individual setup steps, higher risk of cascading failures during initialization
- Fix approach: Refactor into smaller, composable functions (MountHandler, UserSetupHandler, LimitValidator) that can be tested and modified independently

**Health check system complexity:**
- Issue: `internal/health/checks.go` is 1040 lines with repetitive patterns for parsing YAML-like output from Incus commands. Manual parsing with string operations is brittle.
- Files: `internal/health/checks.go:200-400+`
- Impact: Changes to Incus output format (even whitespace) break health checks; hard to diagnose why a check failed
- Fix approach: Use structured output format (JSON) from Incus with proper unmarshaling; implement a YAML parser library

**API error handling lacks granularity:**
- Issue: Frontend API functions in `web/src/lib/api.ts` throw generic errors: `throw new Error(await res.text())`. Response body may be HTML error page, not JSON.
- Files: `web/src/lib/api.ts:5-24`
- Impact: Client receives opaque error messages; difficult to distinguish auth failures, server errors, network errors
- Fix approach: Parse response Content-Type header, decode JSON error objects separately, wrap with typed error classes

**Incomplete test coverage:**
- Issue: Test-to-code ratio is low (3892 test lines / 13146 source lines = 29%). Core packages like container/commands.go (513 lines) and cli/shell.go (637 lines) lack corresponding test files.
- Files: Across `internal/cli/`, `internal/container/`, `internal/image/`
- Impact: Regressions in terminal handling, container lifecycle management not caught until production
- Fix approach: Add integration tests for container lifecycle (launch→mount→exec→stop), CLI argument parsing, error scenarios

## Known Bugs

**WebSocket client reconnection may leak memory:**
- Symptoms: Long-running dashboard sessions accumulate connections; browser memory grows unbounded
- Files: `web/src/lib/ws.ts:44-83` (connectEvents), `internal/server/ws_events.go:65-73` (event watcher)
- Trigger: Keep dashboard open for > 1 hour with network interruption; check memory usage
- Workaround: Restart browser or refresh dashboard page
- Root cause: `connectEvents()` spawns new WebSocket and reconnect timer on each reconnect, but if connection drops during cleanup race, old references may persist

**Cleanup timing race condition in non-persistent mode:**
- Symptoms: Occasionally containers are not deleted after session exits; container appears orphaned
- Files: `internal/session/cleanup.go:81-87` (sleep polling loop)
- Trigger: User exits with `sudo shutdown 0` while clincus is processing exit; timing window ~500ms*10
- Workaround: Manual `clincus kill <container>`
- Root cause: 10 retries with 500ms sleep (5s total) may not be enough for slow containers; no exponential backoff

**Config watcher loses track of newly-created files:**
- Symptoms: User creates `~/.config/clincus/config.toml` while clincus serve is running; changes not picked up
- Files: `internal/config/watcher.go:54-71` (NewWatcher directory watch logic)
- Trigger: Start serve, wait 10s, create config file, edit it
- Workaround: Restart `clincus serve`
- Root cause: When parent directory is watched, file creation event fires but watching of the new file fails silently (log.Printf only)

## Security Considerations

**Insufficient input validation on API endpoints:**
- Risk: User-controlled paths in workspace mount operations could escape workspace boundary; container shell escape could lead to host filesystem access
- Files: `internal/cli/mount_parser.go:1-50` (path parsing), `internal/session/setup.go:58-78` (setupMounts)
- Current mitigation: Paths are validated as absolute/relative but no symlink resolution or canonicalization before mount
- Recommendations:
  - Canonicalize paths with `filepath.EvalSymlinks()` before mounting
  - Reject paths containing `..` after canonicalization
  - Whitelist allowed mount locations in config

**Credentials exposure in CLI debug output:**
- Risk: Users may accidentally paste `clincus --help` output or error messages containing tool config paths; these may be world-readable in shared shells
- Files: `internal/cli/run.go`, `internal/cli/shell.go` (command execution logging)
- Current mitigation: None; full command lines logged to stderr
- Recommendations:
  - Never log values of `--env` flags
  - Mask paths containing sensitive directories (`.ssh`, `.aws`, `.claude`)
  - Use separate log level for sensitive operations

**Shell escaping in image builder:**
- Risk: User-provided build script contains backticks or command substitution; executed with root privileges in container
- Files: `internal/image/builder.go:484+` (ExecCommand with build script)
- Current mitigation: Script is passed directly to `incus exec`, not wrapped in `sh -c`, but no validation of content
- Recommendations:
  - Parse build script as shell tokens before execution
  - Reject scripts with forbidden patterns (backticks, `$()`, pipe chains)
  - Require script to be signed or pre-approved

**WebSocket terminal connection lacks per-container authorization:**
- Risk: If container ID is guessable (UUID format) or leaked, attacker can connect to any container on the system without credentials
- Files: `internal/server/ws_terminal.go` (connection handler), `web/src/lib/ws.ts:3-37` (connectTerminal)
- Current mitigation: Assumes running on localhost only (127.0.0.1)
- Recommendations:
  - Verify WebSocket connection originates from container owner (check session directory ownership)
  - Bind to Unix socket instead of TCP, or add bearer token to WebSocket handshake
  - Log all terminal connections with timestamp and user

**Protected paths mount check is incomplete:**
- Risk: `.git/hooks` mounted read-only prevents hook execution but repo could contain other dangerous files (`.gitconfig`, `.git/objects` symlinks)
- Files: `internal/session/security.go` (ProtectPaths logic)
- Current mitigation: `.git/hooks`, `.vscode` mounted read-only
- Recommendations:
  - Mount entire `.git` directory read-only
  - Validate symlink targets don't escape mount point
  - Consider disabling all file execution in mounted workspace

## Performance Bottlenecks

**Health check system does sequential exec calls:**
- Problem: `health check` command spawns a new Incus process for each check (OS, Incus, permissions, image, network, IP forwarding, directory, disk space, etc.). Total runtime: 5-10 seconds on cold start.
- Files: `internal/health/checks.go` (CheckOS, CheckIncus, CheckPermissions, etc.)
- Cause: Each check is independent and calls `IncusExec()` or `IncusOutput()` separately
- Improvement path:
  - Batch Incus queries: run `incus list` once and cache results
  - Run independent checks in parallel with goroutines + sync.WaitGroup
  - Cache results for 30s to avoid repeated calls

**Event watcher spawns OS-level monitor process:**
- Problem: `clincus serve` spawns `incus monitor --type lifecycle --format json` which consumes CPU polling Incus daemon, even with no active sessions
- Files: `internal/server/ws_events.go:65-110` (watchIncusEvents, monitor loop with 2s retry)
- Cause: Lost events trigger restart after 2s; if Incus is slow, this becomes a busy loop
- Improvement path:
  - Implement exponential backoff for restart (2s → 4s → 8s → 30s)
  - Add metric/log when restart happens to detect systemwide issues
  - Consider websocket-based event API if available in Incus

**Terminal bridge broadcasts to all connected clients:**
- Problem: Output from single container terminal is re-broadcast to ALL connected WebSocket clients (including other users' sessions)
- Files: `internal/server/bridge.go:1-200+`, `internal/server/ws_events.go:54-62` (EventHub.Broadcast)
- Cause: Terminal pane output goes to EventHub which broadcasts to all clients indiscriminately
- Impact: Network traffic scales with number of clients; privacy concern if multiple users on same dashboard
- Improvement path:
  - Route terminal output only to clients who requested that specific container
  - Separate EventHub for terminal events vs. container lifecycle events
  - Add rate limiting: drop messages if client send channel is full

**Config file watcher debounces slowly:**
- Problem: Changes to config files trigger reload after debounce delay (likely 1-2s), causing perceived lag in dashboard response
- Files: `internal/config/watcher.go:36-120` (Watcher debounce logic, reload callback)
- Cause: 1s+ debounce prevents repeated reloads but delays detection
- Improvement path:
  - Reduce debounce to 100-200ms
  - Validate config before signaling change, only reload on valid config
  - Cache parsed config to avoid re-parsing on every change

## Fragile Areas

**Container lifecycle state management:**
- Files: `internal/session/setup.go`, `internal/session/cleanup.go`, `internal/container/manager.go`
- Why fragile: Setup/cleanup assume containers are always non-ephemeral and can be resumed, but state can diverge if Incus crashes or user manually deletes container. Session metadata (stored in `~/.clincus/sessions-*`) may reference deleted containers.
- Safe modification:
  - Always check container existence before assuming state
  - Wrap state-changing operations (mount, config set) in transactions
  - Add idempotency: operations should succeed if run twice
- Test coverage: Core container lifecycle (launch→mount→exec→stop) only tested in integration tests, not unit tests

**PTY/Terminal bridging:**
- Files: `internal/server/bridge.go`, `internal/terminal/terminal.go`
- Why fragile: Complex interactions between WebSocket messages, PTY resize events, and Incus exec. Missing/delayed resize events cause terminal width/height mismatches.
- Safe modification:
  - Test with varied terminal sizes (80x24, 200x50, 1x1)
  - Add explicit acks for resize operations
  - Validate terminal dimensions before applying to container
- Test coverage: Only tested manually or in E2E tests; no unit tests for resize logic

**Session persistence across restarts:**
- Files: `internal/cli/resume.go`, `internal/session/cleanup.go:111-150`, `internal/cli/persist.go`
- Why fragile: Session resume assumes tool config directory was successfully saved and can be restored. If pull fails silently or corruption occurs, resumed session loses all context (open files, workspace state, AI history).
- Safe modification:
  - Verify checksum of saved config directory
  - Test resume with corrupted config (should fail gracefully, not corrupt container)
  - Add dry-run mode to test resume without launching
- Test coverage: Resume tested only in integration tests

**WebSocket event broadcasting under load:**
- Files: `internal/server/ws_events.go:32-45` (eventClient receive loop), `internal/server/ws_events.go:54-62` (Broadcast with select/default)
- Why fragile: If client is slow to read from `c.send` channel, messages are dropped silently (default case). Client sees gaps in output/events.
- Safe modification:
  - Log when messages are dropped; expose as metric
  - Increase channel buffer only if necessary (signals actual bottleneck)
  - Consider backpressure: slow clients should disconnect
- Test coverage: Only end-to-end; no unit tests for concurrent message patterns

## Scaling Limits

**Maximum concurrent container sessions:**
- Current capacity: Incus can run ~50-100 containers on typical hardware (depends on CPUs, memory, disk)
- Limit: Beyond 50 concurrent containers, Incus daemon response time degrades significantly; health checks may timeout
- Scaling path:
  - Implement container pooling: pre-launch empty containers, allocate on demand
  - Monitor Incus daemon CPU/memory; reject new sessions if capacity exceeded
  - Add per-user session limits in config

**WebSocket client connection limits:**
- Current capacity: Single clincus serve process can handle ~500 concurrent WebSocket connections (goroutines)
- Limit: Beyond 500 clients, memory usage grows to 1GB+; GC pauses become noticeable
- Scaling path:
  - Use pub/sub broker (Redis, NATS) to distribute events across multiple serve instances
  - Implement connection pooling with load balancer
  - Add metrics: active connections, message throughput, queue depth

**Dashboard state refresh throughput:**
- Current capacity: ~10 state updates/sec before message queue backs up
- Limit: If many containers are starting/stopping simultaneously, event queue fills and clients see stale state
- Scaling path:
  - Batch event messages (group 10 events into 1 message)
  - Add event deduplication (don't send duplicate status events)
  - Implement client-side caching with conflict-free merge strategy

## Dependencies at Risk

**Gorilla WebSocket library:**
- Risk: Gorilla is deprecated and no longer maintained as of 2021. Security vulnerabilities may not be patched.
- Impact: If WebSocket protocol vulnerability discovered, clincus vulnerable to denial-of-service or message injection attacks
- Migration plan: Upgrade to stdlib `net/http` with WebSocket support (Go 1.21+) or evaluate `nhooyr.io/websocket` as drop-in replacement

**fsnotify file watcher:**
- Risk: Uses OS-specific syscalls (inotify on Linux, kqueue on macOS). Changes to kernel behavior can break watching.
- Impact: Config changes may not be detected if kernel drops events; users forced to restart serve
- Migration plan: Add timeout-based periodic polling as fallback; implement config version tracking to detect changes

**Svelte 5 frontend framework:**
- Risk: Early version with potential breaking changes. Type definitions may be incomplete.
- Impact: Frontend build failures, type safety gaps in component props
- Migration plan: Lock to specific Svelte version; consider migrating to stable SvelteKit for full SSR support if dashboard scales

**GoReleaser Pro:**
- Risk: Requires paid license; no open-source alternative available. Build pipeline depends on vendor.
- Impact: If vendor discontinues service, releases blocked
- Migration plan: Implement equivalent release automation using GitHub Actions + goreleaser-like multi-platform builds

## Missing Critical Features

**No rate limiting on API endpoints:**
- Problem: Any user on system can spam API endpoints (session creation, file operations) without throttling
- Blocks: Cannot safely expose dashboard on network-accessible port
- Impact: Denial-of-service risk; malicious actor could spam session creation and exhaust container resources

**No multi-user access control:**
- Problem: Dashboard assumes single user running on localhost. No concept of user identity, permissions, or role-based access.
- Blocks: Cannot safely share dashboard across team; all users see all sessions
- Impact: Security concern for shared systems; privacy concern for multi-tenant deployments

**No session isolation from host filesystem:**
- Problem: User can mount arbitrary host directories into containers; no restrictions on what can be accessed
- Blocks: Cannot safely run untrusted code (e.g., public LLM prompts) in containers
- Impact: Malicious code could access/modify sensitive host files via mounted paths

**No audit logging:**
- Problem: No record of who ran what command, when, with what result. Only container-level history exists.
- Blocks: Cannot investigate security incidents or trace state changes
- Impact: Compliance gap; difficult to debug production issues

## Test Coverage Gaps

**Container commands module untested:**
- What's not tested: `internal/container/commands.go` (513 lines) — Incus CLI wrappers (IncusExec, LaunchContainer, StopContainer, etc.)
- Files: `internal/container/commands.go`
- Risk: Changes to error handling or command format break core functionality silently until integration test
- Priority: High — these are critical paths

**CLI command execution untested:**
- What's not tested: `internal/cli/shell.go` (637 lines), `internal/cli/run.go` (338 lines) — User command execution with PTY/capture modes
- Files: `internal/cli/shell.go`, `internal/cli/run.go`
- Risk: Terminal handling, exit code propagation, signal handling not validated
- Priority: High — user-facing functionality

**Error scenarios in session setup:**
- What's not tested: What happens when mount fails partway through setup? When user doesn't have permission to mount? When workspace path is invalid?
- Files: `internal/session/setup.go:58-200`
- Risk: Partial state left in container; cleanup may fail or corrupt state
- Priority: Medium — relatively uncommon but high impact when triggered

**Frontend WebSocket reconnection:**
- What's not tested: WebSocket close/reopen cycle; message loss during reconnection; rapid reconnects
- Files: `web/src/lib/ws.ts:44-83` (connectEvents reconnection loop)
- Risk: Race condition on reconnect; stale subscriptions; memory leaks
- Priority: Medium — affects long-running dashboard sessions

**Config reload under active operations:**
- What's not tested: Config file changed while session is being created? While terminal is open? During cleanup?
- Files: `internal/config/watcher.go`, `internal/cli/serve.go` (config hot-reload)
- Risk: Inconsistent state if config changes mid-operation
- Priority: Low — requires specific timing but impacts reliability

---

*Concerns audit: 2026-03-17*
