package tool

import (
	"testing"
)

func TestCopilotTool_Basics(t *testing.T) {
	cp := NewCopilot()

	if cp.Name() != "copilot" {
		t.Errorf("Name() = %q, want %q", cp.Name(), "copilot")
	}
	if cp.Binary() != "copilot" {
		t.Errorf("Binary() = %q, want %q", cp.Binary(), "copilot")
	}
	if cp.ConfigDirName() != ".copilot" {
		t.Errorf("ConfigDirName() = %q, want %q", cp.ConfigDirName(), ".copilot")
	}
	if cp.SessionsDirName() != "sessions-copilot" {
		t.Errorf("SessionsDirName() = %q, want %q", cp.SessionsDirName(), "sessions-copilot")
	}
}

func TestCopilotTool_BuildCommand(t *testing.T) {
	cp := NewCopilot()
	cmd := cp.BuildCommand("some-session-id", false, "")
	expected := []string{"copilot", "--allow-all-tools"}
	if len(cmd) != len(expected) {
		t.Fatalf("BuildCommand() returned %d args, want %d", len(cmd), len(expected))
	}
	for i, v := range expected {
		if cmd[i] != v {
			t.Errorf("BuildCommand()[%d] = %q, want %q", i, cmd[i], v)
		}
	}
}

func TestCopilotTool_BuildCommand_Resume(t *testing.T) {
	cp := NewCopilot()
	// copilot doesn't support CLI-based resume — same command either way
	cmd := cp.BuildCommand("", true, "some-id")
	if len(cmd) != 2 || cmd[0] != "copilot" || cmd[1] != "--allow-all-tools" {
		t.Errorf("BuildCommand(resume) = %v, want [copilot --allow-all-tools]", cmd)
	}
}

func TestCopilotTool_DiscoverSessionID(t *testing.T) {
	cp := NewCopilot()
	id := cp.DiscoverSessionID("/some/path")
	if id != "" {
		t.Errorf("DiscoverSessionID() = %q, want %q", id, "")
	}
}

func TestCopilotTool_GetSandboxSettings(t *testing.T) {
	cp := NewCopilot()
	settings := cp.GetSandboxSettings()
	if len(settings) != 0 {
		t.Errorf("GetSandboxSettings() returned %d keys, want 0", len(settings))
	}
}

func TestCopilotTool_EssentialFiles(t *testing.T) {
	cp := NewCopilot()
	tef, ok := cp.(ToolWithEssentialFiles)
	if !ok {
		t.Fatal("CopilotTool does not implement ToolWithEssentialFiles")
	}

	files := tef.EssentialFiles()
	expectedFiles := []string{"config.json", "mcp-config.json"}
	if len(files) != len(expectedFiles) {
		t.Fatalf("EssentialFiles() returned %d items, want %d", len(files), len(expectedFiles))
	}
	for i, v := range expectedFiles {
		if files[i] != v {
			t.Errorf("EssentialFiles()[%d] = %q, want %q", i, files[i], v)
		}
	}

	dirs := tef.EssentialDirs()
	expectedDirs := []string{"agents"}
	if len(dirs) != len(expectedDirs) {
		t.Fatalf("EssentialDirs() returned %d items, want %d", len(dirs), len(expectedDirs))
	}
	for i, v := range expectedDirs {
		if dirs[i] != v {
			t.Errorf("EssentialDirs()[%d] = %q, want %q", i, dirs[i], v)
		}
	}
}

func TestCopilotTool_AutoEnv(t *testing.T) {
	cp := NewCopilot()
	tae, ok := cp.(ToolWithAutoEnv)
	if !ok {
		t.Fatal("CopilotTool does not implement ToolWithAutoEnv")
	}

	// AutoEnv should return a map (possibly with GH_TOKEN if gh is installed)
	env := tae.AutoEnv()
	if env == nil {
		t.Fatal("AutoEnv() returned nil, want non-nil map")
	}

	// If GH_TOKEN is returned, it should be non-empty
	if token, ok := env["GH_TOKEN"]; ok && token == "" {
		t.Error("AutoEnv() returned empty GH_TOKEN")
	}
}

func TestCopilotTool_AutoEnv_FromEnv(t *testing.T) {
	cp := NewCopilot()
	tae := cp.(ToolWithAutoEnv)

	// Set GH_TOKEN in environment and verify it's picked up
	t.Setenv("GH_TOKEN", "test-token-123")
	env := tae.AutoEnv()
	if env["GH_TOKEN"] != "test-token-123" {
		t.Errorf("AutoEnv()[GH_TOKEN] = %q, want %q", env["GH_TOKEN"], "test-token-123")
	}
}

func TestCopilotTool_AutoEnv_FromGitHubToken(t *testing.T) {
	cp := NewCopilot()
	tae := cp.(ToolWithAutoEnv)

	// Clear GH_TOKEN, set GITHUB_TOKEN — should be forwarded as GH_TOKEN
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "github-token-456")
	env := tae.AutoEnv()
	if env["GH_TOKEN"] != "github-token-456" {
		t.Errorf("AutoEnv()[GH_TOKEN] = %q, want %q", env["GH_TOKEN"], "github-token-456")
	}
}

func TestCopilotTool_RegistryLookup(t *testing.T) {
	cp, err := Get("copilot")
	if err != nil {
		t.Fatalf("Get(\"copilot\") returned error: %v", err)
	}
	if cp.Name() != "copilot" {
		t.Errorf("Name() = %q, want %q", cp.Name(), "copilot")
	}
}

func TestListSupported_IncludesCopilot(t *testing.T) {
	supported := ListSupported()
	found := false
	for _, name := range supported {
		if name == "copilot" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListSupported() = %v, does not include 'copilot'", supported)
	}
}
