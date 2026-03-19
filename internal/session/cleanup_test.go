package session

import (
	"strings"
	"testing"
	"time"

	"github.com/bketelsen/clincus/internal/container"
)

func TestCleanup_TimeoutKeepsContainerRunning(t *testing.T) {
	// This test verifies exponential backoff behavior without waiting for
	// the full 15s timeout. We mock Running() to return true for 2 calls,
	// then false. With exponential backoff (500ms + 1s + 2s = 3.5s),
	// elapsed should be well under the 15s context timeout.
	// Uses generous upper bound to account for -race detector overhead.

	mock := newSessionMockRunner()
	// Exists() uses --format=csv: container exists
	mock.on("--format=csv", mockResponse{
		stdout:   "test-container",
		exitCode: 0,
	})
	// Running() uses --format=json: return Running x2, then Stopped
	mock.onSequence("--format=json",
		mockResponse{stdout: `[{"name":"test-container","status":"Running"}]`, exitCode: 0},
		mockResponse{stdout: `[{"name":"test-container","status":"Running"}]`, exitCode: 0},
		mockResponse{stdout: `[{"name":"test-container","status":"Stopped"}]`, exitCode: 0},
	)
	// Delete succeeds
	mock.on("delete", mockResponse{stdout: "", exitCode: 0})
	defer container.SetRunner(mock)()

	var logged []string
	opts := CleanupOptions{
		ContainerName: "test-container",
		Logger:        func(msg string) { logged = append(logged, msg) },
	}

	start := time.Now()
	err := Cleanup(opts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the mock was called the expected number of times (2 Running + 1 Stopped = 3)
	jsonCalls := mock.callsContaining("--format=json")
	if len(jsonCalls) != 3 {
		t.Errorf("expected 3 Running() calls, got %d", len(jsonCalls))
	}

	// Should complete well before the 15s timeout
	if elapsed > 13*time.Second {
		t.Errorf("took too long (%v), expected to stop polling after container reports stopped", elapsed)
	}
}

func TestCleanup_StoppedContainerDeleted(t *testing.T) {
	mock := newSessionMockRunner()
	// Exists: container exists
	mock.on("--format=csv", mockResponse{
		stdout:   "test-stopped",
		exitCode: 0,
	})
	// Running: already stopped
	mock.on("--format=json", mockResponse{
		stdout:   `[{"name":"test-stopped","status":"Stopped"}]`,
		exitCode: 0,
	})
	// Delete succeeds
	mock.on("delete", mockResponse{stdout: "", exitCode: 0})
	defer container.SetRunner(mock)()

	var logged []string
	opts := CleanupOptions{
		ContainerName: "test-stopped",
		Logger:        func(msg string) { logged = append(logged, msg) },
	}

	err := Cleanup(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify delete was called
	deleteCalls := mock.callsContaining("delete")
	if len(deleteCalls) == 0 {
		t.Error("expected delete to be called for stopped container")
	}
}

func TestCleanup_TimeoutLogsKeptRunning(t *testing.T) {
	// This test waits for the full 15s timeout. Skip in short mode.
	if testing.Short() {
		t.Skip("skipping long timeout test in short mode")
	}

	mock := newSessionMockRunner()
	// Exists: container exists
	mock.on("--format=csv", mockResponse{
		stdout:   "test-stuck",
		exitCode: 0,
	})
	// Running: always running
	mock.on("--format=json", mockResponse{
		stdout:   `[{"name":"test-stuck","status":"Running"}]`,
		exitCode: 0,
	})
	defer container.SetRunner(mock)()

	var logged []string
	opts := CleanupOptions{
		ContainerName: "test-stuck",
		Logger:        func(msg string) { logged = append(logged, msg) },
	}

	err := Cleanup(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should log that container is kept running
	found := false
	for _, msg := range logged {
		if strings.Contains(msg, "keeping it alive") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'keeping it alive' log message, got: %v", logged)
	}
}
