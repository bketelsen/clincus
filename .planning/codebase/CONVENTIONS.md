# Coding Conventions

**Analysis Date:** 2026-03-17

## Naming Patterns

**Files:**
- Go source files: lowercase with underscores (`config.go`, `config_test.go`, `config_loader.go`)
- Test files: `{name}_test.go` (unit tests) or `{name}_integration_test.go` (integration tests)
- TypeScript files: PascalCase for components (`SessionCard.svelte`, `LaunchDialog.svelte`), camelCase for utilities (`api.ts`, `types.ts`)
- Svelte stores: `{name}.svelte.ts` suffix (e.g., `sessions.svelte.ts`)

**Functions:**
- Go: PascalCase for exported functions (`GetDefaultConfig`, `ExpandPath`, `GenerateSessionID`), camelCase for unexported (`loadConfigFile`, `setupMounts`)
- TypeScript: camelCase for all functions (`listSessions`, `stopSession`, `loadSessions`)

**Variables:**
- Go: PascalCase for exported, camelCase for unexported; standard abbreviations like `err`, `mgr`, `cfg` are common
- TypeScript: camelCase for all (`sessionID`, `containerName`, `workspacePath`)

**Types:**
- Go: PascalCase struct names (`Config`, `DefaultsConfig`, `SecurityConfig`, `IncusConfig`)
- Go: Add comment above exported type: `// Config represents the complete configuration`
- TypeScript: PascalCase interfaces (`Session`, `Workspace`, `HistoryEntry`, `ClincusConfig`)

**Constants:**
- Go: PascalCase for exported (`DefaultImage`, `ClincusImage`), SCREAMING_SNAKE_CASE for sentinel values (e.g., in comments)
- Env var prefix: `CLINCUS_` (e.g., `CLINCUS_IMAGE`, `CLINCUS_SESSIONS_DIR`, `CLINCUS_LIMIT_CPU`)

## Code Style

**Formatting:**
- Go: Uses `gofmt` (simplified with `-s` flag) and `gofumpt`
- Go imports organized automatically by `goimports`
- TypeScript: No Prettier/ESLint config in repo; uses ESNext target with strict type checking

**Linting:**
- Go: `golangci-lint` v2 config in `.golangci.yml`
- Enabled linters: `bodyclose`, `copyloopvar`, `dupl`, `errname`, `gocyclo`, `gosec`, `misspell`, `predeclared`, `unconvert`, `wastedassign`, `whitespace`
- Revive rules enable: `exported`, `package-comments`, `error-return`, `error-strings`, `error-naming`, `if-return`, etc.
- Max complexity: 50 (cyclop), 90 (gocognit)
- Duplication threshold: 200 lines
- Test files excluded from complexity/dupl/errcheck linters in `.golangci.yml`

## Import Organization

**Go Order:**
1. Standard library (`fmt`, `os`, `path/filepath`)
2. Third-party packages (`github.com/BurntSushi/toml`)
3. Internal packages (`github.com/bketelsen/clincus/internal/...`)

Organized automatically by `goimports`. Standard library imports from `fmt` are dot-imported (`. "fmt"`) in some contexts.

**TypeScript/JavaScript:**
1. Standard/built-in imports
2. Third-party packages (`@xterm/xterm`, `svelte`)
3. Internal modules (`../lib/types`, `../lib/api`)
4. Relative imports (same-level files)

**Path Aliases:**
- Go: Internal imports use full module path: `github.com/bketelsen/clincus/internal/...`
- TypeScript: Relative paths preferred; no path alias configuration in tsconfig.json

## Error Handling

**Patterns:**
- Go: Explicit error handling with `if err != nil` checks
- Go: Return errors using `fmt.Errorf("context: %w", err)` for error wrapping
- Go: Errors are returned as second return value
- Go: For recovery operations (like defer cleanup), use `_ = mgr.Stop(true)` to ignore error if operation is best-effort
- TypeScript: Try-catch blocks around API calls; missing error details often lead to fallback refresh (e.g., `await loadSessions()`)
- TypeScript: Functions throw errors on failed API calls (not returning null/undefined)

**Error Messages:**
- Go: Start with context, use lowercase (e.g., "failed to load config from %s")
- Go: Use `: %w` for wrapped errors to preserve stack trace
- TypeScript: Async functions throw on API failure; components catch and handle

## Logging

**Framework:** `fmt.Sprintf` for formatted logging in Go; `console.*` not heavily used in TypeScript

**Patterns:**
- Go: Pass logger functions around (e.g., `func setupMounts(..., logger func(string))`) rather than global logger
- Go: Log informational messages via the passed logger function: `logger(fmt.Sprintf("Adding mount: %s -> %s", ...))`
- TypeScript: Minimal logging; errors are thrown and caught by components

## Comments

**When to Comment:**
- Go: Export every exported type and function with a comment starting with the name: `// Config represents the complete configuration`
- Go: Complex logic gets inline comments explaining why, not what (e.g., "Lima uses virtiofs for mounting; this is detected to avoid UID shift issues")
- Go: Package-level comments required by `revive` linter rule

**JSDoc/TSDoc:**
- Not heavily used; TypeScript interfaces are self-documenting
- Go comments are single-line English sentences starting with the name of the item

## Function Design

**Size:**
- Functions typically 20-50 lines; larger functions (100+ lines) are broken into helpers
- Test functions are often longer (100-200 lines) if testing multiple scenarios with setup/cleanup

**Parameters:**
- Go: Use receivers for methods (e.g., `(m *Manager) Stop(force bool) error`)
- Go: Pointers for struct receivers; avoid value receivers for large structs
- TypeScript: Use object destructuring for props in Svelte: `let { session }: { session: Session } = $props();`

**Return Values:**
- Go: Multiple return values; last is always error: `func(...) (T, error)`
- Go: Exported functions return named error, not nil success
- TypeScript: Async functions return Promise; error is thrown if API fails

## Module Design

**Exports:**
- Go: Use PascalCase for all exported identifiers
- Go: Internal packages live in `internal/` and are not importable by external code
- Go: Keep exports minimal; unexported helpers inside packages

**Barrel Files:**
- TypeScript: No barrel index files used; direct imports preferred (e.g., `import { api } from '../lib/api'`)
- TypeScript: Stores are functions exported from `*.svelte.ts` files, not classes

**Package Organization:**
- Go: One responsibility per package (e.g., `config` for config, `session` for session lifecycle, `container` for Incus operations)
- Go: Shared interfaces/types in main package file (e.g., `config.go`)
- TypeScript: Components in `web/src/components/`, utilities in `web/src/lib/`, stores in `web/src/stores/`

## Type System (Go)

**Pointers:**
- Use pointers for struct receivers (methods on structs)
- Use pointers for optional fields (e.g., `WritableHooks *bool`)
- Use value types for function returns unless mutation needed

**Nil Checks:**
- Always check `if err != nil` immediately after operations
- Use `if cfg == nil` for defensive checks
- Pointer fields must be nil-checked: `if base.Git.WritableHooks == nil`

## Type System (TypeScript)

**Interfaces:**
- All data structures defined as interfaces (e.g., `Session`, `Workspace`, `Config`)
- Interfaces match Go struct field names exactly (e.g., `code_uid` in both)
- Use `Record<string, T>` for maps (e.g., `Record<string, ProfileConfig>`)

**Nullability:**
- Use `null` explicitly for optional values (e.g., `writable_hooks: boolean | null`)
- Prefer optional fields (`field?: Type`) over `| null`

## Testing (Inline with Tests Section)

See TESTING.md for full test patterns and structure.

---

*Convention analysis: 2026-03-17*
