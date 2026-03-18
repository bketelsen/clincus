# Phase 3: Setup Decomposition - Research

**Researched:** 2026-03-18
**Domain:** Go function decomposition, characterization testing, session package refactoring
**Confidence:** HIGH

## Summary

Phase 3 decomposes the 476-line `Setup()` function (886-line `setup.go` file) in `internal/session/setup.go` into independently testable step functions across 4 new files. The work is purely structural -- no behavioral changes, no new features, no API changes. The `SetupOptions` and `SetupResult` types remain stable; callers in `internal/cli/shell.go` and `internal/cli/run.go` are unaffected.

The primary challenge is testing. Setup() depends heavily on `container.Manager` methods which route through `container.CommandRunner` -- the same mock infrastructure established in Phase 1 and refined in Phase 2. Additionally, Setup() reads `/proc/mounts`, calls `os.Stat` on host paths, reads environment variables, and interacts with the filesystem. Characterization tests must mock the Incus layer via `container.SetRunner()` while managing filesystem dependencies through temp dirs and env var manipulation.

**Primary recommendation:** Use the `healthMockRunner` pattern from Phase 2 (multi-response command matching) rather than the simpler single-response `mockRunner` from Phase 1, since Setup() issues many sequential Incus commands with different expected outputs. Write characterization tests per extraction target, extract code to new files, verify tests still pass after each extraction.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **setup_container.go**: Steps 1-5 -- container naming, image check, container existence/state handling, container creation (init + UID mapping + workspace mount + tmpfs + mounts + security mounts + limits), and container start. This is the largest extraction (~200 lines).
- **setup_mounts.go**: Absorbs the existing `setupMounts()` helper plus workspace mount logic, security mount setup, and tmpfs configuration currently inline in Setup(). The `SetupSecurityMounts()` call stays in session package (it's already exported).
- **setup_postlaunch.go**: Steps 6-7 -- `waitForReady`, metadata labels, history recording, timeout monitor setup. These are post-start, pre-config operations.
- **setup_toolconfig.go**: Steps 9-11 -- session resume/restore, credential injection, CLI config setup. Absorbs the complex tool config branching logic. The existing `restoreSessionData()`, `injectCredentials()`, `setupCLIConfig()`, `setupHomeConfigFile()` helpers move here.
- Characterization tests: focused per extraction target, use CommandRunner mock, cover happy path + key error branches before extraction
- Error scenario prioritization: 3 named scenarios (partial mount failure, permission errors, invalid workspace path) get dedicated tests; additional error paths covered as part of characterization tests
- Orchestrator: `Setup()` becomes thin caller of `resolveContainer()`, `createAndConfigureContainer()`, `postLaunchSetup()`, `configureToolAccess()` (or similar)
- Each step function takes subset of `SetupOptions` + `SetupResult`; returns errors propagated by Setup()
- Existing exported types (`SetupOptions`, `SetupResult`) don't change

### Claude's Discretion
- Exact unexported function signatures and parameter grouping
- Whether to use step-specific param structs vs passing SetupOptions directly
- Helper function placement (e.g., `hasLimits`, `isColimaOrLimaEnvironment`, `buildJSONFromSettings` stay in setup.go or move)
- Test file organization (one test file per new source file, or consolidated)
- Extraction order within the implementation plan

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| REFAC-01 | Setup() decomposed into 4-5 composable unexported functions across setup_container.go, setup_mounts.go, setup_postlaunch.go, setup_toolconfig.go | File splitting boundaries locked in CONTEXT.md; code analysis confirms clean cut points at steps 1-5, mounts, steps 6-7, steps 9-11 |
| REFAC-02 | Setup() orchestrator function reduced to ~80 lines calling extracted step functions | Current Setup() is lines 115-476 (361 lines); thin orchestrator calling 4 step functions + logger init + result construction fits in ~80 lines |
| REFAC-03 | Characterization tests written before each extraction to pin current behavior | healthMockRunner pattern from Phase 2 supports multi-command mocking needed for Setup(); test strategy detailed below |
| TEST-03 | Unit tests for session setup error scenarios (partial mount failure, permission errors, invalid workspace path) | Three dedicated test functions using mockRunner responses; filesystem mocking via temp dirs for os.Stat-dependent paths |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| testing | stdlib | Go test framework | Project standard, no external test deps |
| os/exec | stdlib | Command execution mocking via TestHelperProcess | Established in Phase 1-2 |
| container.SetRunner() | internal | Inject mock CommandRunner for Incus calls | Phase 1 infrastructure, `defer SetRunner(mock)()` pattern |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os | stdlib | Temp dirs, env var manipulation for tests | Filesystem-dependent test scenarios |
| path/filepath | stdlib | Path construction in test assertions | Validating path arguments to mocked commands |
| strings | stdlib | Command string matching in healthMockRunner | Routing mock responses by Incus subcommand |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| TestHelperProcess pattern | testify/gomock | Project uses stdlib-only testing; subprocess pattern already proven in Phase 1-2 |
| Manual mock routing | Interface-based mocks for Manager | Would require refactoring Manager to interface; out of scope for this phase |

## Architecture Patterns

### Recommended Project Structure
```
internal/session/
├── setup.go               # Thin orchestrator (~80 lines) + SetupOptions/SetupResult types
├── setup_container.go     # resolveContainer() + createAndConfigureContainer()
├── setup_mounts.go        # setupMounts() + workspace mount + security mount + tmpfs helpers
├── setup_postlaunch.go    # postLaunchSetup(): waitForReady, metadata, history, timeout
├── setup_toolconfig.go    # configureToolAccess(): restore, inject, CLI config, home config
├── setup_test.go          # Existing isColimaOrLimaEnvironment tests (keep)
├── setup_container_test.go
├── setup_mounts_test.go
├── setup_postlaunch_test.go
├── setup_toolconfig_test.go
├── setup_helpers_test.go  # Shared mock infrastructure (healthMockRunner variant)
├── types.go               # MountConfig, MountEntry (unchanged)
├── security.go            # SetupSecurityMounts (unchanged)
└── ...                    # Other existing files unchanged
```

### Pattern 1: Multi-Response Mock Runner for Session Tests
**What:** A `sessionMockRunner` (adapted from Phase 2's `healthMockRunner`) that routes different responses based on Incus command substrings. Setup() issues 10+ different Incus commands in sequence; a single-response mock cannot support this.
**When to use:** Any test that exercises more than one Incus command in a call chain.
**Example:**
```go
// In setup_helpers_test.go
type sessionMockRunner struct {
    responses map[string]mockResponse
    fallback  mockResponse
    calls     [][]string
}

func newSessionMockRunner() *sessionMockRunner {
    return &sessionMockRunner{
        responses: make(map[string]mockResponse),
    }
}

func (m *sessionMockRunner) on(substring string, resp mockResponse) *sessionMockRunner {
    m.responses[substring] = resp
    return m
}

// Usage in test:
func TestCreateAndConfigureContainer_HappyPath(t *testing.T) {
    mock := newSessionMockRunner()
    mock.on("image list", mockResponse{stdout: `[{"aliases":[{"name":"clincus"}]}]`, exitCode: 0})
    mock.on("list ^clincus", mockResponse{stdout: "", exitCode: 0})  // container doesn't exist
    mock.on("init", mockResponse{stdout: "", exitCode: 0})
    mock.on("config device add", mockResponse{stdout: "", exitCode: 0})
    mock.on("start", mockResponse{stdout: "", exitCode: 0})
    defer container.SetRunner(mock)()
    // ... test body
}
```

### Pattern 2: Step Function Signature Convention
**What:** Each step function receives the full `SetupOptions` and mutates/returns a `*SetupResult`. This avoids creating intermediate param structs for what are package-internal functions.
**When to use:** All 4 extracted step functions.
**Example:**
```go
// setup_container.go
func resolveContainer(opts *SetupOptions, result *SetupResult) error {
    // Steps 1-3: container naming, image check, execution context
    // Populates result.ContainerName, result.Manager, result.Image,
    // result.RunAsRoot, result.HomeDir
    return nil
}

func createAndConfigureContainer(opts *SetupOptions, result *SetupResult) (skipLaunch bool, err error) {
    // Steps 4-5: container existence check, creation, configuration, start
    // Returns skipLaunch flag for downstream steps
    return false, nil
}
```

### Pattern 3: Characterization Test Before Extraction
**What:** Write a test that exercises the current inline code path, verifying exact outputs and error messages. Then extract the code to a new function. Run the test again -- it must pass unchanged.
**When to use:** Before every extraction.
**Example workflow:**
1. Write `TestResolveContainer_HappyPath` testing Setup() lines 126-164
2. Write `TestResolveContainer_ImageNotFound` testing the error path at line 152
3. Extract lines 126-164 to `resolveContainer()` in setup_container.go
4. Update Setup() to call `resolveContainer()`
5. Run tests -- must pass identically

### Anti-Patterns to Avoid
- **Testing through Setup() only:** Don't write one big Setup() test. Test each step function independently so failures are isolated.
- **Over-mocking OS dependencies:** Functions like `isColimaOrLimaEnvironment()` read `/proc/mounts`. Leave these as-is (tested in existing setup_test.go). Don't abstract OS calls behind interfaces just for this phase.
- **Changing error messages:** Characterization tests pin exact `fmt.Errorf` strings. Do not change error messages during extraction -- that defeats the purpose of characterization tests.
- **Breaking the skipLaunch flow:** The `skipLaunch` boolean controls whether steps 5-onwards create a new container or reuse an existing one. This must be threaded through the step function boundary cleanly. Recommend making it a return value from `createAndConfigureContainer()`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Incus command mocking | Custom process spawning | `container.SetRunner()` + TestHelperProcess | Already built in Phase 1, tested in Phase 2 |
| Multi-command routing | Sequential call counting | `healthMockRunner`-style substring matching | Proven pattern, less brittle than call order |
| Temp directory management | Manual mkdir/cleanup | `t.TempDir()` | Auto-cleaned by testing framework |
| Env var save/restore | Manual save/restore | Pattern from existing `setup_test.go` | Already established in codebase |

## Common Pitfalls

### Pitfall 1: Package-Level Variable Pollution in Parallel Tests
**What goes wrong:** `container.SetRunner()` sets a package-level variable. If tests run in parallel, one test's mock overwrites another's.
**Why it happens:** `go test` runs test functions in the same package concurrently by default.
**How to avoid:** Do NOT use `t.Parallel()` in session tests that call `container.SetRunner()`. The `defer SetRunner(mock)()` pattern is only safe for sequential execution within a single test binary.
**Warning signs:** Flaky test failures, "unexpected exit code" from mock subprocess.

### Pitfall 2: TestHelperProcess Collision Across Packages
**What goes wrong:** Each package that uses the subprocess test pattern needs its own `TestHelperProcess` function. The session package tests must include this in their test file.
**Why it happens:** `TestHelperProcess` is matched by name via `-test.run=TestHelperProcess`.
**How to avoid:** Include `TestHelperProcess` in `setup_helpers_test.go`. Copy the exact pattern from `container/commands_test.go`.

### Pitfall 3: Filesystem Dependencies in Setup()
**What goes wrong:** Several code paths call `os.Stat()`, `os.ReadFile()`, `os.MkdirAll()`, `os.UserHomeDir()`. These fail or behave unpredictably in test environments.
**Why it happens:** Setup() was designed for real container environments, not unit testing.
**How to avoid:** For characterization tests, use `t.TempDir()` to create realistic directory structures. Set up credential files, config dirs, etc. in temp dirs. Override `os.Getenv` values where needed (CI, USER). Accept that some paths (like History recording) are best-effort and non-fatal.

### Pitfall 4: skipLaunch State Leaking Between Steps
**What goes wrong:** The `skipLaunch` boolean computed in step 4 affects steps 5, 9, 10, 11. If the extraction boundary doesn't preserve this, behavior changes silently.
**Why it happens:** `skipLaunch` is a local variable in Setup() with wide scope.
**How to avoid:** Make `skipLaunch` a return value from `createAndConfigureContainer()` and pass it explicitly to `configureToolAccess()`. Do not store it on SetupResult (it's an internal control flag, not a result).

### Pitfall 5: Extracting Functions That Mutate Result
**What goes wrong:** Several code paths set fields on `result` (ContainerWorkspacePath, TimeoutMonitor). If a step function receives `*SetupResult` but the extraction misses a field assignment, Setup() silently returns incomplete results.
**Why it happens:** result is built incrementally across 400+ lines.
**How to avoid:** For each extraction, audit every `result.X = Y` assignment in the target lines. Characterization tests should assert on ALL result fields, not just the "interesting" ones.

## Code Examples

### Setup() Orchestrator After Extraction
```go
// setup.go - thin orchestrator
func Setup(opts SetupOptions) (*SetupResult, error) {
    result := &SetupResult{}

    if opts.Logger == nil {
        opts.Logger = func(msg string) {
            fmt.Fprintf(os.Stderr, "[setup] %s\n", msg)
        }
    }

    // Step 1-3: Resolve container name, image, execution context
    if err := resolveContainer(&opts, result); err != nil {
        return nil, err
    }

    // Step 4-5: Check existing container, create/configure/start if needed
    skipLaunch, err := createAndConfigureContainer(&opts, result)
    if err != nil {
        return nil, err
    }

    // Step 6-7: Wait for ready, set metadata, record history, start timeout
    if err := postLaunchSetup(&opts, result); err != nil {
        return nil, err
    }

    // Step 9-11: Restore session, inject credentials, setup tool config
    configureToolAccess(&opts, result, skipLaunch)

    opts.Logger("Container setup complete!")
    return result, nil
}
```

### Session Mock Runner Infrastructure
```go
// setup_helpers_test.go
package session

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "testing"
)

type mockResponse struct {
    stdout   string
    exitCode int
}

type sessionMockRunner struct {
    responses map[string]mockResponse
    fallback  mockResponse
    calls     [][]string
}

func newSessionMockRunner() *sessionMockRunner {
    return &sessionMockRunner{
        responses: make(map[string]mockResponse),
    }
}

func (m *sessionMockRunner) on(substring string, resp mockResponse) *sessionMockRunner {
    m.responses[substring] = resp
    return m
}

func (m *sessionMockRunner) matchResponse(cmdArgs []string) mockResponse {
    var cmdStr string
    if len(cmdArgs) >= 3 {
        cmdStr = cmdArgs[2]
    } else {
        cmdStr = strings.Join(cmdArgs, " ")
    }
    for key, resp := range m.responses {
        if strings.Contains(cmdStr, key) {
            return resp
        }
    }
    return m.fallback
}

func (m *sessionMockRunner) Command(cmdArgs []string) *exec.Cmd {
    m.calls = append(m.calls, cmdArgs)
    resp := m.matchResponse(cmdArgs)
    cs := []string{"-test.run=TestHelperProcess", "--"}
    cs = append(cs, resp.stdout, fmt.Sprintf("%d", resp.exitCode))
    cmd := exec.Command(os.Args[0], cs...)
    cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
    return cmd
}

func (m *sessionMockRunner) CommandContext(ctx context.Context, cmdArgs []string) *exec.Cmd {
    m.calls = append(m.calls, cmdArgs)
    resp := m.matchResponse(cmdArgs)
    cs := []string{"-test.run=TestHelperProcess", "--"}
    cs = append(cs, resp.stdout, fmt.Sprintf("%d", resp.exitCode))
    cmd := exec.CommandContext(ctx, os.Args[0], cs...)
    cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
    return cmd
}

func TestHelperProcess(t *testing.T) {
    if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
        return
    }
    args := os.Args
    for i, arg := range args {
        if arg == "--" {
            args = args[i+1:]
            break
        }
    }
    if len(args) < 2 {
        os.Exit(1)
    }
    stdout := args[0]
    exitCode, _ := strconv.Atoi(args[1])
    fmt.Fprint(os.Stdout, stdout)
    os.Exit(exitCode)
}
```

### Dedicated Error Scenario Tests (TEST-03)
```go
// setup_container_test.go (partial)

func TestCreateAndConfigureContainer_PartialMountFailure(t *testing.T) {
    mock := newSessionMockRunner()
    mock.on("image list", mockResponse{
        stdout: `[{"aliases":[{"name":"clincus"}]}]`, exitCode: 0,
    })
    mock.on("list ^clincus", mockResponse{stdout: "", exitCode: 0})
    mock.on("init", mockResponse{stdout: "", exitCode: 0})
    // First device add (workspace) succeeds
    // But we can't easily distinguish two "config device add" calls with substring matching
    // Solution: make the second call fail by default
    mock.fallback = mockResponse{stdout: "", exitCode: 0}
    // Override specific mount failure
    mock.on("setupMounts", mockResponse{stdout: "", exitCode: 1})
    defer container.SetRunner(mock)()

    // ... test that partial mount failure returns error and doesn't start container
}

func TestSetup_InvalidWorkspacePath(t *testing.T) {
    // Use t.TempDir() to create a non-existent workspace path
    tmpDir := t.TempDir()
    nonExistent := filepath.Join(tmpDir, "does-not-exist")

    // Mock the Incus calls up to the workspace mount
    mock := newSessionMockRunner()
    mock.on("image list", mockResponse{
        stdout: `[{"aliases":[{"name":"clincus"}]}]`, exitCode: 0,
    })
    mock.on("list ^clincus", mockResponse{stdout: "", exitCode: 0})
    mock.on("init", mockResponse{stdout: "", exitCode: 0})
    mock.on("config device add", mockResponse{stdout: "Error: path not found", exitCode: 1})
    defer container.SetRunner(mock)()

    opts := SetupOptions{
        WorkspacePath: nonExistent,
        Logger:        func(string) {},
    }
    _, err := Setup(opts)
    if err == nil {
        t.Fatal("expected error for invalid workspace path")
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Monolithic Setup() with nolint:gocyclo | Decomposed step functions | This phase | Testability, maintainability |
| No unit tests for session setup | Characterization tests + error scenario tests | This phase | TEST-03 coverage |
| Single-response mockRunner | Multi-response healthMockRunner | Phase 2 | Enables complex multi-command test scenarios |

## Open Questions

1. **Mock response ordering for duplicate command patterns**
   - What we know: `sessionMockRunner` matches by substring. Some commands (e.g., multiple `config device add` calls) will match the same substring.
   - What's unclear: Whether ordered-response queues are needed, or if the fallback response is sufficient.
   - Recommendation: Start with substring matching + fallback. If tests need finer control for mount ordering, add a `onNth(substring, n, resp)` method. This is Claude's discretion per CONTEXT.md.

2. **Where to place utility functions**
   - What we know: `hasLimits()`, `isColimaOrLimaEnvironment()`, `buildJSONFromSettings()` are currently in setup.go.
   - What's unclear: Whether they belong in setup.go (generic helpers) or in their respective new files.
   - Recommendation: Keep `isColimaOrLimaEnvironment()` and `buildJSONFromSettings()` in setup.go (they're used across multiple steps). Move `hasLimits()` to setup_container.go (only used in createAndConfigureContainer). This is Claude's discretion per CONTEXT.md.

3. **ContainerWorkspacePath assignment location**
   - What we know: `result.ContainerWorkspacePath` is set inside the `!skipLaunch` block (line 286). When `skipLaunch` is true, it's never set (defaults to empty string).
   - What's unclear: Whether this is a pre-existing bug or intentional (callers may not need it when reusing containers).
   - Recommendation: Document in characterization tests. Do not fix during extraction -- preserving existing behavior is the goal.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None needed -- `go test ./internal/session/ -v` |
| Quick run command | `go test ./internal/session/ -count=1 -run TestSetup -v` |
| Full suite command | `go test ./internal/session/ -count=1 -v` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REFAC-01 | Step functions exist in 4 new files | unit | `go test ./internal/session/ -count=1 -run "Test(Resolve\|Create\|PostLaunch\|Configure)" -v` | Wave 0 |
| REFAC-02 | Setup() orchestrator under 100 lines | manual | Count lines: `sed -n '/^func Setup/,/^}/p' internal/session/setup.go \| wc -l` | N/A |
| REFAC-03 | Characterization tests pin behavior before extraction | unit | `go test ./internal/session/ -count=1 -run "TestCharacterize" -v` | Wave 0 |
| TEST-03 | Error scenarios: partial mount, permissions, invalid workspace | unit | `go test ./internal/session/ -count=1 -run "Test.*_(PartialMount\|Permission\|InvalidWorkspace)" -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/session/ -count=1 -v`
- **Per wave merge:** `go test ./... -count=1 -race`
- **Phase gate:** Full suite green before /gsd:verify-work

### Wave 0 Gaps
- [ ] `internal/session/setup_helpers_test.go` -- shared mock infrastructure (sessionMockRunner, TestHelperProcess)
- [ ] `internal/session/setup_container_test.go` -- characterization + error tests for container resolution and creation
- [ ] `internal/session/setup_mounts_test.go` -- characterization + error tests for mount setup
- [ ] `internal/session/setup_postlaunch_test.go` -- characterization tests for post-launch steps
- [ ] `internal/session/setup_toolconfig_test.go` -- characterization tests for tool config

## Sources

### Primary (HIGH confidence)
- `/workspace/internal/session/setup.go` -- Full Setup() function analyzed (886 lines, 476-line Setup())
- `/workspace/internal/container/commands_test.go` -- mockRunner and TestHelperProcess pattern
- `/workspace/internal/health/health_test_helpers_test.go` -- healthMockRunner multi-response pattern
- `/workspace/internal/container/runner.go` -- CommandRunner interface and SetRunner()
- `/workspace/internal/session/setup_test.go` -- Existing test coverage (isColimaOrLimaEnvironment only)
- `/workspace/internal/container/manager.go` -- Manager methods called from Setup()

### Secondary (MEDIUM confidence)
- `/workspace/.planning/phases/03-setup-decomposition/03-CONTEXT.md` -- User decisions and extraction boundaries

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- Go stdlib testing only, patterns established in Phase 1-2
- Architecture: HIGH -- File splitting boundaries locked by user, code analysis confirms clean cut points
- Pitfalls: HIGH -- Based on direct code analysis of Setup() dependencies and testing constraints

**Research date:** 2026-03-18
**Valid until:** 2026-04-18 (stable -- no external dependency changes)
