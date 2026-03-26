# CLI Reference

Complete reference for all `clincus` commands and flags.

---

## Global Flags

These flags apply to all commands that start or interact with sessions.

| Flag | Default | Description |
|------|---------|-------------|
| `--workspace`, `-w` | `.` | Workspace directory to mount into the container |
| `--slot` | `0` | Slot number for parallel sessions (0 = auto-allocate) |
| `--image` | `clincus` | Incus image alias to use |
| `--persistent` | `false` | Keep container alive across sessions |
| `--resume` | — | Resume from a session ID (omit value for auto-detect) |
| `--continue` | — | Alias for `--resume` |
| `--profile` | — | Apply a named profile from config |
| `--env`, `-e` | — | Environment variable in `KEY=VALUE` form (repeatable) |
| `--mount` | — | Extra mount in `HOST:CONTAINER` form (repeatable) |
| `--writable-git-hooks` | `false` | Allow writing to `.git/hooks` (disables security protection) |

### Resource Limit Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--limit-cpu` | CPU count or pin | `2`, `0-3`, `0,1,3` |
| `--limit-cpu-allowance` | CPU time allowance | `50%`, `25ms/100ms` |
| `--limit-cpu-priority` | Scheduler priority (0–10) | `5` |
| `--limit-memory` | Memory limit | `2GiB`, `512MiB`, `50%` |
| `--limit-memory-swap` | Swap behavior | `true`, `false`, `1GiB` |
| `--limit-memory-enforce` | Enforcement mode | `hard`, `soft` |
| `--limit-disk-read` | Disk read rate | `10MiB/s`, `1000iops` |
| `--limit-disk-write` | Disk write rate | `5MiB/s` |
| `--limit-disk-max` | Combined disk I/O | `20MiB/s` |
| `--limit-disk-priority` | Disk scheduler priority | `5` |
| `--limit-processes` | Max processes (0 = unlimited) | `500` |
| `--limit-duration` | Max runtime | `2h`, `30m`, `1h30m` |

---

## Commands

### `clincus shell`

Start an interactive AI coding session in a container.

```
clincus shell [flags]
```

All sessions run inside a tmux session in the container. Attach interactively by default;
use `--background` to run detached.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--tool` | (from config) | AI tool to run: `claude`, `copilot`, `opencode` |
| `--debug` | `false` | Launch bash instead of the AI tool (for debugging) |
| `--background` | `false` | Run the AI tool in a detached background tmux session |
| `--tmux` | `true` | Use tmux for session management |
| `--container` | — | Attach to an existing container by name (for testing) |

**Examples:**

```bash
clincus shell                              # Interactive session in tmux
clincus shell --tool opencode             # Use opencode
clincus shell --background                # Run in background
clincus shell --resume                    # Resume latest session (auto-detect)
clincus shell --resume=abc123             # Resume specific session (= required)
clincus shell --continue=abc123           # Same as --resume
clincus shell --slot 2                    # Use slot 2
clincus shell --debug                     # Open bash for debugging
clincus shell --persistent ~/my-project   # Persistent container
```

**tmux key bindings inside the session:**

| Key | Action |
|-----|--------|
| `Ctrl+B d` | Detach (session keeps running) |
| `Ctrl+C` | Interrupt the AI tool (falls back to bash) |

---

### `clincus attach`

Attach to a running AI coding session.

```
clincus attach [container-name] [flags]
```

If no container name is given: lists sessions if multiple are running, or auto-attaches if
exactly one is running.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--bash` | `false` | Attach to a bash shell instead of the tmux session |
| `--slot` | `0` | Attach to the slot for the current workspace |
| `--workspace`, `-w` | `.` | Workspace directory (used with `--slot`) |

**Examples:**

```bash
clincus attach                            # Auto-attach or list
clincus attach clincus-a1b2c3-1          # Attach to specific container
clincus attach --slot 1                   # Attach to slot 1 for current workspace
clincus attach --bash                     # Bash shell instead of tmux
```

---

### `clincus run`

Execute a command in an ephemeral container (non-interactive).

```
clincus run COMMAND [flags]
```

