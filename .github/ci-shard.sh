#!/usr/bin/env bash
# Prints the packages of a shard of the unit tests, one a line: cmd and ui
# take theirs, and rest everything else, so that the three are `go list ./...`
# between them; all is every package. The jobs of ci.yml run one shard each,
# together.
#
# A pull request tests only the packages its change affects (internal/
# ciaffected): CI_DIFF is the arguments of the `git diff` that finds the
# changed files; without it, the shard is whole. No line at all is a shard
# with nothing to test.
set -euo pipefail
mod=$(go list -m)
all=$(go list ./...)
case "${1:-}" in
cmd | ui) shard=$(grep -E "^$mod/$1(/|\$)" <<<"$all" || true) ;;
rest) shard=$(grep -v -E "^$mod/(cmd|ui)(/|\$)" <<<"$all" || true) ;;
all) shard=$all ;;
*) echo "usage: ci-shard.sh cmd|ui|rest|all" >&2; exit 2 ;;
esac
if [ -n "${CI_DIFF:-}" ]; then
  # shellcheck disable=SC2086
  affected=$(git diff --name-only --no-renames $CI_DIFF | go run ./internal/ciaffected)
  if [ "$affected" != FULL ]; then
    if [ -n "$affected" ]; then
      shard=$(grep -x -F -f <(printf '%s\n' "$affected") <<<"$shard" || true)
    else
      shard=
    fi
  fi
fi
if [ -n "$shard" ]; then printf '%s\n' "$shard"; fi
