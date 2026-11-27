package widget

import (
	"fmt"
	"image/color"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// editLog records, in order, what the event goroutine ran.
type editLog struct {
	mu  sync.Mutex
	got []string
}

func (l *editLog) add(s string) func() {
	return func() { l.mu.Lock(); l.got = append(l.got, s); l.mu.Unlock() }
}

func (l *editLog) all(t *testing.T) []string {
	t.Helper()
	flushEvents(t)
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.got)
}

func (l *editLog) reset() { l.mu.Lock(); l.got = nil; l.mu.Unlock() }

// editTester shows b, 300 wide, in a window of 400 by 120, and returns
// its text field.
func editTester(t *testing.T, b *ButtonEditWidget) (*Tester, *Node) {
	t.Helper()
	calmOff()
	tt := NewTester(t, b.SetSize(300, 0), 400, 120)
	tt.Settle()
	return tt, tt.Find(ByRole(ui.RoleTextField))
}

// editButton is the field's button named name.
func editButton(tt *Tester, name string) *Node {
	return tt.Find(ByRole(ui.RoleButton).Label(name))
}

func TestButtonEditButtonsInOrder(t *testing.T) {
	b := ButtonEdit(
		EditButton{Kind: EditEllipsis, Tooltip: "One"},
		EditButton{Kind: EditSearch, Tooltip: "Two"},
		EditButton{Kind: EditPlus, Tooltip: "Three"},
	).SetText("hello")
	tt, in := editTester(t, b)
	check := func(rtl bool) {
		t.Helper()
		f := in.Bounds()
		var at []ui.Rect
		for _, name := range []string{"One", "Two", "Three"} {
			at = append(at, editButton(tt, name).Bounds())
		}
		// The frame is 300 wide: at the window's left, or its right in RTL.
		left, right := float32(0), float32(300)
		if rtl {
			left, right = 100, 400
		}
		for i, r := range at {
			if r.W <= 0 || r.H <= 0 {
				t.Fatalf("rtl %v: button %d is %v", rtl, i, r)
			}
			if r.X < left-0.5 || r.X+r.W > right+0.5 || r.Y < f.Y-1 || r.Y+r.H > f.Y+f.H+1 {
				t.Fatalf("rtl %v: button %d at %v is outside the frame [%v, %v] beside the text %v", rtl, i, r, left, right, f)
			}
			if !rtl && r.X < f.X+f.W-0.5 || rtl && r.X+r.W > f.X+0.5 {
				t.Fatalf("rtl %v: button %d at %v is not at the field's end, past the text %v", rtl, i, r, f)
			}
			if i > 0 && (!rtl && r.X < at[i-1].X+at[i-1].W-0.5 || rtl && r.X+r.W > at[i-1].X+0.5) {
				t.Fatalf("rtl %v: button %d at %v is not after button %d at %v", rtl, i, r, i-1, at[i-1])
			}
		}
	}
	check(false)
	tt.SetDirection(ui.RightToLeft)
	tt.Settle()
	check(true)
}

func TestButtonEditClick(t *testing.T) {
	var log editLog
	b := ButtonEdit(
		EditButton{Kind: EditEllipsis, Tooltip: "One", OnClick: log.add("one")},
		EditButton{Kind: EditDropDown, Tooltip: "Two", OnClick: log.add("two")},
		EditButton{Kind: EditMinus, Tooltip: "Three", OnClick: log.add("three")},
	)
	tt, in := editTester(t, b)
	in.Focus()
	for _, name := range []string{"One", "Two", "Three"} {
		editButton(tt, name).Click()
	}
	if got := log.all(t); !slices.Equal(got, []string{"one", "two", "three"}) {
		t.Fatalf("clicks ran %q, want one, two, three, each once", got)
	}
	if !in.Focused() {
		t.Fatal("a click on a button moved the focus out of the text")
	}
}

