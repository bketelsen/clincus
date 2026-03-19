package session

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bketelsen/clincus/internal/config"
	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/limits"
	"github.com/bketelsen/clincus/internal/tool"
)

const (
	DefaultImage = "images:ubuntu/24.04"
	ClincusImage = "clincus"
)

// isColimaOrLimaEnvironment detects if we're running inside a Colima or Lima VM
// These VMs use virtiofs for mounting host directories and already handle UID mapping
// at the VM level, making Incus's shift=true unnecessary and problematic
func isColimaOrLimaEnvironment() bool {
	// Check for virtiofs mounts which are characteristic of Lima/Colima
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}

	// Lima mounts host directories via virtiofs (e.g., "mount0 on /Users/... type virtiofs")
	// Colima uses Lima under the hood, so same detection applies
	mounts := string(data)
	if strings.Contains(mounts, "virtiofs") {
		return true
	}

	// Additional check: Lima typically runs as the "lima" user
	if user := os.Getenv("USER"); user == "lima" {
		return true
	}

	return false
}

// buildJSONFromSettings converts a settings map to a properly escaped JSON string
// Uses json.Marshal to ensure proper escaping and avoid command injection
func buildJSONFromSettings(settings map[string]interface{}) (string, error) {
	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("failed to marshal settings: %w", err)
	}
	return string(jsonBytes), nil
}

// SetupOptions contains options for setting up a session
type SetupOptions struct {
	WorkspacePath         string
	Image                 string
	Persistent            bool // Keep container between sessions (don't delete on cleanup)
	ResumeFromID          string
	Slot                  int
	MountConfig           *MountConfig         // Multi-mount support
	SessionsDir           string               // e.g., ~/.clincus/sessions-claude
	CLIConfigPath         string               // e.g., ~/.claude (host CLI config to copy credentials from)
	Tool                  tool.Tool            // AI coding tool being used
	DisableShift          bool                 // Disable UID shifting (for Colima/Lima environments)
	LimitsConfig          *config.LimitsConfig // Resource and time limits
	IncusProject          string               // Incus project name
	ProtectedPaths        []string             // Paths to mount read-only for security (e.g., .git/hooks, .vscode)
	PreserveWorkspacePath bool                 // Mount workspace at same path as host instead of /workspace
	Logger                func(string)
	ContainerName         string // Use existing container (for testing) - skips container creation
}

// SetupResult contains the result of setup
type SetupResult struct {
	ContainerName          string
	Manager                *container.Manager
	TimeoutMonitor         *limits.TimeoutMonitor
	HomeDir                string
	RunAsRoot              bool
	Image                  string
	ContainerWorkspacePath string // Path where workspace is mounted inside container (default: /workspace)
}

// Setup initializes a container for a session.
// This configures the container with workspace mounting and user setup.
func Setup(opts SetupOptions) (*SetupResult, error) {
	result := &SetupResult{}

	// Default logger
	if opts.Logger == nil {
		opts.Logger = func(msg string) {
			fmt.Fprintf(os.Stderr, "[setup] %s\n", msg)
		}
	}

	// Steps 1-3: Resolve container name, image, execution context
	if err := resolveContainer(&opts, result); err != nil {
		return nil, err
	}

	// Steps 4-5: Check existing container, create/configure/start if needed
	skipLaunch, err := createAndConfigureContainer(&opts, result)
	if err != nil {
		return nil, err
	}

	// Steps 6-7: Wait for ready, set metadata, record history, start timeout
	if err := postLaunchSetup(&opts, result); err != nil {
		return nil, err
	}

	// Steps 9-11: Restore session, inject credentials, setup tool config
	configureToolAccess(&opts, result, skipLaunch)

	opts.Logger("Container setup complete!")
	return result, nil
}
