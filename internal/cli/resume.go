package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume [container-name]",
	Short: "Resume a frozen (paused) container",
	Long: `Resume a container that was paused/frozen.

Use this command to resume a frozen container.

Examples:
  clincus resume clincus-abc123-1  # Resume a specific frozen container
  clincus resume                    # Resume all frozen clincus containers`,
	Args: cobra.MaximumNArgs(1),
	RunE: runResume,
}

func runResume(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		// Resume specific container
		containerName := args[0]
		return resumeContainer(containerName)
	}

	// Resume all frozen Clincus containers
	return resumeAllFrozen()
}

func resumeContainer(name string) error {
	// Check if container exists and is frozen
	status, err := getContainerStatus(name)
	if err != nil {
		return fmt.Errorf("container %s not found: %w", name, err)
	}

	if status != "Frozen" {
		return fmt.Errorf("container %s is not frozen (status: %s)", name, status)
	}

	// Resume the container
	_, err = container.IncusOutput("start", name)
	if err != nil {
		return fmt.Errorf("failed to resume container: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Resumed container: %s\n", name)
	return nil
}

func resumeAllFrozen() error {
	// List all Clincus containers
	output, err := container.IncusOutput("list", "--format", "csv", "-c", "ns")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	resumedCount := 0

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		status := parts[1]

		// Only resume frozen clincus containers
		if !strings.HasPrefix(name, "clincus-") {
			continue
		}

		if status == "FROZEN" {
			if err := resumeContainer(name); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to resume %s: %v\n", name, err)
			} else {
				resumedCount++
			}
		}
	}

	if resumedCount == 0 {
		fmt.Fprintln(os.Stderr, "No frozen clincus containers found")
	} else {
		fmt.Fprintf(os.Stderr, "Resumed %d container(s)\n", resumedCount)
	}

	return nil
}

func getContainerStatus(name string) (string, error) {
	output, err := container.IncusOutput("list", name, "--format", "csv", "-c", "s")
	if err != nil {
		return "", err
	}

	status := strings.TrimSpace(output)
	if status == "" {
		return "", fmt.Errorf("container not found")
	}
	// Normalize status
	if status == "FROZEN" {
		return "Frozen", nil
	}
	return status, nil
}
