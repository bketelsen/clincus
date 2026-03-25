package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/tool"
)

// CleanupOptions contains options for cleaning up a session
type CleanupOptions struct {
	ContainerName string
	SessionID     string    // Clincus session ID for saving tool config data
	Persistent    bool      // If true, stop but don't delete container
	SessionsDir   string    // e.g., ~/.clincus/sessions-claude
	SaveSession   bool      // Whether to save tool config directory
	Workspace     string    // Workspace directory path
	Tool          tool.Tool // AI coding tool being used
	Logger        func(string)
}

// Cleanup stops and deletes a container, optionally saving session data
func Cleanup(opts CleanupOptions) error {
	// Default logger
	if opts.Logger == nil {
		opts.Logger = func(msg string) {
			fmt.Fprintf(os.Stderr, "[cleanup] %s\n", msg)
		}
	}

	if opts.ContainerName == "" {
		opts.Logger("No container to clean up")
		return nil
	}

	mgr := container.NewManager(opts.ContainerName)

	// Check if container exists
	// Containers are always launched as non-ephemeral, so they should exist even when stopped
	exists, err := mgr.Exists()
	if err != nil {
		opts.Logger(fmt.Sprintf("Warning: Could not check container existence: %v", err))
	}

	// Always save session data if container exists (works even from stopped containers)
	// This ensures --resume works regardless of how the user exited (including sudo shutdown 0)
	// Skip if tool uses ENV-based auth (no config directory to save)
	if opts.SaveSession && exists && opts.SessionID != "" && opts.SessionsDir != "" && opts.Tool != nil && opts.Tool.ConfigDirName() != "" {
		if err := saveSessionData(mgr, opts.SessionID, opts.Persistent, opts.Workspace, opts.SessionsDir, opts.Tool, opts.Logger); err != nil {
			opts.Logger(fmt.Sprintf("Warning: Failed to save session data: %v", err))
		}
	}

	// Record session stop in history
	home, _ := os.UserHomeDir()
	histPath := filepath.Join(home, ".clincus", "history.jsonl")
	hist := &History{Path: histPath}
	//nolint:errcheck // history recording failure is non-fatal
	_ = hist.RecordStop(opts.ContainerName, 0)

	// Handle container based on persistence mode
	if opts.Persistent {
		// Persistent mode: keep container for reuse (with all its data/modifications)
		if exists {
			opts.Logger("Container kept running - use 'clincus attach' to reconnect, 'clincus shutdown' to stop, or 'clincus kill' to force stop")
		} else {
			opts.Logger("Container was stopped but kept for reuse")
		}
	} else {
		// Non-persistent mode: behavior depends on how user exited
		// - If container is running (user typed 'exit' or detached): keep it running
		// - If container is stopped (user did 'sudo shutdown 0'): delete it
		if exists {
			// Check if container is stopped, with exponential backoff to handle shutdown delay
			// Poweroff/shutdown can take several seconds to complete
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			running := true
			backoff := 500 * time.Millisecond
			maxBackoff := 4 * time.Second

		loop:
			for running {
				select {
				case <-ctx.Done():
					// Timeout -- keep container running (non-destructive)
					opts.Logger("Container still running after timeout, keeping it alive")
					break loop
				case <-time.After(backoff):
					running, _ = mgr.Running()
					if backoff < maxBackoff {
						backoff *= 2
					}
				}
			}

			if running {
				// Container still running - user exited normally, keep it for potential re-attach
				opts.Logger("Container kept running - use 'clincus attach' to reconnect, 'clincus shutdown' to stop, or 'clincus kill' to force stop")
			} else {
				// Container stopped (user did 'sudo shutdown 0') - delete it
				opts.Logger("Container was stopped, removing...")

				// Delete container
				if err := mgr.Delete(true); err != nil {
					opts.Logger(fmt.Sprintf("Warning: Failed to delete container: %v", err))
				} else {
					opts.Logger("Container removed (session data saved for --resume)")
				}
			}
		} else {
			opts.Logger("Container was already removed")
		}
	}

	return nil
}

// saveSessionData saves the tool config directory from the container
func saveSessionData(mgr *container.Manager, sessionID string, persistent bool, workspace string, sessionsDir string, t tool.Tool, logger func(string)) error {
	// Determine home directory
	// For clincus images, we always use /home/code
	// For other images, we use /root
	// Since we currently only support clincus images, always use /home/code
	homeDir := "/home/" + container.CodeUser

	configDirName := t.ConfigDirName()
	stateDir := filepath.Join(homeDir, configDirName)

	// Create local session directory
	localSessionDir := filepath.Join(sessionsDir, sessionID)
	if err := os.MkdirAll(localSessionDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	logger(fmt.Sprintf("Saving session data to %s", localSessionDir))

	// Remove old config directory if it exists (when resuming)
	localConfigDir := filepath.Join(localSessionDir, configDirName)
	if _, err := os.Stat(localConfigDir); err == nil {
		logger("Removing old session data before saving new state")
		if err := os.RemoveAll(localConfigDir); err != nil {
			return fmt.Errorf("failed to remove old %s directory: %w", configDirName, err)
		}
	}

	// Pull config directory from container
	// Note: incus file pull works on stopped containers, so we don't need to check if running
	// If config dir doesn't exist, PullDirectory will fail and we handle it gracefully
	if err := mgr.PullDirectory(stateDir, localConfigDir); err != nil {
		// Check if it's a "not found" error - this is expected if config dir doesn't exist
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No such file") {
			logger(fmt.Sprintf("No %s directory found in container", configDirName))
			return nil
		}
		return fmt.Errorf("failed to pull %s directory: %w", configDirName, err)
	}

	// Save metadata
	metadata := SessionMetadata{
		SessionID:     sessionID,
		ContainerName: mgr.ContainerName,
		Persistent:    persistent,
		Workspace:     workspace,
		SavedAt:       getCurrentTime(),
	}

	metadataPath := filepath.Join(localSessionDir, "metadata.json")
	if err := SaveMetadata(metadataPath, metadata); err != nil {
		// Non-fatal - session data is already saved
		logger(fmt.Sprintf("Warning: Failed to save metadata: %v", err))
	}

	logger("Session data saved successfully")
	return nil
}

