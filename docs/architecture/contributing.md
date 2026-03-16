# Contributing

Thank you for your interest in contributing to Clincus. This guide covers the development
setup, testing, code style, and pull request process.

---

## Development Setup

### Prerequisites

- **Go 1.24+** — `go version`
- **Node.js 20+** — `node --version`
- **npm** — `npm --version`
- **Incus** — see [Prerequisites](../getting-started/prerequisites.md)
- **golangci-lint** — `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **pre-commit** — `pip install pre-commit` or `brew install pre-commit`

### Clone and Build

```bash
git clone https://github.com/bketelsen/clincus.git
cd clincus

# Install pre-commit hooks
pre-commit install

# Build everything (Go binary + Svelte frontend)
make build

# Install to $GOPATH/bin
make install
```

### Frontend Development

The Svelte app lives in `web/`. For hot-reloading during UI development:

```bash
cd web
npm install
npm run dev
```

The dev server proxies API requests to `http://127.0.0.1:3000` (assumes `clincus serve` is
running).

To build the frontend for embedding:

```bash
make web
# or
cd web && npm run build
```

---

## Running Tests

### Unit Tests

```bash
go test ./...

# With race detector (recommended before submitting a PR)
go test -race ./...

# Verbose output
go test -race -v ./...
```

### Integration Tests

Integration tests require a running Incus instance and are skipped automatically when Incus
is not available. They are marked with `//go:build integration` or detected via
`container.Available()`.

```bash
go test -race -v ./... -run Integration
```

### Python Tests (CLI integration)

The `tests/` directory contains pytest-based integration tests that exercise the `clincus`
binary end-to-end.

```bash
pip install -e .      # installs test dependencies from pyproject.toml
pytest tests/
```

These tests use a `dummy` stub tool (instead of the real Claude Code) to run quickly without
requiring an API key. Set `CLINCUS_USE_DUMMY=1` to use the dummy in any test run.

---

## Code Style

### Go

- Format with `gofmt`: `make fmt`
- Lint with golangci-lint: `make lint`
- Follow standard Go idioms and the project's existing patterns
- All exported symbols must have doc comments
- Avoid `//nolint` unless the lint rule is genuinely wrong; add a comment explaining why

### TypeScript / Svelte

- The frontend uses the default ESLint and Prettier configuration from the Svelte template
- Run `npm run lint` and `npm run format` from the `web/` directory

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add --timeout flag to clincus run
fix: correctly handle SIGTERM during session cleanup
docs: document resource limit config options
refactor: extract session naming into session.Naming type
test: add integration test for persistent session resume
chore: bump golangci-lint to v1.62
```

---

## Adding a New Tool

1. Create `internal/tool/<toolname>.go` implementing the `tool.Tool` interface:

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "mytool" }
func (t *MyTool) ConfigDirName() string { return ".mytool" }
func (t *MyTool) BuildCommand(sessionID string, resume bool, cliSessionID string) []string {
    cmd := []string{"mytool"}
    if resume && cliSessionID != "" {
        cmd = append(cmd, "--session", cliSessionID)
    }
    return cmd
}
func (t *MyTool) DiscoverSessionID(statePath string) string {
    // read session ID from state files
    return ""
}
```

2. Register it in `internal/tool/registry.go`:

```go
func init() {
    Register("mytool", &MyTool{})
}
```

3. Add tests in `internal/tool/<toolname>_test.go`.

4. Document the tool in [docs/guides/tools.md](../guides/tools.md).

---

## Pull Request Process

1. **Open an issue first** for significant changes. For small bug fixes or doc improvements,
   a PR without a prior issue is fine.

2. **Fork and branch**:

```bash
git checkout -b feat/my-feature
```

3. **Make your changes** — keep commits focused and atomic.

4. **Run the full check suite** before pushing:

```bash
make fmt
make lint
make test
```

5. **Push and open a PR** against `main` on `github.com/bketelsen/clincus`.

6. **PR description** should:
   - Summarize what the change does and why
   - Reference any related issues (`Closes #123`)
   - List any manual testing you did

7. A maintainer will review. Address review comments and push new commits to the same branch
   (do not force-push after a review has started).

8. Once approved, the PR is squash-merged.

---

## Release Process

Releases are automated via GoReleaser and GitHub Actions. To cut a new release:

```bash
# Ensure the working tree is clean and tests pass
make bump
```

`make bump` runs `svu next` to determine the next semantic version, creates an annotated
git tag, and pushes it. The GitHub Actions release workflow picks up the tag and runs
GoReleaser to build binaries, create `.deb`/`.rpm`/`.apk` packages, and publish a GitHub
Release.

---

## Project Governance

Clincus is a community-maintained project. For questions, open a
[GitHub Discussion](https://github.com/bketelsen/clincus/discussions). For bugs, open a
[GitHub Issue](https://github.com/bketelsen/clincus/issues).
