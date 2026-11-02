//go:build windows && (amd64 || arm64)

package windows

// uninstallKey is the registry key of the entry of an app in Settings > Apps
// that the Windows installer writes (cmd/mygo nsisUninstallEntry), by the
// app's identifier. This package does not import internal/update, which
// package mygo, linking this one, must not link.
func uninstallKey(identifier string) string {
	return `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + identifier
}

// Uninstall is the entry of the installed app in Settings > Apps
// (HKEY_CURRENT_USER\...\Uninstall\<identifier>), where the installer wrote
// its version.
type Uninstall struct{}

// Location returns the InstallLocation of the entry of the identifier.
func (Uninstall) Location(identifier string) (string, bool) {
	return regGet(hkeyCurrentUser, uninstallKey(identifier), "InstallLocation")
}

// SetVersion sets the DisplayVersion of the entry of the identifier.
func (Uninstall) SetVersion(identifier, version string) error {
	return regSet(hkeyCurrentUser, uninstallKey(identifier), "DisplayVersion", version)
}

// DeleteUninstall removes the entry of the identifier, for tests.
func DeleteUninstall(identifier string) error {
	return regDeleteTree(hkeyCurrentUser, uninstallKey(identifier))
}

// SetUninstallValue writes a value of the entry of the identifier, for tests.
func SetUninstallValue(identifier, name, value string) error {
	return regSet(hkeyCurrentUser, uninstallKey(identifier), name, value)
}

// UninstallValue reads a value of the entry of the identifier.
func UninstallValue(identifier, name string) (string, bool) {
	return regGet(hkeyCurrentUser, uninstallKey(identifier), name)
}
