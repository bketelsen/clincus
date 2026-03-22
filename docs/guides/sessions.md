# Sessions

A session is a pairing of a workspace directory with an Incus container running an AI coding
tool. Sessions can be ephemeral (container deleted on exit) or persistent (container kept for
reuse), and Clincus saves conversation history so you can resume where you left off.

---

## Session Lifecycle

```
clincus shell
    │
    ├─ Allocate slot (0 = auto, 1–10 explicit)
    ├─ Generate session ID and container name
    ├─ Launch Incus container from image
    ├─ Mount workspace into container
    ├─ Copy tool config (e.g., ~/.claude/) into container
    ├─ Start AI tool inside tmux session
    ├─ Attach terminal to tmux
    │
    [user works...]
    │
    ├─ User exits AI tool or detaches (Ctrl+B d)
    ├─ Clincus saves session data back to ~/.clincus/
    └─ Container is cleaned up (ephemeral) or stopped (persistent)
```

---

## Slots

Clincus uses "slots" to run multiple sessions for the same workspace in parallel. Each slot
gets its own container. The default slot is auto-allocated (the lowest available number 1–10).

```bash
clincus shell                  # auto-allocate slot
clincus shell --slot 2         # use slot 2 explicitly
clincus shell --slot 3 --workspace ~/other-project
```

Container names encode the workspace hash and slot:

```
clincus-a1b2c3-1    # workspace hash a1b2c3, slot 1
clincus-a1b2c3-2    # same workspace, slot 2
clincus-d4e5f6-1    # different workspace, slot 1
```

---

## Ephemeral vs Persistent Sessions

By default, containers are **ephemeral**: they are deleted when the session ends. This is the
most secure and resource-efficient mode for short-lived tasks.

**Persistent** containers survive across sessions. The container is stopped (not deleted) on
exit and restarted on the next `clincus shell` for the same workspace and slot. Persistent mode
is useful when the tool stores state inside the container (e.g., installed packages, build
caches).

```bash
# Ephemeral (default)
clincus shell

# Persistent
clincus shell --persistent

# Set persistent as default in config
# ~/.config/clincus/config.toml
# [defaults]
# persistent = true
```

---

## Session Persistence and History

Regardless of ephemeral/persistent mode, Clincus saves your conversation history to disk so
you can resume it in a future session.

Session data is stored in `~/.clincus/sessions-<tool>/`:

```
~/.clincus/
  sessions-claude/
    <session-id>/
      .claude/          # Claude's config and conversation history
      metadata.json     # session metadata (workspace, timestamp, etc.)
  sessions-opencode/
    <session-id>/
      ...
```

### Resume a Session

```bash
# Auto-detect the latest session for the current workspace
clincus shell --resume

# Resume a specific session by ID
clincus shell --resume=<session-id>

# --continue is an alias for --resume
clincus shell --continue
```

List saved sessions to find IDs:

```bash
clincus list --all
```

---

## Attaching to Running Sessions

```bash
# Auto-attach (if only one session is running)
clincus attach

# Attach to a specific container
clincus attach clincus-a1b2c3-1

# Attach to slot 1 for the current workspace
clincus attach --slot 1

# Get a bash shell instead of tmux
clincus attach --bash
clincus attach clincus-a1b2c3-1 --bash
```

---

## Background Sessions

Run the AI tool in the background (detached from your terminal):

```bash
clincus shell --background
```

The session runs in a detached tmux session inside the container. Capture its output or send
commands with the `tmux` subcommand:

```bash
# View output of background session
clincus tmux capture clincus-a1b2c3-1

# Send a command to the background session
clincus tmux send clincus-a1b2c3-1 "fix the tests"
```

---

## Listing Sessions

```bash
# Active containers only
clincus list

# Active containers + saved session history
clincus list --all

# JSON output for scripting
clincus list --format json
```

---

## Killing and Shutting Down Sessions

```bash
# Force-delete a container
clincus kill clincus-a1b2c3-1

# Graceful shutdown (from inside the container)
# Run this in the container shell:
sudo shutdown 0
```

The `clincus kill` command forcibly deletes the Incus container and performs cleanup.

---

## Cleaning Up Saved Sessions

Remove old session data from disk:

```bash
clincus clean
```

This removes orphaned session directories (sessions whose containers no longer exist).

---

## Multi-Slot Workflows

Run two AI tools on the same project simultaneously:

```bash
# Terminal 1: Claude Code for architecture work
clincus shell --slot 1 --tool claude ~/my-project

# Terminal 2: opencode for quick edits
clincus shell --slot 2 --tool opencode ~/my-project
```

Both sessions share the same workspace directory on the host; each has its own isolated
container with its own copies of tool config, credentials, and state.
