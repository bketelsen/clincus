# Session Tabbed Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a tabbed workspace UI to the session view with Session (tmux), Shell (bash), and Editor (Monaco) tabs.

**Architecture:** Replace `Terminal.svelte` with `SessionView.svelte` that renders a session header, tab bar, and lazy-loaded panes. Backend adds a `/ws/shell/{id}` WebSocket endpoint (reusing a refactored Bridge) and REST endpoints for file operations. Monaco editor provides read-write file editing in the container workspace.

**Tech Stack:** Go (WebSocket + REST handlers), Svelte 5, xterm.js, Monaco Editor, Vite 6

**Spec:** `docs/superpowers/specs/2026-03-19-session-tabbed-workspace-design.md`

---

## File Structure

### New Files

| File | Responsibility |
| ---- | -------------- |
| `internal/server/ws_shell.go` | Shell WebSocket handler (mirrors ws_terminal.go) |
| `internal/server/api_files.go` | File listing, read, write REST endpoints |
| `internal/server/api_files_test.go` | Tests for file API path validation and response parsing |
| `web/src/routes/SessionView.svelte` | Tab container with header, tab bar, lazy pane mounting |
| `web/src/components/SessionHeader.svelte` | Session metadata display (name, tool, status, path) |
| `web/src/components/TabBar.svelte` | Tab navigation (Session, Shell, Editor) |
| `web/src/components/ShellPane.svelte` | Independent bash terminal via xterm.js |
| `web/src/components/EditorPane.svelte` | Editor layout (file tree + Monaco) |
| `web/src/components/FileTree.svelte` | Workspace file tree with lazy folder expansion |
| `web/src/components/MonacoEditor.svelte` | Monaco editor wrapper with save support |

### Modified Files

| File | Change |
| ---- | ------ |
| `internal/server/bridge.go` | Refactor `NewBridge` to accept `execArgs []string` instead of `tmuxSession string` |
| `internal/server/ws_terminal.go` | Update `NewBridge` call to pass tmux command as slice |
| `internal/server/server.go` | Register 4 new routes |
| `web/src/App.svelte` | Import `SessionView` instead of `Terminal` |
| `web/src/lib/ws.ts` | Refactor `connectTerminal` to accept URL path parameter |
| `web/src/lib/api.ts` | Add `put` helper and file API functions |
| `web/src/lib/types.ts` | Add file API response types |
| `web/package.json` | Add `monaco-editor` dependency |
| `web/vite.config.ts` | Configure Monaco worker loading |

### Removed Files

| File | Reason |
| ---- | ------ |
| `web/src/routes/Terminal.svelte` | Replaced by `SessionView.svelte` |

---

## Task 1: Refactor Bridge to Accept Generic Command

**Files:**
- Modify: `internal/server/bridge.go:32-49`
- Modify: `internal/server/ws_terminal.go:37-45`

- [ ] **Step 1: Update `NewBridge` signature**

Change `bridge.go` line 32 from:

```go
func NewBridge(ws *websocket.Conn, containerName, tmuxSession string, uid int) (*Bridge, error) {
```

to:

```go
func NewBridge(ws *websocket.Conn, containerName string, execArgs []string, uid int) (*Bridge, error) {
```

Then replace lines 33-38 (the `incusArgs` construction) with:

```go
	incusArgs := execArgs
```

Everything else in `NewBridge` stays the same -- the `sg` wrapping, `pty.Start`, and return are command-agnostic.

- [ ] **Step 2: Update `handleTerminalWS` to pass tmux command as slice**

In `ws_terminal.go`, replace lines 37-45:

```go
	tmuxSession := fmt.Sprintf("clincus-%s", containerID)

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	bridge, err := NewBridge(ws, containerID, tmuxSession, codeUID)
```

with:

```go
	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	tmuxSession := fmt.Sprintf("clincus-%s", containerID)
	execArgs := []string{
		"exec", "--force-interactive",
		"--env", "TERM=xterm-256color",
		"--user", fmt.Sprintf("%d", codeUID),
		"--group", fmt.Sprintf("%d", codeUID),
		containerID, "--", "tmux", "attach-session", "-t", tmuxSession,
	}

	bridge, err := NewBridge(ws, containerID, execArgs, codeUID)
```

- [ ] **Step 3: Verify existing terminal still works**

Run: `make build`
Expected: Compiles with no errors.

Run: `make test`
Expected: All existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/server/bridge.go internal/server/ws_terminal.go
git commit -m "refactor: generalize NewBridge to accept arbitrary exec args"
```

---

## Task 2: Add Shell WebSocket Handler

**Files:**
- Create: `internal/server/ws_shell.go`
- Modify: `internal/server/server.go:82`

- [ ] **Step 1: Create `ws_shell.go`**

Create `internal/server/ws_shell.go`:

```go
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

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	workspacePath := mgr.GetWorkspacePath()
	execArgs := []string{
		"exec", "--force-interactive",
		"--env", "TERM=xterm-256color",
		"--user", fmt.Sprintf("%d", codeUID),
		"--group", fmt.Sprintf("%d", codeUID),
		containerID, "--", "bash", "-c",
		fmt.Sprintf("cd %s && exec bash", workspacePath),
	}

	bridge, err := NewBridge(ws, containerID, execArgs, codeUID)
	if err != nil {
		//nolint:errcheck // best-effort error notification to client
		_ = ws.WriteJSON(WSMessage{Type: "error", Msg: err.Error()})
		return
	}
	defer bridge.Close()

	bridge.Run()
}
```

- [ ] **Step 2: Register the route**

In `server.go`, add after line 82 (`GET /ws/terminal/{id}`):

```go
	s.mux.HandleFunc("GET /ws/shell/{id}", s.handleShellWS)
