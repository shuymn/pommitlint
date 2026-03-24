# ADR-005: Normalize upstream preset at build time and keep runtime Node-free

## Status

Accepted

## Context

`pommitlint` is intended to provide `@commitlint/config-conventional` equivalent linting as a single Go binary. The upstream commitlint ecosystem resolves configuration dynamically through Node.js package resolution, config discovery, parser preset loading, and executable JS/TS config files. That model conflicts with the runtime goals for this repository:

- no Node/Bun/npm dependency in hooks or CI runtime,
- predictable single-binary distribution,
- stable and reviewable diffs when upstream preset data changes,
- fail-closed behavior when upstream introduces unsupported shape changes.

The repository is currently starting from a Go template, so this decision needs to be fixed before runtime code is implemented.

## Decision

Adopt a two-layer design:

- A maintainer-only sync tool under `tools/sync-preset/` uses Bun and upstream JS packages to resolve `@commitlint/config-conventional` plus its parser preset.
- The sync tool emits a normalized runtime artifact at `internal/preset/preset.json`.
- The Go runtime embeds that artifact and never performs config discovery, `extends` resolution, parser preset lookup, or JS/TS execution.
- The normalization layer includes only schema fields required by the Go runtime and rejects unknown rule names or unsupported parser fields.

The runtime remains intentionally narrow:

- fixed compatibility target: `@commitlint/config-conventional`,
- built-in Go implementation of default ignores,
- minimal AST and rule engine driven by the normalized preset,
- public CLI limited to `lint`, `hook install`, `print-preset`, and `version`.

## Rejected Alternatives

- Resolve commitlint config at runtime with Node/Bun.
  - Rejected because it breaks the single-binary runtime goal and makes hook/CI environments depend on external JS tooling.
- Embed the raw resolved upstream config object directly.
  - Rejected because it leaks unnecessary prompt/function/data shape into the runtime boundary and makes unsupported upstream drift harder to detect.
- Reimplement commitlint configuration compatibility wholesale.
  - Rejected because v1 explicitly targets `config-conventional` only, and full compatibility would expand scope well beyond the requested product.

## Consequence

- Upstream preset drift is handled in maintainer workflows and reviewed as a JSON artifact diff.
- Runtime code can be tested deterministically against a fixed schema.
- Unsupported upstream changes stop at sync time instead of silently altering runtime behavior.
- The project must maintain Bun-based sync tooling and associated notices, even though runtime remains Go-only.
- Any future requirement for custom presets or runtime config loading will require revisiting this ADR.

## Revisit trigger

- The project expands beyond `config-conventional`.
- Upstream commitlint/parser preset changes cannot be represented by the normalized schema without weakening fail-closed guarantees.
- Distribution goals change and runtime JS dependencies become acceptable.
