#!/usr/bin/env bash
set -euo pipefail
script=$(cd "$(dirname "$0")" && pwd)/auto-patch.sh
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT

# Isolate Git fixtures from user hooks, signing agents, and configuration (ADR-007).
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR GIT_OBJECT_DIRECTORY \
  GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_PARAMETERS GIT_CONFIG_COUNT
export HOME="$root/home" XDG_CONFIG_HOME="$root/xdg" GNUPGHOME="$root/gnupg"
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_TERMINAL_PROMPT=0
export GIT_ASKPASS=/bin/false SSH_ASKPASS=/bin/false GCM_INTERACTIVE=Never
mkdir -p "$HOME" "$XDG_CONFIG_HOME" "$GNUPGHOME" "$root/hooks" "$root/repo"
cd "$root/repo"
git() { command git -c commit.gpgsign=false -c tag.gpgsign=false "$@"; }
git -c init.defaultBranch=main init -q
git config user.name test
git config user.email test@example.invalid
git config core.hooksPath "$root/hooks"
mkdir -p internal/preset internal/lint tools/{sync-preset,compat-check}
printf '2.0.1\n' > VERSION
printf '{"rules":{},"parserPreset":{"name":"conventionalcommits"}}\n' > internal/preset/preset.json
printf 'package main\n' > main.go
printf 'module example.invalid/test\n\ngo 1.26.0\ntoolchain go1.26.1\n' > go.mod

upstream() {
  printf '{"dependencies":{"@commitlint/config-conventional":"%s"}}\n' "$1" > tools/sync-preset/package.json
  printf '{"dependencies":{"@commitlint/lint":"%s"}}\n' "$1" > tools/compat-check/package.json
}
commit() { git add .; git commit -qm fixture; }
begin() { git checkout -qb "$1" v2.0.1; printf '2.0.2\n' > VERSION; }
expect() {
  local actual
  commit
  actual=$(bash "$script" v2.0.1 HEAD)
  [[ "$actual" == "$1" ]] || { echo "$2: expected $1, got $actual" >&2; exit 1; }
  echo "PASS: $2"
}
expect_error() {
  commit
  if bash "$script" "$1" HEAD > "$root/output" 2> "$root/error"; then
    echo "$2: expected a failure" >&2
    exit 1
  fi
  echo "PASS: $2"
}

upstream 21.0.1
commit
git tag v2.0.1

begin maintenance
printf 'new workflow\n' > workflow.yml
expect false 'maintenance alone waits for a compatibility update'

begin compatibility
upstream 21.2.3
printf 'bundled maintenance\n' > workflow.yml
expect true 'compatibility update includes accumulated maintenance'

begin upstream-major
upstream 22.0.0
expect true 'upstream major does not imply a pommitlint major'

begin preset-change
upstream 21.2.3
printf '{"rules":{"type-empty":[2,"never"]}}\n' > internal/preset/preset.json
expect false 'preset changes require manual release review'

begin formatting
upstream 21.2.3
printf '{"parserPreset":{"name":"conventionalcommits"}, "rules": {}}\n' > internal/preset/preset.json
expect true 'JSON formatting and key order are not behavioral changes'

begin runtime-change
upstream 21.2.3
printf 'package main\n// a runtime change\n' > main.go
expect false 'runtime changes require manual release review'

begin runtime-deletion
upstream 21.2.3
rm main.go
expect false 'runtime deletion cannot be auto released'

begin test-change
upstream 21.2.3
printf 'package lint\n' > internal/lint/example_test.go
expect true 'test-only changes can be bundled'

begin toolchain
upstream 21.2.3
printf 'module example.invalid/test\n\ngo 1.27.0\ntoolchain go1.27.2\n' > go.mod
expect true 'toolchain updates can be bundled'

begin go-dependency
upstream 21.2.3
printf '\nrequire example.invalid/library v1.2.3\n' >> go.mod
expect false 'Go dependency changes require manual review'

begin major-version
upstream 21.2.3
printf '3.0.0\n' > VERSION
expect false 'explicit major version is not auto merged'

begin minor-version
upstream 21.2.3
printf '2.1.0\n' > VERSION
expect false 'explicit minor version is not auto merged'

begin non-target-package
printf '{"dependencies":{"@commitlint/lint":"21.0.1","other":"1.0.0"}}\n' > tools/compat-check/package.json
expect false 'non-target dependencies do not trigger a release'

begin invalid-manifest
printf 'invalid JSON\n' > tools/sync-preset/package.json
expect_error v2.0.1 'invalid first manifest fails instead of being masked by the second'

begin missing-manifest
rm tools/sync-preset/package.json
expect_error v2.0.1 'missing manifest fails closed'

begin unknown-base
expect_error v9.9.9 'missing published tag fails closed'

begin prerelease
expect_error v2.0.1-rc.1 'prerelease is not a stable baseline'

# Exercise the actual workflow glue as well as the policy. In particular, the
# published tag must be a raw string, not a JSON-quoted string that never matches.
yq -r '.jobs.merge.steps[] | select(.id == "eligible") | .run' \
  "$(dirname "$script")/../../.github/workflows/auto-release.yml" > "$root/eligible.sh"
git checkout -q compatibility
export TESTED_SHA FIXTURE_HEAD
TESTED_SHA=$(git rev-parse HEAD)
FIXTURE_HEAD=$TESTED_SHA
export GITHUB_REPOSITORY=example/repo GITHUB_REPOSITORY_OWNER=example GH_TOKEN=fixture
export HEAD_BRANCH=tagpr-from-v2.0.1 GITHUB_OUTPUT="$root/outputs" GITHUB_STEP_SUMMARY="$root/summary"
git remote add origin "$PWD"
git update-ref refs/pull/42/head "$TESTED_SHA"
git update-ref refs/remotes/origin/main "$(git rev-parse v2.0.1)"
mkdir -p "$root/bin" tools/release
cp "$script" tools/release/auto-patch.sh
cat > "$root/bin/gh" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *'/releases?'*) printf 'v2.0.0\nv2.0.1\n' ;;
  *'/pulls?'*) echo 42 ;;
  *'/pulls/42/files?'*) echo '[[{"filename":"VERSION","status":"modified"},{"filename":"CHANGELOG.md","status":"modified"}]]' ;;
  *'/pulls/42') jq -n --arg sha "$FIXTURE_HEAD" '{head:{sha:$sha},labels:[]}' ;;
  *'/rules/branches/main')
    if [[ "${MISSING_RULE:-}" == true ]]; then
      echo '[]'
    else
      echo '[{"type":"required_status_checks","parameters":{"strict_required_status_checks_policy":true,"required_status_checks":[{"context":"Release checks","integration_id":15368}]}}]'
    fi ;;
  *) echo "unexpected gh call: $*" >&2; exit 1 ;;
esac
FAKE
chmod +x "$root/bin/gh"
export PATH="$root/bin:$PATH"
: > "$GITHUB_OUTPUT"
bash "$root/eligible.sh"
[[ "$(< "$GITHUB_OUTPUT")" == "$(printf 'number=42\nhead=%s' "$TESTED_SHA")" ]]
echo 'PASS: workflow accepts a checked patch PR against the published tag'

: > "$GITHUB_OUTPUT"
MISSING_RULE=true bash "$root/eligible.sh"
[[ ! -s "$GITHUB_OUTPUT" ]]
grep -q 'Auto release is disabled' "$GITHUB_STEP_SUMMARY"
echo 'PASS: workflow stays disabled without a strict required CI check'
