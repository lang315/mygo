package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestCSVQuotes(t *testing.T) {
	var b bytes.Buffer
	err := CSV(&b, []string{"Name", "Note, with a comma"}, [][]string{
		{"plain", `say "hi"`},
		{"two\nlines", "a\r\nb"},
		{"", " leading space"},
		{"Tiếng Việt", `\.`},
	})
	if err != nil {
		t.Fatal(err)
	}
	// RFC 4180: CRLF ends each record, and inside a quoted field a newline
	// is CRLF too.
	want := "Name,\"Note, with a comma\"\r\n" +
		"plain,\"say \"\"hi\"\"\"\r\n" +
		"\"two\r\nlines\",\"a\r\nb\"\r\n" +
		",\" leading space\"\r\n" +
		"Tiếng Việt,\"\\.\"\r\n"
	if got := b.String(); got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestCSVHeaderOnlyAndRaggedRows(t *testing.T) {
	var b bytes.Buffer
	if err := CSV(&b, []string{"A", "B"}, nil); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "A,B\r\n" {
		t.Fatalf("header only: %q", got)
	}
	b.Reset()
	if err := CSV(&b, []string{"A"}, [][]string{{"1", "2"}, {}}); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "A\r\n1,2\r\n\r\n" {
		t.Fatalf("rows of other lengths: %q", got)
	}
}

// failAfter fails every write once n bytes are written.
type failAfter struct{ n int }

var errFull = errors.New("disk full")

func (f *failAfter) Write(p []byte) (int, error) {
	if len(p) > f.n {
		k := f.n
		f.n = 0
		return k, errFull
	}
	f.n -= len(p)
	return len(p), nil
}

// A writer that fails midway fails the export with its error.
func TestExportWriteFails(t *testing.T) {
	rows := make([][]string, 5000)
	for i := range rows {
		rows[i] = []string{"some text", "more text"}
	}
	if err := CSV(&failAfter{n: 100}, []string{"A", "B"}, rows); !errors.Is(err, errFull) {
		t.Errorf("CSV: %v", err)
	}
	if err := CSV(&failAfter{n: 0}, []string{"A", "B"}, nil); !errors.Is(err, errFull) {
		t.Errorf("CSV of the header alone: %v", err)
	}
	if err := XLSX(&failAfter{n: 100}, []string{"A", "B"}, rows); !errors.Is(err, errFull) {
		t.Errorf("XLSX: %v", err)
	}
	if err := XLSX(&failAfter{n: 0}, []string{"A"}, nil); !errors.Is(err, errFull) {
		t.Errorf("XLSX of the header alone: %v", err)
	}
}

// parts opens an XLSX file's parts by name.
func parts(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = b
	}
	return out
}

func xlsx(t *testing.T, header []string, rows [][]string) map[string][]byte {
	t.Helper()
	var b bytes.Buffer
	if err := XLSX(&b, header, rows); err != nil {
		t.Fatal(err)
	}
	return parts(t, b.Bytes())
}

// wellFormed parses an XML document to its end, and returns its root.
func wellFormed(t *testing.T, name string, data []byte) xml.Name {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	var root xml.Name
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, data)
		}
		if se, ok := tok.(xml.StartElement); ok && root.Local == "" {
			root = se.Name
		}
	}
	return root
}

const (
	wantMain = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	wantPkg  = "http://schemas.openxmlformats.org/package/2006/relationships"
	wantCT   = "http://schemas.openxmlformats.org/package/2006/content-types"
	wantRel  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
)

