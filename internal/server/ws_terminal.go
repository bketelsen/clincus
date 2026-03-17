package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bketelsen/clincus/internal/container"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
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

	tmuxSession := fmt.Sprintf("clincus-%s", containerID)

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	bridge, err := NewBridge(ws, containerID, tmuxSession, codeUID)
	if err != nil {
		//nolint:errcheck // best-effort error notification to client
		_ = ws.WriteJSON(WSMessage{Type: "error", Msg: err.Error()})
		return
	}
	defer bridge.Close()

	bridge.Run()
}
