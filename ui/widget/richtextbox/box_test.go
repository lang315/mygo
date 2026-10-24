package richtextbox

import (
	"github.com/egoist/mygo"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/egoist/mygo/ui/widget"
)

func TestRichTextBoxShowsAndTypesText(t *testing.T) {
	r := New().SetText("Hello")
	tt := widget.NewTester(t, r, 500, 300)
	box := tt.Find(widget.ByRole(ui.RoleTextField))
	if box.Value() != "Hello" {
		t.Fatalf("the box shows %q", box.Value())
	}
	box.Focus().Key(ui.Cmd, ui.KeyEnd)
	box.Type(" world")
	if r.Text() != "Hello world" {
		t.Fatalf("Text %q", r.Text())
	}
}

func TestRichTextBoxOnChangeIsTheUsersOnce(t *testing.T) {
	changed := make(chan struct{}, 8)
	r := New().OnChange(func() { changed <- struct{}{} })
	tt := widget.NewTester(t, r, 500, 300)
	r.SetText("by the program")
	tt.Settle()
	select {
	case <-changed:
		t.Fatal("SetText ran OnChange")
	default:
	}
	tt.Find(widget.ByRole(ui.RoleTextField)).Type("x")
	wait.For(t, "the handler to run", changed)
	tt.Frame()
	tt.Settle()
	select {
	case <-changed:
		t.Fatal("one edit reported twice")
	default:
	}
}

func TestRichTextBoxReadOnlyKeepsTheText(t *testing.T) {
	r := New().SetText("fixed").SetReadOnly(true)
	tt := widget.NewTester(t, r, 500, 300)
	tt.Find(widget.ByRole(ui.RoleTextField)).Focus().Key(ui.Cmd, ui.KeyEnd).Type("x")
	if r.Text() != "fixed" {
		t.Fatalf("Text %q: a read-only box took typing", r.Text())
	}
}

func TestRichTextBoxMarkupRoundTrip(t *testing.T) {
	for _, in := range []string{
		"plain",
		"a <b>bold</b> and <i>italic</i> and <u>under</u>",
		"<span foreground='#ff0000'>red</span> and <span background='#00ff00'>green</span>",
		"two\nlines with <b>bold</b>",
		"1 &lt; 2 &amp; 3 &gt; 2",
		"<b><i>both</i></b> then <span weight='300'>light</span>",
	} {
		r := New().SetMarkup(in)
		out := r.Markup()
		again := New().SetMarkup(out)
		if again.Markup() != out || again.Text() != r.Text() {
			t.Errorf("markup %q -> %q -> %q (text %q vs %q)", in, out, again.Markup(), r.Text(), again.Text())
		}
	}
	// Markup of the subset comes back as it went in.
	for _, in := range []string{
		"a <b>bold</b> text",
		"<i>i</i> <u>u</u>",
		"<span foreground='#ff0000'>red</span>",
		"<span background='#00ff00'>bg</span>",
		"<span weight='300'>light</span>",
		"x &lt; y &amp; z",
		"two\nlines",
	} {
		if got := New().SetMarkup(in).Markup(); got != in {
			t.Errorf("Markup() = %q, want %q", got, in)
		}
	}
	r := New().SetMarkup("a <b>bold</b> text")
	if r.Text() != "a bold text" {
		t.Fatalf("Text %q", r.Text())
	}
	if m := r.Markup(); m != "a <b>bold</b> text" {
		t.Fatalf("Markup %q", m)
	}
}

func TestRichTextBoxMarkupDropsWhatItCannotSay(t *testing.T) {
	r := New().SetMarkdown("# Title\n\nsome [link](https://example.com)")
	m := r.Markup()
	if strings.Contains(m, "#") || strings.Contains(m, "http") || !strings.Contains(m, "Title") || !strings.Contains(m, "link") {
		t.Fatalf("Markup %q: headings and links are dropped, their text stays", m)
	}
}

func TestRichTextBoxMarkdownAndRich(t *testing.T) {
	r := New().SetMarkdown("**bold** text")
	if r.Text() != "bold text" || !strings.Contains(r.Markdown(), "**bold**") {
		t.Fatalf("Text %q, Markdown %q", r.Text(), r.Markdown())
	}
	r2 := New().SetRich(richtext.NewText("rich"))
	if r2.Text() != "rich" || r2.Rich().String() != "rich" {
		t.Fatalf("Text %q", r2.Text())
	}
}

