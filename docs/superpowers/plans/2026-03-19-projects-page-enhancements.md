# Projects Page Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance the Projects landing page with workspace root grouping, alphabetical sorting, and a "new folder + launch" flow.

**Architecture:** Backend gets one new field (`root` on `WorkspaceInfo`) and one new endpoint (`POST /api/workspaces/folder`). Frontend restructures Dashboard to iterate roots and render grouped panels, sorts cards within each group, and adds a new-folder button/dialog per root.

**Tech Stack:** Go 1.24+, Svelte 5 (runes), TypeScript, pure CSS (dark theme)

**Spec:** `docs/superpowers/specs/2026-03-19-projects-page-enhancements-design.md`

---

## File Structure

### Backend changes

| File | Responsibility |
|------|----------------|
| `internal/server/api_workspaces.go` | Add `Root` field to `WorkspaceInfo`, add `handleCreateFolder` handler |
| `internal/server/server.go:70` | Register `POST /api/workspaces/folder` route |
| `internal/server/server_test.go` | Tests for `handleCreateFolder` and `WorkspaceInfo.Root` |

### Frontend changes

| File | Responsibility |
|------|----------------|
| `web/src/lib/types.ts:10-15` | Add `root` field to `Workspace` interface |
| `web/src/lib/api.ts:100-123` | Add `createFolder` method |
| `web/src/stores/workspaces.svelte.ts` | Add `getWorkspacesForRoot()` helper with sorting |
| `web/src/routes/Dashboard.svelte` | Iterate roots, render root container panels |
| `web/src/components/WorkspaceGrid.svelte` | Accept props instead of reading store directly |
| `web/src/components/NewFolderDialog.svelte` | **New** — modal with name input, tool picker, validation |

### Documentation

| File | Responsibility |
|------|----------------|
| `docs/reference/api.md` | Document new endpoint and updated response shape |

---

## Task 1: Add `root` field to `WorkspaceInfo` (backend)

**Files:**
- Modify: `internal/server/api_workspaces.go:13-18` (WorkspaceInfo struct)
- Modify: `internal/server/api_workspaces.go:50-54` (populate root in discovery loop)
- Test: `internal/server/server_test.go`

- [ ] **Step 1: Write failing test — `WorkspaceInfo` includes `root` field**

Add `"os"` and `"path/filepath"` to the import block of `internal/server/server_test.go` (they are not currently imported). Then add:

```go
func TestListWorkspacesIncludesRoot(t *testing.T) {
	// Create a temp directory structure: root/project/.git
	tmpDir := t.TempDir()
	projDir := filepath.Join(tmpDir, "my-project")
	if err := os.MkdirAll(filepath.Join(projDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()
	cfg.Dashboard.WorkspaceRoots = []string{tmpDir}

	srv := New(Options{
		Port:      0,
		Assets:    fstest.MapFS{"index.html": {Data: []byte("<html/>")}},
		AppConfig: cfg,
	})

	req := httptest.NewRequest("GET", "/api/workspaces", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp WorkspacesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(resp.Workspaces))
	}
	if resp.Workspaces[0].Root != tmpDir {
		t.Errorf("expected root %q, got %q", tmpDir, resp.Workspaces[0].Root)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/bjk/projects/clincus && go test ./internal/server/ -run TestListWorkspacesIncludesRoot -v`
Expected: compilation error — `WorkspaceInfo` has no `Root` field.

- [ ] **Step 3: Add `Root` field to `WorkspaceInfo` and populate it**

In `internal/server/api_workspaces.go`, add field to struct:

```go
type WorkspaceInfo struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	Root           string `json:"root"`
	HasConfig      bool   `json:"has_config"`
	ActiveSessions int    `json:"active_sessions"`
}
```

In the discovery loop (line ~50), set the `Root` field:

```go
workspaces = append(workspaces, WorkspaceInfo{
	Path:      fullPath,
	Name:      e.Name(),
	Root:      expanded,
	HasConfig: coiErr == nil,
})
```

Also add an `ExpandedRoots` field to `WorkspacesResponse` so the frontend can render root containers even when a root has no discovered projects (spec requirement: empty roots still render with the "new folder" button):

