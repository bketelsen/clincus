package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bketelsen/clincus/internal/config"
)

type Options struct {
	Port      int
	Assets    fs.FS
	AppConfig *config.Config
}

type Server struct {
	cfg    Options
	cfgMu  sync.RWMutex // protects cfg.AppConfig during hot-reload
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

// GetConfig returns the current AppConfig, safe for concurrent use.
func (s *Server) GetConfig() *config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.AppConfig
}

// UpdateConfig swaps the server's config reference. Returns the old port so
// the caller can detect whether an HTTP listener restart is needed (AC4).
func (s *Server) UpdateConfig(newCfg *config.Config) (oldPort int) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	oldPort = s.cfg.AppConfig.Dashboard.Port
	s.cfg.AppConfig = newCfg
	return oldPort
}

func (s *Server) Start() {
	s.StartIncusEventWatcher()
}

// BroadcastConfigReloaded sends a config.reloaded event to all connected
// WebSocket clients. Called by the config watcher onChange callback after a
// successful reload. Only successful reloads trigger this (AC5).
func (s *Server) BroadcastConfigReloaded() {
	s.events.Broadcast(map[string]any{
		"type":      "config.reloaded",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
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
	//nolint:errcheck // response write errors are non-actionable at this point
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	//nolint:errcheck // response write errors are non-actionable at this point
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clincus", "history.jsonl")
}