```

- [ ] **Step 3: Build and test**

Run: `make build`
Expected: Compiles with no errors.

Run: `make test`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/server/ws_shell.go internal/server/server.go
git commit -m "feat: add shell WebSocket endpoint for independent bash sessions"
```

---

## Task 3: Add File REST API

**Files:**
- Create: `internal/server/api_files.go`
- Create: `internal/server/api_files_test.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Write path validation tests**

Create `internal/server/api_files_test.go`:

```go
package server

import (
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"root", "/", false},
		{"simple file", "src/main.go", false},
		{"nested dir", "src/pkg/util", false},
		{"dotfile", ".gitignore", false},
		{"dot-dot traversal", "../etc/passwd", true},
		{"embedded dot-dot", "src/../../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"empty", "", false},
		{"dot", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateFilePath(tt.path, "/workspace")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestValidateFilePath -v`
Expected: FAIL -- `validateFilePath` not defined.

- [ ] **Step 3: Create `api_files.go` with path validation and all three handlers**

Create `internal/server/api_files.go`:

```go
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bketelsen/clincus/internal/container"
)

// validateFilePath cleans and validates a requested path, returning the full
// container path. Rejects traversal attempts and absolute paths.
func validateFilePath(reqPath, workspaceRoot string) (string, error) {
	if reqPath == "" || reqPath == "/" || reqPath == "." {
		return workspaceRoot, nil
	}

	// Reject absolute paths
	if filepath.IsAbs(reqPath) {
		return "", fmt.Errorf("absolute paths not allowed")
	}

	cleaned := filepath.Clean(reqPath)

	// Reject traversal after cleaning
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	full := filepath.Join(workspaceRoot, cleaned)

	// Double-check resolved path is under workspace root (use "/" suffix to
	// prevent prefix collision, e.g. /workspace vs /workspaceevil)
	if !strings.HasPrefix(full, workspaceRoot+"/") && full != workspaceRoot {
		return "", fmt.Errorf("path escapes workspace")
	}

	return full, nil
}

type fileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type listFilesResponse struct {
	Path    string      `json:"path"`
	Entries []fileEntry `json:"entries"`
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		s.writeError(w, "missing container id", 400)
		return
	}

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		s.writeError(w, "container not running", 404)
		return
	}

	reqPath := r.URL.Query().Get("path")
	workspacePath := mgr.GetWorkspacePath()

	fullPath, err := validateFilePath(reqPath, workspacePath)
	if err != nil {
		s.writeError(w, err.Error(), 400)
		return
	}

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	// Use find to list directory contents with tab-delimited output
	output, err := mgr.ExecArgsCapture(
		[]string{"find", fullPath, "-maxdepth", "1", "-not", "-name", ".", "-printf", `%f\t%y\t%s\n`},
		container.ExecCommandOptions{User: &codeUID},
	)
	if err != nil {
		s.writeError(w, "failed to list directory", 500)
		return
	}

	var entries []fileEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		name := parts[0]
		typeChar := parts[1]
		var size int64
		fmt.Sscanf(parts[2], "%d", &size)

		entryType := "file"
		if typeChar == "d" {
			entryType = "dir"
		} else if typeChar == "l" {
			entryType = "symlink"
		}

		entries = append(entries, fileEntry{Name: name, Type: entryType, Size: size})
	}

	s.writeJSON(w, listFilesResponse{Path: reqPath, Entries: entries})
}

type readFileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

const maxFileSize = 5 * 1024 * 1024 // 5MB

func (s *Server) handleReadFile(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		s.writeError(w, "missing container id", 400)
		return
	}

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		s.writeError(w, "container not running", 404)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" || reqPath == "/" || reqPath == "." {
		s.writeError(w, "path required", 400)
		return
	}

	workspacePath := mgr.GetWorkspacePath()
	fullPath, err := validateFilePath(reqPath, workspacePath)
	if err != nil {
		s.writeError(w, err.Error(), 400)
		return
	}

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	// Pre-check file size via stat
	sizeOutput, err := mgr.ExecArgsCapture(
		[]string{"stat", "-c", "%s", fullPath},
		container.ExecCommandOptions{User: &codeUID},
	)
	if err != nil {
		s.writeError(w, "file not found", 404)
		return
	}

	var fileSize int64
	fmt.Sscanf(strings.TrimSpace(sizeOutput), "%d", &fileSize)
	if fileSize > maxFileSize {
		s.writeError(w, "file too large to edit (max 5MB)", 413)
		return
	}

	// Read file content (cap at 5MB for safety)
	content, err := mgr.ExecArgsCapture(
		[]string{"head", "-c", fmt.Sprintf("%d", maxFileSize), fullPath},
		container.ExecCommandOptions{User: &codeUID},
	)
	if err != nil {
		s.writeError(w, "failed to read file", 500)
		return
	}

	// Binary detection: check first 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			s.writeError(w, "binary file, cannot display", 422)
			return
		}
	}

	s.writeJSON(w, readFileResponse{Path: reqPath, Content: content, Size: fileSize})
}

type writeFileRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		s.writeError(w, "missing container id", 400)
		return
	}

	// Enforce 5MB request body limit
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	mgr := container.NewManager(containerID)
	running, err := mgr.Running()
	if err != nil || !running {
		s.writeError(w, "container not running", 404)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" || reqPath == "/" || reqPath == "." {
		s.writeError(w, "path required", 400)
		return
	}

	workspacePath := mgr.GetWorkspacePath()
	fullPath, err := validateFilePath(reqPath, workspacePath)
	if err != nil {
		s.writeError(w, err.Error(), 400)
		return
	}

	var req writeFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request body", 400)
		return
	}

	codeUID := 1000
	appCfg := s.GetConfig()
	if appCfg != nil && appCfg.Incus.CodeUID != 0 {
		codeUID = appCfg.Incus.CodeUID
	}

	// Write to temp file on host, push to container, then fix ownership.
	// Note: Chown uses -R which is a no-op for single files but harmless.
	tmpFile, err := writeTempFile(req.Content)
	if err != nil {
		s.writeError(w, "failed to write file", 500)
		return
	}
	defer removeTempFile(tmpFile)

	if err := mgr.PushFile(tmpFile, fullPath); err != nil {
		s.writeError(w, "failed to push file to container", 500)
		return
	}

	// Fix ownership to code user
	if err := mgr.Chown(fullPath, codeUID, codeUID); err != nil {
		s.writeError(w, "failed to set file ownership", 500)
		return
	}

	s.writeJSON(w, map[string]string{"status": "ok", "path": reqPath})
}

// writeTempFile writes content to a temporary file and returns its path.
func writeTempFile(content string) (string, error) {
	f, err := os.CreateTemp("", "clincus-edit-*")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// removeTempFile removes a temporary file, ignoring errors.
func removeTempFile(path string) {
	os.Remove(path) //nolint:errcheck // best-effort cleanup
}
```

- [ ] **Step 4: Run path validation tests**

Run: `go test ./internal/server/ -run TestValidateFilePath -v`
Expected: All cases pass.

- [ ] **Step 5: Register routes in `server.go`**

Add after the shell WebSocket route:

```go
	s.mux.HandleFunc("GET /api/sessions/{id}/files", s.handleListFiles)
	s.mux.HandleFunc("GET /api/sessions/{id}/files/content", s.handleReadFile)
	s.mux.HandleFunc("PUT /api/sessions/{id}/files/content", s.handleWriteFile)
```

- [ ] **Step 6: Build and test**

Run: `make build && make test`
Expected: Compiles. All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/server/api_files.go internal/server/api_files_test.go internal/server/server.go
git commit -m "feat: add file listing, read, and write REST endpoints"
```

---

## Task 4: Add Frontend Types and API Functions

**Files:**
- Modify: `web/src/lib/types.ts`
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/lib/ws.ts`

- [ ] **Step 1: Add file API types to `types.ts`**

Append to the end of `web/src/lib/types.ts`:

```typescript
export interface FileEntry {
  name: string;
  type: 'file' | 'dir' | 'symlink';
  size: number;
}

export interface FileListResponse {
  path: string;
  entries: FileEntry[];
}

export interface FileContentResponse {
  path: string;
  content: string;
  size: number;
}
```

- [ ] **Step 2: Add `put` helper and file API functions to `api.ts`**

Add the `put` helper after the existing `del` function (around line 85):

```typescript
async function put<T>(path: string, body?: unknown, opts?: RequestOptions): Promise<T> {
    const res = await fetchWithRetry(
        `${BASE}${path}`,
        {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: body ? JSON.stringify(body) : undefined,
        },
        opts,
    );
    return res.json();
}
```

Add the file types to the import at the top of `api.ts`:

```typescript
import type { Session, Workspace, HistoryEntry, ClincusConfig, FileListResponse, FileContentResponse } from './types';
```

Add to the `api` export object:

```typescript
    listFiles: (sessionId: string, path = '/', opts?: RequestOptions) =>
        get<FileListResponse>(`/api/sessions/${sessionId}/files?path=${encodeURIComponent(path)}`, opts),
    readFile: (sessionId: string, path: string, opts?: RequestOptions) =>
        get<FileContentResponse>(`/api/sessions/${sessionId}/files/content?path=${encodeURIComponent(path)}`, opts),
    writeFile: (sessionId: string, path: string, content: string, opts?: RequestOptions) =>
        put<{ status: string; path: string }>(`/api/sessions/${sessionId}/files/content?path=${encodeURIComponent(path)}`, { content }, opts),
```

- [ ] **Step 3: Refactor `connectTerminal` in `ws.ts` to accept URL path**

Replace the `connectTerminal` function (lines 3-37) with a generic `connectWS` and a thin wrapper:

```typescript
function connectWS(
  url: string,
  onOutput: (data: string) => void,
  onExit: (code: number) => void,
  onError: (msg: string) => void,
): { send: (msg: WSMessage) => void; close: () => void } {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${proto}//${location.host}${url}`);

  ws.onmessage = (evt) => {
    const msg: WSMessage = JSON.parse(evt.data);
    switch (msg.type) {
      case 'output':
        if (msg.data) onOutput(msg.data);
        break;
      case 'exit':
        onExit(msg.code ?? 0);
        break;
      case 'error':
        onError(msg.message ?? 'unknown error');
        break;
    }
  };

  ws.onerror = () => onError('WebSocket connection error');

  return {
    send: (msg) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(msg));
      }
    },
    close: () => ws.close(),
  };
}

