//go:build linux && (amd64 || arm64)

package ui

import "github.com/egoist/mygo/internal/linux"

// linkMenuIcons links the backend's code of the icons of menu items, which
// an app that gives none does not carry (Item.Icon, Item.IconPNG).
func linkMenuIcons() { linux.LinkMenuIcons() }
