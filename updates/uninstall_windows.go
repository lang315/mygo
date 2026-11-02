//go:build windows && (amd64 || arm64)

package updates

import (
	"github.com/egoist/mygo/internal/update"
	"github.com/egoist/mygo/internal/windows"
)

// uninstallEntry is the entry of the installed app in Settings > Apps.
var uninstallEntry update.Registry = windows.Uninstall{}