func TestButtonEditClear(t *testing.T) {
	var log editLog
	b := ButtonEdit(EditButton{Kind: EditClear, Tooltip: "Clear it", OnClick: log.add("click")}).
		OnChange(log.add("change")).SetText("abc")
	tt, in := editTester(t, b)
	editButton(tt, "Clear it").Click()
	if s := b.Text(); s != "" {
		t.Fatalf("after Clear the text is %q", s)
	}
	if v := in.Value(); v != "" {
		t.Fatalf("after Clear the field shows %q", v)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change", "click"}) {
		t.Fatalf("Clear ran %q, want change then click", got)
	}
	if tt.Has(ByRole(ui.RoleButton).Label("Clear it")) {
		t.Fatal("Clear shows while the text is empty")
	}
	log.reset()
	in.Type("x")
	if s := b.Text(); s != "x" {
		t.Fatalf("typing x after Clear made %q: the field kept the old text", s)
	}
	if !tt.Has(ByRole(ui.RoleButton).Label("Clear it")) {
		t.Fatal("Clear did not show once typing started")
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("typing x ran %q, want one change", got)
	}
	b.SetText("")
	tt.Settle()
	if tt.Has(ByRole(ui.RoleButton).Label("Clear it")) {
		t.Fatal("Clear shows after SetText(\"\")")
	}
}

func TestButtonEditReadOnly(t *testing.T) {
	var log editLog
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse", OnClick: log.add("click")}).
		OnChange(log.add("change")).SetText("abc").SetReadOnly(true)
	tt, in := editTester(t, b)
	in.Type("x")
	if s := b.Text(); s != "abc" {
		t.Fatalf("typing in a read-only field made %q", s)
	}
	editButton(tt, "Browse").Click()
	if got := log.all(t); !slices.Equal(got, []string{"click"}) {
		t.Fatalf("read-only: ran %q, want the button's click alone", got)
	}
	b.SetReadOnly(false)
	tt.Settle()
	in.Type("x")
	if s := b.Text(); s != "abcx" {
		t.Fatalf("SetReadOnly(false): typing made %q, want abcx", s)
	}
}

// Clear with no OnChange, OnClick or binding clears, and nothing panics
// (Post recovers a panic: OnPanic sees it).
func TestButtonEditClearWithoutHandlers(t *testing.T) {
	var log editLog
	OnPanic(func(v any, _ []byte) { log.add(fmt.Sprint(v))() })
	t.Cleanup(func() { OnPanic(nil) })
	b := ButtonEdit(EditButton{Kind: EditClear, Tooltip: "Clear it"}, EditButton{Kind: EditEllipsis, Tooltip: "Browse"}).SetText("abc")
	tt, _ := editTester(t, b)
	editButton(tt, "Browse").Click()
	editButton(tt, "Clear it").Click()
	if s := b.Text(); s != "" {
		t.Fatalf("after Clear the text is %q", s)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("Clear panicked: %q", got)
	}
}

// Clear is a button: it still works in a read-only field.
func TestButtonEditReadOnlyClear(t *testing.T) {
	var log editLog
	b := ButtonEdit(EditButton{Kind: EditClear, Tooltip: "Clear it"}).
		OnChange(log.add("change")).SetText("abc").SetReadOnly(true)
	tt, _ := editTester(t, b)
	editButton(tt, "Clear it").Click()
	if s := b.Text(); s != "" {
		t.Fatalf("Clear in a read-only field left %q", s)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("Clear in a read-only field ran %q, want one change", got)
	}
}

// The buttons are no stops of Tab: Tab from the text goes to the next
// control.
func TestButtonEditButtonsAreNoTabStops(t *testing.T) {
	calmOff()
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse"}, EditButton{Kind: EditClear, Tooltip: "Clear it"}).SetText("x")
	tt := NewTester(t, VBox(0).Add(b, Button("Next")), 400, 200)
	tt.Settle()
	tt.Find(ByRole(ui.RoleTextField)).Key(0, ui.KeyTab)
	if !tt.Find(ByRole(ui.RoleButton).Label("Next")).Focused() {
		for _, n := range tt.FindAll(ByRole(ui.RoleButton)) {
			if n.Focused() {
				t.Fatalf("Tab from the text focused %v, want Next", n)
			}
		}
		t.Fatal("Tab from the text did not focus Next")
	}
}

