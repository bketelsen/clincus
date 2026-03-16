package server

import (
	"io/fs"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/mensfeld/code-on-incus/internal/config"
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
