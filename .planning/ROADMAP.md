# Roadmap: Clincus Codebase Cleanup & Testing

## Overview

This milestone strengthens the Clincus codebase by introducing test infrastructure, refactoring the two largest problem areas (health checks and session setup), fixing three known bugs, and raising test coverage from 29% to 50%+. The phases follow a strict dependency chain: the CommandRunner interface (Phase 1) unlocks testability for health checks (Phase 2) and session setup (Phase 3), while bug fixes and frontend work (Phase 4) are independent but placed last to absorb any schedule variance.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Test Infrastructure** - Extract CommandRunner interface, add container package tests, enable race detector in CI
- [x] **Phase 2: Health Check Refactoring** - Migrate health checks from string parsing to YAML/structured output with defensive fallbacks (completed 2026-03-18)
- [ ] **Phase 3: Setup Decomposition** - Break 886-line Setup() into composable functions with characterization tests
- [ ] **Phase 4: Bug Fixes & Frontend** - Fix 3 known bugs with reproduction tests, add frontend error handling and coverage gate

## Phase Details

### Phase 1: Test Infrastructure
**Goal**: Developers can write unit tests for any package that depends on Incus CLI commands without requiring a running Incus daemon
**Depends on**: Nothing (first phase)
**Requirements**: TEST-01, TEST-02, TEST-04
**Success Criteria** (what must be TRUE):
  1. A CommandRunner interface exists in the container package and all Incus CLI calls route through it
  2. Container package has unit tests covering IncusExec, IncusOutput, Manager.Exists, Manager.Launch, Manager.Stop, and error paths -- all passing without Incus installed
  3. `go test -race` runs in CI for session, config, and server packages and the pipeline fails on detected races
**Plans:** 2 plans

Plans:
- [ ] 01-01-PLAN.md — Extract CommandRunner interface and wire into commands.go
- [ ] 01-02-PLAN.md — Unit tests for container package core functions using mock runner

### Phase 2: Health Check Refactoring
**Goal**: Health check results are parsed from structured data instead of brittle string matching, eliminating false failures across Incus versions
**Depends on**: Phase 1
**Requirements**: REFAC-04, REFAC-05
**Success Criteria** (what must be TRUE):
  1. CheckNetworkBridge and CheckIncusStoragePool use YAML output (from `incus profile show` and `incus network show`) and --bytes flag (for `incus storage info`) instead of brittle string parsing
  2. A defensive fallback parser handles parse failures gracefully, returning a warning status instead of crashing
  3. Unit tests verify both structured parsing and fallback paths using mock CommandRunner output
**Plans:** 2/2 plans complete

Plans:
- [ ] 02-01-PLAN.md — YAML parsing infrastructure, refactor CheckNetworkBridge with tests
- [ ] 02-02-PLAN.md — Refactor CheckIncusStoragePool with --bytes parsing and tests

### Phase 3: Setup Decomposition
**Goal**: Session setup logic is decomposed into independently testable functions so developers can verify setup behavior without running full container lifecycle
**Depends on**: Phase 1, Phase 2
**Requirements**: REFAC-01, REFAC-02, REFAC-03, TEST-03
**Success Criteria** (what must be TRUE):
  1. Setup() orchestrator in setup.go is under 100 lines and calls extracted step functions
  2. Four new files exist (setup_container.go, setup_mounts.go, setup_postlaunch.go, setup_toolconfig.go) with unexported step functions
  3. Characterization tests exist that were written before each extraction, pinning current behavior including error paths
  4. Session setup error scenarios (partial mount failure, permission errors, invalid workspace) have dedicated test coverage
**Plans:** 2 plans

Plans:
- [ ] 03-01-PLAN.md — Characterization tests and error scenario tests for Setup() code regions
- [ ] 03-02-PLAN.md — Extract step functions to 4 new files, reduce Setup() to thin orchestrator

### Phase 4: Bug Fixes & Frontend
**Goal**: Three known user-facing bugs are fixed with regression tests, frontend error handling is improved, and the coverage target is met
**Depends on**: Phase 1
**Requirements**: BUG-01, BUG-02, BUG-03, BUG-04, BUG-05, REFAC-06, TEST-05
**Success Criteria** (what must be TRUE):
  1. Cleanup race condition is fixed -- container cleanup uses context-based timeout with exponential backoff instead of fixed 500ms x 10 polling
  2. WebSocket reconnection no longer leaks event listeners -- onReconnect fires inside the new connection's onopen handler and old connections are properly closed
  3. Config watcher detects newly-created config files by watching the parent directory
  4. Each bug fix has a reproduction test that was written before the fix was applied
  5. Overall test coverage across packages that received changes in this milestone is 50% or higher
**Plans:** 3 plans

Plans:
- [ ] 04-01-PLAN.md — Go bug fixes: config watcher (BUG-03) and cleanup race (BUG-01) with reproduction tests
- [ ] 04-02-PLAN.md — Frontend: WebSocket reconnection fix (BUG-02), ApiError class (REFAC-06), frontend tests (BUG-05)
- [ ] 04-03-PLAN.md — Coverage gate: verify 50%+ coverage (TEST-05) and audit reproduction tests (BUG-04)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Test Infrastructure | 0/2 | Not started | - |
| 2. Health Check Refactoring | 2/2 | Complete   | 2026-03-18 |
| 3. Setup Decomposition | 1/2 | In progress | - |
| 4. Bug Fixes & Frontend | 0/3 | Not started | - |