export function connectTerminal(
  containerId: string,
  onOutput: (data: string) => void,
  onExit: (code: number) => void,
  onError: (msg: string) => void,
) {
  return connectWS(`/ws/terminal/${containerId}`, onOutput, onExit, onError);
}

export function connectShell(
  containerId: string,
  onOutput: (data: string) => void,
  onExit: (code: number) => void,
  onError: (msg: string) => void,
) {
  return connectWS(`/ws/shell/${containerId}`, onOutput, onExit, onError);
}
```

- [ ] **Step 4: Build frontend**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/types.ts web/src/lib/api.ts web/src/lib/ws.ts
git commit -m "feat: add file API types, put helper, and shell WebSocket connection"
```

---

## Task 5: Install Monaco and Configure Vite

**Files:**
- Modify: `web/package.json`
- Modify: `web/vite.config.ts`

- [ ] **Step 1: Install Monaco**

Run: `cd /home/bjk/projects/clincus/web && npm install monaco-editor`

- [ ] **Step 2: Configure Vite for Monaco workers**

Replace `web/vite.config.ts` with:

```typescript
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: '../webui/dist',
    emptyOutDir: true,
  },
});
```

No plugin needed -- Monaco ESM workers are configured directly in the component that initializes Monaco (Task 9). Vite 6 handles the worker imports natively via `import.meta.url`.

- [ ] **Step 3: Verify build**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully. Monaco chunks appear in `../webui/dist/assets/`.

- [ ] **Step 4: Commit**

```bash
git add web/package.json web/package-lock.json web/vite.config.ts
git commit -m "feat: add monaco-editor dependency"
```

---

## Task 6: Create SessionHeader and TabBar Components

**Files:**
- Create: `web/src/components/SessionHeader.svelte`
- Create: `web/src/components/TabBar.svelte`

- [ ] **Step 1: Create `SessionHeader.svelte`**

Create `web/src/components/SessionHeader.svelte`:

```svelte
<script lang="ts">
  import type { Session } from '../lib/types';
  let { session }: { session: Session } = $props();
</script>

<div class="session-header">
  <div class="session-info">
    <span class="session-name">{session.workspace.split('/').pop()}</span>
    <span class="separator">&middot;</span>
    <span class="session-tool">{session.tool}</span>
    <span class="status-badge" class:running={session.status === 'Running'}>
      {session.status.toLowerCase()}
    </span>
  </div>
  <div class="session-path">{session.workspace}</div>
</div>

<style>
  .session-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    background: #1a1a2e;
    border-bottom: 1px solid #222;
  }
  .session-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .session-name {
    color: #eee;
    font-size: 13px;
    font-weight: 600;
  }
  .separator { color: #555; }
  .session-tool {
    color: #888;
    font-size: 12px;
  }
  .status-badge {
    padding: 1px 6px;
    border-radius: 8px;
    font-size: 10px;
    background: #333;
    color: #888;
  }
  .status-badge.running {
    background: #2d4a2d;
    color: #6d6;
  }
  .session-path {
    color: #666;
    font-size: 11px;
  }
</style>
```

- [ ] **Step 2: Create `TabBar.svelte`**

Create `web/src/components/TabBar.svelte`:

```svelte
<script lang="ts">
  let { activeTab, onTabChange }: {
    activeTab: string;
    onTabChange: (tab: string) => void;
  } = $props();

  const tabs = [
    { id: 'session', label: 'Session' },
    { id: 'shell', label: 'Shell' },
    { id: 'editor', label: 'Editor' },
  ];
</script>

<div class="tab-bar">
  {#each tabs as tab}
    <button
      class="tab"
      class:active={activeTab === tab.id}
      onclick={() => onTabChange(tab.id)}
    >
      {tab.label}
    </button>
  {/each}
</div>

<style>
  .tab-bar {
    display: flex;
    align-items: center;
    padding: 0 12px;
    gap: 2px;
    height: 32px;
    background: #1a1a2e;
    border-bottom: 1px solid #333;
  }
  .tab {
    padding: 5px 14px;
    color: #888;
    font-size: 12px;
    font-family: inherit;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s;
  }
  .tab:hover {
    color: #bbb;
  }
  .tab.active {
    color: #eee;
    border-bottom-color: #7c6fe0;
  }
</style>
```

