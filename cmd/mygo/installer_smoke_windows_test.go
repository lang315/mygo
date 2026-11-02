package main

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/wait"
)

// TestInstallerSmoke is the path of a user on Windows, with a real app:
// mygo build makes the installer of examples/hello, a silent install puts
// the app in a folder, the app runs and shows its window and quits when
// the window is closed, a newer installer run over the folder while the app
// runs quits the old process and replaces its files, and the uninstaller
// leaves nothing. TestWindowsInstaller covers the installer's files and
// registrations with an app that never runs.
//
// It starts a window and needs the WebView2 Runtime, so it runs only with
// MYGO_E2E=1 (the Windows installer job of CI), on windows/amd64.
func TestInstallerSmoke(t *testing.T) {
	if os.Getenv("MYGO_E2E") != "1" {
		t.Skip("opens windows: set MYGO_E2E=1 on Windows")
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("the installer is built for amd64")
	}
	hello, err := os.ReadFile(filepath.Join("..", "..", "examples", "hello", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	const identifier = "com.example.smokeapp"
	build := func(version string) string {
		_, _, artifacts := buildTestInstaller(t, map[string]string{
			"main.go":   string(hello),
			"mygo.json": `{"name": "Smoke App", "identifier": "` + identifier + `", "version": "` + version + `"}`,
		})
		return artifacts[len(artifacts)-1]
	}
	setup1, setup2 := build("1.0.0"), build("2.0.0")

	install := filepath.Join(t.TempDir(), "SmokeApp")
	exe := filepath.Join(install, "Smoke App.exe")
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\` + identifier
	displayVersion := func() string {
		out, _ := exec.Command("reg", "query", key, "/v", "DisplayVersion").CombinedOutput()
		return string(out)
	}
	silentInstall := func(setup string) {
		t.Helper()
		// NSIS takes /D= unquoted, and Go quotes arguments with spaces.
		if out, err := exec.Command(setup, "/S", "/D="+install).CombinedOutput(); err != nil {
			t.Fatalf("installing %s: %v\n%s", filepath.Base(setup), err, out)
		}
	}
	t.Cleanup(func() {
		// Best effort, if the test failed before its own uninstall.
		if fileExists(filepath.Join(install, "Uninstall.exe")) {
			exec.Command(filepath.Join(install, "Uninstall.exe"), "/S").Run()
		}
	})

	silentInstall(setup1)
	for _, f := range []string{"Smoke App.exe", "Uninstall.exe"} {
		if !fileExists(filepath.Join(install, f)) {
			t.Fatalf("not installed: %s", f)
		}
	}
	if !strings.Contains(displayVersion(), "1.0.0") {
		t.Fatalf("Settings > Apps lists %q, want version 1.0.0", displayVersion())
	}
	before := fileHash(t, exe)

	// The installed app runs, shows a window and quits when it is closed.
	app := startApp(t, exe)
	app.waitWindow(t)
	app.closeWindow(t)
	if err := wait.For(t, "the app to quit after its window closed", app.done); err != nil {
		t.Fatalf("the app quit with %v after its window closed", err)
	}

	// The newer installer runs over the folder while the app runs.
	app = startApp(t, exe)
	app.waitWindow(t)
	silentInstall(setup2)
	wait.For(t, "the old app to be quit by the installer", app.done)
	if fileHash(t, exe) == before {
		t.Error("the installer did not replace the app's executable")
	}
	if !strings.Contains(displayVersion(), "2.0.0") {
		t.Errorf("Settings > Apps lists %q, want version 2.0.0", displayVersion())
	}

	// The new version runs.
	app = startApp(t, exe)
	app.waitWindow(t)
	app.closeWindow(t)
	if err := wait.For(t, "the new app to quit after its window closed", app.done); err != nil {
		t.Fatalf("the new app quit with %v after its window closed", err)
	}

	uninstall(t, install)
	if out, err := exec.Command("reg", "query", key).CombinedOutput(); err == nil {
		t.Errorf("the uninstaller left %s:\n%s", key, out)
	}
}

func fileHash(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

// A running app: its process, and done, which gets the result of Wait when
// it exits.
type smokeApp struct {
	cmd  *exec.Cmd
	done chan error
}

func startApp(t *testing.T, exe string) *smokeApp {
	t.Helper()
	a := &smokeApp{cmd: exec.Command(exe), done: make(chan error, 1)}
	if err := a.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { a.done <- a.cmd.Wait() }()
	t.Cleanup(func() { a.cmd.Process.Kill() }) // before the folder goes: Cleanup runs last in, first out
	return a
}

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procPostMessage              = user32.NewProc("PostMessageW")
)

const wmClose = 0x0010

// windows returns the visible top-level windows of the app's own process:
// its web view's processes have windows of their own.
func (a *smokeApp) windows() []uintptr {
	var found []uintptr
	pid := uint32(a.cmd.Process.Pid)
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var owner uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
		if visible, _, _ := procIsWindowVisible.Call(hwnd); owner == pid && visible != 0 {
			found = append(found, hwnd)
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

func (a *smokeApp) waitWindow(t *testing.T) {
	t.Helper()
	wait.Until(t, "the app's window to show", func() bool {
		select {
		case err := <-a.done:
			t.Fatalf("the app quit before its window showed: %v", err)
		default:
		}
		return len(a.windows()) > 0
	})
}

func (a *smokeApp) closeWindow(t *testing.T) {
	t.Helper()
	for _, hwnd := range a.windows() {
		procPostMessage.Call(hwnd, wmClose, 0, 0)
	}
}
