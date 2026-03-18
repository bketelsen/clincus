package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// mockRunner implements CommandRunner for unit tests.
// It uses the Go subprocess test pattern: commands re-invoke the test binary
// with TestHelperProcess, which writes mock stdout and exits with mock exit code.
type mockRunner struct {
	// stdout is what the mock command writes to stdout
	stdout string
	// exitCode is the exit code the mock command returns
	exitCode int
	// lastArgs records the args passed to Command/CommandContext for assertion
	lastArgs []string
}

func (m *mockRunner) Command(cmdArgs []string) *exec.Cmd {
	m.lastArgs = cmdArgs
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, m.stdout, fmt.Sprintf("%d", m.exitCode))
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func (m *mockRunner) CommandContext(ctx context.Context, cmdArgs []string) *exec.Cmd {
	m.lastArgs = cmdArgs
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, m.stdout, fmt.Sprintf("%d", m.exitCode))
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess is the subprocess entry point for mockRunner.
// It is not a real test — it exits immediately unless GO_WANT_HELPER_PROCESS=1.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Find args after "--"
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

// --- IncusExec tests ---

func TestIncusExec_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := IncusExec("list")
	if err != nil {
		t.Errorf("IncusExec(list) returned error: %v, want nil", err)
	}
}

func TestIncusExec_Failure(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 1}
	defer SetRunner(mock)()

	err := IncusExec("list")
	if err == nil {
		t.Error("IncusExec(list) returned nil error, want non-nil")
	}
}

// --- IncusOutput tests ---

func TestIncusOutput_Success(t *testing.T) {
	mock := &mockRunner{stdout: "  hello  \n", exitCode: 0}
	defer SetRunner(mock)()

	output, err := IncusOutput("list")
	if err != nil {
		t.Fatalf("IncusOutput(list) returned error: %v, want nil", err)
	}
	if output != "hello" {
		t.Errorf("IncusOutput(list) = %q, want %q", output, "hello")
	}
}

func TestIncusOutput_ExitError(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 2}
	defer SetRunner(mock)()

	_, err := IncusOutput("list")
	if err == nil {
		t.Fatal("IncusOutput(list) returned nil error, want ExitError")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("IncusOutput(list) error type = %T, want *ExitError", err)
	}
	if exitErr.ExitCode != 2 {
		t.Errorf("ExitError.ExitCode = %d, want 2", exitErr.ExitCode)
	}
}

// --- IncusExecQuiet tests ---

func TestIncusExecQuiet_Success(t *testing.T) {
	mock := &mockRunner{stdout: "some output", exitCode: 0}
	defer SetRunner(mock)()

	err := IncusExecQuiet("list")
	if err != nil {
		t.Errorf("IncusExecQuiet(list) returned error: %v, want nil", err)
	}
}

// --- IncusOutputRaw tests ---

func TestIncusOutputRaw_PreservesWhitespace(t *testing.T) {
	mock := &mockRunner{stdout: "  hello  \n", exitCode: 0}
	defer SetRunner(mock)()

	output, err := IncusOutputRaw("list")
	if err != nil {
		t.Fatalf("IncusOutputRaw(list) returned error: %v, want nil", err)
	}
	if output != "  hello  \n" {
		t.Errorf("IncusOutputRaw(list) = %q, want %q", output, "  hello  \n")
	}
}

// --- ContainerRunning tests ---

func TestContainerRunning_True(t *testing.T) {
	mock := &mockRunner{
		stdout:   `[{"name":"test-container","status":"Running"}]`,
		exitCode: 0,
	}
	defer SetRunner(mock)()

	running, err := ContainerRunning("test-container")
	if err != nil {
		t.Fatalf("ContainerRunning() returned error: %v", err)
	}
	if !running {
		t.Error("ContainerRunning() = false, want true")
	}
}

func TestContainerRunning_False(t *testing.T) {
	mock := &mockRunner{
		stdout:   `[{"name":"other","status":"Running"}]`,
		exitCode: 0,
	}
	defer SetRunner(mock)()

	running, err := ContainerRunning("test-container")
	if err != nil {
		t.Fatalf("ContainerRunning() returned error: %v", err)
	}
	if running {
		t.Error("ContainerRunning() = true, want false")
	}
}