func TestRichTextBoxSelection(t *testing.T) {
	r := New().SetText("hello world")
	tt := widget.NewTester(t, r, 500, 300)
	moved := make(chan struct{}, 8)
	r.OnSelectionChange(func() { moved <- struct{}{} })
	r.SetSelection(6, 11)
	tt.Settle()
	if a, b := r.Selection(); a != 6 || b != 11 || r.SelectedText() != "world" {
		t.Fatalf("selection %d..%d %q", a, b, r.SelectedText())
	}
	tt.Find(widget.ByRole(ui.RoleTextField)).Focus().Key(0, ui.KeyLeft)
	wait.For(t, "the handler to run", moved)
}

func TestRichTextBoxDocumentFormatsTheSelection(t *testing.T) {
	r := New().SetText("make bold")
	tt := widget.NewTester(t, r, 500, 300)
	r.SetSelection(5, 9)
	r.Edit(func(d *richtext.Document) { d.ToggleBold() })
	tt.Settle()
	if m := r.Markup(); m != "make <b>bold</b>" {
		t.Fatalf("Markup %q", m)
	}
}

func TestRichTextBoxLabelNamesTheBox(t *testing.T) {
	tt := widget.NewTester(t, New().SetLabel("Notes"), 500, 300)
	if !tt.Has(widget.ByRole(ui.RoleTextField).Label("Notes")) {
		t.Fatalf("nodes %+v", tt.Nodes())
	}
	if !widget.NewTester(t, New(), 500, 300).Has(widget.ByRole(ui.RoleTextField).Label("Rich text")) {
		t.Fatal("no default label")
	}
}

// Offsets are runes; a selection never splits a grapheme.
func TestRichTextBoxSelectionCountsRunesAndKeepsGraphemesWhole(t *testing.T) {
	precomposed := "Xin chào thế giới"
	combining := "Xin cha\u0300o the\u0302\u0301 gio\u0301i" // à ế ớ as base + marks
	for _, c := range []struct{ name, text, want string }{
		{"precomposed", precomposed, "chào"},
		{"combining", combining, "cha\u0300o"},
	} {
		r := New().SetText(c.text)
		start := 4 // "Xin " is four runes in both forms
		r.SetSelection(start, start+len([]rune(c.want)))
		if got := r.SelectedText(); got != c.want {
			t.Errorf("%s: selected %q, want %q", c.name, got, c.want)
		}
		if s, e := r.Selection(); e-s != len([]rune(c.want)) {
			t.Errorf("%s: selection %d..%d is not counted in runes", c.name, s, e)
		}
	}
	// A selection that ends between 'a' and its grave accent grows to take both.
	r := New().SetText(combining)
	a := strings.Index(combining, "a\u0300")
	from := len([]rune(combining[:a]))
	r.SetSelection(from-1, from+1) // ends between a and the combining mark
	if got := r.SelectedText(); got != "ha\u0300" {
		t.Errorf("a selection ending inside a grapheme: %q", got)
	}
	r.SetSelection(from+1, from+3) // starts inside it
	if got := r.SelectedText(); got != "a\u0300o" {
		t.Errorf("a selection starting inside a grapheme: %q", got)
	}
	if s, e := r.Selection(); e-s != 3 {
		t.Errorf("selection %d..%d: counted in runes, a\u0300o is 3", s, e)
	}
	r.SetSelection(-5, 999)
	if s, e := r.Selection(); s != 0 || e != len([]rune(combining)) {
		t.Errorf("out of range selection %d..%d", s, e)
	}
}

func TestRichTextBoxOnChangeForUndoRedoToolbarAndPaste(t *testing.T) {
	changed := make(chan struct{}, 16)
	r := New().SetText("abc").OnChange(func() { changed <- struct{}{} })
	tt := widget.NewTester(t, r, 500, 300)
	next := func(what string) {
		t.Helper()
		tt.Settle()
		wait.For(t, "the handler to run", changed)
		tt.Settle()
		select {
		case <-changed:
			t.Fatalf("%s reported twice", what)
		default:
		}
	}
	box := tt.Find(widget.ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyEnd).Type("d")
	next("typing")
	r.Edit(func(d *richtext.Document) { d.Undo() })
	next("undo")
	r.Edit(func(d *richtext.Document) { d.Redo() })
	next("redo")
	r.SetSelection(0, 2)
	r.Edit(func(d *richtext.Document) { d.ToggleBold() })
	next("ToggleBold through Edit")
	tt.SetClipboard("pasted")
	box.Key(ui.Cmd, ui.KeyV)
	next("paste")
	if !strings.Contains(r.Text(), "pasted") {
		t.Fatalf("Text %q", r.Text())
	}
}

