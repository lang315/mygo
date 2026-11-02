//go:build windows && (amd64 || arm64)

package windows

import "github.com/egoist/mygo/internal/update"

// Uninstall is the entry of the installed app in Settings > Apps
// (HKEY_CURRENT_USER\...\Uninstall\<identifier>), where the installer wrote
// its version.
type Uninstall struct{}

// Location returns the InstallLocation of the entry of the identifier.
func (Uninstall) Location(identifier string) (string, bool) {
	return regGet(hkeyCurrentUser, update.UninstallKeyPath(identifier), "InstallLocation")
}

// SetVersion sets the DisplayVersion of the entry of the identifier.
func (Uninstall) SetVersion(identifier, version string) error {
	return regSet(hkeyCurrentUser, update.UninstallKeyPath(identifier), "DisplayVersion", version)
}

// DeleteUninstall removes the entry of the identifier, for tests.
func DeleteUninstall(identifier string) error {
	return regDeleteTree(hkeyCurrentUser, update.UninstallKeyPath(identifier))
}

// SetUninstallValue writes a value of the entry of the identifier, for tests.
func SetUninstallValue(identifier, name, value string) error {
	return regSet(hkeyCurrentUser, update.UninstallKeyPath(identifier), name, value)
}

// UninstallValue reads a value of the entry of the identifier.
func UninstallValue(identifier, name string) (string, bool) {
	return regGet(hkeyCurrentUser, update.UninstallKeyPath(identifier), name)
}
