//go:build !js && !(windows && (amd64 || arm64))

package updates

import "github.com/egoist/mygo/internal/update"

// uninstallEntry is nil: only Windows has the entry.
var uninstallEntry update.Registry
