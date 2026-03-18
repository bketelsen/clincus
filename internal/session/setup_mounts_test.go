package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bketelsen/clincus/internal/container"
)

func TestSetupMounts_HappyPath(t *testing.T) {
	mock := newSessionMockRunner()
	defer container.SetRunner(mock)()

	mgr := container.NewManager("test-container")
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	config := &MountConfig{
		Mounts: []MountEntry{
			{HostPath: dir1, ContainerPath: "/data1", DeviceName: "mount1"},
			{HostPath: dir2, ContainerPath: "/data2", DeviceName: "mount2"},
		},
	}

	err := setupMounts(mgr, config, true, noopLogger())
	if err != nil {
		t.Fatalf("setupMounts() returned error: %v", err)
	}

	// Verify device add was called for each mount
	mount1Calls := mock.callsContaining("mount1")
	if len(mount1Calls) == 0 {
		t.Error("Expected device add call for mount1")
	}
	mount2Calls := mock.callsContaining("mount2")
	if len(mount2Calls) == 0 {
		t.Error("Expected device add call for mount2")
	}
}

func TestSetupMounts_NilConfig(t *testing.T) {
	mock := newSessionMockRunner()
	defer container.SetRunner(mock)()

	mgr := container.NewManager("test-container")
	err := setupMounts(mgr, nil, true, noopLogger())
	if err != nil {
		t.Fatalf("setupMounts(nil config) returned error: %v", err)
	}
}

func TestSetupMounts_EmptyMounts(t *testing.T) {
	mock := newSessionMockRunner()
	defer container.SetRunner(mock)()

	mgr := container.NewManager("test-container")
	err := setupMounts(mgr, &MountConfig{}, true, noopLogger())
	if err != nil {
		t.Fatalf("setupMounts(empty config) returned error: %v", err)
	}
}

func TestSetupMounts_MountFailure(t *testing.T) {
	mock := newSessionMockRunner()
	mock.on("failmount", mockResponse{stdout: "Error: mount failed", exitCode: 1})
	defer container.SetRunner(mock)()

	mgr := container.NewManager("test-container")
	config := &MountConfig{
		Mounts: []MountEntry{
			{HostPath: t.TempDir(), ContainerPath: "/fail", DeviceName: "failmount"},
		},
	}

	err := setupMounts(mgr, config, true, noopLogger())
	if err == nil {
		t.Fatal("expected error for mount failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to add mount") {
		t.Errorf("error = %q, want it to contain 'failed to add mount'", err.Error())
	}
}

func TestSetupMounts_HostDirCreation(t *testing.T) {
	mock := newSessionMockRunner()
	defer container.SetRunner(mock)()

	mgr := container.NewManager("test-container")
	// Use a non-existent subdirectory of t.TempDir()
	hostDir := filepath.Join(t.TempDir(), "subdir", "nested")

	config := &MountConfig{
		Mounts: []MountEntry{
			{HostPath: hostDir, ContainerPath: "/data", DeviceName: "newdir"},
		},
	}

	err := setupMounts(mgr, config, true, noopLogger())
	if err != nil {
		t.Fatalf("setupMounts() returned error: %v", err)
	}

	// Verify the host directory was created
	info, err := os.Stat(hostDir)
	if err != nil {
		t.Fatalf("expected host directory %q to exist, got error: %v", hostDir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %q to be a directory", hostDir)
	}
}
