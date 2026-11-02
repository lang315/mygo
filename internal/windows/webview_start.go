//go:build windows && (amd64 || arm64) && !mygo_nowebview

package windows

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/webview2"
)

// startEnvironment starts creating the WebView2 environment, the first time
// a window that shows a web page is created, and returns why it cannot,
// such as a missing WebView2 Runtime. Windows that show native UI need
// none: an app whose windows all do runs without the runtime. o are the
// options of the page that needs it, nil for none.
//
// The environment's browser runs with unthrottledArguments when that first
// page asked not to be throttled, and with autoplayArguments when it asked
// to play without a user gesture: the arguments are the browser's, which
// all the app's pages share, and a page that asks once it runs is too
// late, which the log tells.
func (b *Backend) startEnvironment(o *platform.WindowOptions) error {
	unthrottled := o != nil && o.NoBackgroundThrottling
	autoplay := o != nil && o.Autoplay
	if b.envStarted {
		if unthrottled && !b.envUnthrottled && !b.envWarned {
			b.envWarned = true
			log.Print("mygo: ThrottlingOff of a page shown after the first one: WebView2's browser already runs without the arguments that keep hidden pages running; give the option to the first window or web view that shows a page")
		}
		if autoplay && !b.envAutoplay && !b.envAutoplayWarned {
			b.envAutoplayWarned = true
			log.Print("mygo: AutoplayAllow of a page shown after the first one: WebView2's browser already runs without the argument that lets pages play sound without a user gesture; give the option to the first window or web view that shows a page")
		}
		return nil
	}
	dir := os.Getenv("LOCALAPPDATA")
	if dir == "" {
		dir = os.TempDir()
	}
	userData := filepath.Join(dir, sanitize(b.name), "WebView2")
	_ = os.MkdirAll(userData, 0o755)
	language := ""
	if len(b.languages) > 0 {
		language = b.languages[0]
	}
	// What the first page asked for counts from now, so that a page shown
	// while the environment is being made isn't told it is too late.
	b.envUnthrottled, b.envAutoplay = unthrottled, autoplay
	err := createEnvironment(userData, language, browserArguments(unthrottled, autoplay), func(env uintptr, withArgs bool, err error) {
		b.env, b.envErr = env, err
		if err != nil || !withArgs {
			b.envUnthrottled, b.envAutoplay = false, false
		}
		if err != nil {
			log.Print(err)
			b.envResult.Store(2)
		} else {
			b.envResult.Store(1)
		}
		waiters := b.envWaiters
		b.envWaiters = nil
		for _, fn := range waiters {
			fn()
		}
	})
	// A runtime installed later is found by the next window.
	b.envStarted = err == nil
	return err
}

func (w *window) createWebView() {
	if w.closed {
		return
	}
	if w.b.envErr != nil {
		w.webViewFailed(w.b.envErr)
		return
	}
	if w.opts.DataStore != nil && !w.b.hasProfiles() {
		w.webViewFailed(errNoProfiles)
		return
	}
	fail := w.failsWebView()
	hr := withHandler(func(hr, controller uintptr) {
		// A web view in a window of native UI gets a composition
		// controller, whose controller closes it.
		closeController := func() {
			if c := queryInterface(controller, &iidICoreWebView2Controller); c != 0 {
				comCall(c, ctlClose)
				release(c)
			}
		}
		if fail && controller != 0 { // TestFailWebViews
			closeController()
			hr, controller = eFail, 0
		}
		// Closing the window aborts the creation (E_ABORT).
		if w.closed || w.destroying || w.host != nil && w.host.closed {
			if controller != 0 {
				closeController()
			}
			return
		}
		if failed(hr) || controller == 0 {
			w.creationFailed(hr)
			return
		}
		addRef(controller)
		if w.host != nil {
			controller = w.setUpEmbedded(controller)
		}
		w.setUp(controller)
	}, func(h uintptr) uintptr {
		if w.host != nil {
			return w.createEmbedded(h)
		}
		return w.b.createController(storeOf(w.opts.DataStore), w.hwnd, false, h)
	})
	if failed(hr) {
		w.creationFailed(hr)
	}
}

// runtimeBox shows the message about a missing WebView2 Runtime; tests
// replace it.
var runtimeBox = func(b *Backend, o *platform.MessageBoxOptions) platform.MessageBoxResult {
	return b.messageBox(0, o)
}

// openDownload opens the page of the runtime's download; tests replace it.
var openDownload = func() error { return shell{}.OpenExternal(webview2.Download) }

// explainMissingRuntime tells the user, once, that a window showing a web
// page needs a WebView2 Runtime the machine lacks or has broken, and offers
// the page to get it from: a window that cannot be made panics (NewWindow of
// package mygo), which an app without a console (-H=windowsgui), as
// `mygo build` makes them, would do in silence. err is why the environment
// could not start; it says nothing when the runtime is there. Main thread
// only, as the backend's methods.
func (b *Backend) explainMissingRuntime(err error) {
	if b.envExplained || err == nil || loaderProc() != nil {
		return
	}
	_, _, why := webview2.Find()
	if why == nil {
		return // the runtime is there: the failure is another's
	}
	b.envExplained = true
	name := b.name
	if name == "" {
		name = "This app"
	}
	r := runtimeBox(b, &platform.MessageBoxOptions{
		Type:    "error",
		Title:   name,
		Message: name + " needs the Microsoft Edge WebView2 Runtime to show its windows, and this computer has none that works.",
		Detail:  why.Error() + "\n\nInstall the runtime from " + webview2.Download + " and start the app again.",
		Buttons: []string{"Open the download page", "Close"}, DefaultID: 0, CancelID: 1,
	})
	if r.Response == 0 {
		if oerr := openDownload(); oerr != nil {
			log.Print(oerr)
		}
	}
}
