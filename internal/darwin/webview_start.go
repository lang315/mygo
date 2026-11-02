//go:build darwin && !mygo_nowebview

package darwin

import (
	"errors"
	"os"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/egoist/mygo/internal/platform"
)

// The parts of the backend that make web views: WKWebView, its class, the
// scheme handler and the delegate methods WebKit calls. webview_none.go
// stands in for this file in a build with the tag mygo_nowebview (`mygo build`
// of a project with nativeOnly), which links none of it.

// webViewProtocols are the protocols of WebKit that the window delegate
// answers, besides NSWindowDelegate's.
var webViewProtocols = []string{"WKNavigationDelegate", "WKUIDelegate", "WKScriptMessageHandler", "WKDownloadDelegate"}

// registerWebViewClasses registers the classes of web views, once, with the
// others (registerClasses): the web view and the handler of the app's schemes.
func registerWebViewClasses() {
	registerSchemeHandler()
	classDef("MyGoWebView", "WKWebView", nil, append(append(webViewMethods(), contextMenuMethods()...), []objc.MethodDef{
		method("mouseDown:", func(self id, cmd objc.SEL, ev id) {
			if w := theBackend.byWebView[self]; w != nil {
				if w.host != nil {
					w.pressed(ev, 0)
				}
				// A web view's page drags the window it is in (StartDrag).
				w.top().pressedLeft(ev)
			}
			sendSuper(self, "MyGoWebView", cmd, uintptr(ev))
		}),
		// Files dropped on the page: the page gets File objects, the app
		// their paths.
		method("performDragOperation:", func(self id, cmd objc.SEL, info id) bool {
			if w := theBackend.byWebView[self]; w != nil {
				w.dropped = draggedFiles(info)
			}
			return byte(sendSuper(self, "MyGoWebView", cmd, uintptr(info))) != 0
		}),
	}...))
}

