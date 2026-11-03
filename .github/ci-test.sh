#!/usr/bin/env bash
# Runs go test with the flags before -- on the packages after it. When ui or
# cmd/mygo is among them, its slowest tests (splits below) run in processes of
# their own, all at once: ./ui runs its tests one after another on one core,
# and its Testers share the process's text system, so they cannot take
# t.Parallel (Tester in ui/headless.go); the tests of cmd/mygo run one after
# another as well. Each group is a few walks of many seeds, or builds of apps;
# the rest of the tests, of every package, skip them, so that every test runs
# once. The skip is of every package given, so a name of the splits must be no
# test's name in another of them (--check looks).
#
# CI_PART=a or b splits the ui shard between two jobs (Windows has the longest
# job, and 4 cores): a runs ./ui itself, without the groups marked #b, b the
# packages below ui and the groups marked #b; between them every test runs
# once (--check looks too).
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
# #a and #b say which job of CI_PART has a group.
splits=(
  'ui#a:TestPaintCullMatches|TestRetainSliderRowsRandomWalk|TestRetainShiftPaintMixedScales|TestRetainComboboxRowsRandomWalk|TestPureDrawsReachNoNode'
  'ui#b:TestLayoutCacheMatches|TestRetainScrollFuzz|TestRetainFocusableRowsRandomWalk|TestRetainLayoutInnerScrollModelSeed'
  'ui#a:TestRetainSwitchRowsRandomWalk|TestRetainRowsOfTextInputUsersRandomWalk|TestRetainSelectProgressRowsRandomWalk|TestRetainMatchesFresh'
  'ui#b:TestRetainRandomWalkOfReaders|TestReplayMatches'
  'cmd/mygo@4:TestUpdateEndToEnd|TestSkipInstaller'
  'cmd/mygo@4:TestBuildResources|TestBuildDMGSize|TestBuildDMG'
  'cmd/mygo@4:TestBuildLinuxSlug|TestRev338eScanCost|TestResourceErrors'
)
mod=$(go list -m)

# The packages the main process of part $1 runs, of those after it: with no
# part, all; a, ./ui alone; b, the others.
part_pkgs() {
  local part=$1 p
  shift
  for p in "$@"; do
    case $part in
    a) [ "$p" = "$mod/ui" ] && echo "$p" ;;
    b) [ "$p" = "$mod/ui" ] || echo "$p" ;;
    *) echo "$p" ;;
    esac
  done
  return 0
}

if [ "${1:-}" = --check ]; then
  all=$(go list ./...)
  fail=0
  seen=$'\n' in_a=0 in_b=0
  for s in "${splits[@]}"; do
    pkg=${s%%:*} tag=""
    case $pkg in *"#"*) tag=${pkg#*#} pkg=${pkg%%#*} ;; esac
    pkg=${pkg%%@*}
    if [ "$pkg" = ui ]; then
      case $tag in a) in_a=1 ;; b) in_b=1 ;; *) echo "ci-test.sh: a group of ui needs #a or #b" >&2; fail=1 ;; esac
    elif [ -n "$tag" ]; then
      echo "ci-test.sh: only the groups of ui have a part" >&2; fail=1
    fi
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
  # Both parts have groups, and the packages of their main processes are those
  # of the ui shard, each in one.
  [ "$in_a" = 1 ] && [ "$in_b" = 1 ] || { echo "ci-test.sh: each part of ui needs a group" >&2; fail=1; }
  shard=$(grep -E "^$mod/ui(/|\$)" <<<"$all")
  pa=$(part_pkgs a $shard) pb=$(part_pkgs b $shard)
  if [ -z "$pa" ] || [ -z "$pb" ] || [ "$(sort <<<"$pa"$'\n'"$pb")" != "$(sort <<<"$shard")" ]; then
    echo "ci-test.sh: the parts a and b must run the packages of the ui shard between them, each in one" >&2
    fail=1
  fi
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
  p=${s%%:*} min=0 tag=""
  case $p in *"#"*) tag=${p#*#} p=${p%%#*} ;; esac
  case $p in *@*) min=${p#*@} p=${p%%@*} ;; esac
  p=$mod/$p
  if has "$p" && [ "$cores" -ge "$min" ]; then
    # Every name is skipped by the main process; the part has its groups.
    names+=("${s#*:}")
    if [ -z "$tag" ] || [ -z "${CI_PART:-}" ] || [ "$tag" = "${CI_PART:-}" ]; then groups+=("${s#*:}"); gpkgs+=("$p"); fi
  fi
done
main=($(part_pkgs "${CI_PART:-}" "${pkgs[@]}"))
all=$(IFS='|'; echo "${names[*]:-}")
# One process, as before, when there is nothing to split.
if [ "${#groups[@]}" = 0 ]; then
  if [ "${#main[@]}" = 0 ]; then echo "Nothing of the part ${CI_PART:-} to test."; exit 0; fi
  exec go test ${flags[@]+"${flags[@]}"} "${main[@]}"
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
    have=$(awk -v pkg="${gpkgs[$i]}" '$1 == "ok" { if ($2 == pkg) { print last; exit } last = ""; next } { last = last "\n" $0 }' <<<"$listed")
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
if [ "${#main[@]}" -gt 0 ]; then start "" go test ${flags[@]+"${flags[@]}"} "${skip[@]}" "${main[@]}"; fi
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
