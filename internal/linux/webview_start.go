//go:build linux && (amd64 || arm64) && !mygo_nowebview

package linux

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/ebitengine/purego"

	"github.com/egoist/mygo/internal/platform"
)

// The parts of the backend that make web views: WebKitGTK, JavaScriptCore and
// libsoup, loaded the first time a window shows a page, the web view of a
// window, and the signal handlers WebKit calls. webview_none.go stands in for
// this file in a build with the tag mygo_nowebview (`mygo build` of a project
// with nativeOnly), which links none of it.

func loadWebKit() error {
	var err error
	if libWebKit, err = open("libwebkit2gtk-4.1.so.0", "libwebkit2gtk-4.0.so.37"); err != nil {
		return fmt.Errorf("%w (install WebKitGTK: libwebkit2gtk-4.1-0 on Debian/Ubuntu, webkit2gtk4.1 on Fedora)", err)
	}
	// 4.1 links libsoup 3 and 4.0 libsoup 2.4. Opening the loaded library
	// again returns the same handle, which tells the two apart.
	h41, _ := open("libwebkit2gtk-4.1.so.0")
	soup3 = h41 != 0 && h41 == libWebKit
	jsc, soup := "libjavascriptcoregtk-4.1.so.0", "libsoup-3.0.so.0"
	if !soup3 {
		jsc, soup = "libjavascriptcoregtk-4.0.so.18", "libsoup-2.4.so.1"
	}
	// SoupCookie pointers must never cross Soup ABI generations.
	if libJSC, err = open(jsc); err != nil {
		return err
	}
	if libSoup, err = open(soup); err != nil {
		return err
	}
	bindCookies()
	bindAuth()

	w := libWebKit
	mustBind(w, &webkitWebContextGetDefault, "webkit_web_context_get_default")
	mustBind(w, &webkitWebContextRegisterURIScheme, "webkit_web_context_register_uri_scheme")
	mustBind(w, &webkitWebContextGetSecurityManager, "webkit_web_context_get_security_manager")
	mustBind(w, &webkitWebContextGetWebsiteDataManager, "webkit_web_context_get_website_data_manager")
	bind(w, &webkitWebContextSetPreferredLanguages, "webkit_web_context_set_preferred_languages")
	mustBind(w, &webkitWebsiteDataManagerClear, "webkit_website_data_manager_clear")
	mustBind(w, &webkitWebsiteDataManagerClearFinish, "webkit_website_data_manager_clear_finish")
	mustBind(w, &webkitDownloadGetWebView, "webkit_download_get_web_view")
	mustBind(w, &webkitDownloadGetRequest, "webkit_download_get_request")
	mustBind(w, &webkitDownloadSetDestination, "webkit_download_set_destination")
	mustBind(w, &webkitDownloadSetAllowOverwrite, "webkit_download_set_allow_overwrite")
	mustBind(w, &webkitDownloadCancel, "webkit_download_cancel")
	mustBind(w, &webkitResponsePolicyDecisionGetResponse, "webkit_response_policy_decision_get_response")
	mustBind(w, &webkitResponsePolicyDecisionIsMIMETypeSupported, "webkit_response_policy_decision_is_mime_type_supported")
	mustBind(w, &webkitPolicyDecisionDownload, "webkit_policy_decision_download")
	mustBind(w, &webkitURIResponseGetURI, "webkit_uri_response_get_uri")
	mustBind(w, &webkitURIResponseGetHTTPHeaders, "webkit_uri_response_get_http_headers")
	mustBind(w, &webkitSecurityManagerRegisterSecure, "webkit_security_manager_register_uri_scheme_as_secure")
	mustBind(w, &webkitSecurityManagerRegisterCORS, "webkit_security_manager_register_uri_scheme_as_cors_enabled")
	mustBind(w, &webkitUserContentManagerNew, "webkit_user_content_manager_new")
	mustBind(w, &webkitUserContentManagerRegisterHandler, "webkit_user_content_manager_register_script_message_handler")
	mustBind(w, &webkitUserContentManagerUnregisterHandler, "webkit_user_content_manager_unregister_script_message_handler")
	mustBind(w, &webkitUserContentManagerAddScript, "webkit_user_content_manager_add_script")
	mustBind(w, &webkitUserContentManagerRemoveAllScripts, "webkit_user_content_manager_remove_all_scripts")
	mustBind(w, &webkitUserScriptNew, "webkit_user_script_new")
	mustBind(w, &webkitUserScriptUnref, "webkit_user_script_unref")
	mustBind(w, &webkitWebViewNewWithUserContentManager, "webkit_web_view_new_with_user_content_manager")
	mustBind(w, &webkitWebViewGetSettings, "webkit_web_view_get_settings")
	mustBind(w, &webkitSettingsSetEnableDeveloperExtras, "webkit_settings_set_enable_developer_extras")
	mustBind(w, &webkitSettingsSetUserAgent, "webkit_settings_set_user_agent")
	mustBind(w, &webkitSettingsGetUserAgent, "webkit_settings_get_user_agent")
	mustBind(w, &webkitSettingsSetAllowFileAccessFromFileURLs, "webkit_settings_set_allow_file_access_from_file_urls")
	mustBind(w, &webkitSettingsSetJavascriptCanAccessClipboard, "webkit_settings_set_javascript_can_access_clipboard")
	mustBind(w, &webkitWebViewLoadURI, "webkit_web_view_load_uri")
	mustBind(w, &webkitWebViewLoadHTML, "webkit_web_view_load_html")
	mustBind(w, &webkitWebViewReload, "webkit_web_view_reload")
	mustBind(w, &webkitWebViewReloadBypassCache, "webkit_web_view_reload_bypass_cache")
	mustBind(w, &webkitWebViewStopLoading, "webkit_web_view_stop_loading")
	mustBind(w, &webkitWebViewGoBack, "webkit_web_view_go_back")
	mustBind(w, &webkitWebViewGoForward, "webkit_web_view_go_forward")
	mustBind(w, &webkitWebViewCanGoBack, "webkit_web_view_can_go_back")
	mustBind(w, &webkitWebViewCanGoForward, "webkit_web_view_can_go_forward")
	mustBind(w, &webkitWebViewIsLoading, "webkit_web_view_is_loading")
	mustBind(w, &webkitWebViewGetURI, "webkit_web_view_get_uri")
	mustBind(w, &webkitWebViewGetTitle, "webkit_web_view_get_title")
	mustBind(w, &webkitWebViewSetZoomLevel, "webkit_web_view_set_zoom_level")
	mustBind(w, &webkitWebViewGetZoomLevel, "webkit_web_view_get_zoom_level")
	mustBind(w, &webkitWebViewSetBackgroundColor, "webkit_web_view_set_background_color")
	bind(w, &webkitWebViewIsPlayingAudio, "webkit_web_view_is_playing_audio")
	// To turn WebKit's zoom gesture off (stopPinchZoom).
	bind(libGObject, &gObjectGetData, "g_object_get_data")
	bind(libGTK, &gtkEventControllerSetPropagationPhase, "gtk_event_controller_set_propagation_phase")
	if bind(w, &webkitWebViewSetIsMuted, "webkit_web_view_set_is_muted") {
		mustBind(w, &webkitWebViewGetIsMuted, "webkit_web_view_get_is_muted")
	}
	// The capture states of the camera, the microphone and the display
	// (2.34+), in the order of platform.CaptureKind.
	for kind, device := range [...]string{"camera", "microphone", "display"} {
		if bind(w, &webkitWebViewGetCaptureState[kind], "webkit_web_view_get_"+device+"_capture_state") {
			mustBind(w, &webkitWebViewSetCaptureState[kind], "webkit_web_view_set_"+device+"_capture_state")
		}
	}
	bind(w, &webkitWebViewEvaluateJavascript, "webkit_web_view_evaluate_javascript")
	bind(w, &webkitWebViewRunJavascript, "webkit_web_view_run_javascript")
	bind(w, &webkitWebViewCallAsyncJavascriptFunction, "webkit_web_view_call_async_javascript_function")
	bind(w, &webkitWebViewCallAsyncJavascriptFunctionFinish, "webkit_web_view_call_async_javascript_function_finish")
	mustBind(w, &webkitWebViewGetInspector, "webkit_web_view_get_inspector")
	mustBind(w, &webkitWebInspectorShow, "webkit_web_inspector_show")
	mustBind(w, &webkitWebInspectorClose, "webkit_web_inspector_close")
	mustBind(w, &webkitWebInspectorGetWebView, "webkit_web_inspector_get_web_view")
	mustBind(w, &webkitWebViewGetSnapshot, "webkit_web_view_get_snapshot")
	mustBind(w, &webkitWebViewGetSnapshotFinish, "webkit_web_view_get_snapshot_finish")
	mustBind(w, &webkitWebViewExecuteEditingCommand, "webkit_web_view_execute_editing_command")
	mustBind(w, &webkitPrintOperationNew, "webkit_print_operation_new")
	mustBind(w, &webkitPrintOperationRunDialog, "webkit_print_operation_run_dialog")
	mustBind(w, &webkitPrintOperationSetPrintSettings, "webkit_print_operation_set_print_settings")
	mustBind(w, &webkitPrintOperationSetPageSetup, "webkit_print_operation_set_page_setup")
	mustBind(w, &webkitPrintOperationPrint, "webkit_print_operation_print")
	mustBind(w, &webkitSettingsGetPrintBackgrounds, "webkit_settings_get_print_backgrounds")
	mustBind(w, &webkitSettingsSetPrintBackgrounds, "webkit_settings_set_print_backgrounds")
	mustBind(w, &webkitUserMediaPermissionRequestGetType, "webkit_user_media_permission_request_get_type")
	mustBind(w, &webkitGeolocationPermissionRequestGetType, "webkit_geolocation_permission_request_get_type")
	mustBind(w, &webkitNotificationPermissionRequestGetType, "webkit_notification_permission_request_get_type")
	mustBind(w, &webkitUserMediaPermissionIsForAudioDevice, "webkit_user_media_permission_is_for_audio_device")
	mustBind(w, &webkitUserMediaPermissionIsForVideoDevice, "webkit_user_media_permission_is_for_video_device")
	mustBind(w, &webkitPermissionRequestAllow, "webkit_permission_request_allow")
	mustBind(w, &webkitPermissionRequestDeny, "webkit_permission_request_deny")
	mustBind(w, &webkitJavascriptResultGetJSValue, "webkit_javascript_result_get_js_value")
	mustBind(w, &webkitNavigationPolicyDecisionGetNavigationAction, "webkit_navigation_policy_decision_get_navigation_action")
	bind(w, &webkitNavigationActionGetTargetFrameInfo, "webkit_navigation_action_get_target_frame_info")
	bind(w, &webkitFrameInfoIsMainFrame, "webkit_frame_info_is_main_frame")
	mustBind(w, &webkitNavigationActionGetRequest, "webkit_navigation_action_get_request")
	mustBind(w, &webkitNavigationActionGetNavigationType, "webkit_navigation_action_get_navigation_type")
	mustBind(w, &webkitNavigationActionIsUserGesture, "webkit_navigation_action_is_user_gesture")
	mustBind(w, &webkitURIRequestGetURI, "webkit_uri_request_get_uri")
	mustBind(w, &webkitPolicyDecisionUse, "webkit_policy_decision_use")
	bind(w, &webkitPolicyDecisionUseWithPolicies, "webkit_policy_decision_use_with_policies")
	bind(w, &webkitWebsitePoliciesNewWithPolicies, "webkit_website_policies_new_with_policies")
	mustBind(w, &webkitPolicyDecisionIgnore, "webkit_policy_decision_ignore")
	mustBind(w, &webkitURISchemeRequestGetURI, "webkit_uri_scheme_request_get_uri")
	mustBind(w, &webkitURISchemeRequestGetWebView, "webkit_uri_scheme_request_get_web_view")
	mustBind(w, &webkitURISchemeRequestFinishError, "webkit_uri_scheme_request_finish_error")
	mustBind(w, &webkitURISchemeRequestFinish, "webkit_uri_scheme_request_finish")
	bind(w, &webkitURISchemeRequestGetHTTPMethod, "webkit_uri_scheme_request_get_http_method")
	bind(w, &webkitURISchemeRequestGetHTTPHeaders, "webkit_uri_scheme_request_get_http_headers")
	bind(w, &webkitURISchemeRequestGetHTTPBody, "webkit_uri_scheme_request_get_http_body")
	bind(w, &webkitURISchemeRequestFinishWithResponse, "webkit_uri_scheme_request_finish_with_response")
	bind(w, &webkitURISchemeResponseNew, "webkit_uri_scheme_response_new")
	bind(w, &webkitURISchemeResponseSetStatus, "webkit_uri_scheme_response_set_status")
	bind(w, &webkitURISchemeResponseSetContentType, "webkit_uri_scheme_response_set_content_type")
	bind(w, &webkitURISchemeResponseSetHTTPHeaders, "webkit_uri_scheme_response_set_http_headers")

	mustBind(libJSC, &jscValueToString, "jsc_value_to_string")
	mustBind(libSoup, &soupMessageHeadersNew, "soup_message_headers_new")
	mustBind(libSoup, &soupMessageHeadersGetOne, "soup_message_headers_get_one")
	mustBind(libSoup, &soupMessageHeadersAppend, "soup_message_headers_append")
	mustBind(libSoup, &soupMessageHeadersForeach, "soup_message_headers_foreach")
	return nil
}

