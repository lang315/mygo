//go:build !js

// Package updatefeed is the HTTP source of the updates that mygo.config.ts
// configures: the manifest and the files of an update, fetched over HTTPS
// through the system's proxy.
//
// Package mygo reads no feed itself, so that the apps that do not update
// themselves over HTTP do not link net/http, which costs an app 2.2 MB.
// Importing this package gives package updates its feed:
//
//	import _ "github.com/egoist/mygo/updatefeed"
//
// `mygo build` adds that import to the apps with updates in
// mygo.config.ts, and the update window of plugins/updater has it, so most
// apps never write it. An app built another way, with the -ldflags that
// name the feed, and an app that wraps the feed in its own source
// (updates.SetSource) need it themselves; updates.Check says so when it is
// missing.
package updatefeed

import (
	"github.com/egoist/mygo/internal/updatehook"
	"github.com/egoist/mygo/internal/updatehttp"
	"github.com/egoist/mygo/updates"
)

func init() {
	feed := func(feed string) updatehook.Source { return updatehttp.Feed(feed) }
	updatehook.Feed.Store(&feed)
}

// Source returns the source of the feed at url, the manifest of a platform
// or a feed that names it (GitHub's included), as mygo.config.ts configures
// it. updates.SetSource takes it, for an app that adds to what the feed
// does, such as a header or a channel:
//
//	updates.SetSource(channelSource{feed: updatefeed.Source(url)})
func Source(url string) updates.Source { return updatehttp.Feed(url) }
