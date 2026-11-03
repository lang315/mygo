// Updatesmoke is the app of TestUpdaterSmoke: it checks for an update, installs
// it and relaunches into it, and reports each step as a line of the file
// MYGO_SMOKE_OUT. The update files come from the local server
// MYGO_SMOKE_SERVER, which serves what `mygo build` wrote: the app is built
// with updates.url on an HTTPS host that does not exist, and the source
// reads the manifest's URLs from the server instead, as docs/updates.md
// ("Your own update service") describes.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/updatefeed"
	"github.com/egoist/mygo/updates"
)

// published is the directory the manifest names its files in
// (updates.url in mygo.json).
const published = "https://updates.example.test/app/"

type local struct{ base string }

func (s local) Manifest(ctx context.Context, pre bool) (io.ReadCloser, error) {
	return s.Open(ctx, published+"update-windows-amd64.json")
}

func (s local) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	return updatefeed.Source(s.base).Open(ctx, s.base+strings.TrimPrefix(url, published))
}

func report(format string, args ...any) {
	f, err := os.OpenFile(os.Getenv("MYGO_SMOKE_OUT"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", args...)
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Width: 320, Height: 200}).Page().LoadHTML("<p>updatesmoke</p>", "")
		go update()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func update() {
	report("started %s enabled=%v", mygo.App.Version(), updates.Enabled())
	updates.SetSource(local{os.Getenv("MYGO_SMOKE_SERVER")})
	ctx := context.Background()
	up, err := updates.Check(ctx)
	switch {
	case err != nil:
		report("error check: %v", err)
	case up == nil:
		report("uptodate")
	default:
		report("found %s", up.Version)
		if err := up.Install(ctx, nil); err != nil {
			report("error install: %v", err)
			break
		}
		report("installed")
		mygo.App.Relaunch()
		return
	}
	mygo.App.Quit()
}
