package health

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mensfeld/code-on-incus/internal/config"
	"github.com/mensfeld/code-on-incus/internal/container"
	"github.com/mensfeld/code-on-incus/internal/session"
	"github.com/mensfeld/code-on-incus/internal/tool"
)

// CheckOS reports the operating system information
func CheckOS() HealthCheck {
	// Get OS and architecture
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Try to get more detailed OS info on Linux
	var details string
	var environment string

	if osName == "linux" {
		// Try to read /etc/os-release for distribution info
		if content, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(content), "\n")
			var prettyName string
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					prettyName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
					break
				}
			}
			if prettyName != "" {
				details = prettyName
			}
		}

		// Detect if running in Colima/Lima VM
		if isColimaEnvironment() {
			environment = "colima"
		}
	} else if osName == "darwin" {
		// Get macOS version
		cmd := exec.Command("sw_vers", "-productVersion")
		if output, err := cmd.Output(); err == nil {
			details = "macOS " + strings.TrimSpace(string(output))
		}
	}

	message := fmt.Sprintf("%s/%s", osName, arch)
	if details != "" {
		message = fmt.Sprintf("%s (%s)", details, arch)
	}
	if environment != "" {
		message += fmt.Sprintf(" [%s]", environment)
	}

	return HealthCheck{
		Name:    "os",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"os":          osName,
			"arch":        arch,
			"details":     details,
			"environment": environment,
		},
	}
}

// isColimaEnvironment detects if running inside a Colima/Lima VM
func isColimaEnvironment() bool {
	// Check for virtiofs mounts (characteristic of Lima VMs)
	if content, err := os.ReadFile("/proc/mounts"); err == nil {
		if strings.Contains(string(content), "virtiofs") {
			return true
		}
	}

	// Check for lima user
	if currentUser, err := user.Current(); err == nil {
		if currentUser.Username == "lima" {
			return true
		}
	}

	return false
}

// CheckIncus verifies that Incus is available and running
func CheckIncus() HealthCheck {
	// Check if incus binary exists
	if _, err := exec.LookPath("incus"); err != nil {
		return HealthCheck{
			Name:    "incus",
			Status:  StatusFailed,
			Message: "Incus binary not found",
		}
	}

	// Check if Incus is available (daemon running and accessible)
	if !container.Available() {
		return HealthCheck{
			Name:    "incus",
			Status:  StatusFailed,
			Message: "Incus daemon not running or not accessible",
		}
	}

	// Get Incus version
	versionOutput, err := container.IncusOutput("version")
	if err != nil {
		return HealthCheck{
			Name:    "incus",
			Status:  StatusOK,
			Message: "Running (version unknown)",
		}
	}

	// Parse version - extract server version from multi-line output
	// Example output: "Client version: 6.20\nServer version: 6.20"
	version := strings.TrimSpace(versionOutput)
	for _, line := range strings.Split(version, "\n") {
		if strings.HasPrefix(line, "Server version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Server version:"))
			break
		}
	}

	return HealthCheck{
		Name:    "incus",
		Status:  StatusOK,
		Message: fmt.Sprintf("Running (version %s)", version),
		Details: map[string]interface{}{
			"version": version,
		},
	}
}

// CheckPermissions verifies user has correct group membership
func CheckPermissions() HealthCheck {
	// On macOS, no group check needed
	if runtime.GOOS == "darwin" {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusOK,
			Message: "macOS - no group required",
		}
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not determine current user: %v", err),
		}
	}

	// Get user's groups
	groups, err := currentUser.GroupIds()
	if err != nil {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not determine user groups: %v", err),
		}
	}

	// Look for incus-admin group
	incusGroup, err := user.LookupGroup("incus-admin")
	if err != nil {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusFailed,
			Message: "incus-admin group not found",
		}
	}

	// Check if user is in the group
	for _, gid := range groups {
		if gid == incusGroup.Gid {
			return HealthCheck{
				Name:    "permissions",
				Status:  StatusOK,
				Message: "User in incus-admin group",
				Details: map[string]interface{}{
					"user":  currentUser.Username,
					"group": "incus-admin",
				},
			}
		}
	}

	return HealthCheck{
		Name:    "permissions",
		Status:  StatusFailed,
		Message: fmt.Sprintf("User '%s' not in incus-admin group", currentUser.Username),
	}
}

