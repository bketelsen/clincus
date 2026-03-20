# Projects Page Enhancements Design

## Summary

Three enhancements to the web dashboard's Projects landing page:

1. **Root container grouping** — visually separate workspace roots into distinct panels with headers
2. **Alphabetical sorting** — sort project cards within each root by name
3. **New folder creation** — add a button per root to create a new project folder and launch a session in it

## Approach

Frontend-only for grouping and sorting. One new backend endpoint for folder creation. The existing API shape (`WorkspacesResponse` with `roots` + flat `workspaces`) is preserved — grouping is a display concern handled in the frontend.

## Design

### 1. Root Container Grouping

**Component changes:**

- `Dashboard.svelte` iterates over `getRoots()` instead of rendering a single `WorkspaceGrid`
- For each root, it renders a **root container panel**: a styled `<div>` with a header bar and nested project grid
- `WorkspaceGrid.svelte` receives a filtered list of workspaces for its root only

**Root container panel styling:**

- Header bar displays the root's basename (e.g., `projects`) with full path as a tooltip
- Background slightly lighter than the page background (`#1a1a2e` or similar), subtle border
- Consistent with existing dark theme aesthetic
- If a root has no discovered projects, the container still renders (with the "new folder" button)

**Data flow:**

- `getRoots()` returns roots in config order (no sorting)
- For each root, filter `getWorkspaces()` by matching path prefix
- Pass filtered list to `WorkspaceGrid`

### 2. Alphabetical Sorting

- When filtering workspaces per root, sort by `workspace.name` using `localeCompare` (case-insensitive)
- Applied in `Dashboard.svelte` or `workspaces.svelte.ts` when building per-root lists
- Roots remain in config-defined order — no sorting applied to the `roots` array

### 3. New Folder Button & Dialog

**Button:**

- Rendered inside each root container, after the project cards grid
- Card-sized element with dashed border, muted "+ New Project" text/icon
- Visually matches `WorkspaceCard` dimensions but styled as an action affordance

**Dialog (modal):**

- **Title:** "New Project in {root basename}"
- **Folder name input:** Text field with real-time kebab-case validation
  - Valid pattern: `^[a-z0-9]+(-[a-z0-9]+)*$`
  - Inline validation message for invalid input
- **Tool picker:** Dropdown or radio buttons populated from `GET /api/tools`
  - Pre-selects the default tool from config
- **Create & Launch button:** Disabled until folder name is valid

**Submit flow:**

1. `POST /api/workspaces/folder` with `{ root: "<root-path>", name: "<folder-name>" }`
2. `POST /api/sessions` with `{ workspace: "<created-path>", tool: "<selected-tool>" }`
3. Navigate to `#/terminal/{session.id}`

**New backend endpoint:**

- **Route:** `POST /api/workspaces/folder`
- **File:** `internal/server/api_workspaces.go`
- **Request body:** `{ "root": string, "name": string }`
- **Response:** `{ "path": string }` (full path of created directory)
- **Validation:**
  - `name` matches `^[a-z0-9]+(-[a-z0-9]+)*$`
  - `root` is in configured `workspace_roots` (prevents arbitrary path creation)
  - Directory does not already exist at `{root}/{name}`
- **Implementation:** `os.MkdirAll(filepath.Join(root, name), 0755)`
- **Error responses:**
  - `400` — invalid name or root not in config
  - `409` — directory already exists

**Error handling:**

- Name validation happens client-side (instant feedback) and server-side (security)
- If folder creation fails: toast error, dialog stays open
- If session creation fails after folder was created: toast error, navigate back to dashboard

## Files Changed

### Frontend (new or modified)

| File | Change |
|------|--------|
| `web/src/routes/Dashboard.svelte` | Iterate roots, render root containers, filter/sort workspaces per root |
| `web/src/components/WorkspaceGrid.svelte` | Accept filtered workspace list (may simplify) |
| `web/src/components/NewFolderButton.svelte` | **New** — card-sized "+" button |
| `web/src/components/NewFolderDialog.svelte` | **New** — modal with name input, tool picker, validation |
| `web/src/lib/api.ts` | Add `createFolder(root, name)` method |
| `web/src/stores/workspaces.svelte.ts` | Possibly add helper for per-root filtering/sorting |

### Backend (new or modified)

| File | Change |
|------|--------|
| `internal/server/api_workspaces.go` | Add `handleCreateFolder` handler |
| `internal/server/server.go` | Register `POST /api/workspaces/folder` route |

### Documentation

| File | Change |
|------|--------|
| `docs/reference/api.md` | Document new `POST /api/workspaces/folder` endpoint |

## Out of Scope

- Changing the API response shape (grouping stays frontend-only)
- Recursive workspace discovery
- Empty folder persistence in the discovery list
- Folder deletion or renaming from the dashboard
- Collapsible/accordion root sections
