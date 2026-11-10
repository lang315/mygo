#!/usr/bin/env bash
# usage: measure-size.sh <package> [tags] [goos]
# Builds an app as `mygo build` and the benchmarks do (AGENTS.md), under heavy.lock,
# into a temp dir outside the repo, and prints its size in bytes.
#   measure-size.sh ./examples/counter-native mygo_noinspector,mygo_nowebview
#   measure-size.sh ./examples/hello                  # tags default: mygo_noinspector
#   measure-size.sh ./examples/hello mygo_noinspector windows
set -euo pipefail
pkg="${1:?usage: measure-size.sh <package> [tags] [goos]}"
tags="${2:-mygo_noinspector}"
goos="${3:-$(go env GOOS)}"
ldflags="-s -w -X github.com/egoist/mygo.production=1"
[ "$goos" = windows ] && ldflags="$ldflags -H=windowsgui"
out="$(mktemp -d)"
trap 'rm -r "$out"' EXIT
"$(dirname "$0")/heavy-lock" GOOS="$goos" go build -trimpath -tags "$tags" -ldflags "$ldflags" -o "$out/app" "$pkg"
wc -c <"$out/app" | tr -d ' '
