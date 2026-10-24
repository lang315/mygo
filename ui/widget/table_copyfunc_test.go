package widget

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// copyTable is a table of three accounts with a CopyFunc that copies their
// ids, as fhtool copies its UIDs; got collects what the function was given.
func copyTable(got *[][]int) *DataGridViewWidget[acct, int] {
	return testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100}).
		CopyFunc(func(ids []int) string {
			*got = append(*got, slices.Clone(ids))
			var s []string
			for _, id := range ids {
				s = append(s, "uid"+string(rune('0'+id)))
			}
			return strings.Join(s, "\n")
		})
}

// Ctrl+C and the Edit menu's Copy give the same string: the function's.
func TestCopyFuncServesCtrlCAndTheEditMenuAlike(t *testing.T) {
	var got [][]int
	tb := copyTable(&got)
	tt := tester(t, tb)
	click(t, tt, "Name") // ann, bob, cat
	click(t, tt, "cat")
	if err := tt.ClickWith(ui.Cmd, "ann"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyC)
	byKey := tt.Clipboard()
	tt.SetClipboard("")
	tt.Command("copy")
	byMenu := tt.Clipboard()
	if byKey != "uid2\nuid1" {
		t.Fatalf("Ctrl+C copied %q, want the ids of ann and cat in the order shown", byKey)
	}
	if byMenu != byKey {
		t.Fatalf("Edit>Copy copied %q, Ctrl+C %q: they must agree", byMenu, byKey)
	}
	if want := [][]int{{2, 1}, {2, 1}}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("the function was given %v, want %v", got, want)
	}
}

// Without CopyFunc, the table copies its cells, as before.
func TestWithoutCopyFuncTheTableCopiesItsCells(t *testing.T) {
	tb := testTable(acct{1, "cat", 10}, acct{2, "ann", 9})
	tt := tester(t, tb)
	click(t, tt, "cat")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "cat\t10" {
		t.Fatalf("copied %q", got)
	}
}

// nil gives the cells back, and the function may call the table's getters:
// it runs off the package lock.
func TestCopyFuncNilRestoresTheCellsAndTheFuncMayReadTheTable(t *testing.T) {
	var tb *DataGridViewWidget[acct, int]
	tb = testTable(acct{1, "cat", 10}).CopyFunc(func(ids []int) string {
		return "n=" + string(rune('0'+tb.CountSelected())) // takes the package lock
	})
	tt := tester(t, tb)
	click(t, tt, "cat")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "n=1" {
		t.Fatalf("copied %q", got)
	}
	tb.CopyFunc(nil)
	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "cat\t10" {
		t.Fatalf("after CopyFunc(nil) copied %q", got)
	}
}
