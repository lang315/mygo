# Roadmap of this fork

This is the long-term plan of the lang315/mygo fork, dated 2026-10-23. It grows
[`ui/widget`](ui/widget.md) into a complete native UI library for Go. Its
controls feel familiar to WinForms users: Form, TextBox, ComboBox,
DataGridView, MenuStrip and so on. Building a good-looking UI should be as
easy as with HTML and Tailwind. It should be as fast and smooth as egui. A
real app, fhtool, will prove it.

**Status, 2026-10-26.** An item marked **done** names its evidence: a file, a
test or a pull request; `go test ./internal/skillsdocs` checks that every path
named in backticks exists. An item without a mark is open, or has no evidence
here. Numbers are the CI runs on `benchmarks` (commit `c4cf118`, the machines
of GitHub Actions) and a local run on an Apple M1 Pro; times on a laptop differ
by ±25% from layout alone, so allocations and bytes are the figures to compare.
The M1 Pro figures are medians of `go test -run '^$' -bench
'BenchmarkTableFrame100k$' -benchtime 5000x -count 5` in `ui`, and of
`-bench 'BenchmarkWidgetFrame|BenchmarkUIFrame' -benchtime 300x -count 3` and
`-bench 'BenchmarkTableCheckedLarge/click' -benchtime 300x -count 3` in
`ui/widget`, at commit `93a267b` (macOS 15.7, Go 1.27.3; the CI runs Go 1.27.1). The CI figures are
the latest entries of `latest.json` on `benchmarks`, which has a column for
each commit and runner.

The phases run in order. Track 3 runs alongside them. A phase is done when
its exit criteria hold on three real machines: a Mac, Windows 10 amd64 and
Windows 11 ARM64.

## Decisions

- **Where it lives.** `ui/widget` grows in place. Its retained widgets keep
  their state, and every method stays safe from any goroutine.
- **Styling.** `widget.Style` is typed. It has hover, pressed, focused and
  disabled variants and transitions, and it can use everything `ui` draws:
  shadows, gradients, per-corner radius, per-side borders and blur.
  - `ui/widget/tw` compiles Tailwind classes such as
    `"p-2 rounded-l-none hover:bg-blue-600"` to the same Style. It rejects
    unknown classes.
  - `widget.Custom` drops into raw `ui`.
- **Naming.** Controls take WinForms-familiar names where they fit
  naturally. Methods stay Go-style.
- **No Craft suite.** This plan replaces the earlier idea of building the
  Craft apps on mygo in a separate craft-go repository. That study is kept
  as background in [craft-go-port.md](craft-go-port.md).
- **No visual designer.** `mygo dev` live reload, the inspector and the
  widget tour take its place.
- **Out of scope:** a visual designer, native mobile, and D3D12, Vulkan
  and GPU compute. The last three wait for a trigger, listed under Track 3.

## Baseline

