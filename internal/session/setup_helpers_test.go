package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// mockResponse holds the stdout and exit code for a mock command.
type mockResponse struct {
	stdout   string
	exitCode int
}

// responseEntry is an ordered pattern-response pair.
type responseEntry struct {
	substring string
	resp      mockResponse
	// sequence holds additional responses for successive matches.
	// When set, the first call returns resp, the second returns sequence[0], etc.
	// After exhausting the sequence, the last response is repeated.
	sequence []mockResponse
	hitCount int
}

// sessionMockRunner returns different output based on command arguments.
// It inspects the third element of cmdArgs (the "incus ..." command string)
// to match against registered responses via substring matching.
// Patterns are checked in the order they were registered; the first match wins.
type sessionMockRunner struct {
	mu       sync.Mutex
	entries  []*responseEntry
	fallback mockResponse
	calls    [][]string // records all calls for assertions
}

func newSessionMockRunner() *sessionMockRunner {
	return &sessionMockRunner{
		fallback: mockResponse{stdout: "", exitCode: 0},
	}
}

// on registers a response for commands containing the given substring.
// The key is matched against the full incus command string (cmdArgs[2]).
// Patterns are checked in registration order; first match wins.
func (m *sessionMockRunner) on(substring string, resp mockResponse) *sessionMockRunner {
	m.entries = append(m.entries, &responseEntry{substring: substring, resp: resp})
	return m
}

// onSequence registers a sequence of responses for a pattern.
// The first call matching this pattern returns responses[0], the second returns
// responses[1], etc. After the sequence is exhausted, the last response repeats.
func (m *sessionMockRunner) onSequence(substring string, responses ...mockResponse) *sessionMockRunner {
	if len(responses) == 0 {
		return m
	}
	entry := &responseEntry{
		substring: substring,
		resp:      responses[0],
	}
	if len(responses) > 1 {
		entry.sequence = responses[1:]
	}
	m.entries = append(m.entries, entry)
	return m
}

func (m *sessionMockRunner) matchResponse(cmdArgs []string) mockResponse {
	var cmdStr string
	if len(cmdArgs) >= 3 {
		cmdStr = cmdArgs[2]
	} else {
		cmdStr = strings.Join(cmdArgs, " ")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range m.entries {
		if strings.Contains(cmdStr, entry.substring) {
			hit := entry.hitCount
			entry.hitCount++
			if hit == 0 {
				return entry.resp
			}
			if entry.sequence != nil {
				idx := hit - 1
				if idx >= len(entry.sequence) {
					idx = len(entry.sequence) - 1
				}
				return entry.sequence[idx]
			}
			return entry.resp
		}
	}
	return m.fallback
}

func (m *sessionMockRunner) Command(cmdArgs []string) *exec.Cmd {
	m.mu.Lock()
	m.calls = append(m.calls, cmdArgs)
	m.mu.Unlock()
	resp := m.matchResponse(cmdArgs)
	cs := []string{"-test.run=TestSessionHelperProcess", "--"}
	cs = append(cs, resp.stdout, fmt.Sprintf("%d", resp.exitCode))
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func (m *sessionMockRunner) CommandContext(ctx context.Context, cmdArgs []string) *exec.Cmd {
	m.mu.Lock()
	m.calls = append(m.calls, cmdArgs)
	m.mu.Unlock()
	resp := m.matchResponse(cmdArgs)
	cs := []string{"-test.run=TestSessionHelperProcess", "--"}
	cs = append(cs, resp.stdout, fmt.Sprintf("%d", resp.exitCode))
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// callsContaining returns all recorded calls whose cmdArgs[2] contains the given substring.
func (m *sessionMockRunner) callsContaining(substring string) [][]string {
	var matched [][]string
	for _, call := range m.calls {
		if len(call) >= 3 && strings.Contains(call[2], substring) {
			matched = append(matched, call)
		}
	}
	return matched
}

// TestSessionHelperProcess is the subprocess entry point for sessionMockRunner.
// It is not a real test -- it exits immediately unless GO_WANT_HELPER_PROCESS=1.
func TestSessionHelperProcess(t *testing.T) {
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

// noopLogger returns a no-op logger for use in tests.
func noopLogger() func(string) {
	return func(string) {}
}
