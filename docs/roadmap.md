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
| `TableFrame100k` frame | 122–210 µs (CI) | on CI Linux x86 it depends on the runner's CPU: median 113 µs on an EPYC 9V45, 154 on a 9V74, 208 on a 7763 (the common one; 202 in the last run), 199 µs Windows, 115 µs macOS (M1 VM), 104 µs M1 Pro; **22 allocs, 977 B** a frame (was 164 and 5,077) | tens of µs |
| A click on a widget table of 1M rows | 6.5–10 ms, 38 MB allocated | **1.2 ms** CI Linux x86, 1.5 ms Windows, 1.6 ms macOS, 0.63 ms M1 Pro; **18.6 KB and 514 allocs** (`BenchmarkTableCheckedLarge/click`, #19) | ≤ 1 ms, no MB-scale allocation |
| A frame of N plain widgets | not measured | `BenchmarkWidgetFrame` (M1 Pro): 100 widgets 98 µs and 66 allocs, 1000 widgets 0.77 ms and 666 allocs (CI Linux x86: 0.18 and 1.3 ms); the same as raw `ui` (`BenchmarkUIFrame`): 90 µs and 0.72 ms, 133 and 1,933 allocs; a Style on each: +16–21% (`-styled`), the same styles on raw `ui` +13–14% | baseline set in Phase 0 |
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
   `ui/widget/custom.go`, `docs/ui/widget.md`.
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
   shows the cost matters. **Done:** a change redraws the windows that show
   the widget (`docs/ui/widget.md`, first paragraph).

**Exit criteria:**

- The widget frame with Style is at most 10% slower than the baseline.
  *Measured:* a Style on every widget costs +16–21% in `BenchmarkWidgetFrame`
  (`-styled`, M1 Pro; +13–23% on CI), and the same styles written on raw `ui`
  cost +13–14%: this benchmark does not show the criterion.
- App sizes are unchanged. **Done:** see the Baseline table.
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
   - Progress while rows are added in bulk. *Open:* nothing in
     `ui/widget` shows it.
   - `UpdateRow` by key. **Done:** `ui/widget/rowsource.go`.
   - A 1M-row click in ≤ 1 ms. **Done** on the CI Linux x86 runners at
     1.2 ms (0.63 ms on an M1 Pro), 18.6 KB: see the Baseline table.
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
- A click on a table of 1M rows takes ≤ 1 ms. **Done:** see the Baseline
  table.
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
  lists what is still open. **Done**, but for the item groups of ListView in
  Details, in progress (`docs/ui/winforms.md`, "Open rows").
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
  allocations went from 164 to 22 a frame (#169, #179, #182).
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
