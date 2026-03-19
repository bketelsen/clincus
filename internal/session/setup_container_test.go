package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bketelsen/clincus/internal/container"
)

// imageListJSON returns a JSON string for image list mocking.
func imageListJSON(aliases ...string) string {
	type alias struct {
		Name string `json:"name"`
	}
	type image struct {
		Aliases []alias `json:"aliases"`
	}
	var images []image
	if len(aliases) > 0 {
		var as []alias
		for _, a := range aliases {
			as = append(as, alias{Name: a})
		}
		images = append(images, image{Aliases: as})
	}
	b, _ := json.Marshal(images)
	return string(b)
}

// containerListJSON returns JSON for container list (Running check) mocking.
func containerListJSON(name, status string) string {
	if name == "" {
		return "[]"
	}
	type ct struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	b, _ := json.Marshal([]ct{{Name: name, Status: status}})
	return string(b)
}

// setupHappyPathMock creates a mock configured for a successful Setup() run.
// It handles all standard commands: image list, container exists/running checks,
// init, config, start, waitForReady, and metadata labels.
func setupHappyPathMock(containerName, image string) *sessionMockRunner {
	mock := newSessionMockRunner()
	// Image exists -- "image list" matches ImageExists call
	mock.on("image list", mockResponse{stdout: imageListJSON(image), exitCode: 0})
	// Container does not exist (Exists() uses --format=csv --columns=n)
	mock.on("--format=csv", mockResponse{stdout: "", exitCode: 0})
	// Container running check (ContainerRunning uses list <name> --format=json)
	// After start, waitForReady needs this to return running.
	mock.on("--format=json", mockResponse{
		stdout:   containerListJSON(containerName, "Running"),
		exitCode: 0,
	})
	// echo ready for waitForReady
	mock.on("echo ready", mockResponse{stdout: "ready", exitCode: 0})
	return mock
}

func TestResolveContainer_HappyPath(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	result, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	if result.ContainerName != expectedName {
		t.Errorf("ContainerName = %q, want %q", result.ContainerName, expectedName)
	}

	if result.Image != ClincusImage {
		t.Errorf("Image = %q, want %q", result.Image, ClincusImage)
	}

	if result.RunAsRoot {
		t.Error("RunAsRoot = true, want false for clincus image")
	}

	if result.HomeDir != "/home/code" {
		t.Errorf("HomeDir = %q, want /home/code", result.HomeDir)
	}
}

func TestResolveContainer_ImageNotFound(t *testing.T) {
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: "[]", exitCode: 0})
	defer container.SetRunner(mock)()

	_, err := Setup(SetupOptions{
		WorkspacePath: t.TempDir(),
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err == nil {
		t.Fatal("expected error for missing image, got nil")
	}
	if !strings.Contains(err.Error(), "image 'clincus' not found") {
		t.Errorf("error = %q, want it to contain \"image 'clincus' not found\"", err.Error())
	}
}

func TestResolveContainer_CustomImage(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, "images:ubuntu/24.04")
	defer container.SetRunner(mock)()

	result, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Image:         "images:ubuntu/24.04",
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	if !result.RunAsRoot {
		t.Error("RunAsRoot = false, want true for non-clincus image")
	}

	if result.HomeDir != "/root" {
		t.Errorf("HomeDir = %q, want /root", result.HomeDir)
	}
}

