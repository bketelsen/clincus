package container

import (
	"context"
	"strings"
	"testing"
)

// --- ExitError tests ---

func TestExitError_Error(t *testing.T) {
	err := &ExitError{ExitCode: 42}
	got := err.Error()
	want := "exit status 42"
	if got != want {
		t.Errorf("ExitError.Error() = %q, want %q", got, want)
	}
}

// --- NewManager tests ---

func TestNewManager(t *testing.T) {
	mgr := NewManager("test-container")
	if mgr.ContainerName != "test-container" {
		t.Errorf("NewManager().ContainerName = %q, want %q", mgr.ContainerName, "test-container")
	}
}

// --- Manager.Delete tests ---

func TestManagerDelete_Force(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Delete(true)
	if err != nil {
		t.Errorf("Manager.Delete(true) returned error: %v, want nil", err)
	}
}

func TestManagerDelete_NoForce(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Delete(false)
	if err != nil {
		t.Errorf("Manager.Delete(false) returned error: %v, want nil", err)
	}
}

// --- Manager.Running tests ---

func TestManagerRunning_True(t *testing.T) {
	mock := &mockRunner{
		stdout:   `[{"name":"test-container","status":"Running"}]`,
		exitCode: 0,
	}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	running, err := mgr.Running()
	if err != nil {
		t.Fatalf("Manager.Running() returned error: %v", err)
	}
	if !running {
		t.Error("Manager.Running() = false, want true")
	}
}

// --- Manager.Start tests ---

func TestManagerStart(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Start()
	if err != nil {
		t.Errorf("Manager.Start() returned error: %v, want nil", err)
	}
}

// --- Manager.SetConfig tests ---

func TestManagerSetConfig(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.SetConfig("limits.cpu", "4")
	if err != nil {
		t.Errorf("Manager.SetConfig() returned error: %v, want nil", err)
	}
}

// --- Manager.GetConfig tests ---

func TestManagerGetConfig(t *testing.T) {
	mock := &mockRunner{stdout: "4", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	val, err := mgr.GetConfig("limits.cpu")
	if err != nil {
		t.Fatalf("Manager.GetConfig() returned error: %v", err)
	}
	if val != "4" {
		t.Errorf("Manager.GetConfig() = %q, want %q", val, "4")
	}
}

// --- Manager.MountDisk tests ---

func TestManagerMountDisk_Basic(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.MountDisk("workspace", "/host/path", "/container/path", false, false)
	if err != nil {
		t.Errorf("Manager.MountDisk() returned error: %v, want nil", err)
	}
}

func TestManagerMountDisk_ShiftAndReadonly(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.MountDisk("workspace", "/host/path", "/container/path", true, true)
	if err != nil {
		t.Errorf("Manager.MountDisk(shift=true, readonly=true) returned error: %v, want nil", err)
	}
}

// --- Manager.Stop no force tests ---

func TestManagerStop_NoForce(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Stop(false)
	if err != nil {
		t.Errorf("Manager.Stop(false) returned error: %v, want nil", err)
	}
}

// --- DeleteContainer tests ---

func TestDeleteContainer_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := DeleteContainer("test-container")
	if err != nil {
		t.Errorf("DeleteContainer() returned error: %v, want nil", err)
	}
}

// --- StopContainer tests ---

func TestStopContainer_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := StopContainer("test-container")
	if err != nil {
		t.Errorf("StopContainer() returned error: %v, want nil", err)
	}
}

// --- IncusOutputWithStderr tests ---

func TestIncusOutputWithStderr_Success(t *testing.T) {
	mock := &mockRunner{stdout: "combined output", exitCode: 0}
	defer SetRunner(mock)()

	output, err := IncusOutputWithStderr("info")
	if err != nil {
		t.Fatalf("IncusOutputWithStderr() returned error: %v", err)
	}
	if !strings.Contains(output, "combined output") {
		t.Errorf("IncusOutputWithStderr() = %q, want to contain %q", output, "combined output")
	}
}

