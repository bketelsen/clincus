package health

import (
	"math"
	"strings"
	"testing"

	"github.com/bketelsen/clincus/internal/container"
)

// Sample fixtures for storage tests
const storageProfileYAML = `config: {}
devices:
  eth0:
    name: eth0
    network: incusbr0
    type: nic
  root:
    path: /
    pool: mypool
    type: disk
name: default`

const storageInfoBytes = `info:
  description: ""
  driver: zfs
  name: mypool
  status: Created
  total space: 107374182400
  used by: []
  space used: 21474836480`

const storageInfoGiB = `info:
  description: ""
  driver: zfs
  name: mypool
  status: Created
  total space: 100.00GiB
  used by: []
  space used: 20.00GiB`

const storageInfoBytesLow = `info:
  description: ""
  driver: zfs
  name: mypool
  status: Created
  total space: 107374182400
  used by: []
  space used: 105226698752`

const storageInfoBytesCritical = `info:
  description: ""
  driver: zfs
  name: mypool
  status: Created
  total space: 107374182400
  used by: []
  space used: 106300440576`

func TestCheckIncusStoragePool_BytesPath(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: storageInfoBytes, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	if result.Status != StatusOK {
		t.Errorf("Status = %q, want %q", result.Status, StatusOK)
	}
	if result.Details == nil {
		t.Fatal("Details is nil, want non-nil")
	}
	if result.Details["pool"] != "mypool" {
		t.Errorf("Details[pool] = %q, want %q", result.Details["pool"], "mypool")
	}
	usedGiB, ok := result.Details["used_gib"].(float64)
	if !ok {
		t.Fatalf("Details[used_gib] not float64: %T", result.Details["used_gib"])
	}
	if math.Abs(usedGiB-20.0) > 0.1 {
		t.Errorf("Details[used_gib] = %f, want ~20.0", usedGiB)
	}
	totalGiB, ok := result.Details["total_gib"].(float64)
	if !ok {
		t.Fatalf("Details[total_gib] not float64: %T", result.Details["total_gib"])
	}
	if math.Abs(totalGiB-100.0) > 0.1 {
		t.Errorf("Details[total_gib] = %f, want ~100.0", totalGiB)
	}
}

func TestCheckIncusStoragePool_GiBFallback(t *testing.T) {
	// --bytes output has non-numeric values, but GiB values are parseable
	badBytesOutput := `info:
  description: ""
  driver: zfs
  name: mypool
  status: Created
  total space: 100.00GiB
  used by: []
  space used: 20.00GiB`

	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: badBytesOutput, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	if result.Status != StatusOK {
		t.Errorf("Status = %q, want %q (GiB fallback should succeed)", result.Status, StatusOK)
	}
}

func TestCheckIncusStoragePool_BothParseFail(t *testing.T) {
	unparseable := `info:
  description: ""
  driver: zfs
  name: mypool
  status: Created
  total space: UNKNOWN
  used by: []
  space used: UNKNOWN`

	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: unparseable, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	if result.Status != StatusWarning {
		t.Errorf("Status = %q, want %q (both parse paths fail)", result.Status, StatusWarning)
	}
}

func TestCheckIncusStoragePool_StorageCommandError(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: "", exitCode: 1})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	if result.Status != StatusWarning {
		t.Errorf("Status = %q, want %q", result.Status, StatusWarning)
	}
	if !strings.Contains(result.Message, "Could not query storage pool") {
		t.Errorf("Message = %q, want it to contain 'Could not query storage pool'", result.Message)
	}
}

func TestCheckIncusStoragePool_ProfileCommandError(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: "", exitCode: 1})
	mock.on("storage info", mockResponse{stdout: storageInfoBytes, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	// Should continue with "default" pool name
	if result.Status != StatusOK {
		t.Errorf("Status = %q, want %q (should use default pool)", result.Status, StatusOK)
	}
}

func TestCheckIncusStoragePool_LowSpace(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: storageInfoBytesLow, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	// ~98% used, but free is ~2GiB so this is StatusFailed (>90% used)
	if result.Status != StatusFailed {
		t.Errorf("Status = %q, want %q (>90%% used)", result.Status, StatusFailed)
	}
	if !strings.Contains(result.Message, "critically low") {
		t.Errorf("Message = %q, want it to contain 'critically low'", result.Message)
	}
}

func TestCheckIncusStoragePool_CriticalSpace(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: storageInfoBytesCritical, exitCode: 0})
	defer container.SetRunner(mock)()

	result := CheckIncusStoragePool()

	if result.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", result.Status, StatusFailed)
	}
	if !strings.Contains(result.Message, "critically low") {
		t.Errorf("Message = %q, want it to contain 'critically low'", result.Message)
	}
}

func TestCheckIncusStoragePool_UsesCommandRunner(t *testing.T) {
	mock := newHealthMockRunner()
	mock.on("profile show default", mockResponse{stdout: storageProfileYAML, exitCode: 0})
	mock.on("storage info", mockResponse{stdout: storageInfoBytes, exitCode: 0})
	defer container.SetRunner(mock)()

	_ = CheckIncusStoragePool()

	if len(mock.calls) == 0 {
		t.Error("mock.calls is empty; CheckIncusStoragePool did not route through CommandRunner")
	}
}
