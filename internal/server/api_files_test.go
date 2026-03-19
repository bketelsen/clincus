package server

import (
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"root", "/", false},
		{"simple file", "src/main.go", false},
		{"nested dir", "src/pkg/util", false},
		{"dotfile", ".gitignore", false},
		{"dot-dot traversal", "../etc/passwd", true},
		{"embedded dot-dot", "src/../../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"empty", "", false},
		{"dot", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateFilePath(tt.path, "/workspace")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
