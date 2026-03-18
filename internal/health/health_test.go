package health

import (
	"testing"

	"github.com/bketelsen/clincus/internal/config"
)

func TestCalculateSummary(t *testing.T) {
	checks := map[string]HealthCheck{
		"ok1":      {Status: StatusOK},
		"ok2":      {Status: StatusOK},
		"warn1":    {Status: StatusWarning},
		"failed1":  {Status: StatusFailed},
	}

	summary := calculateSummary(checks)
	if summary.Total != 4 {
		t.Errorf("Total = %d, want 4", summary.Total)
	}
	if summary.Passed != 2 {
		t.Errorf("Passed = %d, want 2", summary.Passed)
	}
	if summary.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", summary.Warnings)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
}

func TestDetermineStatus_Healthy(t *testing.T) {
	checks := map[string]HealthCheck{
		"ok1": {Status: StatusOK},
		"ok2": {Status: StatusOK},
	}
	status := determineStatus(checks)
	if status != OverallHealthy {
		t.Errorf("determineStatus() = %q, want %q", status, OverallHealthy)
	}
}

func TestDetermineStatus_Degraded(t *testing.T) {
	checks := map[string]HealthCheck{
		"ok1":   {Status: StatusOK},
		"warn1": {Status: StatusWarning},
	}
	status := determineStatus(checks)
	if status != OverallDegraded {
		t.Errorf("determineStatus() = %q, want %q", status, OverallDegraded)
	}
}

func TestDetermineStatus_Unhealthy(t *testing.T) {
	checks := map[string]HealthCheck{
		"ok1":     {Status: StatusOK},
		"failed1": {Status: StatusFailed},
	}
	status := determineStatus(checks)
	if status != OverallUnhealthy {
		t.Errorf("determineStatus() = %q, want %q", status, OverallUnhealthy)
	}
}

func TestDetermineStatus_FailedOverridesWarning(t *testing.T) {
	checks := map[string]HealthCheck{
		"warn1":   {Status: StatusWarning},
		"failed1": {Status: StatusFailed},
	}
	status := determineStatus(checks)
	if status != OverallUnhealthy {
		t.Errorf("determineStatus() = %q, want %q (failed should override warning)", status, OverallUnhealthy)
	}
}

func TestExitCode_Healthy(t *testing.T) {
	r := &HealthResult{Status: OverallHealthy}
	if r.ExitCode() != 0 {
		t.Errorf("ExitCode() = %d, want 0", r.ExitCode())
	}
}

func TestExitCode_Degraded(t *testing.T) {
	r := &HealthResult{Status: OverallDegraded}
	if r.ExitCode() != 1 {
		t.Errorf("ExitCode() = %d, want 1", r.ExitCode())
	}
}

func TestExitCode_Unhealthy(t *testing.T) {
	r := &HealthResult{Status: OverallUnhealthy}
	if r.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, want 2", r.ExitCode())
	}
}

func TestExitCode_Unknown(t *testing.T) {
	r := &HealthResult{Status: "unknown"}
	if r.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, want 2 (default)", r.ExitCode())
	}
}

// --- Check function tests (no mocking needed, work on any system) ---

func TestCheckOS_ReturnsOK(t *testing.T) {
	result := CheckOS()
	if result.Status != StatusOK {
		t.Errorf("CheckOS().Status = %q, want %q", result.Status, StatusOK)
	}
	if result.Name != "os" {
		t.Errorf("CheckOS().Name = %q, want %q", result.Name, "os")
	}
	if result.Message == "" {
		t.Error("CheckOS().Message is empty")
	}
	if result.Details["os"] == "" {
		t.Error("CheckOS().Details[os] is empty")
	}
}

func TestCheckIPForwarding_DoesNotPanic(t *testing.T) {
	result := CheckIPForwarding()
	if result.Name != "ip_forwarding" {
		t.Errorf("CheckIPForwarding().Name = %q, want %q", result.Name, "ip_forwarding")
	}
	// Status can be OK, Warning, or Failed depending on system -- just verify no panic
}

func TestCheckConfiguration_NilConfig(t *testing.T) {
	result := CheckConfiguration(nil)
	if result.Status != StatusFailed {
		t.Errorf("CheckConfiguration(nil).Status = %q, want %q", result.Status, StatusFailed)
	}
}

func TestCheckConfiguration_ValidConfig(t *testing.T) {
	cfg := config.GetDefaultConfig()
	result := CheckConfiguration(cfg)
	if result.Name != "config" {
		t.Errorf("CheckConfiguration().Name = %q, want %q", result.Name, "config")
	}
	if result.Status != StatusOK {
		t.Errorf("CheckConfiguration().Status = %q, want %q", result.Status, StatusOK)
	}
}

func TestCheckTool_Default(t *testing.T) {
	result := CheckTool("")
	if result.Name != "tool" {
		t.Errorf("CheckTool().Name = %q, want %q", result.Name, "tool")
	}
	// Should default to "claude" which is a known tool
	if result.Status != StatusOK {
		t.Errorf("CheckTool().Status = %q, want %q", result.Status, StatusOK)
	}
}

func TestCheckTool_UnknownTool(t *testing.T) {
	result := CheckTool("nonexistent-tool-xyz")
	if result.Status != StatusWarning {
		t.Errorf("CheckTool(unknown).Status = %q, want %q", result.Status, StatusWarning)
	}
}

func TestCheckDiskSpace_DoesNotPanic(t *testing.T) {
	result := CheckDiskSpace()
	if result.Name != "disk_space" {
		t.Errorf("CheckDiskSpace().Name = %q, want %q", result.Name, "disk_space")
	}
}

func TestCheckCgroupAvailability_DoesNotPanic(t *testing.T) {
	result := CheckCgroupAvailability()
	if result.Name != "cgroup_availability" {
		t.Errorf("CheckCgroupAvailability().Name = %q, want %q", result.Name, "cgroup_availability")
	}
}
