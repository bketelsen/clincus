# Configuration Reference

Clincus uses TOML configuration files. Settings are merged in priority order (highest wins):

1. `CLINCUS_CONFIG` environment variable path
2. `.clincus.toml` in the current working directory (project config)
3. `~/.config/clincus/config.toml` (user config)
4. `/etc/clincus/config.toml` (system config)

Project configs are useful for per-repository settings like the default tool or image.
User configs store personal preferences and credentials paths.

---

## Complete Example

```toml
# ~/.config/clincus/config.toml

[defaults]
image = "clincus"
persistent = false
model = "claude-sonnet-4-5"

[paths]
sessions_dir = "~/.clincus/sessions"
storage_dir = "~/.clincus/storage"
logs_dir = "~/.clincus/logs"
preserve_workspace_path = false

[incus]
project = "default"
group = "incus-admin"
code_uid = 1000
code_user = "code"
disable_shift = false

[tool]
name = "claude"
binary = ""

[tool.claude]
effort_level = "medium"

[[mounts.default]]
host = "~/.npmrc"
container = "/home/code/.npmrc"

[[mounts.default]]
host = "~/shared/pip-cache"
container = "/home/code/.cache/pip"

[limits.cpu]
count = ""
allowance = ""
priority = 0

[limits.memory]
limit = ""
enforce = "soft"
swap = "true"

[limits.disk]
read = ""
write = ""
max = ""
priority = 0
tmpfs_size = ""

[limits.runtime]
max_duration = ""
max_processes = 0
auto_stop = true
stop_graceful = true

[git]
writable_hooks = false

[security]
protected_paths = [".git/hooks", ".git/config", ".husky", ".vscode"]
additional_protected_paths = []
disable_protection = false

[dashboard]
port = 3000
workspace_roots = ["~/projects", "~/work"]

[profiles.restricted]
persistent = false
[profiles.restricted.limits.memory]
limit = "2GiB"
enforce = "hard"
[profiles.restricted.limits.runtime]
max_duration = "1h"
auto_stop = true
```

---

## `[defaults]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `image` | string | `"clincus"` | Default Incus image alias |
| `persistent` | bool | `false` | Default persistence mode |
| `model` | string | `"claude-sonnet-4-5"` | Default model (informational) |

---

## `[paths]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `sessions_dir` | string | `~/.clincus/sessions` | Where session data is saved |
| `storage_dir` | string | `~/.clincus/storage` | General storage directory |
| `logs_dir` | string | `~/.clincus/logs` | Log file directory |
| `preserve_workspace_path` | bool | `false` | Mount workspace at its host path instead of `/workspace` |

---

## `[incus]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `project` | string | `"default"` | Incus project name |
| `group` | string | `"incus-admin"` | Incus admin group for permission checks |
| `code_uid` | int | `1000` | UID of the `code` user inside containers |
| `code_user` | string | `"code"` | Username inside containers |
| `disable_shift` | bool | `false` | Disable UID-shifting (use on macOS Colima/Lima) |

---

## `[tool]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | `"claude"` | Tool to run: `claude`, `copilot`, `opencode` |
| `binary` | string | `""` | Override binary path (empty = use tool default) |

### `[tool.claude]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `effort_level` | string | `"medium"` | Claude thinking effort: `"low"`, `"medium"`, `"high"` |

---

## `[[mounts.default]]`

Repeatable. Each entry defines an additional directory mount applied to every session.

| Key | Type | Description |
|-----|------|-------------|
| `host` | string | Host path (supports `~` expansion) |
| `container` | string | Absolute path inside the container |

```toml
[[mounts.default]]
host = "~/.ssh"
container = "/home/code/.ssh"
```

---

## `[limits.cpu]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `count` | string | `""` (unlimited) | CPU count or pin: `"2"`, `"0-3"`, `"0,1,3"` |
| `allowance` | string | `""` (unlimited) | CPU time: `"50%"`, `"25ms/100ms"` |
| `priority` | int | `0` | Scheduler priority 0–10 |