func TestIncusOutputWithStderr_ExitError(t *testing.T) {
	mock := &mockRunner{stdout: "error output", exitCode: 1}
	defer SetRunner(mock)()

	output, err := IncusOutputWithStderr("info")
	if err == nil {
		t.Fatal("IncusOutputWithStderr() returned nil error, want non-nil")
	}
	if !strings.Contains(output, "error output") {
		t.Errorf("IncusOutputWithStderr() output = %q, want to contain %q", output, "error output")
	}
}

// --- IncusOutputWithArgs tests ---

func TestIncusOutputWithArgs_Success(t *testing.T) {
	mock := &mockRunner{stdout: "args output", exitCode: 0}
	defer SetRunner(mock)()

	output, err := IncusOutputWithArgs("info", "--format=json")
	if err != nil {
		t.Fatalf("IncusOutputWithArgs() returned error: %v", err)
	}
	if output != "args output" {
		t.Errorf("IncusOutputWithArgs() = %q, want %q", output, "args output")
	}
}

// --- IncusFilePush tests ---

func TestIncusFilePush_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := IncusFilePush("/tmp/test", "container/path")
	if err != nil {
		t.Errorf("IncusFilePush() returned error: %v, want nil", err)
	}
}

// --- IncusExecQuietContext tests ---

func TestIncusExecQuietContext_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	ctx := context.Background()
	err := IncusExecQuietContext(ctx, "list")
	if err != nil {
		t.Errorf("IncusExecQuietContext() returned error: %v, want nil", err)
	}
}

// --- LaunchContainerPersistent tests ---

func TestLaunchContainerPersistent_Success(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := LaunchContainerPersistent("clincus", "test-container")
	if err != nil {
		t.Errorf("LaunchContainerPersistent() returned error: %v, want nil", err)
	}
}

// --- Manager.Launch persistent tests ---

func TestManagerLaunch_Persistent(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test-container"}
	err := mgr.Launch("clincus", false)
	if err != nil {
		t.Errorf("Manager.Launch(clincus, false) returned error: %v, want nil", err)
	}
}

// --- toMountSize tests ---

