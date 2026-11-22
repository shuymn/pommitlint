# ADR-004: Project-local tool and build cache

## Status

Accepted

## Context

Go tooling uses several caches (`GOMODCACHE`, `GOCACHE`, `GOLANGCI_LINT_CACHE`). By default these are stored under `$GOPATH` or `$HOME/.cache`, shared across all projects on the machine.

## Decision

Override cache directories to project-local `.cache/` via `[env]` in `mise.toml`:

```toml
[env]
GOMODCACHE = "{{config_root}}/.cache/go-mod"
GOCACHE = "{{config_root}}/.cache/go-build"
GOLANGCI_LINT_CACHE = "{{config_root}}/.cache/golangci-lint"
```

Reasons:

- **Agent write permissions**: Coding agents may lack write access outside the repo root or `/tmp`. Project-local caches ensure `go build` and `go test` work in restricted environments.
- **Isolation**: Different projects do not interfere with each other's caches.
- **LLM/agent accessibility**: AI coding agents can inspect module source code within the project tree without navigating global paths.
- **Easy cleanup**: `rm -rf .cache/` resets all caches for the project.

## Consequences

- `.cache/` is gitignored.
- Cache isolation applies when running through mise (`mise run`, `mise exec`, or a shell with mise activated). Commands run without mise use global caches unless the env vars are set explicitly.