// initWebCallbacks creates the callbacks of the web views, once, with the
// others (initWindowCallbacks).
func initWebCallbacks() {
	b := func() *Backend { return theBackend }
	cbScriptMessage = purego.NewCallback(func(ucm, result, data ptr) {
		if w := b().window(data); w != nil {
			w.h.Message(takeStr(jscValueToString(webkitJavascriptResultGetJSValue(result))))
		}
	})
	cbLoadChanged = purego.NewCallback(func(web ptr, event int32, data ptr) {
		w := b().window(data)
		if w == nil {
			return
		}
		switch event {
		case 0: // WEBKIT_LOAD_STARTED
			w.h.NavigationStarted(w.URL())
		case 2: // WEBKIT_LOAD_COMMITTED
			w.h.NavigationCommitted(w.URL())
		case 3: // WEBKIT_LOAD_FINISHED
			w.h.LoadFinished()
		}
	})
	cbLoadFailed = purego.NewCallback(func(web ptr, event int32, uri, gerr, data ptr) bool {
		w := b().window(data)
		if w == nil || gerr == 0 {
			return false
		}
		code := field[int32](gerr, 4)
		// Cancelled loads (302) and navigations stopped by policy (102)
		// are not failures.
		if code == 302 || code == 102 {
			return false
		}
		w.h.LoadFailed(goStr(uri), int(code), goStr(field[ptr](gerr, 8)))
		return false
	})
	cbTitle = purego.NewCallback(func(web, pspec, data ptr) {
		if w := b().window(data); w != nil {
			w.h.TitleChanged(goStr(webkitWebViewGetTitle(w.web)))
		}
	})
	cbPlayingAudio = purego.NewCallback(func(web, pspec, data ptr) {
		if w := b().window(data); w != nil {
			w.h.AudibleChanged(webkitWebViewIsPlayingAudio(w.web))
		}
	})
	cbCaptureState = purego.NewCallback(func(web, pspec, data ptr) {
		w := b().window(data)
		if w == nil {
			return
		}
		kind := platform.CaptureKind(slices.Index(captureProperties[:], goStr(field[ptr](pspec, 8))))
		if kind < 0 {
			return
		}
		state := w.CaptureState(kind)
		if state == platform.CaptureNone {
			w.captureMuted[kind] = false
		}
		w.h.CaptureStateChanged(kind, state)
	})
	cbDecidePolicy = purego.NewCallback(func(web, decision ptr, kind int32, data ptr) bool {
		w := b().window(data)
		if w == nil {
			return false
		}
		if kind == 2 { // WEBKIT_POLICY_DECISION_TYPE_RESPONSE
			return w.decideResponse(decision)
		}
		if kind != 0 { // WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION
			return false
		}
		action := webkitNavigationPolicyDecisionGetNavigationAction(decision)
		navType := webkitNavigationActionGetNavigationType(action)
		uri := goStr(webkitURIRequestGetURI(webkitNavigationActionGetRequest(action)))
		// WebKitNavigationType: link, form, back/forward, reload, …
		nav := platform.Navigation{
			URL:           uri,
			IsMainFrame:   mainFrameNavigation(action, uri),
			UserInitiated: webkitNavigationActionIsUserGesture(action) || navType == 0 || navType == 1,
			Unasked:       navType == 2 || navType == 3,
			Reload:        navType == 3,
		}
		if nav.IsMainFrame && w.programmatic {
			w.programmatic = false
			nav.Unasked = true
		}
		if w.h.WillNavigate(nav) {
			if nav.IsMainFrame && w.opts.Autoplay && w.useWithAutoplay(decision) {
				return true
			}
			webkitPolicyDecisionUse(decision)
		} else {
			webkitPolicyDecisionIgnore(decision)
		}
		return true
	})
	cbCreate = purego.NewCallback(func(web, action, data ptr) ptr {
		w := b().window(data)
		if w == nil {
			return 0
		}
		// WebKitGTK makes a related view share the opener's user content
		// manager, which would route the new window's IPC to the opener.
		// Open an independent window instead; its page has no opener.
		url := goStr(webkitURIRequestGetURI(webkitNavigationActionGetRequest(action)))
		req := platform.NewWindowRequest{URL: url, UserInitiated: webkitNavigationActionIsUserGesture(action)}
		if child, ok := w.h.NewWindow(req).(*window); ok && child != nil {
			child.LoadURL(url)
		}
		return 0
	})
	cbClose = purego.NewCallback(func(web, data ptr) {
		if w := b().window(data); w != nil {
			w.h.ClosedByPage()
		}
	})
	cbCrashed = purego.NewCallback(func(web ptr, reason int32, data ptr) {
		if w := b().window(data); w != nil {
			w.h.RenderProcessGone(map[int32]string{0: "crashed", 1: "exceeded memory limit", 2: "terminated by API"}[reason])
		}
	})
	cbPermission = purego.NewCallback(func(web, req, data ptr) bool {
		w := b().window(data)
		if w == nil {
			return false
		}
		var kinds []string
		switch {
		case gTypeCheckInstanceIsA(req, webkitUserMediaPermissionRequestGetType()):
			if webkitUserMediaPermissionIsForVideoDevice(req) {
				kinds = append(kinds, "camera")
			}
			if webkitUserMediaPermissionIsForAudioDevice(req) {
				kinds = append(kinds, "microphone")
			}
		case gTypeCheckInstanceIsA(req, webkitGeolocationPermissionRequestGetType()):
			kinds = []string{"geolocation"}
		case gTypeCheckInstanceIsA(req, webkitNotificationPermissionRequestGetType()):
			kinds = []string{"notifications"}
		default:
			return false
		}
		origin := goStr(webkitWebViewGetURI(web))
		if u, err := url.Parse(origin); err == nil {
			origin = (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
		}
		if w.h.PermissionRequested(kinds, origin) {
			webkitPermissionRequestAllow(req)
		} else {
			webkitPermissionRequestDeny(req)
		}
		return true
	})
}

func (w *window) createWebView() {
	o := w.opts
	data := ptr(w.id)
	w.ucm = webkitUserContentManagerNew()
	if !w.b.webLanguages {
		// The first web view: WebKit just loaded.
		w.b.webLanguages = true
		w.b.applyWebLanguages()
	}
	w.web = w.newWebView()
	webkitUserContentManagerRegisterHandler(w.ucm, cs("mygo"))
	connect(w.ucm, "script-message-received::mygo", cbScriptMessage, data)
	w.addUserScripts()

	settings := webkitWebViewGetSettings(w.web)
	webkitSettingsSetEnableDeveloperExtras(settings, o.DevTools)
	webkitSettingsSetAllowFileAccessFromFileURLs(settings, true)
	// Back and Forward load the page again rather than restore it from the
	// back/forward cache. A restored document keeps its bridge, which does not
	// start again, and WebKitGTK does not say which frame posted a message: the
	// frames of a restored page run their pageshow before the main frame's, so
	// one of them could say a hello of its own before the bridge says its secret
	// again. A new load runs the bridge at document start, before any frame
	// exists (see "Trust" in the architecture guide). Pages lose the state the
	// cache kept (form fields, scroll of a page that unloads) on Back.
	gObjectSetBool(settings, cs("enable-page-cache"), false, 0)
	webkitSettingsSetJavascriptCanAccessClipboard(settings, true)
	if o.UserAgent != "" {
		webkitSettingsSetUserAgent(settings, cs(userAgentWithSuffix(o.UserAgent, o.UserAgentSuffix)))
	} else if o.UserAgentSuffix != "" {
		webkitSettingsSetUserAgentSuffix(settings, o.UserAgentSuffix)
	}
	if o.NoBackgroundThrottling {
		disableBackgroundThrottling(settings)
	}
	if o.Zoom > 0 && o.Zoom != 1 {
		webkitWebViewSetZoomLevel(w.web, o.Zoom)
	}
	if o.SwipeNavigation { // off by default
		gObjectSetBool(settings, cs("enable-back-forward-navigation-gestures"), true, 0)
	}
	if !o.PinchZoom {
		stopPinchZoom(w.web)
	}
	switch {
	case o.Transparent:
		webkitWebViewSetBackgroundColor(w.web, &gdkRGBA{})
	case o.BackgroundColor != nil:
		w.SetBackgroundColor(*o.BackgroundColor)
	}

	connect(w.web, "load-changed", cbLoadChanged, data)
	connect(w.web, "load-failed", cbLoadFailed, data)
	connect(w.web, "notify::title", cbTitle, data)
	if webkitWebViewIsPlayingAudio != nil {
		connect(w.web, "notify::is-playing-audio", cbPlayingAudio, data)
	}
	for kind, property := range captureProperties {
		if webkitWebViewGetCaptureState[kind] != nil {
			connect(w.web, "notify::"+property, cbCaptureState, data)
		}
	}
	connect(w.web, "decide-policy", cbDecidePolicy, data)
	connect(w.web, "create", cbCreate, data)
	connect(w.web, "close", cbClose, data)
	connect(w.web, "web-process-terminated", cbCrashed, data)
	connect(w.web, "button-press-event", cbButtonPress, data)
	if w.undecorated() {
		connect(w.web, "motion-notify-event", cbMotion, data)
	}
	connect(w.web, "drag-data-received", cbDragData, data)
	connect(w.web, "drag-drop", cbDragDrop, data)
	connect(w.web, "permission-request", cbPermission, data)
	connect(w.web, "context-menu", cbContextMenu, data)
	w.connectAuth(data)
	hookDownloads()
}
