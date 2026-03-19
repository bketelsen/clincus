package health

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// mockResponse holds the stdout and exit code for a mock command.
type mockResponse struct {
	stdout   string
	exitCode int
}

// healthMockRunner returns different output based on command arguments.
// It inspects the third element of cmdArgs (the "incus ..." command string)
// to match against registered responses.
type healthMockRunner struct {
	responses map[string]mockResponse
	fallback  mockResponse
	calls     [][]string // records all calls for assertions
}

func newHealthMockRunner() *healthMockRunner {
	return &healthMockRunner{
		responses: make(map[string]mockResponse),
	}
}

// on registers a response for commands containing the given substring.
// The key is matched against the full incus command string (cmdArgs[2]).
// Example: mock.on("profile show default", mockResponse{stdout: yamlOutput, exitCode: 0})
func (m *healthMockRunner) on(substring string, resp mockResponse) *healthMockRunner {
	m.responses[substring] = resp
	return m
}

func (m *healthMockRunner) matchResponse(cmdArgs []string) mockResponse {
	// cmdArgs is ["incus-admin", "-c", "incus --project default profile show default"]
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

func (m *healthMockRunner) Command(cmdArgs []string) *exec.Cmd {
	m.calls = append(m.calls, cmdArgs)
	resp := m.matchResponse(cmdArgs)
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, resp.stdout, fmt.Sprintf("%d", resp.exitCode))
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func (m *healthMockRunner) CommandContext(ctx context.Context, cmdArgs []string) *exec.Cmd {
	m.calls = append(m.calls, cmdArgs)
	resp := m.matchResponse(cmdArgs)
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, resp.stdout, fmt.Sprintf("%d", resp.exitCode))
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess is the subprocess entry point for healthMockRunner.
// It is not a real test -- it exits immediately unless GO_WANT_HELPER_PROCESS=1.
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
