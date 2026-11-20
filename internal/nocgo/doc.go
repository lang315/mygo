// Package nocgo holds the test that no Go file of the repository imports
// "C" outside the directories cgoAllowed names: desktops and browsers build
// with CGO_ENABLED=0 and reach native code through purego and syscall, and
// only the glue phones link with their own toolchains may use cgo
// (AGENTS.md, "Hard rules"). The test reads the source whatever its build
// constraints, so it runs on any host.
package nocgo
