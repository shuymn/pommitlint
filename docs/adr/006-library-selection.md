# ADR-006: Use Cobra for the CLI and keep other runtime dependencies narrow

## Status

Accepted

## Context

The v1 product scope is deliberately small:

- single Go binary at runtime,
- fixed compatibility target of `@commitlint/config-conventional`,
- four CLI subcommands,
- deterministic behavior in hooks and CI.

A broad dependency stack would increase review surface, transitive risk, release coordination, and long-term maintenance cost without adding commensurate value. At the same time, the CLI surface is the most likely area to grow after v1, so deferring command-structure choices has its own cost. The main question is where a third-party library materially reduces future rework versus where the standard library is sufficient.

Current candidate areas:

- Go CLI parsing and subcommand wiring,
- UTF-16 length accounting and parsing helpers,
- test diff ergonomics,
- maintainer-only preset sync.

## Decision

Adopt the following library posture for v1.

### Runtime Go code

- Use `github.com/spf13/cobra` for CLI parsing, subcommand dispatch, help generation, and flag validation.
- Use the Go standard library for the rest of the runtime.
  - `os`, `io`, `bufio`, `bytes`, `strings`, `path/filepath`, `errors` for I/O and normalization
  - `regexp` for parser and ignore matching
  - `encoding/json` and `embed` for preset/report handling
  - `unicode/utf16` for JS-like UTF-16 code unit counting
  - `os/exec` for Git interaction in hook path resolution where needed
- Do not add additional runtime convenience layers such as Viper or Afero in v1.

### Go tests

- Allow `github.com/google/go-cmp/cmp` as the single preferred third-party test helper for readable diffs in parser, report, and preset normalization tests.
- Prefer the standard library everywhere else, including `testing`, `testing/fstest`, and fuzzing support.
- Do not add `testify` in v1.

### Maintainer-only sync tooling

- Use Bun as the JS runtime.
- Depend on:
  - `@commitlint/load`
  - `@commitlint/config-conventional`
  - `conventional-changelog-conventionalcommits`
- Avoid adding schema/validation/helper packages unless the sync script hits a concrete maintainability blocker.

## Rejected Alternatives

- Use `alecthomas/kong` for the CLI.
  - Rejected because its declarative model is attractive, but Cobra is the lower-risk choice for a subcommand-oriented CLI with expected future expansion, broader ecosystem familiarity, and established command/help conventions. Kong's repository documents a stable 1.0.0 release, but it offers less value than Cobra for this command tree. [GitHub](https://github.com/alecthomas/kong)
- Use only the standard library `flag` package for the CLI.
  - Rejected because the user explicitly wants to avoid revisiting CLI structure later, and Cobra lets the project establish subcommand/help/validation patterns once instead of rewriting hand-rolled dispatch as the CLI grows. Cobra is current and mature, with v1.10.2 released on December 4, 2025. [GitHub](https://github.com/spf13/cobra/releases)
- Use only `reflect.DeepEqual` and standard diffs in tests.
  - Rejected because parser/report fixtures will benefit from clearer semantic diffs, and `cmp` improves failure readability without affecting production binaries. [pkg.go.dev](https://pkg.go.dev/github.com/google/go-cmp/cmp)

## Consequence

- Runtime implementation stays easy to audit and cheap to ship outside the CLI layer.
- CLI help, subcommand wiring, and input validation will follow Cobra's conventions instead of a hand-rolled dispatcher.
- Test code may use `cmp.Diff` where failure readability matters, but the production module graph remains minimal.
- Sync tooling carries JS dependencies, but they remain quarantined to maintainer workflows and aligned with the already accepted sync/runtime boundary.

## Revisit trigger

- The CLI grows materially beyond the current four-command contract.
- Runtime configuration needs become complex enough that hand-rolled `flag` parsing becomes a maintenance burden.
- Test fixtures become complex enough that additional test helpers provide clear closure value.
