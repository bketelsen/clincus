# Testing Patterns

**Analysis Date:** 2026-03-17

## Test Framework

**Runner:**
- Go: Standard `testing` package (built-in)
- Config: `go test -race` in `Makefile`
- No external test framework (no testify, ginkgo, etc.)

**Assertion Library:**
- Manual assertions using `if` and error reporting via `t.Error()`, `t.Errorf()`, `t.Fatal()`, `t.Fatalf()`

**Run Commands:**
```bash
make test              # Run all tests with -race flag
go test -race -v ./...  # Verbose with race detector
go test ./internal/config  # Single package
go test -run TestName ./...  # Specific test
```

## Test File Organization

**Location:**
- Co-located: `{file}.go` has corresponding `{file}_test.go` in same package and directory
- Example: `config.go` → `config_test.go` in `internal/config/`

**Naming:**
- Unit tests: `{name}_test.go` (e.g., `config_test.go`, `terminal_sanitize_test.go`)
- Integration tests: `{name}_integration_test.go` (e.g., `context_integration_test.go`, `resume_integration_test.go`)
- Test functions: `TestFunctionName` (e.g., `TestGetDefaultConfig`, `TestSanitizeTerm`)

**Structure:**
```
internal/config/
├── config.go
├── config_test.go        # Tests for config.go
├── loader.go
├── loader_test.go        # Tests for loader.go
├── watcher.go
├── watcher_test.go
└── reload_test.go
```

## Test Structure

**Suite Organization:**

Most tests follow a table-driven pattern:

```go
func TestExpandPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expand tilde",
			input:    "~/test",
			expected: filepath.Join(homeDir, "test"),
		},
		{
			name:     "no expansion needed",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Patterns:**
- Setup: Create test data/fixtures directly in test or use `t.TempDir()` for filesystem tests
- Teardown: Use `t.Cleanup()` or `defer` to remove temporary files and cleanup containers
- Assertions: Direct comparison with `if got != expected` followed by `t.Errorf()`
- Subtests: Use `t.Run(tt.name, func(t *testing.T) { ... })` for table-driven tests

## Mocking

**Framework:** No mocking framework used (no `testify/mock`, `golang/mock`, etc.)

**Patterns:**

For unit tests: Create real instances and test them directly:
```go
func TestOpencodeTool_Basics(t *testing.T) {
	oc := NewOpencode()

	if oc.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", oc.Name(), "opencode")
	}
}
```

For functions with side effects: Use temporary directories:
```go
func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	os.WriteFile(configPath, []byte(`[dashboard]
port = 8080`), 0o644)

	// Load and test
}
```

For environment-dependent tests: Temporarily override environment:
```go
func TestLoadFromEnv(t *testing.T) {
	oldImage := os.Getenv("CLINCUS_IMAGE")
	os.Setenv("CLINCUS_IMAGE", "test-image")
	defer func() {
		if oldImage == "" {
			os.Unsetenv("CLINCUS_IMAGE")
		} else {
			os.Setenv("CLINCUS_IMAGE", oldImage)
		}
	}()
	// Test code
}
```

**What to Mock:**
- Skip: Don't mock filesystem operations; use `t.TempDir()` instead
- Skip: Don't mock external tools; use skip conditions in integration tests instead

**What NOT to Mock:**
- Real containers/Incus calls: Use integration tests with skip conditions
- Configuration merging: Test the real merge logic
- ID generation: Test the real UUID generation

## Fixtures and Factories

**Test Data:**

No fixture files; test data created inline:

```go
func TestConfigMerge(t *testing.T) {
	base := GetDefaultConfig()
	base.Defaults.Image = "base-image"

	other := &Config{
		Defaults: DefaultsConfig{
			Image: "other-image",
		},
	}

	base.Merge(other)
	// Assert
}
```

**Helpers:**

Small helper functions for repeated setup:
```go
func ptrBool(b bool) *bool {
	return &b
}

