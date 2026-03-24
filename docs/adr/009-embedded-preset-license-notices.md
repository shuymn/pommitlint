# ADR-009: Ship third-party notices for embedded preset artifacts

## Status

Accepted

## Context

`pommitlint` v1 embeds a normalized `preset.json` derived from upstream packages rather than resolving configuration at runtime. That artifact is expected to contain data originating from:

- `@commitlint/config-conventional`
- `conventional-changelog-conventionalcommits`

Those upstream projects are distributed under permissive licenses, but permissive does not mean notice-free when copies or substantial portions are redistributed.

Primary-source license signals:

- The `commitlint` repository states that all `commitlint` packages are released under the MIT license, and its `license.md` contains the MIT condition requiring the copyright and permission notice to be included in all copies or substantial portions. [commitlint repo](https://github.com/conventional-changelog/commitlint) [license.md](https://github.com/conventional-changelog/commitlint/blob/master/license.md)
- The `conventional-changelog` repository publishes an ISC license requiring the copyright and permission notice to appear in all copies. [LICENSE.md](https://raw.githubusercontent.com/conventional-changelog/conventional-changelog/master/LICENSE.md)

Because `preset.json` is a derived artifact redistributed inside the shipped binary, the project needs an explicit attribution policy before implementation lands.

## Decision

When `pommitlint` ships an embedded preset artifact, the distribution must include third-party notices covering the upstream packages whose data is redistributed through that artifact.

v1 policy:

- Ship the project `LICENSE` for `pommitlint` itself.
- Ship `THIRD_PARTY_NOTICES.md` in repository and release artifacts.
- `THIRD_PARTY_NOTICES.md` must include attribution and license text for:
  - `@commitlint/config-conventional` / commitlint packages under MIT
  - `conventional-changelog-conventionalcommits` / conventional-changelog under ISC
- Treat this as a release blocker, not optional documentation.

Scope boundary:

- This notice requirement is driven by redistributed embedded artifact content.
- Sync-only tooling dependencies that are not redistributed in the shipped binary do not automatically require inclusion in `THIRD_PARTY_NOTICES.md`; add them only if later distribution changes cause their material to ship.

## Rejected Alternatives

- Assume the normalized JSON is too small or transformed to require notices.
  - Rejected because that is a legal-risk optimization with little upside; shipping notices is cheap and low-risk.
- Include every transitive sync dependency in release notices.
  - Rejected because v1 should track redistributed material, not every build-time-only package.
- Defer the decision until release packaging exists.
  - Rejected because the embed boundary is already part of the architecture and should have a fixed compliance rule now.

## Consequence

- Release readiness includes verifying `THIRD_PARTY_NOTICES.md`.
- Sync/runtime boundary documentation now includes a compliance obligation, not just a technical boundary.
- Future changes to what is embedded or redistributed must revisit notice scope.

## Revisit trigger

- The embedded artifact starts incorporating additional upstream material.
- Release packaging changes in a way that redistributes more than the current binary and notices.
- Legal review determines the notice set or wording should be broader than the current policy.