---

## `[limits.memory]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `limit` | string | `""` (unlimited) | Memory limit: `"2GiB"`, `"512MiB"`, `"50%"` |
| `enforce` | string | `"soft"` | Enforcement: `"hard"` or `"soft"` |
| `swap` | string | `"true"` | Swap: `"true"`, `"false"`, or size like `"1GiB"` |

---

## `[limits.disk]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `read` | string | `""` (unlimited) | Read rate: `"10MiB/s"`, `"1000iops"` |
| `write` | string | `""` (unlimited) | Write rate: `"5MiB/s"` |
| `max` | string | `""` (unlimited) | Combined read+write rate |
| `priority` | int | `0` | Disk I/O priority 0–10 |
| `tmpfs_size` | string | `""` (use container root) | `/tmp` tmpfs size: `"2GiB"` |

---

## `[limits.runtime]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `max_duration` | string | `""` (unlimited) | Max runtime: `"2h"`, `"30m"`, `"1h30m"` |
| `max_processes` | int | `0` (unlimited) | Max processes in the container |
| `auto_stop` | bool | `true` | Auto-stop container when duration limit is reached |
| `stop_graceful` | bool | `true` | Send graceful stop signal before force-kill |

---

## `[git]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `writable_hooks` | bool | `false` | Allow container to write to `.git/hooks` |

---

## `[security]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `protected_paths` | []string | `[".git/hooks", ".git/config", ".husky", ".vscode"]` | Paths mounted read-only in the container |
| `additional_protected_paths` | []string | `[]` | Extra paths to protect (appended to `protected_paths`) |
| `disable_protection` | bool | `false` | Disable all read-only path protection |

---

## `[dashboard]`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `port` | int | `3000` | HTTP port for `clincus serve` |
| `workspace_roots` | []string | `[]` | Directories shown as suggestions in the New Session dialog |

---

## `[profiles.NAME]`

Named profiles bundle settings that can be applied with `--profile NAME`.

| Key | Type | Description |
|-----|------|-------------|
| `image` | string | Override default image |
| `persistent` | bool | Override default persistence |
| `environment` | map[string]string | Extra environment variables |
| `limits` | LimitsConfig | Limits block (same structure as `[limits]`) |

```toml
[profiles.heavy]
image = "clincus-gpu"
persistent = true
[profiles.heavy.limits.cpu]
count = "8"
[profiles.heavy.limits.memory]
limit = "16GiB"
```

Apply with:

```bash
clincus shell --profile heavy ~/ml-project
```

---

## Config Hot-Reload

When running `clincus serve`, the server automatically watches the system config (`/etc/clincus/config.toml`) and user config (`~/.config/clincus/config.toml`) for changes. When a file is modified:

- The config is reloaded automatically (debounced with a 1-second window to handle editors that write files in multiple steps).
- If the new config is valid, it takes effect immediately for new sessions and API responses.
- If the new config has errors (invalid TOML, missing fields), the error is logged and the previous valid config is retained.
- If the `[dashboard] port` changes, the HTTP listener is gracefully restarted on the new port. Active WebSocket connections will be dropped during the restart but tmux sessions in containers are unaffected.
- Active Incus container sessions are never affected by a config reload.

Project config (`.clincus.toml`) and `CLINCUS_CONFIG` paths are **not** watched.

---

## Per-Project Config: `.clincus.toml`

Place `.clincus.toml` in your project root to configure Clincus for that project only. Any
key from the above reference is valid. Common per-project settings:

```toml
# .clincus.toml — committed alongside the project

[tool]
name = "opencode"

[defaults]
image = "my-custom-image"

[security]
additional_protected_paths = [".github/workflows"]
```

---

## Environment Variable Override

Set `CLINCUS_CONFIG` to point to an alternative config file:

```bash
CLINCUS_CONFIG=/path/to/my-config.toml clincus shell
```

This path is loaded with the highest priority and merges on top of all other config files.
