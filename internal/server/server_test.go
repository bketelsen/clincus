package server

import (
	"io/fs"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/bketelsen/clincus/internal/config"
	"github.com/gorilla/websocket"
)

func testConfig() *config.Config {
	return config.GetDefaultConfig()
}

func TestServerServesIndex(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>test</html>")},
	}

	srv := New(Options{
		Port:      0,
		Assets:    fs.FS(mockFS),
		AppConfig: testConfig(),
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "<html>test</html>" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestAPIReturnsJSON(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte("<html/>")},
	}

	srv := New(Options{
		Port:      0,
		Assets:    fs.FS(mockFS),
		AppConfig: testConfig(),
	})

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestBroadcastConfigReloaded(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte("<html/>")},
	}

	srv := New(Options{
		Port:      0,
		Assets:    fs.FS(mockFS),
		AppConfig: testConfig(),
	})

	// Start an httptest server so we can connect via WebSocket.
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Connect a WebSocket client to /ws/events.
	wsURL := "ws" + ts.URL[len("http"):] + "/ws/events"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect ws: %v", err)
	}
	defer ws.Close()

	// Broadcast a config.reloaded event.
	srv.BroadcastConfigReloaded()

	// Read the message with a timeout.
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg map[string]any
	if err := ws.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read ws message: %v", err)
	}

	// Verify event type.
	if msg["type"] != "config.reloaded" {
		t.Errorf("expected type config.reloaded, got %v", msg["type"])
	}

	// AC2: Verify timestamp is present.
	if _, ok := msg["timestamp"]; !ok {
		t.Error("expected timestamp field in config.reloaded event")
	}
}