// Ctrl+Enter clicks the button, not the form's accept button; Enter
// clicks that.
func TestButtonEditCtrlEnterInAForm(t *testing.T) {
	calmOff()
	var log editLog
	f := NewForm("T", 400, 200)
	ok := Button("OK").OnClick(log.add("ok"))
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse", OnClick: log.add("browse")})
	f.SetContent(VBox(0).Add(b, ok)).SetAcceptButton(ok)
	tt := TestForm(t, f)
	tt.Settle()
	in := tt.Find(ByRole(ui.RoleTextField))
	in.Key(ui.Ctrl, ui.KeyEnter)
	if got := log.all(t); !slices.Equal(got, []string{"browse"}) {
		t.Fatalf("Ctrl+Enter ran %q, want browse alone", got)
	}
	log.reset()
	in.Key(0, ui.KeyEnter)
	if got := log.all(t); !slices.Equal(got, []string{"ok"}) {
		t.Fatalf("Enter ran %q, want the accept button's ok", got)
	}
}

// In a form fitted to its content (a measuring pass each refit), Ctrl+Enter
// clicks once.
func TestButtonEditCtrlEnterInAFittedForm(t *testing.T) {
	calmOff()
	var log editLog
	f := NewForm("T", 400, 0)
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse", OnClick: log.add("browse")})
	f.SetContent(VBox(0).Add(b, Label("below")))
	tt := TestForm(t, f)
	f.SetSize(400, 0)
	tt.Settle()
	tt.Find(ByRole(ui.RoleTextField)).Key(ui.Ctrl, ui.KeyEnter)
	if got := log.all(t); !slices.Equal(got, []string{"browse"}) {
		t.Fatalf("Ctrl+Enter in a fitted form ran %q, want browse once", got)
	}
}

// A disabled ButtonEdit takes no typing and its buttons no click.
func TestButtonEditDisabled(t *testing.T) {
	var log editLog
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse", OnClick: log.add("browse")},
		EditButton{Kind: EditClear, Tooltip: "Clear it", OnClick: log.add("clear")}).
		OnChange(log.add("change")).SetText("abc").SetDisabled(true)
	tt, in := editTester(t, b)
	for _, name := range []string{"Browse", "Clear it"} {
		r := editButton(tt, name).Bounds()
		tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		tt.Settle()
	}
	r := in.Bounds()
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Type("x")
	tt.Key(ui.Ctrl, ui.KeyEnter)
	tt.Settle()
	if s := b.Text(); s != "abc" {
		t.Fatalf("a disabled ButtonEdit changed to %q", s)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("a disabled ButtonEdit ran %q", got)
	}
}

// Under a skin's Field look of a wider border, as High Contrast's, the
// frame is as high as a TextBox under it. (A look with padding pads the
// frame around the text's own padding, as it does a LookUpEdit's: see the
// report.)
func TestButtonEditBorderLookAsHighAsATextBox(t *testing.T) {
	useLooks(t, map[string]string{"Field": "border-2"})
	box := TextBox().SetText("hello").SetSize(300, 0)
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse"}).SetText("hello").SetSize(300, 0)
	look := LookUpEdit(custKey, custName, custCols()...).SetItems(custs()).SetSize(300, 0)
	tt := NewTester(t, VBox(0).Add(box, b, Label("after edit"), look, Label("after look")), 400, 300)
	tt.Settle()
	r := tt.FindAll(ByRole(ui.RoleTextField))[0].Bounds()
	boxH, boxEnd := r.H, r.Y+r.H
	editEnd := textAt(t, tt, "after edit").Y
	lookH := textAt(t, tt, "after look").Y - (editEnd + textAt(t, tt, "after edit").H)
	editH := editEnd - boxEnd
	t.Logf("TextBox %v, ButtonEdit %v, LookUpEdit %v", boxH, editH, lookH)
	if editH > boxH+0.5 || editH < boxH-0.5 {
		t.Fatalf("under a padded look the ButtonEdit is %v high, a TextBox %v", editH, boxH)
	}
}

