package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bketelsen/clincus/internal/container"
)

// validateFilePath cleans and validates a requested path, returning the full
// container path. Rejects traversal attempts and absolute paths.
func validateFilePath(reqPath, workspaceRoot string) (string, error) {
	if reqPath == "" || reqPath == "/" || reqPath == "." {
		return workspaceRoot, nil
	}

	// Reject absolute paths
	if filepath.IsAbs(reqPath) {
		return "", fmt.Errorf("absolute paths not allowed")
	}

	cleaned := filepath.Clean(reqPath)

	// Reject traversal after cleaning
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	full := filepath.Join(workspaceRoot, cleaned)

	// Double-check resolved path is under workspace root (use "/" suffix to
	// prevent prefix collision, e.g. /workspace vs /workspaceevil)
	if !strings.HasPrefix(full, workspaceRoot+"/") && full != workspaceRoot {
		return "", fmt.Errorf("path escapes workspace")
	}

	return full, nil
}

type fileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type listFilesResponse struct {
	Path    string      `json:"path"`
	Entries []fileEntry `json:"entries"`
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		s.writeError(w, "missing container id", 400)
		return
	}

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		s.writeError(w, "container not running", 404)
		return
	}

	reqPath := r.URL.Query().Get("path")
	workspacePath := mgr.GetWorkspacePath()

	fullPath, err := validateFilePath(reqPath, workspacePath)
	if err != nil {
		s.writeError(w, err.Error(), 400)
		return
	}

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	// Use find to list directory contents with tab-delimited output
	output, err := mgr.ExecArgsCapture(
		[]string{"find", fullPath, "-mindepth", "1", "-maxdepth", "1", "-printf", `%f\t%y\t%s\n`},
		container.ExecCommandOptions{User: &codeUID},
	)
	if err != nil {
		s.writeError(w, "failed to list directory", 500)
		return
	}

	var entries []fileEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		name := parts[0]
		typeChar := parts[1]
		var size int64
		_, _ = fmt.Sscanf(parts[2], "%d", &size) //nolint:errcheck // best-effort size parsing; defaults to 0

		var entryType string
		switch typeChar {
		case "d":
			entryType = "dir"
		case "l":
			entryType = "symlink"
		default:
			entryType = "file"
		}

		entries = append(entries, fileEntry{Name: name, Type: entryType, Size: size})
	}

	s.writeJSON(w, listFilesResponse{Path: reqPath, Entries: entries})
}

type readFileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

const maxFileSize = 5 * 1024 * 1024 // 5MB

func (s *Server) handleReadFile(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		s.writeError(w, "missing container id", 400)
		return
	}

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		s.writeError(w, "container not running", 404)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" || reqPath == "/" || reqPath == "." {
		s.writeError(w, "path required", 400)
		return
	}

	workspacePath := mgr.GetWorkspacePath()
	fullPath, err := validateFilePath(reqPath, workspacePath)
	if err != nil {
		s.writeError(w, err.Error(), 400)
		return
	}

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	// Pre-check file size via stat
	sizeOutput, err := mgr.ExecArgsCapture(
		[]string{"stat", "-c", "%s", fullPath},
		container.ExecCommandOptions{User: &codeUID},
	)
	if err != nil {
		s.writeError(w, "file not found", 404)
		return
	}

	var fileSize int64
	_, _ = fmt.Sscanf(strings.TrimSpace(sizeOutput), "%d", &fileSize) //nolint:errcheck // best-effort; 0 is safe default
	if fileSize > maxFileSize {
		s.writeError(w, "file too large to edit (max 5MB)", 413)
		return
	}

	// Read file content (cap at 5MB for safety)
	content, err := mgr.ExecArgsCapture(
		[]string{"head", "-c", fmt.Sprintf("%d", maxFileSize), fullPath},
		container.ExecCommandOptions{User: &codeUID},
	)
	if err != nil {
		s.writeError(w, "failed to read file", 500)
		return
	}

	// Binary detection: check first 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			s.writeError(w, "binary file, cannot display", 422)
			return
		}
	}

	s.writeJSON(w, readFileResponse{Path: reqPath, Content: content, Size: fileSize})
}

type writeFileRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		s.writeError(w, "missing container id", 400)
		return
	}

	// Enforce 5MB request body limit
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		s.writeError(w, "container not running", 404)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" || reqPath == "/" || reqPath == "." {
		s.writeError(w, "path required", 400)
		return
	}

	workspacePath := mgr.GetWorkspacePath()
	fullPath, err := validateFilePath(reqPath, workspacePath)
	if err != nil {
		s.writeError(w, err.Error(), 400)
		return
	}

	var req writeFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request body", 400)
		return
	}

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	// Write to temp file on host, push to container, then fix ownership.
	// Note: Chown uses -R which is a no-op for single files but harmless.
	tmpFile, err := writeTempFile(req.Content)
	if err != nil {
		s.writeError(w, "failed to write file", 500)
		return
	}
	defer removeTempFile(tmpFile)

	if err := mgr.PushFile(tmpFile, fullPath); err != nil {
		s.writeError(w, "failed to push file to container", 500)
		return
	}

	// Fix ownership to code user
	if err := mgr.Chown(fullPath, codeUID, codeUID); err != nil {
		s.writeError(w, "failed to set file ownership", 500)
		return
	}

	s.writeJSON(w, map[string]string{"status": "ok", "path": reqPath})
}

// writeTempFile writes content to a temporary file and returns its path.
func writeTempFile(content string) (string, error) {
	f, err := os.CreateTemp("", "clincus-edit-*")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name()) //nolint:gosec // path from os.CreateTemp, not user input
		return "", err
	}
	return f.Name(), nil
}

// removeTempFile removes a temporary file, ignoring errors.
func removeTempFile(path string) {
	os.Remove(path) //nolint:errcheck // best-effort cleanup
}
