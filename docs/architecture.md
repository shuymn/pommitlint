# Architecture Baseline

## Goal

Provide a Go CLI that enforces `@commitlint/config-conventional` equivalent lint behavior as a single binary, with no runtime dependency on Node.js, npm, Bun, or JS config execution.

## Constraints

- Runtime must work as a single Go binary in `commit-msg` hooks and CI.
- Compatibility target is fixed to `@commitlint/config-conventional`; full commitlint compatibility is out of scope.
- Upstream preset drift must be handled at build time, not runtime.
- Runtime must fail closed when embedded preset/schema expectations are violated.
- Runtime lint execution must remain offline and must not require external interpreters or parser subprocesses.
- Repository-standard developer entrypoints remain `task`, `go test`, and Go-native verification.

## Core Boundaries

- `tools/sync-preset/` is maintainer-only and runs a four-stage pipeline: (1) resolve the canonical preset via Bun and upstream JS packages, (2) validate that all rule names and parser fields are recognized by the runtime schema (fail-closed), (3) normalize the JS-typed encoding into `preset.Schema`, and (4) write the `preset.json` artifact atomically. The typed wire format between the TS resolver and the Go normalizer (`rawValue` kind-discriminated encoding) is an internal protocol scoped to this directory.
- `internal/preset/` is the sole runtime boundary for upstream-derived data. It stores a normalized schema and embed/load code only.
- Runtime CLI reads commit messages from `stdin`, explicit flags, or Git edit files, applies built-in ignore rules, parses the message into a minimal AST, evaluates embedded rules, and reports findings.
- Runtime does not perform config discovery, `extends` resolution, parser preset lookup, JS/TS config execution, or plugin loading.
- Runtime treats commit messages, edit files, and file paths as untrusted input and must not execute or shell-interpolate their contents.
- Hook installation is a thin convenience layer over `commit-msg` and must respect `core.hooksPath` unless explicitly overridden.

## Key Tech Decisions

- Use a normalized `preset.json` artifact instead of embedding raw resolved commitlint config objects.
- Fail sync on unknown rule names, unsupported parser fields, or other upstream data that cannot be mapped to the runtime schema. This validation is enforced by the `tools/sync-preset/` normalization stage.
- Count header/body/footer lengths using UTF-16 code units to stay close to JS `string.length` behavior.
- Implement default ignore rules as explicit Go logic instead of importing JS ignore functions.
- Keep the runtime contract narrow: `lint`, `hook install`, `print-preset`, `version`.
- Use a CLI framework for runtime command structure so subcommands, help text, and flag validation are established once and not revisited during later feature additions.
- Keep non-CLI runtime dependencies stdlib-first: JSON, embed, regex, filesystem, process execution, UTF-16 accounting, and reporting should use Go standard packages unless a concrete closure blocker appears.
- Limit third-party Go dependencies to test-only helpers with clear signal gain; the default choice is `github.com/google/go-cmp/cmp` for readable diffs, while runtime code remains dependency-light.
- Keep sync-layer JS dependencies minimal and purpose-specific: Bun as the execution environment plus `@commitlint/load`, `@commitlint/config-conventional`, and the resolved parser preset package.
- Keep lint-path performance bounded: runtime linting should be input-size proportional and avoid network access or external parser subprocesses.
- Keep runtime security boundaries explicit: no dynamic config execution, no shell interpolation, and no writes outside the requested hook target.
- Treat embedded preset data as redistributed upstream material for release hygiene: ship third-party notices for upstream licenses tied to the embedded artifact.
- Treat docs artifacts as part of the implementation contract: `TODO.md` for Theme closure, this architecture baseline for long-distance boundaries, and ADRs for durable design choices.

## Open Questions

- None at the architecture-boundary level for v1. Lower-level implementation questions should be resolved inside the owning Theme or escalated if they change these boundaries.

## Revisit Trigger

- v2 introduces custom presets, range linting, plugin/config loading, or another runtime dependency model.
- Upstream commitlint changes require parser/schema capabilities that cannot be represented by the normalized runtime artifact.
- Hook installation needs to integrate with an external hook manager instead of writing a simple `commit-msg` script.
