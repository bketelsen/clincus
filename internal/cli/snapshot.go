package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/session"
	"github.com/spf13/cobra"
)

// snapshotCmd is the parent command for all snapshot operations
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage container snapshots",
	Long: `Manage Incus container snapshots for checkpointing, rollback, and branching.

Snapshots capture the complete container state including all files and session data.
This enables agent workflows to experiment with different approaches and roll back
if needed.

Examples:
  clincus snapshot create                     # Create auto-named snapshot
  clincus snapshot create checkpoint-1        # Create named snapshot
  clincus snapshot create --stateful live     # Include process memory
  clincus snapshot list                       # List snapshots for current workspace
  clincus snapshot list --format json         # JSON output
  clincus snapshot restore checkpoint-1       # Restore from snapshot (requires confirmation)
  clincus snapshot restore checkpoint-1 -f    # Restore without confirmation
  clincus snapshot delete checkpoint-1        # Delete a snapshot
  clincus snapshot info checkpoint-1          # Show snapshot details
`,
}

// Flags for snapshot commands
var (
	snapshotContainer string
	snapshotFormat    string
	snapshotStateful  bool
	snapshotForce     bool
	snapshotAll       bool
)

// snapshotCreateCmd creates a new snapshot
var snapshotCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a container snapshot",
	Long: `Create a snapshot of the current container state.

If no name is provided, an auto-generated name will be used (snap-YYYYMMDD-HHMMSS).

Examples:
  clincus snapshot create                          # Auto-named snapshot
  clincus snapshot create checkpoint-1             # Named snapshot
  clincus snapshot create --stateful live          # Include process memory state
  clincus snapshot create -c clincus-abc-1 backup  # Specific container
`,
	Args: cobra.MaximumNArgs(1),
	RunE: snapshotCreateCommand,
}

// snapshotListCmd lists snapshots
var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List container snapshots",
	Long: `List snapshots for a container.

By default, lists snapshots for the current workspace's container.

Examples:
  clincus snapshot list                              # Current workspace container
  clincus snapshot list -c clincus-abc-1             # Specific container
  clincus snapshot list --all                        # All Clincus containers
  clincus snapshot list --format json                # JSON output
`,
	RunE: snapshotListCommand,
}

// snapshotRestoreCmd restores from a snapshot
var snapshotRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore container from a snapshot",
	Long: `Restore a container to a previous snapshot state.

IMPORTANT: This operation overwrites the current container state.
The container must be stopped before restore.

Examples:
  clincus snapshot restore checkpoint-1                     # Restore (with confirmation)
  clincus snapshot restore checkpoint-1 -f                  # Restore without confirmation
  clincus snapshot restore checkpoint-1 -c clincus-abc-1    # Specific container
`,
	Args: cobra.ExactArgs(1),
	RunE: snapshotRestoreCommand,
}

// snapshotDeleteCmd deletes snapshots
var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a container snapshot",
	Long: `Delete a snapshot from a container.

Examples:
  clincus snapshot delete checkpoint-1        # Delete specific snapshot
  clincus snapshot delete --all               # Delete all snapshots (with confirmation)
  clincus snapshot delete --all -f            # Delete all without confirmation
`,
	Args: cobra.MaximumNArgs(1),
	RunE: snapshotDeleteCommand,
}

// snapshotInfoCmd shows snapshot details
var snapshotInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show snapshot details",
	Long: `Show detailed information about a snapshot.

Examples:
  clincus snapshot info checkpoint-1                  # Show details
  clincus snapshot info checkpoint-1 --format json    # JSON output
`,
	Args: cobra.ExactArgs(1),
	RunE: snapshotInfoCommand,
}

