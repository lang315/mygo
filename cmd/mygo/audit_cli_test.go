package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestNSISInstallsIntoTheAppFolderOfAFolderWithFiles(t *testing.T) {
	c := &Config{Name: "My App", Version: "1.0.0", Identifier: "com.example.app"}
	script, err := installerScript(c, "setup.exe", "My App.exe", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	_, section, _ := strings.Cut(script, "\nSection\n")
	section, _, _ = strings.Cut(section, "SectionEnd")
	want := []string{
		`IfFileExists "$INSTDIR\Uninstall.exe" mygo_dir_ok`,
		`FindFirst $0 $1 "$INSTDIR\*.*"`,
		`StrCmp $1 "." mygo_scan_next`,
		`StrCmp $1 ".." mygo_scan_next`,
		`StrCpy $INSTDIR "$INSTDIR\My App"`,
	}
	last := strings.Index(section, "WriteUninstaller")
	for _, s := range want {
		i := strings.Index(section, s)
		if i < 0 || i > last {
			t.Errorf("no %q before WriteUninstaller in:\n%s", s, section)
		}
	}
	if strings.Index(section, "StrCpy $INSTDIR") > strings.Index(section, "SetOutPath") {
		t.Error("the folder is chosen after files are written")
	}
}

func TestValidateRefusesABlankName(t *testing.T) {
	if err := (&Config{Name: "   "}).validate(); err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("blank name: %v", err)
	}
	if err := (&Config{Name: "   ", Windows: Windows{InstallDirName: "App"}}).validate(); err != nil {
		t.Errorf("blank name with installDirName: %v", err)
	}
}

func TestValidateRefusesControlCharacters(t *testing.T) {
	for field, c := range map[string]*Config{
		"name":             {Name: "A\nB"},
		"linux.comment":    {Name: "A", Linux: Linux{Comment: "x\nExec=evil"}},
		"linux.categories": {Name: "A", Linux: Linux{Categories: []string{"Utility\n"}}},
	} {
		if err := c.validate(); err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("%s: %v", field, err)
		}
	}
}

func TestDebVersionSanitized(t *testing.T) {
	re := regexp.MustCompile(`^[A-Za-z0-9.+~]+$`)
	for _, v := range []string{"1.0_beta 2", "1.0/../x", "v1.2.0-rc.1+b/5", "///"} {
		if got := debVersion(v); !re.MatchString(got) {
			t.Errorf("debVersion(%q) = %q", v, got)
		}
	}
	if got := debVersion("1.2.0-rc.1"); got != "1.2.0~rc.1" {
		t.Errorf("debVersion = %q", got)
	}
}

func TestDebFileNameStaysInStage(t *testing.T) {
	c := &Config{root: t.TempDir(), Name: "My App", Version: "1.0/../x"}
	stage := t.TempDir()
	if err := os.WriteFile(filepath.Join(stage, "my-app"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	deb, err := writeDeb(c, stage, "my/app", "amd64", []string{"my-app"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(deb) != stage {
		t.Errorf("the deb is at %s, not in %s", deb, stage)
	}
}

func TestTempBinaryIsPrivate(t *testing.T) {
	a, cleanA, err := tempBinary("My App")
	if err != nil {
		t.Fatal(err)
	}
	b, cleanB, err := tempBinary("My App")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(a) == filepath.Dir(b) {
		t.Fatal("the same folder twice")
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(filepath.Dir(a)); err != nil || info.Mode().Perm() != 0o700 {
			t.Errorf("mode %v, %v", info, err)
		}
	}
	cleanA()
	cleanB()
	for _, p := range []string{a, b} {
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			t.Errorf("%s was not removed", filepath.Dir(p))
		}
	}
}

// TestNotarizeWithoutDMG checks that -skip-dmg does not skip the notarization
// of the app, which the update archive and the upload ship.
func TestNotarizeWithoutDMG(t *testing.T) {
	if testing.Short() || runtime.GOOS != "darwin" {
		t.Skip("compiles a program, notarizes on macOS")
	}
	dir := testModule(t, map[string]string{
		"main.go":   "package main\n\nimport \"github.com/egoist/mygo\"\n\nfunc main() { mygo.App.Run() }\n",
		"mygo.json": `{"name": "Notary App", "macos": {"notarize": {"keychainProfile": "p"}}}`,
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildOptions{sign: "-", skipDMG: true, work: t.TempDir()}
	if opts.pkg, err = packageDir(c); err != nil {
		t.Fatal(err)
	}
	var apps []string
	defer func(f func(*Notarize, string) error) { notarizeAppStep = f }(notarizeAppStep)
	notarizeAppStep = func(n *Notarize, app string) error {
		apps = append(apps, filepath.Base(app))
		return nil
	}
	if _, err := buildPlatform(c, "darwin", "arm64", opts); err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0] != "Notary App.app" {
		t.Errorf("notarized %q", apps)
	}
	opts.skipNotarize = true
	apps = nil
	if _, err := buildPlatform(c, "darwin", "arm64", opts); err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Errorf("-skip-notarize notarized %q", apps)
	}
}
