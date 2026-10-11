#!/usr/bin/env bash
# Print true only for a compatibility-target update with no runtime/preset change.
# Compare with the last published stable tag, not the previous commit or PR base.
set -euo pipefail

base=${1:?usage: auto-patch.sh PUBLISHED_TAG [HEAD_REF]}
head=${2:-HEAD}

manual() {
  echo "$1" >&2
  echo false
  exit 0
}

if [[ ! "$base" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Expected a canonical stable published tag: $base" >&2
  exit 1
fi
git merge-base --is-ancestor "$base" "$head"
expected=$(jq -nr --arg tag "$base" '$tag[1:] | split(".") | map(tonumber) | .[2] += 1 | map(tostring) | join(".")')
proposed=$(git show "$head:VERSION")
[[ "$proposed" == "$expected" ]] || manual "Version is not the next patch; merge the release PR manually."

# Tests and fixture-only changes do not alter the distributed runtime.
runtime=$(git diff --name-only "$base" "$head" -- main.go internal \
  ':!**/*_test.go' ':!**/testdata/**' ':!internal/preset/preset.json')
[[ -z "$runtime" ]] || manual "Runtime changed; review the release version and merge manually."

preset() {
  git show "$1:internal/preset/preset.json" | jq -Sc .
}
previous_preset=$(preset "$base")
current_preset=$(preset "$head")
[[ "$previous_preset" == "$current_preset" ]] || manual "Preset changed; review the compatibility impact and merge manually."

# Toolchain updates may ride along; a module/dependency change needs review.
modules() {
  git show "$1:go.mod" | awk '!/^(go|toolchain) /'
}
previous_modules=$(modules "$base")
current_modules=$(modules "$head")
[[ "$previous_modules" == "$current_modules" ]] || manual "Go dependencies changed; merge the release PR manually."

upstream() {
  local file
  for file in tools/sync-preset/package.json tools/compat-check/package.json; do
    git show "$1:$file" | jq -Sc '.dependencies | with_entries(select(.key | test("^@commitlint/|^conventional-changelog-")))' || return 1
  done
}
previous_upstream=$(upstream "$base")
current_upstream=$(upstream "$head")
[[ "$previous_upstream" != "$current_upstream" ]] || manual "Maintenance only; keep these changes for the next release."
echo true