func init() {
	// Add flags to create command
	snapshotCreateCmd.Flags().StringVarP(&snapshotContainer, "container", "c", "", "Container name (default: auto-detect from workspace)")
	snapshotCreateCmd.Flags().BoolVar(&snapshotStateful, "stateful", false, "Include process memory state in snapshot")

	// Add flags to list command
	snapshotListCmd.Flags().StringVarP(&snapshotContainer, "container", "c", "", "Container name (default: auto-detect from workspace)")
	snapshotListCmd.Flags().StringVar(&snapshotFormat, "format", "text", "Output format: text or json")
	snapshotListCmd.Flags().BoolVar(&snapshotAll, "all", false, "List snapshots for all Clincus containers")

	// Add flags to restore command
	snapshotRestoreCmd.Flags().StringVarP(&snapshotContainer, "container", "c", "", "Container name (default: auto-detect from workspace)")
	snapshotRestoreCmd.Flags().BoolVarP(&snapshotForce, "force", "f", false, "Skip confirmation prompt")
	snapshotRestoreCmd.Flags().BoolVar(&snapshotStateful, "stateful", false, "Restore with process memory state")

	// Add flags to delete command
	snapshotDeleteCmd.Flags().StringVarP(&snapshotContainer, "container", "c", "", "Container name (default: auto-detect from workspace)")
	snapshotDeleteCmd.Flags().BoolVarP(&snapshotForce, "force", "f", false, "Skip confirmation prompt")
	snapshotDeleteCmd.Flags().BoolVar(&snapshotAll, "all", false, "Delete all snapshots")

	// Add flags to info command
	snapshotInfoCmd.Flags().StringVarP(&snapshotContainer, "container", "c", "", "Container name (default: auto-detect from workspace)")
	snapshotInfoCmd.Flags().StringVar(&snapshotFormat, "format", "text", "Output format: text or json")

	// Add subcommands to snapshot command
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRestoreCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotCmd.AddCommand(snapshotInfoCmd)
}

