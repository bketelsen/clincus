# Technology Stack

**Analysis Date:** 2026-03-17

## Languages

**Primary:**
- Go 1.25.0 - Backend CLI and web server (`cmd/clincus/`, `internal/`)
- TypeScript 5.7.0 - Frontend type safety
- Svelte 5.0.0 - Web dashboard UI (`web/src/`)

**Secondary:**
- Python 3.11+ - Integration tests (`tests/`)
- Bash - Build scripts and shell completions (`scripts/`, `Makefile`)

## Runtime

**Environment:**
- Go runtime (statically compiled, CGO_ENABLED=0)
- Node.js 20 LTS - Development only (for building Svelte frontend)
- Linux kernel (primary), Darwin/macOS support via GoReleaser builds

**Package Manager:**
- Go modules (go.mod)
- npm 10+ (web/package.json)
- pip - Development only (pytest, ruff)

## Frameworks

**Core:**
- Cobra v1.10.2 - CLI framework with subcommands and shell completion
- Gorilla WebSocket v1.5.3 - WebSocket transport for terminal/event streaming

**Frontend:**
- @sveltejs/vite-plugin-svelte 5.0.0 - Svelte compilation
- Vite 6.0.0 - Frontend build tool and dev server
- xterm 6.0.0 + @xterm/addon-fit 0.11.0 - Terminal emulation UI

**Testing:**
- pytest - Python integration tests (`tests/`)
- Go's native testing package - Unit tests
- Ruff - Python linting and formatting

**Build/Dev:**
- Makefile - Primary build orchestration
- GoReleaser Pro v2 - Multi-platform binary release and packaging
- golangci-lint v2.11 - Go linting (v2 config format)
- gofmt/gofumpt - Go formatting
- git-cliff - Changelog generation from conventional commits
- cosign - Binary signing (GoReleaser integration)
- MkDocs Material - Documentation site building and hosting

## Key Dependencies

**Critical:**
- charm.land/fang/v2 v2.0.1 - Terminal UI styling (Charmbracelet)
- github.com/creack/pty v1.1.24 - PTY/terminal handling for session I/O
- github.com/BurntSushi/toml v1.6.0 - TOML config parsing
- github.com/spf13/cobra v1.10.2 - CLI framework (subcommands, completion, man pages)
- github.com/fsnotify/fsnotify v1.8.0 - File system watching for config hot-reload

**Infrastructure:**
- github.com/gorilla/websocket v1.5.3 - WebSocket protocol (events, terminal streaming)
- golang.org/x/sys - OS-specific system calls (process management)
- golang.org/x/sync - Synchronization primitives for goroutines

**Terminal/UI (Indirect):**
- Charmbracelet libraries (lipgloss, ultraviolet, x/ansi, x/term, x/termios, x/windows) - TUI rendering

## Configuration

**Environment:**
- Env var prefix: `CLINCUS_` (e.g., `CLINCUS_CONFIG`, `CLINCUS_CONTAINER_PREFIX`)
- Config format: TOML
- Config search path: `~/.config/clincus/config.toml` → `.clincus.toml` → `/etc/clincus/config.toml`
- `CLINCUS_CONFIG` env var can override search path

**Build:**
- `Makefile` - Primary build interface
- `.goreleaser.yaml` - Multi-platform binary builds, archiving, signing, nfpm packaging
- `web/vite.config.ts` - Svelte frontend build config
- `web/tsconfig.json` - TypeScript compilation for frontend
- `.eslintrc` - Not found (no frontend linting currently)
- `.golangci.yml` - Go linting v2 config (bodyclose, copyloopvar, dupl, errname, gocyclo, gosec, etc.)
- `pyproject.toml` - Python project config (ruff linter/formatter, pytest)

## Platform Requirements

**Development:**
- Go 1.25.0+
- Node.js 20 LTS
- Python 3.11+ (for tests)
- golangci-lint v2.11+
- make
- git (for version string generation)

**Production:**
- Linux (primary) or macOS
- Incus container manager (must be installed and running)
- User must be in `incus-admin` group (on Linux, for `sg` escalation)
- No external services required (fully self-contained)

---

*Stack analysis: 2026-03-17*
