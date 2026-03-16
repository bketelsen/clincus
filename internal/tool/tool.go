package tool

import (
	"os"
	"path/filepath"
	"strings"
)

// Tool represents an AI coding tool that can be run in COI containers
type Tool interface {
	// Name returns the tool name (e.g., "claude", "aider", "cursor")
	Name() string

	// Binary returns the binary name to execute
	Binary() string

	// ConfigDirName returns config directory name (e.g., ".claude", ".aider")
	// Return "" if tool uses ENV API keys instead of config files
	ConfigDirName() string

	// SessionsDirName returns the sessions directory name for this tool
	// (e.g., "sessions-claude", "sessions-aider")
	SessionsDirName() string

	// BuildCommand builds the command line for execution
	// sessionID: COI session ID
	// resume: whether to resume an existing session
	// resumeSessionID: the tool's internal session ID (if resuming)
	BuildCommand(sessionID string, resume bool, resumeSessionID string) []string

	// DiscoverSessionID finds the tool's internal session ID from saved state
	// stateDir: path to the tool's config directory with saved state
	// Return "" if tool doesn't support session resume (will start fresh each time)
	DiscoverSessionID(stateDir string) string

	// GetSandboxSettings returns settings to inject for sandbox/bypass permissions
	// Return empty map if tool doesn't need settings injection
	GetSandboxSettings() map[string]interface{}
}

// ClaudeTool implements Tool for Claude Code
type ClaudeTool struct {
	effortLevel string // "low", "medium", "high" - defaults to "medium" if empty
}

// NewClaude creates a new Claude tool instance
func NewClaude() Tool {
	return &ClaudeTool{}
}

func (c *ClaudeTool) Name() string {
	return "claude"
}

func (c *ClaudeTool) Binary() string {
	return "claude"
}

func (c *ClaudeTool) ConfigDirName() string {
	return ".claude"
}

func (c *ClaudeTool) SessionsDirName() string {
	return "sessions-claude"
}

func (c *ClaudeTool) BuildCommand(sessionID string, resume bool, resumeSessionID string) []string {
	// Base command with flags
	cmd := []string{"claude", "--verbose", "--permission-mode", "bypassPermissions"}

	// Add session/resume flag
	if resume {
		if resumeSessionID != "" {
			cmd = append(cmd, "--resume", resumeSessionID)
		} else {
			cmd = append(cmd, "--resume")
		}
	} else {
		cmd = append(cmd, "--session-id", sessionID)
	}

	return cmd
}

func (c *ClaudeTool) DiscoverSessionID(stateDir string) string {
	// Claude stores sessions as .jsonl files in projects/-workspace/
	// This logic is extracted from cleanup.go:387-411
	projectsDir := filepath.Join(stateDir, "projects", "-workspace")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return ""
	}

	// Find the first .jsonl file (Claude session file)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			return strings.TrimSuffix(entry.Name(), ".jsonl")
		}
	}

	return ""
}

func (c *ClaudeTool) GetSandboxSettings() map[string]interface{} {
	// Settings to inject into .claude/settings.json for bypassing permissions
	// and setting effort level
	settings := map[string]interface{}{
		"allowDangerouslySkipPermissions": true,
		"bypassPermissionsModeAccepted":   true,
		"permissions": map[string]string{
			"defaultMode": "bypassPermissions",
		},
	}

	// Set effort level (default to "medium" if not configured)
	// We set both effortLevel directly AND via env var for maximum compatibility
	effortLevel := c.effortLevel
	if effortLevel == "" {
		effortLevel = "medium"
	}
	settings["effortLevel"] = effortLevel

	// Effort prompt suppression flags (discovered by examining .claude.json after manual selection)
	settings["effortLevelAccepted"] = true
	settings["hasSeenEffortPrompt"] = true
	settings["effortCalloutDismissed"] = true

	// Also set via env section (CLAUDE_CODE_EFFORT_LEVEL) as documented
	settings["env"] = map[string]string{
		"CLAUDE_CODE_EFFORT_LEVEL": effortLevel,
	}

	return settings
}

// SetEffortLevel sets the effort level for Claude Code.
// Valid values: "low", "medium", "high".
// This prevents the interactive effort prompt during autonomous sessions.
func (c *ClaudeTool) SetEffortLevel(level string) {
	c.effortLevel = level
}

// ToolWithHomeConfigFile is an optional interface for tools that store their
// configuration in a single JSON file in the user's home directory
// (e.g., ~/.opencode.json), rather than a subdirectory.
type ToolWithHomeConfigFile interface {
	Tool
	// HomeConfigFileName returns the dot-prefixed filename in the home dir
	// (e.g., ".opencode.json").
	HomeConfigFileName() string
}

// ToolWithEffortLevel is an optional interface for tools that support
// configurable effort levels (e.g., Claude's low/medium/high effort).
type ToolWithEffortLevel interface {
	Tool
	// SetEffortLevel sets the effort level for the tool.
	// Valid values depend on the tool (e.g., "low", "medium", "high" for Claude).
	SetEffortLevel(level string)
}
