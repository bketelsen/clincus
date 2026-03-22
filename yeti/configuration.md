# Configuration Reference

Configuration is loaded by `internal/config/loader.go` with hot-reload support via `internal/config/reload.go`.

## Config File Locations (Precedence Low → High)

1. **System**: `/etc/clincus/config.toml`
2. **User**: `~/.config/clincus/config.toml`
3. **Project**: `./.clincus.toml` (current working directory)
4. **Env override**: `CLINCUS_CONFIG` points to a specific file

All files are TOML format. Higher precedence overrides lower.

## Config Sections

### [defaults]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `image` | string | `"clincus"` | Default container image |
| `persistent` | bool | `false` | Reuse containers by default |
| `model` | string | `"claude-sonnet-4-5"` | AI model |

### [paths]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `sessions_dir` | string | `~/.clincus/sessions` | Saved session storage |
| `storage_dir` | string | `~/.clincus/storage` | General storage |
| `logs_dir` | string | `~/.clincus/logs` | Logs directory |
| `preserve_workspace_path` | bool | `false` | Mount workspace at same host path instead of `/workspace` |

### [incus]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `project` | string | `"default"` | Incus project name |
| `group` | string | `"incus-admin"` | User group for Incus access |
| `code_uid` | int | `1000` | UID for code user in container |
| `code_user` | string | `"code"` | Username in container |
| `disable_shift` | bool | `false` | Disable UID/GID shifting (for Colima/Lima) |

### [tool]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | `"claude"` | Tool: `claude`, `copilot`, `opencode` |
| `binary` | string | `""` | Binary path override (empty = tool default) |

#### [tool.claude]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `effort_level` | string | `"medium"` | Effort level: `low`, `medium`, `high` |

### [[mounts.default]]

Array of additional mount entries:

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | Host path (supports `~` expansion) |
| `container` | string | Container path (must be absolute) |

### [git]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `writable_hooks` | bool | `false` | Allow container writes to .git/hooks |

### [security]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `protected_paths` | []string | `[".git/hooks", ".git/config", ".husky", ".vscode"]` | Paths mounted read-only |
| `additional_protected_paths` | []string | `[]` | Extra paths to protect |
| `disable_protection` | bool | `false` | Disable all read-only protection |

### [limits]

#### [limits.cpu]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `count` | string | `""` | CPU count: `"2"`, `"0-3"`, `"0,1,3"` |
| `allowance` | string | `""` | `"50%"`, `"25ms/100ms"` |
| `priority` | int | `0` | Priority 0-10 |

#### [limits.memory]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | string | `""` | `"512MiB"`, `"2GiB"`, `"50%"` |
| `enforce` | string | `"soft"` | `"hard"` or `"soft"` |
| `swap` | string | `"true"` | `"true"`, `"false"`, or size |

#### [limits.disk]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `read` | string | `""` | `"10MiB/s"`, `"1000iops"` |
| `write` | string | `""` | `"5MiB/s"`, `"1000iops"` |
| `max` | string | `""` | Combined read+write limit |
| `priority` | int | `0` | Priority 0-10 |
| `tmpfs_size` | string | `""` | `/tmp` size: `"2GiB"` (empty = root disk) |

#### [limits.runtime]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_duration` | string | `""` | `"2h"`, `"30m"`, `"1h30m"` |
| `max_processes` | int | `0` | Process limit (0 = unlimited) |
| `auto_stop` | bool | `true` | Stop container when limit reached |
| `stop_graceful` | bool | `true` | Graceful vs force stop |

### [dashboard]

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | `3000` | Dashboard listen port |
| `workspace_roots` | []string | `[]` | Workspace root directories |

### [profiles.\<name\>]

Named configuration profiles applied via `--profile` flag.

| Field | Type | Description |
|-------|------|-------------|
| `image` | string | Override default image |
| `environment` | map[string]string | Environment variables |
| `persistent` | bool | Override persistent mode |
| `limits` | LimitsConfig | Override resource limits |

## Environment Variables

| Variable | Maps To |
|----------|---------|
| `CLINCUS_CONFIG` | Config file path (highest priority) |
| `CLINCUS_IMAGE` | `defaults.image` |
| `CLINCUS_SESSIONS_DIR` | `paths.sessions_dir` |
| `CLINCUS_STORAGE_DIR` | `paths.storage_dir` |
| `CLINCUS_PERSISTENT` | `defaults.persistent` (`true` or `1`) |
| `CLINCUS_LIMIT_CPU` | `limits.cpu.count` |
| `CLINCUS_LIMIT_CPU_ALLOWANCE` | `limits.cpu.allowance` |
| `CLINCUS_LIMIT_MEMORY` | `limits.memory.limit` |
| `CLINCUS_LIMIT_MEMORY_SWAP` | `limits.memory.swap` |
| `CLINCUS_LIMIT_DISK_READ` | `limits.disk.read` |
| `CLINCUS_LIMIT_DISK_WRITE` | `limits.disk.write` |
| `CLINCUS_LIMIT_DURATION` | `limits.runtime.max_duration` |
| `CLINCUS_CONTAINER_PREFIX` | Container name prefix (for listing/detection) |
| `CLINCUS_CONTAINER` | Specific container name (for snapshot ops) |

## Hot-Reload

The `ConfigManager` (`internal/config/reload.go`) watches system and user config files:

1. Uses `fsnotify` to watch for file changes
2. 1-second debounce before reload
3. `onChange` callback fires on successful reload
4. Failed reloads retain previous valid config
5. Watcher failure is non-fatal (server continues with last config)

Used by the web dashboard server to dynamically update port and workspace roots.
