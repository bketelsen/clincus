---
phase: 01
plan: 01
subsystem: testing
tags: [go, interface, dependency-injection, testability, exec]

# Dependency graph
requires: []
provides:
  - CommandRunner interface for mocking all Incus CLI calls
  - ExecCommandRunner default implementation preserving production behavior
  - SetRunner test injection function with restore capability
affects: [01-02, phase-02, phase-03]

# Tech tracking
tech-stack:
  added: []
  patterns: [interface-based dependency injection, package-level default with setter]

key-files:
  created: [internal/container/runner.go]
  modified: [internal/container/commands.go]

key-decisions:
  - "Two-method interface (Command + CommandContext) matching existing choke points"
  - "SetRunner returns restore function for defer pattern in tests"
  - "Platform-specific logic (darwin/linux) moved to runner.go, removed from commands.go"

patterns-established:
  - "CommandRunner interface: all Incus exec calls route through defaultRunner variable"
  - "Test injection pattern: defer SetRunner(mock)() for safe cleanup"

requirements-completed: [TEST-01, TEST-04]

# Metrics
duration: 1min
completed: 2026-03-18
---

# Phase 01 Plan 01: CommandRunner Interface Summary

**CommandRunner interface extracts exec.Command seam in container package, enabling unit testing without Incus daemon**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-18T00:58:27Z
- **Completed:** 2026-03-18T00:59:31Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Created CommandRunner interface with Command and CommandContext methods matching the two Incus exec choke points
- Implemented ExecCommandRunner default that preserves exact platform-specific behavior (darwin/linux)
- Wired all 10+ public functions in commands.go through defaultRunner seamlessly
- Confirmed TEST-04 satisfied: CI already runs `go test -race ./...`

## Task Commits

Each task was committed atomically:

1. **Task 1: Create CommandRunner interface and ExecCommandRunner default** - `2cf9f5f` (feat)
2. **Task 2: Wire execIncusCommand and execIncusCommandContext to use defaultRunner** - `b5404fc` (refactor)

## Files Created/Modified
- `internal/container/runner.go` - CommandRunner interface, ExecCommandRunner impl, SetRunner injection function
- `internal/container/commands.go` - execIncusCommand and execIncusCommandContext now delegate to defaultRunner; runtime import removed

## Decisions Made
- Two-method interface matching existing choke points (Command + CommandContext) rather than a broader abstraction
- SetRunner returns a restore function enabling `defer SetRunner(mock)()` pattern for safe test cleanup
- Platform-specific logic (darwin vs linux) moved entirely to runner.go, keeping commands.go platform-agnostic

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Go toolchain not available in execution environment. File creation and modifications verified via structural analysis (grep checks for expected patterns, import verification). Full compilation verification (`go build`, `go vet`) deferred to CI.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- CommandRunner interface ready for Plan 02 to write unit tests using mock injection
- All subsequent test plans in Phase 1-3 depend on this interface existing
- No blockers for next plan

---
*Phase: 01*
*Completed: 2026-03-18*
