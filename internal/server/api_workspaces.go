package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mensfeld/code-on-incus/internal/config"
)

type WorkspaceInfo struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	HasConfig      bool   `json:"has_config"`
	ActiveSessions int    `json:"active_sessions"`
}

type WorkspacesResponse struct {
	Roots      []string        `json:"roots"`
	Workspaces []WorkspaceInfo `json:"workspaces"`
}

var projectMarkers = []string{
	".git", "go.mod", "package.json", "Cargo.toml",
	"pyproject.toml", ".coi.toml", "Makefile",
	"pom.xml", "build.gradle",
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	roots := s.cfg.AppConfig.Dashboard.WorkspaceRoots
	var workspaces []WorkspaceInfo

	for _, root := range roots {
		expanded := config.ExpandPath(root)
		entries, err := os.ReadDir(expanded)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			fullPath := filepath.Join(expanded, e.Name())
			if !isProject(fullPath) {
				continue
			}
			_, coiErr := os.Stat(filepath.Join(fullPath, ".coi.toml"))
			workspaces = append(workspaces, WorkspaceInfo{
				Path:      fullPath,
				Name:      e.Name(),
				HasConfig: coiErr == nil,
			})
		}
	}

	if workspaces == nil {
		workspaces = []WorkspaceInfo{}
	}
	s.writeJSON(w, WorkspacesResponse{
		Roots:      roots,
		Workspaces: workspaces,
	})
}

func (s *Server) handleAddWorkspace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request", 400)
		return
	}

	expanded := config.ExpandPath(req.Path)
	info, err := os.Stat(expanded)
	if err != nil || !info.IsDir() {
		s.writeError(w, "path is not a valid directory", 400)
		return
	}

	roots := s.cfg.AppConfig.Dashboard.WorkspaceRoots
	for _, r := range roots {
		if config.ExpandPath(r) == expanded {
			s.writeError(w, "already registered", 409)
			return
		}
	}

	s.cfg.AppConfig.Dashboard.WorkspaceRoots = append(roots, req.Path)
	w.WriteHeader(201)
	s.writeJSON(w, map[string]string{"status": "added"})
}

func (s *Server) handleRemoveWorkspace(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		s.writeError(w, "path query parameter required", 400)
		return
	}
	roots := s.cfg.AppConfig.Dashboard.WorkspaceRoots
	var newRoots []string
	found := false
	for _, root := range roots {
		if config.ExpandPath(root) == config.ExpandPath(path) {
			found = true
			continue
		}
		newRoots = append(newRoots, root)
	}
	if !found {
		s.writeError(w, "workspace root not found", 404)
		return
	}
	s.cfg.AppConfig.Dashboard.WorkspaceRoots = newRoots
	w.WriteHeader(204)
}

func isProject(dir string) bool {
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}
