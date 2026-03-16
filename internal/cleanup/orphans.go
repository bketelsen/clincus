package cleanup

// OrphanedResources holds information about orphaned system resources
type OrphanedResources struct {
	// Reserved for future orphan types (e.g., stopped containers)
}

// DetectAll detects all orphaned resources
func DetectAll() (*OrphanedResources, error) {
	return &OrphanedResources{}, nil
}

// CleanupAll cleans up all orphaned resources
func CleanupAll(_ func(string)) error {
	return nil
}

// HasOrphans returns true if there are any orphaned resources
func HasOrphans() bool {
	return false
}
