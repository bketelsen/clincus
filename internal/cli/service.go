package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

const serviceUnitName = "clincus.service"

func serviceUnitPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	return filepath.Join(homeDir, ".config", "systemd", "user", serviceUnitName)
}

var serviceUnitTemplate = `[Unit]
Description=Clincus Web Dashboard
After=network.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

type unitTemplateData struct {
	BinaryPath string
}

func resolveBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	return resolved, nil
}

func renderUnitFile(binaryPath string) (string, error) {
	tmpl, err := template.New("unit").Parse(serviceUnitTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse unit template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, unitTemplateData{BinaryPath: binaryPath}); err != nil {
		return "", fmt.Errorf("failed to render unit template: %w", err)
	}

	return buf.String(), nil
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the clincus systemd user service",
	Long: `Manage the clincus web dashboard as a systemd user service.

This installs a systemd --user unit that runs 'clincus serve' automatically
on login, and provides commands to start, stop, and remove it.

Examples:
  clincus service install    # Install the systemd user unit
  clincus service start      # Start the service
  clincus service stop       # Stop the service
  clincus service remove     # Remove the systemd user unit
`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the systemd user unit",
	Long: `Install the clincus systemd user service unit.

This writes a unit file to ~/.config/systemd/user/clincus.service,
reloads the systemd user daemon, and enables the service to start on login.

The unit runs 'clincus serve' and restarts on failure.
`,
	RunE: serviceInstallCommand,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the clincus service",
	RunE:  serviceStartCommand,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the clincus service",
	RunE:  serviceStopCommand,
}

func serviceStartCommand(cmd *cobra.Command, args []string) error {
	if err := exec.Command("systemctl", "--user", "start", serviceUnitName).Run(); err != nil {
		return exitError(1, fmt.Sprintf("failed to start service: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Service started.\n")
	return nil
}

func serviceStopCommand(cmd *cobra.Command, args []string) error {
	if err := exec.Command("systemctl", "--user", "stop", serviceUnitName).Run(); err != nil {
		return exitError(1, fmt.Sprintf("failed to stop service: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Service stopped.\n")
	return nil
}

func serviceInstallCommand(cmd *cobra.Command, args []string) error {
	binaryPath, err := resolveBinaryPath()
	if err != nil {
		return exitError(1, err.Error())
	}

	unitContent, err := renderUnitFile(binaryPath)
	if err != nil {
		return exitError(1, err.Error())
	}

	unitPath := serviceUnitPath()

	// Ensure parent directory exists
	unitDir := filepath.Dir(unitPath)
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return exitError(1, fmt.Sprintf("failed to create directory %s: %v", unitDir, err))
	}

	// Write unit file
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return exitError(1, fmt.Sprintf("failed to write unit file: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Installed unit file to %s\n", unitPath)

	// Reload systemd user daemon
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return exitError(1, fmt.Sprintf("failed to reload systemd daemon: %v", err))
	}

	// Enable the service
	if err := exec.Command("systemctl", "--user", "enable", serviceUnitName).Run(); err != nil {
		return exitError(1, fmt.Sprintf("failed to enable service: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Service enabled. Run 'clincus service start' to start it.\n")
	return nil
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
}
