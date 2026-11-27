package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

// exportTo has the save dialog answer path.
func exportTo(t *testing.T, path string) {
	saved := saveFile
	saveFile = func(*widget.Form, string, string, ...widget.Filter) string { return path }
	t.Cleanup(func() { saveFile = saved })
}

// The Export CSV button writes the table's 100,000 rows, the header first,
// its columns as shown: not the index column or the check boxes.
func TestTourTableExportCSV(t *testing.T) {
	tt := tourTableTab(t)
	path := filepath.Join(t.TempDir(), "accounts.csv")
	exportTo(t, path)
	if err := tt.Click("Export CSV"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(b), "\r\n")
	if len(lines) != 100_002 || lines[0] != "ID,Name,Status,Score" || lines[1] != "0,acct000000,ok,0" || lines[100_001] != "" {
		t.Fatalf("%d lines, first %q", len(lines), lines[:min(3, len(lines))])
	}
	if !tt.HasText("Exported 100000 rows to accounts.csv") {
		t.Fatalf("no word of the export: %q", tt.Texts())
	}
}

// The Export XLSX button writes a workbook whose sheet starts with the
// header.
func TestTourTableExportXLSX(t *testing.T) {
	tt := tourTableTab(t)
	path := filepath.Join(t.TempDir(), "accounts.xlsx")
	exportTo(t, path)
	if err := tt.Click("Export XLSX"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	f, err := z.Open("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	const head = `<row r="1"><c r="A1" t="inlineStr" s="1"><is><t xml:space="preserve">ID</t></is></c>`
	if s := string(b); !strings.Contains(s, head) || !strings.Contains(s, `<c r="B100001" t="inlineStr"><is><t xml:space="preserve">acct099999</t>`) {
		t.Fatalf("sheet1.xml starts %q", s[:min(400, len(s))])
	}
	if !tt.HasText("Exported 100000 rows to accounts.xlsx") {
		t.Fatalf("no word of the export: %q", tt.Texts())
	}
}

// A file that cannot be made says so.
func TestTourTableExportFails(t *testing.T) {
	tt := tourTableTab(t)
	exportTo(t, filepath.Join(t.TempDir(), "no such folder", "accounts.csv"))
	if err := tt.Click("Export CSV"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "Export failed: open ") { // the OS words the rest
			return
		}
	}
	t.Fatalf("no word of the failure: %q", tt.Texts())
}

// A canceled save dialog writes nothing.
func TestTourTableExportCanceled(t *testing.T) {
	tt := tourTableTab(t)
	exportTo(t, "")
	if err := tt.Click("Export CSV"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !tt.HasText("Double-click a row") {
		t.Fatalf("a canceled export changed the status: %q", tt.Texts())
	}
}