| Measure | At the start | 2026-10-26 | Target |
|---|---|---|---|
| `TableFrame100k` frame | 122–210 µs (CI) | CI after #179: 199–205 µs (median 202) on an EPYC 7763 runner, 192 and 148 µs on a 9V74 (the runner's CPU decides), 199 µs Windows, 115 µs macOS (M1 VM); 104 µs on an M1 Pro; **22 allocs, 977 B** a frame (223 and 5,989 B on 2026-10-23, 164 and 5,077 B after #169, 22 and 977 B after #179; #182 did not change them). **2026-10-28, an M1 Pro: 80 µs, 21 allocs, 961 B** (85 µs after the rows reuse their layout, #225; then -1.3% with #238 and -5.45% with #241, each measured against main in turns, so the sum is about 80 µs; then 64.3 µs with the retained rows of #247 and #251, `BenchmarkTableFrame100kFrames/idle`, 21 allocs and 961 B as before; then 26.5 µs of app work idle with a RowVersion and the layout and commit of retained rows skipped, #258, `BenchmarkTableFrame100kPhases/idle-rv-app`, 6.9 µs of layout), the split in Track 3 | tens of µs |
| A click on a widget table of 1M rows | 6.0–10.4 ms in the first CI runs, 38 MB allocated | **`clickApp`, the app's work (build + layout + paint, no rasterizing): 0.67 ms** CI Linux x86, 0.35 ms Windows, 0.37 ms macOS (commit `98bf370`; 0.37 ms on an M1 Pro); **4.96 KB and 238 allocs**. `click`, with the tester's CPU raster standing in for the GPU: 0.74 ms Linux, 0.39 ms Windows, 0.42 ms macOS (1.22, 1.49 and 1.61 ms before #196–#200; 0.42 ms on an M1 Pro, 0.62 before #200) (`BenchmarkTableCheckedLarge`, #19) | `clickApp` ≤ 1 ms on the three machines, no MB-scale allocation. **Met** |
| A frame of N plain widgets | not measured | `BenchmarkWidgetFrame` (M1 Pro): 100 widgets 98 µs and 66 allocs, 1000 widgets 0.77 ms and 666 allocs (CI Linux x86: 0.18 and 1.3 ms); the same as raw `ui` (`BenchmarkUIFrame`): 90 µs and 0.72 ms, 133 and 1,933 allocs; a Style on each: +16–21% (`-styled`), the same styles on raw `ui` +13–14%, so the widget layer adds ×1.03–1.06 (the Phase 0 gate, ≤ 1.10) | baseline set in Phase 0 |
| `counter-native` release size | 9.2–9.7 MB | 9.22 MB macOS, 9.45 MB Linux, 9.77 MB Windows; `hello` 9.37, 9.47 and 9.95 MB | unchanged by widget work |

## Phase 0: foundations

1. **Make `widget` the recommended layer.** Update the docs to say so for
   app UI. **Done:** `docs/ui/README.md`, `docs/ui/widget.md`.
2. **Benchmark the widget layer.** Measure the widget frame for 100 and
   1000 widgets, and what invalidating every window on a change costs.
   **Done:** `ui/widget/bench_test.go` (the frames) and
   `ui/widget/rowsource_test.go` (`BenchmarkRowSourceUpdate`, a change).
3. **Let apps make their own controls.** Add a `widget.Control` interface,
   `widget.Custom`, and a guide to composite controls. **Done:**
   `ui/widget/widget.go` (`Control`), `ui/widget/custom.go` (`Custom`),
   `docs/ui/widget.md`.
4. **Rename once to WinForms-familiar names.**
   - Entry→TextBox, Combo→ComboBox, Notebook→TabControl,
     Spin→NumericUpDown, Frame→GroupBox, NewWindow→Form,
     Table→DataGridView.
   - Add a mapping from gota to widget.
   **Done:** the names table of `docs/ui/widget.md`.
5. **Add `widget.Style`** with its state variants and transitions. **Done:**
   `ui/widget/stylevalue.go`.
6. **Add `ui/widget/tw`.**
   - It covers every class fhtool uses.
   - Semantic colours follow the theme.
   - A `uivet` rule rejects unknown classes.
   **Done:** `ui/widget/tw/tw.go`, `internal/twclass`, `internal/uivet`,
   `docs/ui/widget/tw.md`.
7. **Invalidate only the windows that show a changed widget,** if step 2
   shows the cost matters. **Done:** a change redraws the windows that hold
   the widget (`core.change` in `ui/widget/lock.go`), checked by
   `ui/widget/invalidate_test.go` (`TestChangeRedrawsOnlyTheWindowHoldingTheWidget`,
   `TestOneChangeAmongTwentyWindowsRedrawsOne`).

**Exit criteria:**

- The widget frame with Style is at most 10% slower than the baseline. The
  gate is the cost the widget layer adds to styling, not styling itself:
  (widget-styled ÷ widget-plain) ÷ (ui-styled ÷ ui-plain) ≤ 1.10, from
  `BenchmarkWidgetFrame` and `BenchmarkUIFrame` (`ui/widget/bench_test.go`).
  **Done:** each pair of one runner and one commit, the median of the last ten
  CI runs of `c4cf118` and before: Linux x86 1.06–1.08 (100 and 1000 widgets,
  with and without a transition), Windows 1.07–1.09, macOS 1.01–1.09 on a
  noisy VM (single runs go to 1.5); an M1 Pro run gives 1.03 (100 widgets) and
  1.06 (1000), with a transition. Styling itself, at `c4cf118` on CI, costs a widget +2–24% (Linux +15% without and +21% with a transition at 100 widgets, +15% and +24% at 1000; Windows +15%, +24%, +13%, +23%; macOS +7%, +2%, +16%, +3%); on an M1 Pro +16–21%, and the same styles on raw `ui` +13–14%.
- App sizes are unchanged by widget work. **Done:** no symbol of `ui/widget`,
  `tw`, the inspector or the tour is linked into `counter-native` (`go tool nm`,
  three OSes, at `8840633b`). The widget layer's own cost is the tour minus
  `counter-native`, at most 3.29 MB on macOS, 3.39 on Linux and 3.60 on Windows
  (CI, `e8070ecb`; an upper bound, since the tour has demo code for every
  control). `counter-native` itself grew since 2026-10-23 (Linux 9.40 to 9.45
  MB, Windows 9.70 to 9.77, past the 9.2–9.7 MB of the Baseline, macOS 9.17 to
  9.22) by core work every app links, not widget work. The Windows steps, the
  one measure not rounded to a page (stripped, 512 bytes), from `30f5f81a` to
  `8840633b` (+59.9 KB):

  | Change | Windows | What grew |
  |---|---|---|
  | #101 transitions at rest | +4.6 KB | `animateLayout`, the transitions |
  | #131 masked text box, and its span | +13.8 KB | `ui` text input: replace a range (`editor.replace`, `ReplaceText`), `accessBounds` |
  | #149 skip present when occluded | +3.1 KB | `Occluded` on Windows |
  | #152 D3D debug layer | +5.1 KB | `device.Report` and its helpers, active only with `MYGO_D3D_DEBUG=1`; compiling them out would save 4 KB, not worth a build tag |
  | #160–#168 list and update fixes | +6.7 KB | `listFrame.extendBy` (Cmd+Shift), `update.RetryInUse` |
  | #172 app file writes off the main thread | +10.8 KB | `writeAppFileLater`, `runAppWrites`, `saveWindowStates` |
  | #179 static checkbox paint, ghost themes | +9.7 KB | `checkPath`, `paintCheckmark`, `compactGhostThemes` |

  The other 6.1 KB is small steps (#106 and #109 3.1 KB, #141 0.5 KB, #182 to
  #201 2.6 KB). macOS moves in 16 KB page steps (two of them here, with 1.5 KB
  of content), Linux in 4 KB ones. About half of every step
  is the linker's tables (pclntab, types), one byte for each byte of code. A
  PR that grows `hello` or `counter-native` by more than 4 KB stripped says
  why in its body, as AGENTS.md asks of any growth of an app that doesn't use
  the feature; how to find the cause is "Binary size" in the architecture
  guide.
  - *Open, past the baseline:* `counter-native` (an app with no List, no text
    input and no SVG) links code it does not use. The table below is an
    estimate, **not** a measurement: from `go tool nm -size` and
    `-ldflags=-dumpdep` on main at `8958c04b`, the bytes that fall when a
    group's symbols are removed from the graph. `-dumpdep` prints a tree of
    discovery, one edge per symbol (the one that first marked it), not every
    edge, so a cut edge can drop a subtree that another path still keeps; the
    groups overlap, and a group the app uses cannot be cut. Only a build
    counts: change the code, build stripped for the three OSes, compare. The
    real results so far: SVG -220 KB on macOS (the table said 131 KB, #256);
    the i18n formats below.

    | Group | macOS | Linux | What keeps it | Next step |
    |---|---|---|---|---|
    | editor and text areas (`ui/editor*.go`, `textarea.go`, `textbuffer*.go`) | 285 KB | 250 KB | the type descriptor of `widgetTextInput` (used in an interface), `System.Shape` of `internal/text`, `engine.keyDownPhysical` | break the interface conversion; touches several files |
    | i18n catalog and messages | 215 KB | 188 KB | `mygo.platformTexts`, the package init | the platform's own strings need it: kept |
    | i18n formats (`cldr`, `number`, `format`) | 154 KB | 133 KB | `mygo.regionOf` (on the path of every app), `i18n.loadTables`, `Catalog.format` | **Tried, not merged, measured:** region filled in only when read (`App.Region`, `OnRegionChanged`, native UI) alone: macOS +16.6 KB (a 16 KB page step), Linux 0, Windows -0.5 KB, since `loadTables` and the language-matching tables of the Go text library stay reachable through `Catalog.format` and `i18n.Normalize`; with the number and plural arms of the message formatter removed as well (the most a hook could take): macOS -18 KB, Linux -64 KB (one step), Windows -19 KB. Under the 30 KB bar, and a hook that is not linked would print numbers and plurals wrongly without telling anyone: dropped. The branch feat/lazy-region stays local as a reference |
    | SVG (`ui/svg.go`, `internal/svg`) | 131 KB | 132 KB | the search icon of the combobox parsed by a package variable, and `*svg.Doc` named by the painting | **Done in #256**: -220 KB macOS, -197 KB Linux, -215 KB Windows, `hello` 0 |
    | drag and drop (`ui/dragdrop.go`, `data_drag.go`) | 128 KB | 107 KB | `engine.event` and `engine.runFrame`, the surface type on macOS | a hook; touches `ui/input.go` and `ui/runtime.go` |
    | accessibility bridge (`internal/darwin/access.go`) | 66 KB | | `registerClasses`, `registerSurfaceClass` | kept: VoiceOver needs it |
    | list (`ui/list.go`) | 41 KB | 41 KB | `computeBox` calls `layoutList` (`ui/layout.go`), which pulls `listFrame.build`, `extendBy`, the sorts of `listRow` | a `listLayout` function variable set by `buildList` (as `rowPainter` is), in its own PR after P3 |
    | raster effects, transitions, text selection and menu, path drawing, grid, toast, rich text | 34, 26, 20, 20, 19, 11, 9.5 KB | the same | calls from `ui/layout.go`, `ui/runtime.go`, `ui/paint.go` | after P3 |

    Table, Outline and TreeView add 1.3 KB: their hooks work. Next, each by a
    build before a PR: the list (`listLayout`), the drag and drop and the
    editor, after P3; only what a build shows to be worth it.
- fhtool's classes parse. **Done:** `internal/twclass/twclass_test.go`
  (`TestFhtoolClasses`).
- e2e is green on the three machines.

## Phase 1: what fhtool needs

1. **Label-and-control rows.** Add `widget.Fields`, a two-column grid of
   labels and controls. **Done:** `ui/widget/fields.go`.
2. **Two-way binding.** Add `widget.Bind(control, get, set)`. It uses no
   reflection. **Done:** `ui/widget/binding.go`.
3. **Input details.**
   - TextBox gets `MaxLength` and an icon.
   - Add a standalone RadioButton and Suggested and Destructive buttons.
   - ComboBox becomes editable.
   **Done:** `ui/widget/text.go` (`MaxLength`, an icon), `ui/widget/buttons.go`
   (`RadioButton`, `Suggested`, `Destructive`), `ui/widget/choices.go`
   (`Editable`).
4. **DataGridView for fhtool.**
   - A shared `RowProvider`. **Done** as `RowSource`: `ui/widget/rowsource.go`.
   - A context menu built on click. **Done:** `ContextMenuFunc`,
     `ui/widget/table_events.go`.
   - Rich cells. **Done:** `Column.Spans`, `ui/widget/table.go`.
   - Progress while rows are added in bulk. **Done:**
     `RowSource.AddRowsInBatches` in `ui/widget/rowsource_batches.go`: batches
     in a goroutine, a progress func for a `ProgressBar` or `StatusProgress`, a
     cancel.
   - `UpdateRow` by key. **Done:** `ui/widget/rowsource.go`.
   - A 1M-row click in ≤ 1 ms. **Done:** `BenchmarkTableCheckedLarge`
     (`ui/widget/table_test.go`) reports two figures, and says what each
     measures. **`clickApp`**, the app's work (build + layout + paint of the two
     frames of a click, no rasterizing: `Tester.SkipRaster(true)`,
     `ui/headless.go`), is the ≤ 1 ms target, and CI shows it on the three
     systems at commit `98bf370` (`latest.json` on `benchmarks`): 0.67 ms Linux
     x86, 0.35 ms Windows, 0.37 ms macOS, with 4.96 KB and 238 allocs (18.6 KB and
     514 before #196, which reuses the check mark's path; #201 took the rest).
     The slowest of the last three CI runs is 0.68 ms (Linux; the runner's CPU
     decides, so runs differ by up to 1.7×). **`click`**, with the tester's software raster standing in for the
     GPU, is kept for its history and reported alongside: 0.74 ms Linux, 0.39 ms
     Windows, 0.42 ms macOS (it was 1.22, 1.49 and 1.61 ms at `c4cf118`).
     Two causes of the fall, besides the allocations: a press frame inserted the
     pressed row's background in the middle of the display list and the scrollbar
     at its end, so `scene.Differ` damaged everything between (824,294 of
     1,024,000 pixels on a 1280×800 window, the CPU raster and the GPU's union
     rectangle alike) and now damages 43,116 (the GPU draws their union, about 161,000) (#200: `click` 624 → 424 µs on an M1
     Pro, `present` of that frame 0.246 → 0.009 ms); and `click` depends on the
     cores (on an M1 Pro, 1.14, 0.86, 0.70 and 0.66 ms at `GOMAXPROCS` 1, 2, 4
     and 10, because the raster is shared among goroutines). Evidence:
     `internal/scene/damage.go`, `ui/tester_skipraster_test.go`,
     `internal/raster/damage_multi_test.go`.
5. **App shell.** Add MenuStrip, StatusStrip, ToolStrip and NotifyIcon.
   **Done:** `ui/widget/menustrip.go`, `ui/widget/statusstrip.go`,
   `ui/widget/toolstrip.go`, `ui/widget/notifyicon.go`.
6. **Threading.** `Invoke` waits for the UI goroutine and does not deadlock
   when called on it. `BeginInvoke` returns at once. A handler panic is
   recovered and reported. **Done:** `ui/widget/events.go` (`Invoke`,
   `BeginInvoke`, `OnPanic`).

**Exit gate** (the port is in the fhtool repository, which this repository
cannot show: those items are not marked):

- Three fhtool screens are ported in the fhtool repository: the main
  window with its table, a settings dialog and the toolbar.
- They run on Windows 10 amd64 and Windows 11 ARM64.
- Vietnamese Telex typing works in text boxes and table cells, on Windows
  and macOS. *Open, the owner's decision:* `DataGridView` has no editing in a
  cell (no `ui/widget` control edits the text of a cell in place), so Telex in
  table cells cannot be tested yet; text boxes can. Either the grid gets
  in-cell editing, or the gate drops the table cells.
- A click on a table of 1M rows takes ≤ 1 ms of the app's work (`clickApp`). **Done**, see above: 0.67 ms at worst on CI.
- API changes the port forces land before Phase 2.

## Phase 2: the rest of the WinForms catalogue

**Wrap what `ui` has:**

- TreeView, ListBox, CheckedListBox and ListView;
- SplitContainer;
- DateTimePicker, MonthCalendar, TrackBar and DomainUpDown;
- ColorDialog, RichTextBox, WebBrowser, Chart and PictureBox;
- Timer, drag and drop, and shortcuts.

**Add new controls:**

- ErrorProvider and validation;
- MaskedTextBox;
- ImageList and named icons;
- `&` mnemonics, AcceptButton and CancelButton;
- FontDialog;
- PropertyGrid.

**Larger pieces, each with its own design:**

- **Docking:** document tabs stand in for MDI.
- **Printing:** PrintDocument, PrintPreview and PageSetup.

**Exit criteria:**

- Every WinForms control is implemented, or the reason it was dropped is
  written down, in the [WinForms catalogue](ui/winforms.md), which also
  lists what is still open. **Done:** `docs/ui/winforms.md` has no open
  row. The item groups of ListView, the last, work in all four views:
  `SetGroupBy`, `SetGroupOrder` and `Groups` in `ui/widget/listview.go`, on the
  groups of `ui.TableView` (`ui/tableview_groups.go`) and `GridState.SetHeaders`
  (`ui/gridheader.go`); tests in `ui/widget/listview_groups_test.go`,
  `ui/widget/listview_details_groups_test.go` and
  `ui/tableview_groups_test.go`; benchmarks in
  `ui/tableview_groups_bench_test.go` and
  `ui/widget/listview_details_groups_bench_test.go`. Not built, and not exit
  criteria: collapsible groups, a header with an icon, and a header text the app
  chooses (it is "Title (n)").
- Each control has docs, tests and an entry in the widget tour. **Done:**
  `docs/ui/widget/`, `ui/widget/`, `examples/widget-tour`.

## Track 3: engine and platform, alongside

**Soon** (all done):

- A D3D11 debug layer and DRED, behind an environment variable. **Done:**
  `internal/gpu/d3d11/device/debug.go` (the debug layer; Direct3D 11 has no
  DRED).
- A test that bounds purego and syscall callbacks. **Done:**
  `internal/e2e/callbacks_test.go`.
- Correct Objective-C encodings for `uintptr` parameters. **Done:**
  `internal/darwin/encoding_test.go`.
- `-race`, `-shuffle=on` and a Linux arm64 job in CI. **Done:**
  `.github/workflows/ci.yml`.
- A gate before each Go upgrade. purego's fakecgo uses runtime linknames.
  **Done:** `.github/workflows/go-next.yml`.
- Skip Present while a window is occluded. **Done:** `ui/window.go`
  (`Occluded`), `internal/platform/surface.go`.

**Later:**

- Bring `TableFrame100k` down to tens of µs. *Open:* 104 µs on an M1 Pro
  (85 µs once the rows reuse their layout, below);
  allocations went from 223 to 22 a frame (#169: 164, #179: 22).
  Where the 104 µs goes (an M1 Pro, 195 elements, the scene of
  `BenchmarkTableFrame100k`, the mean of 100,000 frames; 0.55 µs an element):

  | | build | layout | paint | present | other | frame |
  |---|---|---|---|---|---|---|
  | text in the cells | 26.8 µs | 44.1 µs | 25.4 µs | 7.9 µs | 3.4 µs | 107.5 µs |
  | boxes in the cells, same 195 elements | 25.8 µs | 37.9 µs | 9.6 µs | 4.5 µs | 3.2 µs | 80.9 µs |

  The text costs about 5 µs of layout, 16 of paint and 3 of present; the
  layout of 195 plain elements is still 38 µs. The frame is the work itself:
  the process CPU (`getrusage`) equals the wall time, 105.5 and 105.4 µs.
  *How to measure:* time the frame with `time.Now` and read the phases from
  the engine's own timers, `frameStats.laps` (`ui/framestats.go`; a test sets
  `rt.stats = newFrameStats(time.Hour)`; `MYGO_FRAME_STATS=all` logs them to a
  tenth of a millisecond). Do not split a frame with pprof on macOS: it
  samples about 85% of the time, counts the setup's `madvise` and gave a
  frame of 37 µs.
  *Layout cache:* in an idle frame of this table 183 of its 195 elements are
  laid out again, and a plain tree of 62 elements lays out none
  (`ui/layoutcache.go`: a List builds its rows while it lays out and a Table
  fits its columns, so they and what is inside them are never "the same").
  Reusing the layout of the rows a List builds is the lead for the 44 µs.

  **Done** (the owner asked for it on 2026-10-27, which overrode the "do it
  when" condition): once `layoutList` has built a row, `sameRow` runs
  `sameLayout` on the row's subtree (its elements keep their state by the
  row's key, so they are compared with the frame before) and `layoutBox` and
  `heightAt` reuse what a row that is the same laid out, though the List
  itself stays volatile (`layRows` in `ui/layoutcache.go`, the calls in
  `layoutList`). A Table's cells get their widths when they are built, so
  the comparison sees a column's width changing; the table's `colFit` is set
  only while a column is being fitted, and then nothing inside is reused.
  *Measured:* an idle frame of the bench table lays out again 3 of its 195
  elements (183 before), by `engine.computed`; `BenchmarkTableFrame100k` on
  an M1 Pro, medians of 6: 107.7 to 84.5 µs (-21%), 961 B and 21 allocs a
  frame as before; `BenchmarkWidgetFrame` is unchanged (within 1%).
  `TestLayoutCacheMatches` now builds Lists and Tables with rows to reuse,
  and scrolls, sorts, resizes and fits them; with `sameRow` told that
  nothing around the rows changed, it fails. The 70 µs the estimate hoped
  for is not reached: what remains is build (27 µs), paint and present.

  **Done** (text of paint, #238, the owner's go on 2026-10-28): `textForeground`
  (`ui/paint.go`) asks the text system for the glyphs of a line in one call,
  `System.Glyphs` (`internal/text/raster.go`): one lock and one placement
  lookup per font run, where `Glyph` took the lock for each glyph, with the
  same keys and the same atlas entries, so the pixels do not change. The 190
  glyph lookups of the idle frame cost 9.4 µs one by one and 5.3 µs under
  one lock. *Measured:* `BenchmarkTableFrame100k` on an M1 Pro with other
  sessions' tests running (load average 10), 40 runs of each in turns,
  `benchstat`: 88.24 to 87.10 µs (-1.3%, p=0.012), 961 B and 21 allocs a frame
  as before. That is about a fourth of the 4–5 µs the lock was worth: a cell's
  text is a line of three to nine glyphs, so a call serves few of them, and
  the placement and map lookups, 5 µs, stay. `BenchmarkWidgetFrame` and
  `BenchmarkUIFrame` show no difference (3 runs, within the noise). Ruled
  out in the design of that day (a local spec, not in the repository): replaying the ops of a row whose layout is reused, which needs the glyph
  LRU stamps, positions that are not translations at a fractional scroll and
  the paint-time state of hover and focus (P1-B, below, does it at a zero
  offset, with those in the key); and a per-text cache of resolved
  glyphs, which a fractional scroll misses. *Present:* `scene.Differ.Diff`
  of the idle frame's scene (144 ops, 191 glyphs) takes about 6.9 µs of the
  8.0 µs present lap, so telling an identical scene earlier was worth a
  follow-up (its gate was 3 µs): #241, below.

  **Where the frame goes** (measured 2026-10-28, an M1 Pro with other
  sessions' tests running, load average 16; medians of 3 runs of 5,000
  frames; the laps of `frameStats` in `ui/framestats.go`, and timers placed
  around `layoutTree` and the parts of `commit`, never committed; the total
  of 87.0 µs is that of a quiet machine, 85–88):

  | scene of `BenchmarkTableFrame100k` | build | layout | paint | present | frame |
  |---|---|---|---|---|---|
  | the benchmark | 26.89 | 22.96 | 25.73 | 8.07 | 87.04 |
  | cells are empty boxes, no text | 25.87 | 23.01 | 10.10 | 4.83 | 67.28 |
  | the same Table with no rows | 2.08 | 1.96 | 2.32 | 0.98 | 8.29 |
  | the app's code alone, no Table | 0.29 | 0.38 | 0.16 | 0.14 | 1.61 |
  | an empty window | 0.24 | 0.37 | 0.16 | 0.13 | 1.54 |

  So the floor of a frame is 1.54 µs (the 17 µs written here before was
  another measure), the Table's own chrome (header, scrollbar, background)
  8.3 µs, and the rows about 79: 24 µs of build (the builders of the rows and
  cells; `ui.Text` and `ui.Textf` are 1.0 of it), 11.7 of `layoutTree`, 8.5
  of the walk of `commit` (about 47 ns a node), 23.4 of paint (7.8 for the
  rows' boxes and 15.6 for their text) and 7 of present. The app's code
  (`view.Sync` 5 ns, the key closure 24 B and 10–14 ns) is nothing.

  **Done** (present, #241 and #242): `Differ.Diff` in
  `internal/scene/damage.go` takes a scene identical to the last one by its
  bytes (`identical`: ops, glyphs, stops, wide colors, the atlases, the image
  versions, `Reach`, `Wide` and folding) and answers as the long comparison
  does, without computing the bounds, `Match` and `Bounds` included; any
  difference, and any scene with effects, takes the long way. *Measured:*
  `BenchmarkTableFrame100k` on an M1 Pro, 40 runs of each in turns,
  `benchstat`: 85.02 to 80.39 µs (-5.45%, p=0.000), the present lap 8.0 to
  2.6 µs, 961 B and 21 allocs as before. Evidence:
  `internal/scene/damage_identical_test.go` compares the answers with and
  without the path for 25 changes of a scene, and every condition of
  `identical` fails it when dropped. A probe over the tests of `ui`,
  `internal/raster`, `internal/scene` and `internal/gpu` found 3,204 frames
  whose scenes had not changed missing the path because the padding of an
  op or a glyph differed in bytes (the Painter fills them field by field in
  a slot it does not clear); #242 compares such slices by values where the
  bytes differ: 118,614 hits became 121,821.

  **Done** (retained rows, the owner's go on 2026-10-28; the design is a local
  spec, not in the repository). The phases, each with a measured gate:

  - *P0 (#246):* painting leaves the nodes as they were built (the hover looks
    of `ui/hoverstyle.go` no longer write the node), the themes keep their
    indexes across frames (`themeGen`), and `BenchmarkTableFrame100kFrames`
    measures the frames users feel: idle, hover, one row updated, scrolled by
    1 px and by half of one.
  - *#249:* `BenchmarkTableFrame100kPhases` reports the phases of those frames,
    and its variants ending in `-app` skip the Tester's rasterizer
    (`Tester.SkipRaster`) to measure the app's work alone. The `scroll1px`
    frame is 96% the Tester's CPU rasterizer, not the app
    (`ui/api_bench_test.go`).
  - *P1-A (#247):* as the layout cache walks each row (`sameRow`,
    `ui/layoutcache.go`) it also tells whether the row's elements paint with
    the same inputs as in the last layout: `layPaint`, from `paintStyle` and
    `textPaint` (`ui/paintstyle.go`), which hold the node fields that paint
    and the text styles that paint and do not lay out; static `styleFn`s carry
    an identity (`ui/staticstyle.go`); `TestPaintInputs` places every field of
    `node`, so a field added without a decision fails it. It costs the layout
    lap 3.6 to 4.1 µs and shows nothing alone, so it merged with P1-B.
  - *P1-B (#251):* a row with `layPaint` that paints twice in the same place
    records its operations, glyphs and the atlas entries it used
    (`ui/replay.go`, `internal/text/uses.go`), and replays them while the key
    holds: its box, the clip, scale, opacity, the opaque backgrounds around it,
    the atlas generations, the text parameters, `themeGen`, the focus, and each
    element's id, flags and box, with no element hovered, pressed or focused.
    Rows are replayed at a zero offset only; a row that moves does not record.
  - *Gate*, measured on the stack against `3763a968` (main before #246) with
    #246's benchmarks, 8 interleaved runs on an M1 Pro: the idle frame 79.4 to
    64.3 µs (-15.1; the gate was -10), the idle paint lap 25.4 to 5.8 µs (at
    most 8), the hover paint lap 26.5 to 9.0-9.3 µs (at most 10), allocations
    and bytes a frame unchanged; the half-pixel scroll frame costs 5.6 µs
    more (+6%), since a moving row records nothing and P1-A's comparison runs.
    `counter-native` grows about 3 KB (`hello` 0): the rest is behind
    `linkPaintCompare`, which building a list calls.
  - *The oracle:* `rt.replayCheck` paints every replayed row fresh as well and
    compares the operations, glyphs and flags; `TestReplayMatches`
    (`ui/replay_model_test.go`) runs it over random trees with lists and tables
    that change, restyle, hide, scroll, hover and take focus, at scales 1 and
    2, against a painting that replays nothing. It found a row's inner clip
    changing with a table column while the row looked the same (every element's
    box is in the key now).
  - *The bug the second review found:* a row that painted anew without
    recording (its look changed, or an ancestor was hidden while it changed)
    kept an older recording, and replayed it once it stood still again: the row
    chosen with the keys looked unchosen. The row's `at` (the layout it last
    painted in) now keeps a recording older than the frame before from
    replaying; three tests and the model's paint-only restyles and idle frames
    fail without it. The first review and the tests had missed it because the
    layout cache's styles all change layout.

  - *P2 (#254):* `ListState.RowVersion func(i int) uint64`: a List or Table
    whose app sets it keeps the rows it built and links them again instead of
    building them while the row's key, version, choice, checks, theme and the
    pending input of its states hold (`ui/retain.go`, `state.pending` in
    `ui/state_pending.go` with every field of `state` placed by
    `TestStatePendingPlacesEveryField`). The widget layer versions its rows
    (`RowSource`, `ListBox`, `ListView`, `Table`, `TreeView`), so apps on it
    get kept rows without writing a RowVersion. *Gate*, 15 interleaved runs on
    an M1 Pro (load 7.2 to 5.4), app-only laps, against main before #254, with
    a RowVersion on the table: build lap idle 26.38 to 10.77 µs (the gate was
    at most 10, **missed** by 0.77), hover 26.20 to 12.44 (-52.5%), update
    26.49 to 11.70 (-55.8%); idle frame 61.84 to 47.48 µs (-14.36; the gate was
    -15, **missed** by 0.64); allocations unchanged (21), bytes unchanged;
    **layout lap +1.36 idle, +3.10 hover, +2.39 update** (the gate was +2: hover
    and update **missed**, since layout still walks kept rows and the build now
    reads the choice around a row), which moves to P3. Without a RowVersion the
    benchmark is within 1% and `BenchmarkWidgetFrame` +0.26% (0 allocs).
    Skipping `state.pending` on "no event since the last frame" was rejected:
    pending fields are also set by programmatic calls, accessibility actions
    and posted work. `counter-native` grew 624 bytes on macOS, 0 on Linux and
    10.7 KB on Windows (`go tool nm`: `listRow` gained `ver`, `chosen` and
    `checked`, so every sort of it is a new instance, `listFrame.build`
    +928 bytes, `chosenAround`; Windows rounds to 512 bytes).
  - *The oracle and the reviews of P2:* `TestRetainMatchesFresh` runs a
    retaining Tester against one that builds every row over random Lists and
    Tables (tint changes with a version, choice and check changes without
    one, theme, hover, clicks, Tab, scroll, scales 1 and 2) with `replayCheck`
    on; 18 mutations of `ui/retain.go` (each check dropped in turn) each fail a
    test, except two that are equivalent (keeping `lay` and the per-frame
    measures on a kept node). An independent review found three critical
    problems, all fixed with a test that failed first: rows linked in an
    earlier pass of a frame kept states that prune had recycled when a later
    handler hid or scrolled the list; a row with an `EditableText` was kept,
    so Enter opened it instead of renaming; a `Draw` closure reading its
    Element panicked on a kept row. A second review found that the copy of a
    row was recorded with the version read at the end of the frame, not the
    one it was built with. The oracle does not yet generate two-pass hides,
    keys, context menus, `EditableText`, `After` or `Draw` (direct tests do);
    the branch rows of an Outline and TreeView are built every frame (their
    arrow animation leaves `motions` set).

  - *P3 (#258):* a kept row that stays where it was, at the same width and
    with the same surroundings, keeps the results of its last layout and
    commit (`ui/retain_layout.go`): `layoutList` neither compares nor lays it
    out (`settle`), `place` finds it where it was (`keptPlace`), and `commit`
    appends the hits and labels it recorded (`commitRow`) while what commit
    reads around it (clip, hidden, parent disabled, focus scope, page, parent
    world) is the same. A row that moved is laid out and committed as built.
    `place` and `commitElement` reach the code only through engine function
    fields set beside `rt.keepRows`. *Gate*, 30 interleaved runs on an M1 Pro
    (load 5 to 6), against main before #258, with a RowVersion and the raster
    skipped (`-rv-app`): idle frame 47.79 to **26.47 µs** (the gate was at
    most 30), idle layout lap 27.57 to **6.88 µs** (at most 8), hover -33.4%
    and update -38.6% (at least 25%), scroll1px and scrollhalf -0.7% and
    -1.1% (at most +3%), allocations and bytes unchanged;
    `BenchmarkWidgetFrame` within noise, 0 allocs. **Missed:** without a
    RowVersion the Table variants are +1.0 to +2.0% (the gate was 1%) and the
    build lap, which P3 does not touch there, is +2.2%; a control (one unused
    `uint64` in `state` on main) moves them by +0.5 to +1.6%, and removing
    each hook alone moved it by less than one point, so it is read as the
    binary's layout on the M1; main's `bench.yml` run of `TableFrame100k`
    without a RowVersion is to be compared with the commit before the merge,
    and a stable x86 slowdown over 3% is a follow-up. `hello` +0 bytes;
    `counter-native` +16944 bytes on macOS (text +2112, the rest is the 16 KB
    rounding of the segments), +6144 on Windows, 0 on Linux (`go tool nm`:
    `layoutList` links `settle`; `keptPlace`, `commitRow` and `checkReplay`
    are not in the binary).
  - *The oracle and the reviews of P3:* `TestRetainMatchesFresh` compares
    nodes and states as well as pictures and hits, and runs with a twin
    (`rt.layoutCheck`: every held row is laid out and committed anyway and
    the differences counted); 450 seeds at scales 1 and 2 pass with no
    difference. Each guard of `settle`, `keptPlace` and `commitRow` is undone
    in turn: all but a few are killed by a directed test that flips that
    input alone, one (`buildLate` zero) by a pinned model seed, and the rest
    were found equivalent and either dropped or kept with a comment (the
    table is in #258). The audit of what walks into a list's rows after the
    build (`ui/retain_audit_test.go`) found no hole. The independent review
    found one: **a scroll container inside a kept row showed its old offset**
    when the offset changed with no build of the row (a wheel over it while
    the pointer is pressed elsewhere, a drag of its bar after the pointer left
    the window, `dragScroll`), since `keptPlace` returns before `place`, which
    applies the offsets, and nothing marks the row. `commitRow` now records the
    offsets of the row's scroll containers and `settle` compares them. The
    model had missed it too: its rows had no scroll container and its wheel
    never aimed at one; both were added and the model only counted once the
    mutant (the comparison dropped) failed at seed 103. The directed tests of
    the moved path used a list that did not scroll (a column hugging its
    content); fixed, and the helper now fails when the list does not move.

  - *P4a (#262):* a kept row that moved (a scroll) is no longer laid out
    again: it keeps the relative `x, y` of its elements (`rowKeep.rel`, saved
    at the `place` hook before `place`) and `place` runs again over them, so the
    boxes have the bits of a layout (a float shift of the absolute boxes
    differs in the last bit, and the oracles compare with `==`). `scroll1px-rv-app`
    81.2 to 68.3 µs on its own. Its review found a row placed twice in one
    layout (a List inside a Transition's `Match`, placed again by `showAt`) and
    an oracle that ended by zeroing the record it checked (`rowSnap` now
    snapshots and restores); #265 added the nested `Match` whose inner subtree
    shifts after the place (`forgetKeptRows`).
  - *P4b (#270):* the paint of a moved row is replayed shifted, by whole
    device pixels, when every coordinate that is shifted is an integer (solid
    fills, shadows, clips and glyphs; `shiftable`, `translate` in
    `ui/retain_translate.go`), the row is whole inside the clip in Y when it
    records and when it replays, and the cover by an opaque background (which
    decides subpixel text) is the same. A shift by a fraction of a device pixel
    paints anew and keeps the record; a row that stands still there records
    where it stands. The shift is `float32(dy * scale)`: arm64 fused the
    product into the add and left `119.00001` for `119` in the record (a
    tolerance cannot do it: it takes a real shift of 6e-5 pixel). The twins of
    the tests (`newPair`, the model) had replayed too (`noRetain` does not turn
    replay off) and now set `noReplay`.
  - *Final gate, P3 (b8609dbe) against main with P4 and the fixes below*, M1
    Pro, 25 interleaved runs, `-benchtime 3000x`, benchstat, 100k-row table
    with a RowVersion and the raster skipped (`-rv-app`): `scroll1px`
    **79.3 to 52.1 µs (-34.2%)**, `scrollhalf` **79.3 to 60.3 µs (-23.9%)**,
    idle 25.9 to 25.4 (-1.8%), hover 38.2 to 38.6 (+0.9%, p=0.006), update
    31.1 to 30.7 (-1.2%); allocations of a scroll frame 47 to 45. Plain table
    (no RowVersion): `scroll1px-app` 96.8 to 81.8 µs (-15.5%), allocations 69
    to 46; idle -0.8%, update -0.9%, **hover +1.4%** (69.0 to 70.0 µs,
    p=0.000), which was not chased. Linux and Windows numbers were not
    measured.
  - *Fixes found by the reviews and the models:* #264 (a row holding the
    element whose menu is drawn is built, not linked), #268 (the measures of
    the text of held rows are forgotten when the list's text styles changed:
    a box hugging a List kept the old width, model seed 11; found when the
    model gained right-clicks), #272 (a held row that was compared as painting
    differently, as after a change of theme, was kept without the comparison
    that gives it `layPaint` and painted anew for good: it is laid out again in
    up to two layouts, `rowLay.tries`; 21 of 201 rows replayed after
    `SetDark`), #274 (a pointer outside the window tests no hit, a move is
    undone by a subtraction, `forgetInput` visits the states the pass built:
    the idle frame at a scrolled offset 31.7 to 28.6 µs), #275 (`ListState.Key`
    is asked once or twice a row in a pass, not four times with a Selection; a
    key is asked again after a row's callback ran, which may change the data:
    the first version kept a wrong row for good) and #277 (`ListBox`,
    `DataGridView`, `ListView` and `DataTable` keep their keys boxed: a
    `ListBox` of 5000 items makes 5 allocations in an idle frame, from 118).
    `-race` runs the model, the fuzz and `TestRetainShiftPaintMixedScales`
    with fewer seeds (the package took 1036 s and 488 s now).
  "Tens of µs" in the sense of 20–30 needed the build skipped (P2, 47 µs idle
  on an M1 Pro) and the layout and commit (P3, 26.5 µs); a scroll frame is 52 µs.
  *Open:* the layout twin (`rt.layoutCheck`) compares the text layouts and
  lay bits of a kept row as well as its boxes (#271, open). A pointer resting
  on a row adds 3.5 to 4 µs and 26 allocations to an idle frame: the row under
  it is hovered, so it is built, laid out and painted in every frame and never
  replayed (`rowInteractive`), even when it only has a hover look and no hover
  style; such a row need not be built again. `idle-scrolled-rv-app` (offset
  2000, the pointer outside the window, #273) is 28.8 µs against 25.4 at the
  top; its 46 allocations a frame against 21 are the benchmark's own `Key`
  boxing an id of 256 or more (`ListState.Key` says so), not the library.
- Stop allocating closures every frame in the widget layer. **Done** for the
  controls `BenchmarkWidgetFrame` builds: #169 and #179 took them from the
  table's rows, and a button's and a text input's look are now static functions
  reading the node (`buttonPaint` in `ui/widgets.go`, `inputStyle` in
  `ui/textinput.go`), so the benchmark makes no allocation a frame (66 to 0
  at 100 widgets, 666 to 0 at 1000). #201 did the same for the hover looks of
  rows, items, triggers and cells (`hoverSurface`, `hoverRow`, `pressSurface`
  in `ui/hoverstyle.go`; `inputStyle` for the combobox's and the buffer input's
  borders), per instance in a frame: calendar 92 to 50 allocations, tree item
  6.03 to 5.03, sidebar item 4.03 to 3.02, breadcrumbs 12 to 10, collapsible 8
  to 7, stepper 15 to 13; list rows (0.23) and grid cells (0.15) made few. The
  closures that remain are those of tabs, the radio dot, a form's label, the
  combobox field, the time input and the toast: they read another node or
  slices, or state the node does not hold.
- Merge batches of shader effects.
- Compile shaders ahead of time.

**Each when it is needed:**

| Item | When |
|---|---|
| An atlas of bitmaps | image-heavy grids are slow |
| GPU path filling | charts or large paths are slow |
| Re-syncing the Kage fork | a newer Kage feature is needed |
| A newer purego | a tag carries the fixes we need |
| TSF input on Windows | IMM32 misbehaves with Telex |
| D3D12, Vulkan or GPU compute | an app needs image processing |
| Present with damage where it is not yet (`eglSwapBuffersWithDamageKHR`; Direct3D presents dirty rectangles already, `Present1`) | a frame is measured compositor-bound in presenting the whole surface. Redrawing several damage rectangles apart, not their union, is built (`gpu.Ages` in `internal/gpu/damage.go`, from upstream's #754, fed by `scene.Differ`'s damage rectangles); the fill it saves is 1–4 µs of a frame whose empty floor is 1.54 µs on a 10-core Apple-silicon Mac (17 µs was written here before it was measured with the engine's laps, 2026-10-28), so presenting only the damage is the likelier win left |

## Phase 4: developer tools

1. **`widget.Tester`.** It finds widgets by their accessible label or role,
   as egui's kittest does. **Done:** `ui/widget/tester.go`,
   `docs/ui/widget/tester.md`.
2. **A widget layer in the inspector.** It shows the widget tree, the Style
   in effect, and the Tailwind classes that made it. It can force a hover,
   pressed or focus state. **Done** (#187–#189, #204, #205):
   - the notes a control leaves for the inspector: `InspectorField` (with
     `Struck`) and `Element.InspectNote` in `ui/inspector_api.go`, and the
     control's note in `ui/widget/inspect.go`: its type, whether it is
     disabled, the Styles that style it by layer (`Apply`'s, then its own
     methods') and by variant, and the Tailwind classes that made each (a
     `tw.S` style, with "then changed by methods" when a method changed it
     after);
   - the buttons that force hover, pressed and focus;
     `TestInspectorShowsWhatAControlIsStyledWith`,
     `TestInspectorForcesTheStatesOfAWidget` in `ui/widget/inspect_test.go`,
     `TestInspectorShowsTheClassesOfEachAppliedStyle` in
     `ui/widget/tw/tw_source_test.go`, `ui/inspector_note_test.go`;
   - the widget tree: the Widgets tab, which lists the controls as the app
     wrote them, nested by their notes, and chooses a control's element
     (`ui/inspector.go`, `ui/inspector_panel.go`); `ui/inspector_widgets_test.go`,
     `TestInspectorWidgetsShowTheControlTree` in `ui/widget/inspect_test.go`;
   - the Style in effect: for the state of the frame, forced states
     included, each property's winning value with its layer and variant, and
     the values it lost to struck through (`ui/widget/inspect_effect.go`);
     `TestInspectorShowsTheStyleInEffect` and, against what `Merge` and
     `effective` resolve, `TestInEffectIsWhatEffectiveResolves` with its
     siblings in `ui/widget/inspect_effect_test.go`;
   - `docs/ui/inspector.md`.
3. **Docs to copy from.** The widget tour, a Tailwind cheat sheet, and
   mappings from WinForms and gota. **Done:** `examples/widget-tour`,
   `docs/ui/widget/tw.md` (checked against the parser by
   `internal/twclass/cheatsheet_test.go`), `docs/ui/winforms.md` and the names
   table of `docs/ui/widget.md`.
4. **Skills for agents.** Add a `skills/` folder and matching AGENTS.md
   rules. **Done** in the CLI's bundled skills, installed by `mygo
   install-skills`: `cmd/mygo/template/shared/.agents/skills`, checked by
   `internal/skillsdocs`, and the rule in `AGENTS.md`.
5. **An inspection server** for agents and e2e tests, when one is needed.

## How work lands

- A large item gets its own design first. Each phase has a plan before
  work starts.
- Every PR carries evidence: build, vet, tests, benchmarks and app sizes,
  and its e2e runs on the three machines.
- A PR merges after CI is green. CI shards the unit tests and, for a pull
  request, tests only what the change affects (#235): the longest job takes
  13.5 minutes, where the critical path took 24.7 before.
- A test never passes or fails on wall-clock time (AGENTS.md). The flakes
  that broke this on 2026-10-28, each fixed by waiting for the event or
  driving a fake clock, and not by a longer timeout (the last is a
  diagnostic, for the alloc-count flakes):
  - `examples/account-manager/stress.go`: `stress` takes its ticks and start
    time from the caller, and its test sends them (#231);
  - `internal/watchdriver/watch_test.go`: `TestWatch` waits for `Remove` to
    run before it fails `Wait` (#236);
  - `ui/widget/window_test.go`: `TestOnCloseAfterQuitAttemptDoesNotQuit`
    waits for the first `OnClose` to return (#237);
  - `examples/account-manager/main_test.go`,
    `examples/account-manager/worker_test.go` and
    `internal/update/rename_test.go`: a wait on the run's end, scheduler
    turns for the overlap, and the file let go at `Rename`'s first wait
    (#240);
  - `ui/allocsites_test.go` and `ui/memory_test.go`: the alloc-count tests
    report where the mallocs came from when they are over the limit, so a
    failure on a CI runner names the call and not only the count (#239).
- The fork syncs with egoist/mygo on request, and before each phase.
- This fork sends no pull requests upstream.
