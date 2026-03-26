package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/tool"
)

// configureToolAccess handles resume/restore, credential injection, and CLI tool
// config setup. Steps 9-11 of Setup().
// Note: this function has no error return -- all errors are logged as warnings.
func configureToolAccess(opts *SetupOptions, result *SetupResult, skipLaunch bool) {
	// 9. When resuming: restore session data if container was recreated, then inject credentials
	// Skip if tool uses ENV-based auth (no config directory and not file-based)
	isFileBased := func(t tool.Tool) bool {
		twh, ok := t.(tool.ToolWithHomeConfigFile)
		return ok && twh.HomeConfigFileName() != ""
	}

	if opts.ResumeFromID != "" && opts.Tool != nil &&
		(opts.Tool.ConfigDirName() != "" || isFileBased(opts.Tool)) {
		// If we launched a new container (not reusing persistent one), restore config from saved session
		// Only for directory-based tools (claude-style) - file-based tools (opencode) store sessions
		// in the workspace which is already bind-mounted
		if !skipLaunch && opts.SessionsDir != "" && opts.Tool.ConfigDirName() != "" {
			if err := restoreSessionData(result.Manager, opts.ResumeFromID, result.HomeDir, opts.SessionsDir, opts.Tool, opts.Logger); err != nil {
				opts.Logger(fmt.Sprintf("Warning: Could not restore session data: %v", err))
			}
		}

		// Always inject fresh credentials when resuming (whether persistent container or restored session)
		if opts.CLIConfigPath != "" {
			if twh, ok := opts.Tool.(tool.ToolWithHomeConfigFile); ok {
				setupHomeConfigFile(result.Manager, opts.CLIConfigPath, result.HomeDir, twh, opts.Tool, opts.Logger)
			} else {
				if err := injectCredentials(result.Manager, opts.CLIConfigPath, result.HomeDir, opts.Tool, opts.Logger); err != nil {
					opts.Logger(fmt.Sprintf("Warning: Could not inject credentials: %v", err))
				}
			}
		}
	}

	// 10. Workspace and configured mounts are already mounted (added before container start in step 5)
	if skipLaunch {
		opts.Logger("Reusing existing workspace and mount configurations")
	}

	// 11. Setup CLI tool config (skip if resuming - config already restored)
	if opts.Tool != nil {
		if twh, ok := opts.Tool.(tool.ToolWithHomeConfigFile); ok {
			// File-based config injection (opencode-style)
			if opts.CLIConfigPath != "" && opts.ResumeFromID == "" && !skipLaunch {
				setupHomeConfigFile(result.Manager, opts.CLIConfigPath, result.HomeDir, twh, opts.Tool, opts.Logger)
			} else if opts.ResumeFromID != "" {
				opts.Logger(fmt.Sprintf("Resuming session - using restored %s config", opts.Tool.Name()))
			}
		} else if opts.Tool.ConfigDirName() != "" {
			// Directory-based config injection (claude-style)
			if opts.CLIConfigPath != "" && opts.ResumeFromID == "" {
				// Check if host config directory exists
				if _, err := os.Stat(opts.CLIConfigPath); err == nil {
					// Copy and inject settings (but only if NOT resuming)
					// Only run on first launch, not when restarting persistent container
					if !skipLaunch {
						opts.Logger(fmt.Sprintf("Setting up %s config...", opts.Tool.Name()))
						if err := setupCLIConfig(result.Manager, opts.CLIConfigPath, result.HomeDir, opts.Tool, opts.Logger); err != nil {
							opts.Logger(fmt.Sprintf("Warning: Failed to setup %s config: %v", opts.Tool.Name(), err))
						}
					} else {
						opts.Logger(fmt.Sprintf("Reusing existing %s config (persistent container)", opts.Tool.Name()))
					}
				} else if !os.IsNotExist(err) {
					// Note: in the original Setup(), this was `return nil, fmt.Errorf(...)` but
					// configureToolAccess has no error return. However, the original code path
					// here would return an error. We preserve the error message as a log warning
					// since this function signature doesn't return errors.
					opts.Logger(fmt.Sprintf("Warning: failed to check %s config directory: %v", opts.Tool.Name(), err))
				}
			} else if opts.ResumeFromID != "" {
				opts.Logger(fmt.Sprintf("Resuming session - using restored %s config", opts.Tool.Name()))
			}
		} else {
			opts.Logger(fmt.Sprintf("Tool %s uses ENV-based auth, skipping config setup", opts.Tool.Name()))
		}
	}
}

