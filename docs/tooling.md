# Tooling Pipeline

Use this file only when the task explicitly points to repository-specific tooling flow.

## Source of Truth

- Use mise tasks (`mise run <task>`, list with `mise tasks`) as the single interface for local development, CI, and git hooks.
- `mise.toml` is the source of truth for task definitions, tool versions, and project-local cache paths under `.cache/`.
- `mise.toml` declares each tool by major version; `mise.lock` (`locked = true`) pins the exact versions. `.github/workflows/mise-update.yml` advances the lockfile daily; Renovate's mise manager is disabled.

## Hooks and CI

- `lefthook.yml` maps git hook events to `mise run` tasks, or to `mise exec -- <tool>` for a pinned tool without a task. Do not duplicate hook logic in shell scripts.
- Hooks run in `piped` mode, can auto-stage formatter fixes, and skip merge or rebase flows.
- CI installs tools with `jdx/mise-action` and runs the same `mise run` tasks used locally.

## Adding Tools

1. Add the tool to `[tools]` in `mise.toml` with a major-version range (a tool only one task needs goes in that task's `tools`).
2. Run `mise lock` and commit `mise.lock`.
3. Add a Lefthook entry only when the tool must run on commit or push.