// webViewDelegateMethods are the methods of the window delegate that WebKit
// calls.
func webViewDelegateMethods() []objc.MethodDef {
	b := func() *Backend { return theBackend }
	return []objc.MethodDef{
		method("webView:requestMediaCapturePermissionForOrigin:initiatedByFrame:type:decisionHandler:", func(self id, _ objc.SEL, web, origin, frame id, typ int, blk objc.Block) {
			handler := uintptr(blk)
			const grant, deny = 1, 2 // WKPermissionDecision
			decision := deny
			if w := b().windowFor(self); w != nil {
				kinds := map[int][]string{0: {"camera"}, 1: {"microphone"}, 2: {"camera", "microphone"}}[typ]
				if w.h.PermissionRequested(kinds, securityOrigin(origin)) {
					decision = grant
				}
			}
			callBlock(handler, uintptr(decision))
		}),
		method("userContentController:didReceiveScriptMessage:", func(self id, _ objc.SEL, ucc, msg id) {
			w := b().windowFor(self)
			if w == nil {
				return
			}
			// The handler is reachable from every frame; only the
			// page's own bridge may talk to Go.
			if frame := send(msg, "frameInfo"); frame == 0 || !sendBool(frame, "isMainFrame") {
				return
			}
			body := send(msg, "body")
			if body == 0 || !sendBool(body, "isKindOfClass:", uintptr(class("NSString"))) {
				return
			}
			if goString(send(msg, "name")) == pipHandler {
				w.pipActive = goString(body) == "true"
				return
			}
			w.h.Message(goString(body))
		}),
		method("observeValueForKeyPath:ofObject:change:context:", func(self id, _ objc.SEL, keyPath, object, change id, ctx unsafe.Pointer) {
			w := b().windowFor(self)
			if w == nil {
				return
			}
			switch goString(keyPath) {
			case "title":
				w.h.TitleChanged(goString(send(w.web, "title")))
			case "_isPlayingAudio":
				w.h.AudibleChanged(w.IsAudible())
			case "cameraCaptureState":
				w.captureChanged(platform.CaptureCamera)
			case "microphoneCaptureState":
				w.captureChanged(platform.CaptureMicrophone)
			case "_displayCaptureState":
				w.captureChanged(platform.CaptureDisplay)
			case "effectiveAppearance":
				// AppKit lays the title bar out again after telling us.
				b().post(w.layoutTrafficLights)
			}
		}),
		method("webView:decidePolicyForNavigationAction:decisionHandler:", func(self id, _ objc.SEL, web, action id, blk objc.Block) {
			handler := uintptr(blk)
			// A link with the download attribute (macOS 11.3).
			if respondsTo(action, "shouldPerformDownload") && sendBool(action, "shouldPerformDownload") {
				callBlock(handler, 2) // WKNavigationActionPolicyDownload
				return
			}
			allow := true
			if w := b().windowFor(self); w != nil {
				frame := send(action, "targetFrame")
				navType := sendInt(action, "navigationType")
				// WKNavigationType: link, form, back/forward, reload, …
				nav := platform.Navigation{
					URL:           goString(send(send(send(action, "request"), "URL"), "absoluteString")),
					IsMainFrame:   frame != 0 && sendBool(frame, "isMainFrame"),
					UserInitiated: userInitiated(action, navType),
					Unasked:       navType == 2 || navType == 3,
					Reload:        navType == 3,
				}
				if nav.IsMainFrame && w.programmatic {
					w.programmatic = false
					nav.Unasked = true
				}
				allow = w.h.WillNavigate(nav)
				if allow && nav.IsMainFrame {
					w.leavingPage()
				}
			}
			callBlock(handler, boolArg(allow))
		}),
		method("webView:decidePolicyForNavigationResponse:decisionHandler:", func(self id, _ objc.SEL, web, nav id, blk objc.Block) {
			handler := uintptr(blk)
			const cancel, allow, download = 0, 1, 2 // WKNavigationResponsePolicy
			resp := send(nav, "response")
			url := goString(send(send(resp, "URL"), "absoluteString"))
			disposition, custom := schemeDispositions[url]
			if !custom && respondsTo(resp, "allHeaderFields") {
				disposition = goString(send(send(resp, "allHeaderFields"), "objectForKey:", uintptr(nsString("Content-Disposition"))))
			}
			attachment := strings.HasPrefix(strings.ToLower(strings.TrimSpace(disposition)), "attachment")
			switch {
			case !attachment && sendBool(nav, "canShowMIMEType"):
				callBlock(handler, allow)
			case custom:
				// WebKit cannot turn what a scheme handler serves into a
				// download: serve it again into one.
				callBlock(handler, cancel)
				if w := b().windowFor(self); w != nil {
					w.h.SchemeDownload(url)
				}
			case hasClass("WKDownload"):
				callBlock(handler, download)
			default:
				callBlock(handler, cancel)
			}
		}),
		method("webView:navigationAction:didBecomeDownload:", func(self id, _ objc.SEL, web, action, download id) {
			send(download, "setDelegate:", uintptr(self))
		}),
		method("webView:navigationResponse:didBecomeDownload:", func(self id, _ objc.SEL, web, nav, download id) {
			send(download, "setDelegate:", uintptr(self))
		}),
		method("download:decideDestinationUsingResponse:suggestedFilename:completionHandler:", func(self id, _ objc.SEL, download, resp, suggested id, blk objc.Block) {
			handler := uintptr(blk)
			w := b().windowFor(self)
			if w == nil {
				callBlock(handler, 0)
				return
			}
			url := goString(send(send(send(download, "originalRequest"), "URL"), "absoluteString"))
			path := w.h.DownloadStarted(url, goString(suggested))
			if path == "" {
				callBlock(handler, 0) // cancels the download
				return
			}
			os.Remove(path) // WebKit refuses to replace a file
			w.downloads[download] = [2]string{url, path}
			withPool(func() { callBlock(handler, uintptr(send(class("NSURL"), "fileURLWithPath:", uintptr(nsString(path))))) })
		}),
		method("downloadDidFinish:", func(self id, _ objc.SEL, download id) {
			if w := b().windowFor(self); w != nil {
				if d, ok := w.downloads[download]; ok {
					delete(w.downloads, download)
					w.h.DownloadFinished(d[0], d[1], nil)
				}
			}
		}),
		method("download:didFailWithError:resumeData:", func(self id, _ objc.SEL, download, nserr, resume id) {
			if w := b().windowFor(self); w != nil {
				if d, ok := w.downloads[download]; ok {
					delete(w.downloads, download)
					w.h.DownloadFinished(d[0], d[1], errors.New(goString(send(nserr, "localizedDescription"))))
				}
			}
		}),
		method("webView:didStartProvisionalNavigation:", func(self id, _ objc.SEL, web, nav id) {
			if w := b().windowFor(self); w != nil {
				w.provisionalURL = w.URL()
				w.h.NavigationStarted(w.provisionalURL)
			}
		}),
		method("webView:didCommitNavigation:", func(self id, _ objc.SEL, web, nav id) {
			if w := b().windowFor(self); w != nil {
				w.h.NavigationCommitted(w.URL())
			}
		}),
		method("webView:didFinishNavigation:", func(self id, _ objc.SEL, web, nav id) {
			if w := b().windowFor(self); w != nil {
				w.h.LoadFinished()
			}
		}),
		method("webView:didFailProvisionalNavigation:withError:", func(self id, _ objc.SEL, web, nav, err id) {
			if w := b().windowFor(self); w != nil {
				w.loadFailed(err, w.provisionalURL)
			}
		}),
		method("webView:didFailNavigation:withError:", func(self id, _ objc.SEL, web, nav, err id) {
			if w := b().windowFor(self); w != nil {
				w.loadFailed(err, w.URL())
			}
		}),
		method("webView:didReceiveAuthenticationChallenge:completionHandler:", func(self id, _ objc.SEL, web, challenge id, handler objc.Block) {
			if w := b().windowFor(self); w != nil {
				w.authChallenge(challenge, uintptr(handler))
			} else {
				callBlock(uintptr(handler), authDefault, 0)
			}
		}),
		method("webViewWebContentProcessDidTerminate:", func(self id, _ objc.SEL, web id) {
			if w := b().windowFor(self); w != nil {
				w.h.RenderProcessGone("terminated")
			}
		}),
		method("webView:createWebViewWithConfiguration:forNavigationAction:windowFeatures:", func(self id, _ objc.SEL, web, cfg, action, features id) id {
			w := b().windowFor(self)
			if w == nil {
				return 0
			}
			num := func(key string) int {
				if n := send(features, key); n != 0 {
					return sendInt(n, "integerValue")
				}
				return 0
			}
			req := platform.NewWindowRequest{
				URL:    goString(send(send(send(action, "request"), "URL"), "absoluteString")),
				X:      num("x"),
				Y:      num("y"),
				Width:  num("width"),
				Height: num("height"),
				// The action of the link, or of window.open(), whose
				// WKNavigationType is "other".
				UserInitiated: userInitiated(action, sendInt(action, "navigationType")),
				Native:        uintptr(cfg),
			}
			child, ok := w.h.NewWindow(req).(*window)
			if !ok || child == nil {
				return 0
			}
			return child.web
		}),
		method("webViewDidClose:", func(self id, _ objc.SEL, web id) {
			if w := b().windowFor(self); w != nil {
				w.h.ClosedByPage()
			}
		}),
		method("webView:runJavaScriptAlertPanelWithMessage:initiatedByFrame:completionHandler:", func(self id, _ objc.SEL, web, message, frame id, blk objc.Block) {
			handler := uintptr(blk)
			w := b().windowFor(self)
			jsAlert(w, goString(message), handler)
		}),
		method("webView:runJavaScriptConfirmPanelWithMessage:initiatedByFrame:completionHandler:", func(self id, _ objc.SEL, web, message, frame id, blk objc.Block) {
			handler := uintptr(blk)
			w := b().windowFor(self)
			jsConfirm(w, goString(message), handler)
		}),
		method("webView:runJavaScriptTextInputPanelWithPrompt:defaultText:initiatedByFrame:completionHandler:", func(self id, _ objc.SEL, web, prompt, defaultText, frame id, blk objc.Block) {
			handler := uintptr(blk)
			w := b().windowFor(self)
			jsPrompt(w, goString(prompt), goString(defaultText), handler)
		}),
		method("webView:runOpenPanelWithParameters:initiatedByFrame:completionHandler:", func(self id, _ objc.SEL, web, params, frame id, blk objc.Block) {
			handler := uintptr(blk)
			w := b().windowFor(self)
			jsFileInput(w, params, handler)
		}),
	}
}

