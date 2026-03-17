package tool

import (
	"os"
	"os/exec"
	"strings"
)

// CopilotTool implements Tool for GitHub Copilot CLI (https://gh.io/copilot)
type CopilotTool struct{}

// NewCopilot creates a new Copilot tool instance
func NewCopilot() Tool { return &CopilotTool{} }

func (c *CopilotTool) Name() string { return "copilot" }

func (c *CopilotTool) Binary() string { return "copilot" }

func (c *CopilotTool) ConfigDirName() string { return ".copilot" }

func (c *CopilotTool) SessionsDirName() string { return "sessions-copilot" }

// BuildCommand builds the copilot launch command.
// --allow-all-tools bypasses interactive permission prompts for autonomous use.
func (c *CopilotTool) BuildCommand(sessionID string, resume bool, resumeSessionID string) []string {
	return []string{"copilot", "--allow-all-tools"}
}

// DiscoverSessionID returns "" because copilot doesn't support CLI-based session resume.
func (c *CopilotTool) DiscoverSessionID(stateDir string) string { return "" }

// GetSandboxSettings returns an empty map — permissions are handled via the
// --allow-all-tools CLI flag rather than config file injection.
func (c *CopilotTool) GetSandboxSettings() map[string]interface{} {
	return map[string]interface{}{}
}

// EssentialFiles implements ToolWithEssentialFiles.
func (c *CopilotTool) EssentialFiles() []string {
	return []string{"config.json", "mcp-config.json"}
}

// EssentialDirs implements ToolWithEssentialFiles.
func (c *CopilotTool) EssentialDirs() []string {
	return []string{"agents"}
}

// AutoEnv implements ToolWithAutoEnv.
// Resolves GH_TOKEN for copilot authentication by checking environment
// variables first, then falling back to `gh auth token`.
func (c *CopilotTool) AutoEnv() map[string]string {
	env := map[string]string{}

	// Check if GH_TOKEN or GITHUB_TOKEN is already in the environment
	if token := os.Getenv("GH_TOKEN"); token != "" {
		env["GH_TOKEN"] = token
		return env
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		env["GH_TOKEN"] = token
		return env
	}

	// Fall back to gh CLI auth
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		token := strings.TrimSpace(string(out))
		if token != "" {
			env["GH_TOKEN"] = token
		}
	}

	return env
}
