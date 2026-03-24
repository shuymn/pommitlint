# ADR-007: Isolate Git-backed tests from user Git and GPG configuration

## Status

Accepted

## Context

`pommitlint` v1 needs Git-backed tests for `--edit`, `core.hooksPath`, and `hook install` scenarios. Some of those tests may create temporary commits to exercise real `commit-msg` behavior. On this machine, the global Git configuration sets `commit.gpgsign=true` in `/Users/shuymn/.gitconfig`. In that environment, an unisolated test can trigger GPG signing through 1Password or other interactive agents, which is both slow and unsafe for unattended test runs.

The risk is broader than GPG alone. Global or system Git config can also introduce:

- credential prompts,
- hooks path overrides,
- aliases,
- init templates,
- default branch changes,
- signing or tag policies unrelated to the repository under test.

Git-backed tests therefore need a hermetic execution envelope instead of relying on the developer's ambient environment.

## Decision

All Git-backed tests must run inside a temporary repository with an isolated Git environment and explicit no-sign defaults.

Required isolation rules:

- Use `t.TempDir()` to create a dedicated repository root, HOME-like directory, hooks directory, and GNUPGHOME.
- Never run Git-mutating test commands in the real repository.
- Set Git process environment for test subprocesses to ignore ambient config:
  - `HOME=<temp-home>`
  - `XDG_CONFIG_HOME=<temp-xdg>`
  - `GNUPGHOME=<temp-gnupg>`
  - `GIT_CONFIG_NOSYSTEM=1`
  - `GIT_CONFIG_GLOBAL=/dev/null`
  - `GIT_TERMINAL_PROMPT=0`
  - `GIT_ASKPASS=/bin/false`
  - `SSH_ASKPASS=/bin/false`
  - `GCM_INTERACTIVE=Never`
- Initialize repositories with explicit config instead of inheriting defaults:
  - `git -c init.defaultBranch=main init`
  - local `user.name` and `user.email`
  - local `core.hooksPath` only when the scenario needs it
- Any test command that may create a commit or tag must explicitly disable signing:
  - `git -c commit.gpgsign=false -c tag.gpgsign=false ...`
- Tests must use an empty temp hooks directory unless the scenario is specifically verifying hook installation or execution.
- Prefer direct file-based assertions for `COMMIT_EDITMSG` behavior; require real commits only when the acceptance evidence depends on Git's actual hook flow.

Implementation guidance:

- Create a shared test helper that builds the isolated environment and shells out to Git with the required env/config.
- Keep the helper narrow and replayable: repo init, file write, hook path setup, and commit execution are enough for v1.

## Rejected Alternatives

- Rely on `git commit --no-gpg-sign` or `--no-verify` ad hoc in individual tests.
  - Rejected because it is easy to miss in one path, does not isolate global config broadly enough, and leaves other prompt vectors in place.
- Stub Git entirely and avoid subprocess tests.
  - Rejected because `core.hooksPath`, `commit-msg`, and edit-file behavior are public boundary contracts that need real Git evidence.
- Assume CI is clean and only local developers will see prompts.
  - Rejected because local unattended test runs are part of the acceptance target, and the current environment already proves the risk is real.

## Consequence

- Git-backed tests will be slightly more verbose to set up, but they will be deterministic and safe to run unattended.
- The test helper becomes the single place to enforce no-sign/no-prompt guarantees.
- Future Git scenarios must use the helper instead of invoking ambient `git` directly.
- The repository can add commit-flow scenario tests without risking 1Password or other interactive signing prompts.

## Revisit trigger

- The project adds Git behaviors that cannot be exercised through the isolated helper model.
- A future Git version changes the relevant environment/config precedence and weakens these guarantees.
- The repository adopts a different strategy for Git-backed contract testing.
