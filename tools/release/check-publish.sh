#!/usr/bin/env bash
set -euo pipefail

: "${RELEASE_TAG:?}" "${RELEASE_SHA:?}" "${GH_REPO:?}" "${GITHUB_OUTPUT:?}"
stable='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
fail() { echo "Release blocked: $*" >&2; exit 1; }

[[ "$RELEASE_TAG" =~ $stable ]] || fail "tag must be canonical stable vX.Y.Z"
[[ "$(git rev-parse HEAD)" == "$RELEASE_SHA" ]] || fail "checkout does not match event SHA"
[[ "$(git rev-parse "$RELEASE_TAG^{commit}")" == "$RELEASE_SHA" ]] || fail "tag does not match event SHA"
[[ "$RELEASE_TAG" == "v$(git show "$RELEASE_SHA:VERSION")" ]] || fail "tag does not match VERSION"
git merge-base --is-ancestor "$RELEASE_SHA" refs/remotes/origin/main || fail "tag is not on main"

# Re-read after waiting for CI. A published release is never handed to GoReleaser.
check_releases() {
  releases=$(gh api --method GET --paginate --slurp "repos/$GH_REPO/releases?per_page=100" |
    jq -ce 'add | if type == "array" and all(.[];
      (.tag_name | type) == "string" and (.draft | type) == "boolean" and
      (.prerelease | type) == "boolean") then . else error("invalid releases response") end')
  if jq -e --arg tag "$RELEASE_TAG" 'any(.[]; .tag_name == $tag and .draft == false)' <<< "$releases" >/dev/null; then
    printf 'publish=false\nlatest=false\n' >> "$GITHUB_OUTPUT"
    echo "$RELEASE_TAG is already published; skipping."
    exit 0
  fi
}
check_releases

# Allow five minutes for a delayed main-push CI run to appear. The workflow job
# timeout bounds gh run watch; --exit-status rejects every non-success conclusion.
run_id=''
for attempt in {1..20}; do
  runs=$(gh run list --repo "$GH_REPO" --workflow ci.yml --branch main --event push \
    --commit "$RELEASE_SHA" --limit 1 --json databaseId,headSha,headBranch,event)
  run_id=$(jq -er --arg sha "$RELEASE_SHA" '
    if type != "array" then error("invalid CI response")
    elif length == 0 then ""
    else .[0] | if .headSha == $sha and .headBranch == "main" and .event == "push"
      and (.databaseId | type) == "number"
      then .databaseId else error("CI run identity mismatch") end end' <<< "$runs")
  [[ -n "$run_id" ]] && break
  if (( attempt < 20 )); then sleep 15; fi
done
[[ -n "$run_id" ]] || fail "no main push CI run found at $RELEASE_SHA"
gh run watch "$run_id" --repo "$GH_REPO" --exit-status --interval 15
check_releases

# GoReleaser's use_existing_draft matches by name, not by tag. Only reuse an
# unambiguous draft with both the expected name and tag; preserve its notes.
jq -e --arg tag "$RELEASE_TAG" '
  [.[] | select(.draft and (.tag_name == $tag or .name == $tag))] |
  length <= 1 and all(.[]; .tag_name == $tag and .name == $tag)
' <<< "$releases" >/dev/null || fail "draft name/tag is ambiguous or mismatched"

latest=$(jq -r --arg tag "$RELEASE_TAG" --arg stable "$stable" '
  # Length then digits gives exact numeric ordering even beyond jq integer precision.
  def semver: ltrimstr("v") | split(".") | map([length, .]);
  ($tag | semver) as $version |
  all(.[] | select(.draft == false and .prerelease == false) |
      .tag_name | select(test($stable)); semver <= $version)
' <<< "$releases")
printf 'publish=true\nlatest=%s\n' "$latest" >> "$GITHUB_OUTPUT"