// SessionMetadata contains information about a saved session
type SessionMetadata struct {
	SessionID     string `json:"session_id"`
	ContainerName string `json:"container_name"`
	Persistent    bool   `json:"persistent"`
	Workspace     string `json:"workspace"`
	SavedAt       string `json:"saved_at"`
}

// SaveMetadata saves session metadata to a JSON file
func SaveMetadata(path string, metadata SessionMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// getCurrentTime returns current time in RFC3339 format
func getCurrentTime() string {
	return time.Now().Format(time.RFC3339)
}

// SaveMetadataEarly saves session metadata at session start so clincus list can show correct status
func SaveMetadataEarly(sessionsDir, sessionID, containerName, workspace string, persistent bool) error {
	// Create session directory if it doesn't exist
	sessionDir := filepath.Join(sessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	metadata := SessionMetadata{
		SessionID:     sessionID,
		ContainerName: containerName,
		Persistent:    persistent,
		Workspace:     workspace,
		SavedAt:       getCurrentTime(),
	}

	metadataPath := filepath.Join(sessionDir, "metadata.json")
	return SaveMetadata(metadataPath, metadata)
}

// SessionExists checks if a session with the given ID exists and is valid
func SessionExists(sessionsDir, sessionID string) bool {
	statePath := filepath.Join(sessionsDir, sessionID, ".claude")
	info, err := os.Stat(statePath)
	return err == nil && info.IsDir()
}

// ListSavedSessions lists all saved sessions in the sessions directory
func ListSavedSessions(sessionsDir string) ([]string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it contains a .claude directory
			statePath := filepath.Join(sessionsDir, entry.Name(), ".claude")
			if info, err := os.Stat(statePath); err == nil && info.IsDir() {
				sessions = append(sessions, entry.Name())
			}
		}
	}

	return sessions, nil
}

// GetLatestSession returns the most recently saved session ID
func GetLatestSession(sessionsDir string) (string, error) {
	sessions, err := ListSavedSessions(sessionsDir)
	if err != nil {
		return "", err
	}

	if len(sessions) == 0 {
		return "", fmt.Errorf("no saved sessions found")
	}

	// Find the most recent session by reading metadata
	var latestSession string
	var latestTime time.Time

	for _, sessionID := range sessions {
		metadataPath := filepath.Join(sessionsDir, sessionID, "metadata.json")
		metadata, err := LoadSessionMetadata(metadataPath)
		if err != nil {
			continue // Skip sessions without valid metadata
		}

		savedTime, err := time.Parse(time.RFC3339, metadata.SavedAt)
		if err != nil {
			continue
		}

		if latestSession == "" || savedTime.After(latestTime) {
			latestSession = sessionID
			latestTime = savedTime
		}
	}

	if latestSession == "" {
		return "", fmt.Errorf("no valid sessions found")
	}

	return latestSession, nil
}

// GetLatestSessionForWorkspace returns the most recent session ID for a specific workspace
func GetLatestSessionForWorkspace(sessionsDir, workspacePath string) (string, error) {
	sessions, err := ListSavedSessions(sessionsDir)
	if err != nil {
		return "", err
	}

	if len(sessions) == 0 {
		return "", fmt.Errorf("no saved sessions found")
	}

	// Get the workspace hash to match against
	workspaceHash := WorkspaceHash(workspacePath)

	// Find the most recent session for this workspace
	var latestSession string
	var latestTime time.Time

	for _, sessionID := range sessions {
		metadataPath := filepath.Join(sessionsDir, sessionID, "metadata.json")
		metadata, err := LoadSessionMetadata(metadataPath)
		if err != nil {
			continue // Skip sessions without valid metadata
		}

		// Extract workspace hash from container name (format: claude-<hash>-<slot>)
		sessionHash, _, err := ParseContainerName(metadata.ContainerName)
		if err != nil {
			continue
		}

		// Only consider sessions from the same workspace
		if sessionHash != workspaceHash {
			continue
		}

		savedTime, err := time.Parse(time.RFC3339, metadata.SavedAt)
		if err != nil {
			continue
		}

		if latestSession == "" || savedTime.After(latestTime) {
			latestSession = sessionID
			latestTime = savedTime
		}
	}

	if latestSession == "" {
		return "", fmt.Errorf("no saved sessions found for workspace %s", workspacePath)
	}

	return latestSession, nil
}

// LoadSessionMetadata loads session metadata from a JSON file
func LoadSessionMetadata(path string) (*SessionMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata SessionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.SessionID == "" {
		return nil, fmt.Errorf("invalid metadata: missing session_id")
	}

	return &metadata, nil
}

// GetCLISessionID extracts the CLI tool's session ID from a saved clincus session.
// CLI tools store sessions in .claude/projects/-workspace/<session-id>.jsonl
// Returns empty string if no session found.
func GetCLISessionID(sessionsDir, coiSessionID string) string {
	projectsDir := filepath.Join(sessionsDir, coiSessionID, ".claude", "projects", "-workspace")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return ""
	}

	// Look for .jsonl files (session files)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".jsonl") {
			// Extract session ID from filename (remove .jsonl suffix)
			return strings.TrimSuffix(name, ".jsonl")
		}
	}

	return ""
}