// Without a skin the frame looks as a TextBox: as high, edged with the
// outline, and with the accent while the text has the focus.
func TestButtonEditLooksAsATextBox(t *testing.T) {
	calmOff()
	th := new(ui.Theme)
	keep := Custom(func(c *ui.Context) ui.Element { *th = *c.Theme(); return ui.Box(c) })
	box := TextBox().SetText("hello").SetSize(300, 0)
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse"}).SetText("hello").SetSize(300, 0)
	tt := NewTester(t, VBox(0).Add(keep, box, b), 400, 200)
	tt.Settle()
	fields := tt.FindAll(ByRole(ui.RoleTextField))
	boxH := fields[0].Bounds().H
	in := fields[1]
	r := in.Bounds()
	if r.H+2 < boxH-0.5 || r.H+2 > boxH+0.5 {
		t.Fatalf("the ButtonEdit's frame is %v high (text %v and its edge), a TextBox %v", r.H+2, r.H, boxH)
	}
	edge := func() color.RGBA { return tt.Image().RGBAAt(0, int(r.Y+r.H/2)) }
	if c := edge(); !near(c, th.Outline().Over(th.FieldFill())) && !near(c, th.Outline().Over(th.Background)) {
		t.Fatalf("the frame's edge is %v, want the outline %v", c, th.Outline())
	}
	in.Focus()
	if c := edge(); !near(c, th.Accent) {
		t.Fatalf("focused, the frame's edge is %v, want the accent %v", c, th.Accent)
	}
}

