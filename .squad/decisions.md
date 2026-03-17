# Squad Decisions

## Active Decisions

### DEC-001: Go as the Backend Language

- **Status:** Accepted
- **Context:** Clincus needs to ship as a self-contained CLI tool that manages system-level container operations, mounts filesystems, and bridges PTY sessions. It must cross-compile for Linux and macOS (amd64/arm64) with zero runtime dependencies.
- **Decision:** Use Go (currently 1.25) as the sole backend language.
- **Rationale:** Go produces statically-linked binaries (`CGO_ENABLED=0`), eliminating runtime dependency management. `go:embed` allows bundling the entire Svelte frontend into the binary. The `os/exec`, `creack/pty`, and `context` packages provide first-class support for the process and terminal management clincus requires. The ecosystem offers mature libraries for CLI frameworks (Cobra), config parsing (BurntSushi/toml), and WebSocket servers (gorilla/websocket).

---

### DEC-002: Cobra for CLI Framework

- **Status:** Accepted
- **Context:** Clincus exposes 19+ subcommands (`shell`, `run`, `list`, `build`, `serve`, `snapshot`, etc.) with extensive global and per-command flags, persistent pre-run hooks for config loading, and automatic help/completion generation.
- **Decision:** Use `spf13/cobra` (v1.10.2) as the CLI framework.
- **Rationale:** Cobra is the de facto standard for Go CLIs (used by kubectl, Hugo, GitHub CLI). It provides subcommand routing, flag parsing with `pflag`, automatic shell completion generation, man page generation, and `PersistentPreRunE` hooks — all features clincus uses heavily. The default-command pattern (`clincus` with no subcommand runs `shell`) is cleanly expressed through Cobra's `RunE` on the root command.

---

### DEC-003: Fang v2 for CLI Styling and Version Handling

- **Status:** Accepted
- **Context:** The CLI needed styled help output, consistent version/commit display, and visual polish without custom formatting code. Cobra's built-in help and version output is plain text.
- **Decision:** Use `charm.land/fang/v2` (v2.0.1) to wrap Cobra command execution.
- **Rationale:** Fang integrates with the Charm Bracelet ecosystem (lipgloss v2 for TUI styling) and provides styled help and version output with minimal code — a single `fang.Execute()` call with `WithVersion` and `WithCommit` options replaces custom version formatting. Version, commit, and date are injected via ldflags at build time. Added in PR #5.

---

### DEC-004: TOML for Configuration

- **Status:** Accepted
- **Context:** Clincus needs a hierarchical configuration system supporting system-wide defaults, user preferences, per-project overrides, environment variables, and CLI flags. The config format must be human-readable and support nested structures (limits, mounts, profiles, security).
- **Decision:** Use TOML as the configuration format, parsed by `BurntSushi/toml` (v1.6.0).
- **Rationale:** TOML is more readable than YAML for configuration (no indentation-sensitivity bugs) and more expressive than JSON (supports comments, typed values, inline tables). The config hierarchy — `/etc/clincus/config.toml` → `~/.config/clincus/config.toml` → `.clincus.toml` → `$CLINCUS_CONFIG` → env vars → CLI flags — follows the XDG Base Directory Specification and Unix convention of increasing specificity. BurntSushi/toml is the reference Go implementation.

---

### DEC-005: Svelte 5 with Vite for the Frontend Dashboard

- **Status:** Accepted
- **Context:** The web dashboard needs to display container status, provide terminal access (xterm.js), and receive real-time updates via WebSocket. The frontend must be lightweight since it's embedded into the Go binary via `go:embed`.
- **Decision:** Use Svelte 5 with Vite 6 and TypeScript 5.7 for the frontend.
- **Rationale:** Svelte compiles to minimal vanilla JS (no virtual DOM runtime), keeping the embedded bundle small — critical since every byte ships inside the Go binary. Vite provides near-instant dev builds and optimized production output. Svelte 5's runes-based reactivity simplifies state management for real-time WebSocket data. The dashboard was inspired by [wingthing](https://github.com/ehrlich-b/wingthing). Key dependency: `@xterm/xterm` v6 for in-browser terminal emulation.

---

### DEC-006: go:embed for Single-Binary Distribution

