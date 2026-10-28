#!/usr/bin/env bash
# Prints the packages of a shard of the unit tests, one a line: cmd, ui and
# internal take theirs, and rest everything else, so that the four are
# `go list ./...` between them. The jobs of ci.yml run one each, together.
set -euo pipefail
mod=$(go list -m)
all=$(go list ./...)
case "${1:-}" in
cmd | ui | internal) grep -E "^$mod/$1(/|\$)" <<<"$all" ;;
rest) grep -v -E "^$mod/(cmd|ui|internal)(/|\$)" <<<"$all" ;;
*) echo "usage: ci-shard.sh cmd|ui|internal|rest" >&2; exit 2 ;;
esac
