package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bketelsen/clincus/internal/config"
)

type Options struct {
	Port      int
	Assets    fs.FS
	AppConfig *config.Config
}

type Server struct {
	cfg    Options
	mux    *http.ServeMux
	events *EventHub
}

func New(cfg Options) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux(), events: NewEventHub()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Start() {
	s.StartIncusEventWatcher()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /api/config", s.handleUpdateConfig)
	s.mux.HandleFunc("GET /api/tools", s.handleGetTools)
	s.mux.HandleFunc("GET /api/sessions", s.handleListSessions)
	s.mux.HandleFunc("GET /api/sessions/history", s.handleSessionHistory)
	s.mux.HandleFunc("POST /api/sessions", s.handleCreateSession)
	s.mux.HandleFunc("DELETE /api/sessions/{id}", s.handleStopSession)
	s.mux.HandleFunc("POST /api/sessions/{id}/resume", s.handleResumeSession)
	s.mux.HandleFunc("GET /api/workspaces", s.handleListWorkspaces)
	s.mux.HandleFunc("POST /api/workspaces", s.handleAddWorkspace)
	s.mux.HandleFunc("DELETE /api/workspaces", s.handleRemoveWorkspace)
	s.mux.HandleFunc("GET /ws/terminal/{id}", s.handleTerminalWS)
	s.mux.HandleFunc("GET /ws/events", s.handleEventsWS)

	if s.cfg.Assets != nil {
		fileServer := http.FileServer(http.FS(s.cfg.Assets))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Strip leading slash for fs.Stat (embed.FS uses relative paths)
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			// If file exists in assets, serve it; otherwise serve index.html (SPA fallback)
			if _, err := fs.Stat(s.cfg.Assets, path); err != nil {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clincus", "history.jsonl")
}