// CheckImage verifies that the default image exists
func CheckImage(imageName string) HealthCheck {
	if imageName == "" {
		imageName = "clincus"
	}

	exists, err := container.ImageExists(imageName)
	if err != nil {
		return HealthCheck{
			Name:    "image",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not check image: %v", err),
		}
	}

	if !exists {
		return HealthCheck{
			Name:    "image",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Image '%s' not found (run 'clincus build')", imageName),
			Details: map[string]interface{}{
				"expected": imageName,
			},
		}
	}

	// Get image fingerprint
	output, err := container.IncusOutput("image", "list", imageName, "--format=csv", "-c", "f")
	fingerprint := ""
	if err == nil && output != "" {
		lines := strings.Split(output, "\n")
		if len(lines) > 0 {
			fingerprint = strings.TrimSpace(lines[0])
			if len(fingerprint) > 12 {
				fingerprint = fingerprint[:12] + "..."
			}
		}
	}

	message := imageName
	if fingerprint != "" {
		message = fmt.Sprintf("%s (fingerprint: %s)", imageName, fingerprint)
	}

	return HealthCheck{
		Name:    "image",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"alias":       imageName,
			"fingerprint": fingerprint,
		},
	}
}

// CheckNetworkBridge verifies the network bridge is configured
func CheckNetworkBridge() HealthCheck {
	// Get default profile to find network device
	output, err := container.IncusOutput("profile", "device", "show", "default")
	if err != nil {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not get default profile: %v", err),
		}
	}

	// Parse network name from profile (looking for eth0 device)
	var networkName string
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "eth0:" {
			// Look for network: line
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.Contains(lines[j], "network:") {
					parts := strings.Split(lines[j], ":")
					if len(parts) >= 2 {
						networkName = strings.TrimSpace(parts[1])
						break
					}
				}
			}
			break
		}
	}

	if networkName == "" {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusFailed,
			Message: "No eth0 network device in default profile",
		}
	}

	// Get network configuration
	networkOutput, err := container.IncusOutput("network", "show", networkName)
	if err != nil {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not get network info for %s: %v", networkName, err),
		}
	}

	// Parse IPv4 address
	var ipv4Address string
	for _, line := range strings.Split(networkOutput, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ipv4.address:") {
			ipv4Address = strings.TrimSpace(strings.TrimPrefix(line, "ipv4.address:"))
			break
		}
	}

	if ipv4Address == "" || ipv4Address == "none" {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s has no IPv4 address", networkName),
		}
	}

	return HealthCheck{
		Name:    "network_bridge",
		Status:  StatusOK,
		Message: fmt.Sprintf("%s (%s)", networkName, ipv4Address),
		Details: map[string]interface{}{
			"name": networkName,
			"ipv4": ipv4Address,
		},
	}
}

// CheckIPForwarding verifies IP forwarding is enabled
func CheckIPForwarding() HealthCheck {
	// On macOS, IP forwarding works differently
	if runtime.GOOS == "darwin" {
		return HealthCheck{
			Name:    "ip_forwarding",
			Status:  StatusOK,
			Message: "macOS - managed by Incus",
		}
	}

	// Read /proc/sys/net/ipv4/ip_forward
	content, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return HealthCheck{
			Name:    "ip_forwarding",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not check: %v", err),
		}
	}

	value := strings.TrimSpace(string(content))
	if value == "1" {
		return HealthCheck{
			Name:    "ip_forwarding",
			Status:  StatusOK,
			Message: "Enabled",
		}
	}

	return HealthCheck{
		Name:    "ip_forwarding",
		Status:  StatusWarning,
		Message: "Disabled (may affect container networking)",
	}
}

