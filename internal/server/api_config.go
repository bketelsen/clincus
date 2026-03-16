package server

import (
	"encoding/json"
	"net/http"

	"github.com/bketelsen/clincus/internal/tool"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]any{
		"port":            s.cfg.AppConfig.Dashboard.Port,
		"workspace_roots": s.cfg.AppConfig.Dashboard.WorkspaceRoots,
	})
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
	if req.Port != nil {
		s.cfg.AppConfig.Dashboard.Port = *req.Port
	}
	if req.WorkspaceRoots != nil {
		s.cfg.AppConfig.Dashboard.WorkspaceRoots = *req.WorkspaceRoots
	}
	s.writeJSON(w, map[string]string{"status": "updated"})
}

func (s *Server) handleGetTools(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, tool.ListSupported())
}
