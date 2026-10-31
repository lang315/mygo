//go:build !js

// Package updates installs new versions of the app, which `mygo build`
// publishes when mygo.config.ts configures updates: signed archives and a
// manifest per platform. It is not part of package mygo, so that an app
// that does not update itself does not link the cryptography and the
// archives, and Go's HTTP when the feed is fetched (package updatefeed).
//
//	update, err := updates.Check(ctx)
//	if err == nil && update != nil {
//		if err := update.Install(ctx, nil); err == nil {
//			mygo.App.Relaunch()
//		}
//	}
//
// Builds that cannot update themselves, such as development builds, apps
// without updates in mygo.config.ts and apps installed by a package manager,
// say so (Enabled, ErrDisabled).
package updates

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fsretry"
	"github.com/egoist/mygo/internal/update"
	"github.com/egoist/mygo/internal/updatehook"
)

// Set by `mygo build` with -ldflags -X when mygo.config.ts configures updates:
// the manifest of this build's target, or with a tag prefix on GitHub its
// feed in the newest tagged release (update.TaggedFeed), and the public key
// updates must be signed with.
var (
	buildFeed string
	buildKey  string
)

// startExe is the executable the app started from. On Linux, os.Executable
// reports the old file once an update replaced it.
var startExe, _ = os.Executable()

func init() {
	// package mygo relaunches the app into an update staged on macOS.
	f := relaunchUpdated
	updatehook.RelaunchStaged.Store(&f)
}

// prereleases is set while Check offers prereleases (SetPrereleases).
var prereleases atomic.Bool

// updateSource is the source of SetSource, nil for the feed of
// mygo.config.ts.
var updateSource atomic.Pointer[Source]

// Source is where updates come from instead of the feed that
// mygo.config.ts configures (SetSource): an update service the app already
// has, an endpoint that needs the user's credentials, or a channel the app
// chooses while it runs. It serves what `mygo build` publishes, from
// wherever the app keeps it, and package updates checks it as it checks the feed's:
// the archive and deltas must be signed with the key of updates.publicKey
// that the app was built with, or they are not installed.
type Source interface {
	// Manifest returns the manifest of the running platform: the JSON that
	// `mygo build` writes as update-<target>.json, such as
	// update-darwin-arm64.json, of at most 1 MiB.
	// prereleases reports whether the app offers them (SetPrereleases).
	Manifest(ctx context.Context, prereleases bool) (io.ReadCloser, error)
	// Open returns the file at url, which the manifest names: the archive
	// of the update or one of its deltas. The URL is the manifest's, so it
	// may be anything the source understands, such as an id of its own.
	Open(ctx context.Context, url string) (io.ReadCloser, error)
}

// SetSource makes Check read the manifest from s, and Install download the
// files it names through s, instead of from the feed of mygo.config.ts; nil
// restores the feed. Updates are still enabled only in builds with updates
// in mygo.config.ts (Enabled), which give the key that updates are checked
// with. An Update that Check returned keeps downloading from the source it
// came from. It may be called from any goroutine.
func SetSource(s Source) {
	if s == nil {
		updateSource.Store(nil)
		return
	}
	updateSource.Store(&s)
}

// ErrDisabled is returned by Check for builds without updates:
// development builds, and builds of apps without updates in mygo.config.ts.
var ErrDisabled = errors.New("mygo: updates are not enabled for this build")

// Update is a newer version of the app, found by Check.
type Update struct {
	// Version of the update, and its release notes in Markdown.
	Version string
	Notes   string
	// Date is when the update was built.
	Date time.Time

	manifest update.Manifest
	key      string
	// source is where the manifest came from, which serves its files.
	source Source
	// current is the version that Check compared with, which delta updates
	// update.
	current string
}

// Enabled reports whether the app can update itself: it was built with
// updates and can write where it is installed. Apps installed by a package
// manager, such as a .deb in /opt, are updated by it instead.
func Enabled() bool {
	if buildFeed == "" || buildKey == "" || mygo.IsDev() {
		return false
	}
	target, err := installTarget()
	if err != nil {
		return false
	}
	probe, err := os.MkdirTemp(filepath.Dir(target), ".mygo-update-")
	if err != nil {
		return false
	}
	os.Remove(probe)
	return true
}

