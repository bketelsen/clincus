# Requirements: Clincus Codebase Cleanup & Testing

**Defined:** 2026-03-18
**Core Value:** Increase confidence in the codebase so future feature work doesn't break existing functionality

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Test Infrastructure

- [x] **TEST-01**: CommandRunner interface extracted in container package, replacing direct exec.Command calls in execIncusCommand/execIncusCommandContext
- [x] **TEST-02**: Unit tests for container/commands.go covering IncusExec, IncusOutput, Manager.Exists, Manager.Launch, Manager.Stop, and error scenarios
- [x] **TEST-03**: Unit tests for session setup error scenarios (partial mount failure, permission errors, invalid workspace path)
- [x] **TEST-04**: `go test -race` added to CI pipeline for concurrent packages (session, config, server)
- [ ] **TEST-05**: Overall test coverage reaches 50%+ across packages that receive changes

### Refactoring

- [x] **REFAC-01**: Setup() in internal/session/setup.go decomposed into 4-5 composable unexported functions across setup_container.go, setup_mounts.go, setup_postlaunch.go, setup_toolconfig.go
- [x] **REFAC-02**: Setup() orchestrator function reduced to ~80 lines calling extracted step functions
- [x] **REFAC-03**: Characterization tests written before each extraction to pin current behavior
- [x] **REFAC-04**: Health check functions CheckNetworkBridge and CheckIncusStoragePool migrated from string parsing to --format=json where supported
- [x] **REFAC-05**: Defensive fallback parser for health check commands that don't support --format=json
- [x] **REFAC-06**: Frontend ApiError class in web/src/lib/ with status code, Content-Type aware parsing, and typed error categories (auth, server, network)

### Bug Fixes

- [ ] **BUG-01**: Cleanup race condition in internal/session/cleanup.go fixed — replace fixed 500ms x 10 polling with context-based timeout and exponential backoff
- [x] **BUG-02**: WebSocket reconnection memory leak in web/src/lib/ws.ts fixed — onReconnect moved into new WebSocket's onopen handler, old connections properly closed
- [ ] **BUG-03**: Config watcher in internal/config/watcher.go fixed — newly-created config files detected when parent directory is watched
- [ ] **BUG-04**: Reproduction test written for each bug fix before the fix is applied
- [x] **BUG-05**: Frontend WebSocket reconnection test validating the memory leak fix doesn't regress

## v2 Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### Performance

- **PERF-01**: Health check system parallelized with goroutines + sync.WaitGroup
- **PERF-02**: Event watcher exponential backoff for restart (2s -> 4s -> 8s -> 30s)
- **PERF-03**: Terminal bridge routes output to requesting client only, not broadcast

### Security

- **SEC-01**: Path canonicalization with filepath.EvalSymlinks() before mounting
- **SEC-02**: Credential masking in CLI debug output
- **SEC-03**: Shell escaping validation in image builder

### Dependencies

- **DEP-01**: Gorilla WebSocket replacement with nhooyr.io/websocket or stdlib
- **DEP-02**: Add frontend linting (ESLint/Prettier)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Rate limiting on API endpoints | Feature work, not cleanup |
| Multi-user access control | Feature work, requires design |
| Audit logging | Feature work, separate milestone |
| 80%+ test coverage | Diminishing returns above 50% target |
| New CLI commands or flags | No external behavior changes in this milestone |
| Architecture changes | Internal decomposition only, no new packages or public APIs |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| TEST-01 | Phase 1 | Complete |
| TEST-02 | Phase 1 | Complete |
| TEST-03 | Phase 3 | Complete |
| TEST-04 | Phase 1 | Complete |
| TEST-05 | Phase 4 | Pending |
| REFAC-01 | Phase 3 | Complete |
| REFAC-02 | Phase 3 | Complete |
| REFAC-03 | Phase 3 | Complete |
| REFAC-04 | Phase 2 | Complete |
| REFAC-05 | Phase 2 | Complete |
| REFAC-06 | Phase 4 | Complete |
| BUG-01 | Phase 4 | Pending |
| BUG-02 | Phase 4 | Complete |
| BUG-03 | Phase 4 | Pending |
| BUG-04 | Phase 4 | Pending |
| BUG-05 | Phase 4 | Complete |

**Coverage:**
- v1 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0

---
*Requirements defined: 2026-03-18*
*Last updated: 2026-03-18 after roadmap creation*
