//go:build windows && (amd64 || arm64)

package ui

import win "github.com/egoist/mygo/internal/windows"

// linkMenuIcons links the backend's code of the icons of menu items, which
// an app that gives none does not carry (Item.Icon, Item.IconPNG).
func linkMenuIcons() { win.LinkMenuIcons() }
