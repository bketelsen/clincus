package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bketelsen/clincus/internal/container"
)

func (s *Server) handleShellWS(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		http.Error(w, "missing container id", 400)
		return
	}

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		http.Error(w, "container not running", 404)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	codeUID := s.codeUID()

	codeUser := container.CodeUser
	homeDir := fmt.Sprintf("/home/%s", codeUser)

	workspacePath := mgr.GetWorkspacePath()
	execArgs := []string{
		"exec", "--force-interactive",
		"--env", "TERM=xterm-256color",
		"--env", fmt.Sprintf("HOME=%s", homeDir),
		"--user", fmt.Sprintf("%d", codeUID),
		"--group", fmt.Sprintf("%d", codeUID),
		containerID, "--", "bash", "--login", "-c",
		fmt.Sprintf("cd %s && exec bash --login", workspacePath),
	}

	bridge, err := NewBridge(ws, containerID, execArgs)
	if err != nil {
		//nolint:errcheck // best-effort error notification to client
		_ = ws.WriteJSON(WSMessage{Type: "error", Msg: err.Error()})
		return
	}
	defer bridge.Close()

	bridge.Run()
}