func TestRichTextBoxInputMethodCommitIsOneChange(t *testing.T) {
	changed := make(chan struct{}, 16)
	r := New().SetText("tie").OnChange(func() { changed <- struct{}{} })
	tt := widget.NewTester(t, r, 500, 300)
	box := tt.Find(widget.ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyEnd)
	tt.Compose("ê", 1) // Telex: e + e
	tt.Tester.Type("ê")
	tt.Settle()
	wait.For(t, "the handler to run", changed)
	tt.Settle()
	select {
	case <-changed:
		t.Fatal("a composition and its commit reported twice")
	default:
	}
	if !strings.HasSuffix(r.Text(), "ê") {
		t.Fatalf("Text %q", r.Text())
	}
}

// shownBox shows b in a form on the fake backend, drains the frames asked
// so far, and returns whether a frame has been asked for since.
func shownBox(t *testing.T, b *Box) (asked func() bool) {
	t.Helper()
	f := widget.NewForm("Rich", 400, 300).SetContent(b)
	f.Show()
	t.Cleanup(f.Close)
	ws := fb.Windows()
	s := ws[len(ws)-1].FakeSurface()
	mygo.RunOnMain(func() {
		for s.Frame() {
		}
	})
	return func() bool {
		var pending bool
		mygo.RunOnMain(func() { pending = s.Frame() })
		return pending
	}
}

// Changes from a toolbar or a worker must schedule a frame by themselves; a
// Tester's Settle forces one, which hides the lack.
func TestRichTextBoxEditAndSetSelectionAskForAFrame(t *testing.T) {
	b := New().SetText("some text")
	asked := shownBox(t, b)
	if asked() {
		t.Fatal("a frame was still pending")
	}
	b.SetSelection(0, 4)
	if !asked() {
		t.Error("SetSelection asked for no frame")
	}
	b.Edit(func(d *richtext.Document) { d.ToggleBold() })
	if !asked() {
		t.Error("Edit asked for no frame")
	}
	b.SetText("other")
	if !asked() {
		t.Error("SetText asked for no frame")
	}
}

func TestRichTextBoxASetterDoesNotSwallowAPendingUserEdit(t *testing.T) {
	changed := make(chan struct{}, 8)
	r := New().SetText("abc").OnChange(func() { changed <- struct{}{} })
	tt := widget.NewTester(t, r, 500, 300)
	r.SetSelection(0, 2)
	r.Document().ToggleBold() // an edit of the user's, not yet reported
	r.SetText("replaced")     // a setter comes in before the next frame
	tt.Settle()
	wait.For(t, "the pending edit's report", changed)
}

func TestRichTextBoxMarkupKeepsTranslucency(t *testing.T) {
	for _, in := range []string{
		"<span foreground='#ff000080'>half</span>",
		"<span background='#00ff0040'>faint</span>",
	} {
		if got := New().SetMarkup(in).Markup(); got != in {
			t.Errorf("Markup() = %q, want %q", got, in)
		}
	}
}

func TestRichTextBoxACaretInAGraphemeGoesBeforeIt(t *testing.T) {
	flag := "ab\U0001F1FB\U0001F1F3cd" // a, b, a flag of two regional indicators, c, d
	r := New().SetText(flag)
	r.SetSelection(3, 3) // between the two halves of the flag
	if s, e := r.Selection(); s != 2 || e != 2 {
		t.Fatalf("caret at %d..%d, want a caret at 2", s, e)
	}
	r.SetSelection(2, 4) // the whole flag
	if s, e := r.Selection(); s != 2 || e != 4 {
		t.Fatalf("selection %d..%d", s, e)
	}
}

func TestRichTextBoxSetTextDoesNotReportASelectionChange(t *testing.T) {
	moved := make(chan struct{}, 8)
	r := New().OnSelectionChange(func() { moved <- struct{}{} })
	tt := widget.NewTester(t, r, 500, 300)
	r.SetText("hello")
	tt.Settle()
	select {
	case <-moved:
		t.Fatal("SetText ran OnSelectionChange")
	default:
	}
}

func TestRichTextBoxASetterDoesNotSwallowAPendingSelectionReport(t *testing.T) {
	moved := make(chan struct{}, 8)
	r := New().SetText("abcdef").OnSelectionChange(func() { moved <- struct{}{} })
	tt := widget.NewTester(t, r, 500, 300)
	tt.Settle()
	for len(moved) > 0 {
		<-moved
	}
	r.SetSelection(0, 2)  // a move nobody reported yet
	r.SetText("replaced") // a setter comes in before the next frame
	tt.Settle()
	wait.For(t, "the pending selection report", moved)
}
