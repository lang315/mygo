#!/usr/bin/env bash
# Runs go test with the flags before -- on the packages after it, in several
# processes at once, so that the cores of the runner are all busy from the
# start and no long test or package is left for the end:
#
#  - The slowest tests of some packages (splits below) run in processes of
#    their own, each a few tests that walk many seeds or build apps: ./ui
#    runs its tests one after another on one core, and its Testers share the
#    process's text system, so they cannot take t.Parallel (Tester in
#    ui/headless.go); the tests of cmd/mygo run one after another as well.
#    Every other test of the packages runs in the main process, which skips
#    them, so that every test runs once. The skip is of every package the
#    main process has, so a name of the splits must be the name of no test
#    in another of them (--check looks).
#  - go test starts packages in the order it is given them, so the slowest
#    packages (slowest below) go first and the quick ones fill the cores
#    as these are busy: with the packages in the order of their names, one
#    that starts late, as internal/raster, is what the job waits for.
#
#   ci-test.sh -race -shuffle=on -- $(bash .github/ci-shard.sh ui)
#   ci-test.sh --check     # the lists below against the packages and tests
#
# CI_TEST_FLAGS adds flags to every process (-json lets a run be counted).
set -euo pipefail

# package:names of its tests, about 3 minutes each under -race on an
# M-series core (cmd/mygo: 1 to 1.5; its tests run the Go toolchain, which
# uses every core, so on a runner of fewer than 4 cores, macOS's 3, their
# processes only slow one another down: cmd/mygo@4 splits from 4 cores).
splits=(
  'ui:TestPaintCullMatches|TestRetainSliderRowsRandomWalk|TestRetainShiftPaintMixedScales|TestRetainComboboxRowsRandomWalk|TestPureDrawsReachNoNode'
  'ui:TestLayoutCacheMatches|TestRetainScrollFuzz|TestRetainFocusableRowsRandomWalk|TestRetainLayoutInnerScrollModelSeed'
  'ui:TestRetainSwitchRowsRandomWalk|TestRetainRowsOfTextInputUsersRandomWalk|TestRetainSelectProgressRowsRandomWalk|TestRetainMatchesFresh'
  'ui:TestRetainRandomWalkOfReaders|TestReplayMatches'
  'cmd/mygo@4:TestUpdateEndToEnd|TestSkipInstaller'
  'cmd/mygo@4:TestBuildResources|TestBuildDMGSize|TestBuildDMG'
  'cmd/mygo@4:TestBuildLinuxSlug|TestRev338eScanCost|TestResourceErrors'
)
# The packages that take the longest under -race, longest first (the
# order of the slowest runner of each); those of another system or another
# shard are not among the arguments, and are left out.
slowest=(
  internal/gpu/metal internal/raster internal/gpu/gl ui examples/wind
  internal/e2e internal/gpu ui/widget internal/gpu/d3d11 ui/markdown/parser
  plugins/terminal examples/account-manager ui/editor cmd/mygo .
)

mod=$(go list -m)

if [ "${1:-}" = --check ]; then
  all=$(go list ./...)
  fail=0
  seen=$'\n' seen_slowest=$'\n'
  for s in "${splits[@]}"; do
    pkg=${s%%:*} pkg=${pkg%%@*}
    if ! grep -qx "$mod/$pkg" <<<"$all"; then echo "ci-test.sh: split package $pkg is no package" >&2; fail=1; continue; fi
    dir=$pkg
    IFS='|' read -ra names <<<"${s#*:}"
    for n in "${names[@]}"; do
      if grep -qxF "$n" <<<"$seen"; then echo "ci-test.sh: $n is in two splits" >&2; fail=1; fi
      seen+="$n"$'\n'
      # Defined in the package (a Test function of a _test.go file of its
      # directory), and in no other directory.
      files=$(git ls-files -z '*_test.go' | xargs -0 grep -lE "^func $n\\(" || true)
      if [ -z "$files" ] || [ "$(dirname -- "$files" | sort -u | tr '\n' ' ')" != "$dir " ]; then
        echo "ci-test.sh: $n must be a test of $pkg alone; it is in: ${files:-nowhere}" >&2
        fail=1
      fi
    done
  done
  for p in "${slowest[@]}"; do
    if grep -qxF "$p" <<<"$seen_slowest"; then echo "ci-test.sh: slowest package $p is listed twice" >&2; fail=1; fi
    seen_slowest+="$p"$'\n'
    # A directory: a package of another system is no package of this one.
    if [ ! -d "./$p" ]; then echo "ci-test.sh: slowest package $p is no directory" >&2; fail=1; fi
  done
  exit "$fail"
