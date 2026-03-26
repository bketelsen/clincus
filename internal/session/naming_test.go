package session

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"testing"
)

func TestWorkspaceHash(t *testing.T) {
	tests := []struct {
		name          string
		workspacePath string
		wantLength    int
	}{
		{
			name:          "simple path",
			workspacePath: "/home/user/project",
			wantLength:    8,
		},
		{
			name:          "complex path",
			workspacePath: "/home/user/my-workspace/project-1",
			wantLength:    8,
		},
		{
			name:          "empty path",
			workspacePath: "",
			wantLength:    8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := WorkspaceHash(tt.workspacePath)

			if len(hash) != tt.wantLength {
				t.Errorf("WorkspaceHash() returned length %d, want %d", len(hash), tt.wantLength)
			}

			// Verify it's hex
			for _, c := range hash {
				if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
					t.Errorf("WorkspaceHash() contains non-hex character: %c", c)
				}
			}
		})
	}

	// Test deterministic hashing
	path := "/test/path"
	hash1 := WorkspaceHash(path)
	hash2 := WorkspaceHash(path)

	if hash1 != hash2 {
		t.Errorf("WorkspaceHash() not deterministic: %s != %s", hash1, hash2)
	}

	// Test different paths produce different hashes
	hash3 := WorkspaceHash("/different/path")
	if hash1 == hash3 {
		t.Error("WorkspaceHash() produced same hash for different paths")
	}
}

func TestWorkspaceHashFormat(t *testing.T) {
	// Test that the hash matches the expected SHA256 format
	path := "/test/workspace"
	hash := WorkspaceHash(path)

	// Calculate expected hash
	h := sha256.New()
	h.Write([]byte(path))
	expected := fmt.Sprintf("%x", h.Sum(nil))[:8]

	if hash != expected {
		t.Errorf("WorkspaceHash() = %s, want %s", hash, expected)
	}
}

func TestContainerName(t *testing.T) {
	tests := []struct {
		name          string
		workspacePath string
		slot          int
		wantPrefix    string
	}{
		{
			name:          "slot 1",
			workspacePath: "/home/user/project",
			slot:          1,
			wantPrefix:    "clincus-",
		},
		{
			name:          "slot 5",
			workspacePath: "/home/user/project",
			slot:          5,
			wantPrefix:    "clincus-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := ContainerName(tt.workspacePath, tt.slot)

			// Check prefix
			if len(name) < len(tt.wantPrefix) || name[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("ContainerName() = %s, want prefix %s", name, tt.wantPrefix)
			}

			// Check format: clincus-<hash>-<slot>
			expectedHash := WorkspaceHash(tt.workspacePath)
			expectedName := fmt.Sprintf("clincus-%s-%d", expectedHash, tt.slot)

			if name != expectedName {
				t.Errorf("ContainerName() = %s, want %s", name, expectedName)
			}
		})
	}

	// Test deterministic naming
	path := "/test/path"
	slot := 1
	name1 := ContainerName(path, slot)
	name2 := ContainerName(path, slot)

	if name1 != name2 {
		t.Errorf("ContainerName() not deterministic: %s != %s", name1, name2)
	}
}

func TestContainerNameDifferentSlots(t *testing.T) {
	path := "/test/workspace"
	name1 := ContainerName(path, 1)
	name2 := ContainerName(path, 2)

	if name1 == name2 {
		t.Error("ContainerName() produced same name for different slots")
	}

	// Check that only the slot number differs
	hash := WorkspaceHash(path)
	expected1 := fmt.Sprintf("clincus-%s-1", hash)
	expected2 := fmt.Sprintf("clincus-%s-2", hash)

	if name1 != expected1 {
		t.Errorf("ContainerName(slot 1) = %s, want %s", name1, expected1)
	}

	if name2 != expected2 {
		t.Errorf("ContainerName(slot 2) = %s, want %s", name2, expected2)
	}
}

func TestParseContainerName(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		wantHash      string
		wantSlot      int
		wantErr       bool
	}{
		{
			name:          "valid container name",
			containerName: "clincus-abc12345-1",
			wantHash:      "abc12345",
			wantSlot:      1,
			wantErr:       false,
		},
		{
			name:          "valid container name slot 10",
			containerName: "clincus-abc12345-10",
			wantHash:      "abc12345",
			wantSlot:      10,
			wantErr:       false,
		},
		{
			name:          "invalid format - no prefix",
			containerName: "container-abc12345-1",
			wantErr:       true,
		},
		{
			name:          "invalid format - short hash",
			containerName: "claude-abc123-1",
			wantErr:       true,
		},
		{
			name:          "invalid format - no slot",
			containerName: "claude-abc12345",
			wantErr:       true,
		},
		{
			name:          "invalid format - non-numeric slot",
			containerName: "claude-abc12345-abc",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, slot, err := ParseContainerName(tt.containerName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseContainerName() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseContainerName() unexpected error: %v", err)
				return
			}

			if hash != tt.wantHash {
				t.Errorf("ParseContainerName() hash = %s, want %s", hash, tt.wantHash)
			}

			if slot != tt.wantSlot {
				t.Errorf("ParseContainerName() slot = %d, want %d", slot, tt.wantSlot)
			}
		})
	}
}

func TestParseContainerNames(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "valid JSON",
			output: `[{"name":"clincus-abc12345-1"},{"name":"clincus-abc12345-2"},{"name":"other-container"}]`,
			want:   []string{"clincus-abc12345-1", "clincus-abc12345-2", "other-container"},
		},
		{
			name:   "empty JSON array",
			output: `[]`,
			want:   []string{},
		},
		{
			name:   "malformed JSON falls back to regex",
			output: `{"name": "clincus-abc12345-1"} some garbage {"name": "clincus-abc12345-2"}`,
			want:   []string{"clincus-abc12345-1", "clincus-abc12345-2"},
		},
		{
			name:   "completely invalid output",
			output: `not json at all`,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseContainerNames(tt.output)
			if len(got) != len(tt.want) {
				t.Errorf("parseContainerNames() returned %d names, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseContainerNames()[%d] = %s, want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMatchingSlots(t *testing.T) {
	re := regexp.MustCompile(`^clincus-abc12345-(\d+)$`)

	names := []string{
		"clincus-abc12345-1",
		"clincus-abc12345-3",
		"clincus-other000-2",
		"unrelated",
	}

	slots := matchingSlots(names, re)

	if !slots[1] {
		t.Error("matchingSlots() missing slot 1")
	}
	if !slots[3] {
		t.Error("matchingSlots() missing slot 3")
	}
	if slots[2] {
		t.Error("matchingSlots() incorrectly matched slot 2 from different hash")
	}
	if len(slots) != 2 {
		t.Errorf("matchingSlots() returned %d slots, want 2", len(slots))
	}
}

// TestAllocateSlot is an integration test that would need container mocking
// Skipping for now as it requires Incus interaction
func TestAllocateSlotLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// This would test AllocateSlot but requires mocking Incus commands
	// TODO: Add integration test
}

// TestAllocateSlotFrom is an integration test that would need container mocking
// Skipping for now as it requires Incus interaction
func TestAllocateSlotFromLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// This would test AllocateSlotFrom but requires mocking Incus commands
	// TODO: Add integration test
}
