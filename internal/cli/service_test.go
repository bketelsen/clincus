package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUnitFile(t *testing.T) {
	result, err := renderUnitFile("/home/testuser/.local/bin/clincus")
	if err != nil {
		t.Fatalf("renderUnitFile() error: %v", err)
	}

	// Must contain required systemd sections
	if !strings.Contains(result, "[Unit]") {
		t.Error("missing [Unit] section")
	}
	if !strings.Contains(result, "[Service]") {
		t.Error("missing [Service] section")
	}
	if !strings.Contains(result, "[Install]") {
		t.Error("missing [Install] section")
	}

	// Must reference the binary path
	if !strings.Contains(result, "/home/testuser/.local/bin/clincus serve") {
		t.Error("ExecStart should reference 'clincus serve'")
	}

	// Must target user session
	if !strings.Contains(result, "WantedBy=default.target") {
		t.Error("should be WantedBy=default.target for user units")
	}

	// Must use notify or simple type
	if !strings.Contains(result, "Type=simple") {
		t.Error("should use Type=simple")
	}

	// Must have restart policy
	if !strings.Contains(result, "Restart=on-failure") {
		t.Error("should restart on failure")
	}
}

func TestServiceUnitPath(t *testing.T) {
	path := serviceUnitPath()

	// Must be under ~/.config/systemd/user/
	if !strings.Contains(path, filepath.Join(".config", "systemd", "user")) {
		t.Errorf("expected path under .config/systemd/user/, got %s", path)
	}

	// Must end with clincus.service
	if !strings.HasSuffix(path, "clincus.service") {
		t.Errorf("expected path ending in clincus.service, got %s", path)
	}
}

func TestResolveBinaryPath(t *testing.T) {
	path, err := resolveBinaryPath()
	if err != nil {
		t.Fatalf("resolveBinaryPath() error: %v", err)
	}

	// Must be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}

	// During testing, os.Executable() returns the test binary, so just
	// verify it returns a valid absolute path without error.
	if path == "" {
		t.Error("expected non-empty path")
	}
}
