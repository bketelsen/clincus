package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bketelsen/clincus/internal/config"
	"github.com/bketelsen/clincus/internal/session"
)

// ParseMountConfig creates MountConfig from config file and CLI flags
func ParseMountConfig(cfg *config.Config, mountPairs []string) (*session.MountConfig, error) {
	// Step 1: Add config file default mounts
	mountConfig, err := session.MountConfigFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	deviceNameCounter := len(mountConfig.Mounts)

	// Step 2: Add --mount flags (can override config mounts)
	for _, pair := range mountPairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mount format '%s': expected HOST:CONTAINER", pair)
		}

		hostPath := strings.TrimSpace(parts[0])
		containerPath := strings.TrimSpace(parts[1])

		// Expand host path
		hostPath = config.ExpandPath(hostPath)
		absHost, err := filepath.Abs(hostPath)
		if err != nil {
			return nil, fmt.Errorf("invalid mount host path '%s': %w", hostPath, err)
		}

		// Validate container path
		if !filepath.IsAbs(containerPath) {
			return nil, fmt.Errorf("container path must be absolute: %s", containerPath)
		}
		containerPath = filepath.Clean(containerPath)

		// Check if this container path already exists (override)
		mountExists := false
		for i, m := range mountConfig.Mounts {
			if m.ContainerPath == containerPath {
				// CLI mount overrides config/storage mount
				mountConfig.Mounts[i].HostPath = absHost
				mountExists = true
				break
			}
		}

		if !mountExists {
			mountConfig.Mounts = append(mountConfig.Mounts, session.MountEntry{
				HostPath:      absHost,
				ContainerPath: containerPath,
				DeviceName:    fmt.Sprintf("mount-%d", deviceNameCounter),
			})
			deviceNameCounter++
		}
	}

	return mountConfig, nil
}
