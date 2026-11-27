// Package export writes the cells of a table, as DataGridView's ShownCells
// gives them, to a file: CSV, or an Excel workbook (XLSX). It is a package
// of its own so that an app that does not export links neither writer.
//
//	header, rows := grid.ShownCells()
//	err := export.XLSX(f, header, rows)
package export

import (
	"archive/zip"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// CSV writes header, then rows, as RFC 4180 CSV: fields apart by commas,
// quoted when they hold a comma, a quote (doubled) or a line break, each
// record and each line break in a field ended by CRLF. A lone CR in a
// field is dropped, as encoding/csv's UseCRLF does. It writes no byte
// order mark.
func CSV(w io.Writer, header []string, rows [][]string) error {
	c := csv.NewWriter(w)
	c.UseCRLF = true
	c.Write(header) // buffered: an error of w comes back from WriteAll
	return c.WriteAll(rows)
}

// Excel's bounds: the columns and the rows of a sheet, and the UTF-16
// units of a cell's text.
const (
	maxColumns = 16384
	maxRows    = 1 << 20
	maxCell    = 32767
)

// XLSX writes header, then rows, as an Excel workbook of one sheet,
// Sheet1, whose cells are text (inline strings), the header's bold. XML
// cannot hold control characters but tab and line breaks: they are written
// as U+FFFD. Past Excel's bounds (16,384 columns, 1,048,576 rows with the
// header, 32,767 characters in a cell) it returns an error and writes
// nothing.
func XLSX(w io.Writer, header []string, rows [][]string) error {
	if err := fits(header, rows); err != nil {
		return err
	}
	// The zip writer buffers, and keeps the first error of w: Create returns
	// it once the part before is written out, and Close at the end, so the
	// writes into a part need no check of their own.
	z := zip.NewWriter(w)
	for _, p := range [...]struct {
		name  string
		write func(io.Writer)
	}{
		{"[Content_Types].xml", text(contentTypes)},
		{"xl/worksheets/sheet1.xml", func(f io.Writer) { writeSheet(f, header, rows) }},
		{"_rels/.rels", text(rootRels)},
		{"xl/workbook.xml", text(workbook)},
		{"xl/_rels/workbook.xml.rels", text(workbookRels)},
		{"xl/styles.xml", text(styles)},
	} {
		f, err := z.Create(p.name)
		if err != nil {
			return err
		}
		p.write(f)
	}
	return z.Close()
}

// text writes the XML document body.
func text(body string) func(io.Writer) {
	return func(f io.Writer) { io.WriteString(f, xml.Header+body) }
}

// fits returns an error when the cells are past Excel's bounds.
func fits(header []string, rows [][]string) error {
	if len(rows)+1 > maxRows {
		return fmt.Errorf("export: %d rows and the header are more than the %d rows of an Excel sheet", len(rows), maxRows)
	}
	for i := -1; i < len(rows); i++ {
		r := header
		if i >= 0 {
			r = rows[i]
		}
		if len(r) > maxColumns {
			return fmt.Errorf("export: row %d has %d cells, more than the %d columns of an Excel sheet", i+2, len(r), maxColumns)
		}
		for j, s := range r {
			if len(s) <= maxCell { // a byte is a unit at most
				continue
			}
			n := 0
			for _, c := range s {
				n++
				if c > 0xFFFF {
					n++ // a surrogate pair
				}
			}
			if n > maxCell {
				return fmt.Errorf("export: cell %s%d holds %d characters, more than the %d of an Excel cell", column(j), i+2, n, maxCell)
			}
		}
	}
	return nil
}

// writeSheet writes the worksheet part: the header in row 1, in style 1
// (bold), then the rows.
func writeSheet(w io.Writer, header []string, rows [][]string) {
	io.WriteString(w, xml.Header+`<worksheet xmlns="`+nsMain+`"><sheetData>`)
	writeRow(w, 1, header, ` s="1"`)
	for i, r := range rows {
		writeRow(w, i+2, r, "")
	}
	io.WriteString(w, `</sheetData></worksheet>`)
}

// writeRow writes row n (from 1) of cells in style.
func writeRow(w io.Writer, n int, cells []string, style string) {
	num := strconv.Itoa(n)
	io.WriteString(w, `<row r="`+num+`">`)
	for j, s := range cells {
		io.WriteString(w, `<c r="`+column(j)+num+`" t="inlineStr"`+style+`><is><t xml:space="preserve">`)
		xml.EscapeText(w, []byte(s))
		io.WriteString(w, `</t></is></c>`)
	}
	io.WriteString(w, `</row>`)
}

// column returns the letters of column i, from 0: A to Z, then AA.
func column(i int) string {
	var b [3]byte // XFD, the last, has three
	k := len(b)
	for i++; i > 0; i = (i - 1) / 26 {
		k--
		b[k] = byte('A' + (i-1)%26)
	}
	return string(b[k:])
}

const (
	nsMain = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsRel  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsPkg  = "http://schemas.openxmlformats.org/package/2006/relationships"
	ctBase = "application/vnd.openxmlformats-officedocument.spreadsheetml."

	contentTypes = `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="` + ctBase + `sheet.main+xml"/>` +
		`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="` + ctBase + `worksheet+xml"/>` +
		`<Override PartName="/xl/styles.xml" ContentType="` + ctBase + `styles+xml"/>` +
		`</Types>`
	rootRels = `<Relationships xmlns="` + nsPkg + `">` +
		`<Relationship Id="rId1" Type="` + nsRel + `/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`
	workbook = `<workbook xmlns="` + nsMain + `" xmlns:r="` + nsRel + `">` +
		`<sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets>` +
		`</workbook>`
	workbookRels = `<Relationships xmlns="` + nsPkg + `">` +
		`<Relationship Id="rId1" Type="` + nsRel + `/worksheet" Target="worksheets/sheet1.xml"/>` +
		`<Relationship Id="rId2" Type="` + nsRel + `/styles" Target="styles.xml"/>` +
		`</Relationships>`
	// styles is the least style sheet Excel opens without a repair: a
	// font, and a bold one; the two fills every sheet has; a border; a
	// cell style format; and two cell formats, the plain and the bold.
	styles = `<styleSheet xmlns="` + nsMain + `">` +
		`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
		`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
		`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
		`<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
		`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs>` +
		`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>` +
		`</styleSheet>`
)
