package config

import (
	"log"
	"sync"
	"time"
)

// ConfigManager provides thread-safe access to the application configuration
// and manages file-watching for hot-reload.
type ConfigManager struct {
	mu       sync.RWMutex
	cfg      *Config
	watcher  *Watcher
	onChange func(old, updated *Config) // optional callback for config changes
}

// NewConfigManager creates a ConfigManager, loads the initial config, and
// starts watching the system and user config files for changes.
// onChange is called (in a separate goroutine) after a successful reload.
func NewConfigManager(onChange func(old, updated *Config)) (*ConfigManager, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	cm := &ConfigManager{
		cfg:      cfg,
		onChange: onChange,
	}

	paths := WatchPaths()
	w, err := NewWatcher(paths, cm.reload, 1*time.Second)
	if err != nil {
		// Watcher failure is non-fatal — the server works, just without hot-reload.
		log.Printf("[config-manager] warning: file watcher unavailable: %v", err)
		return cm, nil
	}

	cm.watcher = w
	w.Start()
	log.Printf("[config-manager] watching config files: %v", paths)

	return cm, nil
}

// Get returns the current configuration. Thread-safe for concurrent reads.
func (cm *ConfigManager) Get() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cfg
}

// reload re-reads config from disk and swaps the reference. On error the
// previous valid config is retained (AC7).
func (cm *ConfigManager) reload() error {
	newCfg, err := Load()
	if err != nil {
		log.Printf("[config-manager] reload error (keeping previous config): %v", err)
		// AC7: do not crash, retain previous config.
		return nil
	}

	cm.mu.Lock()
	oldCfg := cm.cfg
	cm.cfg = newCfg
	cm.mu.Unlock()

	log.Printf("[config-manager] config reloaded successfully")

	if cm.onChange != nil {
		cm.onChange(oldCfg, newCfg)
	}

	return nil
}

// Close stops the file watcher and releases resources.
func (cm *ConfigManager) Close() {
	if cm.watcher != nil {
		_ = cm.watcher.Close()
	}
}
