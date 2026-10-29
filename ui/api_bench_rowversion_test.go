package ui_test

import "github.com/egoist/mygo/ui"

// A RowVersion for the tables of tableFrame100k, which the -rv sub-benchmarks
// of BenchmarkTableFrame100kPhases run with: every row has the version 0 until
// its row changes (bump). It is a file of its own so that the benchmarks build
// with the common file where lists have no RowVersion.
func init() {
	tableRowVersions = func(list *ui.ListState, rows []tableAccount, view *ui.TableView[tableAccount]) func(r int) {
		vers := make([]uint64, len(rows))
		list.RowVersion = func(i int) uint64 { return vers[view.Row(i)] }
		return func(r int) { vers[r]++ }
	}
}
