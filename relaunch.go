//go:build !js

package mygo

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo/internal/fsretry"
	"github.com/egoist/mygo/internal/updatehook"
)

// startExe is the executable the app started from. On Linux, os.Executable
// reports the old file once an update replaced it.
var startExe, _ = os.Executable()

// relaunchNow starts the app again; it runs after the quit sequence.
func relaunchNow() {
	args := relaunchArguments()
	if launchedByDev() {
		devRelaunch(App.relaunchArgs)
	}
	if runtime.GOOS == "darwin" && (relaunchStaged(args) || openBundle(args)) {
		return
	}
	// The executable the app started from, which an update may have
	// replaced.
	cmd := exec.Command(startExe, args...)
	cmd.Dir = startDir
	if runtime.GOOS != "windows" {
		// GUI executables on Windows have no standard streams to pass on.
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	}
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// devRelaunch exits for `mygo dev` to start the build again, with args
// unless they are nil, which it reads from the file that
// MYGO_DEV_RELAUNCH names.
func devRelaunch(args []string) {
	if args != nil {
		if devRelaunchFile == "" {
			log.Print("mygo: this mygo dev cannot pass Relaunch's arguments to the build; update it")
		} else if err := os.WriteFile(devRelaunchFile, devRelaunchData(args), 0o600); err != nil {
			log.Printf("mygo: cannot pass Relaunch's arguments to mygo dev: %v", err)
		}
	}
	os.Exit(devRelaunchCode)
}

// devRelaunchData is what devRelaunch writes: each argument, ended with
// a NUL byte, which no argument holds.
func devRelaunchData(args []string) []byte {
	var b []byte
	for _, a := range args {
		b = append(append(b, a...), 0)
	}
	return b
}

// relaunchStaged asks the update that package updates staged, if the app links
// it and has one, to open the app once it has installed itself, with args,
// reporting whether it did.
func relaunchStaged(args []string) bool {
	f := updatehook.RelaunchStaged.Load()
	return f != nil && (*f)(args)
}

// openBundle starts the app bundle again through Launch Services, as the
// Dock does, with args, reporting whether it did. A child of this process would stay
// in this instance's coalition, which has no app once this instance exits:
// when an app that the new instance started quits, as one a terminal's
// shell ran, loginwindow takes the coalition's other processes for that
// app's and ends them with SIGTERM.
func openBundle(args []string) bool {
	exe := startExe
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(bundle, ".app") {
		return false
	}
	// -n: this instance still runs, and would only be brought forward.
	open := []string{"-n", bundle}
	if len(args) > 0 {
		open = append(append(open, "--args"), args...)
	}
	return exec.Command("/usr/bin/open", open...).Run() == nil
}

// cleanUpdateLeftovers removes what an update could not remove while the
// previous version ran (Windows keeps running executables). A version that
// an update relaunched starts while the one that installed it still quits, and
// its executable stays in use until then: fsretry waits for it for a few
// seconds, and what is left is removed the next time the app starts.
func cleanUpdateLeftovers() {
	if runtime.GOOS == "darwin" || startExe == "" {
		return
	}
	olds, _ := filepath.Glob(filepath.Join(filepath.Dir(startExe), ".*.old-*"))
	for _, p := range olds {
		_ = fsretry.RetryInUse(func() error { return os.RemoveAll(p) })
	}
}
