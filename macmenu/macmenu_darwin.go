//go:build darwin && !ios

package macmenu

import "github.com/egoist/mygo/internal/darwin"

func init() { darwin.LinkMenuExtras() }
