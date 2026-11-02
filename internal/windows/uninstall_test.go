//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/internal/update"
)

// The entry of the installed app in Settings > Apps is read and its version
// written in the real registry, under a key of its own that the test removes.
func TestUninstallEntryVersion(t *testing.T) {
	id := "com.mygo.test.uninstall." + filepath.Base(t.TempDir())
	t.Cleanup(func() { DeleteUninstall(id) })
	var u Uninstall
	if _, ok := u.Location(id); ok {
		t.Fatal("the test entry exists already")
	}
	dir := filepath.Join(t.TempDir(), "Thư mục cài đặt")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// No entry: RecordVersion creates none.
	if ok, err := update.RecordVersion(u, id, dir, "1.1.0"); ok || err != nil {
		t.Fatalf("no entry: %v, %v", ok, err)
	}
	if _, ok := UninstallValue(id, "DisplayVersion"); ok {
		t.Fatal("RecordVersion created an entry")
	}
	for name, value := range map[string]string{"InstallLocation": dir, "DisplayVersion": "1.0.0", "DisplayName": "Test"} {
		if err := SetUninstallValue(id, name, value); err != nil {
			t.Fatal(err)
		}
	}
	// Another folder: left alone.
	if ok, err := update.RecordVersion(u, id, filepath.Join(filepath.Dir(dir), "Portable"), "1.1.0"); ok || err != nil {
		t.Fatalf("another folder: %v, %v", ok, err)
	}
	if v, _ := UninstallValue(id, "DisplayVersion"); v != "1.0.0" {
		t.Errorf("DisplayVersion = %q after another folder", v)
	}
	// The folder of the entry, written as Windows may case it and with a
	// trailing backslash.
	if ok, err := update.RecordVersion(u, id, dir+`\`, "1.1.0"); !ok || err != nil {
		t.Fatalf("installed folder: %v, %v", ok, err)
	}
	if v, _ := UninstallValue(id, "DisplayVersion"); v != "1.1.0" {
		t.Errorf("DisplayVersion = %q, want 1.1.0", v)
	}
	if n, _ := UninstallValue(id, "DisplayName"); n != "Test" {
		t.Errorf("DisplayName = %q: the other values changed", n)
	}
}