- [ ] **Step 3: Build**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/SessionHeader.svelte web/src/components/TabBar.svelte
git commit -m "feat: add SessionHeader and TabBar components"
```

---

## Task 7: Create SessionView and Wire Up Routing

**Files:**
- Create: `web/src/routes/SessionView.svelte`
- Modify: `web/src/App.svelte`
- Remove: `web/src/routes/Terminal.svelte`

- [ ] **Step 1: Create `SessionView.svelte`**

Create `web/src/routes/SessionView.svelte`:

```svelte
<script lang="ts">
  import SessionHeader from '../components/SessionHeader.svelte';
  import TabBar from '../components/TabBar.svelte';
  import TerminalPane from '../components/TerminalPane.svelte';
  import { getSessions } from '../stores/sessions.svelte';
  import type { Session } from '../lib/types';

  let { containerId }: { containerId: string } = $props();

  let activeTab = $state('session');
  let shellInitialized = $state(false);
  let editorInitialized = $state(false);

  function onTabChange(tab: string) {
    activeTab = tab;
    if (tab === 'shell') shellInitialized = true;
    if (tab === 'editor') editorInitialized = true;
  }

  let session = $derived(
    getSessions().find((s: Session) => s.id === containerId)
  );
</script>

<div class="session-view">
  {#if session}
    <SessionHeader {session} />
  {/if}
  <TabBar {activeTab} {onTabChange} />
  <div class="pane-container">
    <div class="pane" class:hidden={activeTab !== 'session'}>
      <TerminalPane {containerId} visible={activeTab === 'session'} />
    </div>
    {#if shellInitialized}
      <div class="pane" class:hidden={activeTab !== 'shell'}>
        <!-- ShellPane added in Task 8 -->
        <div style="padding: 20px; color: #888;">Shell pane (coming soon)</div>
      </div>
    {/if}
    {#if editorInitialized}
      <div class="pane" class:hidden={activeTab !== 'editor'}>
        <!-- EditorPane added in Task 11 -->
        <div style="padding: 20px; color: #888;">Editor pane (coming soon)</div>
      </div>
    {/if}
  </div>
</div>

<style>
  .session-view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .pane-container {
    flex: 1;
    position: relative;
    overflow: hidden;
  }
  .pane {
    position: absolute;
    inset: 0;
  }
  .pane.hidden {
    display: none;
  }
</style>
```

- [ ] **Step 2: Update `TerminalPane.svelte` to accept `visible` prop for resize**

Add the `visible` prop and trigger `fitAddon.fit()` when it changes. In `web/src/components/TerminalPane.svelte`, update the script section:

```svelte
<script lang="ts">
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { connectTerminal } from '../lib/ws';
  import { onMount } from 'svelte';

  let { containerId, visible = true }: { containerId: string; visible?: boolean } = $props();

  let termDiv: HTMLDivElement;
  let fitAddon: FitAddon;

  $effect(() => {
    if (visible && fitAddon) {
      // Small delay to allow DOM to update display before measuring
      setTimeout(() => fitAddon.fit(), 50);
    }
  });

  onMount(() => {
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      theme: { background: '#1a1a2e', foreground: '#eee' },
    });
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termDiv);
    fitAddon.fit();

    const conn = connectTerminal(
      containerId,
      (data) => term.write(data),
      (code) => term.write(`\r\n[Process exited with code ${code}]\r\n`),
      (msg) => term.write(`\r\n[Error: ${msg}]\r\n`),
    );

    term.onData((data) => conn.send({ type: 'input', data }));
    term.onResize(({ cols, rows }) => conn.send({ type: 'resize', cols, rows }));

    setTimeout(() => {
      fitAddon.fit();
      conn.send({ type: 'resize', cols: term.cols, rows: term.rows });
    }, 100);

    const onResize = () => fitAddon.fit();
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      conn.close();
      term.dispose();
    };
  });
</script>

<div class="terminal-container" bind:this={termDiv}></div>

<style>
  .terminal-container { width: 100%; height: 100%; }
  :global(.xterm) { height: 100%; }
</style>
```

- [ ] **Step 3: Update `App.svelte` routing**

In `web/src/App.svelte`, change the import from `Terminal` to `SessionView`:

Replace:
```typescript
  import Terminal from './routes/Terminal.svelte';
```
With:
```typescript
  import SessionView from './routes/SessionView.svelte';
```

Replace the route rendering (around line 66-67):
```svelte
    {:else if routeParam}
      <Terminal containerId={routeParam} />
```
With:
```svelte
    {:else if routeParam}
      <SessionView containerId={routeParam} />