// CheckClincusDirectory verifies the clincus directory exists and is writable
func CheckClincusDirectory() HealthCheck {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return HealthCheck{
			Name:    "clincus_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not determine home directory: %v", err),
		}
	}

	clincusDir := filepath.Join(homeDir, ".clincus")

	// Check if directory exists
	info, err := os.Stat(clincusDir)
	if os.IsNotExist(err) {
		return HealthCheck{
			Name:    "clincus_directory",
			Status:  StatusWarning,
			Message: fmt.Sprintf("%s does not exist (will be created on first run)", clincusDir),
		}
	}
	if err != nil {
		return HealthCheck{
			Name:    "clincus_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not access %s: %v", clincusDir, err),
		}
	}

	if !info.IsDir() {
		return HealthCheck{
			Name:    "clincus_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not a directory", clincusDir),
		}
	}

	// Check if writable by creating a temp file
	testFile := filepath.Join(clincusDir, ".health-check-test")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return HealthCheck{
			Name:    "clincus_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not writable", clincusDir),
		}
	}
	os.Remove(testFile)

	return HealthCheck{
		Name:    "clincus_directory",
		Status:  StatusOK,
		Message: "~/.clincus (writable)",
		Details: map[string]interface{}{
			"path": clincusDir,
		},
	}
}

// CheckSessionsDirectory verifies the sessions directory exists and is writable
func CheckSessionsDirectory(cfg *config.Config) HealthCheck {
	// Get configured tool to determine sessions directory
	toolName := cfg.Tool.Name
	if toolName == "" {
		toolName = "claude"
	}
	toolInstance, err := tool.Get(toolName)
	if err != nil {
		toolInstance = tool.GetDefault()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not determine home directory: %v", err),
		}
	}

	baseDir := filepath.Join(homeDir, ".clincus")
	sessionsDir := session.GetSessionsDir(baseDir, toolInstance)

	// Check if directory exists
	info, err := os.Stat(sessionsDir)
	if os.IsNotExist(err) {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusOK,
			Message: fmt.Sprintf("%s (will be created)", filepath.Base(sessionsDir)),
			Details: map[string]interface{}{
				"path": sessionsDir,
			},
		}
	}
	if err != nil {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not access %s: %v", sessionsDir, err),
		}
	}

	if !info.IsDir() {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not a directory", sessionsDir),
		}
	}

	// Check if writable
	testFile := filepath.Join(sessionsDir, ".health-check-test")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not writable", sessionsDir),
		}
	}
	os.Remove(testFile)

	return HealthCheck{
		Name:    "sessions_directory",
		Status:  StatusOK,
		Message: fmt.Sprintf("~/.clincus/%s (writable)", filepath.Base(sessionsDir)),
		Details: map[string]interface{}{
			"path": sessionsDir,
		},
	}
}

// CheckConfiguration verifies the configuration is loaded correctly
func CheckConfiguration(cfg *config.Config) HealthCheck {
	if cfg == nil {
		return HealthCheck{
			Name:    "config",
			Status:  StatusFailed,
			Message: "Configuration not loaded",
		}
	}

	// Find which config files exist
	configPaths := config.GetConfigPaths()
	var loadedFrom []string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			loadedFrom = append(loadedFrom, path)
		}
	}

	message := "Defaults only (no config files)"
	if len(loadedFrom) > 0 {
		message = loadedFrom[len(loadedFrom)-1] // Show highest priority
	}

	return HealthCheck{
		Name:    "config",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"loaded_from": loadedFrom,
		},
	}
}

// CheckTool reports the configured tool
func CheckTool(toolName string) HealthCheck {
	if toolName == "" {
		toolName = "claude"
	}

	_, err := tool.Get(toolName)
	if err != nil {
		return HealthCheck{
			Name:    "tool",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Unknown tool: %s", toolName),
		}
	}

	return HealthCheck{
		Name:    "tool",
		Status:  StatusOK,
		Message: toolName,
		Details: map[string]interface{}{
			"name": toolName,
		},
	}
}

