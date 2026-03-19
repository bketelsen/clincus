package session

import (
	"strings"
	"testing"

	"github.com/bketelsen/clincus/internal/config"
	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/tool"
)

func TestPostLaunch_WaitForReady_Success(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	// Verify echo ready was called (waitForReady succeeded)
	readyCalls := mock.callsContaining("echo ready")
	if len(readyCalls) == 0 {
		t.Error("Expected echo ready call from waitForReady")
	}
}

func TestPostLaunch_WaitForReady_Timeout(t *testing.T) {
	// Test waitForReady directly with maxRetries=2 to avoid 30-second timeout.
	mock := newSessionMockRunner()
	// Running() returns false (container never becomes ready)
	mock.on("--format=json", mockResponse{stdout: "[]", exitCode: 0})
	defer container.SetRunner(mock)()

	mgr := container.NewManager("test-timeout-container")
	err := waitForReady(mgr, 2, noopLogger())
	if err == nil {
		t.Fatal("expected timeout error from waitForReady, got nil")
	}
	if !strings.Contains(err.Error(), "container failed to become ready after 2 seconds") {
		t.Errorf("error = %q, want it to contain 'container failed to become ready after 2 seconds'", err.Error())
	}
}

func TestPostLaunch_MetadataLabels(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	claudeTool := tool.NewClaude()
	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Tool:          claudeTool,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	// Check that metadata labels were set
	expectedLabels := []string{
		"user.clincus.managed",
		"user.clincus.workspace",
		"user.clincus.tool",
		"user.clincus.persistent",
		"user.clincus.created",
	}
	for _, label := range expectedLabels {
		calls := mock.callsContaining(label)
		if len(calls) == 0 {
			t.Errorf("Expected config set call for %q, found none", label)
		}
	}
}

func TestPostLaunch_TimeoutMonitor(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	result, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		LimitsConfig: &config.LimitsConfig{
			Runtime: config.RuntimeLimits{
				MaxDuration: "1h",
			},
		},
		Logger: noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	if result.TimeoutMonitor == nil {
		t.Error("expected TimeoutMonitor to be non-nil when MaxDuration is set")
	} else {
		result.TimeoutMonitor.Stop()
	}
}
