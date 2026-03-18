package health

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// incusProfile represents the YAML output of "incus profile show <name>".
// Only fields needed for health checks are included; unknown fields are ignored.
type incusProfile struct {
	Config  map[string]string            `yaml:"config"`
	Devices map[string]map[string]string `yaml:"devices"`
	Name    string                       `yaml:"name"`
}

// incusNetwork represents the YAML output of "incus network show <name>".
type incusNetwork struct {
	Config  map[string]string `yaml:"config"`
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Status  string            `yaml:"status"`
	Managed bool              `yaml:"managed"`
}

// parseProfileYAML parses YAML output from "incus profile show <name>".
func parseProfileYAML(output string) (*incusProfile, error) {
	var profile incusProfile
	if err := yaml.Unmarshal([]byte(output), &profile); err != nil {
		return nil, fmt.Errorf("parsing profile YAML: %w", err)
	}
	return &profile, nil
}

// parseNetworkYAML parses YAML output from "incus network show <name>".
func parseNetworkYAML(output string) (*incusNetwork, error) {
	var network incusNetwork
	if err := yaml.Unmarshal([]byte(output), &network); err != nil {
		return nil, fmt.Errorf("parsing network YAML: %w", err)
	}
	return &network, nil
}

// networkNameFromProfile extracts the network name from a parsed profile.
// Checks eth0 first for backward compatibility, then falls back to
// finding the first device with type=nic.
func networkNameFromProfile(profile *incusProfile) string {
	if eth0, ok := profile.Devices["eth0"]; ok {
		if name, ok := eth0["network"]; ok {
			return name
		}
	}
	// Fallback: find first nic device
	for _, dev := range profile.Devices {
		if dev["type"] == "nic" {
			return dev["network"]
		}
	}
	return ""
}

// parseStorageInfoBytes parses the output of "incus storage info <pool> --bytes".
// Returns used and total bytes. The --bytes flag outputs integer byte values
// instead of human-friendly GiB suffixes, making parsing reliable.
// Returns (0, 0, error) if either field is not found or not parseable.
func parseStorageInfoBytes(output string) (usedBytes, totalBytes uint64, err error) {
	var foundUsed, foundTotal bool
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "space used:"):
			val := strings.TrimSpace(strings.TrimPrefix(line, "space used:"))
			usedBytes, err = strconv.ParseUint(val, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parsing space used bytes: %w", err)
			}
			foundUsed = true
		case strings.HasPrefix(line, "total space:"):
			val := strings.TrimSpace(strings.TrimPrefix(line, "total space:"))
			totalBytes, err = strconv.ParseUint(val, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parsing total space bytes: %w", err)
			}
			foundTotal = true
		}
	}
	if !foundUsed || !foundTotal {
		return 0, 0, fmt.Errorf("storage info output missing space used or total space fields")
	}
	return usedBytes, totalBytes, nil
}

// parseStorageInfoGiB is the fallback parser for "incus storage info <pool>"
// without --bytes. Parses human-friendly "X.XXGiB" suffix format using fmt.Sscanf.
// Returns values in GiB as float64. Returns (0, 0, error) if parsing fails.
func parseStorageInfoGiB(output string) (usedGiB, totalGiB float64, err error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "space used:") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "space used:"), "%f", &usedGiB)
		} else if strings.HasPrefix(line, "total space:") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "total space:"), "%f", &totalGiB)
		}
	}
	if totalGiB == 0 {
		return 0, 0, fmt.Errorf("could not parse GiB values from storage info output")
	}
	return usedGiB, totalGiB, nil
}

// poolNameFromProfile extracts the storage pool name from a parsed profile.
// Looks for a device with type=disk and returns its "pool" value.
// Returns "default" if no disk device is found.
func poolNameFromProfile(profile *incusProfile) string {
	// Check "root" device first (standard naming)
	if root, ok := profile.Devices["root"]; ok {
		if pool, ok := root["pool"]; ok {
			return pool
		}
	}
	// Fallback: find first disk device
	for _, dev := range profile.Devices {
		if dev["type"] == "disk" {
			if pool, ok := dev["pool"]; ok {
				return pool
			}
		}
	}
	return "default"
}
