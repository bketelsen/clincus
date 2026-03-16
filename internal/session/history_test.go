package session

import (
	"path/filepath"
	"testing"
)

func TestRecordStartAndList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	h := &History{Path: path}

	err := h.RecordStart("clincus-abc12345-1", "/home/user/project", "claude", false)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := h.ListHistory(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "clincus-abc12345-1" {
		t.Errorf("expected clincus-abc12345-1, got %s", entries[0].ID)
	}
	if entries[0].Workspace != "/home/user/project" {
		t.Errorf("wrong workspace: %s", entries[0].Workspace)
	}
}

func TestRecordStop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	h := &History{Path: path}

	h.RecordStart("clincus-abc12345-1", "/home/user/project", "claude", false)
	h.RecordStop("clincus-abc12345-1", 0)

	entries, err := h.ListHistory(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Stopped == "" {
		t.Error("expected stopped time to be set")
	}
}

func TestListHistoryEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	h := &History{Path: path}

	entries, err := h.ListHistory(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil for nonexistent file, got %v", entries)
	}
}