// CheckDNS verifies DNS resolution is working
func CheckDNS() HealthCheck {
	// Try to resolve a well-known domain
	testDomain := "api.anthropic.com"

	ips, err := net.LookupIP(testDomain)
	if err != nil {
		return HealthCheck{
			Name:    "dns_resolution",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Failed to resolve %s: %v", testDomain, err),
		}
	}

	if len(ips) == 0 {
		return HealthCheck{
			Name:    "dns_resolution",
			Status:  StatusWarning,
			Message: fmt.Sprintf("No IPs found for %s", testDomain),
		}
	}

	return HealthCheck{
		Name:    "dns_resolution",
		Status:  StatusOK,
		Message: fmt.Sprintf("Working (%s -> %d IPs)", testDomain, len(ips)),
		Details: map[string]interface{}{
			"test_domain": testDomain,
			"ip_count":    len(ips),
		},
	}
}

// CheckDiskSpace checks available disk space in ~/.clincus directory
func CheckDiskSpace() HealthCheck {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return HealthCheck{
			Name:    "disk_space",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not determine home directory: %v", err),
		}
	}

	clincusDir := filepath.Join(homeDir, ".clincus")

	// Use the parent directory if .clincus doesn't exist yet
	checkDir := clincusDir
	if _, err := os.Stat(clincusDir); os.IsNotExist(err) {
		checkDir = homeDir
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(checkDir, &stat); err != nil {
		return HealthCheck{
			Name:    "disk_space",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not check disk space: %v", err),
		}
	}

	// Calculate available space in bytes
	// #nosec G115 - Bsize is always positive on real filesystems
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	availableGB := float64(availableBytes) / (1024 * 1024 * 1024)

	// Warn if less than 5GB available
	if availableGB < 5 {
		return HealthCheck{
			Name:    "disk_space",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Low disk space: %.1f GB available", availableGB),
			Details: map[string]interface{}{
				"available_gb": availableGB,
				"path":         checkDir,
			},
		}
	}

	return HealthCheck{
		Name:    "disk_space",
		Status:  StatusOK,
		Message: fmt.Sprintf("%.1f GB available", availableGB),
		Details: map[string]interface{}{
			"available_gb": availableGB,
			"path":         checkDir,
		},
	}
}

// CheckIncusStoragePool checks the Incus storage pool usage.
// It queries `incus storage info <pool>` for the default pool and warns
// when free space is critically low (< 3 GiB free or > 90% used).
func CheckIncusStoragePool() HealthCheck {
	// Find the default storage pool from the default profile
	poolName := "default"
	profileOut, err := exec.Command("incus", "profile", "show", "default").Output()
	if err == nil {
		for _, line := range strings.Split(string(profileOut), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "pool:") {
				poolName = strings.TrimSpace(strings.TrimPrefix(line, "pool:"))
				break
			}
		}
	}

	out, err := exec.Command("incus", "storage", "info", poolName).Output()
	if err != nil {
		return HealthCheck{
			Name:    "incus_storage_pool",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not query storage pool '%s': %v", poolName, err),
		}
	}

	// Parse "space used: X.XXGiB" and "total space: Y.YYGiB"
	var usedGiB, totalGiB float64
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "space used:") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "space used:"), "%f", &usedGiB)
		} else if strings.HasPrefix(line, "total space:") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "total space:"), "%f", &totalGiB)
		}
	}

	if totalGiB == 0 {
		return HealthCheck{
			Name:    "incus_storage_pool",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not parse storage pool '%s' usage", poolName),
		}
	}

	freeGiB := totalGiB - usedGiB
	usedPct := (usedGiB / totalGiB) * 100

	details := map[string]interface{}{
		"pool":      poolName,
		"used_gib":  usedGiB,
		"total_gib": totalGiB,
		"free_gib":  freeGiB,
		"used_pct":  usedPct,
	}

	switch {
	case freeGiB < 2 || usedPct > 90:
		return HealthCheck{
			Name:    "incus_storage_pool",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Pool '%s' critically low: %.1f GiB free of %.1f GiB (%.0f%% used)", poolName, freeGiB, totalGiB, usedPct),
			Details: details,
		}
	case freeGiB < 5 || usedPct > 80:
		return HealthCheck{
			Name:    "incus_storage_pool",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Pool '%s' low: %.1f GiB free of %.1f GiB (%.0f%% used)", poolName, freeGiB, totalGiB, usedPct),
			Details: details,
		}
	default:
		return HealthCheck{
			Name:    "incus_storage_pool",
			Status:  StatusOK,
			Message: fmt.Sprintf("Pool '%s': %.1f GiB free of %.1f GiB (%.0f%% used)", poolName, freeGiB, totalGiB, usedPct),
			Details: details,
		}
	}
}

