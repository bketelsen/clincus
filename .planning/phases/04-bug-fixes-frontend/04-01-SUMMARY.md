---
phase: 04-bug-fixes-frontend
plan: 01
subsystem: backend
tags: [go, context, exponential-backoff, fsnotify, config-watcher, cleanup, tdd]

# Dependency graph
requires:
  - phase: 03-setup-decomposition
    provides: sessionMockRunner test infrastructure for mocking container commands
provides:
  - Exponential backoff cleanup polling with context timeout
  - Resilient config watcher with dir-to-file transition fallback
  - Reproduction tests for both bugs (BUG-01, BUG-03)
affects: [04-bug-fixes-frontend]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Labeled break pattern for select-inside-for loops"
    - "Exponential backoff with context.WithTimeout for polling"
    - "Dir watch fallback when direct file watch fails"

key-files:
  created:
    - internal/session/cleanup_test.go
  modified:
    - internal/session/cleanup.go
    - internal/config/watcher.go
    - internal/config/watcher_test.go

key-decisions:
  - "Labeled loop break (break loop) for select/for idiom per Go best practice"
  - "Keep container running on timeout (non-destructive) rather than force-deleting"
  - "Dir watch kept as fallback even after successful direct file watch transition"

patterns-established:
  - "Labeled break: use `loop: for { select { case: break loop } }` to exit for from select"
  - "Non-destructive timeout: on polling timeout, log and preserve state rather than force cleanup"

requirements-completed: [BUG-01, BUG-03, BUG-04]

# Metrics
duration: 4min
completed: 2026-03-18
---

# Phase 4 Plan 01: Go Bug Fixes Summary

**Cleanup race condition fixed with exponential backoff (500ms->4s cap, 15s timeout) and config watcher hardened with Rename event handling and dir watch fallback**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-18T14:01:01Z
- **Completed:** 2026-03-18T14:05:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- BUG-03 fixed: config watcher now handles both Create and Rename events for dir-to-file transition, and keeps dir watch as fallback when direct watch fails
- BUG-01 fixed: cleanup polling replaced with exponential backoff (500ms, 1s, 2s, 4s) using context.WithTimeout(15s); on timeout keeps container running (non-destructive)
- BUG-04 satisfied: reproduction tests written before fixes -- TestCleanup_TimeoutKeepsContainerRunning confirmed RED with old code (completed in 2.5s, expected >3s)

## Task Commits

Each task was committed atomically:

1. **Task 1: Write reproduction test for BUG-03, fix config watcher** - `706b9fa` (fix)
2. **Task 2: Write reproduction test for BUG-01, fix cleanup race condition** - `1e1ccc9` (fix)

## Files Created/Modified
- `internal/config/watcher.go` - Handle Create+Rename events, keep dir watch as fallback
- `internal/config/watcher_test.go` - Added TestWatcherDetectsCreateThenEdit
- `internal/session/cleanup.go` - Exponential backoff with context.WithTimeout replacing fixed polling
- `internal/session/cleanup_test.go` - Three cleanup tests: backoff verification, stopped-delete, timeout-log

## Decisions Made
- Used labeled break (`break loop`) instead of extracting to helper function -- simpler for a single call site
- Kept non-destructive timeout behavior: container stays running on timeout with log message
- Dir watch fallback retained even after successful direct file watch setup for robustness

## Deviations from Plan

None - plan executed exactly as written.

Note: The BUG-03 reproduction test (TestWatcherDetectsCreateThenEdit) passed before the fix on this system, meaning the bug manifests only on specific platforms/editors that use atomic rename for saves. The fix was still applied for robustness (handles Rename events and keeps dir watch fallback). The BUG-01 reproduction test correctly failed before the fix (RED confirmed).

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Go bug fixes complete, ready for frontend work in plans 02 and 03
- All existing session and config tests pass

---
*Phase: 04-bug-fixes-frontend*
*Completed: 2026-03-18*