// Used in test
tests := []struct {
	expected *bool
}{
	{
		expected: ptrBool(true),
	},
}
```

**Location:**
- Inline in test functions
- No separate `fixtures/` or `testdata/` directories for unit tests
- Large test datasets would go in `testdata/` directory (none currently)

## Coverage

**Requirements:** Not enforced; no coverage threshold configured

**View Coverage:**
```bash
go test -cover ./...         # Quick coverage summary
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # HTML report
```

## Test Types

**Unit Tests:**
- Scope: Single function or small set of related functions
- Approach: Fast, no external dependencies (use temp directories for files)
- Located in: `*_test.go` files
- Examples: `TestExpandPath`, `TestSanitizeTerm`, `TestGenerateSessionID`

**Integration Tests:**
- Scope: Multiple components working together; often involve Incus/container operations
- Approach: Slower; require external tools (incus binary); use skip conditions
- Located in: `*_integration_test.go` files
- Preconditions: Check for incus binary, daemon availability, and required images
- Examples:
  ```go
  func TestResumeCommand_FrozenContainer(t *testing.T) {
      if _, err := exec.LookPath("incus"); err != nil {
          t.Skip("incus not found, skipping integration test")
      }
      if !container.Available() {
          t.Skip("incus daemon not running, skipping integration test")
      }
      exists, err := container.ImageExists("clincus")
      if err != nil || !exists {
          t.Skip("clincus image not found, skipping integration test (run 'clincus build' first)")
      }
      // Test code
  }
  ```

**E2E Tests:**
- Not used in Go; Python integration tests in `tests/` directory (pytest) require Incus + built image

## Common Patterns

**Async Testing (N/A):**
- Go uses no async/await; context cancellation uses `context.WithTimeout()` and `t.Cleanup()`

**Error Testing:**

Test error conditions explicitly:
```go
func TestGenerateSessionID(t *testing.T) {
	id, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() failed: %v", err)
	}
	// Assert id format
}
```

Test error messages contain expected content:
```go
func TestResumeCommand_NotFrozen(t *testing.T) {
	err := resumeContainer(containerName)
	if err == nil {
		t.Error("Expected error when resuming non-frozen container, got nil")
	}

	if !strings.Contains(err.Error(), "not frozen") {
		t.Errorf("Expected error message about 'not frozen', got: %v", err)
	}
}
```

**Context Testing:**

Test context cancellation behavior:
```go
func TestIncusOutputContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err = IncusOutputContext(ctx, "exec", containerName, "--", "sleep", "30")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error from cancelled context, got nil")
	}
	if elapsed > 5*time.Second {
		t.Errorf("IncusOutputContext took %v, expected cancellation within ~2s", elapsed)
	}
}
```

**Table-Driven Subtests:**

Standard pattern for multiple scenarios:
```go
func TestWorkspaceHash(t *testing.T) {
	tests := []struct {
		name          string
		workspacePath string
		wantLength    int
	}{
		{
			name:          "simple path",
			workspacePath: "/home/user/project",
			wantLength:    8,
		},
		{
			name:          "empty path",
			workspacePath: "",
			wantLength:    8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := WorkspaceHash(tt.workspacePath)
			if len(hash) != tt.wantLength {
				t.Errorf("WorkspaceHash() returned length %d, want %d", len(hash), tt.wantLength)
			}
		})
	}
}
```

## Integration Test Patterns

**Cleanup:**

Use `t.Cleanup()` to ensure resources are freed:
```go
t.Cleanup(func() {
	_ = mgr.Stop(true)
	_ = mgr.Delete(true)
})
```

Also clean up before test to handle previous crashes:
```go
if exists, _ := mgr.Exists(); exists {
	_ = mgr.Stop(true)
	_ = mgr.Delete(true)
}
```

**Skip Conditions:**

Always check prerequisites and skip gracefully:
```go
if _, err := exec.LookPath("incus"); err != nil {
	t.Skip("incus not found, skipping integration test")
}
if !container.Available() {
	t.Skip("incus daemon not running, skipping integration test")
}
exists, err := container.ImageExists("clincus")
if err != nil || !exists {
	t.Skip("clincus image not found, skipping integration test (run 'clincus build' first)")
}
```

**Timing:**

Use `time.Sleep()` when waiting for container state changes:
```go
time.Sleep(2 * time.Second)
```

Check elapsed time for performance assertions:
```go
start := time.Now()
// Operation
elapsed := time.Since(start)
if elapsed > 5*time.Second {
	t.Errorf("Operation took %v, expected < 5s", elapsed)
}
```

## Test Count

Current test suite: ~56 unit tests across 7 packages, plus integration tests.

Covered packages:
- `internal/config/`: 5+ test files (config_test, loader_test, watcher_test, reload_test)
- `internal/session/`: 5+ test files (id_test, naming_test, setup_test, security_test, mount_validator_test)
- `internal/terminal/`: sanitize_test, command_integration_test, sanitize_integration_test
- `internal/tool/`: opencode_test, copilot_test
- `internal/server/`: server_test
- `internal/container/`: context_integration_test, tmpfs_integration_test, workspace_path_integration_test
- `internal/cli/`: resume_integration_test

---

*Testing analysis: 2026-03-17*
