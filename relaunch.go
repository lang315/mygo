//go:build !js

package mygo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo/internal/updatehook"
)

// startExe is the executable the app started from. On Linux, os.Executable
// reports the old file once an update replaced it.
var startExe, _ = os.Executable()

// relaunchNow starts the app again; it runs after the quit sequence.
func relaunchNow() {
	if launchedByDev() {
		os.Exit(devRelaunchCode)
	}
	if runtime.GOOS == "darwin" && (relaunchStaged() || openBundle()) {
		return
	}
	// The executable the app started from, which an update may have
	// replaced.
	cmd := exec.Command(startExe, os.Args[1:]...)
	cmd.Dir = startDir
	if runtime.GOOS != "windows" {
		// GUI executables on Windows have no standard streams to pass on.
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	}
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// relaunchStaged asks the update that package updates staged, if the app links
// it and has one, to open the app once it has installed itself, reporting
// whether it did.
func relaunchStaged() bool {
	f := updatehook.RelaunchStaged.Load()
	return f != nil && (*f)()
}

// openBundle starts the app bundle again through Launch Services, as the
// Dock does, reporting whether it did. A child of this process would stay
// in this instance's coalition, which has no app once this instance exits:
// when an app that the new instance started quits, as one a terminal's
// shell ran, loginwindow takes the coalition's other processes for that
// app's and ends them with SIGTERM.
func openBundle() bool {
	exe := startExe
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(bundle, ".app") {
		return false
	}
	// -n: this instance still runs, and would only be brought forward.
	args := []string{"-n", bundle}
	if len(os.Args) > 1 {
		args = append(append(args, "--args"), os.Args[1:]...)
	}
	return exec.Command("/usr/bin/open", args...).Run() == nil
}

// cleanUpdateLeftovers removes what an update could not remove while the
// previous version ran (Windows keeps running executables).
func cleanUpdateLeftovers() {
	if runtime.GOOS == "darwin" || startExe == "" {
		return
	}
	olds, _ := filepath.Glob(filepath.Join(filepath.Dir(startExe), ".*.old-*"))
	for _, p := range olds {
		_ = os.RemoveAll(p)
	}
}
