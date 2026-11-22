# ADR-011: mise for tools and tasks

## Status

Accepted. Supersedes [ADR-001](001-task-over-make.md).

## Context

Task ran the development commands, but tool versions were pinned in several places: golangci-lint in a Taskfile variable installed by a curl script, Go via `setup-go` in CI (`stable` in most jobs), Bun, oxlint, and oxfmt resolved from the developer machine or npm at run time. The owner's other repositories pin tools with mise and keep a lockfile that a scheduled workflow advances.

## Decision

Use mise for both tool versions and tasks.

- `mise.toml` declares tools by major version, project-local cache paths in `[env]` ([ADR-004](004-local-tool-cache.md)), and the tasks (same names as the former Taskfile tasks).
- `GOTOOLCHAIN=local` makes every run use the Go that `mise.lock` pins instead of a toolchain that `go.mod` selects.
- `mise.lock` pins exact versions with checksums for `macos-arm64` and `linux-x64`, enforced by `locked = true`.
- `.github/workflows/mise-update.yml` bumps the lockfile daily through a pull request; Renovate's mise manager is disabled.
- CI and Lefthook call `mise run` (or `mise exec`), so local runs, hooks, and CI use the same tool versions.

Reasons:

- **One pinning mechanism**: Tools, including Go, golangci-lint, Bun, oxlint, oxfmt, and the workflow and secret scanners, resolve from one lockfile instead of per-tool install scripts and CI actions.
- **Consistency across repositories**: Matches the owner's standard repository layout.
- **Fewer moving parts**: Removes the Task binary, the `setup-task` action, and the golangci-lint install task.

## Consequences

- `mise` must be available on PATH; `mise install` provisions every tool.
- `release.yml` still sets up Go from `go.mod` for GoReleaser.
