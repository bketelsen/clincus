package health

import (
	"testing"

	"github.com/bketelsen/clincus/internal/container"
)

// Sample YAML fixtures for tests
const profileYAML = `config: {}
description: Default Incus profile
devices:
  eth0:
    name: eth0
    network: incusbr0
    type: nic
  root:
    path: /
    pool: default
    type: disk
name: default
used_by: []`

const profileYAMLCustomNic = `config: {}
devices:
  mynet:
    name: mynet
    network: custombr0
    type: nic
  root:
    path: /
    pool: default
    type: disk
name: default`

const profileYAMLNoNic = `config: {}
devices:
  root:
    path: /
    pool: default
    type: disk
name: default`

const networkYAML = `config:
  ipv4.address: 10.0.0.1/24
  ipv4.nat: "true"
  ipv6.address: fd42::1/64
description: ""
managed: true
name: incusbr0
status: Created
type: bridge`

const networkYAMLNoIPv4 = `config:
  ipv4.address: none
description: ""
managed: true
name: incusbr0
status: Created
type: bridge`

func TestCheckNetworkBridge_YAMLProfile(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: profileYAML, exitCode: 0})
	mock.on("network show", mockResponse{stdout: networkYAML, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusOK {
		t.Errorf("Status = %q, want %q", result.Status, StatusOK)
	}
	if result.Details == nil {
		t.Fatal("Details is nil, want non-nil")
	}
	if result.Details["name"] != "incusbr0" {
		t.Errorf("Details[name] = %q, want %q", result.Details["name"], "incusbr0")
	}
	if result.Details["ipv4"] != "10.0.0.1/24" {
		t.Errorf("Details[ipv4] = %q, want %q", result.Details["ipv4"], "10.0.0.1/24")
	}
}

func TestCheckNetworkBridge_CustomNicDevice(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: profileYAMLCustomNic, exitCode: 0})
	mock.on("network show", mockResponse{stdout: networkYAML, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusOK {
		t.Errorf("Status = %q, want %q", result.Status, StatusOK)
	}
	if result.Details == nil {
		t.Fatal("Details is nil, want non-nil")
	}
	if result.Details["name"] != "custombr0" {
		t.Errorf("Details[name] = %q, want %q", result.Details["name"], "custombr0")
	}
}

func TestCheckNetworkBridge_NoNicDevice(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: profileYAMLNoNic, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", result.Status, StatusFailed)
	}
	if result.Message == "" {
		t.Error("Message is empty, want non-empty")
	}
}

func TestCheckNetworkBridge_NoIPv4(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: profileYAML, exitCode: 0})
	mock.on("network show", mockResponse{stdout: networkYAMLNoIPv4, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", result.Status, StatusFailed)
	}
	if result.Message == "" {
		t.Error("Message is empty, want non-empty")
	}
}

func TestCheckNetworkBridge_ProfileYAMLFallback(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: "<<<not yaml", exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusWarning {
		t.Errorf("Status = %q, want %q (fallback on unparseable profile)", result.Status, StatusWarning)
	}
}

func TestCheckNetworkBridge_NetworkYAMLFallback(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: profileYAML, exitCode: 0})
	mock.on("network show", mockResponse{stdout: "<<<not yaml", exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusWarning {
		t.Errorf("Status = %q, want %q (fallback on unparseable network)", result.Status, StatusWarning)
	}
}

func TestCheckNetworkBridge_ProfileCommandError(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: "", exitCode: 1})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusWarning {
		t.Errorf("Status = %q, want %q", result.Status, StatusWarning)
	}
}

func TestCheckNetworkBridge_EnrichedDetails(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: profileYAML, exitCode: 0})
	mock.on("network show", mockResponse{stdout: networkYAML, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckNetworkBridge()

	if result.Status != StatusOK {
		t.Fatalf("Status = %q, want %q", result.Status, StatusOK)
	}
	if result.Details == nil {
		t.Fatal("Details is nil, want non-nil")
	}

	expectedKeys := []string{"name", "ipv4", "driver", "status"}
	for _, key := range expectedKeys {
		if _, ok := result.Details[key]; !ok {
			t.Errorf("Details missing key %q", key)
		}
	}

	if result.Details["driver"] != "bridge" {
		t.Errorf("Details[driver] = %q, want %q", result.Details["driver"], "bridge")
	}
	if result.Details["status"] != "Created" {
		t.Errorf("Details[status] = %q, want %q", result.Details["status"], "Created")
	}
}
