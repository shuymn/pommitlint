# ADR-010: Release compatibility updates, not every maintenance change

## Status

Accepted

## Decision

Keep tagpr for VERSION, CHANGELOG, and tags, and GoReleaser for distribution. Do not introduce a release framework or mirror upstream dependency SemVer. Compare with the highest **published stable release**, not the preceding commit, PR base, or an unpublished tag.

- When direct commitlint/parser target versions change and preset semantics, runtime files, and Go dependencies remain unchanged, automatically merge the next patch release PR after CI succeeds. Include accumulated maintenance changes in that release.
- When only workflows, tooling, tests, lockfiles, or the Go toolchain change, leave the release PR open for the next compatibility update.
- When preset semantics, runtime files, or Go dependencies change, require a human to choose the release version and merge the tagpr PR. Use `tagpr:minor` / `tagpr:major`, or a signed VERSION edit, as appropriate. Preset differences alone cannot determine compatibility impact.
- Keep manual release PR merges available for urgent fixes, including security updates. Do not require a compatibility-target update for a manual release.
- When publishing a tag, require VERSION agreement, main ancestry, and successful main-push CI at that exact commit. Reuse draft releases on retry, skip published releases, and never let an older version take `latest`.

A conservative file comparison may request manual review for a behavior-preserving runtime refactor. Accept that false negative instead of maintaining a semantic code-change classifier. JSON key order/formatting and Go toolchain directives do not count as runtime changes. Compatibility tests establish agreement for the maintained corpus, not a proof of universal behavioral equivalence.

## Workflow boundaries

- CI always regenerates the preset, rejects uncommitted artifact drift, and runs differential compatibility tests. `Release checks` aggregates build/test and compatibility success; do not path-filter it.
- `auto-release.yml` handles only successful same-repository release-PR CI runs. Execute scripts from trusted main, inspect the PR through Git object reads, and never execute triggering-run artifacts or PR code with its App token.
- Automatic merge requires the expected bot/branch, only VERSION/CHANGELOG changes, the tested head SHA, a current main base, and the next patch version. Do not enable deferred auto-merge or bypass protections.
- Remove dependency-label-driven VERSION commits. Sync preset changes with GitHub's signed commit API and an expected-head guard; rerun sync for refreshed lockfiles instead of trusting an earlier sync commit message.

## Activation and recovery

1. Land this pipeline before regenerating PRs containing old `chore: bump version` commits. Regenerate those PRs through Renovate so unsigned VERSION commits are not carried into main. Do not weaken signature requirements.
2. Manage repository settings in `.github/infra.yaml` with gh-infra (validated with v0.14.0). After `Release checks` succeeds on main, run `task infra:validate` and `task infra:plan`, then apply the reviewed plan with `task infra:apply`. The `Default` ruleset requires `Release checks` from `app: github-actions` (integration ID `15368`) and **Require branches to be up to date before merging**. Preserve its signature/deletion/force-push protections and existing admin bypass; do not grant the release App a bypass. Automatic merge deliberately remains disabled until this strict check is configured: a head-SHA guard cannot atomically guard an advancing base branch.
3. Preserve the existing `APP_ID` / `PRIVATE_KEY` GitHub App credentials and its contents/pull-requests write access. The release workflow uses the repository token for publication.
4. Retry a failed tag workflow after fixing the failure. GitHub concurrency keeps one running and one pending publication; if several manual tags supersede a pending run, rerun the cancelled tag workflow. Do not move tags or overwrite a published release. A pending predecessor publication can leave the next release PR waiting; rerun its CI after publication completes to reevaluate eligibility.
5. Verify signed preset commits, strict stale-base rejection, App-driven merge/tag events, and draft retries on GitHub during rollout. Local fixture tests cannot establish those platform behaviors.

Manage rulesets authoritatively, but keep labels additive so gh-infra does not delete tagpr's dynamically created release-control labels. Import other repository settings without changing their values. Do not add an automatic infra-apply workflow or give the release App administration access. Run `task infra:plan` after apply and require no remaining changes; inspect `repos/shuymn/pommitlint/rules/branches/main` to verify the strict check and its provider. Reapply changes through the manifest rather than making UI-only edits.

Run `task test:release`, `task check`, `task sync-preset` (require no preset diff), `task compat-check`, `actionlint`, and `ghalint run` when changing this pipeline.
