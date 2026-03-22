# Tools

Clincus supports multiple AI coding tools. The active tool is determined by the `--tool` flag,
the project config (`.clincus.toml`), or the user config (`~/.config/clincus/config.toml`).

---

## Supported Tools

| Tool name | Description | Config mechanism |
|-----------|-------------|-----------------|
| `claude` | [Claude Code](https://docs.anthropic.com/en/docs/claude-code) | `~/.claude/` directory |
| `copilot` | [GitHub Copilot CLI](https://gh.io/copilot) | `~/.copilot/` directory |
| `opencode` | [opencode](https://github.com/nicholasgasior/opencode) | `~/.opencode.json` file |

---

## Selecting a Tool

### Per-session (CLI flag)

```bash
clincus shell --tool claude ~/my-project
clincus shell --tool opencode ~/my-project
clincus shell --tool copilot ~/my-project
```

### Per-project (`.clincus.toml`)

Create `.clincus.toml` in your project directory:

```toml
[tool]
name = "opencode"
```

This overrides the user-level default for any session started from that directory.

### User default (`~/.config/clincus/config.toml`)

```toml
[tool]
name = "claude"
```

If no tool is specified anywhere, `claude` is the built-in default.

---

## Claude Code

Claude Code stores its credentials and conversation history in `~/.claude/`. Clincus copies
this directory into the container at session start and restores it on exit, preserving your
API key and all conversation history.

### Effort Level

Control Claude's thinking effort (affects token usage and response quality):

```toml
# .clincus.toml or ~/.config/clincus/config.toml
[tool.claude]
effort_level = "high"   # "low", "medium" (default), or "high"
```

### Custom Binary

If you have a non-standard Claude Code installation:

```toml
[tool]
name = "claude"
binary = "/usr/local/bin/claude-custom"
```

### GitHub CLI (`gh`) Authentication

Clincus auto-detects your GitHub token and injects it as `GH_TOKEN` into Claude
containers, so the `gh` CLI works out of the box. If `gh` is authenticated on
your host, Claude sessions can create PRs, check issues, and use any `gh` command.

You can also provide a token explicitly via `--env` (overrides auto-detection):

```bash
clincus shell --env GH_TOKEN=$GH_TOKEN ~/my-project
```

Token resolution order:

1. `--env GH_TOKEN=...` (explicit, highest priority)
2. `GH_TOKEN` environment variable
3. `GITHUB_TOKEN` environment variable
4. `gh auth token` output (automatic)

---

## opencode

opencode stores its configuration in a single file (`~/.opencode.json`). Clincus mounts this
file directly into the container so your API credentials and settings are available.

opencode also stores session data in the workspace directory itself (`.opencode/` folder),
which means sessions persist naturally alongside your project — no special Clincus session
tracking is needed.

```toml
[tool]
name = "opencode"
```

When resuming an opencode session, Clincus detects the `.opencode/` directory in your
workspace and passes it through automatically.

---

## GitHub Copilot CLI

GitHub Copilot CLI stores its credentials and configuration in `~/.copilot/`. Clincus copies
this directory into the container at session start, preserving your authentication and settings.

```toml
[tool]
name = "copilot"
```

### Authentication

Clincus auto-detects your GitHub token from `gh auth token` and injects it as `GH_TOKEN`
into the container. If you have `gh` CLI authenticated on the host, copilot sessions
authenticate automatically — no extra flags needed.

You can also provide a token explicitly via `--env` (overrides auto-detection):

```bash
clincus shell --tool copilot --env GH_TOKEN=$GH_TOKEN ~/my-project
```

Token resolution order:
1. `--env GH_TOKEN=...` (explicit, highest priority)
2. `GH_TOKEN` environment variable
3. `GITHUB_TOKEN` environment variable
4. `gh auth token` output (automatic)

### Configuration Files

Clincus copies these files from `~/.copilot/` into the container:

- `config.json` — main configuration
- `mcp-config.json` — MCP server configuration
- `agents/` — custom agent definitions

---

## Custom Tools

You can run any command-line tool in an Incus container by configuring a custom binary name.
For a tool that does not match any built-in tool name, Clincus falls back to running the
binary name directly. Session history and config copying will be best-effort.

```toml
[tool]
name = "cursor"
binary = "cursor-headless"
```

### Aider Example

[Aider](https://aider.chat) can be used as a custom tool. It uses environment variables for
credentials (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.). Pass them with `--env`:

```bash
clincus shell --tool aider --env ANTHROPIC_API_KEY=sk-ant-... ~/my-project
```

Or set it in your project config:

```toml
# .clincus.toml
[tool]
name = "aider"
binary = "aider"
```

Then pass the API key via `--env`:

```bash
clincus shell --env ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY
```

---

## Tool Profiles

Use named profiles to switch tool configurations quickly:

```toml
# ~/.config/clincus/config.toml
[profiles.aider-gpt4]
[profiles.aider-gpt4.limits]
memory = { limit = "4GiB" }

[profiles.claude-fast]
[profiles.claude-fast]
# reuse existing tool config, just override defaults
```

Apply a profile:

```bash
clincus shell --profile aider-gpt4
```

---

## Verifying Tool Configuration

Check which tool is configured and whether its credentials are present:

```bash
clincus health
```

The health check reports the active tool and whether it appears to be properly configured.
