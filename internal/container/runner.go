package container

import (
	"context"
	"os/exec"
	"runtime"
)

// CommandRunner abstracts command creation for testability.
// The default implementation uses exec.Command/exec.CommandContext;
// tests can inject a mock that returns pre-canned *exec.Cmd values.
type CommandRunner interface {
	// Command creates an *exec.Cmd for the given arguments.
	// On Linux the default impl wraps with sg; on macOS it uses sh -c.
	Command(cmdArgs []string) *exec.Cmd
	// CommandContext creates a context-aware *exec.Cmd.
	// WaitDelay is set by the caller (in execIncusCommandContext), not here.
	CommandContext(ctx context.Context, cmdArgs []string) *exec.Cmd
}

// ExecCommandRunner is the default CommandRunner that uses os/exec.
type ExecCommandRunner struct{}

func (r ExecCommandRunner) Command(cmdArgs []string) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		incusCmd := cmdArgs[2]
		return exec.Command("sh", "-c", incusCmd)
	}
	return exec.Command("sg", cmdArgs...)
}

func (r ExecCommandRunner) CommandContext(ctx context.Context, cmdArgs []string) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		incusCmd := cmdArgs[2]
		return exec.CommandContext(ctx, "sh", "-c", incusCmd)
	}
	return exec.CommandContext(ctx, "sg", cmdArgs...)
}

// defaultRunner is the package-level CommandRunner used by all Incus command functions.
// Override via SetRunner in tests.
var defaultRunner CommandRunner = ExecCommandRunner{}

// SetRunner replaces the package-level CommandRunner. Returns a restore function
// for use in tests: `defer SetRunner(mockRunner)()`.
func SetRunner(r CommandRunner) func() {
	old := defaultRunner
	defaultRunner = r
	return func() { defaultRunner = old }
}
