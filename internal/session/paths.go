package session

import (
	"path/filepath"

	"github.com/bketelsen/clincus/internal/tool"
)

// GetSessionsDir returns the sessions directory path for a given tool.
// For example: ~/.clincus/sessions-claude, ~/.clincus/sessions-aider, etc.
func GetSessionsDir(baseDir string, t tool.Tool) string {
	return filepath.Join(baseDir, t.SessionsDirName())
}