- **Status:** Accepted
- **Context:** Clincus includes both a Go backend and a Svelte frontend. Distributing them separately would complicate installation, versioning, and deployment.
- **Decision:** Embed the built frontend assets into the Go binary using `go:embed` via a dedicated `webui/` package.
- **Rationale:** `//go:embed all:dist` in `webui/embed.go` bundles the entire Vite build output into the binary. The `clincus serve` command reads from `embed.FS` and serves it over HTTP. This yields a single self-contained binary with no external file dependencies. The `webui/dist/.gitkeep` file is tracked to satisfy `go:embed` requirements even when the dist directory is empty (pre-build).

---

### DEC-007: MkDocs Material for Documentation

- **Status:** Accepted
- **Context:** The project needs a documentation site with guides, reference pages, API docs, and architecture overviews. It should support dark/light themes, code highlighting, search, and be deployable to GitHub Pages.
- **Decision:** Use MkDocs with the Material theme, managed via `pyproject.toml`.
- **Rationale:** MkDocs Material provides instant navigation, search suggestions, dark/light theme switching, admonitions, tabbed content, and code copy buttons out of the box. Documentation is written in Markdown (low barrier for contributors). The `docs.yml` GitHub Actions workflow deploys to GitHub Pages on push. Python tooling (ruff for linting, pytest for integration tests) was already in the project, so `pyproject.toml` manages docs dependencies naturally.

---

### DEC-008: Incus as the Container Runtime

- **Status:** Accepted
- **Context:** Clincus needs container isolation for AI coding sessions with features like filesystem mounts with UID shifting, resource limits (CPU, memory, disk I/O), snapshots, and multi-container concurrency. The runtime must support both ephemeral and persistent containers.
- **Decision:** Use Incus (the community fork of LXD) as the container runtime, invoked via the `incus` CLI.
- **Rationale:** Incus provides system containers (not application containers like Docker), which means full OS environments where AI tools can install packages, run services, and operate naturally. Native cgroup2 integration gives fine-grained resource limits. Incus supports disk mounts with `shift=true` for UID mapping, stateful snapshots, and project-level isolation. The project wraps the `incus` CLI via `os/exec` (not a Go SDK) for simplicity and to stay compatible with Incus version changes. macOS compatibility is handled via `runtime.GOOS` checks with `sg` group wrapping on Linux.

---

### DEC-009: Ephemeral-by-Default Container Architecture

- **Status:** Accepted
- **Context:** AI coding sessions need clean, isolated environments to prevent state leakage between runs, but some workflows benefit from persistent tool installations (npm packages, conda environments).
- **Decision:** Containers are ephemeral by default (deleted after session ends), with an opt-in `--persistent` flag for reuse across sessions.
- **Rationale:** Ephemeral containers ensure reproducibility — each session starts from a known image state, preventing "works on my container" problems. Session metadata is persisted to disk regardless, enabling resume workflows. Persistent mode is available for power users who need installed tooling to survive across sessions. The container is always launched as non-ephemeral internally (to allow session data saving even after `shutdown` from within), with cleanup handling deletion for non-persistent sessions.

---

### DEC-010: Multi-Slot Concurrent Sessions

- **Status:** Accepted
- **Context:** Users may want to run multiple AI coding sessions simultaneously on the same workspace (e.g., Claude in slot 1, Aider in slot 2) or on different workspaces.
- **Decision:** Support up to 10 concurrent session slots per workspace, with container names derived from `WORKSPACE_HASH-SLOT-TOOL`.
- **Rationale:** Slot-based naming (`a1b2c3d4-1-claude`, `a1b2c3d4-2-aider`) provides deterministic, collision-free container names. `AllocateSlot()` finds the first free slot, allowing parallel sessions without coordination. The default of 10 slots balances concurrency with resource constraints. This design supports the primary use case of comparing AI tool outputs side-by-side on the same codebase.

---

### DEC-011: Layered Configuration Hierarchy

- **Status:** Accepted
- **Context:** Different deployment contexts need different defaults — a corporate sysadmin sets resource limits system-wide, a user configures their preferred AI tool, and a project pins a specific container image.
- **Decision:** Configuration loads in order of increasing specificity: built-in defaults → `/etc/clincus/config.toml` → `~/.config/clincus/config.toml` → `.clincus.toml` → `$CLINCUS_CONFIG` → `CLINCUS_*` env vars → CLI flags.
- **Rationale:** This follows Unix convention and the XDG Base Directory Specification. Each layer merges into the previous (non-zero values override), so partial configs work naturally. CLI flags use `cmd.Flags().Changed()` to only override when explicitly set, preventing default flag values from clobbering config file settings. Environment variables follow a `CLINCUS_` prefix convention for 12-factor app compatibility.

