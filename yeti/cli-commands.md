# CLI Commands Reference

All 21 top-level commands are defined in `internal/cli/`. The root command (`clincus`) defaults to `shell` when no subcommand is given. Some commands (`attach`, `shutdown`) register themselves via `init()` functions rather than in `root.go`'s central `AddCommand` block.

## Global Flags

Available on all commands (defined in `internal/cli/root.go`):

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--workspace`, `-w` | string | `.` | Workspace directory to mount |
| `--slot` | int | 0 | Slot number (0 = auto-allocate) |
| `--image` | string | config | Custom container image |
| `--persistent` | bool | false | Reuse container across sessions |
| `--resume` | string | `auto` | Resume from session ID |
| `--continue` | string | `auto` | Alias for `--resume` |
| `--profile` | string | | Named config profile |
| `--env`, `-e` | []string | | Environment variables (KEY=VALUE) |
| `--mount` | []string | | Mount directories (HOST:CONTAINER) |
| `--writable-git-hooks` | bool | false | Allow writes to .git/hooks |

### Resource Limit Flags (Global)

| Flag | Example |
|------|---------|
| `--limit-cpu` | `2`, `0-3`, `0,1,3` |
| `--limit-cpu-allowance` | `50%`, `25ms/100ms` |
| `--limit-cpu-priority` | `0`-`10` |
| `--limit-memory` | `2GiB`, `512MiB`, `50%` |
| `--limit-memory-swap` | `true`, `false`, size |
| `--limit-memory-enforce` | `hard`, `soft` |
| `--limit-disk-read` | `10MiB/s`, `1000iops` |
| `--limit-disk-write` | `5MiB/s`, `1000iops` |
| `--limit-disk-max` | Combined I/O limit |
| `--limit-disk-priority` | `0`-`10` |
| `--limit-processes` | 0 = unlimited |
| `--limit-duration` | `2h`, `30m`, `1h30m` |

## Commands

### shell (default)

Start an interactive AI coding session. All sessions run in tmux.

```
clincus [shell] [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--debug` | false | Launch bash instead of AI tool |
| `--background` | false | Run detached tmux session |
| `--tmux` | true | Use tmux (always true) |
| `--container` | | Use existing container |
| `--tool` | config | Override AI tool |

Calls: `session.Resolve` → `session.Setup` → tmux launch → `session.Cleanup`

### run

Run a command in an ephemeral container.

```
clincus run [flags] -- <command>
```

| Flag | Default | Description |
|------|---------|-------------|
| `--capture` | false | Capture output instead of streaming |
| `--timeout` | 120 | Command timeout (seconds) |
| `--format` | pretty | Output format (pretty\|json) |

### list

List active containers and saved sessions.

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | false | Include saved sessions |
| `--format` | text | Output format (text\|json) |

### attach

Attach to a running session. Auto-attaches if only one session active.

```
clincus attach [container-name] [flags]
```

| Flag | Description |
|------|-------------|
| `--bash` | Attach to bash instead of tmux |
| `--slot` | Slot number (requires workspace context) |

### info

Show session details: ID, container, data size, resume command.

```
clincus info [session-id]
```

### build

Build Incus image for AI coding sessions.

```
clincus build [--force]
clincus build custom <name> --script <path> [--base clincus] [--force]
```

### image

Image management subcommands:

| Subcommand | Description |
|------------|-------------|
| `image list [--all] [--prefix X]` | List images (table\|json) |
| `image publish <container> <alias>` | Publish container as image |
| `image delete <alias>` | Delete image |
| `image exists <alias>` | Exit 0 if exists |
| `image cleanup <prefix> --keep N` | Delete old versioned images |

### container

Low-level container operations:

| Subcommand | Description |
|------------|-------------|
| `container launch <image> <name> [--ephemeral]` | Create container |
| `container start <name>` | Start container |
| `container stop <name> [--force]` | Stop container |
| `container delete <name> [--force]` | Delete container |
| `container exec <name> -- <cmd> [--user N --group N --env K=V --cwd X --capture --tty]` | Execute command |
| `container exists <name>` | Exit 0 if exists |
| `container running <name>` | Exit 0 if running |
| `container mount <name> <device> <source> <path> [--shift --readonly]` | Add mount |
| `container list [--format text\|json]` | List containers |

Note: `--tty` and `--capture` are mutually exclusive on `container exec`.

### file

File transfer operations:

| Subcommand | Description |
|------------|-------------|
| `file push <local> <container>:<remote> [-r]` | Upload file/dir |
| `file pull <container>:<remote> <local> [-r]` | Download file/dir |

### snapshot

Container snapshot management. Container auto-detected from workspace unless `--container` specified.

| Subcommand | Description |
|------------|-------------|
| `snapshot create [name] [--stateful]` | Create snapshot |
| `snapshot list [--all] [--format text\|json]` | List snapshots |
| `snapshot restore <name> [--stateful --force]` | Restore (container must be stopped) |
| `snapshot delete [name] [--all --force]` | Delete snapshot(s) |
| `snapshot info <name> [--format text\|json]` | Snapshot details |

Container resolution: `--container` flag → `CLINCUS_CONTAINER` env → workspace auto-detect.

### tmux

Interact with tmux sessions in containers:

| Subcommand | Description |
|------------|-------------|
| `tmux send <session> <command>` | Send command to session |
| `tmux capture <session>` | Capture pane output |
| `tmux list` | List active tmux sessions |

### clean

Cleanup resources.

| Flag | Description |
|------|-------------|
| `--all` | Clean containers + sessions + orphans |
| `--sessions` | Clean saved session data |
| `--orphans` | Clean orphaned stopped containers |
| `--force` | Skip confirmation |
| `--dry-run` | Show what would be cleaned |

### kill

Force stop and delete containers immediately.

```
clincus kill [container...] [--all --force]
```

### shutdown

Graceful stop and delete with timeout.

```
clincus shutdown [container...] [--all --force --timeout 60]
```

### resume

Resume frozen (paused) containers.

```
clincus resume [container-name]
```

### persist

Convert ephemeral sessions to persistent.

```
clincus persist [container...] [--all --force]
```

### serve

Start web dashboard.

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 3000 | Listen port |
| `--open` | false | Open browser |

### health

Check system health. Exit codes: 0=healthy, 1=degraded, 2=unhealthy.

| Flag | Description |
|------|-------------|
| `--format` | text\|json |
| `--verbose` | Include additional checks |

Checks: OS, Incus, permissions, image, storage, disk space, config, tool, DNS, sudo.

### service

Manage systemd user service:

| Subcommand | Description |
|------------|-------------|
| `service install` | Write unit file, enable |
| `service start` | Start service |
| `service stop` | Stop service |
| `service remove` | Stop, disable, remove unit |

Unit path: `~/.config/systemd/user/clincus.service`

### version

Print version info: `clincus {Version} (commit: {Commit}, built: {Date})`