// Check asks the update feed, or the source of SetSource, for a version
// newer than the running one and returns it, or nil when the app is up to
// date.
func Check(ctx context.Context) (*Update, error) {
	if !Enabled() {
		return nil, ErrDisabled
	}
	src, err := defaultSource()
	if err != nil {
		return nil, err
	}
	return checkUpdate(ctx, src, buildKey, mygo.App.Version(), prereleases.Load())
}

// errNoUpdateFeed is what Check says when the app has a feed and nothing to
// fetch it with: the HTTP feed is package updatefeed, which package mygo does
// not import, so that apps that do not update over HTTP do not link net/http.
var errNoUpdateFeed = errors.New(`mygo: the update feed of this build has no HTTP source linked: add import _ "github.com/egoist/mygo/updatefeed" to the app (mygo build adds it for apps with updates in mygo.config.ts), or give SetSource a source`)

// defaultSource returns where Check reads from: the source of SetSource, else
// the feed of mygo.config.ts, which package updatefeed fetches.
func defaultSource() (Source, error) {
	if s := updateSource.Load(); s != nil {
		return *s, nil
	}
	feed := updatehook.Feed.Load()
	if feed == nil {
		return nil, errNoUpdateFeed
	}
	return (*feed)(buildFeed), nil
}

// SetPrereleases makes Check offer prereleases too, such as 1.3.0-beta.1,
// for the users of an app who chose its beta: with updates on GitHub,
// Check then reads the release of the highest version that is not a
// draft, through the GitHub API (usually one request a check), so that a
// release newer than the prereleases is offered too. Check never offers a
// version older than the running one, so turning it off waits for the
// release after the prerelease the app runs. Updates of updates.url are
// the manifest the server has, whichever is set. It is off at launch; the
// app keeps the user's choice and sets it again. It may be called from
// any goroutine.
func SetPrereleases(on bool) { prereleases.Store(on) }

// Prereleases reports whether Check offers prereleases (SetPrereleases).
func Prereleases() bool { return prereleases.Load() }

func checkUpdate(ctx context.Context, src Source, key, current string, pre bool) (*Update, error) {
	body, err := src.Manifest(ctx, pre)
	if err != nil {
		return nil, fmt.Errorf("mygo: checking for updates: %w", err)
	}
	defer body.Close()
	var m update.Manifest
	if err := json.UnmarshalRead(io.LimitReader(body, update.MaxManifestSize), &m); err != nil {
		return nil, fmt.Errorf("mygo: checking for updates: %w", err)
	}
	if m.Version == "" || m.URL == "" || m.Signature == "" {
		return nil, errors.New("mygo: checking for updates: incomplete manifest")
	}
	if update.Compare(m.Version, current) <= 0 {
		return nil, nil
	}
	up := &Update{Version: m.Version, Notes: m.Notes, manifest: m, key: key, current: current, source: src}
	up.Date, _ = time.Parse(time.RFC3339, m.Date)
	return up, nil
}

// Install downloads the update, checks that it is signed with the app's
// key and replaces the app with it; progress, when not nil, is called with
// the bytes downloaded so far. When the update has a delta for the running
// version, only the delta is downloaded, unless it fails to make the new
// version: the whole update is downloaded then, and the progress starts
// over. The running app is not affected: the new version runs after
// App.Relaunch, or at the next launch.
//
// The app must be able to write where it is installed: a bundle in
// /Applications of an administrator, or an app directory the user owns on
// Linux and Windows. Apps installed by a package manager are updated by it.
func (up *Update) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	if up.source == nil {
		return errors.New("mygo: installing the update: the update did not come from Check")
	}
	target, err := installTarget()
	if err != nil {
		return err
	}
	if err := installUpdate(ctx, up.source.Open, up.manifest, up.key, up.current, target, progress); err != nil {
		return fmt.Errorf("mygo: installing the update: %w", err)
	}
	if runtime.GOOS == "linux" {
		// install.sh registered the desktop entry of the version it installed.
		if err := update.RefreshDesktopEntry(target, filepath.Base(startExe)); err != nil {
			log.Printf("mygo: registering the desktop entry of the update: %v", err)
		}
	}
	return nil
}