---

### DEC-012: GoReleaser Pro for Release and Packaging

- **Status:** Accepted
- **Context:** Clincus targets multiple platforms (linux/darwin × amd64/arm64) and distribution formats (binary archives, deb, rpm, apk). Releases need cryptographic signing, build provenance attestation, and changelog generation.
- **Decision:** Use GoReleaser Pro (~v2) for the full release pipeline.
- **Rationale:** GoReleaser Pro provides cross-compilation, nfpm-based package generation (deb/rpm/apk with proper file placement for completions and man pages), Cosign signing for supply chain security, and nightly build support — all from a single `.goreleaser.yaml`. Pre-release hooks run `make web`, `make completions`, and `make manpages` to ensure all artifacts are current. The Pro tier is required for features like `nightly` builds and advanced hooks.

---

### DEC-013: Conventional Commits and git-cliff for Changelog

- **Status:** Accepted
- **Context:** The project needs automated, consistent changelog generation tied to the release process. Manual changelog maintenance doesn't scale and is error-prone.
- **Decision:** Enforce conventional commit messages (`feat:`, `fix:`, `docs:`, etc.) and use git-cliff for changelog generation.
- **Rationale:** Conventional commits provide machine-parseable commit history. git-cliff (configured in `cliff.toml`) groups commits by type (Added, Fixed, Documentation, etc.), formats them with Tera templates, and outputs `CHANGELOG.md`. The release workflow runs git-cliff automatically and commits the updated changelog back to `main`. GoReleaser also parses conventional commits for GitHub release notes. This eliminates manual changelog editing while keeping a human-readable history. Branch naming convention (`feat/`, `fix/`, `docs/`) mirrors commit types for consistency.

---

### DEC-014: GitHub Actions for CI/CD

- **Status:** Accepted
- **Context:** The project needs automated build verification, test execution, linting, documentation deployment, and release publishing.
- **Decision:** Use GitHub Actions for all CI/CD workflows.
- **Rationale:** GitHub Actions integrates natively with the GitHub repository, providing free CI for public repos. The CI pipeline (`ci.yml`) runs golangci-lint v2, builds the Svelte frontend, compiles Go, runs tests with race detection and coverage, and lints Python tests with ruff. The release workflow (`release.yml`) chains CI verification, GoReleaser Pro execution, Cosign signing, build attestation (SLSA level 3), and changelog generation. The docs workflow (`docs.yml`) deploys MkDocs to GitHub Pages. Additional squad workflows handle AI team coordination (heartbeat, issue triage, label sync).

---

### DEC-015: Squad AI Team Structure

- **Status:** Accepted
- **Context:** The project uses AI coding assistants extensively. Without structure, AI contributions lack consistency in architecture, testing, and documentation standards.
- **Decision:** Organize AI agents into a "Squad" team with specialized roles, charters, and a coordinator for work routing.
- **Rationale:** The squad structure (defined in `.squad/`) assigns clear ownership: Deckard (Lead — architecture, code review), Batty (Backend — Go implementation), Rachael (Frontend — Svelte dashboard), Pris (QA — testing), Gaff (DevOps — CI/CD, releases), Scribe (Session logging), and Ralph (Work monitoring). Each agent has a charter defining expertise, boundaries, and handoff rules. The coordinator routes work to the appropriate agent and enforces review gates. This prevents AI agents from making changes outside their domain and ensures architectural consistency. GitHub Actions workflows automate squad coordination (heartbeat checks, issue assignment, triage).

---

### DEC-016: Security-First Mount and Path Protection

- **Status:** Accepted
- **Context:** AI coding tools running inside containers have write access to mounted workspaces. Malicious or accidental writes to sensitive paths (`.git/hooks`, `.husky`, `.vscode`) could compromise the host or inject code.
- **Decision:** Implement a protected paths system that mounts security-sensitive directories as read-only by default, configurable via `[security].protected_paths` in config.
- **Rationale:** Git hooks (`.git/hooks`) are a known attack vector — a compromised hook runs arbitrary code on the next `git commit`. By default, clincus protects these paths while allowing users to opt into writable hooks via `[git].writable_hooks`. The mount validator (`internal/session/mount_validator`) enforces these restrictions before container launch. This defense-in-depth approach protects users who may not be aware of the risks.

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
