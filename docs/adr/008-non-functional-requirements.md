# ADR-008: Fix v1 security and performance requirements early

## Status

Accepted

## Context

`pommitlint` is intended to run in `commit-msg` hooks and CI, which means even small inefficiencies or unsafe execution boundaries have outsized impact:

- hook latency is paid on every commit,
- commit messages and edit files are untrusted input,
- hook installation touches executable files and Git metadata,
- preset sync introduces an explicit boundary between trusted build-time artifacts and untrusted runtime input.

The project should therefore define a small set of non-functional requirements now, while the architecture is still thin.

## Decision

Adopt the following v1 non-functional requirements.

### Performance requirements

- Runtime lint execution must be offline.
  - `pommitlint lint` must not perform network access.
- Runtime work must be proportional to input size.
  - Parsing and rule evaluation should be single-pass or near single-pass over the message and avoid algorithmic blowups on long lines or repeated delimiters.
- Runtime startup and execution must stay lightweight enough for hook usage.
  - No runtime dependency on external interpreters or subprocesses for parsing or rule evaluation.
  - Git subprocesses are allowed only for Git-specific commands such as hook-path resolution or installation, not for linting itself.
- Performance checks should focus on regression detection, not premature micro-optimization.
  - Add focused benchmarks for parser and lint hot paths only if implementation complexity or fixture size suggests regression risk.

### Security requirements

- Treat all commit message content, edit files, and file paths as untrusted input.
- Runtime must not execute commit message content, shell fragments, config text, or preset-derived strings.
- Runtime must not perform JS/TS config execution, dynamic plugin loading, or arbitrary `extends` resolution.
- File operations must stay within explicit user-selected inputs and hook targets.
  - `--file` and `--edit` read only the requested path or the documented default.
  - `hook install` writes only the resolved hook destination and must refuse overwrite unless `--force` is set.
- JSON and text output must escape or delimit data so terminals and downstream tools do not reinterpret findings as commands.
- Any subprocess use must use argument arrays, not shell interpolation.
- Tests must cover malformed, oversized, and hostile-looking input shapes well enough to prove fail-safe behavior.

## Rejected Alternatives

- Add broad sandboxing or resource-limiting infrastructure in v1.
  - Rejected because the product is a local CLI, and the higher-value control is a narrow runtime boundary with no dynamic execution.
- Define hard latency SLOs now.
  - Rejected because there is no baseline implementation yet; the useful v1 requirement is qualitative: offline, input-bounded, and no external parser subprocesses.
- Treat security as only a future review concern.
  - Rejected because the runtime boundary and hook behavior are being fixed now, and weak defaults would be expensive to unwind later.

## Consequence

- Runtime implementation must stay simple and explicit at input/file/subprocess boundaries.
- Future changes that add network access, dynamic config loading, shell execution, or broad filesystem writes will require an ADR revisit.
- Verification should include adversarial parser/report tests and, where useful, targeted benchmarks to catch regressions in the hook path.

## Revisit trigger

- The product starts linting ranges, repositories, or remote resources instead of a single message.
- A future version introduces user-configurable presets or plugins.
- Real-world usage produces hook-latency evidence that requires explicit numeric budgets.
