# ADR-002: Lefthook for git hooks

## Status

Accepted

## Context

Git hook management options:

- **Manual scripts in `.git/hooks/`**: No dependency, but not shareable via version control.
- **pre-commit (Python)**: Large ecosystem, but requires Python runtime.
- **Husky (Node.js)**: Popular in JS ecosystems, but requires Node.js runtime.
- **Lefthook (Go)**: Fast, single binary, YAML config, no runtime dependency.

## Decision

Use Lefthook for git hook management.

Reasons:

- **No runtime dependency**: Single binary, fits naturally in a Go project.
- **YAML configuration**: Declarative, version-controlled hook definitions.
- **`piped` execution**: Hooks run sequentially, failing fast on first error.
- **`stage_fixed: true`**: Auto-stages formatter fixes.
- **`glob` filtering**: Hooks only run when relevant files change.
- **Skips on merge/rebase**: Avoids blocking non-code operations.

## Consequences

- `lefthook` is pinned in `mise.toml`. `lefthook install` is enforced via `mise run check`.
- Hook configuration lives in `lefthook.yml` at the repository root.
- Hook commands delegate to mise tasks (`mise run`), or run mise-pinned tools directly with `mise exec` when no task wraps them.