// installTarget returns what an update replaces: the app bundle on macOS,
// the directory of the executable elsewhere.
func installTarget() (string, error) {
	exe := startExe
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if runtime.GOOS == "darwin" {
		bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
		if !strings.HasSuffix(bundle, ".app") {
			return "", errors.New("mygo: only app bundles can update themselves")
		}
		return bundle, nil
	}
	if os.Getenv("APPIMAGE") != "" {
		return "", errors.New("mygo: AppImages cannot update themselves")
	}
	return filepath.Dir(exe), nil
}

// installUpdate installs the version of m in place of target, the app of
// version current: from its delta for current when there is one, else, or
// when the delta fails, from its archive, both downloaded with get.
func installUpdate(ctx context.Context, get update.Getter, m update.Manifest, key, current, target string, progress func(downloaded, total int64)) error {
	pub, err := update.ParsePublicKey(key)
	if err != nil {
		return err
	}
	// Next to the app, so that it can be renamed into place.
	work, err := os.MkdirTemp(filepath.Dir(target), "."+filepath.Base(target)+".update-")
	if err != nil {
		return fmt.Errorf("cannot write next to the app: %w", err)
	}
	staged := false
	defer func() {
		if !staged {
			os.RemoveAll(work)
		}
	}()
	unpacked := filepath.Join(work, "app")
	swap := swapFiles
	if runtime.GOOS == "darwin" {
		swap = func(dir, target string) error {
			err := stageBundle(work, dir, target)
			staged = err == nil
			return err
		}
	}

	if d := m.Delta(current); d != nil {
		err := applyDelta(ctx, get, *d, pub, m.Version, target, work, unpacked, progress)
		if err == nil {
			return swap(unpacked, target)
		}
		if ctx.Err() != nil {
			return err
		}
		log.Printf("mygo: downloading the whole update, as its delta update failed: %v", err)
		if err := os.RemoveAll(unpacked); err != nil {
			return err
		}
	}

	if m.Size <= 0 || m.Size > update.MaxArchiveSize {
		return fmt.Errorf("invalid size %d", m.Size)
	}
	archive := filepath.Join(work, "update.tar.gz")
	if err := download(ctx, get, m.URL, m.Size, m.Signature, pub, archive, progress); err != nil {
		return err
	}
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	err = update.ExtractArchive(f, unpacked)
	f.Close()
	if err != nil {
		return err
	}
	return swap(unpacked, target)
}

// applyDelta downloads the delta d into work and makes the new version of
// the app at target from it, in dir: its bundle on macOS, in dir too, as
// the bundle of an archive is.
func applyDelta(ctx context.Context, get update.Getter, d update.Delta, pub ed25519.PublicKey, version, target, work, dir string, progress func(downloaded, total int64)) error {
	if d.Size <= 0 || d.Size > update.MaxArchiveSize {
		return fmt.Errorf("invalid size %d", d.Size)
	}
	path := filepath.Join(work, "update.delta")
	if err := download(ctx, get, d.URL, d.Size, d.Signature, pub, path, progress); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if runtime.GOOS == "darwin" {
		if err := os.Mkdir(dir, 0o755); err != nil {
			return err
		}
		dir = filepath.Join(dir, filepath.Base(target))
	}
	return update.ApplyDelta(f, d.Size, d.From, version, target, dir)
}

// download fetches url with get into the file path and checks its size and
// signature.
func download(ctx context.Context, get update.Getter, url string, size int64, signature string, pub ed25519.PublicKey, path string, progress func(downloaded, total int64)) error {
	body, err := get(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	sum := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, sum), &progressReader{r: io.LimitReader(body, size+1), total: size, fn: progress})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	if n != size {
		return fmt.Errorf("downloaded %d bytes, want %d", n, size)
	}
	return update.Verify(pub, sum.Sum(nil), signature)
}

// stagedUpdate is a new version of the app bundle target, at bundle in
// work, a directory next to target that the update was made in, and the
// helper process that installs it once the app has exited.
type stagedUpdate struct {
	work, bundle, target string
	helper               *exec.Cmd
}

