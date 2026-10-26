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
| `TableFrame100k` frame | 122–210 µs (CI) | CI after #179: 199–205 µs (median 202) on an EPYC 7763 runner, 192 and 148 µs on a 9V74 (the runner's CPU decides), 199 µs Windows, 115 µs macOS (M1 VM); 104 µs on an M1 Pro; **22 allocs, 977 B** a frame (223 and 5,989 B on 2026-10-23, 164 and 5,077 B after #169, 22 and 977 B after #179; #182 did not change them) | tens of µs |
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
  and macOS.
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

- Bring `TableFrame100k` down to tens of µs. *Open:* 104 µs on an M1 Pro;
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

  *Proposal, not done:* once `layoutList` has built a row, run `sameLayout`
  on the row's subtree (its elements keep their state by the row's key, so
  they can be compared with the frame before) and let `layoutBox` reuse a
  box laid out at the same size, though the List itself stays volatile.
  *Measured:* an idle frame of the table lays out again 183 of 195 elements
  (12 reused); a plain tree of 62 elements, 0 (62 reused), by the counters
  `engine.laySeq` and `engine.computed`. *Gain, estimated:* the layout of
  44 µs to about 10 µs, a frame of 105 to about 70 µs, and a scroll by a few
  rows too, as most rows stay. *Risks:* the correctness of the layout cache
  (`TestLayoutCacheMatches` has no List or Table case), and a row's height
  that follows the columns a Table fits; a wrong reuse shows as a row drawn
  at a stale size, which is hard to see. *Why not now:* a frame of 105 µs is
  0.6% of a 16 ms budget (about 2.5% on a Windows machine 3–4 times slower).
  *Do it when* a measurement on a real machine shows the table's frame as the
  bottleneck.
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
| Multi-rect GPU damage (`gpu.Ages` keeps one union rect) | a frame is measured GPU-bound where the union is much larger than the rects. Fill costs 8–35 ns per kilopixel on a 10-core Apple-silicon Mac, so 118k pixels saved are 1–4 µs of a frame whose empty floor is 17 µs; a tile GPU reloads the attachment anyway. Present with damage (DXGI `Present1` dirty rects, `eglSwapBuffersWithDamageKHR`) is the likelier win |

## Phase 4: developer tools

1. **`widget.Tester`.** It finds widgets by their accessible label or role,
   as egui's kittest does. **Done:** `ui/widget/tester.go`,
   `docs/ui/widget/tester.md`.
2. **A widget layer in the inspector.** It shows the widget tree, the Style
   in effect, and the Tailwind classes that made it. It can force a hover,
   pressed or focus state. *In part* (#187, #188, #189). **Done:** the notes
   a control leaves for the inspector (`ui/inspector_api.go`: `InspectorField`,
   `Element.InspectNote`; `ui/widget/inspect.go`), shown in the Widget tab:
   the control's type, the Styles that style it by layer (`Apply`'s, then its
   own methods') and by variant, and the Tailwind classes that made each (a
   `tw.S` style, `classes: …`, with "then changed by methods" when a method
   changed it after); and the buttons that force hover, pressed and focus
   (`TestInspectorShowsWhatAControlIsStyledWith`,
   `TestInspectorForcesTheStatesOfAWidget` in `ui/widget/inspect_test.go`,
   `TestInspectorShowsTheClassesOfEachAppliedStyle` in
   `ui/widget/tw/tw_source_test.go`, `ui/inspector_note_test.go`;
   `docs/ui/inspector.md`). *Open:* the widget tree itself, since the Elements
   tab shows the tree of `ui` elements (a `VBox` is a `<Column>`, a `Button` a
   `<Button>`), and the control's type shows only in the Widget tab of the
   element chosen; and the Style in effect as the widget layer resolves it
   (which layer or variant won each property, for the state of the frame),
   since the tab lists each Style as it is set, and only the Styles and
   Computed tabs show what the element ended up with.
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
- A PR merges after CI is green.
- The fork syncs with egoist/mygo on request, and before each phase.
- This fork sends no pull requests upstream.