The container is created, the command runs, and the container is deleted on completion.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--capture` | `false` | Capture output instead of streaming |
| `--timeout` | `120` | Command timeout in seconds |
| `--format` | `pretty` | Output format: `pretty` or `json` |

**Examples:**

```bash
clincus run "npm test"
clincus run "pytest" --slot 2
clincus run --workspace ~/project "make build"
clincus run "echo hello" --capture
```

---

### `clincus list`

List active containers and saved sessions.

```
clincus list [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Include saved (stopped) sessions |
| `--format` | `text` | Output format: `text` or `json` |

**Examples:**

```bash
clincus list
clincus list --all
clincus list --format json
```

---

### `clincus build`

Build the Incus image for AI coding sessions.

```
clincus build [flags]
clincus build custom NAME --script SCRIPT [flags]
```

**Flags (base image):**

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Rebuild even if image exists |

**Subcommand `custom`:**

Build a custom image from a shell script.

| Flag | Default | Description |
|------|---------|-------------|
| `--script` | (required) | Path to the build script |
| `--base` | `clincus` | Base image alias |
| `--force` | `false` | Force rebuild |

**Examples:**

```bash
clincus build
clincus build --force
clincus build custom my-rust-image --script setup-rust.sh
clincus build custom node-20 --base images:ubuntu/24.04 --script install-node.sh
```

---

### `clincus images`

List available Incus images (alias for `clincus image list`).

```
clincus images
```

---

### `clincus image`

Subcommand group for image management.

```
clincus image list
clincus image info NAME
```

---

### `clincus kill`

Force-delete a running or stopped container.

```
clincus kill CONTAINER-NAME
```

**Example:**

```bash
clincus kill clincus-a1b2c3-1
```

---

### `clincus snapshot`

Manage container snapshots.

```
clincus snapshot create [name] [flags]
clincus snapshot list [flags]
clincus snapshot restore NAME [flags]
clincus snapshot delete NAME [flags]
clincus snapshot info NAME [flags]
```

#### `snapshot create`

| Flag | Default | Description |
|------|---------|-------------|
| `--container`, `-c` | (auto-detect) | Container name |
| `--stateful` | `false` | Include process memory state |

#### `snapshot list`

| Flag | Default | Description |
|------|---------|-------------|
| `--container`, `-c` | (auto-detect) | Container name |
| `--all` | `false` | All Clincus containers |
| `--format` | `text` | `text` or `json` |

#### `snapshot restore`

| Flag | Default | Description |
|------|---------|-------------|
| `--container`, `-c` | (auto-detect) | Container name |
| `--force`, `-f` | `false` | Skip confirmation |
| `--stateful` | `false` | Restore process memory state |

#### `snapshot delete`

| Flag | Default | Description |
|------|---------|-------------|
| `--container`, `-c` | (auto-detect) | Container name |
| `--force`, `-f` | `false` | Skip confirmation |
| `--all` | `false` | Delete all snapshots |

#### `snapshot info`

| Flag | Default | Description |
|------|---------|-------------|
| `--container`, `-c` | (auto-detect) | Container name |
| `--format` | `text` | `text` or `json` |

**Examples:**

```bash
clincus snapshot create
clincus snapshot create checkpoint-1
clincus snapshot create --stateful live
clincus snapshot list
clincus snapshot list --all --format json
clincus snapshot restore checkpoint-1
clincus snapshot restore checkpoint-1 -f
clincus snapshot delete checkpoint-1
clincus snapshot delete --all -f
clincus snapshot info checkpoint-1
```

---

### `clincus file`

Transfer files between host and containers.

```
clincus file push LOCAL CONTAINER:REMOTE [flags]
clincus file pull CONTAINER:REMOTE LOCAL [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-r` | `false` | Push/pull recursively (for directories) |

**Examples:**

```bash
clincus file push ./config.json clincus-a1b2c3-1:/workspace/config.json
clincus file push -r ./src clincus-a1b2c3-1:/workspace/src
clincus file pull clincus-a1b2c3-1:/workspace/output.log ./output.log
clincus file pull -r clincus-a1b2c3-1:/root/.claude ./saved-session/
```

---

### `clincus serve`

Start the web dashboard.

```
clincus serve [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `3000` | Port to listen on |
| `--open` | `false` | Open browser after starting |

**Examples:**

```bash
clincus serve
clincus serve --port 8080
clincus serve --open
```

---

### `clincus service`

Manage the clincus web dashboard as a systemd user service.

```
clincus service install
clincus service start
clincus service stop
clincus service remove
```

#### `service install`

Install the systemd user unit file to `~/.config/systemd/user/clincus.service`, reload the daemon, and enable the service to start on login. The unit runs `clincus serve` and restarts on failure.

#### `service start`

Start the clincus service via `systemctl --user start`.

#### `service stop`

Stop the clincus service via `systemctl --user stop`.

#### `service remove`

Stop the service, disable it, remove the unit file, and reload the systemd daemon.

**Examples:**

```bash
clincus service install   # Install and enable the service
clincus service start     # Start the dashboard
clincus service stop      # Stop the dashboard
clincus service remove    # Uninstall the service
```

---

### `clincus health`

Check system health and dependencies.

```
clincus health [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text` or `json` |
| `--verbose`, `-v` | `false` | Include additional checks |

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | Warnings (functional but degraded) |
| `2` | Critical failures |

**Examples:**

```bash
clincus health
clincus health --format json
clincus health --verbose
```

---

### `clincus info`

Show information about the current workspace or a specific container.

```
clincus info [container-name]
```

---

### `clincus resume`

Resume a previously saved session.

```
clincus resume SESSION-ID
```

This is a shorthand for `clincus shell --resume=SESSION-ID`.

---

### `clincus clean`

Cleanup stopped containers, saved session data, and orphaned resources.

By default, cleans only stopped containers. Use flags to control what gets cleaned.

Orphaned containers (`--orphans`) are stopped containers whose workspace directory no longer exists on the host filesystem.

```
clincus clean                    # Clean stopped containers
clincus clean --sessions         # Clean saved session data
clincus clean --orphans          # Clean containers with missing workspaces
clincus clean --all              # Clean everything
clincus clean --all --force      # Clean without confirmation
clincus clean --dry-run          # Show what would be cleaned
```

---

### `clincus shutdown`

Gracefully stop and delete one or more containers. Attempts a graceful shutdown first,
waiting for the timeout before force-killing if necessary.

```
clincus shutdown [container-name...] [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--timeout` | `60` | Seconds to wait for graceful shutdown before force-killing |
| `--force` | `false` | Skip confirmation prompts |
| `--all` | `false` | Shutdown all containers |

**Examples:**

```bash
clincus shutdown clincus-a1b2c3-1                # Graceful shutdown (60s timeout)
clincus shutdown --timeout=30 clincus-a1b2c3-1   # 30 second timeout
clincus shutdown --all                            # Shutdown all containers
clincus shutdown --all --force                    # Shutdown all without confirmation
```

---

### `clincus persist`

Convert ephemeral sessions to persistent mode. Persistent containers are not automatically
deleted when stopped, preserving installed tools and configurations across sessions.

```
clincus persist [container-name...] [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation prompts |
| `--all` | `false` | Persist all containers |

**Examples:**

```bash
clincus persist clincus-a1b2c3-1                              # Persist specific container
clincus persist clincus-a1b2c3-1 clincus-xyz78901-2           # Persist multiple containers
clincus persist --all                                          # Persist all (with confirmation)
clincus persist --all --force                                  # Persist all without confirmation
```

---

### `clincus version`

Print version, commit, and build date.

```
clincus version
```

Output:

```
clincus v0.1.0 (commit: abc1234, built: 2026-01-01T00:00:00Z)
https://github.com/bketelsen/clincus
```

---

### `clincus container`

Subcommand group for direct container management.

```
clincus container list
clincus container stop CONTAINER-NAME
clincus container start CONTAINER-NAME
```

---

### `clincus tmux`

Interact with tmux sessions inside containers.

```
clincus tmux capture CONTAINER-NAME
clincus tmux send CONTAINER-NAME COMMAND
```

**Examples:**

```bash
# View output of a background session
clincus tmux capture clincus-a1b2c3-1

# Send a prompt to a background session
clincus tmux send clincus-a1b2c3-1 "add unit tests for the auth module"
```