// CheckActiveContainers counts running COI containers
func CheckActiveContainers() HealthCheck {
	prefix := session.GetContainerPrefix()
	pattern := fmt.Sprintf("^%s", prefix)

	output, err := container.IncusOutput("list", pattern, "--format=json")
	if err != nil {
		return HealthCheck{
			Name:    "active_containers",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not list containers: %v", err),
		}
	}

	var containers []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &containers); err != nil {
		return HealthCheck{
			Name:    "active_containers",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not parse container list: %v", err),
		}
	}

	// Count running containers
	running := 0
	for _, c := range containers {
		if status, ok := c["status"].(string); ok && status == "Running" {
			running++
		}
	}

	total := len(containers)
	message := fmt.Sprintf("%d running", running)
	if total > running {
		message = fmt.Sprintf("%d running, %d stopped", running, total-running)
	}
	if total == 0 {
		message = "None"
	}

	return HealthCheck{
		Name:    "active_containers",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"running": running,
			"total":   total,
		},
	}
}

// CheckSavedSessions counts saved sessions
func CheckSavedSessions(cfg *config.Config) HealthCheck {
	// Get configured tool
	toolName := cfg.Tool.Name
	if toolName == "" {
		toolName = "claude"
	}
	toolInstance, err := tool.Get(toolName)
	if err != nil {
		toolInstance = tool.GetDefault()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return HealthCheck{
			Name:    "saved_sessions",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not determine home directory: %v", err),
		}
	}

	baseDir := filepath.Join(homeDir, ".clincus")
	sessionsDir := session.GetSessionsDir(baseDir, toolInstance)

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return HealthCheck{
				Name:    "saved_sessions",
				Status:  StatusOK,
				Message: "None",
				Details: map[string]interface{}{
					"count": 0,
				},
			}
		}
		return HealthCheck{
			Name:    "saved_sessions",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not read sessions directory: %v", err),
		}
	}

	// Count directories (sessions)
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}

	message := fmt.Sprintf("%d session(s)", count)
	if count == 0 {
		message = "None"
	}

	return HealthCheck{
		Name:    "saved_sessions",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"count": count,
			"path":  sessionsDir,
		},
	}
}

// CheckImageAge checks if the clincus image is outdated
func CheckImageAge(imageName string) HealthCheck {
	if imageName == "" {
		imageName = "clincus"
	}

	// Get image info
	output, err := container.IncusOutput("image", "list", imageName, "--format=json")
	if err != nil {
		return HealthCheck{
			Name:    "image_age",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not get image info: %v", err),
		}
	}

	var images []struct {
		CreatedAt time.Time `json:"created_at"`
		Aliases   []struct {
			Name string `json:"name"`
		} `json:"aliases"`
	}

	if err := json.Unmarshal([]byte(output), &images); err != nil {
		return HealthCheck{
			Name:    "image_age",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not parse image info: %v", err),
		}
	}

	// Find the image
	for _, img := range images {
		for _, alias := range img.Aliases {
			if alias.Name == imageName {
				age := time.Since(img.CreatedAt)
				days := int(age.Hours() / 24)

				// Warn if older than 30 days
				if days > 30 {
					return HealthCheck{
						Name:    "image_age",
						Status:  StatusWarning,
						Message: fmt.Sprintf("%d days old (consider rebuilding with 'clincus build --force')", days),
						Details: map[string]interface{}{
							"created_at": img.CreatedAt.Format("2006-01-02"),
							"age_days":   days,
						},
					}
				}

				return HealthCheck{
					Name:    "image_age",
					Status:  StatusOK,
					Message: fmt.Sprintf("%d days old", days),
					Details: map[string]interface{}{
						"created_at": img.CreatedAt.Format("2006-01-02"),
						"age_days":   days,
					},
				}
			}
		}
	}

	return HealthCheck{
		Name:    "image_age",
		Status:  StatusWarning,
		Message: fmt.Sprintf("Image '%s' not found", imageName),
	}
}

