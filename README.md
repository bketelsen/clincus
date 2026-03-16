# Clincus

Secure and fast container runtime for AI coding tools on Linux.

Run [Claude Code](https://docs.anthropic.com/en/docs/claude-code),
[opencode](https://github.com/nicholasgasior/opencode),
[Aider](https://aider.chat), and other AI assistants
in isolated [Incus](https://linuxcontainers.org/incus/) containers
with session persistence, a web dashboard, and resource limits.

## Features

- **Container isolation** — each session runs in its own Incus container
- **Session persistence** — save and resume AI conversations with full history
- **Web dashboard** — launch and manage sessions from your browser
- **Multi-tool support** — Claude Code, opencode, Aider, and custom tools
- **Workspace mounting** — project files mounted in isolated containers
- **Snapshots** — checkpoint and rollback container state
- **Resource limits** — CPU, memory, and time limits per session
- **File transfer** — push/pull files between host and containers

## Quick Start

### Install

Download the latest release from [GitHub Releases](https://github.com/bketelsen/clincus/releases)
or build from source:

```bash
git clone https://github.com/bketelsen/clincus.git
cd clincus
make install
```

### First Session

```bash
# Build the container image (one-time setup)
clincus build

# Start a Claude Code session
clincus shell --tool claude ~/my-project
```

## Documentation

Full documentation is available at [bketelsen.github.io/clincus](https://bketelsen.github.io/clincus).

## Attribution

Clincus is derived from [code-on-incus](https://github.com/mensfeld/code-on-incus)
by [Maciej Mensfeld](https://github.com/mensfeld). The web dashboard was inspired by
[wingthing](https://github.com/ehrlich-b/wingthing) by [ehrlich-b](https://github.com/ehrlich-b).

## License

MIT — see [LICENSE](LICENSE) for details.
