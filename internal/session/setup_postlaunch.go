package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/bketelsen/clincus/internal/limits"
)

// postLaunchSetup performs post-launch setup: wait for ready, set metadata, record history,
// and start timeout monitor. Steps 6-7 of Setup().
func postLaunchSetup(opts *SetupOptions, result *SetupResult) error {
	// 6. Wait for ready
	opts.Logger("Waiting for container to be ready...")
	if err := waitForReady(result.Manager, 30, opts.Logger); err != nil {
		return err
	}

	// 6b. Set metadata labels for dashboard discovery
	if opts.Tool != nil {
		//nolint:errcheck // metadata labels are best-effort; container is already running
		_ = result.Manager.SetConfig("user.clincus.managed", "true")
		_ = result.Manager.SetConfig("user.clincus.workspace", opts.WorkspacePath)
		_ = result.Manager.SetConfig("user.clincus.tool", opts.Tool.Name())
		_ = result.Manager.SetConfig("user.clincus.persistent", fmt.Sprintf("%v", opts.Persistent))
		_ = result.Manager.SetConfig("user.clincus.created", time.Now().UTC().Format(time.RFC3339))
	}

	// 6c. Record session start in history
	home, _ := os.UserHomeDir()
	histPath := filepath.Join(home, ".clincus", "history.jsonl")
	hist := &History{Path: histPath}
	toolName := ""
	if opts.Tool != nil {
		toolName = opts.Tool.Name()
	}
	//nolint:errcheck // history recording failure is non-fatal
	_ = hist.RecordStart(result.ContainerName, opts.WorkspacePath, toolName, opts.Persistent)

	// 7. Start timeout monitor if max_duration is configured
	if opts.LimitsConfig != nil && opts.LimitsConfig.Runtime.MaxDuration != "" {
		duration, err := limits.ParseDuration(opts.LimitsConfig.Runtime.MaxDuration)
		if err != nil {
			return fmt.Errorf("invalid max_duration: %w", err)
		}
		if duration > 0 {
			result.TimeoutMonitor = limits.NewTimeoutMonitor(
				result.ContainerName,
				duration,
				opts.LimitsConfig.Runtime.AutoStop,
				opts.LimitsConfig.Runtime.StopGraceful,
				opts.IncusProject,
				opts.Logger,
			)
			result.TimeoutMonitor.Start()
		}
	}

	return nil
}

// waitForReady waits for container to be ready
func waitForReady(mgr *container.Manager, maxRetries int, logger func(string)) error {
	for i := 0; i < maxRetries; i++ {
		running, err := mgr.Running()
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}

		if running {
			// Additional check: try to execute a simple command
			_, err := mgr.ExecCommand("echo ready", container.ExecCommandOptions{Capture: true})
			if err == nil {
				return nil
			}
		}

		time.Sleep(1 * time.Second)
		if i%5 == 0 && i > 0 {
			logger(fmt.Sprintf("Still waiting... (%ds)", i))
		}
	}

	return fmt.Errorf("container failed to become ready after %d seconds", maxRetries)
}
