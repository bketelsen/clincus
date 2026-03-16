# Workspaces

A workspace is the host directory that Clincus mounts into the container for the AI tool to
work on. Understanding how workspace mounting works helps you get the most out of Clincus.

---

## Default Mounting

By default, the workspace is mounted at `/workspace` inside the container:

```bash
clincus shell ~/my-project
# host:  ~/my-project  ->  container: /workspace
```

The AI tool's working directory is set to `/workspace` when the session starts.

### Preserve Host Path

If you prefer the workspace to appear at the same path inside the container (useful for tools
that embed absolute paths in their config), enable `preserve_workspace_path`:

```toml
# ~/.config/clincus/config.toml
[paths]
preserve_workspace_path = true
```

With this enabled:

```
host:  /home/user/my-project  ->  container: /home/user/my-project
```

---

## UID Shifting

Clincus uses Incus's UID-shifting (idmapped mounts) so files in the workspace appear owned
by the `code` user (UID 1000) inside the container, while remaining owned by your user on
the host. You can read and write files normally from both sides.

If your system does not support idmapped mounts (e.g., some macOS VMs), disable UID shifting:

```toml
[incus]
disable_shift = true
```

---

## Additional Mounts

Mount extra directories into the container beyond the workspace. Useful for shared credential
stores, package caches, or other project-independent data.

### Via CLI flag

```bash
clincus shell --mount ~/.ssh:/home/code/.ssh ~/my-project
clincus shell --mount ~/shared-data:/data --mount ~/.npmrc:/home/code/.npmrc ~/my-project
```

The format is `HOST_PATH:CONTAINER_PATH`. The `--mount` flag can be specified multiple times.

### Via config file

```toml
# ~/.config/clincus/config.toml
[[mounts.default]]
host = "~/.npmrc"
container = "/home/code/.npmrc"

[[mounts.default]]
host = "~/shared-caches/pip"
container = "/home/code/.cache/pip"
```

Home directory expansion (`~`) is supported in `host` paths.

---

## Security: Protected Paths

By default, Clincus mounts certain workspace subdirectories **read-only** inside the container
to prevent the AI tool from modifying files that execute automatically on your host:

| Protected path | Risk if writable |
|----------------|-----------------|
| `.git/hooks` | Git hooks execute on every commit, push, etc. |
| `.git/config` | Can redirect `core.hooksPath` to bypass hook protection |
| `.husky` | Husky wraps Git hooks; scripts run automatically |
| `.vscode` | `tasks.json` runs on open; `settings.json` injects shell args |

The AI tool can read these files but cannot write to them.

### Add More Protected Paths

```toml
[security]
additional_protected_paths = [".github/workflows", ".circleci"]
```

### Replace the Default List

```toml
[security]
protected_paths = [".git/hooks", ".custom-hooks"]
```

### Disable All Protection

```bash
# One session
clincus shell --writable-git-hooks

# Permanently (not recommended)
# [security]
# disable_protection = true
```

---

## File Transfer

Transfer files between host and container outside of the workspace mount.

### Push (host → container)

```bash
# Push a file
clincus file push ./config.json clincus-a1b2c3-1:/workspace/config.json

# Push a directory
clincus file push -r ./scripts clincus-a1b2c3-1:/workspace/scripts
```

### Pull (container → host)

```bash
# Pull a file
clincus file pull clincus-a1b2c3-1:/workspace/output.log ./output.log

# Pull a directory
clincus file pull -r clincus-a1b2c3-1:/root/.claude ./saved-sessions/
```

### Identify the Container Name

Use `clincus list` to find the container name for a running session:

```bash
clincus list
```

---

## Isolation Model

Each session gets its own container. The workspace mount is shared, but:

- The container filesystem is isolated (changes outside `/workspace` don't affect the host)
- The container has its own network stack (no access to host services by default)
- The container runs processes as UID 1000 (`code`) with limited privileges
- The host cannot be reached via the loopback interface from inside the container

This isolation means the AI tool cannot accidentally (or intentionally) modify files outside
your workspace, install host-level packages, or access secrets that aren't explicitly mounted.
