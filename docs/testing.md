# Testing Conventions

Use this file only when the task explicitly points to repository-specific test rules.

## Running Tests

- Use `task test` for the full suite and `task check` for CI-equivalent verification.
- Prefer `task` over raw `go test` so `GOMODCACHE`, `GOCACHE`, and `GOLANGCI_LINT_CACHE` stay inside `.cache/`.
- Use `go test -run TestName ./path/to/pkg` only for focused runs. If you bypass `task`, preserve the same cache environment.

## Suite Requirements

- The full suite must pass with `-race -shuffle=on -count=10`.
- Tests must be race-free, order-independent, and stable across repeats.
- Call `t.Parallel()` in tests and subtests by default. Skip it only when a shared side effect cannot be isolated.
- Never call `t.Fatal` or `t.FailNow` from a helper goroutine.

## Isolation

- Prefer test-scoped helpers: `t.TempDir()`, `t.Setenv()`, `t.Context()` (Go 1.24+), and `t.Cleanup()`.
- Prefer real resources at package boundaries (`httptest`, temp dirs, subprocesses) over deep mocks.
- Mock only external systems that cannot be run locally.

## Git-backed Tests

- Any test that invokes `git` must use a temporary repository and must not mutate the real workspace repository.
- Git-backed tests must isolate themselves from developer machine config and prompts.
  - Set `HOME`, `XDG_CONFIG_HOME`, and `GNUPGHOME` to temp directories.
  - Set `GIT_CONFIG_NOSYSTEM=1` and `GIT_CONFIG_GLOBAL=/dev/null`.
  - Set `GIT_TERMINAL_PROMPT=0`, `GIT_ASKPASS=/bin/false`, `SSH_ASKPASS=/bin/false`, and `GCM_INTERACTIVE=Never`.
- Any test path that can create a commit or tag must invoke Git with `-c commit.gpgsign=false -c tag.gpgsign=false`.
- Use an empty temp hooks directory by default so global or inherited hooks never run during tests.
- Prefer direct file-based assertions over real commits unless the acceptance contract specifically requires Git hook execution or `core.hooksPath` behavior.
- Centralize this setup in a shared test helper; do not hand-roll partial Git isolation in each test.

## Linter Exceptions

- `_test.go` files relax `exhaustruct`, `funlen`, `gocognit`, `noctx`, and `contextcheck`.
