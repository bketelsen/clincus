package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mensfeld/code-on-incus/internal/container"
	"github.com/mensfeld/code-on-incus/internal/session"
	"github.com/mensfeld/code-on-incus/internal/tool"
)

type SessionInfo struct {
	ID         string `json:"id"`
	Workspace  string `json:"workspace"`
	Tool       string `json:"tool"`
	Slot       int    `json:"slot"`
	Status     string `json:"status"`
	Persistent bool   `json:"persistent"`
}

type CreateSessionRequest struct {
	Workspace  string `json:"workspace"`
	Tool       string `json:"tool"`
	Persistent bool   `json:"persistent"`
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	prefix := session.GetContainerPrefix()
	output, err := container.IncusOutput("list", "--format=json")
	if err != nil {
		s.writeError(w, "failed to list containers", 500)
		return
	}

	var containers []struct {
		Name   string            `json:"name"`
		Status string            `json:"status"`
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal([]byte(output), &containers); err != nil {
		s.writeError(w, "failed to parse container list", 500)
		return
	}

	var sessions []SessionInfo
	for _, c := range containers {
		if !strings.HasPrefix(c.Name, prefix) {
			continue
		}
		si := SessionInfo{
			ID:     c.Name,
			Status: strings.ToLower(c.Status),
		}
		if ws, ok := c.Config["user.clincus.workspace"]; ok {
			si.Workspace = ws
		}
		if t, ok := c.Config["user.clincus.tool"]; ok {
			si.Tool = t
		}
		if p, ok := c.Config["user.clincus.persistent"]; ok {
			si.Persistent = p == "true"
		}
		if _, slot, err := session.ParseContainerName(c.Name); err == nil {
			si.Slot = slot
		}
		sessions = append(sessions, si)
	}

	if sessions == nil {
		sessions = []SessionInfo{}
	}
	s.writeJSON(w, sessions)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request body", 400)
		return
	}

	if req.Workspace == "" {
		s.writeError(w, "workspace is required", 400)
		return
	}
	if req.Tool == "" {
		req.Tool = "claude"
	}

	t, err := tool.Get(req.Tool)
	if err != nil {
		s.writeError(w, fmt.Sprintf("unknown tool: %s", req.Tool), 400)
		return
	}

	resolved, err := session.Resolve(context.Background(), session.ResolveOpts{
		Workspace:  req.Workspace,
		Tool:       t,
		Persistent: req.Persistent,
		MaxSlots:   10,
	})
	if err != nil {
		s.writeError(w, fmt.Sprintf("session resolve failed: %v", err), 500)
		return
	}

	// Determine CLI config path for credential injection (same logic as shell.go)
	homeDir, _ := os.UserHomeDir()
	var cliConfigPath string
	if twh, ok := t.(tool.ToolWithHomeConfigFile); ok {
		cliConfigPath = filepath.Join(homeDir, twh.HomeConfigFileName())
	} else if configDirName := t.ConfigDirName(); configDirName != "" {
		cliConfigPath = filepath.Join(homeDir, configDirName)
	}

	setupOpts := session.SetupOptions{
		WorkspacePath: req.Workspace,
		Persistent:    resolved.Persistent,
		Slot:          resolved.Slot,
		Tool:          t,
		SessionsDir:   resolved.SessionsDir,
		CLIConfigPath: cliConfigPath,
	}

	if s.cfg.AppConfig != nil {
		setupOpts.Image = s.cfg.AppConfig.Defaults.Image
		setupOpts.IncusProject = s.cfg.AppConfig.Incus.Project
		setupOpts.ProtectedPaths = s.cfg.AppConfig.Security.GetEffectiveProtectedPaths()
		setupOpts.LimitsConfig = &s.cfg.AppConfig.Limits
		setupOpts.PreserveWorkspacePath = s.cfg.AppConfig.Paths.PreserveWorkspacePath
	}

	result, err := session.Setup(setupOpts)
	if err != nil {
		s.writeError(w, fmt.Sprintf("session setup failed: %v", err), 500)
		return
	}

	tmuxName := fmt.Sprintf("clincus-%s", resolved.ContainerName)
	toolCmd := t.BuildCommand(resolved.SessionID, false, "")
	cmdStr := fmt.Sprintf("tmux new-session -d -s %s %s", tmuxName, strings.Join(toolCmd, " "))
	codeUID := 1000
	if s.cfg.AppConfig != nil && s.cfg.AppConfig.Incus.CodeUID != 0 {
		codeUID = s.cfg.AppConfig.Incus.CodeUID
	}
	execOpts := container.ExecCommandOptions{
		User: &codeUID,
		Cwd:  result.ContainerWorkspacePath,
	}
	if _, err := result.Manager.ExecCommand(cmdStr, execOpts); err != nil {
		s.writeError(w, fmt.Sprintf("failed to start tool in container: %v", err), 500)
		return
	}

	s.writeJSON(w, SessionInfo{
		ID:         resolved.ContainerName,
		Workspace:  req.Workspace,
		Tool:       req.Tool,
		Slot:       resolved.Slot,
		Status:     "running",
		Persistent: resolved.Persistent,
	})
}

func (s *Server) handleStopSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"

	mgr := container.NewManager(id)
	exists, err := mgr.Exists()
	if err != nil || !exists {
		s.writeError(w, "session not found", 404)
		return
	}

	if force {
		err = mgr.Delete(true)
	} else {
		err = mgr.Stop(false)
	}
	if err != nil {
		s.writeError(w, fmt.Sprintf("failed to stop: %v", err), 500)
		return
	}

	w.WriteHeader(204)
}

func (s *Server) handleResumeSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	mgr := container.NewManager(id)
	exists, err := mgr.Exists()
	if err != nil || !exists {
		s.writeError(w, "session not found", 404)
		return
	}

	if err := mgr.Start(); err != nil {
		s.writeError(w, fmt.Sprintf("failed to start: %v", err), 500)
		return
	}

	toolName := "claude"
	if cfg, err := mgr.GetConfig("user.clincus.tool"); err == nil && cfg != "" {
		toolName = strings.TrimSpace(cfg)
	}
	t, err := tool.Get(toolName)
	if err != nil {
		s.writeError(w, fmt.Sprintf("unknown tool: %s", toolName), 500)
		return
	}

	tmuxName := fmt.Sprintf("clincus-%s", id)
	toolCmd := t.BuildCommand("", true, "")
	cmdStr := fmt.Sprintf("tmux new-session -d -s %s %s", tmuxName, strings.Join(toolCmd, " "))
	if _, err := mgr.ExecCommand(cmdStr, container.ExecCommandOptions{}); err != nil {
		s.writeError(w, fmt.Sprintf("failed to start tool: %v", err), 500)
		return
	}

	s.writeJSON(w, map[string]string{"id": id, "status": "running"})
}

func (s *Server) handleSessionHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	h := &session.History{Path: s.historyPath()}
	entries, err := h.ListHistory(limit, offset)
	if err != nil {
		s.writeError(w, "failed to read history", 500)
		return
	}
	if entries == nil {
		entries = []session.HistoryEntry{}
	}
	s.writeJSON(w, entries)
}
