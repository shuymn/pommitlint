#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
mkdir -p "$root/.cache"
tmp=$(mktemp -d "$root/.cache/check-publish.XXXXXX")
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin" "$tmp/fixtures" "$tmp/repo" "$tmp/hooks" "$tmp/home" "$tmp/xdg" "$tmp/gnupg"
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR GIT_OBJECT_DIRECTORY \
  GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_PARAMETERS GIT_CONFIG_COUNT
export HOME="$tmp/home" XDG_CONFIG_HOME="$tmp/xdg" GNUPGHOME="$tmp/gnupg"
export GIT_TERMINAL_PROMPT=0 GIT_ASKPASS=/bin/false SSH_ASKPASS=/bin/false GCM_INTERACTIVE=Never
export FIXTURES="$tmp/fixtures" GITHUB_OUTPUT="$tmp/output"
export GH_REPO=test/project GH_TOKEN=fake-token
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null
export GIT_AUTHOR_NAME=Test GIT_COMMITTER_NAME=Test
export GIT_AUTHOR_EMAIL=test@example.invalid GIT_COMMITTER_EMAIL=test@example.invalid
cd "$tmp/repo"
git() { command git -c commit.gpgsign=false -c tag.gpgsign=false "$@"; }
git -c init.defaultBranch=main init -q
git config core.hooksPath "$tmp/hooks"

# Synthetic history lives only in the ignored fixture repository.
commit_version() {
  local blob tree
  blob=$(printf '%s\n' "$1" | git hash-object -w --stdin)
  tree=$(printf '100644 blob %s\tVERSION\n' "$blob" | git mktree)
  shift
  git commit-tree "$tree" -m fixture "$@"
}
base=$(commit_version 2.0.0)
target=$(commit_version 2.0.1 -p "$base")
tip=$(commit_version 2.0.2 -p "$target")
side=$(commit_version 2.1.0 -p "$base")
git update-ref refs/remotes/origin/main "$tip"
git tag v2.0.0 "$base"
git tag v2.0.1 "$target"
git tag v2.0.9 "$target"
git tag v2.1.0 "$side"

cat > "$tmp/bin/gh" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$FIXTURES/calls"
case "$1 $2" in
  'api --method')
    [[ "$*" == "api --method GET --paginate --slurp repos/test/project/releases?per_page=100" ]]
    if [[ -e "$FIXTURES/api-error" ]] || { [[ -e "$FIXTURES/api-seen" && -e "$FIXTURES/api-after-error" ]]; }; then
      echo 'fixture API failure' >&2; exit 1
    fi
    file="$FIXTURES/releases.json"
    if [[ -e "$FIXTURES/api-seen" && -e "$FIXTURES/releases-after.json" ]]; then file="$FIXTURES/releases-after.json"; fi
    touch "$FIXTURES/api-seen"
    cat "$file"
    ;;
  'run list')
    [[ "$*" == "run list --repo test/project --workflow ci.yml --branch main --event push --commit $RELEASE_SHA --limit 1 --json databaseId,headSha,headBranch,event" ]]
    if [[ -e "$FIXTURES/ci-error" ]]; then echo 'fixture CI API failure' >&2; exit 1; fi
    if [[ -e "$FIXTURES/ci-delay" ]]; then
      read -r remaining < "$FIXTURES/ci-delay"
      if (( remaining > 0 )); then
        printf '%s\n' "$((remaining - 1))" > "$FIXTURES/ci-delay"
        echo '[]'; exit 0
      fi
    fi
    cat "$FIXTURES/runs.json"
    ;;
  'run watch')
    [[ "$*" == 'run watch 42 --repo test/project --exit-status --interval 15' ]]
    if [[ -e "$FIXTURES/watch-error" ]]; then echo 'fixture CI not successful' >&2; exit 1; fi
    ;;
  *) echo "unexpected gh call: $*" >&2; exit 99 ;;
esac
FAKE
cat > "$tmp/bin/sleep" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == '15' ]]
printf '%s\n' "$*" >> "$FIXTURES/sleeps"
FAKE
chmod +x "$tmp/bin/gh" "$tmp/bin/sleep"
export PATH="$tmp/bin:$PATH"

reset_case() {
  rm -f "$FIXTURES/"*
  : > "$GITHUB_OUTPUT"
  export RELEASE_TAG=v2.0.1 RELEASE_SHA="$target"
  git checkout -q --detach "$target"
  printf '[[]]\n' > "$FIXTURES/releases.json"
  jq -n --arg sha "$target" '[{databaseId:42,headSha:$sha,headBranch:"main",event:"push"}]' > "$FIXTURES/runs.json"
}

passed=0
expect_output() {
  local name=$1 expected=$2
  if ! bash "$root/tools/release/check-publish.sh" > "$tmp/log" 2>&1; then
    echo "FAIL: $name" >&2; cat "$tmp/log" >&2; exit 1
  fi
  if [[ "$(< "$GITHUB_OUTPUT")" != "$expected" ]]; then
    echo "FAIL: $name: unexpected outputs" >&2; cat "$GITHUB_OUTPUT" >&2; exit 1
  fi
  passed=$((passed + 1))
  echo "ok: $name"
}
expect_blocked() {
  local name=$1 message=$2
  if bash "$root/tools/release/check-publish.sh" > "$tmp/log" 2>&1; then
    echo "FAIL: $name: allowed publication" >&2; exit 1
  fi
  if [[ -s "$GITHUB_OUTPUT" ]] || ! grep -q "$message" "$tmp/log"; then
    echo "FAIL: $name: wrong failure" >&2; cat "$tmp/log" >&2; exit 1
  fi
  passed=$((passed + 1))
  echo "ok: $name"
}
allow=$'publish=true\nlatest=true'
older=$'publish=true\nlatest=false'
skip=$'publish=false\nlatest=false'