func (w *window) createWebView(content NSRect) {
	o := w.opts
	var cfg id
	if o.Native != 0 {
		// window.open(): WebKit hands us the configuration to use.
		cfg = retain(id(o.Native))
	} else {
		cfg = alloc("WKWebViewConfiguration")
		for _, s := range o.Schemes {
			send(cfg, "setURLSchemeHandler:forURLScheme:", uintptr(schemeHandler()), uintptr(nsString(s)))
		}
		if o.DataStore != nil { // a session's (session.go)
			send(cfg, "setWebsiteDataStore:", uintptr(websiteDataStore(o.DataStore)))
		}
	}
	prefs := send(cfg, "preferences")
	send(prefs, "setValue:forKey:", uintptr(nsBool(o.DevTools)), uintptr(nsString("developerExtrasEnabled")))
	send(prefs, "setValue:forKey:", uintptr(nsBool(true)), uintptr(nsString("allowFileAccessFromFileURLs")))
	if respondsTo(prefs, "setElementFullscreenEnabled:") {
		send(prefs, "setElementFullscreenEnabled:", 1)
	}
	// Tab reaches buttons, links and the other controls too, as it does in
	// Chrome, Electron and WebView2, whatever the system's keyboard
	// navigation setting, unless the app asked for Safari's behavior.
	if respondsTo(prefs, "setTabFocusesLinks:") {
		send(prefs, "setTabFocusesLinks:", boolArg(o.TabFocusesLinks))
	}
	if o.NoBackgroundThrottling || o.Native != 0 {
		setBackgroundThrottling(cfg, prefs, !o.NoBackgroundThrottling, o.Native != 0)
		// It may have given a window.open window a copy of its opener's.
		prefs = send(cfg, "preferences")
	}
	// Picture in picture, which WKWebView keeps to Safari: the button of
	// videos' controls, their context menu's item and
	// requestPictureInPicture().
	if respondsTo(prefs, "_setAllowsPictureInPictureMediaPlayback:") {
		send(prefs, "_setAllowsPictureInPictureMediaPlayback:", boolArg(o.PictureInPicture))
	}
	setAutoplay(cfg, o.Autoplay, o.Native != 0)

	// Every window gets its own content controller so messages, scripts and
	// the bridge configuration never leak between windows.
	w.ucc = alloc("WKUserContentController")
	send(w.ucc, "addScriptMessageHandler:name:", uintptr(w.delegate), uintptr(nsString("mygo")))
	if o.PictureInPicture {
		send(w.ucc, "addScriptMessageHandler:name:", uintptr(w.delegate), uintptr(nsString(pipHandler)))
	}
	w.addUserScripts(o.Zoom)
	send(cfg, "setUserContentController:", uintptr(w.ucc))

	// WebKit appends the application name to the user agent of the web view,
	// and window.open() hands us the configuration of the window that opens
	// it: the options of the new window decide, as they do on Linux and
	// Windows, where a window never inherits a suffix.
	var appName id
	if o.UserAgentSuffix != "" {
		appName = nsString(o.UserAgentSuffix)
	}
	if o.UserAgentSuffix != "" || o.Native != 0 {
		send(cfg, "setApplicationNameForUserAgent:", uintptr(appName))
	}

	w.web = msgInitRectID(send(class("MyGoWebView"), "alloc"), sel("initWithFrame:configuration:"), NSRect{Size: content.Size}, cfg)
	release(cfg)
	send(w.web, "setNavigationDelegate:", uintptr(w.delegate))
	send(w.web, "setUIDelegate:", uintptr(w.delegate))
	send(w.web, "setAutoresizingMask:", nsViewWidthHeightSizable)
	if o.Transparent || o.BackgroundColor != nil || o.Vibrancy != "" {
		// Let the window color show until the page paints.
		send(w.web, "setValue:forKey:", uintptr(nsBool(false)), uintptr(nsString("drawsBackground")))
		if o.BackgroundColor != nil && respondsTo(w.web, "setUnderPageBackgroundColor:") {
			send(w.web, "setUnderPageBackgroundColor:", uintptr(nsColor(*o.BackgroundColor)))
		}
	}
	if respondsTo(w.web, "setInspectable:") {
		send(w.web, "setInspectable:", boolArg(o.DevTools))
	}
	if o.NoBackgroundThrottling && respondsTo(w.web, "_setWindowOcclusionDetectionEnabled:") {
		// A window behind others counts as visible.
		send(w.web, "_setWindowOcclusionDetectionEnabled:", 0)
	}
	// Both are off in a new WKWebView.
	if o.SwipeNavigation {
		send(w.web, "setAllowsBackForwardNavigationGestures:", 1)
	}
	if o.PinchZoom {
		send(w.web, "setAllowsMagnification:", 1)
	}
	if o.UserAgent != "" {
		ua := o.UserAgent
		if o.UserAgentSuffix != "" {
			ua += " " + o.UserAgentSuffix
		}
		send(w.web, "setCustomUserAgent:", uintptr(nsString(ua)))
	}
	if o.Zoom > 0 && o.Zoom != 1 {
		msgSetFloat(w.web, sel("setPageZoom:"), o.Zoom)
	}
	send(w.web, "addObserver:forKeyPath:options:context:", uintptr(w.delegate), uintptr(nsString("title")), 1, 0)
	if respondsTo(w.web, "_isPlayingAudio") {
		// WebKit tells its changes with willChangeValueForKey: and
		// didChangeValueForKey:, as Safari's tabs' speaker icons hear.
		w.observesAudio = true
		send(w.web, "addObserver:forKeyPath:options:context:", uintptr(w.delegate), uintptr(nsString("_isPlayingAudio")), 0, 0)
	}
	// WebKit tells their changes as for _isPlayingAudio.
	for kind, key := range captureKeys {
		if respondsTo(w.web, key) {
			w.observesCapture[kind] = true
			send(w.web, "addObserver:forKeyPath:options:context:", uintptr(w.delegate), uintptr(nsString(key)), 0, 0)
		}
	}
}
