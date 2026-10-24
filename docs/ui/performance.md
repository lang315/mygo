# Native UI performance

These measurements cover steady frames rendered on the CPU with `ui.Tester`.
They were taken on October 8, 2026 on Apple M5, macOS arm64, Go 1.27.1, with
`CGO_ENABLED=0 GOMAXPROCS=1`. Each timing is the median of six sequential
300 ms samples after warming the view.

| Fixture | Time per frame | Allocations per frame | Allocated bytes per frame |
|---|---:|---:|---:|
| 1,000 static labels | 397 µs | 0 | 0 |
| Visible rows of a million-row list | 14.5 µs | 20 | 176 |
| Diff view with no changes | 389 µs | 2 | 72 |
| A 50 KB Markdown reply, shown whole | 1.3 ms | 353 | 15,894 |
| The same reply streaming, 16 bytes a frame | 0.8 ms | 350 | 35,946 |

The Markdown rows, `BenchmarkViewSteady` and `BenchmarkViewStream` of
`ui/markdown`, were taken on October 14, 2026, on a busier machine; see
[Markdown](markdown.md#performance) for the parser's own costs.

Element handles validate their owner, arena slot and build generation.
Static label handles do not allocate. Input bindings, action queues and
persistent identity bindings reuse storage once warmed. Composite controls
can allocate callback closures; these fixtures do not cover every widget
or interaction.

The label, list and table fixtures use the public API. The diff fixture uses the
private renderer API and measures its steady-frame allocation floor.

## Tables of many rows

`BenchmarkTableFrame100k` builds an idle frame of a table with a checkbox
column and two columns of text, sorted by a `TableView`, and does no work
that grows with the rows. It allocates more than the list, as each row in
view builds a checkbox (about three allocations: its state and two
closures) and a cell per column.

The numbers below were measured on another machine than the first table:
an Apple M1 Pro, macOS arm64, Go 1.27.2, with the same settings
(`CGO_ENABLED=0 GOMAXPROCS=1`, medians of six 300 ms samples), last on
October 16, 2026.

| Fixture (Apple M1 Pro) | Time per frame | Allocations per frame | Allocated bytes per frame |
|---|---:|---:|---:|
| Table of 100,000 rows with checkboxes, sorted by a `TableView` | 119 µs | 203 | 5,781 |

The frame costs about the same at 1,000 rows (121 µs) and at 1,000,000 (122 µs): it
pays for the rows in view of a 600-pixel window, not for the rows behind
them. It misses the design's target of tens of microseconds. A CPU profile
spends the frame on layout (about 37%), building the rows (23%), painting
(16%) and committing (5%), spread over the engine rather than in the table:
each row is measured, then laid out, as rows may be of any height. A faster
frame needs a faster engine, not a different table.

For 100,000 rows, the `TableView` itself takes:

| Operation | Time | Allocations |
|---|---:|---:|
| `Sync` with nothing changed | 5.1 ns | 0 |
| `Changed` for 1 row, then `Sync` | 95 µs | 0 |
| `Changed` for 10 rows, then `Sync` | 88 µs | 0 |
| `Changed` for 100 rows, then `Sync` | 107 µs | 0 |
| `SetFilter`, then `Sync` (a search typed or cleared, rows sorted) | 109 µs | 0 |
| `Changed` for 1 row, then `Sync`, a filter keeping 5 rows in 7 | 155 µs | 0 |
| `Changed` for 100 rows, then `Sync`, a filter keeping 5 rows in 7 | 186 µs | 0 |
| Sorting all rows again (a click on a header) | 11.4 ms | 0 |

Each is the median of six 300 ms samples. A change of a few rows costs a
pass over the order, so about 0.1 ms whether the rows are 1 or 100 (half
again as much under a filter, as the view keeps the order of all rows too);
sorting all of them again takes a dozen milliseconds, below one frame at
60 Hz.

### Checking rows of a million

`BenchmarkTableCheckedLarge` (in `ui/widget`) works on a widget `DataGridView`
of 1,000,000 rows, all checked, as the user does: a click on a row's checkbox,
a Cmd-click on a row with all rows selected, a right-click on the one row
selected (with a context menu), and the header's "Check all"; and by
`SetChecked`, then a frame. `BenchmarkTableSelectedLateKey` asks for the one
key checked, the last in the order shown.

A selection records the keys that changed (`Selection.TakeChanges`), so a
click hands the table only those, not a copy of a million keys: a plain
click, which replaces the whole set, still copies it. `SetChecked` hands ui
only the keys it changed; the header's checkbox takes the count the table
keeps (`ListState.CheckedCount`) and checks the rows in one call
(`ListState.CheckRows`); the rows chosen for a right-click or Copy come from
the chosen keys (`ListState.RowsOf`); and `Selected` and `Checked` of few
keys place them with an inverse of the view's order (4 MB at a million rows,
built on first use after the order changes) instead of walking the table.
Medians of ten runs, Apple M1 Pro, Go 1.27.2:

| Operation, 1,000,000 rows | Before | After | Bytes before, after |
|---|---:|---:|---:|
| A click on a row's checkbox | 4.02 ms | 0.65 ms | 37.9 MB, 34 KB |
| Cmd-click, all rows selected | 3.93 ms | 0.45 ms | 37.8 MB, 35 KB |
| Right-click, one row selected | 17.8 ms | 0.35 ms | 8.0 MB, 29 KB |
| "Check all" | 157 ms | 65 ms | 45.9 MB, 37.9 MB |
| `Checked()`, one key, the last shown | 4.2 ms | 0.1 µs | 8 B, 24 B |
| `SetChecked` of one row, then a frame | 174 µs | 172 µs | 10.2 KB, 10.3 KB |

"Check all" still copies the checks once for the next frame, as it changes a
million of them.

## Retained memory

A warmed view containing 1,000 labels retained approximately **2.50 MB of
Go heap**, measured over three samples. The measurement warms fonts and
rendering, releases a warm view, forces collection, then measures a second
retained view after ten frames and another collection.

This includes the headless rendering host. It does not measure native window
process memory or GPU resources.

## Measurement scope

Timing depends on the machine, rendering backend, content and activity in
other processes. Small timing differences should be rechecked on several
machines; allocation counts are more stable. These results describe steady
frames and do not compare MyGo with another GUI framework.

<!-- repository-only:start -->

## Run the benchmark fixtures

From the MyGo repository root:

```sh
CGO_ENABLED=0 GOMAXPROCS=1 go test ./ui -run '^$' \
  -bench '^(BenchmarkValueFrame|BenchmarkValueList|BenchmarkDiffSteady|BenchmarkTableFrame100k|BenchmarkTableView.*)$' \
  -benchmem -count=6 -benchtime=300ms
CGO_ENABLED=0 GOMAXPROCS=1 go test ./ui/markdown -run '^$' \
  -bench '^BenchmarkView' -benchmem -count=6 -benchtime=300ms
CGO_ENABLED=0 GOMAXPROCS=1 go test ./ui \
  -run '^TestValueFrameRetainedHeap$' -count=3 -v
```

Keep compilation and other tests out of the timed runs.

<!-- repository-only:end -->