// resolveContainer resolves the container name using the following strategy:
// 1. Use --container flag if provided
// 2. Check CLINCUS_CONTAINER environment variable
// 3. Find container for current workspace
func resolveContainer() (string, error) {
	// 1. Use --container flag if provided
	if snapshotContainer != "" {
		// Verify container exists
		mgr := container.NewManager(snapshotContainer)
		exists, err := mgr.Exists()
		if err != nil {
			return "", fmt.Errorf("failed to check container: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("container '%s' not found", snapshotContainer)
		}
		return snapshotContainer, nil
	}

	// 2. Check CLINCUS_CONTAINER environment variable
	if envContainer := os.Getenv("CLINCUS_CONTAINER"); envContainer != "" {
		mgr := container.NewManager(envContainer)
		exists, err := mgr.Exists()
		if err != nil {
			return "", fmt.Errorf("failed to check container: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("container '%s' from CLINCUS_CONTAINER not found", envContainer)
		}
		return envContainer, nil
	}

	// 3. Find container for current workspace
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	sessions, err := session.ListWorkspaceSessions(absWorkspace)
	if err != nil {
		return "", fmt.Errorf("failed to list workspace sessions: %w", err)
	}

	if len(sessions) == 0 {
		return "", fmt.Errorf("no Clincus containers found for current workspace - use --container to specify")
	}

	if len(sessions) > 1 {
		// Multiple containers - list them and ask user to specify
		var names []string
		for _, name := range sessions {
			names = append(names, name)
		}
		return "", fmt.Errorf("multiple Clincus containers found for workspace, use --container to specify: %s", strings.Join(names, ", "))
	}

	// Exactly one container
	for _, name := range sessions {
		return name, nil
	}

	return "", fmt.Errorf("no Clincus containers found for current workspace")
}

// generateSnapshotName generates an auto-named snapshot
func generateSnapshotName() string {
	return fmt.Sprintf("snap-%s", time.Now().Format("20060102-150405"))
}

// confirmAction prompts the user for confirmation
func confirmAction(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func snapshotCreateCommand(cmd *cobra.Command, args []string) error {
	containerName, err := resolveContainer()
	if err != nil {
		return exitError(1, err.Error())
	}

	// Determine snapshot name
	var snapshotName string
	if len(args) > 0 {
		snapshotName = args[0]
	} else {
		snapshotName = generateSnapshotName()
	}

	mgr := container.NewManager(containerName)

	// Check if snapshot already exists
	exists, err := mgr.SnapshotExists(snapshotName)
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to check snapshot: %v", err))
	}
	if exists {
		return exitError(1, fmt.Sprintf("snapshot '%s' already exists for container '%s'", snapshotName, containerName))
	}

	// Create snapshot
	if err := mgr.CreateSnapshot(snapshotName, snapshotStateful); err != nil {
		return exitError(1, fmt.Sprintf("failed to create snapshot: %v", err))
	}

	if snapshotStateful {
		fmt.Fprintf(os.Stderr, "Created stateful snapshot '%s' for container '%s'\n", snapshotName, containerName)
	} else {
		fmt.Fprintf(os.Stderr, "Created snapshot '%s' for container '%s'\n", snapshotName, containerName)
	}

	return nil
}

func snapshotListCommand(cmd *cobra.Command, args []string) error {
	// Validate format
	if snapshotFormat != "text" && snapshotFormat != "json" {
		return exitError(2, fmt.Sprintf("invalid format '%s': must be 'text' or 'json'", snapshotFormat))
	}

	if snapshotAll {
		return listAllSnapshots()
	}

	containerName, err := resolveContainer()
	if err != nil {
		return exitError(1, err.Error())
	}

	mgr := container.NewManager(containerName)
	snapshots, err := mgr.ListSnapshots()
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to list snapshots: %v", err))
	}

	if snapshotFormat == "json" {
		return outputSnapshotJSON(containerName, snapshots)
	}

	return outputSnapshotText(containerName, snapshots)
}

func listAllSnapshots() error {
	// Get all Clincus containers
	prefix := session.GetContainerPrefix()
	pattern := fmt.Sprintf("^%s", prefix)

	containers, err := container.ListContainers(pattern)
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to list containers: %v", err))
	}

	if len(containers) == 0 {
		fmt.Fprintf(os.Stderr, "No Clincus containers found\n")
		return nil
	}

	if snapshotFormat == "json" {
		// Build JSON output for all containers
		allData := make(map[string]interface{})
		for _, containerName := range containers {
			mgr := container.NewManager(containerName)
			snapshots, err := mgr.ListSnapshots()
			if err != nil {
				continue // Skip containers that fail
			}
			allData[containerName] = snapshots
		}

		jsonData, err := json.MarshalIndent(allData, "", "  ")
		if err != nil {
			return exitError(1, fmt.Sprintf("failed to marshal JSON: %v", err))
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Text output for all containers
	for _, containerName := range containers {
		mgr := container.NewManager(containerName)
		snapshots, err := mgr.ListSnapshots()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nContainer %s: (error listing snapshots)\n", containerName)
			continue
		}

		fmt.Printf("\nSnapshots for %s:\n", containerName)
		if len(snapshots) == 0 {
			fmt.Println("  (none)")
		} else {
			fmt.Printf("%-20s %-24s %-8s\n", "NAME", "CREATED", "STATEFUL")
			for _, s := range snapshots {
				stateful := "no"
				if s.Stateful {
					stateful = "yes"
				}
				fmt.Printf("%-20s %-24s %-8s\n",
					s.Name,
					s.CreatedAt.Format("2006-01-02 15:04:05"),
					stateful,
				)
			}
		}
	}

	return nil
}

func outputSnapshotJSON(containerName string, snapshots []container.SnapshotInfo) error {
	output := map[string]interface{}{
		"container": containerName,
		"snapshots": snapshots,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to marshal JSON: %v", err))
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputSnapshotText(containerName string, snapshots []container.SnapshotInfo) error {
	fmt.Printf("Snapshots for %s:\n\n", containerName)

	if len(snapshots) == 0 {
		fmt.Println("(none)")
		fmt.Printf("\nTotal: 0 snapshots\n")
		return nil
	}

	fmt.Printf("%-20s %-24s %-8s\n", "NAME", "CREATED", "STATEFUL")
	for _, s := range snapshots {
		stateful := "no"
		if s.Stateful {
			stateful = "yes"
		}
		fmt.Printf("%-20s %-24s %-8s\n",
			s.Name,
			s.CreatedAt.Format("2006-01-02 15:04:05"),
			stateful,
		)
	}

	fmt.Printf("\nTotal: %d snapshot", len(snapshots))
	if len(snapshots) != 1 {
		fmt.Print("s")
	}
	fmt.Println()

	return nil
}

func snapshotRestoreCommand(cmd *cobra.Command, args []string) error {
	containerName, err := resolveContainer()
	if err != nil {
		return exitError(1, err.Error())
	}

	snapshotName := args[0]
	mgr := container.NewManager(containerName)

	// Check if snapshot exists
	exists, err := mgr.SnapshotExists(snapshotName)
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to check snapshot: %v", err))
	}
	if !exists {
		return exitError(1, fmt.Sprintf("snapshot '%s' not found for container '%s'", snapshotName, containerName))
	}

	// Check if container is running
	running, err := mgr.Running()
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to check container status: %v", err))
	}
	if running {
		return exitError(1, fmt.Sprintf("container '%s' must be stopped before restore (use 'clincus container stop %s')", containerName, containerName))
	}

	// Confirm unless --force
	if !snapshotForce {
		fmt.Fprintf(os.Stderr, "WARNING: This will restore container '%s' to snapshot '%s'.\n", containerName, snapshotName)
		fmt.Fprintf(os.Stderr, "All changes since the snapshot will be lost.\n\n")
		if !confirmAction("Continue?") {
			fmt.Fprintf(os.Stderr, "Aborted\n")
			return nil
		}
	}

	// Restore snapshot
	if err := mgr.RestoreSnapshot(snapshotName, snapshotStateful); err != nil {
		return exitError(1, fmt.Sprintf("failed to restore snapshot: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Restored container '%s' from snapshot '%s'\n", containerName, snapshotName)
	return nil
}

func snapshotDeleteCommand(cmd *cobra.Command, args []string) error {
	containerName, err := resolveContainer()
	if err != nil {
		return exitError(1, err.Error())
	}

	mgr := container.NewManager(containerName)

	if snapshotAll {
		// Delete all snapshots
		snapshots, err := mgr.ListSnapshots()
		if err != nil {
			return exitError(1, fmt.Sprintf("failed to list snapshots: %v", err))
		}

		if len(snapshots) == 0 {
			fmt.Fprintf(os.Stderr, "No snapshots to delete for container '%s'\n", containerName)
			return nil
		}

		// Confirm unless --force
		if !snapshotForce {
			fmt.Fprintf(os.Stderr, "WARNING: This will delete ALL %d snapshot(s) for container '%s':\n", len(snapshots), containerName)
			for _, s := range snapshots {
				fmt.Fprintf(os.Stderr, "  - %s\n", s.Name)
			}
			fmt.Fprintln(os.Stderr)
			if !confirmAction("Continue?") {
				fmt.Fprintf(os.Stderr, "Aborted\n")
				return nil
			}
		}

		// Delete all snapshots
		for _, s := range snapshots {
			if err := mgr.DeleteSnapshot(s.Name); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete snapshot '%s': %v\n", s.Name, err)
			} else {
				fmt.Fprintf(os.Stderr, "Deleted snapshot '%s'\n", s.Name)
			}
		}

		return nil
	}

	// Delete single snapshot
	if len(args) == 0 {
		return exitError(2, "snapshot name required (or use --all to delete all snapshots)")
	}

	snapshotName := args[0]

	// Check if snapshot exists
	exists, err := mgr.SnapshotExists(snapshotName)
	if err != nil {
		return exitError(1, fmt.Sprintf("failed to check snapshot: %v", err))
	}
	if !exists {
		return exitError(1, fmt.Sprintf("snapshot '%s' not found for container '%s'", snapshotName, containerName))
	}

	// Delete snapshot
	if err := mgr.DeleteSnapshot(snapshotName); err != nil {
		return exitError(1, fmt.Sprintf("failed to delete snapshot: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Deleted snapshot '%s' from container '%s'\n", snapshotName, containerName)
	return nil
}

func snapshotInfoCommand(cmd *cobra.Command, args []string) error {
	// Validate format
	if snapshotFormat != "text" && snapshotFormat != "json" {
		return exitError(2, fmt.Sprintf("invalid format '%s': must be 'text' or 'json'", snapshotFormat))
	}

	containerName, err := resolveContainer()
	if err != nil {
		return exitError(1, err.Error())
	}

	snapshotName := args[0]
	mgr := container.NewManager(containerName)

	// Get snapshot info
	info, err := mgr.GetSnapshotInfo(snapshotName)
	if err != nil {
		return exitError(1, err.Error())
	}

	if snapshotFormat == "json" {
		output := map[string]interface{}{
			"container": containerName,
			"snapshot":  info,
		}

		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return exitError(1, fmt.Sprintf("failed to marshal JSON: %v", err))
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Text output
	fmt.Printf("Snapshot: %s\n", info.Name)
	fmt.Printf("Container: %s\n", containerName)
	fmt.Printf("Created: %s\n", info.CreatedAt.Format("2006-01-02 15:04:05"))
	if info.ExpiresAt != nil {
		fmt.Printf("Expires: %s\n", info.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	if info.Stateful {
		fmt.Println("Stateful: yes (includes process memory)")
	} else {
		fmt.Println("Stateful: no")
	}
	if info.Description != "" {
		fmt.Printf("Description: %s\n", info.Description)
	}

	return nil
}
