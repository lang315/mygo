/** What the bridge says to Go: the messages of one document. */
export interface Channel {
  /** JSON.stringify as it was when the bridge started. */
  stringify(value: unknown): string;
  /** Posts a message, with this document's secret in front of it. */
  post(message: string): void;
  /** Posts File objects along with a message (WebView2). */
  postWithFiles?(message: string, files: FileList): void;
  /** Calls fn with each message the host posts as a web message (WebView2). */
  listen?(fn: (data: unknown) => void): void;
}

/**
 * Connects the bridge to Go, and returns the channel its messages go through,
 * or null in a page without a handler.
 *
 * The document makes a secret and tells it to Go in a hello, which is the first
 * message it posts. The script runs at document start: before any script of the
 * page, so before the page can replace anything the bridge calls, and before
 * the page has an iframe, which could post a hello of its own. Go takes the
 * first hello after the main frame commits a navigation and ignores the rest,
 * and it ignores a message that does not start with the secret it took. The
 * handler is reachable from every frame, so the secret is all that tells the
 * bridge from a page or a frame that posts directly.
 *
 * Everything the bridge calls with the secret is taken here, once: the handler's
 * postMessage bound to the handler, JSON.stringify and crypto's
 * getRandomValues. A page that replaces them later, or looks one up again on
 * the handler, gets nothing; a bound function calls its target without looking
 * anything up.
 */
export function connect(w: any): Channel | null {
  const handler = w.webkit?.messageHandlers?.mygo;
  const webview = w.chrome?.webview;
  const addListener: ((type: string, fn: (e: MessageEvent) => void) => void) | undefined =
    webview?.addEventListener?.bind(webview);
  let rawPost: (message: string) => void;
  let rawFiles: ((message: string, files: FileList) => void) | undefined;
  if (handler) {
    rawPost = handler.postMessage.bind(handler);
  } else if (webview) {
    rawPost = webview.postMessage.bind(webview);
    rawFiles = webview.postMessageWithAdditionalObjects?.bind(webview);
  } else {
    return null;
  }
  const stringify: (value: unknown) => string = w.JSON.stringify;
  const random: (array: Uint8Array) => Uint8Array = w.crypto.getRandomValues.bind(w.crypto);

  const bytes = random(new Uint8Array(16));
  let secret = "";
  for (let i = 0; i < bytes.length; i++) secret += (bytes[i]! + 256).toString(16).slice(1);

  const hello = stringify({ t: "hello", k: secret });
  rawPost(hello);
  // WebKit restores a page from its back/forward cache on Back and Forward: the
  // main frame commits, so Go waits for a hello, and the document comes back as
  // it was, without running this script again. Say the secret it has once more.
  // A capturing listener, so it is ahead of any the page adds; an event a script
  // made (isTrusted false) is not the engine's. The page can stop neither this
  // nor learn from it: the hello goes to the handler captured above.
  w.addEventListener(
    "pageshow",
    (e: PageTransitionEvent) => {
      if (e.persisted && e.isTrusted) rawPost(hello);
    },
    true,
  );
  return {
    stringify,
    post: (message) => rawPost(secret + message),
    postWithFiles: rawFiles && ((message, files) => rawFiles(secret + message, files)),
    // Windows delivers the host's messages as web messages, not scripts it runs
    // (see Window.flush): listened to with the function bound here, ahead of
    // any a page adds. Only the engine's own events (isTrusted) count; the
    // host sends only what carries the secret of the current document.
    listen: addListener && ((fn) => addListener("message", (e: MessageEvent) => void (e.isTrusted && fn(e.data)))),
  };
}
