import { describe, expect, test } from "bun:test";
import { type ContentSize, cssZoom, fitContent, fitRule, sizeReporter } from "./fit-content";

const none = { minWidth: 0, minHeight: 0, maxWidth: 0, maxHeight: 0 };

describe("fitRule", () => {
  test("lays out the root at its content's width, without specificity", () => {
    expect(fitRule(none, 0)).toBe(":where(:root){width:max-content}");
  });

  test("bounds the width and the minimum height, not the maximum height", () => {
    expect(fitRule({ minWidth: 100, minHeight: 40, maxWidth: 300, maxHeight: 200 }, 0)).toBe(
      ":where(:root){width:max-content;min-width:100px;max-width:300px;min-height:40px}",
    );
  });

  test("leaves a vertical scrollbar's room out of the width", () => {
    expect(fitRule({ ...none, minWidth: 100, maxWidth: 300 }, 15)).toBe(
      ":where(:root){width:max-content;min-width:85px;max-width:285px}",
    );
    expect(fitRule({ ...none, maxWidth: 10 }, 15)).toBe(":where(:root){width:max-content;max-width:0px}");
  });

  test("takes the fractional CSS pixels of a zoomed page", () => {
    expect(fitRule({ ...none, maxWidth: 133.5 }, 0)).toBe(":where(:root){width:max-content;max-width:133.5px}");
  });
});

describe("sizeReporter", () => {
  test("tells only of changes", () => {
    const told: ContentSize[] = [];
    const report = sizeReporter((s) => told.push(s));
    const size = (w: number, h: number, sw = 0, sh = 0) => ({ w, h, sw, sh });
    report(size(120, 80));
    report(size(120, 80));
    report(size(120.5, 80));
    report(size(120.5, 90));
    report(size(120.5, 90));
    report(size(120.5, 90, 15));
    report(size(120.5, 90, 15, 15));
    report(size(120.5, 90, 15, 15));
    report(size(0, 0));
    expect(told).toEqual([
      size(120, 80),
      size(120.5, 80),
      size(120.5, 90),
      size(120.5, 90, 15),
      size(120.5, 90, 15, 15),
      size(0, 0),
    ]);
  });
});

describe("a page the host zooms with CSS", () => {
  const g = globalThis as Record<string, unknown>;
  const saved = ["document", "innerWidth", "innerHeight", "addEventListener", "__mygoCssZoom"].map((k) => [k, g[k]] as const);
  const restore = () => {
    for (const [k, v] of saved) {
      if (v === undefined) delete g[k];
      else g[k] = v;
    }
  };

  test("cssZoom is 1 unless the host told another", () => {
    delete g.__mygoCssZoom;
    expect(cssZoom()).toBe(1);
    g.__mygoCssZoom = 2;
    expect(cssZoom()).toBe(2);
    g.__mygoCssZoom = 0;
    expect(cssZoom()).toBe(1);
    restore();
  });

  test("tells the content's size in unzoomed CSS pixels", () => {
    // A root of 60 by 60 CSS pixels measures 120 by 120 under html { zoom: 2 }.
    const root = { getBoundingClientRect: () => ({ width: 120, height: 120 }), clientWidth: 240, clientHeight: 200 };
    g.document = { documentElement: root, scrollingElement: root, body: { scrollWidth: 120, clientWidth: 120 } };
    g.innerWidth = 240;
    g.innerHeight = 200;
    g.addEventListener = () => {};
    g.__mygoCssZoom = 2;
    const told: ContentSize[] = [];
    fitContent({ ...none, maxHeight: 100 }, () => {}, (size) => told.push(size)).start();
    restore();
    expect(told).toEqual([{ w: 60, h: 60, sw: 0, sh: 0 }]);
  });
});