// buildJSONMergeCommand builds a Python one-liner that deep-merges a JSON
// string into an existing JSON file.  Dict values are merged with
// setdefault+update; all other values are overwritten.
func buildJSONMergeCommand(filePath, settingsJSON string) string {
	escapedJSON := strings.ReplaceAll(settingsJSON, "'", "'\"'\"'")
	return fmt.Sprintf(
		`python3 -c 'import json; f=open("%s","r+"); d=json.load(f); updates=json.loads('"'"'%s'"'"'); [d.setdefault(k,{}).update(v) if isinstance(v,dict) and isinstance(d.get(k),dict) else d.__setitem__(k,v) for k,v in updates.items()]; f.seek(0); json.dump(d,f,indent=2); f.truncate()'`,
		filePath, escapedJSON,
	)
}

// restoreSessionData restores tool config directory from a saved session
// Used when resuming a non-persistent session (container was deleted and recreated)
func restoreSessionData(mgr *container.Manager, resumeID, homeDir, sessionsDir string, t tool.Tool, logger func(string)) error {
	configDirName := t.ConfigDirName()
	sourceConfigDir := filepath.Join(sessionsDir, resumeID, configDirName)

	// Check if directory exists
	if info, err := os.Stat(sourceConfigDir); err != nil || !info.IsDir() {
		return fmt.Errorf("no saved session data found for %s", resumeID)
	}

	logger(fmt.Sprintf("Restoring session data from %s", resumeID))

	// Push config directory to container
	// PushDirectory extracts the parent from the path and pushes to create the directory there
	// So we pass the full destination path where the config dir should end up
	destConfigPath := filepath.Join(homeDir, configDirName)
	if err := mgr.PushDirectory(sourceConfigDir, destConfigPath); err != nil {
		return fmt.Errorf("failed to push %s directory: %w", configDirName, err)
	}

	// Fix ownership if running as non-root user
	if homeDir != "/root" {
		statePath := destConfigPath
		if err := mgr.Chown(statePath, container.CodeUID, container.CodeUID); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	logger("Session data restored successfully")
	return nil
}

// injectCredentials copies credentials and essential config from host to container when resuming
// This ensures fresh authentication while preserving the session conversation history
func injectCredentials(mgr *container.Manager, hostCLIConfigPath, homeDir string, t tool.Tool, logger func(string)) error {
	logger("Injecting fresh credentials and config for session resume...")

	configDirName := t.ConfigDirName()

	// Copy .credentials.json from host to container
	credentialsPath := filepath.Join(hostCLIConfigPath, ".credentials.json")
	if _, err := os.Stat(credentialsPath); err != nil {
		return fmt.Errorf("credentials file not found: %w", err)
	}

	destCredentials := filepath.Join(homeDir, configDirName, ".credentials.json")
	if err := mgr.PushFile(credentialsPath, destCredentials); err != nil {
		return fmt.Errorf("failed to push credentials: %w", err)
	}

	// Fix ownership if running as non-root user
	if homeDir != "/root" {
		if err := mgr.Chown(destCredentials, container.CodeUID, container.CodeUID); err != nil {
			return fmt.Errorf("failed to set credentials ownership: %w", err)
		}
	}

	// Get sandbox settings from tool
	sandboxSettings := t.GetSandboxSettings()
	if len(sandboxSettings) > 0 {
		// Get the state config filename (e.g., ".claude.json" or ".aider.json")
		stateConfigFilename := fmt.Sprintf(".%s.json", t.Name())
		stateConfigPath := filepath.Join(filepath.Dir(hostCLIConfigPath), stateConfigFilename)

		if _, err := os.Stat(stateConfigPath); err == nil {
			logger(fmt.Sprintf("Copying %s for session resume...", stateConfigFilename))
			stateJsonDest := filepath.Join(homeDir, stateConfigFilename)
			if err := mgr.PushFile(stateConfigPath, stateJsonDest); err != nil {
				logger(fmt.Sprintf("Warning: Failed to copy %s: %v", stateConfigFilename, err))
			} else {
				// Inject sandbox settings using tool's GetSandboxSettings()
				logger(fmt.Sprintf("Injecting sandbox settings into %s...", stateConfigFilename))
				settingsJSON, err := buildJSONFromSettings(sandboxSettings)
				if err != nil {
					logger(fmt.Sprintf("Warning: Failed to build JSON from settings: %v", err))
				} else {
					injectCmd := buildJSONMergeCommand(stateJsonDest, settingsJSON)
					if _, err := mgr.ExecCommand(injectCmd, container.ExecCommandOptions{Capture: true}); err != nil {
						logger(fmt.Sprintf("Warning: Failed to inject settings into %s: %v", stateConfigFilename, err))
					}
				}

				// Fix ownership if running as non-root user
				if homeDir != "/root" {
					if err := mgr.Chown(stateJsonDest, container.CodeUID, container.CodeUID); err != nil {
						logger(fmt.Sprintf("Warning: Failed to set %s ownership: %v", stateConfigFilename, err))
					}
				}
			}
		}
	}

	logger("Credentials and config injected successfully")
	return nil
}

// setupCLIConfig copies tool config directory and injects sandbox settings
func setupCLIConfig(mgr *container.Manager, hostCLIConfigPath, homeDir string, t tool.Tool, logger func(string)) error {
	configDirName := t.ConfigDirName()
	stateDir := filepath.Join(homeDir, configDirName)

	// Create config directory in container
	logger(fmt.Sprintf("Creating %s directory in container...", configDirName))
	mkdirCmd := fmt.Sprintf("mkdir -p %s", stateDir)
	if _, err := mgr.ExecCommand(mkdirCmd, container.ExecCommandOptions{Capture: true}); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", configDirName, err)
	}

	// Determine which files/dirs to copy — tool can override via ToolWithEssentialFiles
	var essentialFiles []string
	var essentialDirs []string
	if tef, ok := t.(tool.ToolWithEssentialFiles); ok {
		essentialFiles = tef.EssentialFiles()
		essentialDirs = tef.EssentialDirs()
	} else {
		// Default: Claude's files (backward compatible)
		essentialFiles = []string{".credentials.json", "config.yml", "settings.json"}
		essentialDirs = []string{"plugins", "hooks"}
	}

	logger(fmt.Sprintf("Copying essential CLI config files from %s", hostCLIConfigPath))
	for _, filename := range essentialFiles {
		srcPath := filepath.Join(hostCLIConfigPath, filename)
		if _, err := os.Stat(srcPath); err == nil {
			destPath := filepath.Join(stateDir, filename)
			logger(fmt.Sprintf("  - Copying %s", filename))
			if err := mgr.PushFile(srcPath, destPath); err != nil {
				logger(fmt.Sprintf("  - Warning: Failed to copy %s: %v", filename, err))
			}
		} else {
			logger(fmt.Sprintf("  - Skipping %s (not found)", filename))
		}
	}

	// Copy essential subdirectories
	for _, dirname := range essentialDirs {
		srcDir := filepath.Join(hostCLIConfigPath, dirname)
		destDir := filepath.Join(stateDir, dirname)
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			logger(fmt.Sprintf("  - Copying %s/ directory", dirname))
			if err := mgr.PushDirectory(srcDir, destDir); err != nil {
				logger(fmt.Sprintf("  - Warning: Failed to copy %s/ directory: %v", dirname, err))
			}
		} else {
			logger(fmt.Sprintf("  - Skipping %s/ (not found)", dirname))
		}
	}

	// Rewrite host home paths to container home paths across all copied config files
	// e.g., /home/bjk/.claude/hooks/... → /home/code/.claude/hooks/...
	hostHomeDir := filepath.Dir(hostCLIConfigPath)
	if hostHomeDir != homeDir {
		logger(fmt.Sprintf("Rewriting paths in config files: %s → %s", hostHomeDir, homeDir))
		rewriteCmd := fmt.Sprintf(
			`find %s -name '*.json' -exec sed -i 's|%s|%s|g' {} +`,
			stateDir, hostHomeDir, homeDir,
		)
		if _, err := mgr.ExecCommand(rewriteCmd, container.ExecCommandOptions{Capture: true}); err != nil {
			logger(fmt.Sprintf("Warning: Failed to rewrite paths in config files: %v", err))
		}
	}

	// Get sandbox settings from tool and merge into settings.json if needed
	sandboxSettings := t.GetSandboxSettings()
	if len(sandboxSettings) > 0 {
		settingsPath := filepath.Join(stateDir, "settings.json")
		logger("Merging sandbox settings into settings.json...")
		settingsJSON, err := buildJSONFromSettings(sandboxSettings)
		if err != nil {
			logger(fmt.Sprintf("Warning: Failed to build JSON from settings: %v", err))
		} else {
			// Check if settings.json exists in container
			checkCmd := fmt.Sprintf("test -f %s && echo exists || echo missing", settingsPath)
			checkResult, err := mgr.ExecCommand(checkCmd, container.ExecCommandOptions{Capture: true})

			if err != nil || strings.TrimSpace(checkResult) == "missing" {
				// File doesn't exist, create it with sandbox settings
				logger("settings.json not found in container, creating with sandbox settings")
				settingsBytes, err := json.MarshalIndent(sandboxSettings, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal sandbox settings: %w", err)
				}
				if err := mgr.CreateFile(settingsPath, string(settingsBytes)+"\n"); err != nil {
					return fmt.Errorf("failed to create settings.json: %w", err)
				}
			} else {
				// File exists, merge sandbox settings into it
				logger("Merging sandbox settings into existing settings.json")
				injectCmd := buildJSONMergeCommand(settingsPath, settingsJSON)
				if _, err := mgr.ExecCommand(injectCmd, container.ExecCommandOptions{Capture: true}); err != nil {
					logger(fmt.Sprintf("Warning: Failed to inject settings into settings.json: %v", err))
				} else {
					logger("Successfully merged sandbox settings into settings.json")
				}
			}
		}
		logger(fmt.Sprintf("%s config copied and sandbox settings merged into settings.json", t.Name()))
	} else {
		logger(fmt.Sprintf("%s config copied (no sandbox settings needed)", t.Name()))
	}

	// Copy and modify tool state config file (e.g., .claude.json, .aider.json)
	// This is a sibling file to the config directory
	stateConfigFilename := fmt.Sprintf(".%s.json", t.Name())
	stateConfigPath := filepath.Join(filepath.Dir(hostCLIConfigPath), stateConfigFilename)
	logger(fmt.Sprintf("Checking for %s at: %s", stateConfigFilename, stateConfigPath))

	if info, err := os.Stat(stateConfigPath); err == nil {
		logger(fmt.Sprintf("Found %s (size: %d bytes), copying to container...", stateConfigFilename, info.Size()))
		stateJsonDest := filepath.Join(homeDir, stateConfigFilename)

		// Push the file to container
		if err := mgr.PushFile(stateConfigPath, stateJsonDest); err != nil {
			return fmt.Errorf("failed to copy %s: %w", stateConfigFilename, err)
		}
		logger(fmt.Sprintf("%s copied to %s", stateConfigFilename, stateJsonDest))

		// Inject sandbox settings if tool provides them
		if len(sandboxSettings) > 0 {
			logger(fmt.Sprintf("Injecting sandbox settings into %s...", stateConfigFilename))
			settingsJSON, err := buildJSONFromSettings(sandboxSettings)
			if err != nil {
				logger(fmt.Sprintf("Warning: Failed to build JSON from settings: %v", err))
			} else {
				injectCmd := buildJSONMergeCommand(stateJsonDest, settingsJSON)
				if _, err := mgr.ExecCommand(injectCmd, container.ExecCommandOptions{Capture: true}); err != nil {
					logger(fmt.Sprintf("Warning: Failed to inject settings into %s: %v", stateConfigFilename, err))
				} else {
					logger(fmt.Sprintf("Successfully injected sandbox settings into %s", stateConfigFilename))
				}
			}
		}

		// Fix ownership if running as non-root user
		if homeDir != "/root" {
			logger(fmt.Sprintf("Fixing ownership of %s to %d:%d", stateConfigFilename, container.CodeUID, container.CodeUID))
			if err := mgr.Chown(stateJsonDest, container.CodeUID, container.CodeUID); err != nil {
				return fmt.Errorf("failed to set %s ownership: %w", stateConfigFilename, err)
			}
		}

		// Fix ownership of entire config directory recursively
		if homeDir != "/root" {
			logger(fmt.Sprintf("Fixing ownership of entire %s directory to %d:%d", configDirName, container.CodeUID, container.CodeUID))
			chownCmd := fmt.Sprintf("chown -R %d:%d %s", container.CodeUID, container.CodeUID, stateDir)
			if _, err := mgr.ExecCommand(chownCmd, container.ExecCommandOptions{Capture: true}); err != nil {
				return fmt.Errorf("failed to set %s directory ownership: %w", configDirName, err)
			}
		}

		logger(fmt.Sprintf("%s setup complete", stateConfigFilename))
	} else if os.IsNotExist(err) {
		// Host file doesn't exist, but we may still need to create it in the container
		// to inject sandbox settings (e.g., effort level for Claude)
		if len(sandboxSettings) > 0 {
			logger(fmt.Sprintf("%s not found on host, creating in container with sandbox settings...", stateConfigFilename))
			stateJsonDest := filepath.Join(homeDir, stateConfigFilename)

			settingsJSON, err := buildJSONFromSettings(sandboxSettings)
			if err != nil {
				logger(fmt.Sprintf("Warning: Failed to build JSON from settings: %v", err))
			} else {
				// Create new file with sandbox settings
				createCmd := fmt.Sprintf("echo '%s' > %s", settingsJSON, stateJsonDest)
				if _, err := mgr.ExecCommand(createCmd, container.ExecCommandOptions{Capture: true}); err != nil {
					logger(fmt.Sprintf("Warning: Failed to create %s: %v", stateConfigFilename, err))
				} else {
					logger(fmt.Sprintf("Created %s with sandbox settings", stateConfigFilename))
				}

				// Fix ownership if running as non-root user
				if homeDir != "/root" {
					if err := mgr.Chown(stateJsonDest, container.CodeUID, container.CodeUID); err != nil {
						logger(fmt.Sprintf("Warning: Failed to set %s ownership: %v", stateConfigFilename, err))
					}
				}
			}
		} else {
			logger(fmt.Sprintf("%s not found at %s and no sandbox settings needed, skipping", stateConfigFilename, stateConfigPath))
		}
	} else {
		return fmt.Errorf("failed to check %s: %w", stateConfigFilename, err)
	}

	return nil
}

