# Quick Start

This guide walks through your first Clincus session from image build to interactive AI coding.

---

## 1. Build the Container Image

The clincus image bundles Node.js LTS, Claude Code CLI, Docker, GitHub CLI, tmux, and common
development tools. Build it once with:

```bash
clincus build
```

Expected output:

```
Building clincus image...
Launching container for setup...
Installing Node.js LTS...
Installing Claude CLI...
Installing GitHub CLI...
Saving image...

 Image 'clincus' built successfully!
  Version: clincus-20260101
  Fingerprint: abc123def456...
```

This step takes a few minutes the first time. Subsequent runs are fast because the image is
cached in Incus. Use `--force` to force a rebuild:

```bash
clincus build --force
```

---

## 2. Start Your First Session

Navigate to a project directory and run:

```bash
cd ~/my-project
clincus shell
```

Clincus will:

1. Allocate an available slot (e.g., slot 1)
2. Launch an Incus container from the `clincus` image
3. Mount `~/my-project` into the container at `/workspace`
4. Start Claude Code inside a tmux session
5. Attach your terminal to the tmux session

Expected output before the session starts:

```
Auto-allocated slot 1
Setting up session abc123...

Starting session...
Session ID: abc123
Container: clincus-a1b2c3-1
Workspace: /home/user/my-project
Mode: Interactive (tmux)

```

You are now inside the container. Claude Code starts in the tmux session. Use it as you
normally would — your project files are available at `/workspace`.

### Detach and Reattach

Press `Ctrl+B d` to detach from the tmux session. The container keeps running.

Reattach at any time:

```bash
clincus shell          # reattach to running session for current workspace
clincus attach         # same, or lists sessions if multiple are running
```

---

## 3. List Active Sessions

```bash
clincus list
```

Output:

```
Active Containers:
------------------
  clincus-a1b2c3-1 (ephemeral)
    Status: Running
    IPv4: 10.77.0.100
    Created: 2026-01-01 12:00:00
    Image: clincus (v20260101)
    Workspace: /home/user/my-project
```

Use `--all` to also see saved (stopped) sessions:

```bash
clincus list --all
```

---

## 4. Use a Different Tool

Clincus supports multiple AI coding tools. Override the tool for a session:

```bash
clincus shell --tool opencode ~/my-project
clincus shell --tool copilot ~/my-project
```

Or set a default in your project config (`.clincus.toml`):

```toml
[tool]
name = "opencode"
```

---

## 5. Open the Web Dashboard

```bash
clincus serve --open
```

The dashboard opens at `http://127.0.0.1:3000` in your browser. From here you can:

- See all running sessions
- Launch new sessions for any workspace
- Connect to a session's terminal in the browser
- Stop sessions

---

## 6. End a Session

When you exit Claude Code (type `/exit` or `Ctrl+D`), the container is cleaned up automatically.
Clincus saves your session history so you can resume later.

To force-stop a running session:

```bash
clincus kill clincus-a1b2c3-1
```

---

## Next Steps

- [Sessions guide](../guides/sessions.md) — persistence, multi-slot, and resuming
- [Workspaces guide](../guides/workspaces.md) — mounting and file transfer
- [CLI Reference](../reference/cli.md) — all commands and flags
- [Configuration](../reference/config.md) — project and user config options
