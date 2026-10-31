//go:build js

package mygo

// Browsers load the app's newest version with the page: they neither update
// nor relaunch it.

func relaunchNow() {}

func cleanUpdateLeftovers() {}