// pendingUpdate is the update that Install staged on macOS. It replaces the
// app's bundle once the app has exited, however it exits: replacing it
// while the app runs moved the bundle of the running app away, the Dock's
// tile of the app followed it, and the new version, opened where the app
// was, got a second tile.
var pendingUpdate atomic.Pointer[stagedUpdate]

// appPid is the process whose exit a staged update waits for: the app's,
// which tests replace.
var appPid = os.Getpid()

// stageBundle leaves the bundle that dir holds to replace target once the
// app has exited, in place of an update staged before.
func stageBundle(work, dir, target string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".app") {
		return errors.New("the update holds no app bundle")
	}
	p := &stagedUpdate{work: work, bundle: filepath.Join(dir, entries[0].Name()), target: target}
	args := []string{"-c", installAfterExit, "mygo-update", strconv.Itoa(appPid), p.target, p.bundle, p.work, p.target}
	if len(os.Args) > 1 {
		args = append(append(args, "--args"), os.Args[1:]...)
	}
	p.helper = exec.Command("/bin/sh", args...)
	if err := p.helper.Start(); err != nil {
		return err
	}
	if old := pendingUpdate.Swap(p); old != nil {
		_ = old.helper.Process.Kill()
		_ = old.helper.Wait()
		_ = os.RemoveAll(old.work)
	}
	return nil
}

// installAfterExit is the script of /bin/sh that installs a staged update:
// with the pid of the app, the target, bundle and work of the update and
// then the arguments of open, it waits for the app to exit, swaps the
// bundles, putting the old one back if the new one cannot be moved in, and
// opens the app again when Relaunch asked for it (relaunchUpdated). Through
// Launch Services, as openBundle, but without -n: the app has exited, and
// the Dock gives the new version the app's tile.
const installAfterExit = `pid=$1 target=$2 new=$3 work=$4
shift 4
trap '' HUP
while kill -0 "$pid" 2>/dev/null; do sleep 0.05; done
if mv "$target" "$work/old"; then
	mv "$new" "$target" || mv "$work/old" "$target"
fi
relaunch=
if [ -e "$work/relaunch" ]; then relaunch=1; fi
rm -rf "$work"
if [ -n "$relaunch" ]; then exec /usr/bin/open "$@"; fi
`

// relaunchUpdated asks the helper of a staged update to open the app once
// it has installed the update, reporting whether there is one.
func relaunchUpdated() bool {
	p := pendingUpdate.Load()
	return p != nil && os.WriteFile(filepath.Join(p.work, "relaunch"), nil, 0o644) == nil
}

// swapFiles replaces the entries of the app directory target with those in
// dir. The running executable is renamed away, which Windows allows, and
// removed at the next launch (cleanUpdateLeftovers).
func swapFiles(dir, target string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("the update is empty")
	}
	suffix := ".old-" + strconv.Itoa(os.Getpid())
	var moved [][2]string // renamed away: old name, new name
	undo := func() {
		for i := len(moved) - 1; i >= 0; i-- {
			_ = os.RemoveAll(moved[i][0])
			_ = fsretry.Rename(moved[i][1], moved[i][0])
		}
	}
	for _, e := range entries {
		dst := filepath.Join(target, e.Name())
		if _, err := os.Lstat(dst); err == nil {
			aside := filepath.Join(target, "."+e.Name()+suffix)
			if err := fsretry.Rename(dst, aside); err != nil {
				undo()
				return err
			}
			moved = append(moved, [2]string{dst, aside})
		}
		if err := fsretry.Rename(filepath.Join(dir, e.Name()), dst); err != nil {
			undo()
			return err
		}
	}
	for _, m := range moved {
		_ = os.RemoveAll(m[1])
	}
	return nil
}

// progressReader reports how much of a download was read.
type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	fn    func(downloaded, total int64)
	last  time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.fn != nil && (err != nil || p.n == p.total || time.Since(p.last) > 100*time.Millisecond) {
		p.last = time.Now()
		p.fn(p.n, p.total)
	}
	return n, err
}
