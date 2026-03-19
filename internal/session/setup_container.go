package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bketelsen/clincus/internal/config"
	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/limits"
)

// resolveContainer resolves the container name, image, and execution context.
// Steps 1-3 of Setup().
func resolveContainer(opts *SetupOptions, result *SetupResult) error {
	// 1. Generate or use existing container name
	var containerName string
	if opts.ContainerName != "" {
		// Use existing container (for testing)
		containerName = opts.ContainerName
		opts.Logger(fmt.Sprintf("Using existing container: %s", containerName))
	} else {
		// Generate new container name
		containerName = ContainerName(opts.WorkspacePath, opts.Slot)
		opts.Logger(fmt.Sprintf("Container name: %s", containerName))
	}
	result.ContainerName = containerName
	result.Manager = container.NewManager(containerName)

	// 2. Determine image
	image := opts.Image
	if image == "" {
		image = ClincusImage
	}
	result.Image = image

	// Check if image exists
	exists, err := container.ImageExists(image)
	if err != nil {
		return fmt.Errorf("failed to check image: %w", err)
	}
	if !exists {
		return fmt.Errorf("image '%s' not found - run 'clincus build' first", image)
	}

	// 3. Determine execution context
	// clincus image has the claude user pre-configured, so run as that user
	// Other images don't have this setup, so run as root
	usingClincusImage := image == ClincusImage
	result.RunAsRoot = !usingClincusImage
	if result.RunAsRoot {
		result.HomeDir = "/root"
	} else {
		result.HomeDir = "/home/" + container.CodeUser
	}

	return nil
}

