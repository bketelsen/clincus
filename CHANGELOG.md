# Changelog

All notable changes to this project will be documented in this file.

## [0.4.0] - 2026-03-17

### Added

- Set up squad team for clincus project
- **cli:** Add charmbracelet/fang for styled CLI output
- **cli:** Upgrade to fang v2, use built-in completions and manpages

### CI

- Add workflow_dispatch trigger to docs workflow

### Documentation

- Document architectural decisions in squad decisions log

### Other

- Add a squad
- Makefile add check

### Styling

- Fix import ordering in root.go

## [0.3.0] - 2026-03-17

### Added

- Add GitHub Copilot CLI as supported tool

### CI

- Gate release on CI passing via reusable workflow
- Auto-generate CHANGELOG.md from conventional commits via git-cliff

### Documentation

- Require feature branch and PR workflow in CLAUDE.md

### Fixed

- **ci:** Pin golangci-lint to v2.11 for lint action compatibility
- Use in-container mount for tmpfs instead of Incus device API

## [0.2.0] - 2026-03-17

### Added

- **web:** Improve session tiles, refresh on lifecycle, rename to Projects

### Documentation

- Add dashboard and session screenshots
- Add TODO list
- Add CLAUDE.md with project conventions and doc requirements

### Fixed

- Restore .gitkeep after web build in Makefile

### Other

- Todo list
- Todo list
- Todo list
- Todo list

## [0.1.0] - 2026-03-17

### Build

- Add Makefile with build, test, release targets
- Add GoReleaser Pro configuration

### CI

- Add GitHub Actions workflows for CI, release, and docs

### Changed

- Remove network and monitoring config, rename coi paths to clincus
- Remove network/bedrock coupling from session, rename coi to clincus
- Remove network/monitor coupling from all CLI commands
- Remove ~750 lines of network/monitoring health checks
- Remove firewall/veth cleanup from orphan detection
- Remove network coupling from image builder, rename to clincus
- Remove network config, rename coi to clincus in server
- Rename Go module to github.com/bketelsen/clincus
- Rename all coi/COI string references to clincus
- Rename coi references in Python integration tests
- Rebrand web frontend from COI to Clincus

### Documentation

- Add README with quick start and attribution
- Set up MkDocs Material site structure
- Write getting started guides
- Write user guides
- Write reference docs
- Write architecture overview and contributing guide
- Add initial changelog for v0.1.0

### Fixed

- Resolve build/test/lint issues found in full verification
- Install to ~/.local/bin instead of GOPATH
- Use golangci-lint v2 action and format Python tests with ruff
- Remove invalid 'publish' field from goreleaser nightly config
- Reorder goreleaser hooks (web before completions) and remove deprecated builds keys
- Gitignore pattern was hiding cmd/clincus/ directory
- Use git changelog instead of github API (no previous tag)

### Other

- Initialize clincus repository
- Copy source files from code-on-incus
- Delete monitor CLI subcommand
- Update .gitignore for completions, manpages, site, and webui dist

[0.4.0]: https://github.com/bketelsen/clincus/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/bketelsen/clincus/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/bketelsen/clincus/compare/v0.1.0...v0.2.0


## Attribution

Derived from [code-on-incus](https://github.com/mensfeld/code-on-incus) by Maciej Mensfeld.
Web dashboard inspired by [wingthing](https://github.com/ehrlich-b/wingthing) by ehrlich-b.
