package health

import (
	"fmt"

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
