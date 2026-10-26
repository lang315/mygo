//go:build darwin

package ui

import "github.com/egoist/mygo/internal/darwin"

// linkMenuIcons links the backend's code of the icons of menu items, which
// an app that gives none does not carry (Item.Icon, Item.IconPNG).
func linkMenuIcons() { darwin.LinkMenuIcons() }