// The package has the parts of ECMA-376's least workbook, each XML of the
// root and namespace its part is, and the content types and relationships
// tie them together.
func TestXLSXParts(t *testing.T) {
	p := xlsx(t, []string{"A"}, [][]string{{"1"}})
	roots := map[string]xml.Name{
		"[Content_Types].xml":        {Space: wantCT, Local: "Types"},
		"_rels/.rels":                {Space: wantPkg, Local: "Relationships"},
		"xl/workbook.xml":            {Space: wantMain, Local: "workbook"},
		"xl/_rels/workbook.xml.rels": {Space: wantPkg, Local: "Relationships"},
		"xl/worksheets/sheet1.xml":   {Space: wantMain, Local: "worksheet"},
		"xl/styles.xml":              {Space: wantMain, Local: "styleSheet"},
	}
	if len(p) != len(roots) {
		var names []string
		for n := range p {
			names = append(names, n)
		}
		t.Fatalf("parts %q", names)
	}
	for name, want := range roots {
		data, ok := p[name]
		if !ok {
			t.Fatalf("no part %s", name)
		}
		if got := wellFormed(t, name, data); got != want {
			t.Errorf("%s: root %v, want %v", name, got, want)
		}
	}

	var types struct {
		Defaults []struct {
			Extension   string `xml:",attr"`
			ContentType string `xml:",attr"`
		} `xml:"Default"`
		Overrides []struct {
			PartName    string `xml:",attr"`
			ContentType string `xml:",attr"`
		} `xml:"Override"`
	}
	if err := xml.Unmarshal(p["[Content_Types].xml"], &types); err != nil {
		t.Fatal(err)
	}
	ct := map[string]string{}
	for _, d := range types.Defaults {
		ct["."+d.Extension] = d.ContentType
	}
	for _, o := range types.Overrides {
		ct[o.PartName] = o.ContentType
	}
	for k, v := range map[string]string{
		".rels":                     "application/vnd.openxmlformats-package.relationships+xml",
		".xml":                      "application/xml",
		"/xl/workbook.xml":          "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml",
		"/xl/worksheets/sheet1.xml": "application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml",
		"/xl/styles.xml":            "application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml",
	} {
		if ct[k] != v {
			t.Errorf("content type of %s: %q, want %q", k, ct[k], v)
		}
	}

	type rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Type   string `xml:",attr"`
			Target string `xml:",attr"`
		} `xml:"Relationship"`
	}
	var root, wb rels
	if err := xml.Unmarshal(p["_rels/.rels"], &root); err != nil {
		t.Fatal(err)
	}
	if len(root.Rel) != 1 || root.Rel[0].Type != wantRel+"/officeDocument" || root.Rel[0].Target != "xl/workbook.xml" {
		t.Errorf("_rels/.rels: %+v", root.Rel)
	}
	if err := xml.Unmarshal(p["xl/_rels/workbook.xml.rels"], &wb); err != nil {
		t.Fatal(err)
	}
	byID := map[string][2]string{}
	for _, r := range wb.Rel {
		byID[r.ID] = [2]string{r.Type, r.Target}
	}
	var book struct {
		Sheets []struct {
			Name    string `xml:"name,attr"`
			SheetID string `xml:"sheetId,attr"`
			RID     string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(p["xl/workbook.xml"], &book); err != nil {
		t.Fatal(err)
	}
	if len(book.Sheets) != 1 || book.Sheets[0].Name != "Sheet1" || book.Sheets[0].SheetID != "1" {
		t.Fatalf("sheets %+v", book.Sheets)
	}
	if got := byID[book.Sheets[0].RID]; got != [2]string{wantRel + "/worksheet", "worksheets/sheet1.xml"} {
		t.Errorf("the sheet's relationship %q: %q", book.Sheets[0].RID, got)
	}
	styles := false
	for _, r := range wb.Rel {
		styles = styles || r.Type == wantRel+"/styles" && r.Target == "styles.xml"
	}
	if !styles || len(wb.Rel) != 2 {
		t.Errorf("the workbook's relationships %+v", wb.Rel)
	}
}

// sheet is what a worksheet part holds of its cells.
type sheet struct {
	Rows []struct {
		R     string `xml:"r,attr"`
		Cells []struct {
			R    string `xml:"r,attr"`
			T    string `xml:"t,attr"`
			S    string `xml:"s,attr"`
			Text string `xml:"is>t"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func readSheet(t *testing.T, p map[string][]byte) sheet {
	t.Helper()
	var s sheet
	if err := xml.Unmarshal(p["xl/worksheets/sheet1.xml"], &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// bold reports whether the cell format index s has a bold font.
func bold(t *testing.T, styles []byte, s string) bool {
	t.Helper()
	var st struct {
		Fonts []struct {
			B *struct{} `xml:"b"`
		} `xml:"fonts>font"`
		Fills []struct {
			Pattern struct {
				Type string `xml:"patternType,attr"`
			} `xml:"patternFill"`
		} `xml:"fills>fill"`
		Borders []struct{} `xml:"borders>border"`
		XFs     []struct {
			FontID int `xml:"fontId,attr"`
		} `xml:"cellXfs>xf"`
		StyleXFs []struct{} `xml:"cellStyleXfs>xf"`
	}
	if err := xml.Unmarshal(styles, &st); err != nil {
		t.Fatal(err)
	}
	// ECMA-376's least style sheet, which Excel asks for: two fills, the
	// second gray125, a border, a cell style format.
	if len(st.Fills) < 2 || st.Fills[0].Pattern.Type != "none" || st.Fills[1].Pattern.Type != "gray125" || len(st.Borders) < 1 || len(st.StyleXFs) < 1 {
		t.Fatalf("styles: %s", styles)
	}
	i := 0 // no s is the first format
	if s != "" {
		var err error
		if i, err = strconv.Atoi(s); err != nil {
			t.Fatal(err)
		}
	}
	if i >= len(st.XFs) || st.XFs[i].FontID >= len(st.Fonts) {
		t.Fatalf("style %q is none of %d formats", s, len(st.XFs))
	}
	return st.Fonts[st.XFs[i].FontID].B != nil
}

func TestXLSXCells(t *testing.T) {
	header := make([]string, 28)
	for i := range header {
		header[i] = "h" + string(rune('a'+i%26))
	}
	header[1] = `<b> & "q" 'a'`
	header[2] = " spaced "
	p := xlsx(t, header, [][]string{{"x", "", "y"}})
	s := readSheet(t, p)
	if len(s.Rows) != 2 || s.Rows[0].R != "1" || s.Rows[1].R != "2" {
		t.Fatalf("rows %+v", s.Rows)
	}
	refs := []string{"A1", "B1", "C1"}
	h := s.Rows[0].Cells
	if len(h) != 28 {
		t.Fatalf("%d header cells", len(h))
	}
	for i, want := range refs {
		if h[i].R != want {
			t.Errorf("cell %d at %q, want %q", i, h[i].R, want)
		}
	}
	if h[25].R != "Z1" || h[26].R != "AA1" || h[27].R != "AB1" {
		t.Errorf("cells 25-27 at %q %q %q", h[25].R, h[26].R, h[27].R)
	}
	for i, c := range h {
		if c.T != "inlineStr" {
			t.Errorf("header cell %s of type %q", c.R, c.T)
		}
		if c.Text != header[i] {
			t.Errorf("header cell %s: %q, want %q", c.R, c.Text, header[i])
		}
		if !bold(t, p["xl/styles.xml"], c.S) {
			t.Errorf("header cell %s is not bold (style %q)", c.R, c.S)
		}
	}
	// The escaping is the XML's, not a reading of it.
	raw := string(p["xl/worksheets/sheet1.xml"])
	if !strings.Contains(raw, `<c r="B1" t="inlineStr" s="1"><is><t xml:space="preserve">&lt;b&gt; &amp; &#34;q&#34; &#39;a&#39;</t></is></c>`) {
		t.Errorf("B1 is not escaped: %s", raw)
	}
	if !strings.Contains(raw, `<t xml:space="preserve"> spaced </t>`) {
		t.Errorf("the spaces around C1 are not kept: %s", raw)
	}
	body := s.Rows[1].Cells
	if len(body) != 3 || body[0].R != "A2" || body[1].R != "B2" || body[2].R != "C2" {
		t.Fatalf("row 2: %+v", body)
	}
	for _, c := range body {
		if c.T != "inlineStr" || bold(t, p["xl/styles.xml"], c.S) {
			t.Errorf("cell %s: type %q, bold", c.R, c.T)
		}
	}
	if body[0].Text != "x" || body[1].Text != "" || body[2].Text != "y" {
		t.Errorf("row 2: %+v", body)
	}
}

// Column letters at the bounds of one, two and three letters.
func TestXLSXColumnRefs(t *testing.T) {
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 51: "AZ", 52: "BA", 701: "ZZ", 702: "AAA", 16383: "XFD"} {
		if got := column(i); got != want {
			t.Errorf("column %d: %q, want %q", i, got, want)
		}
	}
}

func TestXLSXRoundTrip(t *testing.T) {
	header := []string{"Tên", "Ghi chú", "Số"}
	rows := [][]string{
		{"Nguyễn Văn A", "dòng một\ndòng hai", "1.5"},
		{"tab\there", "  two spaces  ", "<&>"},
		{"", "", ""},
		{"only one"},
	}
	s := readSheet(t, xlsx(t, header, rows))
	var got [][]string
	for _, r := range s.Rows {
		var cells []string
		for _, c := range r.Cells {
			cells = append(cells, c.Text)
		}
		got = append(got, cells)
	}
	want := append([][]string{header}, rows...)
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Fatalf("row %d: %q, want %q", i+1, got[i], want[i])
		}
	}
}