```

- [ ] **Step 4: Delete `Terminal.svelte`**

Remove `web/src/routes/Terminal.svelte`.

- [ ] **Step 5: Add `getSessions` export to sessions store if not present**

Check `web/src/stores/sessions.svelte.ts` -- it should already export `getSessions`. If it exports it differently, adjust the import in `SessionView.svelte` to match.

- [ ] **Step 6: Build and verify**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully.

Run: `make build`
Expected: Full Go+frontend build succeeds.

- [ ] **Step 7: Commit**

```bash
git add web/src/routes/SessionView.svelte web/src/components/TerminalPane.svelte web/src/App.svelte
git rm web/src/routes/Terminal.svelte
git commit -m "feat: replace Terminal view with tabbed SessionView"
```

---

## Task 8: Create ShellPane Component

**Files:**
- Create: `web/src/components/ShellPane.svelte`
- Modify: `web/src/routes/SessionView.svelte`

- [ ] **Step 1: Create `ShellPane.svelte`**

Create `web/src/components/ShellPane.svelte`:

```svelte
<script lang="ts">
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { connectShell } from '../lib/ws';
  import { onMount } from 'svelte';

  let { containerId, visible = true }: { containerId: string; visible?: boolean } = $props();

  let termDiv: HTMLDivElement;
  let fitAddon: FitAddon;
  let exited = $state(false);
  let conn: ReturnType<typeof connectShell> | null = null;
  let term: Terminal | null = null;

  $effect(() => {
    if (visible && fitAddon) {
      setTimeout(() => fitAddon.fit(), 50);
    }
  });

  function initShell() {
    exited = false;
    term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      theme: { background: '#1a1a2e', foreground: '#eee' },
    });
    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termDiv);
    fitAddon.fit();

    conn = connectShell(
      containerId,
      (data) => term!.write(data),
      (_code) => {
        term!.write('\r\n[Shell exited - click to restart]\r\n');
        exited = true;
      },
      (msg) => term!.write(`\r\n[Error: ${msg}]\r\n`),
    );

    term.onData((data) => conn!.send({ type: 'input', data }));
    term.onResize(({ cols, rows }) => conn!.send({ type: 'resize', cols, rows }));

    setTimeout(() => {
      fitAddon.fit();
      conn!.send({ type: 'resize', cols: term!.cols, rows: term!.rows });
    }, 100);
  }

  function restart() {
    cleanup();
    initShell();
  }

  function cleanup() {
    if (conn) conn.close();
    if (term) term.dispose();
    conn = null;
    term = null;
  }

  onMount(() => {
    initShell();

    const onResize = () => {
      if (fitAddon) fitAddon.fit();
    };
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      cleanup();
    };
  });
</script>

{#if exited}
  <div class="exit-overlay">
    <button class="restart-btn" onclick={restart}>Restart Shell</button>
  </div>
{/if}
<div class="terminal-container" bind:this={termDiv}></div>

<style>
  .terminal-container { width: 100%; height: 100%; }
  :global(.xterm) { height: 100%; }
  .exit-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 10;
    display: flex;
    justify-content: center;
    padding: 12px;
    background: rgba(15, 15, 26, 0.9);
  }
  .restart-btn {
    padding: 6px 16px;
    background: #2a2a4a;
    color: #ccc;
    border: 1px solid #444;
    border-radius: 4px;
    cursor: pointer;
    font-family: inherit;
    font-size: 12px;
  }
  .restart-btn:hover {
    background: #3a3a5a;
    color: #eee;
  }
</style>
```

- [ ] **Step 2: Wire ShellPane into SessionView**

In `web/src/routes/SessionView.svelte`, add the import:

```typescript
  import ShellPane from '../components/ShellPane.svelte';
```

Replace the shell placeholder:
```svelte
        <!-- ShellPane added in Task 8 -->
        <div style="padding: 20px; color: #888;">Shell pane (coming soon)</div>
```
With:
```svelte
        <ShellPane {containerId} visible={activeTab === 'shell'} />
```

- [ ] **Step 3: Build and verify**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/ShellPane.svelte web/src/routes/SessionView.svelte
git commit -m "feat: add ShellPane component for independent bash sessions"
```

---

## Task 9: Create FileTree Component

**Files:**
- Create: `web/src/components/FileTree.svelte`

- [ ] **Step 1: Create `FileTree.svelte`**

Create `web/src/components/FileTree.svelte`:

```svelte
<script lang="ts">
  import { api } from '../lib/api';
  import type { FileEntry } from '../lib/types';
  import { onMount } from 'svelte';

  let { sessionId, onFileSelect }: {
    sessionId: string;
    onFileSelect: (path: string) => void;
  } = $props();

  interface TreeNode {
    name: string;
    path: string;
    type: string;
    size: number;
    children?: TreeNode[];
    expanded?: boolean;
    loading?: boolean;
  }

  let root = $state<TreeNode[]>([]);
  let error = $state('');

  async function loadDir(path: string): Promise<TreeNode[]> {
    const res = await api.listFiles(sessionId, path);
    return res.entries
      .sort((a, b) => {
        // Directories first, then alphabetical
        if (a.type === 'dir' && b.type !== 'dir') return -1;
        if (a.type !== 'dir' && b.type === 'dir') return 1;
        return a.name.localeCompare(b.name);
      })
      .map((e) => ({
        name: e.name,
        path: path === '/' ? e.name : `${path}/${e.name}`,
        type: e.type,
        size: e.size,
      }));
  }

  async function toggleDir(node: TreeNode) {
    if (node.expanded) {
      node.expanded = false;
      return;
    }
    node.loading = true;
    try {
      node.children = await loadDir(node.path);
      node.expanded = true;
    } catch {
      node.children = [];
    }
    node.loading = false;
  }

  function handleClick(node: TreeNode) {
    if (node.type === 'dir') {
      toggleDir(node);
    } else {
      onFileSelect(node.path);
    }
  }

  async function refresh() {
    error = '';
    try {
      root = await loadDir('/');
    } catch (e) {
      error = 'Failed to load file tree';
    }
  }

  onMount(() => { refresh(); });
</script>

<div class="file-tree">
  <div class="tree-header">
    <span class="tree-title">Files</span>
    <button class="refresh-btn" onclick={refresh} title="Refresh">&#x21bb;</button>
  </div>
  {#if error}
    <div class="tree-error">{error}</div>
  {:else}
    <div class="tree-content">
      {#each root as node}
        {@render treeNode(node, 0)}
      {/each}
    </div>
  {/if}
</div>

{#snippet treeNode(node: TreeNode, depth: number)}
  <button
    class="tree-item"
    class:dir={node.type === 'dir'}
    style="padding-left: {12 + depth * 16}px"
    onclick={() => handleClick(node)}
  >
    <span class="tree-icon">
      {#if node.type === 'dir'}
        {node.loading ? '...' : node.expanded ? '&#x25BC;' : '&#x25B6;'}
      {:else}
        &#x25A0;
      {/if}
    </span>
    <span class="tree-name">{node.name}</span>
  </button>
  {#if node.expanded && node.children}
    {#each node.children as child}
      {@render treeNode(child, depth + 1)}
    {/each}
  {/if}
{/snippet}

<style>
  .file-tree {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #1a1a2e;
    border-right: 1px solid #333;
    min-width: 200px;
    max-width: 300px;
    overflow-y: auto;
  }
  .tree-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-bottom: 1px solid #333;
  }
  .tree-title {
    color: #888;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .refresh-btn {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    font-size: 14px;
    padding: 2px 4px;
  }
  .refresh-btn:hover { color: #ccc; }
  .tree-content { padding: 4px 0; }
  .tree-error {
    padding: 12px;
    color: #e66;
    font-size: 12px;
  }
  .tree-item {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 3px 12px;
    background: none;
    border: none;
    color: #ccc;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 12px;
    cursor: pointer;
    text-align: left;
  }
  .tree-item:hover { background: #2a2a4a; }
  .tree-icon {
    font-size: 8px;
    width: 12px;
    text-align: center;
    color: #888;
  }
  .tree-item.dir .tree-name { color: #8be9fd; }
</style>
```

