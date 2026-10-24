// Package richtextbox is the RichTextBox of package widget: styled text over
// ui/richtext. It is a package of its own so that only apps that use it link
// the rich text editor.
package richtextbox

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/egoist/mygo/internal/text"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/egoist/mygo/ui/widget"
)

// Box edits styled text: bold, italic, underline, colors,
// links, headings and lists, over package ui/richtext. Its Document is the
// handle for a toolbar (Document().ToggleBold(), Formatting()).
//
// Offsets of the selection count runes, as richtext's do, newlines
// included.
type Box struct {
	*widget.CustomWidget
	doc         *richtext.Document
	placeholder string
	label       string
	readOnly    bool
	onChange    func()
	onSelection func()
	onLink      func(string)

	// seen is the document version the last build reported on or was told
	// to skip (a setter's), and seenSel the selection it last saw.
	seen    uint64
	seenSel [2]int
	owed    bool // a user's edit a setter came in front of: still to report
}

// New makes an empty rich text box.
func New() *Box {
	r := &Box{doc: richtext.NewDocument(richtext.NewText(""))}
	r.seen = r.doc.Version()
	r.CustomWidget = widget.Custom(r.build)
	return r
}

// Document returns the document the box edits, to read the formats and the
// history (Formatting, CanUndo). Changing it directly is safe from any
// goroutine but does not redraw the window until some other frame comes, as
// richtext says: change it with Edit.
func (r *Box) Document() *richtext.Document { return r.doc }

// Edit runs fn on the document, such as a toolbar's
// d.ToggleBold(), and redraws the windows that show the box, so that the
// change appears and OnChange runs. What it changes counts as the user's for
// OnChange. fn runs with the package's lock held: it must not call
// widget getters or setters.
func (r *Box) Edit(fn func(d *richtext.Document)) *Box {
	widget.Change(func() { fn(r.doc) })
	return r
}

// replace puts text in the document as a program does: OnChange does not
// run for it.
func (r *Box) replace(t richtext.Text) *Box {
	widget.Change(func() {
		if r.doc.Version() != r.seen {
			r.owed = true // an edit nobody reported yet is still the user's
		}
		r.doc.SetText(t)
		r.seen = r.doc.Version()
		a, c := r.doc.Selection()
		r.seenSel = [2]int{a, c} // a setter moves the caret without a report
	})
	return r
}

// SetText replaces the text with plain text, forgetting the history and the
// formats. The setters of the text do not run OnChange.
func (r *Box) SetText(s string) *Box {
	return r.replace(richtext.NewText(s))
}

// Text returns the characters, without formats.
func (r *Box) Text() string { return r.doc.Text().PlainText() }

// SetRich replaces the text with a styled one.
func (r *Box) SetRich(t richtext.Text) *Box { return r.replace(t) }

// Rich returns the styled text, a value that stays as it is.
func (r *Box) Rich() richtext.Text { return r.doc.Text() }

// SetMarkdown replaces the text with Markdown, as a paste of it would.
func (r *Box) SetMarkdown(s string) *Box {
	return r.replace(richtext.ParseMarkdown(s))
}

// Markdown returns the text as Markdown.
func (r *Box) Markdown() string { return r.doc.Text().Markdown() }

// SetMarkup replaces the text with a subset of Pango markup, as ParseMarkup
// reads it: bold, italic, underline, and span with foreground, background
// and weight.
func (r *Box) SetMarkup(s string) *Box {
	return r.replace(markupToRich(widget.ParseMarkup(s)))
}

// Markup returns the text in the subset SetMarkup reads, so that a text of
// that subset round-trips. What the subset cannot say is dropped and its
// text kept: links, headings, lists, fonts and sizes, code and strikethrough.
func (r *Box) Markup() string { return richToMarkup(r.doc.Text()) }

// SetReadOnly shows the text without letting the user change it; it can
// still be selected and copied.
func (r *Box) SetReadOnly(on bool) *Box {
	widget.Change(func() { r.readOnly = on })
	return r
}

// SetPlaceholder sets the text shown while the box is empty.
func (r *Box) SetPlaceholder(s string) *Box {
	widget.Change(func() { r.placeholder = s })
	return r
}

// SetLabel names the box for assistive technology ("Rich text" by default).
func (r *Box) SetLabel(s string) *Box {
	widget.Change(func() { r.label = s })
	return r
}

// OnLink opens the URL of a link the user clicked, instead of the system.
func (r *Box) OnLink(fn func(url string)) *Box {
	widget.Change(func() { r.onLink = fn })
	return r
}

// OnChange runs fn, on the event goroutine, once for each frame in which the
// text or its formats changed other than by SetText, SetRich, SetMarkdown or
// SetMarkup: typing, a paste, an input method's commit (Telex and Unikey's
// backspace and replacement land in one edit), undo and redo, and what a
// toolbar does through Document(), such as ToggleBold.
func (r *Box) OnChange(fn func()) *Box {
	widget.Change(func() { r.onChange = fn })
	return r
}

