# CLAUDE.md

## Project Overview

Clincus is a CLI tool and web dashboard for managing AI coding tool sessions in Incus containers. It provides container isolation, session persistence, workspace mounting, snapshots, and resource limits.

Derived from [code-on-incus](https://github.com/mensfeld/code-on-incus). Web dashboard inspired by [wingthing](https://github.com/ehrlich-b/wingthing).

## Tech Stack

- **Backend:** Go 1.24+, Cobra CLI, TOML config
- **Frontend:** Svelte 5, Vite, TypeScript
- **Docs:** MkDocs Material (docs/ directory)
- **Build:** Makefile, GoReleaser Pro, GitHub Actions
- **Packages:** deb, rpm, apk via GoReleaser nfpms

## Repository Structure

```
cmd/clincus/       — CLI entry point
internal/cli/      — Cobra command definitions
internal/config/   — TOML config loading and defaults
internal/container/ — Incus container management
internal/session/  — Session lifecycle (setup, cleanup, naming)
internal/server/   — Web dashboard HTTP/WebSocket server
internal/health/   — Health check system
internal/image/    — Container image builder
internal/limits/   — Resource limit enforcement
internal/terminal/ — PTY/tmux bridging
internal/tool/     — AI tool abstraction (Claude, opencode, Aider)
internal/cleanup/  — Orphan resource cleanup
web/               — Svelte 5 frontend source
webui/             — go:embed wrapper for built frontend assets
scripts/           — Build scripts
tests/             — Python integration tests (pytest)
docs/              — MkDocs Material documentation site
```

## Build Commands

```bash
make build       # Build frontend + Go binary
make web         # Build Svelte frontend only
make test        # Run Go unit tests
make lint        # Run golangci-lint (requires v2)
make install     # Install to ~/.local/bin
make completions # Generate shell completions
make manpages    # Generate man pages
make docs        # Build MkDocs site
make clean       # Remove build artifacts
```

## Documentation Requirements

**Every change that affects user-facing behavior MUST update documentation:**

- **README.md** — Update if adding/removing features, changing CLI commands, or modifying quick start steps
- **docs/** — Update the relevant MkDocs pages:
  - New CLI commands/flags → `docs/reference/cli.md`
  - Config changes → `docs/reference/config.md`
  - API changes → `docs/reference/api.md`
  - New features → appropriate guide in `docs/guides/`
  - Install/setup changes → `docs/getting-started/`
- **TODO.md** — Check off completed items, add new items as discovered
- **CHANGELOG.md** is auto-generated from conventional commits by git-cliff during releases — do NOT edit manually

Do not consider a feature complete until docs are updated. Include doc updates in the same commit or PR as the code change.

## Git Workflow

**Never commit directly to `main`.** All changes must go through a feature branch and pull request:

1. Create a feature branch: `git checkout -b <type>/<short-description>` (e.g., `feat/copilot-support`, `fix/session-cleanup`)
2. Make commits using [conventional commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `ci:`, `refactor:`, `test:`, etc.)
3. Push the branch and create a pull request against `main`
4. Ensure CI passes before merging

Branch naming convention: `<type>/<short-description>` where type matches the conventional commit type.

## Configuration

- User config: `~/.config/clincus/config.toml`
- Project config: `.clincus.toml` (in project root)
- System config: `/etc/clincus/config.toml`
- Env var override: `CLINCUS_CONFIG`

## Naming Conventions

- Binary name: `clincus`
- Container prefix: `clincus-` (configurable via `CLINCUS_CONTAINER_PREFIX`)
- Config directory: `~/.clincus/`
- Image alias: `clincus`
- Env var prefix: `CLINCUS_`

## Testing

- Go unit tests: `make test` (56 tests across 7 packages)
- Go integration tests: require running Incus with `clincus` image built
- Python integration tests: `pytest tests/` (require Incus + built image)
- Python linting: `ruff check tests/ && ruff format --check tests/`
- Go linting: `make lint` (golangci-lint v2 config in `.golangci.yml`)

## CI/CD

- `.github/workflows/ci.yml` — Build, test, lint on push/PR to main
- `.github/workflows/release.yml` — GoReleaser on `v*` tags
- `.github/workflows/docs.yml` — MkDocs deploy to GitHub Pages on docs changes
- golangci-lint uses v2 config format — requires `golangci-lint-action@v7` with a specific version (e.g. `version: v2.11`), not just `v2`

## Release Process

```bash
make bump  # Runs build, test, fmt, lint, then tags with svu and pushes
```

Or manually:

```bash
git tag -a v0.x.0 -m "v0.x.0 — description"
git push origin v0.x.0  # Triggers GoReleaser release workflow
```

## Verification

- After frontend/Svelte changes, run `make build` (not just `make web`) to verify the full `go:embed` integration works end-to-end
- After CI workflow changes, check that the golangci-lint version in `.github/workflows/ci.yml` matches the v2 config format — mismatches have caused repeated CI failures (e.g. `version: v2.11`, not just `v2`)
- After changes to `.gitignore` or embed directives, confirm `cmd/clincus/` is not accidentally excluded and `webui/dist/.gitkeep` is still tracked

## Web Dashboard Gotchas

- SPA routing fallback must not intercept requests for static assets (JS, CSS, images) — only serve `index.html` for non-file paths
- Containers need the `TERM` env var set for interactive sessions to work properly
- Exec commands must use the correct container user, not root by default

## Important Notes

- The `webui/dist/` directory needs a `.gitkeep` for `go:embed` — `make clean` removes built assets but the `.gitkeep` must remain tracked
- The `.gitignore` uses `/clincus` (with leading slash) to only ignore the binary at root, not the `cmd/clincus/` directory
- Do NOT rename `incus monitor` in `internal/server/ws_events.go` — that's the Incus CLI lifecycle monitoring command, not a clincus feature