// createAndConfigureContainer checks for an existing container and creates/configures
// a new one if needed. Steps 4-5 of Setup().
// Returns skipLaunch=true if an existing container was reused.
func createAndConfigureContainer(opts *SetupOptions, result *SetupResult) (bool, error) {
	// 4. Check if container already exists
	var skipLaunch bool

	// If using existing container, skip launch
	if opts.ContainerName != "" {
		skipLaunch = true
		opts.Logger("Using existing container, skipping creation...")
	}

	exists, err := result.Manager.Exists()
	if err != nil {
		return false, fmt.Errorf("failed to check if container exists: %w", err)
	}

	if exists {
		// Check if container is currently running
		running, err := result.Manager.Running()
		if err != nil {
			return false, fmt.Errorf("failed to check if container is running: %w", err)
		}

		if running {
			// Container is running - this is an active session!
			if opts.Persistent || opts.ContainerName != "" {
				// Reuse running container if: persistent mode OR --container flag specified
				opts.Logger("Container already running, reusing...")
				skipLaunch = true
			} else {
				// ERROR: A running container exists for this slot, but we're not in persistent mode
				// This means AllocateSlot() gave us a slot that's already in use!
				return false, fmt.Errorf("slot %d is already in use by a running container %s - this should not happen (bug in slot allocation)", opts.Slot, result.ContainerName)
			}
		} else {
			// Container exists but is stopped
			if opts.Persistent || opts.ContainerName != "" {
				// Restart the stopped container
				// This includes: persistent containers OR containers specified via --container flag
				opts.Logger("Starting existing container...")
				if err := result.Manager.Start(); err != nil {
					return false, fmt.Errorf("failed to start container: %w", err)
				}
				skipLaunch = true
			} else {
				// Delete the stopped leftover container
				opts.Logger("Found stopped leftover container from previous session, deleting...")
				if err := result.Manager.Delete(true); err != nil {
					return false, fmt.Errorf("failed to delete leftover container: %w", err)
				}
				// Brief pause to let Incus fully delete
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	// 5. Create and configure container (but don't start yet if we need to add devices)
	// Always launch as non-ephemeral so we can save session data even if container is stopped
	// (e.g., via 'sudo shutdown 0' from within). Cleanup will delete if not --persistent.
	if !skipLaunch {
		opts.Logger(fmt.Sprintf("Creating container from %s...", result.Image))
		// Create container without starting it (init)
		if err := container.IncusExec("init", result.Image, result.ContainerName); err != nil {
			return false, fmt.Errorf("failed to create container: %w", err)
		}

		// Configure UID/GID mapping for bind mounts based on environment
		// Local: Use shift=true (kernel idmap support)
		// CI: Use raw.idmap (kernel lacks idmap support, runner UID 1001 → container UID 1000)
		// Colima/Lima: Disable shift (VM already handles UID mapping via virtiofs)

		// Auto-detect Colima/Lima environment if not explicitly configured
		disableShift := opts.DisableShift
		if !disableShift && isColimaOrLimaEnvironment() {
			disableShift = true
			opts.Logger("Auto-detected Colima/Lima environment - disabling UID shifting")
		}

		useShift := !disableShift
		isCI := os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"

		if isCI {
			opts.Logger("Configuring UID/GID mapping for CI environment...")
			if err := container.IncusExec("config", "set", result.ContainerName, "raw.idmap", "both 1001 1000"); err != nil {
				opts.Logger(fmt.Sprintf("Warning: Failed to set raw.idmap: %v", err))
			}
			useShift = false // Don't use shift=true with raw.idmap
		} else if disableShift {
			if !opts.DisableShift {
				// Was auto-detected, not explicitly configured
				opts.Logger("UID shifting disabled (auto-detected Colima/Lima environment)")
			} else {
				opts.Logger("UID shifting disabled (configured via disable_shift option)")
			}
		}

		// Add disk devices BEFORE starting container
		// Determine container mount path - either /workspace (default) or same as host path
		containerWorkspacePath := "/workspace"
		if opts.PreserveWorkspacePath {
			// Validate that the path doesn't conflict with critical system directories
			cleanPath := filepath.Clean(opts.WorkspacePath)
			disallowedPrefixes := []string{
				"/etc", "/bin", "/sbin", "/usr", "/root", "/boot", "/sys", "/proc", "/dev", "/lib", "/lib64",
			}
			isDisallowed := false
			for _, prefix := range disallowedPrefixes {
				if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+"/") {
					isDisallowed = true
					break
				}
			}
			if isDisallowed {
				opts.Logger(fmt.Sprintf("Warning: preserve_workspace_path requested for %q conflicts with system directories; using /workspace instead", opts.WorkspacePath))
			} else {
				containerWorkspacePath = cleanPath
				opts.Logger(fmt.Sprintf("Adding workspace mount: %s -> %s (preserving host path)", opts.WorkspacePath, containerWorkspacePath))
			}
		}
		if containerWorkspacePath == "/workspace" && !opts.PreserveWorkspacePath {
			opts.Logger(fmt.Sprintf("Adding workspace mount: %s -> %s", opts.WorkspacePath, containerWorkspacePath))
		}
		result.ContainerWorkspacePath = containerWorkspacePath
		if err := result.Manager.MountDisk("workspace", opts.WorkspacePath, containerWorkspacePath, useShift, false); err != nil {
			return false, fmt.Errorf("failed to add workspace device: %w", err)
		}

		// Configure /tmp tmpfs size (prevent space exhaustion during builds/operations)
		if opts.LimitsConfig != nil && opts.LimitsConfig.Disk.TmpfsSize != "" {
			if err := result.Manager.SetTmpfsSize(opts.LimitsConfig.Disk.TmpfsSize); err != nil {
				opts.Logger(fmt.Sprintf("Warning: Failed to set /tmp size: %v", err))
			} else {
				opts.Logger(fmt.Sprintf("Set /tmp size to %s", opts.LimitsConfig.Disk.TmpfsSize))
			}
		}

		// Mount all configured directories
		if err := setupMounts(result.Manager, opts.MountConfig, useShift, opts.Logger); err != nil {
			return false, err
		}

		// Protect security-sensitive paths by mounting read-only (security feature)
		// This must be added after the workspace mount for the overlay to work
		if len(opts.ProtectedPaths) > 0 {
			if err := SetupSecurityMounts(result.Manager, opts.WorkspacePath, containerWorkspacePath, opts.ProtectedPaths, useShift); err != nil {
				opts.Logger(fmt.Sprintf("Warning: Failed to setup security mounts: %v", err))
				// Non-fatal: continue even if protection fails
			} else {
				// Log which paths were actually protected
				protectedPaths := GetProtectedPathsForLogging(opts.WorkspacePath, opts.ProtectedPaths)
				if len(protectedPaths) > 0 {
					opts.Logger(fmt.Sprintf("Protected paths (mounted read-only): %s", strings.Join(protectedPaths, ", ")))
				}
			}
		}

		// Apply resource limits before starting (if configured)
		if opts.LimitsConfig != nil && hasLimits(opts.LimitsConfig) {
			opts.Logger("Applying resource limits...")
			applyOpts := limits.ApplyOptions{
				ContainerName: result.ContainerName,
				CPU: limits.CPULimits{
					Count:     opts.LimitsConfig.CPU.Count,
					Allowance: opts.LimitsConfig.CPU.Allowance,
					Priority:  opts.LimitsConfig.CPU.Priority,
				},
				Memory: limits.MemoryLimits{
					Limit:   opts.LimitsConfig.Memory.Limit,
					Enforce: opts.LimitsConfig.Memory.Enforce,
					Swap:    opts.LimitsConfig.Memory.Swap,
				},
				Disk: limits.DiskLimits{
					Read:     opts.LimitsConfig.Disk.Read,
					Write:    opts.LimitsConfig.Disk.Write,
					Max:      opts.LimitsConfig.Disk.Max,
					Priority: opts.LimitsConfig.Disk.Priority,
				},
				Runtime: limits.RuntimeLimits{
					MaxProcesses: opts.LimitsConfig.Runtime.MaxProcesses,
				},
				Project: opts.IncusProject,
			}
			if err := limits.ApplyResourceLimits(applyOpts); err != nil {
				return false, fmt.Errorf("failed to apply resource limits: %w", err)
			}
		}

		// Now start the container
		opts.Logger("Starting container...")
		if err := result.Manager.Start(); err != nil {
			return false, fmt.Errorf("failed to start container: %w", err)
		}
	}

	return skipLaunch, nil
}

// hasLimits checks if any limits are configured
func hasLimits(cfg *config.LimitsConfig) bool {
	if cfg == nil {
		return false
	}

	// Check if any limit is set (non-empty strings or non-zero integers)
	return cfg.CPU.Count != "" ||
		cfg.CPU.Allowance != "" ||
		cfg.CPU.Priority != 0 ||
		cfg.Memory.Limit != "" ||
		cfg.Memory.Enforce != "" ||
		cfg.Memory.Swap != "" ||
		cfg.Disk.Read != "" ||
		cfg.Disk.Write != "" ||
		cfg.Disk.Max != "" ||
		cfg.Disk.Priority != 0 ||
		cfg.Runtime.MaxProcesses != 0
}
