package config

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcherDetectsFileModification(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")

	// Create initial file.
	if err := os.WriteFile(cfgFile, []byte("[defaults]\nimage = \"old\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var reloadCount atomic.Int32
	w, err := NewWatcher([]string{cfgFile}, func() error {
		reloadCount.Add(1)
		return nil
	}, 200*time.Millisecond) // short debounce for tests
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Start()

	// Give the watcher time to start.
	time.Sleep(100 * time.Millisecond)

	// Modify the file.
	if err := os.WriteFile(cfgFile, []byte("[defaults]\nimage = \"new\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + processing.
	time.Sleep(500 * time.Millisecond)

	if reloadCount.Load() < 1 {
		t.Errorf("expected at least 1 reload, got %d", reloadCount.Load())
	}
}

func TestWatcherDebouncing(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(cfgFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var reloadCount atomic.Int32
	w, err := NewWatcher([]string{cfgFile}, func() error {
		reloadCount.Add(1)
		return nil
	}, 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Start()
	time.Sleep(100 * time.Millisecond)

	// Rapid-fire writes — should collapse into one debounced reload (AC6).
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(cfgFile, []byte("y"), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce to fire.
	time.Sleep(800 * time.Millisecond)

	count := reloadCount.Load()
	if count != 1 {
		t.Errorf("expected exactly 1 debounced reload, got %d", count)
	}
}

func TestWatcherHandlesNonExistentPath(t *testing.T) {
	dir := t.TempDir()
	nonExistent := filepath.Join(dir, "does-not-exist.toml")

	var reloadCount atomic.Int32
	w, err := NewWatcher([]string{nonExistent}, func() error {
		reloadCount.Add(1)
		return nil
	}, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Start()
	time.Sleep(100 * time.Millisecond)

	// Create the file that didn't exist at startup (AC8).
	if err := os.WriteFile(nonExistent, []byte("[defaults]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(600 * time.Millisecond)

	if reloadCount.Load() < 1 {
		t.Errorf("expected at least 1 reload after file creation, got %d", reloadCount.Load())
	}
}

func TestWatcherHandlesNonExistentParentDir(t *testing.T) {
	// If both file and parent dir don't exist, watcher should not crash (AC8).
	nonExistent := "/tmp/clincus-test-nonexistent-dir-12345/config.toml"

	w, err := NewWatcher([]string{nonExistent}, func() error {
		return nil
	}, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Start()

	// Just verify no panic or error. The watcher silently skips this path.
	time.Sleep(100 * time.Millisecond)
}
