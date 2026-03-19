package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/tool"
)

// mockTool implements tool.Tool for testing purposes.
type mockTool struct {
	name            string
	binary          string
	configDirName   string
	sessionsDirName string
	sandboxSettings map[string]interface{}
}

func (m *mockTool) Name() string            { return m.name }
func (m *mockTool) Binary() string          { return m.binary }
func (m *mockTool) ConfigDirName() string   { return m.configDirName }
func (m *mockTool) SessionsDirName() string { return m.sessionsDirName }
func (m *mockTool) BuildCommand(sessionID string, resume bool, resumeSessionID string) []string {
	return []string{m.binary}
}
func (m *mockTool) DiscoverSessionID(stateDir string) string { return "" }
func (m *mockTool) GetSandboxSettings() map[string]interface{} {
	return m.sandboxSettings
}

// mockEnvTool is a tool that uses ENV-based auth (no config dir, no home config file).
type mockEnvTool struct {
	mockTool
}

func TestConfigureToolAccess_DirectoryBased_FirstLaunch(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	// Create a CLIConfigPath with a .credentials.json file
	cliConfigDir := filepath.Join(t.TempDir(), ".claude")
	if err := os.MkdirAll(cliConfigDir, 0o755); err != nil {
		t.Fatalf("failed to create cli config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliConfigDir, ".credentials.json"), []byte(`{"token":"test"}`), 0o644); err != nil {
		t.Fatalf("failed to write credentials: %v", err)
	}

	claudeTool := tool.NewClaude()

	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Tool:          claudeTool,
		CLIConfigPath: cliConfigDir,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	// Verify mkdir was called for config directory
	mkdirCalls := mock.callsContaining("mkdir")
	if len(mkdirCalls) == 0 {
		t.Error("Expected mkdir call for config directory setup")
	}

	// Verify file push operations occurred (credential files)
	pushCalls := mock.callsContaining("file push")
	if len(pushCalls) == 0 {
		t.Error("Expected file push calls for credential files")
	}
}

func TestConfigureToolAccess_EnvBasedAuth(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	envTool := &mockEnvTool{mockTool{
		name:          "codex",
		binary:        "codex",
		configDirName: "",
	}}

	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Tool:          envTool,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	// Verify NO config push operations (ENV-based auth)
	pushCalls := mock.callsContaining("file push")
	if len(pushCalls) > 0 {
		t.Errorf("Expected no file push calls for ENV-based tool, found %d", len(pushCalls))
	}
	mkdirCalls := mock.callsContaining("mkdir -p /home/code/")
	if len(mkdirCalls) > 0 {
		t.Errorf("Expected no mkdir calls for config dir of ENV-based tool, found %d", len(mkdirCalls))
	}
}

func TestConfigureToolAccess_Resume_WithRestore(t *testing.T) {
	workDir := t.TempDir()
	expectedName := ContainerName(workDir, 1)
	mock := setupHappyPathMock(expectedName, ClincusImage)
	defer container.SetRunner(mock)()

	// Create sessions directory with saved session data
	sessionsDir := filepath.Join(t.TempDir(), "sessions-claude")
	sessionID := "session-123"
	savedConfigDir := filepath.Join(sessionsDir, sessionID, ".claude")
	if err := os.MkdirAll(savedConfigDir, 0o755); err != nil {
		t.Fatalf("failed to create saved session dir: %v", err)
	}
	// Create a dummy file in the saved session
	if err := os.WriteFile(filepath.Join(savedConfigDir, "settings.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write session data: %v", err)
	}

	// Create CLI config path with credentials
	cliConfigDir := filepath.Join(t.TempDir(), ".claude")
	if err := os.MkdirAll(cliConfigDir, 0o755); err != nil {
		t.Fatalf("failed to create cli config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliConfigDir, ".credentials.json"), []byte(`{"token":"fresh"}`), 0o644); err != nil {
		t.Fatalf("failed to write credentials: %v", err)
	}

	claudeTool := tool.NewClaude()

	_, err := Setup(SetupOptions{
		WorkspacePath: workDir,
		Slot:          1,
		Tool:          claudeTool,
		ResumeFromID:  sessionID,
		SessionsDir:   sessionsDir,
		CLIConfigPath: cliConfigDir,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	// Verify push operations occurred (session restore + credential injection)
	pushCalls := mock.callsContaining("file push")
	if len(pushCalls) == 0 {
		t.Error("Expected file push calls for session restore and credential injection")
	}
}

func TestConfigureToolAccess_SkipLaunch_Persistent(t *testing.T) {
	containerName := ContainerName("/tmp/persistent-ws", 1)
	mock := newSessionMockRunner()
	mock.on("image list", mockResponse{stdout: imageListJSON("clincus"), exitCode: 0})
	// Container exists and is running
	mock.on("--format=csv", mockResponse{stdout: containerName, exitCode: 0})
	mock.on("--format=json", mockResponse{
		stdout:   containerListJSON(containerName, "Running"),
		exitCode: 0,
	})
	mock.on("echo ready", mockResponse{stdout: "ready", exitCode: 0})

	defer container.SetRunner(mock)()

	// Create CLI config path with credentials
	cliConfigDir := filepath.Join(t.TempDir(), ".claude")
	if err := os.MkdirAll(cliConfigDir, 0o755); err != nil {
		t.Fatalf("failed to create cli config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliConfigDir, ".credentials.json"), []byte(`{"token":"test"}`), 0o644); err != nil {
		t.Fatalf("failed to write credentials: %v", err)
	}

	claudeTool := tool.NewClaude()

	_, err := Setup(SetupOptions{
		WorkspacePath: "/tmp/persistent-ws",
		Persistent:    true,
		Slot:          1,
		Tool:          claudeTool,
		CLIConfigPath: cliConfigDir,
		Logger:        noopLogger(),
	})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}

	// Should NOT push new config (skipLaunch=true, persistent container reused)
	// The setupCLIConfig path has: if !skipLaunch { ... setup config ... }
	mkdirCalls := mock.callsContaining("mkdir -p")
	if len(mkdirCalls) > 0 {
		t.Errorf("Expected no mkdir calls for config dir on persistent reuse, found %d", len(mkdirCalls))
	}
}
