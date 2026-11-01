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
  const listeners: Record<string, [(e: any) => void, unknown][]> = {};
  w.addEventListener = (type: string, fn: (e: any) => void, capture?: unknown) => {
    (listeners[type] ??= []).push([fn, capture]);
  };
  w.JSON = { stringify: JSON.stringify };
  w.crypto = {
    getRandomValues<T extends ArrayBufferView>(a: T): T {
      new Uint8Array(a.buffer, a.byteOffset, a.byteLength).forEach((_, i, u) => (u[i] = (i * 17 + 3) & 0xff));
      return a;
    },
  };
  return { w, target, posted, files, listeners };
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

// WebKit restores a page from its back/forward cache on Back and Forward: the
// main frame commits, and the document comes back as it was, with its bridge,
// which does not start again. Go waits for a hello after the commit, so the
// bridge says the secret it has once more when the page is shown from the cache.
describe("connect (pages restored from the cache)", () => {
  const show = (listeners: ReturnType<typeof windowWith>["listeners"], event: object) =>
    listeners["pageshow"]!.forEach(([fn]) => fn(event));

  for (const kind of ["webkit", "webview2"] as const) {
    test(`says the same secret again when a page is restored (${kind})`, () => {
      const { w, posted, listeners } = windowWith(kind);
      const ch = connect(w)!;
      const hello = posted[0]!;
      // Before the page's own scripts: a capturing listener on the window.
      expect(listeners["pageshow"]).toHaveLength(1);
      expect(listeners["pageshow"]![0]![1]).toBe(true);
      show(listeners, { persisted: false, isTrusted: true }); // the first load
      expect(posted).toEqual([hello]);
      show(listeners, { persisted: true, isTrusted: false }); // a script's event
      expect(posted).toEqual([hello]);
      show(listeners, { persisted: true, isTrusted: true });
      expect(posted).toEqual([hello, hello]);
      // The secret did not change: the messages after it carry the same one.
      ch.post(ch.stringify({ t: "escape" }));
      expect(posted.at(-1)).toBe(JSON.parse(hello).k + '{"t":"escape"}');
    });
  }

  test("a page that replaces postMessage does not get the hello of a restore", () => {
    const { w, target, posted, listeners } = windowWith("webkit");
    connect(w);
    const seen: string[] = [];
    target.postMessage = ((m: string) => void seen.push(m)) as any;
    show(listeners, { persisted: true, isTrusted: true });
    expect(seen).toEqual([]);
    expect(posted).toHaveLength(2);
  });
});

// A page that pollutes the builtins the secret passed through (hex digits,
// concatenation) or the objects it stringifies never sees the secret.
test("prototype pollution after connect does not reach the secret", () => {
  const { w, posted } = windowWith("webkit");
  const ch = connect(w)!;
  const secret = JSON.parse(posted[0]!).k as string;

  const seen: unknown[] = [];
  const saved = {
    toString: Number.prototype.toString,
    slice: String.prototype.slice,
    concat: String.prototype.concat,
    stringify: JSON.stringify,
  };
  try {
    Number.prototype.toString = function (this: number, ...a: any[]) {
      seen.push(this);
      return saved.toString.apply(this, a as any);
    };
    String.prototype.slice = function (this: string, ...a: any[]) {
      seen.push(String(this));
      return saved.slice.apply(this, a as any);
    };
    String.prototype.concat = function (this: string, ...a: any[]) {
      seen.push(String(this), ...a);
      return saved.concat.apply(this, a);
    };
    Object.defineProperty(Object.prototype, "toJSON", {
      configurable: true,
      value(this: unknown) {
        seen.push(this);
        return this;
      },
    });
    JSON.stringify = (v: unknown) => {
      seen.push(v);
      return "{}";
    };
    const arg = {
      [Symbol.toPrimitive]() {
        seen.push("toPrimitive");
        return "";
      },
    };
    ch.post(ch.stringify({ t: "call", a: [arg] }));
  } finally {
    Number.prototype.toString = saved.toString;
    String.prototype.slice = saved.slice;
    String.prototype.concat = saved.concat;
    delete (Object.prototype as any).toJSON;
    JSON.stringify = saved.stringify;
  }
  expect(seen.map(String).some((s) => s.includes(secret))).toBe(false);
  expect(posted.at(-1)!.startsWith(secret)).toBe(true);
});

describe("connect (WebView2 files)", () => {
  test("posts the files with the message, prefixed", () => {
    const { w, posted, files } = windowWith("webview2");
    const ch = connect(w)!;
    const secret = JSON.parse(posted[0]!).k as string;
    ch.postWithFiles!(ch.stringify({ t: "drop", x: 1, y: 2 }), "FILES" as any);
    expect(files).toEqual([[secret + '{"t":"drop","x":1,"y":2}', "FILES"]]);
  });
});