// The frame takes the skin's Field look: a look of bg-chrome fills it with
// Chrome.
func TestButtonEditFieldLook(t *testing.T) {
	useLooks(t, map[string]string{"Field": "bg-chrome"})
	tk := skinhook.Get(false)
	tt := NewTester(t, VBox(0).Add(ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse"})), 400, 120)
	tt.Settle()
	r := tt.Find(ByRole(ui.RoleTextField)).Bounds()
	if c := tt.Image().RGBAAt(int(r.X+r.W-4), int(r.Y+r.H/2)); !near(c, tk.Chrome) || tk.Chrome == tk.Background {
		t.Fatalf("the field fills with %v, want Chrome %v", c, tk.Chrome)
	}
}

func TestButtonEditCtrlEnter(t *testing.T) {
	var log editLog
	b := ButtonEdit(
		EditButton{Kind: EditSearch, Tooltip: "Search", OnClick: log.add("search")},
		EditButton{Kind: EditDropDown, Tooltip: "Open", OnClick: log.add("open")},
		EditButton{Kind: EditEllipsis, Tooltip: "Browse", OnClick: log.add("browse")},
	)
	_, in := editTester(t, b)
	in.Key(ui.Ctrl, ui.KeyEnter)
	if got := log.all(t); !slices.Equal(got, []string{"open"}) {
		t.Fatalf("Ctrl+Enter ran %q, want the first DropDown or Ellipsis: open", got)
	}
	// A hidden button cannot be clicked: the next one is.
	log.reset()
	b.SetButtonVisible(1, false)
	in.Key(ui.Ctrl, ui.KeyEnter)
	if got := log.all(t); !slices.Equal(got, []string{"browse"}) {
		t.Fatalf("Ctrl+Enter with Open hidden ran %q, want browse", got)
	}

	var none editLog
	b2 := ButtonEdit(
		EditButton{Kind: EditSearch, Tooltip: "Search", OnClick: none.add("search")},
		EditButton{Kind: EditPlus, Tooltip: "Add", OnClick: none.add("add")},
	)
	_, in2 := editTester(t, b2)
	in2.Key(ui.Ctrl, ui.KeyEnter)
	if got := none.all(t); len(got) != 0 {
		t.Fatalf("Ctrl+Enter with no Ellipsis or DropDown ran %q", got)
	}
}

func TestButtonEditSetButtonVisible(t *testing.T) {
	b := ButtonEdit(
		EditButton{Kind: EditEllipsis, Tooltip: "One"},
		EditButton{Kind: EditSearch, Tooltip: "Two"},
		EditButton{Kind: EditPlus, Tooltip: "Three"},
	)
	tt, in := editTester(t, b)
	w := in.Bounds().W
	two := editButton(tt, "Two").Bounds().W
	b.SetButtonVisible(1, false)
	tt.Settle()
	if tt.Has(ByRole(ui.RoleButton).Label("Two")) {
		t.Fatal("a hidden button shows")
	}
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 2 {
		t.Fatalf("%d buttons, want 2", n)
	}
	if got := in.Bounds().W; got < w+two-0.5 {
		t.Fatalf("with a button hidden the text is %v wide, want at least %v: the button keeps its room", got, w+two)
	}
	for _, i := range []int{-1, 3, 99} {
		b.SetButtonVisible(i, false)
	}
	tt.Settle()
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 2 {
		t.Fatalf("after indexes out of range %d buttons, want 2", n)
	}
	b.SetButtonVisible(1, true)
	tt.Settle()
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 3 {
		t.Fatalf("shown again: %d buttons, want 3", n)
	}
	if got := in.Bounds().W; got != w {
		t.Fatalf("shown again, the text is %v wide, want %v", got, w)
	}
}

func TestButtonEditGlyph(t *testing.T) {
	red := color.RGBA{220, 0, 0, 255}
	b := ButtonEdit(
		EditButton{Kind: EditGlyph, Tooltip: "Red", Image: solidBitmap(red)},
		EditButton{Kind: EditGlyph, Tooltip: "Nothing"},
	)
	tt, _ := editTester(t, b)
	img := tt.Image()
	reds := func(r ui.Rect) int {
		n := 0
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			for x := int(r.X); x < int(r.X+r.W); x++ {
				if img.RGBAAt(x, y) == red {
					n++
				}
			}
		}
		return n
	}
	if n := reds(editButton(tt, "Red").Bounds()); n < 50 {
		t.Fatalf("the glyph button shows %d red pixels, want its image", n)
	}
	if n := reds(editButton(tt, "Nothing").Bounds()); n != 0 {
		t.Fatalf("a glyph with no image shows %d red pixels", n)
	}
	editButton(tt, "Nothing").Click() // nil Image, no OnClick: no panic
}

func TestButtonEditSetterDoesNotFire(t *testing.T) {
	var log editLog
	b := ButtonEdit(EditButton{Kind: EditClear, Tooltip: "Clear it"}).OnChange(log.add("change"))
	tt, in := editTester(t, b)
	b.SetText("hanoi")
	tt.Settle()
	if v := in.Value(); v != "hanoi" {
		t.Fatalf("the field shows %q, want hanoi", v)
	}
	b.SetText("").SetPlaceholder("Find").SetReadOnly(true).SetButtonVisible(0, false)
	tt.Settle()
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("setters ran %q", got)
	}
}