- [ ] **Step 2: Build**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/FileTree.svelte
git commit -m "feat: add FileTree component with lazy directory loading"
```

---

## Task 10: Create MonacoEditor Component

**Files:**
- Create: `web/src/components/MonacoEditor.svelte`

- [ ] **Step 1: Create `MonacoEditor.svelte`**

Create `web/src/components/MonacoEditor.svelte`:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import * as monaco from 'monaco-editor';
  import { api } from '../lib/api';

  // Configure Monaco workers via import.meta.url (Vite native)
  import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
  import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
  import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
  import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
  import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';

  self.MonacoEnvironment = {
    getWorker(_: string, label: string) {
      if (label === 'json') return new jsonWorker();
      if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker();
      if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker();
      if (label === 'typescript' || label === 'javascript') return new tsWorker();
      return new editorWorker();
    },
  };

  let { sessionId, filePath, visible = true }: {
    sessionId: string;
    filePath: string;
    visible?: boolean;
  } = $props();

  let editorDiv: HTMLDivElement;
  let editor: monaco.editor.IStandaloneCodeEditor;
  let dirty = $state(false);
  let saving = $state(false);
  let loadError = $state('');
  let currentPath = '';

  // Map of filePath -> model, so we preserve state per file
  const models = new Map<string, monaco.editor.ITextModel>();

  $effect(() => {
    if (visible && editor) {
      setTimeout(() => editor.layout(), 50);
    }
  });

  $effect(() => {
    if (filePath && editor && filePath !== currentPath) {
      loadFile(filePath);
    }
  });

  function getLanguageForPath(path: string): string {
    const ext = path.split('.').pop()?.toLowerCase() ?? '';
    const langMap: Record<string, string> = {
      go: 'go', ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
      py: 'python', rs: 'rust', rb: 'ruby', java: 'java', c: 'c', cpp: 'cpp', h: 'c',
      cs: 'csharp', php: 'php', swift: 'swift', kt: 'kotlin',
      html: 'html', css: 'css', scss: 'scss', less: 'less',
      json: 'json', yaml: 'yaml', yml: 'yaml', toml: 'toml', xml: 'xml',
      md: 'markdown', sh: 'shell', bash: 'shell', zsh: 'shell',
      dockerfile: 'dockerfile', sql: 'sql', graphql: 'graphql',
      svelte: 'html', vue: 'html',
    };
    return langMap[ext] || 'plaintext';
  }

  async function loadFile(path: string) {
    loadError = '';
    dirty = false;
    currentPath = path;

    // Reuse existing model if we've already loaded this file
    if (models.has(path)) {
      editor.setModel(models.get(path)!);
      return;
    }

    try {
      const res = await api.readFile(sessionId, path);
      const lang = getLanguageForPath(path);
      const uri = monaco.Uri.parse(`file:///${path}`);
      const model = monaco.editor.createModel(res.content, lang, uri);
      model.onDidChangeContent(() => { dirty = true; });
      models.set(path, model);
      editor.setModel(model);
    } catch (e: any) {
      loadError = e?.message || 'Failed to load file';
    }
  }

  async function save() {
    if (!currentPath || saving) return;
    saving = true;
    try {
      const content = editor.getValue();
      await api.writeFile(sessionId, currentPath, content);
      dirty = false;
    } catch (e: any) {
      loadError = e?.message || 'Failed to save';
    }
    saving = false;
  }

  onMount(() => {
    editor = monaco.editor.create(editorDiv, {
      theme: 'vs-dark',
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      minimap: { enabled: false },
      automaticLayout: false,
      scrollBeyondLastLine: false,
    });

    // Ctrl+S / Cmd+S to save
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => save());

    if (filePath) loadFile(filePath);

    const onResize = () => editor.layout();
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('resize', onResize);
      // Dispose all models
      for (const model of models.values()) model.dispose();
      models.clear();
      editor.dispose();
    };
  });