// XML cannot hold control characters but tab and newlines: they come
// back as U+FFFD, and the file stays well formed.
func TestXLSXControlCharacters(t *testing.T) {
	p := xlsx(t, []string{"a\x01b\x1fc"}, nil)
	wellFormed(t, "sheet1.xml", p["xl/worksheets/sheet1.xml"])
	if got := readSheet(t, p).Rows[0].Cells[0].Text; got != "a\uFFFDb\uFFFDc" {
		t.Fatalf("%q", got)
	}
}

func TestXLSXNoRows(t *testing.T) {
	s := readSheet(t, xlsx(t, []string{"A", "B"}, nil))
	if len(s.Rows) != 1 || len(s.Rows[0].Cells) != 2 {
		t.Fatalf("%+v", s.Rows)
	}
	p := xlsx(t, nil, nil)
	wellFormed(t, "sheet1.xml", p["xl/worksheets/sheet1.xml"])
	if s := readSheet(t, p); len(s.Rows) != 1 || len(s.Rows[0].Cells) != 0 {
		t.Fatalf("no columns: %+v", s.Rows)
	}
}

// Past Excel's bounds, XLSX refuses before it writes a byte; at them, it
// writes.
func TestXLSXLimits(t *testing.T) {
	try := func(header []string, rows [][]string) (error, int) {
		var n countWriter
		err := XLSX(&n, header, rows)
		return err, int(n)
	}
	if err, _ := try(make([]string, 16384), nil); err != nil {
		t.Errorf("16384 columns: %v", err)
	}
	if err, n := try(make([]string, 16385), nil); err == nil || n != 0 {
		t.Errorf("16385 columns: %v, %d bytes written", err, n)
	}
	if err, n := try([]string{"A"}, [][]string{make([]string, 16385)}); err == nil || n != 0 {
		t.Errorf("a row of 16385 cells: %v, %d bytes written", err, n)
	}
	if err, _ := try([]string{"A"}, make([][]string, 1<<20-1)); err != nil {
		t.Errorf("1048576 rows with the header: %v", err)
	}
	if err, n := try([]string{"A"}, make([][]string, 1<<20)); err == nil || n != 0 {
		t.Errorf("1048577 rows with the header: %v, %d bytes written", err, n)
	}
	long := strings.Repeat("a", 32767)
	if err, _ := try([]string{long}, [][]string{{long}}); err != nil {
		t.Errorf("a cell of 32767 characters: %v", err)
	}
	if err, n := try([]string{"A"}, [][]string{{"b", long + "a"}}); err == nil || n != 0 {
		t.Errorf("a cell of 32768 characters: %v, %d bytes written", err, n)
	}
	// Excel counts UTF-16 units: an emoji is two.
	if err, n := try([]string{strings.Repeat("😀", 16384)}, nil); err == nil || n != 0 {
		t.Errorf("a cell of 32768 UTF-16 units: %v, %d bytes written", err, n)
	}
	if err, _ := try([]string{strings.Repeat("ệ", 32767)}, nil); err != nil {
		t.Errorf("a cell of 32767 characters of several bytes: %v", err)
	}
}

