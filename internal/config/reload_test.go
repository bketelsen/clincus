package config

import (
	"testing"
)

func TestConfigManagerGet(t *testing.T) {
	// NewConfigManager calls Load() which reads real config paths.
	// In test environments those paths don't exist, so defaults are used.
	cm, err := NewConfigManager(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cm.Close()

	cfg := cm.Get()
	if cfg == nil {
		t.Fatal("expected non-nil config from Get()")
	}

	// Should have default values.
	if cfg.Defaults.Image != "clincus" {
		t.Errorf("expected default image 'clincus', got '%s'", cfg.Defaults.Image)
	}
}

func TestConfigManagerReloadRetainsOnError(t *testing.T) {
	cm, err := NewConfigManager(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cm.Close()

	// Get the initial config.
	initial := cm.Get()
	initialImage := initial.Defaults.Image

	// Reload — since no config files changed, should return equivalent config.
	if err := cm.reload(); err != nil {
		t.Fatal(err)
	}

	reloaded := cm.Get()
	if reloaded.Defaults.Image != initialImage {
		t.Errorf("expected image '%s' after reload, got '%s'", initialImage, reloaded.Defaults.Image)
	}
}

func TestConfigManagerOnChangeCallback(t *testing.T) {
	callbackCalled := false
	cm, err := NewConfigManager(func(old, new *Config) {
		callbackCalled = true
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cm.Close()

	// Force a reload — onChange should fire.
	if err := cm.reload(); err != nil {
		t.Fatal(err)
	}

	if !callbackCalled {
		t.Error("expected onChange callback to be called after reload")
	}
}
