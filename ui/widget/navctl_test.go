package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// navControls are one of each control of this file's batch.
type navControls struct {
	search *SearchBoxWidget
	crumb  *BreadCrumbWidget
	acc    *AccordionWidget
	color  *ColorEditWidget
	box    *TextBoxWidget
}

func newNavControls() navControls {
	return navControls{
		search: SearchBox().SetPlaceholder("Find"),
		crumb:  BreadCrumb("C:", "Users", "ada"),
		acc:    threeSections().SetOpen(0),
		color:  ColorEdit().SetColor(ui.Hex("#3b82f6")),
		box:    TextBox().SetSuggestions("Hanoi", "Hue").SetMRU(4),
	}
}

func (n navControls) all() Control {
	return VBox(8).Add(n.search, n.crumb, n.acc, n.color, n.box)
}

func TestNavControlsFromGoroutine(t *testing.T) {
	n := newNavControls()
	tt := NewTester(t, n.all(), 500, 900)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			n.search.SetText(string(rune('a' + i%26)))
			n.search.SetPlaceholder("Find")
			n.crumb.SetItems("C:", "Users", string(rune('a'+i%26)))
			n.acc.SetOpen(i % 3)
			n.color.SetColor(ui.RGB(uint8(i), 0, 0))
			n.box.SetSuggestions("Hanoi", "Hue", string(rune('a'+i%26)))
			n.box.SetMRU(i%4 + 1)
			_, _, _, _ = n.search.Text(), n.acc.Open(), n.color.Color(), n.box.Text()
		}
		n.search.SetText("last")
		n.crumb.SetItems("D:", "Final")
		n.acc.SetOpen(2)
		n.color.SetColor(ui.Hex("#22c55e"))
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	closeRuns(tt)
	if v := tt.Find(ByRole(ui.RoleTextField).Nth(0)).Value(); v != "last" {
		t.Fatalf("the search field shows %q, want last", v)
	}
	if c := currents(tt); !slices.Equal(c, []string{"Final"}) {
		t.Fatalf("current %q, want Final", c)
	}
	if n.acc.Open() != 2 || !tt.HasText("Proxy field") || tt.HasText("Name field") {
		t.Fatalf("Open %d, texts %q", n.acc.Open(), tt.Texts())
	}
	if v := tt.Find(ByRole(ui.RoleColorWell)).Value(); v != "#22c55e" {
		t.Fatalf("the well shows %q", v)
	}
}

// Added: Review focus 4. A disabled control takes no clicks or keys.
func TestNavControlsDisabled(t *testing.T) {
	fired := make(chan string, 16)
	n := newNavControls()
	n.search.OnChange(func() { fired <- "search" }).OnSubmit(func() { fired <- "submit" })
	n.crumb.OnSelect(func(int) { fired <- "crumb" })
	n.acc.OnChange(func() { fired <- "accordion" })
	n.color.OnChange(func() { fired <- "color" })
	n.search.SetDisabled(true)
	n.crumb.SetDisabled(true)
	n.acc.SetDisabled(true)
	n.color.SetDisabled(true)
	tt := NewTester(t, n.all(), 500, 900)
	click := func(q Query) {
		t.Helper()
		nd := tt.Find(q)
		if !nd.Disabled() {
			t.Fatalf("%s is not disabled", nd)
		}
		b := nd.Bounds()
		tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
		tt.Settle()
	}
	click(ByRole(ui.RoleTextField).Nth(0))
	tt.Tester.Type("x")
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	click(ByLabel("Users").Role(ui.RoleLink))
	click(ByRole(ui.RoleDisclosure).Label("Privacy"))
	click(ByRole(ui.RoleColorWell))
	closeRuns(tt)
	if len(fired) != 0 {
		t.Fatalf("a disabled control ran %q", <-fired)
	}
	if n.search.Text() != "" || n.acc.Open() != 0 || tt.HasText("Red") {
		t.Fatalf("search %q, Open %d, texts %q", n.search.Text(), n.acc.Open(), tt.Texts())
	}
}

// A frame of the controls at rest allocates no more than a frame of the
// bare ui elements they wrap.
func TestNavControlsAllocNothingAtRest(t *testing.T) {
	if raceEnabled {
		// As ui's alloc tests: ui.ColorWell alone counts 4 or 5 a frame
		// under the race detector, bare or wrapped.
		t.Skip("the race detector allocates")
	}
	query, chosen, color, city := "", -1, ui.Hex("#3b82f6"), ""
	open := [3]bool{true, false, false}
	path, cities := []string{"C:", "Users", "ada"}, []string{"Hanoi", "Hue"}
	titles := [3]string{"General", "Privacy", "Advanced"}
	// The closures are made once, as the widgets make theirs, so that the
	// two frames differ only by what the widgets add.
	var ctx *ui.Context
	var items, column func()
	bodies := [3]func(){}
	bare := ui.NewTester(func(c *ui.Context) {
		ctx = c
		if column == nil {
			for i := range titles {
				text := titles[i] + " field"
				bodies[i] = func() { ui.Text(ctx, text) }
			}
			items = func() {
				for i := range titles {
					ui.AccordionItem(ctx, titles[i], &open[i], bodies[i])
				}
			}
			column = func() {
				ui.SearchField(ctx, &query).Placeholder("Find")
				ui.Breadcrumbs(ctx, path, &chosen)
				ui.Accordion(ctx, items)
				ui.ColorWell(ctx, &color)
				ui.Autocomplete(ctx, &city, cities)
			}
		}
		ui.Column(c).Fill().Children(column)
	}, 500, 900)
	nav := ui.NewTester(content(newNavControls().all()), 500, 900)
	for range 10 {
		bare.Frame()
		nav.Frame()
	}
	want, got := allocrun.Least(30, bare.Frame), allocrun.Least(30, nav.Frame)
	t.Logf("a frame allocates %v, the bare elements' %v", got, want)
	if got > want {
		t.Fatalf("a frame of the controls at rest allocates %v, more than the bare elements' %v", got, want)
	}
}
