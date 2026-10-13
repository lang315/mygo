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
(`CGO_ENABLED=0 GOMAXPROCS=1`, medians of six 300 ms samples).

| Fixture (Apple M1 Pro) | Time per frame | Allocations per frame | Allocated bytes per frame |
|---|---:|---:|---:|
| Table of 100,000 rows with checkboxes, sorted by a `TableView` | 116 µs | 203 | 5,765 |

For 100,000 rows, the `TableView` itself takes:

| Operation | Time | Allocations |
|---|---:|---:|
| `Sync` with nothing changed | 3.6 ns | 0 |
| `Changed` for 1 row, then `Sync` | 97 µs | 0 |
| `Changed` for 10 rows, then `Sync` | 89 µs | 0 |
| `Changed` for 100 rows, then `Sync` | 106 µs | 0 |
| Sorting all rows again (a click on a header) | 11.7 ms | 0 |

Each is the median of six 300 ms samples. A change of a few rows costs a
pass over the order, so about 0.1 ms whether the rows are 1 or 100; sorting
all of them again takes a dozen milliseconds, below one frame at 60 Hz.

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
CGO_ENABLED=0 GOMAXPROCS=1 go test ./ui \
  -run '^TestValueFrameRetainedHeap$' -count=3 -v
```

Keep compilation and other tests out of the timed runs.

<!-- repository-only:end -->
