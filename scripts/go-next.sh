#!/usr/bin/env bash
# Builds and tests MyGo with a Go toolchain that is not the one in go.mod, to
# learn early that a Go upgrade breaks it. purego's fakecgo links against the
# runtime's internals (go:linkname: runtime.iscgo, runtime.set_crosscall2,
# syscall.cgo_libc_*, _cgo_*), which Go does not promise to keep, so an
# upgrade can fail to link, or to start, with no change of ours.
#
#   scripts/go-next.sh                 # gotip: the tip of Go's master
#   scripts/go-next.sh go1.28rc1       # a release or a candidate (GOTOOLCHAIN downloads it)
#   scripts/go-next.sh --quick gotip   # build, vet and link the tests only
#
# The same runs weekly and on demand in .github/workflows/go-next.yml.
set -euo pipefail

quick=0
if [ "${1:-}" = "--quick" ]; then
	quick=1
	shift
fi
toolchain="${1:-gotip}"

if [ "$toolchain" = "gotip" ]; then
	# Builds Go from source, a few minutes the first time.
	go install golang.org/dl/gotip@latest
	"$(go env GOPATH)/bin/gotip" download
	go="$(go env GOPATH)/bin/gotip"
else
	# go.mod's go line may be older than the toolchain asked for.
	export GOTOOLCHAIN="$toolchain"
	go=go
fi

step() { printf '\n== %s\n' "$*"; }

step "toolchain"
"$go" version

step "build for every platform (no cgo: purego's fakecgo is what runs)"
for os in darwin linux windows; do
	GOOS=$os CGO_ENABLED=0 "$go" build ./...
done

step "vet"
"$go" vet ./...

# A test binary links the package's imports and starts them, which is where a
# broken linkname shows (a link error, or a crash before main): build and run
# every one of them without running a test.
step "link and start every test binary"
"$go" test -count=1 -run '^$' ./...

if [ "$quick" = 1 ]; then
	exit 0
fi

step "tests"
"$go" test -count=1 ./internal/... ./ui/... .