// OnSelectionChange runs fn, on the event goroutine, after the selection
// moved, by the user or SetSelection.
func (r *Box) OnSelectionChange(fn func()) *Box {
	widget.Change(func() { r.onSelection = fn })
	return r
}

// Selection returns the selection in runes: start and end, equal for a
// caret.
func (r *Box) Selection() (start, end int) {
	a, c := r.doc.Selection()
	return min(a, c), max(a, c)
}

// SelectedText returns the characters selected.
func (r *Box) SelectedText() string {
	a, b := r.Selection()
	rs := []rune(r.doc.Text().String())
	return string(rs[min(a, len(rs)):min(b, len(rs))])
}

// SetSelection selects the runes from start to end, or puts the caret at
// start when they are equal. The ends move out to whole graphemes (a letter
// with its combining marks, an emoji sequence), never splitting one, and
// into the text.
func (r *Box) SetSelection(start, end int) *Box {
	rs := []rune(r.doc.Text().String())
	var b text.Boundaries
	b.Reset(rs)
	offs := b.GraphemeOffsets()
	if start > end {
		start, end = end, start
	}
	if start == end {
		// A caret between a letter and its mark goes before the letter.
		end = snapGrapheme(offs, start, false)
		start = end
	} else {
		start = snapGrapheme(offs, start, false)
		end = snapGrapheme(offs, end, true)
	}
	widget.Change(func() { r.doc.SetSelection(start, end) })
	return r
}

// snapGrapheme moves i to the grapheme boundary at or before it, or at or
// after it with up.
func snapGrapheme(offs []int, i int, up bool) int {
	i = max(0, min(i, offs[len(offs)-1]))
	k := sort.SearchInts(offs, i)
	if k < len(offs) && offs[k] == i {
		return i
	}
	if up {
		return offs[k]
	}
	return offs[k-1]
}

func (r *Box) build(c *ui.Context) ui.Element {
	opts := richtext.EditorOptions{
		Label: r.label, Placeholder: r.placeholder, ReadOnly: r.readOnly, OnLink: r.onLink,
	}
	e := richtext.Editor(c, r.doc, opts)
	if v := r.doc.Version(); v != r.seen || r.owed {
		r.seen, r.owed = v, false
		if h := r.onChange; h != nil {
			widget.Post(h)
		}
	}
	a, caret := r.doc.Selection()
	if sel := [2]int{a, caret}; sel != r.seenSel {
		r.seenSel = sel
		if h := r.onSelection; h != nil {
			widget.Post(h)
		}
	}
	return e
}

// markupToRich converts the spans of ParseMarkup to a text: paragraphs are
// the lines.
func markupToRich(spans []ui.Span) richtext.Text {
	var paras []richtext.Paragraph
	cur := richtext.Paragraph{}
	for _, sp := range spans {
		attrs := richtext.Attributes{Weight: sp.Weight, Italic: sp.Italic, Underline: sp.Underline, Color: sp.Color, Background: sp.Background}
		for i, line := range strings.Split(sp.Text, "\n") {
			if i > 0 {
				paras = append(paras, cur)
				cur = richtext.Paragraph{}
			}
			if line != "" {
				cur.Runs = append(cur.Runs, richtext.Run{Text: line, Attributes: attrs})
			}
		}
	}
	paras = append(paras, cur)
	return richtext.TextOf(paras...)
}

// richToMarkup writes the subset of Pango markup a text can say.
func richToMarkup(t richtext.Text) string {
	var b strings.Builder
	first := true
	for _, p := range t.Paragraphs() {
		if !first {
			b.WriteByte('\n')
		}
		first = false
		for _, run := range p.Runs {
			b.WriteString(runMarkup(run))
		}
	}
	return b.String()
}

var markupEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func runMarkup(run richtext.Run) string {
	a := run.Attributes
	s := markupEscaper.Replace(run.Text)
	var open, closing []string
	if a.Weight == 700 {
		open, closing = append(open, "<b>"), append(closing, "</b>")
	}
	if a.Italic {
		open, closing = append(open, "<i>"), append(closing, "</i>")
	}
	if a.Underline {
		open, closing = append(open, "<u>"), append(closing, "</u>")
	}
	var attrs []string
	if a.Weight != 0 && a.Weight != 700 {
		attrs = append(attrs, fmt.Sprintf("weight='%d'", a.Weight))
	}
	if a.Color.A != 0 {
		attrs = append(attrs, "foreground='"+hexColor(a.Color)+"'")
	}
	if a.Background.A != 0 {
		attrs = append(attrs, "background='"+hexColor(a.Background)+"'")
	}
	if len(attrs) > 0 {
		open, closing = append(open, "<span "+strings.Join(attrs, " ")+">"), append(closing, "</span>")
	}
	slices.Reverse(closing)
	return strings.Join(open, "") + s + strings.Join(closing, "")
}

// hexColor writes #rrggbb, or #rrggbbaa for a translucent color.
func hexColor(c ui.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}
