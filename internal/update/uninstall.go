package update

import (
	"path/filepath"
	"strings"
)

// UninstallKeyPath is the registry key, under HKEY_CURRENT_USER, of the entry
// of an app in Settings > Apps that the Windows installer writes
// (cmd/mygo nsisUninstallEntry), by the app's identifier.
func UninstallKeyPath(identifier string) string {
	return `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + identifier
}

// SameInstallDir reports whether the InstallLocation an installer wrote and
// the directory of the running executable are one folder: Windows paths are
// compared without case, and without a trailing backslash or quotes.
func SameInstallDir(location, dir string) bool {
	clean := func(p string) string {
		p = strings.Trim(strings.TrimSpace(p), `"`)
		if p == "" {
			return ""
		}
		return strings.ToLower(strings.TrimRight(filepath.Clean(strings.ReplaceAll(p, "/", `\`)), `\`))
	}
	a, b := clean(location), clean(dir)
	return a != "" && a == b
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
