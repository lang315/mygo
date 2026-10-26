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
| A click on a widget table of 1M rows | 6.0–10.4 ms in the first CI runs, 38 MB allocated | **1.22 ms** CI Linux x86, 1.49 ms Windows, 1.61 ms macOS, 0.63 ms on an M1 Pro; **18.6 KB and 514 allocs** (`BenchmarkTableCheckedLarge/click`, #19) | ≤ 1 ms on the three machines, no MB-scale allocation |
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
- App sizes are unchanged. *Open:* the item does not hold, though it can now be
  measured. The size job (`scripts/bench.ts`) also builds `examples/widget-tour`,
  which imports `ui/widget`, so CI shows what the widget layer adds. At
  `e8070ecb` the tour is, beside `counter-native`, 12.84 MB against 9.45 on
  Linux, 13.37 against 9.77 on Windows and 12.51 against 9.22 on macOS: the
  widget layer adds at most 3.39, 3.60 and 3.29 MB. It is an upper bound, since
  the tour has demo code for every control. `hello` and `counter-native` grew
  since 2026-10-23, Linux 9.40 to 9.45 MB, Windows 9.70 to 9.77, macOS 9.17 to
  9.22, the Windows figure past the 9.2–9.7 MB of the Baseline.
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
   - A 1M-row click in ≤ 1 ms. *Open:* 1.22 ms CI Linux x86, 1.49 ms Windows,
     1.61 ms macOS (commit `c4cf118`), 0.63 ms on an M1 Pro; 18.6 KB. The
     phases' rule asks for the three machines.
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
- A click on a table of 1M rows takes ≤ 1 ms. *Open*, see above.
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
- Stop allocating closures every frame in the widget layer. *In part:*
  #169 and #179 took them from the table's rows; `BenchmarkWidgetFrame`
  still makes 0.67 allocation a widget.
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

## Phase 4: developer tools

1. **`widget.Tester`.** It finds widgets by their accessible label or role,
   as egui's kittest does. **Done:** `ui/widget/tester.go`,
   `docs/ui/widget/tester.md`.
2. **A widget layer in the inspector.** It shows the widget tree, the Style
   in effect, and the Tailwind classes that made it. It can force a hover,
   pressed or focus state.
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
