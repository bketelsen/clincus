package server

import (
	"encoding/json"
	"net/http"

	"github.com/bketelsen/clincus/internal/tool"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, s.GetConfig())
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port           *int      `json:"port"`
		WorkspaceRoots *[]string `json:"workspace_roots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request", 400)
		return
	}
	// Clone the current config, apply mutations to the clone, then swap
	// atomically via UpdateConfig to avoid partial mutation during concurrent
	// hot-reloads.
	current := s.GetConfig()
	updated := *current
	updated.Dashboard = current.Dashboard
	if req.Port != nil {
		updated.Dashboard.Port = *req.Port
	}
	if req.WorkspaceRoots != nil {
		roots := make([]string, len(*req.WorkspaceRoots))
		copy(roots, *req.WorkspaceRoots)
		updated.Dashboard.WorkspaceRoots = roots
	}
	s.UpdateConfig(&updated)
	s.BroadcastConfigReloaded()
	s.writeJSON(w, map[string]string{"status": "updated"})
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, tool.ListSupported())
}
