package update

import (
	"errors"
	"testing"
)

type fakeRegistry struct {
	entries map[string]string // identifier: InstallLocation
	version map[string]string
	err     error
}

func (f *fakeRegistry) Location(id string) (string, bool) { l, ok := f.entries[id]; return l, ok }
func (f *fakeRegistry) SetVersion(id, v string) error {
	if f.err != nil {
		return f.err
	}
	f.version[id] = v
	return nil
}

func TestSameInstallDir(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{`D:\Apps\My App`, `D:\Apps\My App`, true},
		{`D:\Apps\My App\`, `d:\apps\my app`, true},
		{`"D:\Apps\My App"`, `D:\Apps\My App\`, true},
		{`D:/Apps/My App`, `D:\Apps\My App`, true},
		{`D:\Thư mục cài đặt`, `D:\THƯ MỤC cài ĐẶT`, true},
		{`D:\Apps\My App`, `D:\Apps\Other`, false},
		{`D:\Apps\My App`, `D:\Apps\My App\sub`, false},
		{``, ``, false},
		{`D:\Apps`, ``, false},
	} {
		if got := SameInstallDir(c.a, c.b); got != c.want {
			t.Errorf("SameInstallDir(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestRecordVersion(t *testing.T) {
	const id = "com.example.app"
	const dir = `D:\Apps\App`
	reg := func() *fakeRegistry {
		return &fakeRegistry{entries: map[string]string{id: dir}, version: map[string]string{}}
	}
	r := reg()
	if ok, err := RecordVersion(r, id, dir+`\`, "1.1.0"); !ok || err != nil || r.version[id] != "1.1.0" {
		t.Errorf("installed copy: %v, %v, %v", ok, err, r.version)
	}
	// Another copy of the app, a portable one, does not take the entry.
	r = reg()
	if ok, err := RecordVersion(r, id, `E:\Portable\App`, "1.1.0"); ok || err != nil || len(r.version) != 0 {
		t.Errorf("another copy: %v, %v, %v", ok, err, r.version)
	}
	// No entry: nothing is created.
	r = reg()
	if ok, err := RecordVersion(r, "com.example.other", dir, "1.1.0"); ok || err != nil || len(r.version) != 0 {
		t.Errorf("no entry: %v, %v, %v", ok, err, r.version)
	}
	if ok, err := RecordVersion(r, "", dir, "1.1.0"); ok || err != nil {
		t.Errorf("no identifier: %v, %v", ok, err)
	}
	// A write that fails is reported for the caller to log.
	r = reg()
	r.err = errors.New("access denied")
	if ok, err := RecordVersion(r, id, dir, "1.1.0"); ok || err == nil {
		t.Errorf("failed write: %v, %v", ok, err)
	}
}
