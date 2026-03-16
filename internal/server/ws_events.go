package server

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mensfeld/code-on-incus/internal/session"
)

type EventHub struct {
	mu      sync.Mutex
	clients map[*eventClient]struct{}
}

type eventClient struct {
	ws   *websocket.Conn
	send chan any
}

func NewEventHub() *EventHub {
	return &EventHub{clients: make(map[*eventClient]struct{})}
}

func (h *EventHub) Add(ws *websocket.Conn) *eventClient {
	c := &eventClient{ws: ws, send: make(chan any, 32)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	go func() {
		for msg := range c.send {
			if err := c.ws.WriteJSON(msg); err != nil {
				return
			}
		}
	}()
	return c
}

func (h *EventHub) Remove(c *eventClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
}

func (h *EventHub) Broadcast(msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
		}
	}
}

func (s *Server) StartIncusEventWatcher() {
	go func() {
		for {
			if err := s.watchIncusEvents(); err != nil {
				log.Printf("incus event watcher error: %v, restarting...", err)
				time.Sleep(2 * time.Second)
			}
		}
	}()
}

func (s *Server) watchIncusEvents() error {
	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		cmd = exec.Command("sg", "incus-admin", "-c", "incus monitor --type lifecycle --format json")
	} else {
		cmd = exec.Command("incus", "monitor", "--type", "lifecycle", "--format", "json")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()

	prefix := session.GetContainerPrefix()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var evt struct {
			Type     string `json:"type"`
			Metadata struct {
				Action string `json:"action"`
				Source string `json:"source"`
				Name   string `json:"name"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			continue
		}

		name := evt.Metadata.Name
		if name == "" {
			parts := strings.Split(evt.Metadata.Source, "/")
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}

		if !strings.HasPrefix(name, prefix) {
			continue
		}

		switch evt.Metadata.Action {
		case "instance-started":
			s.events.Broadcast(map[string]string{
				"type": "session.started",
				"id":   name,
			})
		case "instance-stopped", "instance-deleted":
			s.events.Broadcast(map[string]string{
				"type": "session.stopped",
				"id":   name,
			})
		}
	}

	return cmd.Wait()
}

func (s *Server) handleEventsWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("events ws upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	client := s.events.Add(ws)
	defer s.events.Remove(client)

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}
}
