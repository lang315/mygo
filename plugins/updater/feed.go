//go:build !js

package updater

// The update window checks the feed of mygo.config.ts, which mygo.Updater
// reads through package updatefeed, linked here so that an app that uses
// the window needs nothing else (package mygo links no HTTP of its own).
import _ "github.com/egoist/mygo/updatefeed"
