package cleanup

import "os"

// IsOrphanedWorkspace returns true if the given workspace path no longer exists
// on the host filesystem, indicating the container is orphaned.
func IsOrphanedWorkspace(workspacePath string) bool {
	if workspacePath == "" {
		return false
	}
	_, err := os.Stat(workspacePath)
	return os.IsNotExist(err)
}
