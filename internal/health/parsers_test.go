package health

import (
	"testing"
)

func TestParseProfileYAML_Valid(t *testing.T) {
	input := `name: default
config:
  boot.autostart: "true"
devices:
  eth0:
    network: incusbr0
    type: nic
  root:
    path: /
    pool: default
    type: disk`

	profile, err := parseProfileYAML(input)
	if err != nil {
		t.Fatalf("parseProfileYAML() error: %v", err)
	}
	if profile.Name != "default" {
		t.Errorf("profile.Name = %q, want %q", profile.Name, "default")
	}
	if len(profile.Devices) != 2 {
		t.Errorf("len(profile.Devices) = %d, want 2", len(profile.Devices))
	}
}

func TestParseProfileYAML_Invalid(t *testing.T) {
	_, err := parseProfileYAML(":::invalid yaml")
	if err == nil {
		t.Error("parseProfileYAML() returned nil error for invalid YAML")
	}
}

func TestParseNetworkYAML_Valid(t *testing.T) {
	input := `name: incusbr0
config:
  ipv4.address: 10.0.0.1/24
  ipv4.nat: "true"
type: bridge
status: Created
managed: true`

	network, err := parseNetworkYAML(input)
	if err != nil {
		t.Fatalf("parseNetworkYAML() error: %v", err)
	}
	if network.Name != "incusbr0" {
		t.Errorf("network.Name = %q, want %q", network.Name, "incusbr0")
	}
	if network.Type != "bridge" {
		t.Errorf("network.Type = %q, want %q", network.Type, "bridge")
	}
	if !network.Managed {
		t.Error("network.Managed = false, want true")
	}
}

func TestParseNetworkYAML_Invalid(t *testing.T) {
	_, err := parseNetworkYAML(":::invalid yaml")
	if err == nil {
		t.Error("parseNetworkYAML() returned nil error for invalid YAML")
	}
}

func TestNetworkNameFromProfile_Eth0(t *testing.T) {
	profile := &incusProfile{
		Devices: map[string]map[string]string{
			"eth0": {"network": "incusbr0", "type": "nic"},
		},
	}
	got := networkNameFromProfile(profile)
	if got != "incusbr0" {
		t.Errorf("networkNameFromProfile() = %q, want %q", got, "incusbr0")
	}
}

func TestNetworkNameFromProfile_FallbackNic(t *testing.T) {
	profile := &incusProfile{
		Devices: map[string]map[string]string{
			"root":   {"type": "disk", "pool": "default"},
			"mynet0": {"network": "customnet", "type": "nic"},
		},
	}
	got := networkNameFromProfile(profile)
	if got != "customnet" {
		t.Errorf("networkNameFromProfile() = %q, want %q", got, "customnet")
	}
}

func TestNetworkNameFromProfile_NoNic(t *testing.T) {
	profile := &incusProfile{
		Devices: map[string]map[string]string{
			"root": {"type": "disk", "pool": "default"},
		},
	}
	got := networkNameFromProfile(profile)
	if got != "" {
		t.Errorf("networkNameFromProfile() = %q, want empty", got)
	}
}

func TestParseStorageInfoBytes_Valid(t *testing.T) {
	input := `  space used: 5368709120
  total space: 107374182400`

	used, total, err := parseStorageInfoBytes(input)
	if err != nil {
		t.Fatalf("parseStorageInfoBytes() error: %v", err)
	}
	if used != 5368709120 {
		t.Errorf("used = %d, want 5368709120", used)
	}
	if total != 107374182400 {
		t.Errorf("total = %d, want 107374182400", total)
	}
}

func TestParseStorageInfoBytes_MissingFields(t *testing.T) {
	_, _, err := parseStorageInfoBytes("some random output")
	if err == nil {
		t.Error("parseStorageInfoBytes() returned nil error for missing fields")
	}
}

func TestParseStorageInfoBytes_InvalidNumber(t *testing.T) {
	input := `  space used: notanumber
  total space: 107374182400`
	_, _, err := parseStorageInfoBytes(input)
	if err == nil {
		t.Error("parseStorageInfoBytes() returned nil error for invalid number")
	}
}

func TestParseStorageInfoGiB_Valid(t *testing.T) {
	input := `  space used: 5.00GiB
  total space: 100.00GiB`

	used, total, err := parseStorageInfoGiB(input)
	if err != nil {
		t.Fatalf("parseStorageInfoGiB() error: %v", err)
	}
	if used != 5.0 {
		t.Errorf("used = %f, want 5.0", used)
	}
	if total != 100.0 {
		t.Errorf("total = %f, want 100.0", total)
	}
}

func TestParseStorageInfoGiB_ZeroTotal(t *testing.T) {
	_, _, err := parseStorageInfoGiB("some random output")
	if err == nil {
		t.Error("parseStorageInfoGiB() returned nil error for zero total")
	}
}

func TestPoolNameFromProfile_Root(t *testing.T) {
	profile := &incusProfile{
		Devices: map[string]map[string]string{
			"root": {"type": "disk", "pool": "mypool"},
		},
	}
	got := poolNameFromProfile(profile)
	if got != "mypool" {
		t.Errorf("poolNameFromProfile() = %q, want %q", got, "mypool")
	}
}

func TestPoolNameFromProfile_FallbackDisk(t *testing.T) {
	profile := &incusProfile{
		Devices: map[string]map[string]string{
			"storage": {"type": "disk", "pool": "otherPool"},
		},
	}
	got := poolNameFromProfile(profile)
	if got != "otherPool" {
		t.Errorf("poolNameFromProfile() = %q, want %q", got, "otherPool")
	}
}

func TestPoolNameFromProfile_Default(t *testing.T) {
	profile := &incusProfile{
		Devices: map[string]map[string]string{
			"eth0": {"type": "nic", "network": "incusbr0"},
		},
	}
	got := poolNameFromProfile(profile)
	if got != "default" {
		t.Errorf("poolNameFromProfile() = %q, want %q", got, "default")
	}
}