func TestContainerRunning_EmptyList(t *testing.T) {
	mock := &mockRunner{
		stdout:   `[]`,
		exitCode: 0,
	}
	defer SetRunner(mock)()

	running, err := ContainerRunning("test-container")
	if err != nil {
		t.Fatalf("ContainerRunning() returned error: %v", err)
	}
	if running {
		t.Error("ContainerRunning() = true, want false")
	}
}

// --- Manager.Exists tests ---

func TestManagerExists_True(t *testing.T) {
	mock := &mockRunner{stdout: "my-container", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "my-container"}
	exists, err := mgr.Exists()
	if err != nil {
		t.Fatalf("Manager.Exists() returned error: %v", err)
	}
	if !exists {
		t.Error("Manager.Exists() = false, want true")
	}
}

func TestManagerExists_False(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "my-container"}
	exists, err := mgr.Exists()
	if err != nil {
		t.Fatalf("Manager.Exists() returned error: %v", err)
	}
	if exists {
		t.Error("Manager.Exists() = true, want false")
	}
}

// --- Manager.Stop tests ---

func TestManagerStop_Force(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Stop(true)
	if err != nil {
		t.Errorf("Manager.Stop(true) returned error: %v, want nil", err)
	}

	// Verify the args contain the stop command with --force
	args := strings.Join(mock.lastArgs, " ")
	if !strings.Contains(args, "stop") {
		t.Errorf("Manager.Stop(true) lastArgs = %v, want args containing 'stop'", mock.lastArgs)
	}
	if !strings.Contains(args, "--force") {
		t.Errorf("Manager.Stop(true) lastArgs = %v, want args containing '--force'", mock.lastArgs)
	}
}

// --- Manager.Launch tests ---

func TestManagerLaunch_Ephemeral(t *testing.T) {
	// LaunchContainer calls IncusExec once for launch, then enableDockerSupport
	// calls IncusExec 3 more times. The mock handles all 4 calls with exitCode=0.
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Launch("clincus", true)
	if err != nil {
		t.Errorf("Manager.Launch(clincus, true) returned error: %v, want nil", err)
	}
}

// --- buildIncusCommand tests ---

func TestBuildIncusCommand_Args(t *testing.T) {
	// Save and restore package-level vars
	origGroup := IncusGroup
	origProject := IncusProject
	defer func() {
		IncusGroup = origGroup
		IncusProject = origProject
	}()
	IncusGroup = "incus-admin"
	IncusProject = "default"

	result := buildIncusCommand("list", "--format=json")

	if len(result) != 3 {
		t.Fatalf("buildIncusCommand() returned %d args, want 3", len(result))
	}
	if result[0] != "incus-admin" {
		t.Errorf("buildIncusCommand()[0] = %q, want %q", result[0], "incus-admin")
	}
	if result[1] != "-c" {
		t.Errorf("buildIncusCommand()[1] = %q, want %q", result[1], "-c")
	}
	// The third element should be the full incus command with --project flag
	expected := "incus --project default list --format=json"
	if result[2] != expected {
		t.Errorf("buildIncusCommand()[2] = %q, want %q", result[2], expected)
	}
}

// --- IncusExecContext tests ---

func TestIncusExecContext_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	ctx := context.Background()
	err := IncusExecContext(ctx, "list")
	if err != nil {
		t.Errorf("IncusExecContext() returned error: %v, want nil", err)
	}
}

// --- IncusOutputContext with context ---

func TestIncusOutputContext_Success(t *testing.T) {
	mock := &mockRunner{stdout: "output-data", exitCode: 0}
	defer SetRunner(mock)()

	ctx := context.Background()
	output, err := IncusOutputContext(ctx, "info")
	if err != nil {
		t.Fatalf("IncusOutputContext() returned error: %v", err)
	}
	if output != "output-data" {
		t.Errorf("IncusOutputContext() = %q, want %q", output, "output-data")
	}
}