fi

flags=()
while [ "$#" -gt 0 ] && [ "$1" != -- ]; do flags+=("$1"); shift; done
if [ "$#" -gt 0 ]; then shift; fi
listflags=(${flags[@]+"${flags[@]}"})
# shellcheck disable=SC2206
[ -n "${CI_TEST_FLAGS:-}" ] && flags+=($CI_TEST_FLAGS)
pkgs=("$@")
has() { printf '%s\n' "${pkgs[@]}" | grep -qxF "$1"; }

# The splits whose package is among the arguments (and which this many cores
# split), and every name of them.
cores=$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc)
groups=() gpkgs=() names=()
for s in "${splits[@]}"; do
  p=${s%%:*} min=0
  case $p in *@*) min=${p#*@} p=${p%%@*} ;; esac
  p=$mod/$p
  if has "$p" && [ "$cores" -ge "$min" ]; then groups+=("${s#*:}"); gpkgs+=("$p"); names+=("${s#*:}"); fi
done
all=$(IFS='|'; echo "${names[*]:-}")
# The arguments, the slowest first.
ordered=()
for h in "${slowest[@]}"; do
  if [ "$h" = . ]; then q=$mod; else q=$mod/$h; fi
  if has "$q"; then ordered+=("$q"); fi
done
for p in "${pkgs[@]}"; do
  grep -qxF "$p" < <(printf '%s\n' ${ordered[@]+"${ordered[@]}"}) || ordered+=("$p")
done
# One process, as before, when there is nothing to split.
if [ "${#groups[@]}" = 0 ]; then
  exec go test ${flags[@]+"${flags[@]}"} "${ordered[@]}"
fi

if [ "${#groups[@]}" -gt 0 ]; then
  # Built once, before the processes would each build it. A name of the
  # splits that is no test of its package any more (renamed, removed) fails
  # the run: its group would still pass, with fewer tests, and the split
  # drift unseen.
  listed=$(go test ${listflags[@]+"${listflags[@]}"} -list "^($all)\$" $(printf '%s\n' "${gpkgs[@]}" | sort -u))
  missing=""
  for i in "${!groups[@]}"; do
    # The names listed before the line of their package are its own.
    have=$(awk -v pkg="${gpkgs[$i]}" '$1 == "ok" && $2 == pkg { print last; exit } { last = last "\n" $0 }' <<<"$listed")
    IFS='|' read -ra ns <<<"${groups[$i]}"
    for n in "${ns[@]}"; do
      grep -qxF "$n" <<<"$have" || missing+="${gpkgs[$i]}: $n"$'\n'
    done
  done
  if [ -n "$missing" ]; then
    echo "ci-test.sh: these names of its splits are no test of their package; rename or drop them:" >&2
    printf '%s' "$missing" >&2
    exit 1
  fi
fi
out=$(mktemp -d)
pids=() titles=()
start() { # title, then the go test command
  local n=${#pids[@]} title=$1
  shift
  "$@" >"$out/$n" 2>&1 &
  pids+=($!)
  titles+=("$title")
}
skip=(-skip "^($all)\$")
start "--- the other tests of the packages (-p $cores)" go test ${flags[@]+"${flags[@]}"} "${skip[@]}" "${ordered[@]}"
for i in "${!groups[@]}"; do
  k=$(printf '%s\n' "${gpkgs[@]:0:$i}" | grep -cxF "${gpkgs[$i]}" || true)
  start "--- ${gpkgs[$i]#"$mod/"}, group $((k + 1)): ${groups[$i]}" go test ${flags[@]+"${flags[@]}"} -run "^(${groups[$i]})\$" "${gpkgs[$i]}"
done
status=0
for i in "${!pids[@]}"; do
  wait "${pids[$i]}" || status=1
  if [ -n "${titles[$i]}" ]; then echo "${titles[$i]}"; fi
  cat "$out/$i"
done
exit "$status"