func TestCreateContainer_ExistingRunningPersistent(t *testing.T) {
	containerName := ContainerName("/tmp/testws", 1)
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: imageListJSON("clincus"), exitCode: 0})
	// Container exists
	mock.on("--format=csv", mockResponse{stdout: containerName, exitCode: 0})
	// Container is running
	mock.on("--format=json", mockResponse{
		stdout:   containerListJSON(containerName, "Running"),
		exitCode: 0,
	})
	// echo ready
	mock.on("echo ready", mockResponse{stdout: "ready", exitCode: 0})

	defer container.SetRunner(mock)()

	result, err := Setup(SetupOptions{
		WorkspacePath: "/tmp/testws",
		Persistent:    true,
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	if result.ContainerName != containerName {
		t.Errorf("ContainerName = %q, want %q", result.ContainerName, containerName)
	}

	// Should NOT have called init (skip launch path)
	for _, call := range mock.calls {
		if len(call) >= 3 && strings.Contains(call[2], " init ") && !strings.Contains(call[2], "image") {
			t.Error("Expected no init calls (skip launch), but found init call")
			break
		}
	}
}

func TestCreateContainer_ExistingStoppedNonPersistent(t *testing.T) {
	containerName := ContainerName("/tmp/testws2", 1)
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: imageListJSON("clincus"), exitCode: 0})
	// Container exists
	mock.on("--format=csv", mockResponse{stdout: containerName, exitCode: 0})
	// First call: container is stopped (triggers delete + recreate path)
	// Subsequent calls: container is running (for waitForReady after start)
	mock.onSequence("--format=json",
		mockResponse{stdout: containerListJSON(containerName, "Stopped"), exitCode: 0},
		mockResponse{stdout: containerListJSON(containerName, "Running"), exitCode: 0},
	)
	// echo ready
	mock.on("echo ready", mockResponse{stdout: "ready", exitCode: 0})

	defer container.SetRunner(mock)()

	result, err := Setup(SetupOptions{
		WorkspacePath: "/tmp/testws2",
		Persistent:    false,
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	if result.ContainerName != containerName {
		t.Errorf("ContainerName = %q, want %q", result.ContainerName, containerName)
	}

	// Should have called delete (stopped leftover, non-persistent)
	deleteCalls := mock.callsContaining("delete")
	if len(deleteCalls) == 0 {
		t.Error("Expected delete call for stopped non-persistent container")
	}

	// Should have called init (recreate)
	var foundInit bool
	for _, call := range mock.calls {
		if len(call) >= 3 && strings.Contains(call[2], " init ") && !strings.Contains(call[2], "image") {
			foundInit = true
			break
		}
	}
	if !foundInit {
		t.Error("Expected init call after deleting stopped container")
	}
}

func TestSetup_InvalidWorkspacePath(t *testing.T) {
	workDir := "/nonexistent/path/workspace"
	expectedName := ContainerName(workDir, 1)
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: imageListJSON("clincus"), exitCode: 0})
	mock.on("--format=csv", mockResponse{stdout: "", exitCode: 0})
	mock.on("--format=json", mockResponse{
		stdout:   containerListJSON(expectedName, "Running"),
		exitCode: 0,
	})
	// device add fails for workspace
	mock.on("device add", mockResponse{stdout: "Error: path not found", exitCode: 1})

	defer container.SetRunner(mock)()

	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err == nil {
		t.Fatal("expected error for invalid workspace path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to add workspace device") {
		t.Errorf("error = %q, want it to contain 'failed to add workspace device'", err.Error())
	}
}

func TestSetup_PartialMountFailure(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mountDir := t.TempDir()
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: imageListJSON("clincus"), exitCode: 0})
	mock.on("--format=csv", mockResponse{stdout: "", exitCode: 0})
	mock.on("--format=json", mockResponse{
		stdout:   containerListJSON(expectedName, "Running"),
		exitCode: 0,
	})
	// The extra mount's device name is "extra-mount".
	// MountDisk generates: device add <container> extra-mount disk source=... path=...
	// This substring uniquely identifies the extra mount command.
	mock.on("extra-mount", mockResponse{stdout: "Error: mount failed", exitCode: 1})
	mock.on("echo ready", mockResponse{stdout: "ready", exitCode: 0})

	defer container.SetRunner(mock)()

	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		MountConfig: &MountConfig{
			Mounts: []MountEntry{
				{
					HostPath:      mountDir,
					ContainerPath: "/data",
					DeviceName:    "extra-mount",
				},
			},
		},
		Logger: noopLogger(),
	})
	if err == nil {
		t.Fatal("expected error for partial mount failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to add mount") {
		t.Errorf("error = %q, want it to contain 'failed to add mount'", err.Error())
	}
}

func TestSetup_PermissionError(t *testing.T) {
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: imageListJSON("clincus"), exitCode: 0})
	mock.on("--format=csv", mockResponse{stdout: "", exitCode: 0})
	mock.on("--format=json", mockResponse{stdout: "[]", exitCode: 0})
	// init fails with permission denied
	mock.on(" init ", mockResponse{stdout: "Error: permission denied", exitCode: 1})

	defer container.SetRunner(mock)()

	_, err := Setup(SetupOptions{
		WorkspacePath: t.TempDir(),
		Slot:          1,
		Logger:        noopLogger(),
	})
	if err == nil {
		t.Fatal("expected error for permission denied, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create container") {
		t.Errorf("error = %q, want it to contain 'failed to create container'", err.Error())
	}
}