func TestButtonEditFromGoroutine(t *testing.T) {
	b := ButtonEdit(
		EditButton{Kind: EditClear, Tooltip: "Clear it"},
		EditButton{Kind: EditEllipsis, Tooltip: "Browse"},
	)
	tt, in := editTester(t, b)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			b.SetText(fmt.Sprint(i)).SetPlaceholder(fmt.Sprint("p", i)).SetReadOnly(i%2 == 0)
			b.SetButtonVisible(1, i%3 != 0)
			_ = b.Text()
		}
		b.SetText("done").SetReadOnly(false).SetButtonVisible(1, true)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	if v := in.Value(); v != "done" {
		t.Fatalf("the field shows %q, want done", v)
	}
	if !tt.Has(ByRole(ui.RoleButton).Label("Browse")) || !tt.Has(ByRole(ui.RoleButton).Label("Clear it")) {
		t.Fatal("the buttons set from another goroutine do not show")
	}
}

func TestButtonEditBind(t *testing.T) {
	var mu sync.Mutex
	v := "start"
	get := func() string { mu.Lock(); defer mu.Unlock(); return v }
	b := Bind(ButtonEdit(EditButton{Kind: EditClear, Tooltip: "Clear it"}), get,
		func(x string) { mu.Lock(); v = x; mu.Unlock() })
	tt, in := editTester(t, b)
	if s := in.Value(); s != "start" {
		t.Fatalf("the field shows %q, not the source's start", s)
	}
	in.Type("!")
	if s := get(); s != "start!" {
		t.Fatalf("after typing the source has %q", s)
	}
	editButton(tt, "Clear it").Click()
	flushEvents(t)
	if s := get(); s != "" {
		t.Fatalf("after Clear the source has %q", s)
	}
	mu.Lock()
	v = "again"
	mu.Unlock()
	Reload(b)
	tt.Settle()
	if s := in.Value(); s != "again" {
		t.Fatalf("after Reload the field shows %q", s)
	}
}

// An ErrorProvider's mark sits at the frame's end: the buttons stop before
// it.
func TestButtonEditErrorMarkClearsTheButtons(t *testing.T) {
	b := ButtonEdit(EditButton{Kind: EditEllipsis, Tooltip: "Browse"})
	tt, _ := editTester(t, b)
	ErrorProvider().SetError(b, "Wrong")
	tt.Settle()
	r := editButton(tt, "Browse").Bounds()
	if r.X+r.W > 300-errorMarkSize-errorMarkInset {
		t.Fatalf("the button ends at %v, under the error mark at the frame's end (300)", r.X+r.W)
	}
}

// The buttons have names for assistive technology without a tooltip.
func TestButtonEditNames(t *testing.T) {
	b := ButtonEdit(EditButton{Kind: EditClear}, EditButton{Kind: EditSearch}, EditButton{Kind: EditPlus},
		EditButton{Kind: EditMinus}, EditButton{Kind: EditEllipsis}, EditButton{Kind: EditDropDown}).SetText("x")
	tt, _ := editTester(t, b)
	for _, n := range tt.FindAll(ByRole(ui.RoleButton)) {
		if n.Label() == "" {
			t.Fatalf("a button has no name among %v", tt.FindAll(ByRole(ui.RoleButton)))
		}
	}
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 6 {
		t.Fatalf("%d buttons, want 6", n)
	}
}

func TestButtonEditAllocNothingAtRest(t *testing.T) {
	calmOff()
	box := ui.NewTester(content(TextBox().SetText("hello")), 400, 120)
	edit := ui.NewTester(content(ButtonEdit(
		EditButton{Kind: EditClear, Tooltip: "Clear it", OnClick: func() {}},
		EditButton{Kind: EditEllipsis, OnClick: func() {}},
		EditButton{Kind: EditGlyph, Image: solidBitmap(color.RGBA{0, 0, 200, 255})},
	).SetText("hello").SetPlaceholder("Find")), 400, 120)
	for range 10 {
		box.Frame()
		edit.Frame()
	}
	want, got := allocrun.Least(30, box.Frame), allocrun.Least(30, edit.Frame)
	if got > want {
		t.Fatalf("a frame of the ButtonEdit at rest allocates %v, more than a TextBox's %v", got, want)
	}
	t.Logf("a frame allocates %v, a TextBox's %v", got, want)
}