reset_case
expect_output 'exact successful main-push CI, main ancestor, first release' "$allow"
grep -q '^run watch 42 ' "$FIXTURES/calls"

reset_case
printf '2\n' > "$FIXTURES/ci-delay"
expect_output 'wait for CI to appear' "$allow"
[[ $(grep -c '^run list ' "$FIXTURES/calls") == 3 && $(wc -l < "$FIXTURES/sleeps") -eq 2 ]]

reset_case
printf '20\n' > "$FIXTURES/ci-delay"
expect_blocked 'bounded wait for absent CI' 'no main push CI run found'
[[ $(grep -c '^run list ' "$FIXTURES/calls") == 20 && $(wc -l < "$FIXTURES/sleeps") -eq 19 ]]

reset_case
touch "$FIXTURES/watch-error"
expect_blocked 'unsuccessful CI or watch API failure' 'fixture CI not successful'

reset_case
touch "$FIXTURES/ci-error"
expect_blocked 'CI list API failure' 'fixture CI API failure'

reset_case
printf 'null\n' > "$FIXTURES/runs.json"
expect_blocked 'malformed CI response' 'invalid CI response'

for field in headSha headBranch event; do
  reset_case
  jq --arg field "$field" '.[0][$field] = "wrong"' "$FIXTURES/runs.json" > "$tmp/runs.json"
  mv "$tmp/runs.json" "$FIXTURES/runs.json"
  expect_blocked "reject CI with wrong $field" 'CI run identity mismatch'
done

for tag in 2.0.1 v02.0.1 v2.00.1 v2.0.01 v2.0 v2.0.1-rc.1 v2.0.1+build; do
  reset_case
  export RELEASE_TAG=$tag
  expect_blocked "noncanonical tag $tag" 'canonical stable'
done

reset_case
export RELEASE_TAG=v2.0.9
expect_blocked 'VERSION mismatch' 'tag does not match VERSION'

reset_case
export RELEASE_TAG=v2.0.0
expect_blocked 'tag/event SHA mismatch' 'tag does not match event SHA'

reset_case
export RELEASE_SHA=$tip
expect_blocked 'checkout/event SHA mismatch' 'checkout does not match event SHA'

reset_case
git checkout -q --detach "$side"
export RELEASE_TAG=v2.1.0 RELEASE_SHA=$side
expect_blocked 'commit outside main ancestry' 'tag is not on main'

reset_case
printf '[[],[{"name":"renamed","tag_name":"v2.0.1","draft":false,"prerelease":false}]]\n' > "$FIXTURES/releases.json"
expect_output 'published tag on a later API page is skipped' "$skip"
if grep -q '^run ' "$FIXTURES/calls"; then echo 'FAIL: published release waited for CI' >&2; exit 1; fi

reset_case
printf '[[{"name":"v2.0.1","tag_name":"v2.0.1","draft":false,"prerelease":false}]]\n' > "$FIXTURES/releases-after.json"
expect_output 'tag published during CI wait is skipped' "$skip"

for fault in api-error api-after-error; do
  reset_case
  touch "$FIXTURES/$fault"
  expect_blocked "$fault fails closed" 'fixture API failure'
done

for response in 'null' '[{}]' '[[{"tag_name":"v2.0.1"}]]'; do
  reset_case
  printf '%s\n' "$response" > "$FIXTURES/releases.json"
  expect_blocked 'malformed releases response' 'error'
done

reset_case
printf '[[{"name":"v2.0.1","tag_name":"v2.0.1","draft":true,"prerelease":false}]]\n' > "$FIXTURES/releases.json"
expect_output 'reuse matching draft' "$allow"

for release in \
  '{"name":"v2.0.1","tag_name":"v9.0.0","draft":true,"prerelease":false}' \
  '{"name":"renamed","tag_name":"v2.0.1","draft":true,"prerelease":false}'; do
  reset_case
  printf '[[%s]]\n' "$release" > "$FIXTURES/releases.json"
  expect_blocked 'reject draft name/tag collision' 'draft name/tag is ambiguous or mismatched'
done

reset_case
printf '[[{"name":"v2.0.1","tag_name":"v2.0.1","draft":true,"prerelease":false},{"name":"v2.0.1","tag_name":"v2.0.1","draft":true,"prerelease":false}]]\n' > "$FIXTURES/releases.json"
expect_blocked 'reject duplicate drafts' 'draft name/tag is ambiguous or mismatched'

for tag in v10.0.0 v2.10.0 v2.0.10 v9007199254740993.0.0; do
  reset_case
  jq -n --arg tag "$tag" '[[{name:$tag,tag_name:$tag,draft:false,prerelease:false}]]' > "$FIXTURES/releases.json"
  expect_output "newer stable $tag prevents latest" "$older"
done

reset_case
jq -n '[[{tag_name:"v1.99.99",draft:false,prerelease:false},
  {tag_name:"v99.0.0",draft:true,prerelease:false},
  {tag_name:"v99.1.0",draft:false,prerelease:true},
  {tag_name:"v99.0.0-rc.1",draft:false,prerelease:true},
  {tag_name:"v099.0.0",draft:false,prerelease:false}]]' > "$FIXTURES/releases.json"
expect_output 'ignore drafts, prereleases, noncanonical tags; older stable allows latest' "$allow"

reset_case
jq -n '[[{tag_name:"v1.0.0",draft:false,prerelease:false}]]' > "$FIXTURES/releases.json"
jq -n '[[],[{tag_name:"v3.0.0",draft:false,prerelease:false}]]' > "$FIXTURES/releases-after.json"
expect_output 'refresh latest decision after CI wait and across pages' "$older"

echo "Passed $passed publication guard cases."