// setupHomeConfigFile handles config injection for tools that use a single
// home-dir JSON file (e.g., ~/.opencode.json).
func setupHomeConfigFile(mgr *container.Manager, hostConfigFilePath, homeDir string,
	twh tool.ToolWithHomeConfigFile, t tool.Tool, logger func(string),
) {
	destPath := filepath.Join(homeDir, twh.HomeConfigFileName())

	// Copy host config if it exists
	if _, err := os.Stat(hostConfigFilePath); err == nil {
		logger(fmt.Sprintf("Copying %s from host...", twh.HomeConfigFileName()))
		if err := mgr.PushFile(hostConfigFilePath, destPath); err != nil {
			logger(fmt.Sprintf("Warning: Failed to copy %s: %v", twh.HomeConfigFileName(), err))
		}
	}

	// Inject sandbox settings (merge into existing or create fresh)
	sandboxSettings := t.GetSandboxSettings()
	if len(sandboxSettings) > 0 {
		logger(fmt.Sprintf("Injecting sandbox settings into %s...", twh.HomeConfigFileName()))

		// Check if file exists in container
		checkCmd := fmt.Sprintf("test -f %s && echo exists || echo missing", destPath)
		result, _ := mgr.ExecCommand(checkCmd, container.ExecCommandOptions{Capture: true})
		if strings.TrimSpace(result) == "missing" {
			// Create fresh config with sandbox settings
			settingsBytes, err := json.MarshalIndent(sandboxSettings, "", "  ")
			if err != nil {
				logger(fmt.Sprintf("Warning: Failed to marshal sandbox settings: %v", err))
			} else {
				if err := mgr.CreateFile(destPath, string(settingsBytes)+"\n"); err != nil {
					logger(fmt.Sprintf("Warning: Failed to create %s: %v", twh.HomeConfigFileName(), err))
				}
			}
		} else {
			// Merge sandbox settings into existing config
			settingsJSON, err := buildJSONFromSettings(sandboxSettings)
			if err != nil {
				logger(fmt.Sprintf("Warning: Failed to build JSON from settings: %v", err))
			} else {
				mergeCmd := buildJSONMergeCommand(destPath, settingsJSON)
				if _, err := mgr.ExecCommand(mergeCmd, container.ExecCommandOptions{Capture: true}); err != nil {
					logger(fmt.Sprintf("Warning: Failed to inject settings into %s: %v", twh.HomeConfigFileName(), err))
				}
			}
		}

		// Fix ownership
		if homeDir != "/root" {
			if err := mgr.Chown(destPath, container.CodeUID, container.CodeUID); err != nil {
				logger(fmt.Sprintf("Warning: Failed to set %s ownership: %v", twh.HomeConfigFileName(), err))
			}
		}
	}
}
