import { expect, test } from "bun:test";
import { find, stopFind } from "./find";

// An engine limits how many arguments a call may take (a call's arguments are
// on the stack): the stand-in for Highlight fails above 65,535 of them like a
// limited engine does, so the test holds whatever bun's limit is.
class FakeHighlight {
  ranges: unknown[] = [];
  constructor(...ranges: unknown[]) {
    if (ranges.length > 65535) throw new RangeError("Maximum call stack size exceeded.");
    this.ranges.push(...ranges);
  }
  add(r: unknown) {
    this.ranges.push(r);
    return this;
  }
}

test("find highlights 200,000 matches without throwing", () => {
  const g = globalThis as Record<string, unknown>;
  const names = ["document", "CSS", "Highlight", "CSSStyleSheet", "NodeFilter", "getSelection"];
  const saved = names.map((k) => [k, g[k]] as const);
  const n = 200_000;
  const text = { nodeValue: "a".repeat(n), parentElement: { tagName: "P", getClientRects: () => [{}], scrollIntoView() {} } };
  const highlights = new Map<string, FakeHighlight>();
  g.document = {
    body: {},
    adoptedStyleSheets: [],
    createTreeWalker: () => {
      let done = false;
      return { nextNode: () => (done ? null : ((done = true), text)) };
    },
    createRange: () => ({ setStart() {}, setEnd() {}, startContainer: text }),
  };
  g.CSS = { highlights };
  g.Highlight = FakeHighlight;
  g.CSSStyleSheet = class {
    replaceSync() {}
  };
  g.NodeFilter = { SHOW_TEXT: 4, FILTER_ACCEPT: 1, FILTER_REJECT: 2 };
  try {
    expect(find("a")).toEqual({ matches: n, active: 1 });
    expect(highlights.get("mygo-find")!.ranges.length).toBe(n);
    stopFind();
  } finally {
    for (const [k, v] of saved) {
      if (v === undefined) delete g[k];
      else g[k] = v;
    }
  }
});
