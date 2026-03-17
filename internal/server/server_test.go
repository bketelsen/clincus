package server

import (
	"encoding/json"
	"io/fs"
	"net/http/httptest"
	"strings"
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

func TestGetConfigReturnsAllSections(t *testing.T) {
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
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// AC1: All top-level config sections must be present
	requiredSections := []string{
		"defaults", "paths", "incus", "tool", "mounts",
		"limits", "git", "security", "profiles", "dashboard",
	}
	for _, section := range requiredSections {
		if _, ok := result[section]; !ok {
			t.Errorf("missing required config section: %s", section)
		}
	}
}

func TestGetConfigUsesSnakeCaseKeys(t *testing.T) {
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

	body := w.Body.String()

	// AC2: Verify snake_case keys are used (not PascalCase)
	snakeCaseKeys := []string{
		"workspace_roots", "sessions_dir", "storage_dir", "logs_dir",
		"code_uid", "code_user", "disable_shift", "effort_level",
		"writable_hooks", "protected_paths", "max_duration", "max_processes",
		"auto_stop", "stop_graceful", "tmpfs_size",
	}
	for _, key := range snakeCaseKeys {
		if !strings.Contains(body, `"`+key+`"`) {
			t.Errorf("expected snake_case key %q in response", key)
		}
	}
}

func TestGetConfigReturnsMergedDefaults(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte("<html/>")},
	}

	cfg := testConfig()
	srv := New(Options{
		Port:      0,
		Assets:    fs.FS(mockFS),
		AppConfig: cfg,
	})

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// AC4: Verify merged config values (defaults applied)
	var result map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check defaults section has expected values
	var defaults struct {
		Image string `json:"image"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(result["defaults"], &defaults); err != nil {
		t.Fatalf("failed to decode defaults: %v", err)
	}
	if defaults.Image != "clincus" {
		t.Errorf("expected default image 'clincus', got %q", defaults.Image)
	}
	if defaults.Model != "claude-sonnet-4-5" {
		t.Errorf("expected default model 'claude-sonnet-4-5', got %q", defaults.Model)
	}

	// Check dashboard section
	var dashboard struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal(result["dashboard"], &dashboard); err != nil {
		t.Fatalf("failed to decode dashboard: %v", err)
	}
	if dashboard.Port != 3000 {
		t.Errorf("expected default dashboard port 3000, got %d", dashboard.Port)
	}
}

func TestPutConfigStillWorks(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte("<html/>")},
	}

	srv := New(Options{
		Port:      0,
		Assets:    fs.FS(mockFS),
		AppConfig: testConfig(),
	})

	// AC3: PUT /api/config for workspace_roots continues to work
	body := strings.NewReader(`{"workspace_roots": ["/home/test/projects"]}`)
	req := httptest.NewRequest("PUT", "/api/config", body)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "updated" {
		t.Errorf("expected status 'updated', got %q", result["status"])
	}

	// Verify the update was applied by reading config back
	req2 := httptest.NewRequest("GET", "/api/config", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)

	var cfg config.Config
	if err := json.NewDecoder(w2.Body).Decode(&cfg); err != nil {
		t.Fatalf("failed to decode config response: %v", err)
	}
	if len(cfg.Dashboard.WorkspaceRoots) != 1 || cfg.Dashboard.WorkspaceRoots[0] != "/home/test/projects" {
		t.Errorf("workspace_roots not updated correctly: %v", cfg.Dashboard.WorkspaceRoots)
	}
}
