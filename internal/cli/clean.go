package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bketelsen/clincus/internal/config"
	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/session"
	"github.com/spf13/cobra"
)

var (
	cleanAll      bool
	cleanForce    bool
	cleanSessions bool
	cleanOrphans  bool
	cleanDryRun   bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Cleanup containers, sessions, and orphaned resources",
	Long: `Cleanup stopped containers, old session data, and orphaned system resources.

By default, cleans only stopped containers. Use flags to control what gets cleaned.

Examples:
  clincus clean                    # Clean stopped containers
  clincus clean --sessions         # Clean saved session data
  clincus clean --orphans          # Clean orphaned stopped containers
  clincus clean --all              # Clean everything
  clincus clean --all --force      # Clean without confirmation
  clincus clean --orphans --dry-run # Show what orphans would be cleaned
`,
	RunE: cleanCommand,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Clean all containers, sessions, and orphaned resources")
	cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "Skip confirmation prompts")
	cleanCmd.Flags().BoolVar(&cleanSessions, "sessions", false, "Clean saved session data")
	cleanCmd.Flags().BoolVar(&cleanOrphans, "orphans", false, "Clean orphaned stopped containers")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
}

func cleanCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get configured tool to determine tool-specific sessions directory
	toolInstance, err := getConfiguredTool(cfg)
	if err != nil {
		return err
	}

	// Get tool-specific sessions directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".clincus")
	sessionsDir := session.GetSessionsDir(baseDir, toolInstance)

	cleaned := 0

	// Clean stopped containers
	if cleanAll || (!cleanSessions) {
		count, cancelled, err := cleanStoppedContainers()
		if err != nil {
			return err
		}
		if cancelled {
			return nil
		}
		cleaned += count
	}

	// Clean saved sessions
	if cleanAll || cleanSessions {
		count, cancelled, err := cleanSavedSessions(sessionsDir)
		if err != nil {
			return err
		}
		if cancelled {
			return nil
		}
		cleaned += count
	}

	// Clean orphaned resources (veths and firewall rules)
	if cleanAll || cleanOrphans {
		count, cancelled := cleanOrphanedResources()
		if cancelled {
			return nil
		}
		cleaned += count
	}

	if cleanDryRun {
		fmt.Println("\n[Dry run] No changes made.")
		return nil
	}

	if cleaned > 0 {
		fmt.Printf("\n✓ Cleaned %d item(s)\n", cleaned)
	} else {
		fmt.Println("\nNothing to clean.")
	}

	return nil
}

// cleanStoppedContainers finds and removes stopped containers.
// Returns (count cleaned, was cancelled, error).
func cleanStoppedContainers() (int, bool, error) {
	fmt.Println("Checking for stopped clincus containers...")

	containers, err := listActiveContainers()
	if err != nil {
		return 0, false, fmt.Errorf("failed to list containers: %w", err)
	}

	stoppedContainers := []string{}
	for _, c := range containers {
		if c.Status == "Stopped" || c.Status == "STOPPED" {
			stoppedContainers = append(stoppedContainers, c.Name)
		}
	}

	if len(stoppedContainers) == 0 {
		fmt.Println("  (no stopped containers found)")
		return 0, false, nil
	}

	fmt.Printf("Found %d stopped container(s):\n", len(stoppedContainers))
	for _, name := range stoppedContainers {
		fmt.Printf("  - %s\n", name)
	}

	if cleanDryRun {
		return 0, false, nil
	}

	if !cleanForce {
		fmt.Print("\nDelete these containers? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return 0, true, nil
		}
	}

	cleaned := 0
	for _, name := range stoppedContainers {
		fmt.Printf("Deleting container %s...\n", name)
		mgr := container.NewManager(name)
		if err := mgr.Delete(true); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete %s: %v\n", name, err)
		} else {
			cleaned++
		}
	}

	return cleaned, false, nil
}

// cleanSavedSessions finds and removes saved session data.
// Returns (count cleaned, was cancelled, error).
func cleanSavedSessions(sessionsDir string) (int, bool, error) {
	fmt.Println("\nChecking for saved session data...")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil && !os.IsNotExist(err) {
		return 0, false, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	sessionDirs := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			sessionDirs = append(sessionDirs, entry.Name())
		}
	}

	if len(sessionDirs) == 0 {
		fmt.Println("  (no saved sessions found)")
		return 0, false, nil
	}

	fmt.Printf("Found %d session(s):\n", len(sessionDirs))
	for _, name := range sessionDirs {
		fmt.Printf("  - %s\n", name)
	}

	if cleanDryRun {
		return 0, false, nil
	}

	if !cleanForce {
		fmt.Print("\nDelete all session data? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return 0, true, nil
		}
	}

	cleaned := 0
	for _, name := range sessionDirs {
		sessionPath := filepath.Join(sessionsDir, name)
		fmt.Printf("Deleting session %s...\n", name)
		if err := os.RemoveAll(sessionPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete %s: %v\n", name, err)
		} else {
			cleaned++
		}
	}

	return cleaned, false, nil
}

// cleanOrphanedResources finds and removes orphaned stopped containers.
// Returns (count cleaned, was cancelled).
func cleanOrphanedResources() (int, bool) {
	fmt.Println("\nScanning for orphaned stopped containers...")

	containers, err := listActiveContainers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to list containers: %v\n", err)
		return 0, false
	}

	stoppedContainers := []string{}
	for _, c := range containers {
		if c.Status == "Stopped" || c.Status == "STOPPED" {
			stoppedContainers = append(stoppedContainers, c.Name)
		}
	}

	if len(stoppedContainers) == 0 {
		fmt.Println("  (no orphaned resources found)")
		return 0, false
	}

	fmt.Printf("Found %d orphaned stopped container(s):\n", len(stoppedContainers))
	for _, name := range stoppedContainers {
		fmt.Printf("  - %s\n", name)
	}

	if cleanDryRun {
		return 0, false
	}

	if !cleanForce {
		fmt.Print("\nClean up orphaned containers? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return 0, true
		}
	}

	cleaned := 0
	for _, name := range stoppedContainers {
		fmt.Printf("Deleting container %s...\n", name)
		mgr := container.NewManager(name)
		if err := mgr.Delete(true); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete %s: %v\n", name, err)
		} else {
			cleaned++
		}
	}

	return cleaned, false
}