func TestToMountSize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2GiB", "2G"},
		{"512MiB", "512M"},
		{"1024KiB", "1024K"},
		{"1TiB", "1T"},
		{"500MB", "500MB"}, // no conversion for non-Incus suffix
	}
	for _, tt := range tests {
		got := toMountSize(tt.input)
		if got != tt.want {
			t.Errorf("toMountSize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- shellQuote tests ---

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"has space", "'has space'"},
		{"has'quote", "'has'\"'\"'quote'"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- IncusOutputRawContext error tests ---

func TestIncusOutputRawContext_ExitError(t *testing.T) {
	mock := &mockRunner{stdout: "raw error", exitCode: 2}
	defer SetRunner(mock)()

	ctx := context.Background()
	_, err := IncusOutputRawContext(ctx, "list")
	if err == nil {
		t.Fatal("IncusOutputRawContext() returned nil error, want non-nil")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("error type = %T, want *ExitError", err)
	}
	if exitErr.ExitCode != 2 {
		t.Errorf("ExitError.ExitCode = %d, want 2", exitErr.ExitCode)
	}
}

// --- Manager.GetWorkspacePath tests ---

func TestManagerGetWorkspacePath_Found(t *testing.T) {
	yamlOutput := `workspace:
  path: /custom/path
  source: /host/src
  type: disk`
	mock := &mockRunner{stdout: yamlOutput, exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	got := mgr.GetWorkspacePath()
	if got != "/custom/path" {
		t.Errorf("GetWorkspacePath() = %q, want %q", got, "/custom/path")
	}
}

func TestManagerGetWorkspacePath_NotFound(t *testing.T) {
	mock := &mockRunner{stdout: "other:\n  type: disk\n", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	got := mgr.GetWorkspacePath()
	if got != "/workspace" {
		t.Errorf("GetWorkspacePath() = %q, want %q", got, "/workspace")
	}
}

func TestManagerGetWorkspacePath_Error(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 1}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	got := mgr.GetWorkspacePath()
	if got != "/workspace" {
		t.Errorf("GetWorkspacePath() = %q, want %q (fallback)", got, "/workspace")
	}
}

// --- Manager.DirExists tests ---

func TestManagerDirExists_True(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	exists, err := mgr.DirExists("/tmp")
	if err != nil {
		t.Fatalf("DirExists() error: %v", err)
	}
	if !exists {
		t.Error("DirExists() = false, want true")
	}
}

func TestManagerDirExists_False(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 1}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	exists, err := mgr.DirExists("/nonexistent")
	if err != nil {
		t.Fatalf("DirExists() error: %v", err)
	}
	if exists {
		t.Error("DirExists() = true, want false")
	}
}

// --- Manager.FileExists tests ---

func TestManagerFileExists_True(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	exists, err := mgr.FileExists("/etc/passwd")
	if err != nil {
		t.Fatalf("FileExists() error: %v", err)
	}
	if !exists {
		t.Error("FileExists() = false, want true")
	}
}

// --- Manager.Chown tests ---

func TestManagerChown(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	err := mgr.Chown("/workspace", 1000, 1000)
	if err != nil {
		t.Errorf("Chown() returned error: %v, want nil", err)
	}
}

// --- Manager.Exec tests ---

func TestManagerExec(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	err := mgr.Exec("ls", "-la")
	if err != nil {
		t.Errorf("Exec() returned error: %v, want nil", err)
	}
}

// --- Manager.ExecCommand tests ---

func TestManagerExecCommand_Capture(t *testing.T) {
	mock := &mockRunner{stdout: "captured output", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	output, err := mgr.ExecCommand("ls -la", ExecCommandOptions{Capture: true})
	if err != nil {
		t.Fatalf("ExecCommand() returned error: %v", err)
	}
	if output != "captured output" {
		t.Errorf("ExecCommand() = %q, want %q", output, "captured output")
	}
}

func TestManagerExecCommand_NoCapture(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	_, err := mgr.ExecCommand("echo hello", ExecCommandOptions{})
	if err != nil {
		t.Errorf("ExecCommand() returned error: %v, want nil", err)
	}
}

func TestManagerExecCommand_WithUserAndGroup(t *testing.T) {
	mock := &mockRunner{stdout: "uid=1000", exitCode: 0}
	defer SetRunner(mock)()

	uid := 1000
	gid := 1000
	mgr := &Manager{ContainerName: "test"}
	output, err := mgr.ExecCommand("id", ExecCommandOptions{
		Capture: true,
		User:    &uid,
		Group:   &gid,
		Cwd:     "/workspace",
		Env:     map[string]string{"HOME": "/home/user"},
	})
	if err != nil {
		t.Fatalf("ExecCommand() returned error: %v", err)
	}
	if output != "uid=1000" {
		t.Errorf("ExecCommand() = %q, want %q", output, "uid=1000")
	}
}

// --- Manager.ExecArgs tests ---

func TestManagerExecArgs(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	err := mgr.ExecArgs([]string{"ls", "-la"}, ExecCommandOptions{
		Cwd: "/workspace",
		Env: map[string]string{"TERM": "xterm"},
	})
	if err != nil {
		t.Errorf("ExecArgs() returned error: %v, want nil", err)
	}
}

func TestManagerExecArgs_WithUser(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	uid := 1000
	mgr := &Manager{ContainerName: "test"}
	err := mgr.ExecArgs([]string{"whoami"}, ExecCommandOptions{User: &uid})
	if err != nil {
		t.Errorf("ExecArgs() returned error: %v, want nil", err)
	}
}

// --- Manager.ExecArgsCapture tests ---

func TestManagerExecArgsCapture(t *testing.T) {
	mock := &mockRunner{stdout: "  captured\n", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	output, err := mgr.ExecArgsCapture([]string{"cat", "/etc/hostname"}, ExecCommandOptions{
		Cwd: "/",
	})
	if err != nil {
		t.Fatalf("ExecArgsCapture() returned error: %v", err)
	}
	// ExecArgsCapture uses IncusOutputRaw, which preserves whitespace
	if output != "  captured\n" {
		t.Errorf("ExecArgsCapture() = %q, want %q", output, "  captured\n")
	}
}

// --- Manager.PushFile tests ---

func TestManagerPushFile(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	err := mgr.PushFile("/tmp/local-file", "/etc/remote-file")
	if err != nil {
		t.Errorf("PushFile() returned error: %v, want nil", err)
	}
}

func TestManagerPushFile_NoLeadingSlash(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	mgr := &Manager{ContainerName: "test"}
	err := mgr.PushFile("/tmp/local-file", "etc/remote-file")
	if err != nil {
		t.Errorf("PushFile() returned error: %v, want nil", err)
	}
}

// --- Configure tests ---

func TestConfigure(t *testing.T) {
	origGroup := IncusGroup
	origProject := IncusProject
	defer func() {
		IncusGroup = origGroup
		IncusProject = origProject
	}()

	Configure("test-project", "test-group", "testuser", 1000)
	if IncusGroup != "test-group" {
		t.Errorf("IncusGroup = %q, want %q", IncusGroup, "test-group")
	}
	if IncusProject != "test-project" {
		t.Errorf("IncusProject = %q, want %q", IncusProject, "test-project")
	}
	if CodeUser != "testuser" {
		t.Errorf("CodeUser = %q, want %q", CodeUser, "testuser")
	}
	if CodeUID != 1000 {
		t.Errorf("CodeUID = %d, want 1000", CodeUID)
	}
}

// --- ImageExists tests ---

func TestImageExists_Found(t *testing.T) {
	mock := &mockRunner{stdout: `[{"aliases":[{"name":"clincus"}]}]`, exitCode: 0}
	defer SetRunner(mock)()

	exists, err := ImageExists("clincus")
	if err != nil {
		t.Fatalf("ImageExists() error: %v", err)
	}
	if !exists {
		t.Error("ImageExists() = false, want true")
	}
}

func TestImageExists_NotFound(t *testing.T) {
	mock := &mockRunner{stdout: `[{"aliases":[{"name":"other"}]}]`, exitCode: 0}
	defer SetRunner(mock)()

	exists, err := ImageExists("nonexistent")
	if err != nil {
		t.Fatalf("ImageExists() error: %v", err)
	}
	if exists {
		t.Error("ImageExists() = true, want false")
	}
}

// --- ListImagesByPrefix tests ---

func TestListImagesByPrefix(t *testing.T) {
	mock := &mockRunner{stdout: `[{"aliases":[{"name":"clincus"}]},{"aliases":[{"name":"clincus-v2"}]},{"aliases":[{"name":"other"}]}]`, exitCode: 0}
	defer SetRunner(mock)()

	images, err := ListImagesByPrefix("clincus")
	if err != nil {
		t.Fatalf("ListImagesByPrefix() error: %v", err)
	}
	if len(images) != 2 {
		t.Errorf("ListImagesByPrefix() returned %d images, want 2", len(images))
	}
}

// --- ListContainers tests ---

func TestListContainers(t *testing.T) {
	mock := &mockRunner{stdout: `[{"name":"clincus-abc"},{"name":"clincus-def"},{"name":"other"}]`, exitCode: 0}
	defer SetRunner(mock)()

	containers, err := ListContainers("^clincus-")
	if err != nil {
		t.Fatalf("ListContainers() error: %v", err)
	}
	if len(containers) != 2 {
		t.Errorf("ListContainers() returned %d containers, want 2", len(containers))
	}
}

// --- SnapshotCreate tests ---

func TestSnapshotCreate_Stateful(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := SnapshotCreate("test-container", "snap1", true)
	if err != nil {
		t.Errorf("SnapshotCreate() error: %v, want nil", err)
	}
}

func TestSnapshotCreate_Stateless(t *testing.T) {
	mock := &mockRunner{stdout: "", exitCode: 0}
	defer SetRunner(mock)()

	err := SnapshotCreate("test-container", "snap1", false)
	if err != nil {
		t.Errorf("SnapshotCreate() error: %v, want nil", err)
	}
}
