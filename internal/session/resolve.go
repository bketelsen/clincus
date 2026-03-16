package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bketelsen/clincus/internal/tool"
)

// ResolveOpts configures session resolution.
type ResolveOpts struct {
	Workspace  string
	Tool       tool.Tool
	Resume     bool
	ResumeID   string
	Slot       int
	Persistent bool
	MaxSlots   int
	// SessionsDir overrides automatic sessions directory determination.
	// If empty, derived from ~/.clincus/<tool.SessionsDirName()>.
	SessionsDir string
}

// ResolvedSession contains the resolved session parameters.
type ResolvedSession struct {
	ContainerName string
	SessionID     string
	Slot          int
	Persistent    bool
	IsResume      bool
	ResumeID      string
	SessionsDir   string
}

// Resolve determines session parameters: allocates a slot, generates the
// container name, and creates or reuses a session ID. Both the CLI and
// the web server call this function.
func Resolve(_ context.Context, opts ResolveOpts) (*ResolvedSession, error) {
	result := &ResolvedSession{
		Persistent:  opts.Persistent,
		SessionsDir: opts.SessionsDir,
	}

	// Determine sessions directory if not provided
	if result.SessionsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		result.SessionsDir = filepath.Join(home, ".clincus", opts.Tool.SessionsDirName())
	}

	// Handle resume
	if opts.Resume {
		result.IsResume = true
		result.ResumeID = opts.ResumeID
	}

	// Allocate slot
	maxSlots := opts.MaxSlots
	if maxSlots == 0 {
		maxSlots = 10
	}
	if opts.Slot > 0 {
		available, err := IsSlotAvailable(opts.Workspace, opts.Slot)
		if err != nil {
			return nil, fmt.Errorf("slot check failed: %w", err)
		}
		if !available {
			// Find next available slot starting from slot+1
			nextSlot, err := AllocateSlotFrom(opts.Workspace, opts.Slot+1, maxSlots)
			if err != nil {
				return nil, fmt.Errorf("slot %d is occupied and no available slot found: %w", opts.Slot, err)
			}
			result.Slot = nextSlot
		} else {
			result.Slot = opts.Slot
		}
	} else {
		slot, err := AllocateSlot(opts.Workspace, maxSlots)
		if err != nil {
			return nil, fmt.Errorf("slot allocation failed: %w", err)
		}
		result.Slot = slot
	}

	// Generate container name
	result.ContainerName = ContainerName(opts.Workspace, result.Slot)

	// Generate or reuse session ID
	if result.IsResume && result.ResumeID != "" {
		result.SessionID = result.ResumeID
	} else {
		id, err := GenerateSessionID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate session ID: %w", err)
		}
		result.SessionID = id
	}

	return result, nil
}
