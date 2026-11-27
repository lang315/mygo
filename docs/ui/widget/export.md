# Export

Package `ui/widget/export` writes the cells of a table to a file: CSV, or an
Excel workbook (XLSX). A `DataGridView` gives them with `ShownCells`: the
columns shown, in the header's order, and the rows as shown, filtered,
sorted and grouped (see [DataGridView](../widget.md#datagridview)).

```go
import "github.com/egoist/mygo/ui/widget/export"

widget.Button("Export").OnClick(func() {
	path := widget.SaveFile(w, "Export", "accounts.xlsx",
		widget.Filter{Name: "Excel workbook", Extensions: []string{"xlsx"}})
	if path == "" {
		return // canceled
	}
	f, err := os.Create(path)
	if err != nil {
		widget.Alert(w, widget.Error, "Export", err.Error())
		return
	}
	header, rows := grid.ShownCells()
	err = export.XLSX(f, header, rows)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		widget.Alert(w, widget.Error, "Export", err.Error())
	}
})
```

- `export.CSV(w, header, rows)` writes RFC 4180 CSV with `encoding/csv`:
  the header first, fields apart by commas and quoted when they hold a comma,
  a quote (doubled) or a line break, each record ended by CRLF. A line
  break inside a field is written as CRLF too, and a lone CR in one is
  dropped. It writes no byte order mark: Excel on Windows may read
  Vietnamese text in a CSV in the system's code page; use XLSX there.
- `export.XLSX(w, header, rows)` writes a workbook of one sheet, `Sheet1`,
  with `archive/zip` and `encoding/xml` alone: the header in row 1, bold,
  and every cell as text (an inline string), so numbers stay as shown. It
  has no other styles, no formulas and no other sheets. XML cannot hold
  control characters but tab and line breaks: they are written as U+FFFD.
  Past Excel's bounds, 16,384 columns, 1,048,576 rows with the header, or
  32,767 characters (UTF-16 units) in a cell, it returns an error before it
  writes anything.
- Both return the first error of the writer, which they do not close; a
  file cut short by an error is not a workbook.
- The package is its own so that an app that does not import it links
  neither `encoding/csv` nor `archive/zip` (`encoding/xml` every app links
  already). `ShownCells` costs a `DataGridView` app nothing more.
- Both run on the caller's goroutine. `ShownCells` holds the package's lock
  while it reads every row shown, and runs the columns' `Text` under it: call
  it from an event handler, never from inside a column func.