```go
type WorkspacesResponse struct {
	Roots         []string        `json:"roots"`
	ExpandedRoots []string        `json:"expanded_roots"`
	Workspaces    []WorkspaceInfo `json:"workspaces"`
}
```

In `handleListWorkspaces`, build the expanded roots list before the discovery loop:

```go
func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	roots := s.GetConfig().Dashboard.WorkspaceRoots
	var workspaces []WorkspaceInfo
	expandedRoots := make([]string, 0, len(roots))

	for _, root := range roots {
		expanded := config.ExpandPath(root)
		expandedRoots = append(expandedRoots, expanded)
		entries, err := os.ReadDir(expanded)
		// ... rest of discovery loop unchanged ...
	}

	if workspaces == nil {
		workspaces = []WorkspaceInfo{}
	}
	s.writeJSON(w, WorkspacesResponse{
		Roots:         roots,
		ExpandedRoots: expandedRoots,
		Workspaces:    workspaces,
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/bjk/projects/clincus && go test ./internal/server/ -run TestListWorkspacesIncludesRoot -v`
Expected: PASS

- [ ] **Step 5: Run all server tests to check for regressions**

Run: `cd /home/bjk/projects/clincus && go test ./internal/server/ -v`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/server/api_workspaces.go internal/server/server_test.go
git commit -m "feat(api): add root field to WorkspaceInfo for frontend grouping"
```

---

## Task 2: Add `POST /api/workspaces/folder` endpoint (backend)

**Files:**
- Modify: `internal/server/api_workspaces.go` (add handler)
- Modify: `internal/server/server.go:70` (register route)
- Test: `internal/server/server_test.go`

- [ ] **Step 1: Write failing tests for the new endpoint**

Add to `internal/server/server_test.go`:

```go
func TestCreateFolder_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testConfig()
	cfg.Dashboard.WorkspaceRoots = []string{tmpDir}

	srv := New(Options{
		Port:      0,
		Assets:    fstest.MapFS{"index.html": {Data: []byte("<html/>")}},
		AppConfig: cfg,
	})

	body := strings.NewReader(`{"root":"` + tmpDir + `","name":"my-project"}`)
	req := httptest.NewRequest("POST", "/api/workspaces/folder", body)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct{ Path string `json:"path"` }
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	expected := filepath.Join(tmpDir, "my-project")
	if resp.Path != expected {
		t.Errorf("expected path %q, got %q", expected, resp.Path)
	}
	// Verify directory was created
	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestCreateFolder_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testConfig()
	cfg.Dashboard.WorkspaceRoots = []string{tmpDir}

	srv := New(Options{
		Port:      0,
		Assets:    fstest.MapFS{"index.html": {Data: []byte("<html/>")}},
		AppConfig: cfg,
	})

	cases := []struct {
		name string
		body string
	}{
		{"spaces", `{"root":"` + tmpDir + `","name":"my project"}`},
		{"uppercase", `{"root":"` + tmpDir + `","name":"MyProject"}`},
		{"leading-hyphen", `{"root":"` + tmpDir + `","name":"-bad"}`},
		{"trailing-hyphen", `{"root":"` + tmpDir + `","name":"bad-"}`},
		{"empty", `{"root":"` + tmpDir + `","name":""}`},
		{"slashes", `{"root":"` + tmpDir + `","name":"a/b"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.NewReader(tc.body)
			req := httptest.NewRequest("POST", "/api/workspaces/folder", body)
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)
			if w.Code != 400 {
				t.Errorf("expected 400, got %d for %q", w.Code, tc.name)
			}
		})
	}
}

func TestCreateFolder_RootNotInConfig(t *testing.T) {
	cfg := testConfig()
	cfg.Dashboard.WorkspaceRoots = []string{"/some/configured/root"}

	srv := New(Options{
		Port:      0,
		Assets:    fstest.MapFS{"index.html": {Data: []byte("<html/>")}},
		AppConfig: cfg,
	})

	body := strings.NewReader(`{"root":"/not/configured","name":"test"}`)
	req := httptest.NewRequest("POST", "/api/workspaces/folder", body)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateFolder_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "existing"), 0755)

	cfg := testConfig()
	cfg.Dashboard.WorkspaceRoots = []string{tmpDir}

	srv := New(Options{
		Port:      0,
		Assets:    fstest.MapFS{"index.html": {Data: []byte("<html/>")}},
		AppConfig: cfg,
	})

	body := strings.NewReader(`{"root":"` + tmpDir + `","name":"existing"}`)
	req := httptest.NewRequest("POST", "/api/workspaces/folder", body)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != 409 {
		t.Errorf("expected 409, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/bjk/projects/clincus && go test ./internal/server/ -run TestCreateFolder -v`
Expected: FAIL — route not registered, 405 or SPA fallback.

- [ ] **Step 3: Implement `handleCreateFolder` handler**

Add to `internal/server/api_workspaces.go`:

```go
var validFolderName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root string `json:"root"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request body", 400)
		return
	}

	// Validate folder name
	if !validFolderName.MatchString(req.Name) {
		s.writeError(w, "invalid folder name: must be lowercase alphanumeric with hyphens (e.g., my-project)", 400)
		return
	}

	// Validate root is in configured workspace_roots
	expandedRoot := config.ExpandPath(req.Root)
	roots := s.GetConfig().Dashboard.WorkspaceRoots
	found := false
	for _, cfgRoot := range roots {
		if config.ExpandPath(cfgRoot) == expandedRoot {
			found = true
			break
		}
	}
	if !found {
		s.writeError(w, "root is not a configured workspace root", 400)
		return
	}

	// Create directory
	fullPath := filepath.Join(expandedRoot, req.Name)
	if _, err := os.Stat(fullPath); err == nil {
		s.writeError(w, "directory already exists", 409)
		return
	}
	if err := os.Mkdir(fullPath, 0755); err != nil {
		s.writeError(w, "failed to create directory: "+err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	s.writeJSON(w, map[string]string{"path": fullPath})
}
```

Add `"regexp"` to the imports at the top of the file.

- [ ] **Step 4: Register the route**

In `internal/server/server.go`, in the `routes()` method, add after line 81 (after `DELETE /api/workspaces`):

```go
s.mux.HandleFunc("POST /api/workspaces/folder", s.handleCreateFolder)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/bjk/projects/clincus && go test ./internal/server/ -run TestCreateFolder -v`
Expected: all 4 test functions PASS

- [ ] **Step 6: Run full test suite**

Run: `cd /home/bjk/projects/clincus && go test ./internal/server/ -v`
Expected: all tests PASS

- [ ] **Step 7: Commit**

```bash
git add internal/server/api_workspaces.go internal/server/server.go internal/server/server_test.go
git commit -m "feat(api): add POST /api/workspaces/folder endpoint for creating project directories"
```

---

## Task 3: Update frontend types and API client

**Files:**
- Modify: `web/src/lib/types.ts:10-15`
- Modify: `web/src/lib/api.ts:100-123`

- [ ] **Step 1: Add `root` field to `Workspace` interface**

In `web/src/lib/types.ts`, update the `Workspace` interface:

```typescript
export interface Workspace {
  path: string;
  name: string;
  root: string;
  has_config: boolean;
  active_sessions: number;
}
```

- [ ] **Step 2: Add `createFolder` method to API client**

In `web/src/lib/api.ts`, update the `listWorkspaces` return type and add `createFolder`:

Update the existing `listWorkspaces` line:

```typescript
listWorkspaces: (opts?: RequestOptions) =>
    get<{ roots: string[]; expanded_roots: string[]; workspaces: Workspace[] }>('/api/workspaces', opts),
```

Add after `removeWorkspace`:

```typescript
createFolder: (root: string, name: string, opts?: RequestOptions) =>
    post<{ path: string }>('/api/workspaces/folder', { root, name }, opts),
```

- [ ] **Step 3: Verify build succeeds**

Run: `cd /home/bjk/projects/clincus/web && npm run check 2>&1 | head -20`
Expected: no type errors (or only pre-existing ones)

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/types.ts web/src/lib/api.ts
git commit -m "feat(web): add root field to Workspace type and createFolder API method"
```

---

## Task 4: Add workspace store helper with sorting

**Files:**
- Modify: `web/src/stores/workspaces.svelte.ts`

- [ ] **Step 1: Add `getWorkspacesForRoot()` function**

Replace the full contents of `web/src/stores/workspaces.svelte.ts`:

```typescript
import { api } from '../lib/api';
import type { Workspace } from '../lib/types';

const store = $state<{ workspaces: Workspace[]; roots: string[]; expandedRoots: string[] }>({
  workspaces: [],
  roots: [],
  expandedRoots: [],
});

export function getWorkspaces(): Workspace[] {
  return store.workspaces;
}

export function getRoots(): string[] {
  return store.roots;
}

/** Return expanded root paths in config order. Includes roots with no projects. */
export function getExpandedRoots(): string[] {
  return store.expandedRoots;
}

/** Return workspaces for a given expanded root path, sorted alphabetically by name. */
export function getWorkspacesForRoot(root: string): Workspace[] {
  return store.workspaces
    .filter((ws) => ws.root === root)
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));
}

export async function loadWorkspaces() {
  const data = await api.listWorkspaces();
  store.workspaces = data.workspaces;
  store.roots = data.roots;
  store.expandedRoots = data.expanded_roots;
}
```

- [ ] **Step 2: Verify build succeeds**

Run: `cd /home/bjk/projects/clincus/web && npm run check 2>&1 | head -20`

- [ ] **Step 3: Commit**

```bash
git add web/src/stores/workspaces.svelte.ts
git commit -m "feat(web): add getWorkspacesForRoot helper with alphabetical sorting"
```

---

## Task 5: Add NewFolderDialog component

**Files:**
- Create: `web/src/components/NewFolderDialog.svelte`

- [ ] **Step 1: Create `NewFolderDialog.svelte`**

This dialog reuses the styling patterns from `LaunchDialog.svelte` (overlay, dialog panel, tool picker). It adds a name input with kebab-case validation.

Create `web/src/components/NewFolderDialog.svelte`:

```svelte
<script lang="ts">
  import { api } from '../lib/api';
  import { onMount } from 'svelte';

  let { root, onclose }: { root: string; onclose: () => void } = $props();

  let tools = $state<string[]>([]);
  let selectedTool = $state('claude');
  let folderName = $state('');
  let creating = $state(false);
  let error = $state('');
  let nameInput: HTMLInputElement;

  const VALID_NAME = /^[a-z0-9]+(-[a-z0-9]+)*$/;

  $effect(() => {
    // Auto-focus input when dialog opens
    if (nameInput) nameInput.focus();
  });

  onMount(async () => {
    tools = await api.getTools();
    if (tools.length > 0) selectedTool = tools[0];
  });

  function basename(path: string): string {
    const parts = path.split('/');
    return parts[parts.length - 1] || path;
  }

  let nameValid = $derived(VALID_NAME.test(folderName));
  let nameError = $derived(
    folderName.length === 0
      ? ''
      : nameValid
        ? ''
        : 'Lowercase letters, numbers, and hyphens only (e.g., my-project)',
  );

  async function create() {
    if (!nameValid || creating) return;
    creating = true;
    error = '';
    try {
      const folder = await api.createFolder(root, folderName);
      const session = await api.createSession(folder.path, selectedTool);
      location.hash = `#/terminal/${session.id}`;
      onclose();
    } catch (e: any) {
      error = e.message || 'Failed to create project';
      creating = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
    if (e.key === 'Enter' && nameValid && !creating) create();
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="overlay" onclick={onclose} onkeydown={handleKeydown} role="dialog">
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="dialog" onclick={(e) => e.stopPropagation()} onkeydown={handleKeydown}>
    <h3>New Project in {basename(root)}</h3>

    <label>
      Folder name
      <input
        bind:this={nameInput}
        bind:value={folderName}
        type="text"
        placeholder="my-project"
        class:invalid={nameError}
        disabled={creating}
      />
      {#if nameError}
        <span class="validation">{nameError}</span>
      {/if}
    </label>

    <label>
      Tool
      <select bind:value={selectedTool} disabled={creating}>
        {#each tools as t}
          <option value={t}>{t}</option>
        {/each}
      </select>
    </label>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <div class="actions">
      <button onclick={onclose} disabled={creating}>Cancel</button>
      <button class="primary" onclick={create} disabled={!nameValid || creating}>
        {creating ? 'Creating...' : 'Create & Launch'}
      </button>
    </div>
  </div>
</div>

<style>
  .overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6);
             display: flex; align-items: center; justify-content: center; z-index: 100; }
  .dialog { background: #1e1e30; padding: 24px; border-radius: 8px; min-width: 360px;
            color: #ccc; border: 1px solid #333; }
  h3 { margin: 0 0 16px; }
  label { display: block; margin-bottom: 16px; font-size: 13px; color: #999; }
  input, select { display: block; width: 100%; margin-top: 4px; padding: 8px;
           background: #252540; border: 1px solid #333; color: #ccc; border-radius: 4px;
           box-sizing: border-box; }
  input:focus, select:focus { outline: none; border-color: #555; }
  input.invalid { border-color: #f04040; }
  .validation { font-size: 11px; color: #f04040; margin-top: 4px; display: block; }
  .error { font-size: 12px; color: #f04040; margin: 0 0 12px; }
  .actions { display: flex; gap: 8px; justify-content: flex-end; }
  button { padding: 8px 12px; background: #333; border: none; color: #ccc;
           border-radius: 4px; cursor: pointer; }
  button:hover:not(:disabled) { background: #444; }
  button:disabled { opacity: 0.5; cursor: not-allowed; }
  .primary { background: #4a5568; }
  .primary:hover:not(:disabled) { background: #5a6578; }
</style>
```

- [ ] **Step 2: Verify the frontend builds**

Run: `cd /home/bjk/projects/clincus/web && npm run check 2>&1 | head -20`

- [ ] **Step 3: Commit**

```bash
git add web/src/components/NewFolderDialog.svelte
git commit -m "feat(web): add NewFolderDialog component with name validation and tool picker"
```

---

## Task 6: Restructure Dashboard with root panels, grid, and new-folder button

This task rewrites both `Dashboard.svelte` and `WorkspaceGrid.svelte` in a single pass. `WorkspaceGrid` becomes a simple card renderer (no grid wrapper — the grid CSS moves to Dashboard so the new-folder button participates in the same grid). Dashboard gets root container panels, the new-folder button, and the dialog integration.

**Files:**
- Modify: `web/src/routes/Dashboard.svelte`
- Modify: `web/src/components/WorkspaceGrid.svelte`

- [ ] **Step 1: Update `WorkspaceGrid` to be a simple card renderer**

Replace the full contents of `web/src/components/WorkspaceGrid.svelte`:

```svelte
<script lang="ts">
  import WorkspaceCard from './WorkspaceCard.svelte';
  import type { Workspace } from '../lib/types';

  let { workspaces }: { workspaces: Workspace[] } = $props();
</script>

{#each workspaces as ws (ws.path)}
  <WorkspaceCard workspace={ws} />
{/each}
```

No grid wrapper, no empty state — Dashboard handles both.

- [ ] **Step 2: Rewrite `Dashboard.svelte` with root panels, grid, and new-folder button**

Replace the full contents of `web/src/routes/Dashboard.svelte`:

```svelte
<script lang="ts">
  import WorkspaceGrid from '../components/WorkspaceGrid.svelte';
  import NewFolderDialog from '../components/NewFolderDialog.svelte';
  import { getWorkspacesForRoot, getExpandedRoots } from '../stores/workspaces.svelte';

  let dialogRoot = $state<string | null>(null);

  function basename(path: string): string {
    const parts = path.split('/');
    return parts[parts.length - 1] || path;
  }
</script>

<div class="dashboard">
  <h2>Projects</h2>

  {#each getExpandedRoots() as root (root)}
    <div class="root-container">
      <div class="root-header" title={root}>
        {basename(root)}
      </div>
      <div class="root-body">
        <div class="grid-with-new">
          <WorkspaceGrid workspaces={getWorkspacesForRoot(root)} />
          <button class="new-folder" onclick={() => dialogRoot = root}>
            <span class="plus">+</span>
            <span class="label">New Project</span>
          </button>
        </div>
      </div>
    </div>
  {:else}
    <p class="empty">No workspaces found. Add workspace roots in <a href="#/settings">Settings</a>.</p>
  {/each}
</div>

{#if dialogRoot}
  <NewFolderDialog root={dialogRoot} onclose={() => dialogRoot = null} />
{/if}

<style>
  .dashboard { padding: 16px; color: #ccc; }
  h2 { margin: 0 0 16px; font-size: 18px; }

  .root-container {
    background: #1a1a2e;
    border: 1px solid #2a2a40;
    border-radius: 8px;
    margin-bottom: 16px;
    overflow: hidden;
  }

  .root-header {
    padding: 10px 16px;
    font-size: 14px;
    font-weight: 600;
    color: #999;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 1px solid #2a2a40;
  }

  .root-body {
    padding: 16px;
  }

  .grid-with-new {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 12px;
  }

  .new-folder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 80px;
    background: transparent;
    border: 2px dashed #333;
    border-radius: 8px;
    color: #666;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }

  .new-folder:hover {
    border-color: #555;
    color: #999;
  }

  .new-folder .plus {
    font-size: 24px;
    line-height: 1;
  }

  .new-folder .label {
    font-size: 12px;
    margin-top: 4px;
  }

  .empty { color: #666; }
  a { color: #88f; }
</style>
```

- [ ] **Step 3: Verify full build**

Run: `cd /home/bjk/projects/clincus && make build`
Expected: build succeeds (frontend + Go binary with embedded assets)

- [ ] **Step 4: Commit**

```bash
git add web/src/routes/Dashboard.svelte web/src/components/WorkspaceGrid.svelte
git commit -m "feat(web): restructure dashboard with root panels, grid, and new-folder button"
```

---

## Task 7: Update API documentation

**Files:**
- Modify: `docs/reference/api.md`

- [ ] **Step 1: Update `GET /api/workspaces` response documentation**

In `docs/reference/api.md`, find the `GET /api/workspaces` section (around line 205) and replace the current response description with the actual response shape including the `root` field. This section is out of date — it shows a flat string array but the actual response is `WorkspacesResponse`.

Replace the `GET /api/workspaces` section:

```markdown
### `GET /api/workspaces`

Return configured workspace roots and discovered project directories.

**Response:**

```json
{
  "roots": ["~/projects", "~/work"],
  "workspaces": [
    {
      "path": "/home/user/projects/my-app",
      "name": "my-app",
      "root": "/home/user/projects",
      "has_config": true,
      "active_sessions": 1
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `roots` | string[] | Configured workspace roots (raw config values) |
| `workspaces[].path` | string | Absolute path to the project directory |
| `workspaces[].name` | string | Directory basename |
| `workspaces[].root` | string | Expanded absolute path of the workspace root this project belongs to |
| `workspaces[].has_config` | bool | Whether `.clincus.toml` exists in the project |
| `workspaces[].active_sessions` | int | Number of active sessions for this project |
```

- [ ] **Step 2: Add `POST /api/workspaces/folder` documentation**

Insert after the `DELETE /api/workspaces` section (before the WebSocket section):

```markdown
### `POST /api/workspaces/folder`

Create a new project directory inside a configured workspace root.

**Request body:**

```json
{
  "root": "/home/user/projects",
  "name": "my-new-project"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root` | string | yes | Absolute path to a configured workspace root |
| `name` | string | yes | Folder name (kebab-case: lowercase alphanumeric with hyphens) |

**Folder name validation:** Must match `^[a-z0-9]+(-[a-z0-9]+)*$`. No spaces, uppercase, or special characters.

**Response:** `201 Created`

```json
{
  "path": "/home/user/projects/my-new-project"
}
```

**Errors:**

| Code | Condition |
|------|-----------|
| `400` | Invalid folder name or root not in configured workspace roots |
| `409` | Directory already exists |
| `500` | Filesystem error creating the directory |
```

- [ ] **Step 3: Commit**

```bash
git add docs/reference/api.md
git commit -m "docs: update API reference with root field and POST /api/workspaces/folder endpoint"
```

---

## Task 8: Final verification

- [ ] **Step 1: Run Go tests**

Run: `cd /home/bjk/projects/clincus && make test`
Expected: all tests pass

- [ ] **Step 2: Run linter**

Run: `cd /home/bjk/projects/clincus && make lint`
Expected: no lint errors

- [ ] **Step 3: Full build**

Run: `cd /home/bjk/projects/clincus && make build`
Expected: frontend + Go binary build succeeds

- [ ] **Step 4: Commit any fixes if needed, then done**
