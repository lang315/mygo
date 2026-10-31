//go:build js

// Package updates installs new versions of the app; browsers load the newest
// version with the page, so there it disables updates.
package updates

import (
	"context"
	"errors"
	"io"
	"time"
)

// Browsers load the app's newest version with the page: they neither
// update nor relaunch it, and builds for them leave out the updater, with
// net/http and the signatures' cryptography.

// ErrDisabled is returned by Check for builds without updates, as
// those for browsers.
var ErrDisabled = errors.New("mygo: updates are not enabled for this build")

// Update is a newer version of the app, found by Check.
type Update struct {
	// Version of the update, and its release notes in Markdown.
	Version string
	Notes   string
	// Date is when the update was built.
	Date time.Time
}

// Enabled reports whether the app can update itself: not in browsers.
func Enabled() bool { return false }

// Check returns ErrDisabled.
func Check(ctx context.Context) (*Update, error) {
	return nil, ErrDisabled
}

// Source is where Updater gets updates from instead of the feed that
// mygo.config.ts configures. Browsers do not update apps.
type Source interface {
	// Manifest returns the manifest of the running platform.
	Manifest(ctx context.Context, prereleases bool) (io.ReadCloser, error)
	// Open returns the file at url, which the manifest names.
	Open(ctx context.Context, url string) (io.ReadCloser, error)
}

// SetSource does nothing: browsers do not update apps.
func SetSource(s Source) {}

// SetPrereleases does nothing: browsers do not update apps.
func SetPrereleases(on bool) {}

// Prereleases reports false.
func Prereleases() bool { return false }

// Install returns ErrDisabled.
func (up *Update) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	return ErrDisabled
}