// CheckOrphanedResources checks for orphaned system resources
func CheckOrphanedResources() HealthCheck {
	// Check for orphaned veths
	orphanedVeths := 0
	entries, err := os.ReadDir("/sys/class/net")
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, "veth") {
				continue
			}
			masterPath := fmt.Sprintf("/sys/class/net/%s/master", name)
			if _, err := os.Stat(masterPath); os.IsNotExist(err) {
				orphanedVeths++
			}
		}
	}

	if orphanedVeths == 0 {
		return HealthCheck{
			Name:    "orphaned_resources",
			Status:  StatusOK,
			Message: "No orphaned resources",
		}
	}

	message := fmt.Sprintf("%d orphaned resource(s) found", orphanedVeths)
	message += fmt.Sprintf(" (%d veths)", orphanedVeths)
	message += " - run 'clincus clean' to remove"

	return HealthCheck{
		Name:    "orphaned_resources",
		Status:  StatusWarning,
		Message: message,
		Details: map[string]interface{}{
			"orphaned_veths": orphanedVeths,
		},
	}
}

// CheckCgroupAvailability checks if cgroup v2 is available for resource monitoring
func CheckCgroupAvailability() HealthCheck {
	cgroupPath := "/sys/fs/cgroup"

	// Check if cgroup filesystem exists
	info, err := os.Stat(cgroupPath)
	if err != nil {
		return HealthCheck{
			Name:    "cgroup_availability",
			Status:  StatusFailed,
			Message: "Cgroup filesystem not found",
			Details: map[string]interface{}{
				"path":  cgroupPath,
				"error": err.Error(),
				"hint":  "Resource monitoring requires cgroup v2",
			},
		}
	}

	if !info.IsDir() {
		return HealthCheck{
			Name:    "cgroup_availability",
			Status:  StatusFailed,
			Message: "Cgroup path is not a directory",
			Details: map[string]interface{}{
				"path": cgroupPath,
			},
		}
	}

	// Check if it's cgroup v2 by looking for cgroup.controllers
	controllersPath := filepath.Join(cgroupPath, "cgroup.controllers")
	if _, err := os.Stat(controllersPath); err != nil {
		return HealthCheck{
			Name:    "cgroup_availability",
			Status:  StatusWarning,
			Message: "Cgroup v1 detected (v2 recommended for monitoring)",
			Details: map[string]interface{}{
				"path": cgroupPath,
				"hint": "Resource monitoring works best with cgroup v2",
			},
		}
	}

	// Read available controllers
	controllers, err := os.ReadFile(controllersPath)
	if err != nil {
		return HealthCheck{
			Name:    "cgroup_availability",
			Status:  StatusOK,
			Message: "Cgroup v2 is available",
			Details: map[string]interface{}{
				"path": cgroupPath,
			},
		}
	}

	return HealthCheck{
		Name:    "cgroup_availability",
		Status:  StatusOK,
		Message: "Cgroup v2 is available with controllers",
		Details: map[string]interface{}{
			"path":        cgroupPath,
			"controllers": strings.TrimSpace(string(controllers)),
		},
	}
}

