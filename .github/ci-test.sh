#!/usr/bin/env bash
# Runs go test with the flags before -- on the packages after it. When ui is
# among them, its slowest tests run in processes of their own, all at once:
# ./ui runs its tests one after another on one core, and its Testers share
# the process's text system, so they cannot take t.Parallel (Tester in
# ui/headless.go). Each group is a few walks of many seeds; the rest of the
# tests, of every package, skip them, so that every test runs once. The
# skip is of every package given, so a name of the groups must be no test's
# name in another of them.
#
#   ci-test.sh -race -shuffle=on -- $(bash .github/ci-shard.sh ui)
set -euo pipefail
# About 3 minutes each under -race on an M-series core, as is the rest of ./ui.
groups=(
  'TestPaintCullMatches|TestRetainSliderRowsRandomWalk|TestRetainShiftPaintMixedScales|TestRetainComboboxRowsRandomWalk|TestPureDrawsReachNoNode'
  'TestLayoutCacheMatches|TestRetainScrollFuzz|TestRetainFocusableRowsRandomWalk|TestRetainLayoutInnerScrollModelSeed'
  'TestRetainSwitchRowsRandomWalk|TestRetainRowsOfTextInputUsersRandomWalk|TestRetainSelectProgressRowsRandomWalk|TestRetainMatchesFresh'
  'TestRetainRandomWalkOfReaders|TestReplayMatches'
)
flags=()
while [ "$#" -gt 0 ] && [ "$1" != -- ]; do flags+=("$1"); shift; done
if [ "$#" -gt 0 ]; then shift; fi
ui=$(go list -m)/ui
if ! printf '%s\n' "$@" | grep -qx "$ui"; then
  exec go test "${flags[@]}" "$@"
fi
all=$(IFS='|'; echo "${groups[*]}")
# Built once, before the processes would each build it. A name of the groups
# that is no test of ./ui any more (renamed, removed) fails the run: its
# group would still pass, with fewer tests, and the split drift unseen.
listed=$(go test "${flags[@]}" -list "^($all)\$" "$ui")
missing=$(tr '|' '\n' <<<"$all" | grep -vxF -f <(printf '%s\n' "$listed") || true)
if [ -n "$missing" ]; then
  echo "ci-test.sh: these names of its groups are no test of $ui; rename or drop them:" >&2
  echo "$missing" >&2
  exit 1
fi
out=$(mktemp -d)
pids=()
go test "${flags[@]}" -skip "^($all)\$" "$@" >"$out/0" 2>&1 &
pids+=($!)
for i in "${!groups[@]}"; do
  go test "${flags[@]}" -run "^(${groups[$i]})\$" "$ui" >"$out/$((i + 1))" 2>&1 &
  pids+=($!)
done
status=0
for i in "${!pids[@]}"; do
  wait "${pids[$i]}" || status=1
  if [ "$i" -gt 0 ]; then echo "--- ui, group $i: ${groups[$((i - 1))]}"; fi
  cat "$out/$i"
done
exit "$status"
