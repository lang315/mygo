package update

import (
	"path/filepath"
	"strings"
)

// SameInstallDir reports whether the InstallLocation an installer wrote and
// the directory of the running executable are one folder: Windows paths are
// compared without case, and without a trailing backslash or quotes. The
// installer writes the folder as the user gave it (/D=, a short name such as
// C:\Users\RUNNER~1), while the running executable's folder is the real one,
// in long names: when the spellings differ, both are resolved
// (filepath.EvalSymlinks gives Windows paths their long names).
func SameInstallDir(location, dir string) bool {
	trim := func(p string) string { return strings.Trim(strings.TrimSpace(p), `"`) }
	clean := func(p string) string {
		if p == "" {
			return ""
		}
		return strings.ToLower(strings.TrimRight(filepath.Clean(strings.ReplaceAll(p, "/", `\`)), `\`))
	}
	resolved := func(p string) string {
		if p == "" {
			return ""
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		return clean(p)
	}
	location, dir = trim(location), trim(dir)
	a, b := clean(location), clean(dir)
	return a != "" && b != "" && (a == b || resolved(location) == resolved(dir))
}

// Registry is the entry of an app in Settings > Apps, as RecordVersion needs
// it.
type Registry interface {
	// Location returns the InstallLocation of the entry of the identifier,
	// and whether the entry exists.
	Location(identifier string) (string, bool)
	// SetVersion sets its DisplayVersion.
	SetVersion(identifier, version string) error
}

// RecordVersion makes the entry of the installed app say it is version, after
// an update replaced the files in dir: only an entry that exists and whose
// InstallLocation is dir (a copy of the app elsewhere is not the installed
// one). It reports whether it wrote the version; a failure to write is the
// caller's to log, not the update's to fail.
func RecordVersion(r Registry, identifier, dir, version string) (bool, error) {
	if identifier == "" || version == "" {
		return false, nil
	}
	loc, ok := r.Location(identifier)
	if !ok || !SameInstallDir(loc, dir) {
		return false, nil
	}
	if err := r.SetVersion(identifier, version); err != nil {
		return false, err
	}
	return true, nil
}