type countWriter int

func (c *countWriter) Write(p []byte) (int, error) { *c += countWriter(len(p)); return len(p), nil }

// A small app with a DataGridView that does not import this package links
// neither encoding/csv nor archive/zip, whether it binds by reflection
// (which keeps every exported method of the grid, ShownCells too) or with
// mygo_static; one that imports it links both. encoding/xml is left out
// of the check: every grid app links it already, through internal/darwin
// and internal/svg.
func TestExportNotLinkedWhenUnused(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go command: ", err)
	}
	dir := t.TempDir()
	syms := func(app string, tags ...string) string {
		t.Helper()
		out := filepath.Join(dir, app+strings.Join(tags, ""))
		args := append([]string{"build", "-trimpath", "-o", out}, tags...)
		cmd := exec.Command("go", append(args, "./testdata/"+app)...)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", app, err, b)
		}
		b, err := exec.Command("go", "tool", "nm", out).CombinedOutput()
		if err != nil {
			t.Fatalf("go tool nm %s: %v\n%s", app, err, b)
		}
		return string(b)
	}
	for _, tags := range [][]string{nil, {"-tags", "mygo_static"}} {
		nm := syms("nouse", tags...)
		for _, pkg := range []string{" encoding/csv.", " archive/zip."} {
			if strings.Contains(nm, pkg) {
				t.Errorf("tags %q: an app that does not import ui/widget/export links%s", tags, pkg)
			}
		}
		if tags == nil && !strings.Contains(nm, ").ShownCells") {
			t.Errorf("without mygo_static the app does not keep ShownCells: the check proves nothing")
		}
		nm = syms("use", tags...)
		for _, pkg := range []string{" encoding/csv.", " archive/zip."} {
			if !strings.Contains(nm, pkg) {
				t.Errorf("tags %q: an app that exports does not link%s: the check proves nothing", tags, pkg)
			}
		}
	}
}
