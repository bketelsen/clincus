package session

import (
	"fmt"
	"path/filepath"

	"github.com/bketelsen/clincus/internal/config"
)

// MountEntry represents a single directory mount at runtime
type MountEntry struct {
	HostPath      string // Absolute path on host (expanded)
	ContainerPath string // Absolute path in container
	DeviceName    string // Unique device name for Incus
	UseShift      bool   // Whether to use UID shifting
}

// MountConfig holds all mount configurations for a session
type MountConfig struct {
	Mounts []MountEntry
}

// MountConfigFromConfig creates a MountConfig from config file default mounts.
// This handles tilde expansion and path validation for the config-defined mounts.
// CLI-specific mount flags are handled separately in the cli package.
func MountConfigFromConfig(cfg *config.Config) (*MountConfig, error) {
	mountConfig := &MountConfig{
		Mounts: []MountEntry{},
	}

	for i, cfgMount := range cfg.Mounts.Default {
		hostPath := config.ExpandPath(cfgMount.Host)
		absHost, err := filepath.Abs(hostPath)
		if err != nil {
			return nil, fmt.Errorf("invalid config mount host path '%s': %w", cfgMount.Host, err)
		}

		if !filepath.IsAbs(cfgMount.Container) {
			return nil, fmt.Errorf("config mount container path must be absolute: %s", cfgMount.Container)
		}

		mountConfig.Mounts = append(mountConfig.Mounts, MountEntry{
			HostPath:      absHost,
			ContainerPath: filepath.Clean(cfgMount.Container),
			DeviceName:    fmt.Sprintf("mount-%d", i),
		})
	}

	return mountConfig, nil
}
