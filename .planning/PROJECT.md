# Clincus — Codebase Cleanup & Testing

## What This Is

A cleanup milestone for the Clincus CLI/dashboard codebase. Addresses tech debt, known bugs, and test coverage gaps identified during codebase mapping. The goal is to strengthen the codebase foundation — refactor oversized functions, fix known bugs, and raise test coverage from 29% to 50%+.

## Core Value

Increase confidence in the codebase so that future feature work doesn't break existing functionality. Every change must be backed by tests.

## Requirements

### Validated

- ✓ CLI container session management (shell, run, list, kill, clean) — existing
- ✓ Web dashboard with REST API and WebSocket terminal — existing
- ✓ Configuration system with hot-reload and hierarchical loading — existing
- ✓ AI tool abstraction (Claude, Aider, OpenCode) — existing
- ✓ Container image building — existing
- ✓ Resource limits enforcement (CPU, memory, disk, processes, duration) — existing
- ✓ Session persistence and resume — existing
- ✓ Health check system — existing
- ✓ Orphan container cleanup — existing

### Active

**Tech Debt:**
- [ ] Refactor `Setup()` in `internal/session/setup.go` into smaller composable functions
- [ ] Refactor `internal/health/checks.go` to use structured output (JSON) instead of brittle string parsing
- [ ] Improve frontend API error handling with typed errors and Content-Type aware parsing

**Known Bugs:**
- [ ] Fix WebSocket client reconnection memory leak in `web/src/lib/ws.ts`
- [ ] Fix cleanup timing race condition in `internal/session/cleanup.go` (add exponential backoff)
- [ ] Fix config watcher not detecting newly-created files in `internal/config/watcher.go`

**Test Coverage:**
- [ ] Add tests for `internal/container/commands.go` (Incus CLI wrappers)
- [ ] Add tests for `internal/cli/shell.go` and `internal/cli/run.go` (command execution)
- [ ] Add tests for session setup error scenarios (partial mount failure, permission errors, invalid workspace)
- [ ] Add tests for frontend WebSocket reconnection cycle
- [ ] Add tests for config reload under active operations
- [ ] Achieve 50%+ test coverage across touched packages

### Out of Scope

- Performance optimizations (sequential health checks, event watcher backoff, terminal broadcast) — deferred to future milestone
- Security hardening (path canonicalization, credential masking, shell escaping) — separate security-focused milestone
- Missing critical features (rate limiting, multi-user access control, audit logging) — feature work, not cleanup
- Dependency migrations (Gorilla WebSocket replacement) — separate effort

## Context

- Codebase is ~13,000 lines of Go + Svelte frontend
- Current test coverage: 29% (3892 test lines / 13146 source lines)
- 56 existing tests across 7 packages
- `Setup()` function is 886 lines with a `nolint:gocyclo` directive
- `health/checks.go` is 1040 lines with repetitive string parsing
- 3 known bugs documented in codebase mapping CONCERNS.md
- All existing tests must continue to pass — no regressions

## Constraints

- **Tech stack**: Go 1.24+, Svelte 5, existing project structure — no architectural changes
- **Testing**: Go native testing package for backend, existing test patterns
- **Compatibility**: Refactors must not change external behavior (CLI flags, API endpoints, config format)
- **Coverage target**: 50%+ across packages that receive changes

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Address all 3 concern areas (tech debt, bugs, tests) in one milestone | They're interconnected — refactoring enables better testing, bug fixes need tests | — Pending |
| Exclude performance work | Keep scope focused on correctness and maintainability | — Pending |
| Target 50% coverage | Roughly double current 29%, achievable with planned additions | — Pending |

---
*Last updated: 2026-03-18 after initialization*
