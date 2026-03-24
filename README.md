# pommitlint

`pommitlint` is a planned Go CLI that provides `@commitlint/config-conventional` equivalent linting as a single binary. The runtime target is intentionally narrow: no runtime Node.js dependency, no config discovery, no plugin system, and no arbitrary `extends` support.

## Status

The repository is in the design-to-implementation handoff phase. The authoritative planning artifacts are:

- [`TODO.md`](TODO.md) for Theme-level execution and closure criteria
- [`docs/architecture.md`](docs/architecture.md) for the architecture baseline
- [`docs/adr/005-runtime-preset-boundary.md`](docs/adr/005-runtime-preset-boundary.md) through [`docs/adr/009-embedded-preset-license-notices.md`](docs/adr/009-embedded-preset-license-notices.md) for durable design decisions

## Planned CLI

The v1 public contract is:

```text
pommitlint lint
pommitlint hook install
pommitlint print-preset
pommitlint version
```

`lint` will support one input source at a time: `stdin`, `--message`, `--file`, or `--edit [PATH]`.

## Development

Use `task` as the default entrypoint:

```bash
task
task build
task test
task lint
task check
```

Current repository tasks are still generic Go project tasks; product-specific tasks and code will be introduced in the implementation Themes described in [`TODO.md`](TODO.md).

## Dependency Posture

The current design decision is:

- runtime Go code: `cobra` for CLI, standard library for the rest
- test-only helper: `github.com/google/go-cmp/cmp` when clearer diffs matter
- maintainer-only sync: Bun + commitlint packages only
- Git-backed tests: isolated temp repos with no ambient Git/GPG prompts or signing

See [`docs/adr/006-library-selection.md`](docs/adr/006-library-selection.md) for the rationale and rejected alternatives.
See [`docs/adr/007-git-test-isolation.md`](docs/adr/007-git-test-isolation.md) for Git/GPG test isolation policy.
See [`docs/adr/008-non-functional-requirements.md`](docs/adr/008-non-functional-requirements.md) for the v1 security and performance baseline.
See [`docs/adr/009-embedded-preset-license-notices.md`](docs/adr/009-embedded-preset-license-notices.md) for the embedded preset notice policy.

## Scope

v1 is intentionally limited to:

- `@commitlint/config-conventional` compatibility only
- build-time preset sync
- single-binary runtime for hooks and CI
- text and JSON reporting

v1 explicitly excludes:

- runtime JS/TS config execution
- arbitrary shareable configs or `extends`
- plugin support
- prompt/interactive authoring
- full commitlint CLI compatibility
