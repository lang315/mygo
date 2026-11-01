import { describe, expect, test } from "bun:test";
import { connect } from "./transport";

// A window as the engines give one to a script at document start: the handler
// of WebKit, or the chrome.webview of WebView2, JSON and crypto.
function windowWith(kind: "webkit" | "webview2") {
  const posted: string[] = [];
  const files: [string, unknown][] = [];
  const target = {
    postMessage(m: string) {
      if (this !== target) throw new Error("postMessage called with another this");
      posted.push(m);
    },
    postMessageWithAdditionalObjects(m: string, f: unknown) {
      files.push([m, f]);
    },
  };
  const w: any =
    kind === "webkit"
      ? { webkit: { messageHandlers: { mygo: target } } }
      : { chrome: { webview: target } };
  w.JSON = { stringify: JSON.stringify };
  w.crypto = {
    getRandomValues<T extends ArrayBufferView>(a: T): T {
      new Uint8Array(a.buffer, a.byteOffset, a.byteLength).forEach((_, i, u) => (u[i] = (i * 17 + 3) & 0xff));
      return a;
    },
  };
  return { w, target, posted, files };
}

for (const kind of ["webkit", "webview2"] as const) {
  describe(`connect (${kind})`, () => {
    test("says the secret in a hello before anything else, and prefixes the rest", () => {
      const { w, posted } = windowWith(kind);
      const ch = connect(w)!;
      expect(posted).toHaveLength(1);
      const hello = JSON.parse(posted[0]!);
      expect(hello.t).toBe("hello");
      expect(hello.k).toMatch(/^[0-9a-f]{32}$/);
      ch.post(ch.stringify({ t: "dom-ready" }));
      expect(posted[1]).toBe(hello.k + '{"t":"dom-ready"}');
    });

    test("makes a secret for each document", () => {
      const first = windowWith(kind);
      first.w.crypto.getRandomValues = (a: Uint8Array) => (a.fill(1), a);
      const second = windowWith(kind);
      second.w.crypto.getRandomValues = (a: Uint8Array) => (a.fill(2), a);
      connect(first.w);
      connect(second.w);
      expect(JSON.parse(first.posted[0]!).k).not.toBe(JSON.parse(second.posted[0]!).k);
    });

    test("a page that replaces postMessage, JSON.stringify or crypto afterwards does not see the secret", () => {
      const { w, target, posted } = windowWith(kind);
      const ch = connect(w)!;
      const secret = JSON.parse(posted[0]!).k as string;
      const seen: string[] = [];
      const spy = (m: unknown) => void seen.push(String(m));
      target.postMessage = spy as any;
      (target as any).postMessageWithAdditionalObjects = spy;
      w.JSON.stringify = (v: unknown) => (seen.push("stringify:" + JSON.stringify(v)), "{}");
      w.crypto.getRandomValues = (a: unknown) => (seen.push("random"), a);
      // The page can reach prototypes too.
      const callBefore = Function.prototype.call;
      Function.prototype.call = function (this: Function, ...a: unknown[]) {
        seen.push("call");
        return callBefore.apply(this, a as any);
      } as any;
      try {
        ch.post(ch.stringify({ t: "escape" }));
        ch.postWithFiles?.(ch.stringify({ t: "drop", x: 1, y: 2 }), {} as FileList);
      } finally {
        Function.prototype.call = callBefore;
      }
      expect(seen).toEqual([]);
      // What the bridge posted went to the handler it captured.
      expect(posted.at(-1)).toContain(secret);
    });

    test("does nothing without a handler", () => {
      expect(connect({ JSON, crypto })).toBeNull();
    });
  });
}

describe("connect (WebView2 files)", () => {
  test("posts the files with the message, prefixed", () => {
    const { w, posted, files } = windowWith("webview2");
    const ch = connect(w)!;
    const secret = JSON.parse(posted[0]!).k as string;
    ch.postWithFiles!(ch.stringify({ t: "drop", x: 1, y: 2 }), "FILES" as any);
    expect(files).toEqual([[secret + '{"t":"drop","x":1,"y":2}', "FILES"]]);
  });
});