</script>

<div class="monaco-wrapper">
  {#if currentPath}
    <div class="editor-status">
      <span class="editor-filename">
        {currentPath.split('/').pop()}
        {#if dirty}<span class="dirty-dot" title="Unsaved changes"></span>{/if}
      </span>
      {#if saving}
        <span class="save-status">Saving...</span>
      {/if}
      {#if loadError}
        <span class="load-error">{loadError}</span>
      {/if}
    </div>
  {:else}
    <div class="no-file">Select a file to edit</div>
  {/if}
  <div class="editor-container" bind:this={editorDiv}></div>
</div>

<style>
  .monaco-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
    flex: 1;
  }
  .editor-status {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 12px;
    background: #1e1e30;
    border-bottom: 1px solid #333;
    font-size: 12px;
  }
  .editor-filename { color: #ccc; }
  .dirty-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    background: #e8a838;
    border-radius: 50%;
    margin-left: 4px;
    vertical-align: middle;
  }
  .save-status { color: #888; font-size: 11px; }
  .load-error { color: #e66; font-size: 11px; }
  .no-file {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #555;
    font-size: 14px;
  }
  .editor-container { flex: 1; }
</style>
```

- [ ] **Step 2: Build**

Run: `cd /home/bjk/projects/clincus/web && npm run build`
Expected: Builds successfully. If Monaco worker imports fail, try the alternative `?url` import syntax for workers. See Monaco + Vite docs for the exact pattern that works with Vite 6.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/MonacoEditor.svelte
git commit -m "feat: add MonacoEditor component with file loading and save"
```

---

## Task 11: Create EditorPane and Wire Into SessionView

**Files:**
- Create: `web/src/components/EditorPane.svelte`
- Modify: `web/src/routes/SessionView.svelte`

- [ ] **Step 1: Create `EditorPane.svelte`**

Create `web/src/components/EditorPane.svelte`:

```svelte
<script lang="ts">
  import FileTree from './FileTree.svelte';
  import MonacoEditor from './MonacoEditor.svelte';

  let { sessionId, visible = true }: { sessionId: string; visible?: boolean } = $props();

  let selectedFile = $state('');

  function onFileSelect(path: string) {
    selectedFile = path;
  }
</script>

<div class="editor-pane">
  <FileTree {sessionId} {onFileSelect} />
  <MonacoEditor {sessionId} filePath={selectedFile} {visible} />
</div>

<style>
  .editor-pane {
    display: flex;
    height: 100%;
  }
</style>
```

- [ ] **Step 2: Wire EditorPane into SessionView**

In `web/src/routes/SessionView.svelte`, add the import:

```typescript
  import EditorPane from '../components/EditorPane.svelte';
```

Replace the editor placeholder:
```svelte
        <!-- EditorPane added in Task 11 -->
        <div style="padding: 20px; color: #888;">Editor pane (coming soon)</div>
```
With:
```svelte
        <EditorPane sessionId={containerId} visible={activeTab === 'editor'} />
```

- [ ] **Step 3: Build full project**

Run: `make build`
Expected: Frontend and Go binary build successfully.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/EditorPane.svelte web/src/routes/SessionView.svelte
git commit -m "feat: add EditorPane with file tree and Monaco editor"
```

---

## Task 12: Documentation Update

**Files:**
- Modify: `docs/reference/api.md` (if exists)
- Modify: `TODO.md`

- [ ] **Step 1: Update API docs**

If `docs/reference/api.md` exists, add documentation for the new endpoints:

- `GET /ws/shell/{id}` -- Shell WebSocket
- `GET /api/sessions/{id}/files?path=` -- List directory
- `GET /api/sessions/{id}/files/content?path=` -- Read file
- `PUT /api/sessions/{id}/files/content?path=` -- Write file

- [ ] **Step 2: Update TODO.md**

Add a completed item:

```markdown
- [x] Web app session view with tabbed workspace (Session, Shell, Editor)
```

- [ ] **Step 3: Commit**

```bash
git add docs/ TODO.md
git commit -m "docs: update API reference and TODO for tabbed workspace"
```

---

## Task 13: Final Integration Verification

- [ ] **Step 1: Full build**

Run: `make build`
Expected: Success.

- [ ] **Step 2: Run all tests**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 3: Run lint**

Run: `make lint`
Expected: No new lint errors.

- [ ] **Step 4: Manual smoke test (if Incus available)**

1. Start `clincus serve`
2. Open dashboard, create a session
3. Verify Session tab shows tmux terminal (existing behavior preserved)
4. Click Shell tab -- bash prompt appears at `/workspace`
5. Click Editor tab -- file tree loads, click a file, editor shows content
6. Edit a file, press Ctrl+S, verify save works
7. Switch between tabs -- verify state is preserved (terminal scrollback, shell history, editor cursor position)

- [ ] **Step 5: Commit any fixes from smoke testing**
